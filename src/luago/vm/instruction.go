package vm

import (
	"luago/api"
)

const LFIELDS_PER_FLUSH = 50

type Instruction uint32

func (inst Instruction) Opcode() int {
	return int(inst & 0x3f)
}

func (inst Instruction) ABC() (a, b, c int) {
	a = int((inst >> 6) & 0xff)
	c = int((inst >> 14) & 0x1ff)
	b = int((inst >> 23) & 0x1ff)
	return
}

func (inst Instruction) ABx() (a, bx int) {
	a = int((inst >> 6) & 0xff)
	bx = int(inst >> 14)
	return
}

func (inst Instruction) AsBx() (a, sbx int) {
	a = int((inst >> 6) & 0xff)
	sbx = int(inst>>14) - (1 << 17) + 1
	return
}

func (inst Instruction) Ax() (ax int) {
	ax = int(inst >> 6)
	return
}

func (inst Instruction) Name() string {
	return opcodes[inst.Opcode()].name
}

func (inst Instruction) Mode() byte {
	return opcodes[inst.Opcode()].mode
}

func (inst Instruction) BMode() byte {
	return opcodes[inst.Opcode()].argBMode
}

func (inst Instruction) CMode() byte {
	return opcodes[inst.Opcode()].argCMode
}

func (inst Instruction) Execute(vm api.LuaVM) {
	switch inst.Opcode() {
	case OP_MOVE: // R(A) := R(B)
		a, b, _ := inst.ABC()
		a += 1
		b += 1

		vm.Copy(b, a)

	case OP_LOADK: // R(A) := Kst(Bx)
		a, bx := inst.ABx()
		a += 1

		vm.GetConst(bx)
		vm.Replace(a)
	case OP_LOADKX:
		a, _ := inst.ABx()
		a += 1
		ax := Instruction(vm.Fetch()).Ax()

		vm.GetConst(ax)
		vm.Replace(a)
	case OP_LOADBOOL: // R(A) := (bool)B; if (C) pc++
		a, b, c := inst.ABC()
		a += 1

		vm.PushBoolean(b != 0)
		vm.Replace(a)
		if c != 0 {
			vm.AddPC(1)
		}
	case OP_LOADNIL: // R(A), R(A+1), ..., R(A+B) := nil
		a, b, _ := inst.ABC()
		a += 1

		vm.PushNil()
		for i := 0; i <= b; i++ {
			vm.Copy(-1, a+i)
		}
		vm.Pop(1)

	case OP_GETTABLE: // R(A) := R(B)[RK(C)]
		a, b, c := inst.ABC()
		a += 1
		b += 1

		_getRK(vm, c)
		vm.GetTable(b)
		vm.Replace(a)

	case OP_SETTABLE: // R(A)[RK(B)] := RK(C)
		a, b, c := inst.ABC()
		a += 1

		_getRK(vm, b)
		_getRK(vm, c)
		vm.SetTable(a)

	case OP_NEWTABLE: // R(A) := {} (size = B,C)
		a, b, c := inst.ABC()
		a++

		vm.CreateTable(FPB2Int(b), FPB2Int(c))
		vm.Replace(a)

	case OP_ADD:
		_binaryArith(inst, vm, api.LUA_OPADD)
	case OP_SUB:
		_binaryArith(inst, vm, api.LUA_OPSUB)
	case OP_MUL:
		_binaryArith(inst, vm, api.LUA_OPMUL)
	case OP_MOD:
		_binaryArith(inst, vm, api.LUA_OPMOD)
	case OP_POW:
		_binaryArith(inst, vm, api.LUA_OPPOW)
	case OP_DIV:
		_binaryArith(inst, vm, api.LUA_OPDIV)
	case OP_IDIV:
		_binaryArith(inst, vm, api.LUA_OPIDIV)
	case OP_BAND:
		_binaryArith(inst, vm, api.LUA_OPBAND)
	case OP_BOR:
		_binaryArith(inst, vm, api.LUA_OPBOR)
	case OP_BXOR:
		_binaryArith(inst, vm, api.LUA_OPBXOR)
	case OP_SHL:
		_binaryArith(inst, vm, api.LUA_OPSHL)
	case OP_SHR:
		_binaryArith(inst, vm, api.LUA_OPSHR)
	case OP_UNM:
		_unaryArith(inst, vm, api.LUA_OPUNM)
	case OP_BNOT:
		_unaryArith(inst, vm, api.LUA_OPBNOT)
	case OP_NOT: // R(A) := not R(B)
		a, b, _ := inst.ABC()
		a += 1
		b += 1

		vm.PushBoolean(!vm.ToBoolean(b))
		vm.Replace(a)
	case OP_LEN: // R(A) := length of R(B)
		a, b, _ := inst.ABC()
		a += 1
		b += 1

		vm.Len(b)
		vm.Replace(a)
	case OP_CONCAT: // R(A) := R(B) .. ... .. R(C)
		a, b, c := inst.ABC()
		a += 1
		b += 1
		c += 1

		n := c - b + 1
		vm.CheckStack(n)
		for i := b; i <= c; i++ {
			vm.PushValue(i)
		}
		vm.Concat(n)
		vm.Replace(a)
	case OP_JMP:
		a, sbx := inst.AsBx()

		vm.AddPC(sbx)
		if a != 0 {
			panic(a) // TODO
		}
	case OP_EQ:
		_compare(inst, vm, api.LUA_OPEQ)
	case OP_LT:
		_compare(inst, vm, api.LUA_OPLT)
	case OP_LE:
		_compare(inst, vm, api.LUA_OPLE)
	case OP_TEST: // if not (R(A) <=> C) then pc++
		a, _, c := inst.ABC()
		a += 1

		if vm.ToBoolean(a) != (c != 0) {
			vm.AddPC(1)
		}
	case OP_TESTSET: // if (R(B) <=> C) then R(A) := R(B) else pc++
		a, b, c := inst.ABC()
		a += 1
		b += 1

		if vm.ToBoolean(b) == (c != 0) {
			vm.Copy(b, a)
		} else {
			vm.AddPC(1)
		}

	case OP_FORLOOP: // R(A) += R(A+2); if R(A) <?= R(A+1) then { pc += sBx; R(A+3) := R(A) }
		a, sbx := inst.AsBx()
		a += 1

		vm.PushValue(a)
		vm.PushValue(a + 2)
		vm.Arith(api.LUA_OPADD)
		vm.Replace(a)

		step := vm.ToNumber(a + 2)
		if step >= 0 && vm.Compare(a, a+1, api.LUA_OPLE) || step < 0 && vm.Compare(a+1, a, api.LUA_OPLE) {
			vm.AddPC(sbx)
			vm.Copy(a, a+3)
		}
	case OP_FORPREP: // R(A) -= R(A+2); pc += sBx
		a, sbx := inst.AsBx()
		a += 1

		vm.PushValue(a)
		vm.PushValue(a + 2)
		vm.Arith(api.LUA_OPSUB)
		vm.Replace(a)
		vm.AddPC(sbx)

	case OP_SETLIST: // R(A)[(C-1)*FPF+i] := R(A+i), 1 <= i <= B
		a, b, c := inst.ABC()
		a += 1

		if c > 0 {
			c = c - 1
		} else {
			c = Instruction(vm.Fetch()).Ax()
		}
		idx := c * LFIELDS_PER_FLUSH
		for i := 1; i <= b; i++ {
			vm.PushValue(a + i)
			vm.SetI(a, int64(idx+i))
		}

	default:
		panic(inst.Name())
	}
}

// TODO: R(A) := RK(B) op RK(C)
func _binaryArith(inst Instruction, vm api.LuaVM, op api.ArithOp) {
	a, b, c := inst.ABC()
	a += 1

	_getRK(vm, b)
	_getRK(vm, c)
	vm.Arith(op)
	vm.Replace(a)
}

// TODO: R(A) := op R(B)
func _unaryArith(inst Instruction, vm api.LuaVM, op api.ArithOp) {
	a, b, _ := inst.ABC()
	a += 1
	b += 1

	vm.PushValue(b)
	vm.Arith(op)
	vm.Replace(a)
}

// TODO: if ((RK(B) op RK(C) ~= A)) then pc++
func _compare(inst Instruction, vm api.LuaVM, op api.CompareOp) {
	a, b, c := inst.ABC()

	_getRK(vm, b)
	_getRK(vm, c)
	if vm.Compare(-2, -1, op) != (a != 0) {
		vm.AddPC(1)
	}
	vm.Pop(2)
}

func _getRK(vm api.LuaVM, idx int) {
	if idx > 0xff {
		vm.GetConst(idx & 0xff)
	} else {
		vm.PushValue(idx + 1)
	}
}
