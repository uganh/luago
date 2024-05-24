package main

import (
	"fmt"
	"luago/api"
	"luago/binary"
	"luago/state"
	"luago/vm"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			panic(err)
		}
		proto := binary.Parse(data)
		luaMain(proto)
	}
}

func luaMain(proto *binary.Prototype) {
	const spaces = "        "
	nRegs := int(proto.MaxStackSize)
	ls := state.New(nRegs+8, proto)
	ls.SetTop(nRegs)
	for {
		pc := ls.PC()
		inst := vm.Instruction(ls.Fetch())
		if inst.Opcode() == vm.OP_RETURN {
			break
		}
		inst.Execute(ls)

		name := inst.Name()
		fmt.Printf("[%02d] %s%s ", pc+1, name, spaces[len(name):])
		printStack(ls)
	}
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
