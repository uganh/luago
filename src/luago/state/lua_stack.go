package state

import "luago/api"

type luaStack struct {
	slots   []luaValue
	top     int
	state   *luaState
	openuvs map[int]*upvalue
	prev    *luaStack
	closure *luaClosure
	varargs []luaValue
	pc      int
}

func newLuaStack(size int, state *luaState) *luaStack {
	return &luaStack{
		slots: make([]luaValue, size),
		top:   0,
		state: state,
	}
}

func (stack *luaStack) check(n int) {
	free := len(stack.slots) - stack.top
	for i := free; i < n; i++ {
		stack.slots = append(stack.slots, nil)
	}
}

func (stack *luaStack) push(val luaValue) {
	if stack.top == len(stack.slots) {
		panic("stack overflow")
	}
	stack.slots[stack.top] = val
	stack.top++
}

func (stack *luaStack) pop() luaValue {
	if stack.top < 1 {
		panic("stack underflow")
	}
	stack.top--
	val := stack.slots[stack.top]
	stack.slots[stack.top] = nil
	return val
}

func (stack *luaStack) absIndex(idx int) int {
	if idx <= api.LUA_REGISTRYINDEX {
		return idx
	}
	if idx >= 0 {
		return idx
	}
	return idx + stack.top + 1
}

func (stack *luaStack) isValid(idx int) bool {
	if idx == api.LUA_REGISTRYINDEX {
		return true
	}

	if idx < api.LUA_REGISTRYINDEX { // upvalue
		closure := stack.closure
		return closure != nil && (api.LUA_REGISTRYINDEX-idx) <= len(closure.upvals)
	}

	absIdx := stack.absIndex(idx)
	return 0 < absIdx && absIdx <= stack.top
}

func (stack *luaStack) get(idx int) luaValue {
	if idx == api.LUA_REGISTRYINDEX {
		return stack.state.registry
	}

	if idx < api.LUA_REGISTRYINDEX { // upvalue
		uvIdx := api.LUA_REGISTRYINDEX - idx - 1
		closure := stack.closure
		if closure == nil || uvIdx >= len(closure.upvals) {
			return nil
		}
		return *(closure.upvals[uvIdx].val)
	}

	absIdx := stack.absIndex(idx)
	if 0 < absIdx && absIdx <= stack.top {
		return stack.slots[absIdx-1]
	}

	return nil
}

func (stack *luaStack) set(idx int, val luaValue) {
	if idx == api.LUA_REGISTRYINDEX {
		stack.state.registry = val.(*luaTable)
		return
	}

	if idx < api.LUA_REGISTRYINDEX { // upvalue
		uvIdx := api.LUA_REGISTRYINDEX - idx - 1
		closure := stack.closure
		if closure != nil && uvIdx < len(closure.upvals) {
			*(closure.upvals[uvIdx].val) = val
		}
		return
	}

	absIdx := stack.absIndex(idx)
	if 0 < absIdx && absIdx <= stack.top {
		stack.slots[absIdx-1] = val
	} else {
		panic("set at invalid index")
	}
}

func (stack *luaStack) reverse(from, to int) {
	slots := stack.slots
	for from < to {
		slots[from], slots[to] = slots[to], slots[from]
		from++
		to--
	}
}

func (stack *luaStack) popN(n int) []luaValue {
	vals := make([]luaValue, n)
	for i := n - 1; i >= 0; i-- {
		vals[i] = stack.pop()
	}
	return vals
}

func (stack *luaStack) pushN(vals []luaValue, n int) {
	nVals := len(vals)
	for i := 0; i < n; i++ {
		if i < nVals {
			stack.push(vals[i])
		} else {
			stack.push(nil)
		}
	}
}
