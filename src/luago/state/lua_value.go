package state

import (
	"fmt"
	"luago/api"
	"luago/number"
)

type luaValue interface{}

func typeOf(v luaValue) api.LuaType {
	switch v.(type) {
	case nil:
		return api.LUA_TNIL
	case bool:
		return api.LUA_TBOOLEAN
	case int64, float64:
		return api.LUA_TNUMBER
	case string:
		return api.LUA_TSTRING
	case *luaTable:
		return api.LUA_TTABLE
	case *luaClosure:
		return api.LUA_TFUNCTION
	default:
		panic(v)
	}
}

func convertToInteger(val luaValue) (int64, bool) {
	switch x := val.(type) {
	case int64:
		return x, true
	case float64:
		return int64(x), true
	case string:
		if i, ok := number.ParseInteger(x); ok {
			return i, ok
		}
		if f, ok := number.ParseFloat(x); ok {
			return number.FloatToInteger(f)
		}
	}
	return 0, false
}

func convertToFloat(val luaValue) (float64, bool) {
	switch x := val.(type) {
	case float64:
		return x, true
	case int64:
		return float64(x), true
	case string:
		return number.ParseFloat(x)
	default:
		return 0.0, false
	}
}

func convertToBoolean(val luaValue) bool {
	switch x := val.(type) {
	case nil:
		return false
	case bool:
		return x
	default:
		return true
	}
}

func getMetatable(val luaValue, state *luaState) *luaTable {
	if t, ok := val.(*luaTable); ok {
		return t.metatable
	}
	if mt := state.registry.get(fmt.Sprintf("_MT%d", typeOf(val))); mt != nil {
		return mt.(*luaTable)
	}
	return nil
}

func setMetatable(val luaValue, mt *luaTable, state *luaState) {
	if t, ok := val.(*luaTable); ok {
		t.metatable = mt
	} else {
		state.registry.put(fmt.Sprintf("_MT%d", typeOf(val)), mt)
	}
}

func getMetafield(val luaValue, name string, state *luaState) luaValue {
	if mt := getMetatable(val, state); mt != nil {
		return mt.get(name)
	}
	return nil
}

func callMetamethod(a, b luaValue, mName string, state *luaState) (luaValue, bool) {
	var mm luaValue
	if mm = getMetafield(a, mName, state); mm == nil {
		if mm = getMetafield(b, mName, state); mm == nil {
			return nil, false
		}
	}

	state.stack.check(3)
	state.stack.push(mm)
	state.stack.push(a)
	state.stack.push(b)
	state.Call(2, 1)
	return state.stack.pop(), true
}

func equal(a, b luaValue, state *luaState) bool {
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
	case *luaTable:
		if b, ok := b.(*luaTable); ok && a != b && state != nil {
			if r, ok := callMetamethod(a, b, "__eq", state); ok {
				return convertToBoolean(r)
			}
		}
	}
	return a == b
}
