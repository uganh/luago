package state

import (
	"fmt"
	"luago/api"
	"luago/binary"
	"luago/number"
	"luago/vm"
	"math"
)

type luaState struct {
	registry *luaTable
	stack    *luaStack
}

func New() *luaState {
	registry := newLuaTable(0, 0)
	registry.put(api.LUA_RIDX_GLOBALS, newLuaTable(0, 0)) // `_G`
	state := &luaState{registry: registry}
	state.stack = newLuaStack(api.LUA_MINSTACK, state)
	return state
}

func (state *luaState) GetTop() int {
	return state.stack.top
}

func (state *luaState) AbsIndex(idx int) int {
	return state.stack.absIndex(idx)
}

func (state *luaState) CheckStack(n int) bool {
	state.stack.check(n)
	return true // never fails
}

func (state *luaState) Pop(n int) {
	state.SetTop(-n - 1)
}

func (state *luaState) Copy(srcIdx, dstIdx int) {
	state.stack.set(dstIdx, state.stack.get(srcIdx))
}

func (state *luaState) PushValue(idx int) {
	state.stack.push(state.stack.get(idx))
}

func (state *luaState) Replace(idx int) {
	state.stack.set(idx, state.stack.pop())
}

func (state *luaState) Insert(idx int) {
	state.Rotate(idx, 1)
}

func (state *luaState) Remove(idx int) {
	state.Rotate(idx, -1)
	state.stack.pop()
}

func (state *luaState) Rotate(idx, n int) {
	t := state.stack.top - 1
	p := state.stack.absIndex(idx) - 1
	var m int
	if n >= 0 {
		m = t - n
	} else {
		m = p - n - 1
	}
	state.stack.reverse(p, m)
	state.stack.reverse(m+1, t)
	state.stack.reverse(p, t)
}

func (state *luaState) SetTop(idx int) {
	newTop := state.stack.absIndex(idx)
	if newTop < 0 {
		panic("stack underflow")
	}
	n := state.stack.top - newTop
	if n > 0 {
		for i := 0; i < n; i++ {
			state.stack.pop()
		}
	} else if n < 0 {
		for i := 0; i > n; i-- {
			state.stack.push(nil)
		}
	}
}

func (state *luaState) TypeName(t api.LuaType) string {
	switch t {
	case api.LUA_TNONE:
		return "no value"
	case api.LUA_TNIL:
		return "nil"
	case api.LUA_TBOOLEAN:
		return "boolean"
	case api.LUA_TNUMBER:
		return "number"
	case api.LUA_TSTRING:
		return "string"
	case api.LUA_TTABLE:
		return "table"
	case api.LUA_TFUNCTION:
		return "function"
	case api.LUA_TTHREAD:
		return "thread"
	default:
		return "userdata"
	}
}

func (state *luaState) Type(idx int) api.LuaType {
	if state.stack.isValid(idx) {
		return typeOf(state.stack.get(idx))
	}
	return api.LUA_TNONE
}

func (state *luaState) IsNone(idx int) bool {
	return state.Type(idx) == api.LUA_TNONE
}

func (state *luaState) IsNil(idx int) bool {
	return state.Type(idx) == api.LUA_TNIL
}

func (state *luaState) IsNoneOrNil(idx int) bool {
	return state.Type(idx) <= api.LUA_TNIL
}

func (state *luaState) IsBoolean(idx int) bool {
	return state.Type(idx) == api.LUA_TBOOLEAN
}

func (state *luaState) IsInteger(idx int) bool {
	val := state.stack.get(idx)
	_, ok := val.(int64)
	return ok
}

func (state *luaState) IsNumber(idx int) bool {
	_, ok := state.ToNumberX(idx)
	return ok
}

func (state *luaState) IsString(idx int) bool {
	t := state.Type(idx)
	return t == api.LUA_TSTRING || t == api.LUA_TNUMBER
}

func (state *luaState) IsFunction(idx int) bool {
	return state.Type(idx) == api.LUA_TFUNCTION
}

func (state *luaState) IsGoFunction(idx int) bool {
	val := state.stack.get(idx)
	if c, ok := val.(*luaClosure); ok {
		return c.goFun != nil
	}
	return false
}

func (state *luaState) ToBoolean(idx int) bool {
	return convertToBoolean(state.stack.get(idx))
}

func (state *luaState) ToInteger(idx int) int64 {
	n, _ := state.ToIntegerX(idx)
	return n
}

func (state *luaState) ToIntegerX(idx int) (int64, bool) {
	return convertToInteger(state.stack.get(idx))
}

func (state *luaState) ToNumber(idx int) float64 {
	n, _ := state.ToNumberX(idx)
	return n
}

func (state *luaState) ToNumberX(idx int) (float64, bool) {
	return convertToFloat(state.stack.get(idx))
}

func (state *luaState) ToString(idx int) string {
	s, _ := state.ToStringX(idx)
	return s
}

func (state *luaState) ToStringX(idx int) (string, bool) {
	val := state.stack.get(idx)
	switch x := val.(type) {
	case string:
		return x, true
	case int64, float64:
		s := fmt.Sprintf("%v", x)
		state.stack.set(idx, s)
		return s, true
	default:
		return "", false
	}
}

func (state *luaState) ToGoFunction(idx int) api.GoFunction {
	val := state.stack.get(idx)
	if c, ok := val.(*luaClosure); ok {
		return c.goFun
	}
	return nil
}

func (state *luaState) RawLen(idx int) uint {
	val := state.stack.get(idx)
	if s, ok := val.(string); ok {
		return uint(len(s))
	} else if t, ok := val.(*luaTable); ok {
		return uint(t.len())
	} else {
		return 0
	}
}

func (state *luaState) PushNil() {
	state.stack.push(nil)
}

func (state *luaState) PushBoolean(b bool) {
	state.stack.push(b)
}

func (state *luaState) PushInteger(n int64) {
	state.stack.push(n)
}

func (state *luaState) PushNumber(n float64) {
	state.stack.push(n)
}

func (state *luaState) PushString(s string) {
	state.stack.push(s)
}

func (state *luaState) PushGoFunction(f api.GoFunction) {
	state.PushGoClosure(f, 0)
}

func (state *luaState) PushGoClosure(f api.GoFunction, n int) {
	c := newGoClosure(f, n)
	for i := n - 1; i >= 0; i-- {
		val := state.stack.pop()
		c.upvals[i] = &upvalue{&val}
	}
	state.stack.push(c)
}

func (state *luaState) PushGlobalTable() {
	state.stack.push(state.registry.get(api.LUA_RIDX_GLOBALS))
}

func (state *luaState) Arith(op api.ArithOp) {
	var a, b, r luaValue
	b = state.stack.pop()
	if op != api.LUA_OPUNM && op != api.LUA_OPBNOT {
		a = state.stack.pop()
	} else {
		a = b
	}

	var mName string
	var iFunc func(int64, int64) int64
	var fFunc func(float64, float64) float64

	switch op {
	case api.LUA_OPADD:
		mName = "__add"
		iFunc = func(a, b int64) int64 { return a + b }
		fFunc = func(a, b float64) float64 { return a + b }
	case api.LUA_OPSUB:
		mName = "__sub"
		iFunc = func(a, b int64) int64 { return a - b }
		fFunc = func(a, b float64) float64 { return a - b }
	case api.LUA_OPMUL:
		mName = "__mul"
		iFunc = func(a, b int64) int64 { return a * b }
		fFunc = func(a, b float64) float64 { return a * b }
	case api.LUA_OPMOD:
		mName = "__mod"
		iFunc, fFunc = number.IMod, number.FMod
	case api.LUA_OPPOW:
		mName = "__pow"
		fFunc = math.Pow
	case api.LUA_OPDIV:
		mName = "__div"
		fFunc = func(a, b float64) float64 { return a / b }
	case api.LUA_OPIDIV:
		mName = "__idiv"
		iFunc, fFunc = number.IFloorDiv, number.FFloorDiv
	case api.LUA_OPBAND:
		mName = "__band"
		iFunc = func(a, b int64) int64 { return a & b }
	case api.LUA_OPBOR:
		mName = "__bor"
		iFunc = func(a, b int64) int64 { return a | b }
	case api.LUA_OPBXOR:
		mName = "__bxor"
		iFunc = func(a, b int64) int64 { return a ^ b }
	case api.LUA_OPSHL:
		mName = "__shl"
		iFunc = number.ShiftLeft
	case api.LUA_OPSHR:
		mName = "__shr"
		iFunc = number.ShiftRight
	case api.LUA_OPUNM:
		mName = "__unm"
		iFunc = func(a, _ int64) int64 { return -a }
		fFunc = func(a, _ float64) float64 { return -a }
	case api.LUA_OPBNOT:
		mName = "__bnot"
		iFunc = func(a, _ int64) int64 { return ^a }
	default:
		panic("invalid arith op")
	}

	if fFunc == nil { // bitwise operation
		if a, ok := convertToInteger(a); ok {
			if b, ok := convertToInteger(b); ok {
				r = iFunc(a, b)
			}
		}
	} else {
		if iFunc != nil {
			if a, ok := a.(int64); ok {
				if b, ok := b.(int64); ok {
					r = iFunc(a, b)
				}
			}
		}

		if r == nil {
			if a, ok := convertToFloat(a); ok {
				if b, ok := convertToFloat(b); ok {
					r = fFunc(a, b)
				}
			}
		}
	}

	if r != nil {
		state.stack.push(r)
	} else if r, ok := callMetamethod(a, b, mName, state); ok {
		state.stack.push(r)
	} else {
		panic("arithmetic error")
	}
}

func (state *luaState) Compare(idx1, idx2 int, op api.CompareOp) bool {
	if !state.stack.isValid(idx1) || !state.stack.isValid(idx2) {
		return false
	}

	a := state.stack.get(idx1)
	b := state.stack.get(idx2)

	switch op {
	case api.LUA_OPEQ:
		return equal(a, b, state)
	case api.LUA_OPLT:
		switch a := a.(type) {
		case string:
			if b, ok := b.(string); ok {
				return a < b
			}
		case int64:
			switch b := b.(type) {
			case int64:
				return a < b
			case float64:
				return float64(a) < b
			}
		case float64:
			switch b := b.(type) {
			case float64:
				return a < b
			case int64:
				return a < float64(b)
			}
		}
		if r, ok := callMetamethod(a, b, "__lt", state); ok {
			return convertToBoolean(r)
		}
	case api.LUA_OPLE:
		switch a := a.(type) {
		case string:
			if b, ok := b.(string); ok {
				return a <= b
			}
		case int64:
			switch b := b.(type) {
			case int64:
				return a <= b
			case float64:
				return float64(a) <= b
			}
		case float64:
			switch b := b.(type) {
			case float64:
				return a <= b
			case int64:
				return a <= float64(b)
			}
		}
		if r, ok := callMetamethod(a, b, "__le", state); ok {
			return convertToBoolean(r)
		} else if r, ok := callMetamethod(b, a, "__lt", state); ok {
			return !convertToBoolean(r)
		}
	default:
		panic("invalid compare op")
	}
	panic("comparison error")
}

func (state *luaState) RawEqual(idx1, idx2 int) bool {
	if !state.stack.isValid(idx1) || !state.stack.isValid(idx2) {
		return false
	}

	a := state.stack.get(idx1)
	b := state.stack.get(idx2)

	return equal(a, b, nil)
}

func (state *luaState) NewTable() {
	state.CreateTable(0, 0)
}

func (state *luaState) CreateTable(nArr, nRec int) {
	state.stack.push(newLuaTable(nArr, nRec))
}

func (state *luaState) GetTable(idx int) api.LuaType {
	t := state.stack.get(idx)
	k := state.stack.pop()
	return state.getTable(t, k, false)
}

func (state *luaState) GetField(idx int, k string) api.LuaType {
	return state.getTable(state.stack.get(idx), k, false)
}

func (state *luaState) GetI(idx int, i int64) api.LuaType {
	return state.getTable(state.stack.get(idx), i, false)
}

func (state *luaState) RawGet(idx int) api.LuaType {
	t := state.stack.get(idx)
	k := state.stack.pop()
	return state.getTable(t, k, true)
}

func (state *luaState) RawGetI(idx int, i int64) api.LuaType {
	return state.getTable(state.stack.get(idx), i, true)
}

func (state *luaState) GetMetatable(idx int) bool {
	if mt := getMetatable(state.stack.get(idx), state); mt != nil {
		state.stack.push(mt)
		return true
	}
	return false
}

func (state *luaState) GetGlobal(name string) api.LuaType {
	return state.getTable(state.registry.get(api.LUA_RIDX_GLOBALS), name, true)
}

func (state *luaState) SetTable(idx int) {
	t := state.stack.get(idx)
	v := state.stack.pop()
	k := state.stack.pop()
	state.setTable(t, k, v, false)
}

func (state *luaState) SetField(idx int, k string) {
	t := state.stack.get(idx)
	v := state.stack.pop()
	state.setTable(t, k, v, false)
}

func (state *luaState) SetI(idx int, i int64) {
	t := state.stack.get(idx)
	v := state.stack.pop()
	state.setTable(t, i, v, false)
}

func (state *luaState) RawSet(idx int) {
	t := state.stack.get(idx)
	v := state.stack.pop()
	k := state.stack.pop()
	state.setTable(t, k, v, true)
}

func (state *luaState) RawSetI(idx int, i int64) {
	t := state.stack.get(idx)
	v := state.stack.pop()
	state.setTable(t, i, v, true)
}

func (state *luaState) SetMetatable(idx int) {
	val := state.stack.get(idx)
	mtVal := state.stack.pop()
	if mtVal == nil {
		setMetatable(val, nil, state)
	} else if mt, ok := mtVal.(*luaTable); ok {
		setMetatable(val, mt, state)
	} else {
		panic("table expected") // TODO
	}
}

func (state *luaState) SetGlobal(name string) {
	t := state.registry.get(api.LUA_RIDX_GLOBALS)
	v := state.stack.pop()
	state.setTable(t, name, v, true)
}

func (state *luaState) Register(name string, f api.GoFunction) {
	state.PushGoFunction(f)
	state.SetGlobal(name)
}

func (state *luaState) Load(chunk []byte, chunkName, mode string) int {
	proto := binary.Parse(chunk)
	c := newLuaClosure(proto)
	if len(proto.Upvalues) > 0 {
		val := state.registry.get(api.LUA_RIDX_GLOBALS) // `_ENV`
		c.upvals[0] = &upvalue{&val}
	}
	state.stack.push(c)
	return 0
}

func (state *luaState) Call(nArgs, nResults int) {
	val := state.stack.get(-(nArgs + 1))

	c, ok := val.(*luaClosure)
	if !ok {
		if mf := getMetafield(val, "__call", state); mf != nil {
			if c, ok = mf.(*luaClosure); ok {
				state.stack.check(1)
				state.stack.push(val)
				state.Insert(-(nArgs + 2))
				nArgs += 1
			}
		}
	}

	if ok {
		if c.proto != nil {
			state.callLuaClosure(nArgs, nResults, c)
		} else {
			state.callGoClosure(nArgs, nResults, c)
		}
	} else {
		panic("not a function")
	}
}

func (state *luaState) Len(idx int) {
	val := state.stack.get(idx)
	if s, ok := val.(string); ok {
		state.stack.push(int64(len(s)))
	} else if r, ok := callMetamethod(val, val, "__len", state); ok {
		state.stack.push(r)
	} else if t, ok := val.(*luaTable); ok {
		state.stack.push(int64(t.len()))
	} else {
		panic("length error")
	}
}

func (state *luaState) Concat(n int) {
	if n == 0 {
		state.stack.push("")
	} else if n >= 2 {
		for i := 1; i < n; i++ {
			if state.IsString(-1) && state.IsString(-2) {
				s2 := state.ToString(-1)
				s1 := state.ToString(-2)
				state.stack.pop()
				state.stack.pop()
				state.stack.push(s1 + s2)
				continue
			}

			b := state.stack.pop()
			a := state.stack.pop()
			if r, ok := callMetamethod(a, b, "__concat", state); ok {
				state.stack.push(r)
			} else {
				panic("concatenation error")
			}
		}
	}
}

func (state *luaState) PC() int {
	return state.stack.pc
}

func (state *luaState) AddPC(n int) {
	state.stack.pc += n
}

func (state *luaState) Fetch() uint32 {
	c := state.stack.closure.proto.Code[state.stack.pc]
	state.stack.pc++
	return c
}

func (state *luaState) GetConst(idx int) {
	state.stack.push(state.stack.closure.proto.Constants[idx])
}

func (state *luaState) RegisterCount() int {
	return int(state.stack.closure.proto.MaxStackSize)
}

func (state *luaState) LoadVararg(n int) {
	if n < 0 {
		n = len(state.stack.varargs)
	}
	state.stack.check(n)
	state.stack.pushN(state.stack.varargs, n)
}

func (state *luaState) LoadProto(idx int) {
	stack := state.stack
	proto := stack.closure.proto.Protos[idx]
	c := newLuaClosure(proto)
	for i, info := range proto.Upvalues {
		uvIdx := int(info.Index)
		if info.InStack == 1 {
			if stack.openuvs == nil {
				stack.openuvs = map[int]*upvalue{}
			}
			if upval, found := stack.openuvs[uvIdx]; found {
				c.upvals[i] = upval
			} else {
				c.upvals[i] = &upvalue{&stack.slots[uvIdx]}
				stack.openuvs[uvIdx] = c.upvals[i]
			}
		} else {
			c.upvals[i] = stack.closure.upvals[uvIdx]
		}
	}
	stack.push(c)
}

func (state *luaState) CloseUpvalues(a int) {
	for uvIdx, upval := range state.stack.openuvs {
		if uvIdx+1 >= a {
			val := *upval.val
			upval.val = &val
			delete(state.stack.openuvs, uvIdx)
		}
	}
}

func (state *luaState) getTable(t, k luaValue, raw bool) api.LuaType {
	if t, ok := t.(*luaTable); ok {
		v := t.get(k)
		if v != nil || raw || !t.hasMetafield("__index") {
			state.stack.push(v)
			return typeOf(v)
		}
	}

	if !raw {
		if mf := getMetafield(t, "__index", state); mf != nil {
			switch x := mf.(type) {
			case *luaTable:
				return state.getTable(x, k, false)
			case *luaClosure:
				state.stack.check(3)
				state.stack.push(x)
				state.stack.push(t)
				state.stack.push(k)
				state.Call(2, 1)
				return typeOf(state.stack.get(-1))
			}
		}
	}

	panic("index error")
}

func (state *luaState) setTable(t, k, v luaValue, raw bool) {
	if t, ok := t.(*luaTable); ok {
		if raw || t.get(k) != nil || !t.hasMetafield("__newindex") {
			t.put(k, v)
			return
		}
	}

	if !raw {
		if mf := getMetafield(t, "__newindex", state); mf != nil {
			switch x := mf.(type) {
			case *luaTable:
				state.setTable(x, k, v, false)
				return
			case *luaClosure:
				state.stack.check(4)
				state.stack.push(x)
				state.stack.push(t)
				state.stack.push(k)
				state.stack.push(v)
				state.Call(3, 0)
				return
			}
		}
	}

	panic("index error")
}

func (state *luaState) pushLuaStack(stack *luaStack) {
	stack.prev = state.stack
	state.stack = stack
}

func (state *luaState) popLuaStack() {
	stack := state.stack
	state.stack = stack.prev
	stack.prev = nil
}

func (state *luaState) callLuaClosure(nArgs, nResults int, closure *luaClosure) {
	nRegs := int(closure.proto.MaxStackSize)
	nParams := int(closure.proto.NumParams)
	isVararg := closure.proto.IsVararg != 0

	newStack := newLuaStack(nRegs+api.LUA_MINSTACK, state)
	newStack.closure = closure

	args := state.stack.popN(nArgs + 1)[1:]
	newStack.pushN(args, nParams)
	newStack.top = nRegs
	if nArgs > nParams && isVararg {
		newStack.varargs = args[nParams:]
	}

	state.pushLuaStack(newStack)
	state.runLuaClosure()
	state.popLuaStack()

	if nResults != 0 {
		results := newStack.popN(newStack.top - nRegs)
		if nResults < 0 {
			nResults = len(results)
		}
		state.stack.check(nResults)
		state.stack.pushN(results, nResults)
	}
}

func (state *luaState) callGoClosure(nArgs, nResults int, closure *luaClosure) {
	newStack := newLuaStack(nArgs+api.LUA_MINSTACK, state)
	newStack.closure = closure

	args := state.stack.popN(nArgs + 1)[1:]
	newStack.pushN(args, nArgs)

	state.pushLuaStack(newStack)
	r := closure.goFun(state)
	state.popLuaStack()

	if nResults != 0 {
		results := newStack.popN(r)
		if nResults < 0 {
			nResults = len(results)
		}
		state.stack.check(nResults)
		state.stack.pushN(results, nResults)
	}
}

func (state *luaState) runLuaClosure() {
	for {
		inst := vm.Instruction(state.Fetch())
		inst.Execute(state)
		if inst.Opcode() == vm.OP_RETURN {
			break
		}
	}
}
