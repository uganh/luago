package main

import (
	"fmt"
	"os"

	"luago/binary"
	"luago/vm"
)

func main() {
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			panic(err)
		}
		list(binary.Parse(data))
	}
}

func list(proto *binary.Prototype) {
	funcType := "main"
	if proto.LineBegin > 0 {
		funcType = "function"
	}

	varargFlag := ""
	if proto.IsVararg {
		varargFlag = "+"
	}

	spaces := "        "

	fmt.Printf("\n%s <%s:%d.%d> (%d instructions)\n", funcType, proto.Source, proto.LineBegin, proto.LineEnd, len(proto.Code))
	fmt.Printf("%d%s params, %d slots, %d upvalues, %d locals, %d constants, %d functions\n", proto.NumParams, varargFlag, proto.MaxStackSize, len(proto.Upvalues), len(proto.LocVars), len(proto.Constants), len(proto.Protos))
	for i, c := range proto.Code {
		line := "-"
		if len(proto.LineInfo) > 0 {
			line = fmt.Sprintf("%d", proto.LineInfo[i])
		}

		inst := vm.Instruction(c)
		name := inst.Name()


		fmt.Printf("\t%d\t[%s]\t%s%s\t", i + 1, line, name, spaces[len(name):])

		switch inst.Mode() {
		case vm.IABC:
			a, b, c := inst.ABC()
			fmt.Printf("%d", a)
			if inst.BMode() != vm.OpArgN {
				if b > 0xff {
					fmt.Printf(" %d", -1 - (b & 0xff))
				} else {
					fmt.Printf(" %d", b)
				}
			}
			if inst.CMode() != vm.OpArgN {
				if c > 0xff {
					fmt.Printf(" %d", -1 - (c & 0xff))
				} else {
					fmt.Printf(" %d", c)
				}
			}
		case vm.IABx:
			a, bx := inst.ABx()
			fmt.Printf("%d", a)
			if inst.BMode() == vm.OpArgK {
				fmt.Printf(" %d", -1 - bx)
			} else if inst.BMode() == vm.OpArgU {
				fmt.Printf(" %d", bx)
			}
		case vm.IAsBx:
			a, sbx := inst.AsBx()
			fmt.Printf("%d %d", a, sbx)
		case vm.IAx:
			ax := inst.Ax()
			fmt.Printf("%d", -1 - ax)
		}

		fmt.Println()
	}
	
	fmt.Printf("constants (%d):\n", len(proto.Constants))
	for i, k := range proto.Constants {
		fmt.Printf("\t%d\t", i + 1)
		switch k := k.(type) {
		case nil:
			fmt.Print("nil")
		case bool:
			fmt.Printf("%t", k)
		case float64:
			fmt.Printf("%g", k)
		case int64:
			fmt.Printf("%d", k)
		case string:
			fmt.Printf("%q", k)
		default:
			fmt.Print("?")
		}
		fmt.Println()
	}

	fmt.Printf("locals (%d):\n", len(proto.LocVars))
	for i, locVar := range proto.LocVars {
		fmt.Printf("\t%d\t%s\t%d\t%d\n", i + 1, locVar.VarName, locVar.StartPC, locVar.EndPC)
	}

	fmt.Printf("upvalues (%d):\n", len(proto.Upvalues))
	for i, upvalue := range proto.Upvalues {
		name := "-"
		if len(proto.UpvalueNames) > 0 {
			name = proto.UpvalueNames[i]
		}
		fmt.Printf("\t%d\t%s\t%d\t%d\n", i + 1, name, upvalue.InStack, upvalue.Index)
	}
}