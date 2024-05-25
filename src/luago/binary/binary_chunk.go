package binary

type binaryChunk struct {
	header
}

type header struct {
	signature       [4]byte
	version         byte
	format          byte
	luacData        [6]byte
	cintSize        byte
	csizetSize      byte
	instructionSize byte
	luaIntegerSize  byte
	luaNumberSize   byte
	luacInt         int64
	luacNum         float64
}

const (
	LUA_SIGNATURE    = "\x1bLua"
	LUAC_VERSION     = 0x53
	LUAC_FORMAT      = 0
	LUAC_DATA        = "\x19\x93\r\n\x1a\n"
	CINT_SIZE        = 4
	CSIZET_SIZE      = 8
	INSTRUCTION_SIZE = 4
	LUA_INTEGER_SIZE = 8
	LUA_NUMBER_SIZE  = 8
	LUAC_INT         = 0x5678
	LUAC_NUM         = 370.5
)

type Prototype struct {
	Source       string
	LineBegin    uint32
	LineEnd      uint32
	NumParams    byte
	IsVararg     byte
	MaxStackSize byte
	Code         []uint32
	Constants    []interface{}
	Upvalues     []Upvalue
	Protos       []*Prototype
	LineInfo     []uint32
	LocVars      []LocVar
	UpvalueNames []string
}

const (
	TAG_NIL          = 0x00
	TAG_BOOLEAN      = 0x01
	TAG_NUMBER       = 0x03
	TAG_INTEGER      = 0x13
	TAG_SHORT_STRING = 0x04
	TAG_LONG_STRING  = 0x14
)

type Upvalue struct {
	InStack byte
	Index   byte
}

type LocVar struct {
	VarName string
	StartPC uint32
	EndPC   uint32
}

func Parse(data []byte) *Prototype {
	reader := &reader{data}
	reader.checkHeader()
	reader.readByte()
	return reader.readProto("")
}
