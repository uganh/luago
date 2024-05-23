package state

import (
	"fmt"
	"luago/api"
)

type luaState struct {
	stack *luaStack
}

func New() *luaState {
	return &luaState{
		stack: newLuaStack(20),
	}
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
	case api.LUA_TNONE: return "no value"
	case api.LUA_TNIL: return "nil"
	case api.LUA_TBOOLEAN: return "boolean"
	case api.LUA_TNUMBER: return "number"
	case api.LUA_TSTRING: return "string"
	case api.LUA_TTABLE: return "table"
	case api.LUA_TFUNCTION: return "function"
	case api.LUA_TTHREAD: return "thread"
	default: return "userdata";
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
	return t == api.LUA_TNIL || t == api.LUA_TNUMBER
}

func (state *luaState) ToBoolean(idx int) bool {
	val := state.stack.get(idx)
	switch x := val.(type) {
	case nil: return false
	case bool: return x
	default: return true
	}
}

func (state *luaState) ToInteger(idx int) int64 {
	n, _ := state.ToIntegerX(idx)
	return n
}

func (state *luaState) ToIntegerX(idx int) (int64, bool) {
	val := state.stack.get(idx)
	i, ok := val.(int64)
	return i, ok
}

func (state *luaState) ToNumber(idx int) float64 {
	n, _ := state.ToNumberX(idx)
	return n
}

func (state *luaState) ToNumberX(idx int) (float64, bool) {
	val := state.stack.get(idx)
	switch x := val.(type) {
	case float64: return x, true
	case int64: return float64(x), true
	default: return 0, false
	}
}

func (state *luaState) ToString(idx int) string {
	s, _ := state.ToStringX(idx)
	return s
}

func (state *luaState) ToStringX(idx int) (string, bool) {
	val := state.stack.get(idx)
	switch x := val.(type) {
	case string: return x, true
	case int64, float64:
		s := fmt.Sprintf("%v", x)
		state.stack.set(idx, s)
		return s, true
	default: return "", false
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