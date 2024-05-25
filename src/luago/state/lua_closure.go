package state

import "luago/binary"

type luaClosure struct {
	proto *binary.Prototype
}

func newLuaClosure(proto *binary.Prototype) *luaClosure {
	return &luaClosure{
		proto: proto,
	}
}
