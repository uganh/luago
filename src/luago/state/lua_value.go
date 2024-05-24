package state

import (
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
