package api

type LuaVM interface {
	LuaState
	AddPC(n int)
	Fetch() uint32
	GetConst(idx int)
	RegisterCount() int
	LoadVararg(n int)
	LoadProto(idx int)
	CloseUpvalues(a int)
}
