package main

import (
	"fmt"
	"luago/compiler"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		chunkName := os.Args[1]
		chunk, err := os.ReadFile(chunkName)
		if err != nil {
			panic(err)
		}

		lexer := compiler.NewLexer(string(chunk), chunkName)
		for {
			line, kind, token := lexer.Lex()

			fmt.Printf("[%2d] ", line)
			switch {
			case kind == compiler.TOKEN_EOF:
				fmt.Printf("end of file\n")
			case kind <= compiler.TOKEN_WHILE:
				fmt.Printf("%s\n", token)
			case kind == compiler.TOKEN_IDENTIFIER:
				fmt.Printf("identifier %s\n", token)
			case kind == compiler.TOKEN_NUMBER:
				fmt.Printf("number %s\n", token)
			case kind == compiler.TOKEN_STRING:
				fmt.Printf("string %q\n", token)
			default:
				fmt.Printf("'%s'\n", token)
			}

			if kind == compiler.TOKEN_EOF {
				break
			}
		}
	}
}
