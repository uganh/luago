package state

import (
	"fmt"
	"luago/api"
	"luago/binary"
	"luago/number"
	"math"
	"strings"
)

type luaState struct {
	stack *luaStack
	proto *binary.Prototype
	pc    int
}

func New(stackSize int, proto *binary.Prototype) *luaState {
	return &luaState{
		stack: newLuaStack(stackSize),
		proto: proto,
		pc:    0,
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

func (state *luaState) ToBoolean(idx int) bool {
	val := state.stack.get(idx)
	switch x := val.(type) {
	case nil:
		return false
	case bool:
		return x
	default:
		return true
	}
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

func (state *luaState) Arith(op api.ArithOp) {
	var a, b, r luaValue
	b = state.stack.pop()
	if op != api.LUA_OPUNM && op != api.LUA_OPBNOT {
		a = state.stack.pop()
	} else {
		a = b
	}

	var iFunc func(int64, int64) int64
	var fFunc func(float64, float64) float64

	switch op {
	case api.LUA_OPADD:
		iFunc = func(a, b int64) int64 { return a + b }
		fFunc = func(a, b float64) float64 { return a + b }
	case api.LUA_OPSUB:
		iFunc = func(a, b int64) int64 { return a - b }
		fFunc = func(a, b float64) float64 { return a - b }
	case api.LUA_OPMUL:
		iFunc = func(a, b int64) int64 { return a * b }
		fFunc = func(a, b float64) float64 { return a * b }
	case api.LUA_OPMOD:
		iFunc, fFunc = number.IMod, number.FMod
	case api.LUA_OPPOW:
		fFunc = math.Pow
	case api.LUA_OPDIV:
		fFunc = func(a, b float64) float64 { return a / b }
	case api.LUA_OPIDIV:
		iFunc, fFunc = number.IFloorDiv, number.FFloorDiv
	case api.LUA_OPBAND:
		iFunc = func(a, b int64) int64 { return a & b }
	case api.LUA_OPBOR:
		iFunc = func(a, b int64) int64 { return a | b }
	case api.LUA_OPBXOR:
		iFunc = func(a, b int64) int64 { return a ^ b }
	case api.LUA_OPSHL:
		iFunc = number.ShiftLeft
	case api.LUA_OPSHR:
		iFunc = number.ShiftRight
	case api.LUA_OPUNM:
		iFunc = func(a, _ int64) int64 { return -a }
		fFunc = func(a, _ float64) float64 { return -a }
	case api.LUA_OPBNOT:
		iFunc = func(a, _ int64) int64 { return ^a }
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
					goto end
				}
			}
		}

		if a, ok := convertToFloat(a); ok {
			if b, ok := convertToFloat(b); ok {
				r = fFunc(a, b)
			}
		}
	}

end:
	if r != nil {
		state.stack.push(r)
	} else {
		panic("arithmetic error")
	}
}

func (state *luaState) Compare(idx1, idx2 int, op api.CompareOp) bool {
	a := state.stack.get(idx1)
	b := state.stack.get(idx2)
	switch op {
	case api.LUA_OPEQ:
		switch a := a.(type) {
		case nil:
			return b == nil
		case bool:
			b, ok := b.(bool)
			return ok && a == b
		case string:
			b, ok := b.(string)
			return ok && a == b
		case int64:
			switch b := b.(type) {
			case int64:
				return a == b
			case float64:
				return float64(a) == b
			default:
				return false
			}
		case float64:
			switch b := b.(type) {
			case float64:
				return a == b
			case int64:
				return a == float64(b)
			default:
				return false
			}
		default:
			return a == b // TODO
		}
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
	default:
		panic("invalid compare op")
	}
	panic("comparison error")
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
	return state.getTable(t, k)
}

func (state *luaState) GetField(idx int, k string) api.LuaType {
	return state.getTable(state.stack.get(idx), k)
}

func (state *luaState) GetI(idx int, i int64) api.LuaType {
	return state.getTable(state.stack.get(idx), i)
}

func (state *luaState) SetTable(idx int) {
	t := state.stack.get(idx)
	v := state.stack.pop()
	k := state.stack.pop()
	state.setTable(t, k, v)
}

func (state *luaState) SetField(idx int, k string) {
	t := state.stack.get(idx)
	v := state.stack.pop()
	state.setTable(t, k, v)
}

func (state *luaState) SetI(idx int, i int64) {
	t := state.stack.get(idx)
	v := state.stack.pop()
	state.setTable(t, i, v)
}

func (state *luaState) Len(idx int) {
	val := state.stack.get(idx)
	if s, ok := val.(string); ok {
		state.stack.push(int64(len(s)))
	} else if t, ok := val.(*luaTable); ok {
		state.stack.push(int64(t.len()))
	} else {
		panic("length error") // TODO
	}
}

func (state *luaState) Concat(n int) {
	if n == 0 {
		state.stack.push("")
	} else if n >= 2 {
		l := make([]string, n)
		for n > 0 {
			if state.IsString(-1) {
				n--
				l[n] = state.ToString(-1)
				state.stack.pop()
			}
		}
		if n > 0 {
			panic("concatenation error")
		}
		state.stack.push(strings.Join(l, ""))
	}
}

func (state *luaState) PC() int {
	return state.pc
}

func (state *luaState) AddPC(n int) {
	state.pc += n
}

func (state *luaState) Fetch() uint32 {
	c := state.proto.Code[state.pc]
	state.pc++
	return c
}

func (state *luaState) GetConst(idx int) {
	state.stack.push(state.proto.Constants[idx])
}

func (state *luaState) getTable(t, k luaValue) api.LuaType {
	if t, ok := t.(*luaTable); ok {
		v := t.get(k)
		state.stack.push(v)
		return typeOf(v)
	}
	panic("not a table") // TODO
}

func (state *luaState) setTable(t, k, v luaValue) {
	if t, ok := t.(*luaTable); ok {
		t.put(k, v)
		return
	}
	panic("not a table") // TODO
}
