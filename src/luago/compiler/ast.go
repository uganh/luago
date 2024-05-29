package compiler

type Block struct {
	LastLine int
	Stmts    []Stmt
}

type Stmt interface{}

// ';'
type EmptyStmt struct {
}

// break
type BreakStmt struct {
	Line int
}

// '::' <Name> '::'
type LabelStmt struct {
	Name string
}

// goto <Name>
type GotoStmt struct {
	Name string
}

// return [<Expr> {','} <>]
type ReturnStmt struct {
	Exprs []Expr
}

// do <Block> end
type DoStmt struct {
	Block *Block
}

type FuncCallStmt = FuncCallExpr

// while <Expr> do <Block> end
type WhileStmt struct {
	Expr  Expr
	Block *Block
}

// repeat <Block> until <Expr>
type RepeatStmt struct {
	Block *Block
	Expr  Expr
}

// if <Expr> then <Block> {elseif <Expr> then <Block>} [else <Block>] end
type IfStmt struct {
	Exprs  []Expr
	Blocks []*Block
}

// for <Name> '=' <Expr> ',' <Expr> [',' <Expr>] do <Block> end
type ForStmt struct {
	LineOfFor int
	LineOfDo  int
	VarName   string
	Init      Expr
	Limit     Expr
	Step      Expr
	Block     *Block
}

// for <NameList> in <ExprList> do <Block> end
type ForListStmt struct {
	LineOfDo int
	NameList []string
	ExprList []Expr
	Block    *Block
}

// local <NameList> ['=' <ExprList>]
type LocalDeclStmt struct {
	LastLine int
	NameList []string
	ExprList []Expr
}

// <Vars> '=' <ExprList>
type AssignStmt struct {
	LastLine int
	Vars     []Expr
	ExprList []Expr
}

type Expr interface{}

type NilExpr struct {
	Line int
}

type TrueExpr struct {
	Line int
}

type FalseExpr struct {
	Line int
}

type IntegerExpr struct {
	Line  int
	Value int
}

type FloatExpr struct {
	Line  int
	Value float64
}

type StringExpr struct {
	Line  int
	Value string
}

type NameExpr struct {
	Line int
	Name string
}

type VarargExpr struct {
	Line int
}

type UnopExpr struct {
	Line int
	Op   int
	Expr Expr
}

type BinopExpr struct {
	Line int
	Op   int
	LHS  Expr
	RHS  Expr
}

type TableExpr struct {
	Line     int // line of '{'
	LastLine int // line of '}'
	KeyExprs []Expr
	ValExprs []Expr
}

type FunctionExpr struct {
	Line      int
	LastLine  int // line of 'end'
	ParamList []string
	IsVararg  bool
	Block     *Block
}

type ParenExpr struct {
	Expr Expr
}

type IndexExpr struct {
	Expr    Expr
	KeyExpr Expr
}

type FuncCallExpr struct {
	Line     int // line of '('
	LastLine int // line of ')'
	Expr     Expr
	Name     *StringExpr
	Args     []Expr
}
