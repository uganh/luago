package main

import (
	"fmt"
	"luago/api"
	"luago/state"
)

func main() {
	ls := state.New()

	ls.PushBoolean(true)
	printStack(ls) // [true]
	ls.PushInteger(10)
	printStack(ls) // [true][10]
	ls.PushNil()
	printStack(ls) // [true][10][nil]
	ls.PushString("hello")
	printStack(ls) // [true][10][nil]["hello"]
	ls.PushValue(-4)
	printStack(ls) // [true][10][nil]["hello"][true]
	ls.Replace(3)
	printStack(ls) // [true][10][true]["hello"]
	ls.SetTop(6)
	printStack(ls) // [true][10][true]["hello"][nil][nil]
	ls.Remove(-3)
	printStack(ls) // [true][10][true][nil][nil]
	ls.SetTop(-5)
	printStack(ls) // [true]
}

func printStack(ls api.LuaState) {
	top := ls.GetTop()
	for idx := 1; idx <= top; idx++ {
		t := ls.Type(idx)
		switch t {
		case api.LUA_TBOOLEAN:
			fmt.Printf("[%t]", ls.ToBoolean(idx))
		case api.LUA_TNUMBER:
			fmt.Printf("[%g]", ls.ToNumber(idx))
		case api.LUA_TSTRING:
			fmt.Printf("[%q]", ls.ToString(idx))
		default:
			fmt.Printf("[%s]", ls.TypeName(t))
		}
	}
	fmt.Println()
}
