package state

import (
	"luago/api"
	"luago/binary"
)

type upvalue struct {
	val *luaValue
}

type luaClosure struct {
	proto  *binary.Prototype
	goFun  api.GoFunction
	upvals []*upvalue
}

func newLuaClosure(proto *binary.Prototype) *luaClosure {
	closure := &luaClosure{proto: proto}
	if nUpvals := len(proto.Upvalues); nUpvals > 0 {
		closure.upvals = make([]*upvalue, nUpvals)
	}
	return closure
}

func newGoClosure(goFun api.GoFunction, nUpvals int) *luaClosure {
	closure := &luaClosure{goFun: goFun}
	if nUpvals > 0 {
		closure.upvals = make([]*upvalue, nUpvals)
	}
	return closure
}
