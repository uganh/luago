package main

import (
	"encoding/json"
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

		ast := compiler.Parse(string(chunk), chunkName)
		val, err := json.Marshal(ast)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(val))
	}
}
