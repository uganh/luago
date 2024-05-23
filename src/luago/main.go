package main

import (
	"fmt"
	"os"

	"luago/binary"
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

	fmt.Printf("\n%s <%s:%d.%d> (%d instructions)\n", funcType, proto.Source, proto.LineBegin, proto.LineEnd, len(proto.Code))
	fmt.Printf("%d%s params, %d slots, %d upvalues, %d locals, %d constants, %d functions\n", proto.NumParams, varargFlag, proto.MaxStackSize, len(proto.Upvalues), len(proto.LocVars), len(proto.Constants), len(proto.Protos))
	for i, c := range proto.Code {
		line := "-"
		if len(proto.LineInfo) > 0 {
			line = fmt.Sprintf("%d", proto.LineInfo[i])
		}
		fmt.Printf("\t%d\t[%s]\t0x%08X\n", i + 1, line, c)
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