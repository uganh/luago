package vm

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
	sbx = int(inst>>14) - (1 << 17) - 1
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
