package compiler

import (
	"luago/binary"
	"luago/vm"
)

type funcInfo struct {
	constants   map[interface{}]int
	usedRegs    int
	maxRegs     int
	scopeLv     int
	locVars     []*locVarInfo
	locVarNames map[string]*locVarInfo
	breaks      [][]int
	parent      *funcInfo
	upvalues    map[string]upvalInfo
	insts       []uint32
	children    []*funcInfo
	numParams   int
	isVararg    bool
}

type locVarInfo struct {
	prev     *locVarInfo
	name     string
	scopeLv  int
	slot     int
	captured bool
}

type upvalInfo struct {
	locVarslot int
	upvalIndex int
	index      int
}

func newFuncInfo(parent *funcInfo, numParams int, isVararg bool) *funcInfo {
	return &funcInfo{
		constants:   map[interface{}]int{},
		locVarNames: map[string]*locVarInfo{},
		breaks:      make([][]int, 1),
		parent:      parent,
		upvalues:    map[string]upvalInfo{},
		children:    []*funcInfo{},
		numParams:   numParams,
		isVararg:    isVararg,
	}
}

func (f *funcInfo) indexOfConstant(k interface{}) int {
	if idx, found := f.constants[k]; found {
		return idx
	}
	idx := len(f.constants)
	f.constants[k] = idx
	return idx
}

func (f *funcInfo) allocReg() int {
	f.usedRegs++
	if f.usedRegs >= 255 {
		panic("function or expression needs too many registers")
	}
	if f.usedRegs > f.maxRegs {
		f.maxRegs = f.usedRegs
	}
	return f.usedRegs - 1
}

func (f *funcInfo) freeReg() {
	f.usedRegs--
}

func (f *funcInfo) allocRegs(n int) int {
	for i := 0; i < n; i++ {
		f.allocReg()
	}
	return f.usedRegs - n
}

func (f *funcInfo) freeRegs(n int) {
	for i := 0; i < n; i++ {
		f.freeReg()
	}
}

func (f *funcInfo) enterScope(breakable bool) {
	f.scopeLv++
	if breakable {
		f.breaks = append(f.breaks, []int{})
	} else {
		f.breaks = append(f.breaks, nil)
	}
}

func (f *funcInfo) leaveScope() {
	for name, locVar := range f.locVarNames {
		for locVar != nil && locVar.scopeLv >= f.scopeLv {
			f.freeReg()
			locVar = locVar.prev
		}
		if locVar == nil {
			delete(f.locVarNames, name)
		} else {
			f.locVarNames[name] = locVar
		}
	}

	a := f.getJmpArgA()

	pendingBreaks := f.breaks[f.scopeLv]
	f.breaks = f.breaks[:f.scopeLv]

	for _, pc := range pendingBreaks {
		sbx := f.pc() - pc
		f.insts[pc] = encodeAsBx(vm.OP_JMP, a, sbx)
	}

	f.scopeLv--
}

func (f *funcInfo) addLocVar(name string) int {
	newVar := &locVarInfo{
		prev:    f.locVarNames[name],
		name:    name,
		scopeLv: f.scopeLv,
		slot:    f.allocReg(),
	}
	f.locVars = append(f.locVars, newVar)
	f.locVarNames[name] = newVar
	return newVar.slot
}

func (f *funcInfo) slotOfLocVar(name string) int {
	if locVar, found := f.locVarNames[name]; found {
		return locVar.slot
	}
	return -1
}

func (f *funcInfo) addBreak(pc int) {
	for i := f.scopeLv; i >= 0; i-- {
		if f.breaks[i] != nil {
			f.breaks[i] = append(f.breaks[i], pc)
			return
		}
	}
	panic("<break> at line ? not inside a loop!")
}

func (f *funcInfo) indexOfUpvalue(name string) int {
	if uvInfo, found := f.upvalues[name]; found {
		return uvInfo.index
	}
	if f.parent != nil {
		index := len(f.upvalues)
		if locVar, found := f.parent.locVarNames[name]; found {
			f.upvalues[name] = upvalInfo{locVar.slot, -1, index}
			return index
		}
		if uvIdx := f.parent.indexOfUpvalue(name); uvIdx != -1 {
			f.upvalues[name] = upvalInfo{-1, uvIdx, index}
			return index
		}
	}
	return -1
}

func (f *funcInfo) closeOpenUpvalues() {
	if a := f.getJmpArgA(); a > 0 {
		f.emitJMP(a, 0)
	}
}

func (f *funcInfo) getJmpArgA() int {
	hasCaptureLocVars := false
	minSlotOfLocVars := f.maxRegs
	for _, locVar := range f.locVarNames {
		if locVar.scopeLv == f.scopeLv {
			for v := locVar; v != nil && v.scopeLv == f.scopeLv; v = v.prev {
				if v.captured {
					hasCaptureLocVars = true
				}
				if v.slot < minSlotOfLocVars && v.name[0] != '(' {
					minSlotOfLocVars = v.slot
				}
			}
		}
	}
	if hasCaptureLocVars {
		return minSlotOfLocVars + 1
	} else {
		return 0
	}
}

func (f *funcInfo) pc() int {
	return len(f.insts) - 1
}

func (f *funcInfo) fix(pc, sbx int) {
	f.insts[pc] = (f.insts[pc] << 18 >> 18) | (uint32(sbx+vm.MAXARG_sBx) << 14)
}

func (f *funcInfo) toProto() *binary.Prototype {
	proto := &binary.Prototype{
		NumParams: byte(f.numParams),
		MaxStackSize: byte(f.maxRegs),
		Code: f.insts,
		Constants: make([]interface{}, len(f.constants)),
		Upvalues: make([]binary.Upvalue, len(f.upvalues)),
		Protos: make([]*binary.Prototype, len(f.children)),
		UpvalueNames: make([]string, len(f.upvalues)),
	}

	if f.isVararg {
		proto.IsVararg = 1
	}

	for val, idx := range f.constants {
		proto.Constants[idx] = val
	}

	for name, uv := range f.upvalues {
		if uv.locVarslot >= 0 { // in stack
			proto.Upvalues[uv.index] = binary.Upvalue{InStack: 1, Index: byte(uv.locVarslot)}
		} else {
			proto.Upvalues[uv.index] = binary.Upvalue{InStack: 0, Index: byte(uv.upvalIndex)}
		}
		proto.UpvalueNames[uv.index] = name
	}

	for i, subF := range f.children {
		proto.Protos[i] = subF.toProto()
	}

	return proto
}

func encodeABC(opcode, a, b, c int) uint32 {
	return uint32(b<<23|c<<14|a<<6|opcode)
}

func encodeABx(opcode, a, bx int) uint32 {
	return uint32(bx<<14|a<<6|opcode)
}

func encodeAsBx(opcode, a, sbx int) uint32 {
	return uint32((sbx+vm.MAXARG_sBx)<<14|a<<6|opcode)
}

func encodeAx(opcode, ax int) uint32 {
	return uint32(ax<<6|opcode)
}

func (f *funcInfo) emit(inst uint32) {
	f.insts = append(f.insts, inst)
}

func (f *funcInfo) emitMOVE(a, b int) {
	f.emit(encodeABC(vm.OP_MOVE, a, b, 0))
}

func (f *funcInfo) emitLOADK(a int, k interface{}) {
	if idx := f.indexOfConstant(k); idx <= vm.MAXARG_Bx {
		f.emit(encodeABx(vm.OP_LOADK, a, idx))
	} else {
		f.emit(encodeABx(vm.OP_LOADKX, a, 0))
		f.emit(encodeAx(vm.OP_EXTRAARG, idx))
	}
}

func (f *funcInfo) emitLOADBOOL(a, b, c int) {
	f.emit(encodeABC(vm.OP_MOVE, a, b, c))
}

func (f *funcInfo) emitLOADNIL(a, n int) {
	f.emit(encodeABC(vm.OP_LOADNIL, a, n - 1, 0))
}

func (f *funcInfo) emitGETUPVAL(a, b int) {
	f.emit(encodeABC(vm.OP_GETUPVAL, a, b, 0))
}

func (f *funcInfo) emitGETTABUP(a, b int) {
	f.emit(encodeABC(vm.OP_GETTABUP, a, b, 0))
}

func (f *funcInfo) emitGETTABLE(a, b, c int) {
	f.emit(encodeABC(vm.OP_GETTABLE, a, b, c))
}

func (f *funcInfo) emitSETTABUP(a, b, c int) {
	f.emit(encodeABC(vm.OP_SETTABUP, a, b, c))
}

func (f *funcInfo) emitSETUPVAL(a, b int) {
	f.emit(encodeABC(vm.OP_SETUPVAL, a, b, 0))
}

func (f *funcInfo) emitSETTABLE(a, b, c int) {
	f.emit(encodeABC(vm.OP_SETTABLE, a, b, c))
}

func (f *funcInfo) emitNEWTABLE(a, nArr, nRec int) {
	f.emit(encodeABC(vm.OP_NEWTABLE, a, vm.Int2FPB(nArr), vm.Int2FPB(nRec)))
}

func (f *funcInfo) emitSELF(a, b, c int) {
	f.emit(encodeABC(vm.OP_SELF, a, b, c))
}

func (f *funcInfo) emitADD(a, b, c int) {
	f.emit(encodeABC(vm.OP_ADD, a, b, c))
}

func (f *funcInfo) emitSUB(a, b, c int) {
	f.emit(encodeABC(vm.OP_SUB, a, b, c))
}

func (f *funcInfo) emitMUL(a, b, c int) {
	f.emit(encodeABC(vm.OP_MUL, a, b, c))
}

func (f *funcInfo) emitMOD(a, b, c int) {
	f.emit(encodeABC(vm.OP_MOD, a, b, c))
}

func (f *funcInfo) emitPOW(a, b, c int) {
	f.emit(encodeABC(vm.OP_POW, a, b, c))
}

func (f *funcInfo) emitDIV(a, b, c int) {
	f.emit(encodeABC(vm.OP_DIV, a, b, c))
}

func (f *funcInfo) emitIDIV(a, b, c int) {
	f.emit(encodeABC(vm.OP_IDIV, a, b, c))
}

func (f *funcInfo) emitBAND(a, b, c int) {
	f.emit(encodeABC(vm.OP_BAND, a, b, c))
}

func (f *funcInfo) emitBOR(a, b, c int) {
	f.emit(encodeABC(vm.OP_BOR, a, b, c))
}

func (f *funcInfo) emitBXOR(a, b, c int) {
	f.emit(encodeABC(vm.OP_BXOR, a, b, c))
}

func (f *funcInfo) emitSHL(a, b, c int) {
	f.emit(encodeABC(vm.OP_SHL, a, b, c))
}

func (f *funcInfo) emitSHR(a, b, c int) {
	f.emit(encodeABC(vm.OP_SHR, a, b, c))
}

func (f *funcInfo) emitUNM(a, b int) {
	f.emit(encodeABC(vm.OP_UNM, a, b, 0))
}

func (f *funcInfo) emitBNOT(a, b int) {
	f.emit(encodeABC(vm.OP_BNOT, a, b, 0))
}

func (f *funcInfo) emitNOT(a, b int) {
	f.emit(encodeABC(vm.OP_NOT, a, b, 0))
}

func (f *funcInfo) emitLEN(a, b int) {
	f.emit(encodeABC(vm.OP_LEN, a, b, 0))
}

func (f *funcInfo) emitCONCAT(a, b, c int) {
	f.emit(encodeABC(vm.OP_CONCAT, a, b, c))
}

func (f *funcInfo) emitJMP(a, sbx int) {
	f.emit(encodeAsBx(vm.OP_JMP, a, sbx))
}

func (f *funcInfo) emitEQ(a, b, c int) {
	f.emit(encodeABC(vm.OP_EQ, a, b, c))
}

func (f *funcInfo) emitLT(a, b, c int) {
	f.emit(encodeABC(vm.OP_LT, a, b, c))
}

func (f *funcInfo) emitLE(a, b, c int) {
	f.emit(encodeABC(vm.OP_LE, a, b, c))
}

func (f *funcInfo) emitTEST(a, c int) {
	f.emit(encodeABC(vm.OP_TEST, a, 0, c))
}

func (f *funcInfo) emitTESTSET(a, b, c int) {
	f.emit(encodeABC(vm.OP_TESTSET, a, b, c))
}

func (f *funcInfo) emitCALL(a, nArgs, nResults int) {
	f.emit(encodeABC(vm.OP_CALL, a, nArgs + 1, nResults + 1))
}

func (f *funcInfo) emitTAILCALL(a, nArgs int) {
	f.emit(encodeABC(vm.OP_TAILCALL, a, nArgs + 1, 0))
}

func (f *funcInfo) emitRETURN(a, nResults int) {
	f.emit(encodeABC(vm.OP_RETURN, a, nResults + 1, 0))
}

func (f *funcInfo) emitFORLOOP(a, sbx int) {
	f.emit(encodeAsBx(vm.OP_FORLOOP, a, sbx))
}

func (f *funcInfo) emitFORPREP(a, sbx int) {
	f.emit(encodeAsBx(vm.OP_FORPREP, a, sbx))
}

func (f *funcInfo) emitTFORCALL(a, c int) {
	f.emit(encodeABC(vm.OP_TFORCALL, a, 0, c))
}

func (f *funcInfo) emitTFORLOOP(a, sbx int) {
	f.emit(encodeAsBx(vm.OP_TFORLOOP, a, sbx))
}

func (f *funcInfo) emitSETLIST(a, b, c int) {
	f.emit(encodeABC(vm.OP_SETLIST, a, b, c))
}

func (f *funcInfo) emitCLOSURE(a, bx int) {
	f.emit(encodeABx(vm.OP_CLOSURE, a, bx))
}

func (f *funcInfo) emitVARARG(a, n int) {
	f.emit(encodeABC(vm.OP_CLOSURE, a, n+1, 0))
}


func cgenBlock(block *Block, f *funcInfo) {
	for _, stmt := range block.Stmts {
		cgenStmt(stmt, f)
	}
}

func cgenStmt(stmt Stmt, f *funcInfo) {
	switch stmt := stmt.(type) {
	case *EmptyStmt:
	case *BreakStmt:
		f.emitJMP(0, 0)
		f.addBreak(f.pc())
	case *LabelStmt:
	case *GotoStmt:
	case *ReturnStmt:
		nExprs := len(stmt.Exprs)
		lastIsVarargOrFuncCall := false

		a := f.usedRegs

		for i, expr := range stmt.Exprs {
			r := f.allocReg()
			if i == nExprs - 1 && _isVarargOrFuncCall(expr) {
				lastIsVarargOrFuncCall = true
				cgenExpr(expr, f, r, -1)
			} else {
				cgenExpr(expr, f, r, 1)
			}
		}

		if lastIsVarargOrFuncCall {
			f.emitRETURN(a, -1)
		} else {
			f.emitRETURN(a, nExprs)
		}

		f.freeRegs(nExprs)
	case *DoStmt:
		f.enterScope(false)
		cgenBlock(stmt.Block, f)
		f.closeOpenUpvalues()
		f.leaveScope()
	case *FuncCallStmt:
		r := f.allocReg()
		cgenExpr(stmt, f, r, 0)
		f.freeReg()
	case *WhileStmt:
		pc1 := f.pc()

		r := f.allocReg()
		cgenExpr(stmt.Expr, f, r, 1)
		f.freeReg()

		f.emitTEST(r, 0)
		f.emitJMP(0, 0)
		pc2 := f.pc()

		f.enterScope(true)
		cgenBlock(stmt.Block, f)
		f.emitJMP(f.getJmpArgA(), pc1 - f.pc() - 1)
		f.leaveScope()

		f.fix(pc2, f.pc() - pc2)
	case *RepeatStmt:
		f.enterScope(true)

		pc1 := f.pc()
		cgenBlock(stmt.Block, f)

		r := f.allocReg()
		cgenExpr(stmt.Expr, f, r, 1)
		f.freeReg()

		f.emitTEST(r, 0)
		f.emitJMP(f.getJmpArgA(), pc1 - f.pc() - 1)
		f.closeOpenUpvalues()

		f.leaveScope()
	case *IfStmt:
		var pcs []int
		pci := -1
	
		for i, cond := range stmt.Exprs {
			if pci != -1 {
				f.fix(pci, f.pc() - pci)
			}

			if cond != nil {
				r := f.allocReg()
				cgenExpr(cond, f, r, 1)
				f.freeReg()

				f.emitTEST(r, 0)
				f.emitJMP(0, 0)
				pci = f.pc()
			} else {
				pci = -1
			}

			f.enterScope(false)
			cgenBlock(stmt.Blocks[i], f)
			f.emitJMP(f.getJmpArgA(), 0)
			f.leaveScope()

			if i < len(stmt.Exprs) - 1 {
				pcs = append(pcs, f.pc())
			} else if pci != -1 {
				pcs = append(pcs, pci)
			}
		}
	
		for _, pc := range pcs {
			f.fix(pc, f.pc() - pc)
		}
	case *ForStmt:
		f.enterScope(true)

		a := f.usedRegs

		cgenStmt(&LocalDeclStmt{
			NameList: []string{"(for index)", "(for limit)", "(for step)"},
			ExprList: []Expr{stmt.Init, stmt.Limit, stmt.Step},
		}, f)

		f.addLocVar(stmt.VarName)

		f.emitFORPREP(a, 0)
		pc := f.pc()
		cgenBlock(stmt.Block, f)
		f.closeOpenUpvalues()
		f.fix(pc, f.pc() - pc)
		f.emitFORLOOP(a, pc - f.pc() - 1)

		f.leaveScope()
	case *ForListStmt:
		f.enterScope(true)

		a := f.usedRegs

		cgenStmt(&LocalDeclStmt{
			NameList: []string{"(for generator)", "(for state)", "(for control)"},
			ExprList: stmt.ExprList,
		}, f);

		for _, name := range stmt.NameList {
			f.addLocVar(name)
		}

		f.emitJMP(0, 0)
		pc := f.pc()

		cgenBlock(stmt.Block, f)
		f.closeOpenUpvalues()
		f.fix(pc, f.pc() - pc)
		f.emitTFORCALL(a, len(stmt.NameList))
		f.emitTFORLOOP(a + 2, pc - f.pc() - 1)

		f.leaveScope()
	case *LocalDeclStmt:
		nNames := len(stmt.NameList)
		nExprs := len(stmt.ExprList)

		if nExprs >= nNames {
			for i, expr := range stmt.ExprList {
				r := f.allocReg()
				if i < nNames {
					cgenExpr(expr, f, r, 1)
				} else {
					cgenExpr(expr, f, r, 0)
					f.freeReg()
				}
			}
		} else {
			lastIsVarargOrFuncCall := false

			for i, expr := range stmt.ExprList {
				r := f.allocReg()
				if i == nExprs - 1 && _isVarargOrFuncCall(expr) {
					lastIsVarargOrFuncCall = true
					n := nNames - nExprs + 1
					cgenExpr(expr, f, r, n)
					f.allocRegs(n - 1)
				} else {
					cgenExpr(expr, f, r, 1)
				}
			}

			if !lastIsVarargOrFuncCall {
				n := nNames - nExprs
				r := f.allocRegs(n)
				f.emitLOADNIL(r, n)
			}
		}

		f.freeRegs(nNames)
		for _, name := range stmt.NameList {
			f.addLocVar(name)
		}
	case *AssignStmt:
		nLefts := len(stmt.Vars)
		nExprs := len(stmt.ExprList)

		oldUsedRegs := f.usedRegs

		tRegs := make([]int, nLefts)
		kRegs := make([]int, nLefts)

		for i, left := range stmt.Vars {
			if expr, ok := left.(*IndexExpr); ok {
				tRegs[i] = f.allocReg()
				cgenExpr(expr.Expr, f, tRegs[i], 1)
				kRegs[i] = f.allocReg()
				cgenExpr(expr.KeyExpr, f, kRegs[i], 1)
			}
		}

		r := f.usedRegs

		if nExprs >= nLefts {
			for i, expr := range stmt.ExprList {
				r := f.allocReg()
				if i < nLefts {
					cgenExpr(expr, f, r, 1)
				} else {
					cgenExpr(expr, f, r, 0)
					f.freeReg()
				}
			}
		} else {
			lastIsVarargOrFuncCall := false

			for i, expr := range stmt.ExprList {
				r := f.allocReg()
				if i == nExprs - 1 && _isVarargOrFuncCall(expr) {
					lastIsVarargOrFuncCall = true
					n := nLefts - nExprs + 1
					cgenExpr(expr, f, r, n)
					f.allocRegs(n - 1)
				} else {
					cgenExpr(expr, f, r, 1)
				}
			}

			if !lastIsVarargOrFuncCall {
				n := nLefts - nExprs
				r := f.allocRegs(n)
				f.emitLOADNIL(r, n)
			}
		}

		for i, left := range stmt.Vars {
			if expr, ok := left.(*NameExpr); ok {
				if a := f.slotOfLocVar(expr.Name); a >= 0 {
					f.emitMOVE(a, r + i)
				} else if b := f.indexOfUpvalue(expr.Name); b >= 0 {
					f.emitSETUPVAL(r + i, b)
				} else {
					a := f.indexOfUpvalue("_ENV")
					b := 0x100 + f.indexOfConstant(expr.Name)
					f.emitSETTABUP(a, b, r + i)
				}
			} else {
				f.emitSETTABLE(tRegs[i], kRegs[i], r + i)
			}
		}

		f.usedRegs = oldUsedRegs
	}
}

func cgenExpr(expr Expr, f *funcInfo, a, n int) {
	switch expr := expr.(type) {
	case *NilExpr:
		f.emitLOADNIL(a, n)
	case *FalseExpr:
		f.emitLOADBOOL(a, 0, 0)
	case *TrueExpr:
		f.emitLOADBOOL(a, 1, 0)
	case *IntegerExpr:
		f.emitLOADK(a, expr.Val)
	case *FloatExpr:
		f.emitLOADK(a, expr.Val)
	case *StringExpr:
		f.emitLOADK(a, expr.Str)
	case *NameExpr:
		if r := f.slotOfLocVar(expr.Name); r >= 0 {
			f.emitMOVE(a, r)
		} else if idx := f.indexOfUpvalue(expr.Name); idx >= 0 {
			f.emitGETUPVAL(a, idx)
		} else { // x => _ENV["x"]
			cgenExpr(&IndexExpr{&NameExpr{0, "_ENV"}, &StringExpr{0, expr.Name}}, f, a, n) // TODO: Line
		}
	case *VarargExpr:
		if !f.isVararg {
			panic("cannot use '...' outside a vararg function")
		}
		f.emitVARARG(a, n)
	case *UnopExpr:
		cgenExpr(expr.Expr, f, a, 1)
		switch expr.Op {
		case TOKEN_NOT:
			f.emitNOT(a, a)
		case '#':
			f.emitLEN(a, a)
		case '-':
			f.emitUNM(a, a)
		case '~':
			f.emitBNOT(a, a)
		}
	case *BinopExpr:
		if expr.Op == TOKEN_AND || expr.Op == TOKEN_OR {
			cgenExpr(expr.LHS, f, a, 1)
			if expr.Op == TOKEN_AND {
				f.emitTEST(a, 0)
			} else {
				f.emitTEST(a, 1)
			}
			f.emitJMP(0, 0)
			pc := f.pc()
			cgenExpr(expr.RHS, f, a, 1)
			f.fix(pc, f.pc() - pc)
		} else {
			cgenExpr(expr.LHS, f, a, 1)
			c := f.allocReg()
			cgenExpr(expr.RHS, f, c, 1)
			switch expr.Op {
			case TOKEN_CONCAT:
				f.emitCONCAT(a, a, c)
			case '+':
				f.emitADD(a, a, c)
			case '-':
				f.emitSUB(a, a, c)
			case '*':
				f.emitMUL(a, a, c)
			case '%':
				f.emitMOD(a, a, c)
			case '^':
				f.emitPOW(a, a, c)
			case '/':
				f.emitDIV(a, a, c)
			case TOKEN_IDIV:
				f.emitIDIV(a, a, c)
			case '&':
				f.emitBAND(a, a, c)
			case '|':
				f.emitBOR(a, a, c)
			case '~':
				f.emitBXOR(a, a, c)
			case TOKEN_SHL:
				f.emitSHL(a, a, c)
			case TOKEN_SHR:
				f.emitSHR(a, a, c)
			default:
				switch expr.Op {
				case TOKEN_EQ:
					f.emitEQ(1, a, c)
				case TOKEN_NE:
					f.emitEQ(0, a, c)
				case '<':
					f.emitLT(1, a, c)
				case TOKEN_LE:
					f.emitLE(1, a, c)
				case '>':
					f.emitLE(1, c, a)
				case TOKEN_GE:
					f.emitLT(1, c, a)
				}
				f.emitJMP(0, 1)
				f.emitLOADBOOL(a, 0, 1)
				f.emitLOADBOOL(a, 1, 0)
			}
			f.freeReg()
		}
	case *TableExpr:
		nArr := 0
		for _, keyExpr := range expr.KeyExprs {
			if keyExpr == nil {
				nArr++
			}
		}

		size := len(expr.KeyExprs)
		multRet := size > 0 && _isVarargOrFuncCall(expr.ValExprs[size - 1])

		f.emitNEWTABLE(a, nArr, size - nArr)

		arrIdx := 0
		for i, keyExpr := range expr.KeyExprs {
			valExpr := expr.ValExprs[i]
			if keyExpr == nil {
				arrIdx++

				r := f.allocReg()
				if i == size - 1 && multRet {
					cgenExpr(valExpr, f, r, -1)
				} else {
					cgenExpr(valExpr, f, r, 1)
				}

				if arrIdx % vm.LFIELDS_PER_FLUSH == 0 || arrIdx == nArr {
					b := (arrIdx - 1) % vm.LFIELDS_PER_FLUSH + 1
					if i == size - 1 && multRet {
						b = 0
					}
					c := (arrIdx - 1) / vm.LFIELDS_PER_FLUSH
					f.emitSETLIST(a, b, c)
				}
			} else {
				b := f.allocReg()
				cgenExpr(keyExpr, f, b, -1)
				c := f.allocReg()
				cgenExpr(valExpr, f, c, -1)
				f.emitSETTABLE(a, b, c)
				f.freeRegs(2)
			}
		}
	case *FunctionExpr:
		bx := len(f.children)
		subF := newFuncInfo(f, len(expr.ParamList), expr.IsVararg)
		for _, paramName := range expr.ParamList {
			subF.addLocVar(paramName)
		}
		cgenBlock(expr.Block, subF)
		subF.leaveScope()
		subF.emitRETURN(0, 0)
		f.children = append(f.children, subF)
		f.emitCLOSURE(a, bx)
	case *ParenExpr:
		cgenExpr(expr.Expr, f, a, 1)
	case *IndexExpr:
		cgenExpr(expr.Expr, f, a, 1)
		r := f.allocReg()
		cgenExpr(expr.KeyExpr, f, r, 1)
		f.emitGETTABLE(a, a, r)
		f.freeReg()
	case *FuncCallExpr:
		nArgs := len(expr.Args)
		lastArgIsVarargOrFuncCall := false
		cgenExpr(expr.Expr, f, a, 1)
		if expr.Name != nil {
			f.allocReg()
			c := 0x100 + f.indexOfConstant(expr.Name.Str)
			f.emitSELF(a, a, c)
		}
		for i, argExpr := range expr.Args {
			r := f.allocReg()
			if i == nArgs - 1 && _isVarargOrFuncCall(argExpr) {
				lastArgIsVarargOrFuncCall = true
				cgenExpr(argExpr, f, r, -1)
			} else {
				cgenExpr(argExpr, f, r, 1)
			}
		}
		if (expr.Name != nil) {
			nArgs++
		}
		if lastArgIsVarargOrFuncCall {
			f.emitCALL(a, -1, n)
		} else {
			f.emitCALL(a, nArgs, n)
		}
		f.freeRegs(nArgs)
	}
}

func _isVarargOrFuncCall(expr Expr) bool {
	switch expr.(type) {
	case *VarargExpr, *FuncCallExpr:
		return true
	}
	return false
}

func GenProto(block *Block) *binary.Prototype {
	expr := &FunctionExpr{
		IsVararg: true,
		Block: block,
	}
	f := newFuncInfo(nil, 0, false)
	f.addLocVar("_ENV")
	cgenExpr(expr, f, 0, 1)
	return f.children[0].toProto()
}
