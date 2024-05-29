package compiler

func Parse(chunk, chunkName string) *Block {
	lexer := NewLexer(chunk, chunkName)
	lexer.Next()
	block := parseBlock(lexer)
	checkNext(lexer, TOKEN_EOF)
	return block
}

func parseBlock(lexer *Lexer) *Block {
	var stmts []Stmt
	for !blockFollow(lexer) {
		stmt := parseStmt(lexer)
		if _, ok := stmt.(*EmptyStmt); !ok {
			stmts = append(stmts, stmt)
			if _, ok := stmt.(*ReturnStmt); ok {
				break
			}
		}
	}
	return &Block{
		LastLine: lexer.line,
		Stmts:    stmts,
	}
}

func parseStmt(lexer *Lexer) Stmt {
	line := lexer.line
	switch lexer.LookAhead.Kind {
	case ';':
		lexer.Next() // skip ';'
		return &EmptyStmt{}
	case TOKEN_IF:
		return parseIfStmt(lexer)
	case TOKEN_WHILE:
		return parseWhileStmt(lexer)
	case TOKEN_DO:
		return parseDoStmt(lexer)
	case TOKEN_FOR:
		lexer.Next() // skip FOR
		varName := checkName(lexer)
		switch lexer.LookAhead.Kind {
		case ',', TOKEN_IN:
			return parseForListStmt(lexer, varName)
		default: // '='
			return parseForStmt(lexer, varName, line)
		}
	case TOKEN_REPEAT:
		return parseRepeatStmt(lexer)
	case TOKEN_FUNCTION:
		return parseFunctionStmt(lexer)
	case TOKEN_LOCAL:
		lexer.Next() // skip LOCAL
		if testNext(lexer, TOKEN_FUNCTION) {
			return parseLocalFunctionStmt(lexer, line)
		} else {
			return parseLocalStmt(lexer)
		}
	case TOKEN_DBCOLON:
		lexer.Next() // skip '::'
		name := checkName(lexer)
		checkNext(lexer, TOKEN_DBCOLON)
		return &LabelStmt{name}
	case TOKEN_RETURN:
		return parseReturnStmt(lexer)
	case TOKEN_BREAK:
		return &BreakStmt{lexer.line}
	case TOKEN_GOTO:
		lexer.Next() // skip GOTO
		name := checkName(lexer)
		return &GotoStmt{name}
	default:
		return parseExprStmt(lexer)
	}
}

// IF cond THEN block {ELSEIF cond THEN block} [ELSE block] END
func parseIfStmt(lexer *Lexer) *IfStmt {
	var exprs []Expr
	var blocks []*Block

	lexer.Next() // skip IF

	exprs = append(exprs, parseExpr(lexer))
	checkNext(lexer, TOKEN_THEN)
	blocks = append(blocks, parseBlock(lexer))

	for testNext(lexer, TOKEN_ELSEIF) {
		exprs = append(exprs, parseExpr(lexer))
		checkNext(lexer, TOKEN_THEN)
		blocks = append(blocks, parseBlock(lexer))
	}

	if testNext(lexer, TOKEN_ELSE) {
		exprs = append(exprs, nil)
		blocks = append(blocks, parseBlock(lexer))
	}

	checkNext(lexer, TOKEN_END)

	return &IfStmt{exprs, blocks}
}

// WHILE cond DO block END
func parseWhileStmt(lexer *Lexer) *WhileStmt {
	lexer.Next() // skip WHILE
	expr := parseExpr(lexer)
	checkNext(lexer, TOKEN_DO)
	block := parseBlock(lexer)
	checkNext(lexer, TOKEN_END)
	return &WhileStmt{expr, block}
}

// DO block END
func parseDoStmt(lexer *Lexer) *DoStmt {
	lexer.Next() // skip DO
	block := parseBlock(lexer)
	checkNext(lexer, TOKEN_END)
	return &DoStmt{block}
}

// FOR name '=' expr, expr [',' expr] DO block END
func parseForStmt(lexer *Lexer, varName string, lineOfFor int) *ForStmt {
	lexer.Next()             // skip '='
	init := parseExpr(lexer) // initial value
	checkNext(lexer, ',')
	limit := parseExpr(lexer) // limit
	var step Expr
	if testNext(lexer, ',') {
		step = parseExpr(lexer) // optional step
	} else { // default step = 1
		step = &IntegerExpr{lexer.line, 1}
	}
	lineOfDo := lexer.Line()
	checkNext(lexer, TOKEN_DO)
	block := parseBlock(lexer)
	checkNext(lexer, TOKEN_END)
	return &ForStmt{
		LineOfFor: lineOfFor,
		LineOfDo:  lineOfDo,
		VarName:   varName,
		Init:      init,
		Limit:     limit,
		Step:      step,
		Block:     block,
	}
}

// FOR name {',' name} IN expr, [',' expr] DO block END
func parseForListStmt(lexer *Lexer, varName string) *ForListStmt {
	names := []string{varName}
	for testNext(lexer, ',') {
		names = append(names, checkName(lexer))
	}
	checkNext(lexer, TOKEN_IN)
	exprs := parseExprList(lexer)
	lineOfDo := lexer.Line()
	checkNext(lexer, TOKEN_DO)
	block := parseBlock(lexer)
	checkNext(lexer, TOKEN_END)
	return &ForListStmt{
		LineOfDo: lineOfDo,
		NameList: names,
		ExprList: exprs,
		Block:    block,
	}
}

// REPEAT block UNTIL cond
func parseRepeatStmt(lexer *Lexer) *RepeatStmt {
	lexer.Next() // skip REPEAT
	block := parseBlock(lexer)
	checkNext(lexer, TOKEN_UNTIL)
	expr := parseExpr(lexer)
	return &RepeatStmt{block, expr}
}

// FUNCTION NAME {'.' NAME} [':' NAME] body
func parseFunctionStmt(lexer *Lexer) *AssignStmt {
	line := lexer.line
	lexer.Next() // skip FUNCTION

	var fnExpr Expr = NameExpr{lexer.line, checkName(lexer)}

	for testNext(lexer, '.') {
		fnExpr = &IndexExpr{
			Expr:    fnExpr,
			KeyExpr: &StringExpr{lexer.line, checkName(lexer)},
		}
	}

	isMethod := false
	if testNext(lexer, ':') {
		isMethod = true
		fnExpr = &IndexExpr{
			Expr:    fnExpr,
			KeyExpr: &StringExpr{lexer.line, checkName(lexer)},
		}
	}

	fdExpr := parseFunctionExpr(lexer, isMethod, line)

	return &AssignStmt{
		// TODO: line
		Vars:     []Expr{fnExpr},
		ExprList: []Expr{fdExpr},
	}
}

// LOCAL NAME { ',' NAME } [ '=' expr { ',' expr } ]
func parseLocalStmt(lexer *Lexer) *LocalDeclStmt {
	var nameList []string
	var exprList []Expr
	for {
		nameList = append(nameList, checkName(lexer))
		if !testNext(lexer, ',') {
			break
		}
	}
	if testNext(lexer, '=') {
		exprList = parseExprList(lexer)
	}
	return &LocalDeclStmt{
		LastLine: lexer.line,
		NameList: nameList,
		ExprList: exprList,
	}
}

// LOCAL FUNCTION NAME body
func parseLocalFunctionStmt(lexer *Lexer, line int) *LocalDeclStmt {
	name := checkName(lexer)
	expr := parseFunctionExpr(lexer, false, line)

	return &LocalDeclStmt{
		// TODO: LastLine int
		NameList: []string{name},
		ExprList: []Expr{expr},
	}
}

// RETURN [expr {',' expr}] [';']
func parseReturnStmt(lexer *Lexer) *ReturnStmt {
	lexer.Next() // skip RETURN
	var exprs []Expr
	if !blockFollow(lexer) && lexer.LookAhead.Kind != ';' {
		exprs = append(exprs, parseExpr(lexer))
		for testNext(lexer, ',') {
			exprs = append(exprs, parseExpr(lexer))
		}
	}
	testNext(lexer, ';') // skip optional ';'
	return &ReturnStmt{exprs}
}

func parseExprStmt(lexer *Lexer) Stmt {
	expr := parseSuffixedExpr(lexer)

	if stmt, ok := expr.(*FuncCallExpr); ok {
		return stmt
	}

	/* assignment */

	var vars []Expr = []Expr{expr}

	for lexer.LookAhead.Kind == ',' {
		lexer.Next() // skip ','
		expr := parseSuffixedExpr(lexer)
		switch expr.(type) { // check variable
		case *NameExpr, *IndexExpr:
			break
		default:
			lexer.error("syntax error")
		}
		vars = append(vars, expr)
	}

	checkNext(lexer, '=')
	exprList := parseExprList(lexer)

	return &AssignStmt{
		LastLine: lexer.line,
		Vars:     vars,
		ExprList: exprList,
	}
}

// '(' [ param { ',' param } ] ')' block END
func parseFunctionExpr(lexer *Lexer, isMethod bool, line int) *FunctionExpr {
	checkNext(lexer, '(')

	var params []string
	if isMethod {
		params = append(params, "self")
	}

	isVararg := false

	if lexer.LookAhead.Kind != ')' {
		for !isVararg {
			if lexer.LookAhead.Kind == TOKEN_NAME {
				params = append(params, checkName(lexer))
				if !testNext(lexer, ',') {
					break
				}
			} else if lexer.LookAhead.Kind == TOKEN_VARARG {
				lexer.Next()
				isVararg = true
			} else {
				lexer.error("<name> or '...' expected")
			}
		}
	}

	checkNext(lexer, ')')
	block := parseBlock(lexer)
	lastLine := lexer.line
	checkNext(lexer, TOKEN_END)

	return &FunctionExpr{
		Line:      line,
		LastLine:  lastLine,
		ParamList: params,
		IsVararg:  isVararg,
		Block:     block,
	}
}

func parseExprList(lexer *Lexer) []Expr {
	var exprs []Expr
	exprs = append(exprs, parseExpr(lexer))
	for testNext(lexer, ',') {
		exprs = append(exprs, parseExpr(lexer))
	}
	return exprs
}

// ( NAME | '(' expr ')' ) { '.' NAME | '[' expr ']' | ':' NAME funcargs | funcargs }
func parseSuffixedExpr(lexer *Lexer) Expr {
	var expr Expr
	switch lexer.LookAhead.Kind {
	case TOKEN_NAME:
		expr = &NameExpr{lexer.line, checkName(lexer)}
	case '(':
		lexer.Next() // skip '('
		expr = parseExpr(lexer)
		switch expr.(type) {
		case *VarargExpr, *FuncCallExpr, *NameExpr, *IndexExpr:
			expr = &ParenExpr{expr}
		}
		checkNext(lexer, ')')
	default:
		lexer.error("unexpected symbol")
	}

	for {
		var name *StringExpr
		switch lexer.LookAhead.Kind {
		case '.':
			lexer.Next() // skip '.'
			expr = &IndexExpr{
				Expr:    expr,
				KeyExpr: &StringExpr{lexer.line, checkName(lexer)},
			}
		case '[':
			lexer.Next() // skip '['
			expr = &IndexExpr{
				Expr:    expr,
				KeyExpr: parseExpr(lexer),
			}
			checkNext(lexer, ']')
		case ':':
			lexer.Next() // skip ':'
			name = &StringExpr{lexer.line, checkName(lexer)}
			fallthrough
		case '(', TOKEN_STRING, '{': // funcargs
			line := lexer.line
			var args []Expr
			if testNext(lexer, '(') {
				if lexer.LookAhead.Kind != ')' { // arg list is empty?
					args = parseExprList(lexer)
				}
				checkNext(lexer, ')')
			} else if testNext(lexer, '{') {
				args = append(args, parseTableExpr(lexer))
			} else {
				args = append(args, &StringExpr{line, lexer.LookAhead.Value})
				lexer.Next()
			}
			expr = &FuncCallExpr{
				Line:     line,
				LastLine: lexer.line, // TODO
				Expr:     expr,
				Name:     name,
				Args:     args,
			}
		default:
			return expr
		}
	}
}

func parseExpr(lexer *Lexer) Expr {
	return parseExpr1(lexer, 0)
}

func parseExpr0(lexer *Lexer) Expr {
	line := 0
	switch tokenKind := lexer.LookAhead.Kind; tokenKind {
	case TOKEN_NOT, '#', '-', '~':
		lexer.Next()
		return &UnopExpr{line, tokenKind, parseExpr1(lexer, unaryPriority)}
	case TOKEN_NUMBER: // TODO: FLT, INT
		lexer.Next()
		return &IntegerExpr{0, 0}
	case TOKEN_STRING:
		value := lexer.LookAhead.Value
		lexer.Next()
		return &StringExpr{line, value}
	case TOKEN_NIL:
		lexer.Next()
		return &NilExpr{line}
	case TOKEN_TRUE:
		lexer.Next()
		return &TrueExpr{line}
	case TOKEN_FALSE:
		lexer.Next()
		return &FalseExpr{line}
	case TOKEN_VARARG:
		lexer.Next()
		return &VarargExpr{line} // TODO: can use vararg?
	case '{': // constructor
		return parseTableExpr(lexer)
	case TOKEN_FUNCTION:
		return parseFunctionExpr(lexer, false, line)
	default:
		return parseSuffixedExpr(lexer)
	}
}

func parseExpr1(lexer *Lexer, prec int) Expr {
	expr := parseExpr0(lexer)

	for {
		binop := lexer.LookAhead.Kind
		leftPrec, rightPrec := getBinopPrecedence(binop)
		if leftPrec <= prec {
			break
		}
		line := 0    // TODO
		lexer.Next() // skip binop

		expr = &BinopExpr{
			line,
			binop,
			expr,
			parseExpr1(lexer, rightPrec),
		}
	}

	return expr
}

// '{' [ field { ( ',' | ';' ) field } [ ',' | ';' ] ] ';'
func parseTableExpr(lexer *Lexer) *TableExpr {
	line := lexer.line
	lexer.Next() // skip '{'

	var keyList []Expr
	var valList []Expr

	for lexer.LookAhead.Kind != '}' {
		// '[' expr ']' '=' expr | NAME '=' expr | expr

		if lexer.LookAhead.Kind == '[' {
			lexer.Next() // skip '['
			keyList = append(keyList, parseExpr(lexer))
			checkNext(lexer, ']')
			checkNext(lexer, '=')
			valList = append(valList, parseExpr(lexer))
		} else {
			expr := parseExpr(lexer)
			if nameExpr, ok := expr.(*NameExpr); ok {
				keyList = append(keyList, &StringExpr{nameExpr.Line, nameExpr.Name})
				checkNext(lexer, '=')
				valList = append(valList, parseExpr(lexer))
			} else {
				keyList = append(keyList, nil)
				valList = append(valList, expr)
			}
		}

		if !testNext(lexer, ',') && !testNext(lexer, ';') {
			break
		}
	}

	lastLine := lexer.line
	checkNext(lexer, '}')

	return &TableExpr{
		Line:     line,
		LastLine: lastLine,
		KeyExprs: keyList,
		ValExprs: valList,
	}
}

func testNext(lexer *Lexer, kind int) bool {
	if lexer.LookAhead.Kind == kind {
		lexer.Next()
		return true
	}
	return false
}

func check(lexer *Lexer, kind int) {
	if lexer.LookAhead.Kind != kind {
		// TODO <kind> expected near ''
		lexer.error("syntax error")
	}
}

func checkNext(lexer *Lexer, kind int) {
	check(lexer, kind)
	lexer.Next()
}

func checkName(lexer *Lexer) string {
	check(lexer, TOKEN_NAME)
	name := lexer.LookAhead.Value
	lexer.Next()
	return name
}

/**
 * or
 * and
 * < > <= >= ~= ==
 * |
 * ~
 * &
 * << >>
 * .. (right associative)
 * + -
 * * / // %
 * not # - ~ (unary operators)
 * ^ (right associative)
 */

func getBinopPrecedence(binop int) (int, int) {
	switch binop {
	case TOKEN_OR:
		return 1, 1
	case TOKEN_AND:
		return 2, 2
	case '<', '>', TOKEN_LE, TOKEN_GE, TOKEN_NE, TOKEN_EQ:
		return 3, 3
	case '|':
		return 4, 4
	case '~':
		return 5, 5
	case '&':
		return 6, 6
	case TOKEN_SHL, TOKEN_SHR:
		return 7, 7
	case TOKEN_CONCAT:
		return 9, 8
	case '+', '-':
		return 10, 10
	case '*', '/', TOKEN_IDIV, '%':
		return 11, 11
	case '^':
		return 14, 13
	default: // not binop
		return 0, 0
	}
}

const unaryPriority = 12

func blockFollow(lexer *Lexer) bool {
	switch lexer.LookAhead.Kind {
	case TOKEN_EOF, TOKEN_END, TOKEN_ELSE, TOKEN_ELSEIF, TOKEN_UNTIL:
		return true
	}
	return false
}
