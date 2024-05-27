package binary

import (
	"encoding/binary"
	"math"
)

type reader struct {
	data []byte
}

func (reader *reader) readByte() byte {
	b := reader.data[0]
	reader.data = reader.data[1:]
	return b
}

func (reader *reader) readUint32() uint32 {
	i := binary.LittleEndian.Uint32(reader.data)
	reader.data = reader.data[4:]
	return i
}

func (reader *reader) readUint64() uint64 {
	i := binary.LittleEndian.Uint64(reader.data)
	reader.data = reader.data[8:]
	return i
}

func (reader *reader) readLuaInteger() int64 {
	return int64(reader.readUint64())
}

func (reader *reader) readLuaNumber() float64 {
	return math.Float64frombits(reader.readUint64())
}

func (reader *reader) readString() string {
	n := uint64(reader.readByte())
	if n == 0 {
		return ""
	}
	if n == 0xff {
		n = reader.readUint64()
	}
	return string(reader.readBytes(n - 1))
}

func (reader *reader) readBytes(n uint64) []byte {
	bytes := reader.data[:n]
	reader.data = reader.data[n:]
	return bytes
}

func (reader *reader) checkHeader() {
	if string(reader.readBytes(4)) != LUA_SIGNATURE {
		panic("not a precompiled chunk")
	} else if reader.readByte() != LUAC_VERSION {
		panic("version mismatch")
	} else if reader.readByte() != LUAC_FORMAT {
		panic("format mismatch")
	} else if string(reader.readBytes(6)) != LUAC_DATA {
		panic("corrupted")
	} else if reader.readByte() != CINT_SIZE {
		panic("int size missmatch")
	} else if reader.readByte() != CSIZET_SIZE {
		panic("size_t size missmatch")
	} else if reader.readByte() != INSTRUCTION_SIZE {
		panic("instruction size missmatch")
	} else if reader.readByte() != LUA_INTEGER_SIZE {
		panic("lua_Integer size missmatch")
	} else if reader.readByte() != LUA_NUMBER_SIZE {
		panic("lua_Number size missmatch")
	} else if reader.readLuaInteger() != LUAC_INT {
		panic("endianness missmatch")
	} else if reader.readLuaNumber() != LUAC_NUM {
		panic("float format missmatch")
	}
}

func (reader *reader) readProto(parentSource string) *Prototype {
	source := reader.readString()
	if source == "" {
		source = parentSource
	}

	proto := Prototype{
		Source:       source,
		LineBegin:    reader.readUint32(),
		LineEnd:      reader.readUint32(),
		NumParams:    reader.readByte(),
		IsVararg:     reader.readByte(),
		MaxStackSize: reader.readByte(),
	}

	proto.Code = make([]uint32, reader.readUint32())
	for i := range proto.Code {
		proto.Code[i] = reader.readUint32()
	}

	proto.Constants = make([]interface{}, reader.readUint32())
	for i := range proto.Constants {
		switch reader.readByte() {
		case TAG_NIL:
			proto.Constants[i] = nil
		case TAG_BOOLEAN:
			proto.Constants[i] = reader.readByte() != 0
		case TAG_NUMBER:
			proto.Constants[i] = reader.readLuaNumber()
		case TAG_INTEGER:
			proto.Constants[i] = reader.readLuaInteger()
		case TAG_SHORT_STRING, TAG_LONG_STRING:
			proto.Constants[i] = reader.readString()
		}
	}

	proto.Upvalues = make([]Upvalue, reader.readUint32())
	for i := range proto.Upvalues {
		proto.Upvalues[i] = Upvalue{
			InStack: reader.readByte(),
			Index:   reader.readByte(),
		}
	}

	proto.Protos = make([]*Prototype, reader.readUint32())
	for i := range proto.Protos {
		proto.Protos[i] = reader.readProto(source)
	}

	proto.LineInfo = make([]uint32, reader.readUint32())
	for i := range proto.LineInfo {
		proto.LineInfo[i] = reader.readUint32()
	}

	proto.LocVars = make([]LocVar, reader.readUint32())
	for i := range proto.LocVars {
		proto.LocVars[i] = LocVar{
			VarName: reader.readString(),
			StartPC: reader.readUint32(),
			EndPC:   reader.readUint32(),
		}
	}

	proto.UpvalueNames = make([]string, reader.readUint32())
	for i := range proto.UpvalueNames {
		proto.UpvalueNames[i] = reader.readString()
	}

	return &proto
}
