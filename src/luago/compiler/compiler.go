package compiler

import "luago/binary"

func Compile(chunk, chunkName string) *binary.Prototype {
	return GenProto(Parse(chunk, chunkName))
}
