package state

import (
	"luago/api"
	"luago/binary"
)

type luaClosure struct {
	proto *binary.Prototype
	goFun api.GoFunction
}

func newLuaClosure(proto *binary.Prototype) *luaClosure {
	return &luaClosure{
		proto: proto,
	}
}

func newGoClosure(goFun api.GoFunction) *luaClosure {
	return &luaClosure{
		goFun: goFun,
	}
}
