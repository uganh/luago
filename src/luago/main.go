package main

import (
	"fmt"
	"luago/api"
	"luago/state"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			panic(err)
		}
		ls := state.New()
		ls.Register("print", print)
		ls.Register("getmetatable", getMetatable)
		ls.Register("setmetatable", setMetatable)
		ls.Register("next", next)
		ls.Register("pairs", pairs)
		ls.Register("ipairs", ipairs)
		ls.Register("error", error_)
		ls.Register("pcall", pcall)
		ls.Load(data, os.Args[1], "b")
		ls.Call(0, 0)
	}
}

func print(ls api.LuaState) int {
	nArgs := ls.GetTop()
	for i := 1; i <= nArgs; i++ {
		if ls.IsBoolean(i) {
			fmt.Printf("%t", ls.ToBoolean(i))
		} else if ls.IsString(i) {
			fmt.Print(ls.ToString(i))
		} else {
			fmt.Print(ls.TypeName(ls.Type(i)))
		}
		if i < nArgs {
			fmt.Print(" ")
		}
	}
	fmt.Println()
	return 0
}

func getMetatable(ls api.LuaState) int {
	if !ls.GetMetatable(1) {
		ls.PushNil()
	}
	return 1
}

func setMetatable(ls api.LuaState) int {
	ls.SetMetatable(1)
	return 1
}

func next(ls api.LuaState) int {
	ls.SetTop(2)
	if ls.Next(1) {
		return 2
	} else {
		ls.PushNil()
		return 1
	}
}

func inext(ls api.LuaState) int {
	i := ls.ToInteger(2) + 1
	ls.PushInteger(i)
	if ls.GetI(1, i) == api.LUA_TNIL {
		return 1
	} else {
		return 2
	}
}

func pairs(ls api.LuaState) int {
	ls.PushGoFunction(next)
	ls.PushValue(1)
	ls.PushNil()
	return 3
}

func ipairs(ls api.LuaState) int {
	ls.PushGoFunction(inext)
	ls.PushValue(1)
	ls.PushInteger(0)
	return 3
}

func error_(ls api.LuaState) int {
	return ls.Error()
}

func pcall(ls api.LuaState) int {
	nArgs := ls.GetTop() - 1
	ls.PushBoolean(ls.PCall(nArgs, -1, 0) == api.LUA_OK)
	ls.Insert(1)
	return ls.GetTop()
}
