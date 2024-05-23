package main

import (
	"fmt"
	"luago/api"
	"luago/state"
)

func main() {
	ls := state.New()

	ls.PushInteger(1)
	ls.PushString("2.0")
	ls.PushString("3.0")
	ls.PushNumber(4.0)
	printStack(ls) // [1]["2.0"]["3.0"][4]

	ls.Arith(api.LUA_OPADD)
	printStack(ls) // [1]["2.0"][7]

	ls.Arith(api.LUA_OPBNOT)
	printStack(ls) // [1]["2.0"][-8]

	ls.Len(2)
	printStack(ls) // [1]["2.0"][-8][3]

	ls.Concat(3)
	printStack(ls) // [1]["2.0-83"]

	ls.PushBoolean(ls.Compare(1, 2, api.LUA_OPEQ))
	printStack(ls) // [1]["2.0-83"][false]
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
