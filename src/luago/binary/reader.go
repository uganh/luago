package binary

import (
	"encoding/binary"
	"math"
)

type reader struct {
	data []byte
}

func (self *reader) readByte() byte {
	b := self.data[0]
	self.data = self.data[1:]
	return b
}

func (self *reader) readUint32() uint32 {
	i := binary.LittleEndian.Uint32(self.data)
	self.data = self.data[4:]
	return i
}

func (self *reader) readUint64() uint64 {
	i := binary.LittleEndian.Uint64(self.data)
	self.data = self.data[8:]
	return i
}

func (self *reader) readLuaInteger() int64 {
	return int64(self.readUint64())
}

func (self *reader) readLuaNumber() float64 {
	return math.Float64frombits(self.readUint64())
}

func (self *reader) readString() string {
	n := uint64(self.readByte())
	if n == 0 {
		return ""
	}
	if n == 0xff {
		n = self.readUint64()
	}
	return string(self.readBytes(n - 1))
}

func (self *reader) readBytes(n uint64) []byte {
	bytes := self.data[:n]
	self.data = self.data[n:]
	return bytes
}

func (self *reader) checkHeader() {
	if string(self.readBytes(4)) != LUA_SIGNATURE {
		panic("not a precompiled chunk")
	} else if self.readByte() != LUAC_VERSION {
		panic("version mismatch")
	} else if self.readByte() != LUAC_FORMAT {
		panic("format mismatch")
	} else if string(self.readBytes(6)) != LUAC_DATA {
		panic("corrupted")
	} else if self.readByte() != CINT_SIZE {
		panic("int size missmatch")
	} else if self.readByte() != CSIZET_SIZE {
		panic("size_t size missmatch")
	} else if self.readByte() != INSTRUCTION_SIZE {
		panic("instruction size missmatch")
	} else if self.readByte() != LUA_INTEGER_SIZE {
		panic("lua_Integer size missmatch")
	} else if self.readByte() != LUA_NUMBER_SIZE {
		panic("lua_Number size missmatch")
	} else if self.readLuaInteger() != LUAC_INT {
		panic("endianness missmatch")
	} else if self.readLuaNumber() != LUAC_NUM {
		panic("float format missmatch")
	}
}

func (self *reader) readProto(parentSource string) *Prototype {
	source := self.readString();
	if source == "" {
		source = parentSource
	}
	
	proto := Prototype{
		Source: source,
		LineBegin: self.readUint32(),
		LineEnd: self.readUint32(),
		NumParams: self.readByte(),
		IsVararg: self.readByte() != 0,
		MaxStackSize: self.readByte(),
	}
	
	proto.Code = make([]uint32, self.readUint32())
	for i := range proto.Code {
		proto.Code[i] = self.readUint32()
	}

	proto.Constants = make([]interface{}, self.readUint32())
	for i := range proto.Constants {
		switch self.readByte() {
		case TAG_NIL:
			proto.Constants[i] = nil
		case TAG_BOOLEAN:
			proto.Constants[i] = self.readByte() != 0
		case TAG_NUMBER:
			proto.Constants[i] = self.readLuaNumber()
		case TAG_INTEGER:
			proto.Constants[i] = self.readLuaInteger()
		case TAG_SHORT_STRING, TAG_LONG_STRING:
			proto.Constants[i] = self.readString()
		}
	}

	proto.Upvalues = make([]Upvalue, self.readUint32())
	for i := range proto.Upvalues {
		proto.Upvalues[i] = Upvalue{
			InStack: self.readByte(),
			Index: self.readByte(),
		}
	}

	proto.Protos = make([]*Prototype, self.readUint32())
	for i := range proto.Protos {
		proto.Protos[i] = self.readProto(source)
	}

	proto.LineInfo = make([]uint32, self.readUint32())
	for i := range proto.LineInfo {
		proto.LineInfo[i] = self.readUint32()
	}

	proto.LocVars = make([]LocVar, self.readUint32())
	for i := range proto.LocVars {
		proto.LocVars[i] = LocVar{
			VarName: self.readString(),
			StartPC: self.readUint32(),
			EndPC: self.readUint32(),
		}
	}

	proto.UpvalueNames = make([]string, self.readUint32())
	for i := range proto.UpvalueNames {
		proto.UpvalueNames[i] = self.readString()
	}

	return &proto
}
