package state

import "luago/api"

type luaValue interface{}

func typeOf(v luaValue) api.LuaType {
	switch v.(type) {
	case nil: return api.LUA_TNIL
	case bool: return api.LUA_TBOOLEAN
	case int64, float64: return api.LUA_TNUMBER
	case string: return api.LUA_TSTRING
	default: panic(v)
	}
}
