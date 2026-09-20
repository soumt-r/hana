package ast

import (
	"bytes"
	"strings"
	"sync/atomic"

	"github.com/soumt-r/hana/symbol"
)

type Node interface {
	TokenLiteral() string
	String() string
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}
func (p *Program) String() string {
	var out strings.Builder
	for _, s := range p.Statements {
		out.WriteString(s.String())
		out.WriteString("\n")
	}
	return out.String()
}

// ----------------------------------------------------
// 리터럴 & 식별자
// ----------------------------------------------------

type Identifier struct {
	Value string

	sym atomic.Uint32 // Symbol()+1 once worked out; 0 until then
}

// Symbol is the interned form of Value, worked out the first time it is asked
// for: the tree-walker finds variables by Symbol, not by comparing strings.
func (i *Identifier) Symbol() symbol.Symbol {
	if s := i.sym.Load(); s != 0 {
		return symbol.Symbol(s - 1)
	}
	s := symbol.Intern(i.Value)
	i.sym.Store(uint32(s) + 1)
	return s
}

func (i *Identifier) TokenLiteral() string { return i.Value }
func (i *Identifier) String() string       { return i.Value }
func (i *Identifier) expressionNode()      {}

type SelfReference struct{}

func (s *SelfReference) TokenLiteral() string { return "SelfReference" }
func (s *SelfReference) String() string       { return "SelfReference" }
func (s *SelfReference) expressionNode()      {}

type SuperReference struct{}

func (s *SuperReference) TokenLiteral() string { return "SuperReference" }
func (s *SuperReference) String() string       { return "SuperReference" }
func (s *SuperReference) expressionNode()      {}

type StaticReference struct{}

func (s *StaticReference) TokenLiteral() string { return "StaticReference" }
func (s *StaticReference) String() string       { return "StaticReference" }
func (s *StaticReference) expressionNode()      {}

type NumberLiteral struct {
	Value float64
}

func (n *NumberLiteral) TokenLiteral() string { return "NUMBER" }
func (n *NumberLiteral) String() string       { return "NumberLiteral" }
func (n *NumberLiteral) expressionNode()      {}

type StringLiteral struct {
	Value string
}

func (s *StringLiteral) TokenLiteral() string { return s.Value }
func (s *StringLiteral) String() string       { return `"` + s.Value + `"` }
func (s *StringLiteral) expressionNode()      {}

type NullLiteral struct{}

func (n *NullLiteral) TokenLiteral() string { return "Null" }
func (n *NullLiteral) String() string       { return "Null" }
func (n *NullLiteral) expressionNode()      {}

type FunctionReference struct {
	Name string

	cached atomic.Pointer[namedSymbol]
}

// namedSymbol is a Symbol together with the name it was worked out for.
type namedSymbol struct {
	name string
	sym  symbol.Symbol
}

// Symbol is the interned form of Name. Name can be replaced while running (a
// dynamic `<'변수'>()` names another function), so the cached Symbol is only
// used while it still belongs to the current Name.
func (f *FunctionReference) Symbol() symbol.Symbol {
	if c := f.cached.Load(); c != nil && c.name == f.Name {
		return c.sym
	}
	sym := symbol.Intern(f.Name)
	f.cached.Store(&namedSymbol{f.Name, sym})
	return sym
}

func (f *FunctionReference) TokenLiteral() string { return f.Name }
func (f *FunctionReference) String() string       { return "<" + f.Name + ">" }
func (f *FunctionReference) expressionNode()      {}

type TypeReference struct {
	Name string
}

func (t *TypeReference) TokenLiteral() string { return t.Name }
func (t *TypeReference) String() string       { return "[" + t.Name + "]" }
func (t *TypeReference) expressionNode()      {}

// ----------------------------------------------------
// 표현식 (Expressions)
// ----------------------------------------------------

type MemberExpression struct {
	Object   Expression
	Property Expression
}

func (m *MemberExpression) TokenLiteral() string { return "." }
func (m *MemberExpression) String() string       { return m.Object.String() + "." + m.Property.String() }
func (m *MemberExpression) expressionNode()      {}

type CallExpression struct {
	Callee    Expression
	Arguments []Expression
}

func (c *CallExpression) TokenLiteral() string { return "(" }
func (c *CallExpression) String() string       { return c.Callee.String() + "(...)" }
func (c *CallExpression) expressionNode()      {}

type NewExpression struct {
	Class     *TypeReference
	Arguments []Expression
}

func (n *NewExpression) TokenLiteral() string { return "New" }
func (n *NewExpression) String() string       { return "New " + n.Class.String() + "()" }
func (n *NewExpression) expressionNode()      {}

type BinaryExpression struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (b *BinaryExpression) TokenLiteral() string { return b.Operator }
func (b *BinaryExpression) String() string {
	return b.Left.String() + " " + b.Operator + " " + b.Right.String()
}
func (b *BinaryExpression) expressionNode() {}

// LogicalExpression is 그리고/또는 (AND/OR). Unlike BinaryExpression, its
// evaluation must short-circuit (Runtime spec 5.2): Right is only evaluated
// when Left doesn't already determine the result.
type LogicalExpression struct {
	Left     Expression
	Operator string // "그리고" or "또는"
	Right    Expression
}

func (l *LogicalExpression) TokenLiteral() string { return l.Operator }
func (l *LogicalExpression) String() string {
	return l.Left.String() + " " + l.Operator + " " + l.Right.String()
}
func (l *LogicalExpression) expressionNode() {}

// ----------------------------------------------------
// 문장 (Statements)
// ----------------------------------------------------

type SetterInfo struct {
	Param *Identifier
	Body  []Statement
}

type VariableDeclaration struct {
	Name           *Identifier
	TypeRef        *TypeReference
	Value          Expression
	IsConstant     bool
	AccessModifier string // "public", "private", "protected"
	IsStatic       bool
	Getter         []Statement
	Setter         *SetterInfo
}

func (v *VariableDeclaration) TokenLiteral() string { return v.Name.Value }
func (v *VariableDeclaration) String() string {
	if v.Value == nil {
		return "var " + v.Name.String()
	}
	return "var " + v.Name.String() + " = " + v.Value.String()
}
func (v *VariableDeclaration) statementNode() {}

type Assignment struct {
	Target Expression
	Value  Expression
}

func (a *Assignment) TokenLiteral() string { return "=" }
func (a *Assignment) String() string {
	return a.Target.String() + " = " + a.Value.String()
}
func (a *Assignment) statementNode() {}

type PrintStatement struct {
	Value   Expression
	NewLine bool
}

func (p *PrintStatement) TokenLiteral() string { return "Print" }
func (p *PrintStatement) String() string       { return "Print(" + p.Value.String() + ")" }
func (p *PrintStatement) statementNode()       {}

type InputStatement struct {
	Target  *Identifier
	TypeRef *TypeReference
}

func (i *InputStatement) TokenLiteral() string { return "Input" }
func (i *InputStatement) String() string       { return "Input(" + i.Target.String() + ")" }
func (i *InputStatement) statementNode()       {}

type ExpressionStatement struct {
	Expression Expression
}

func (e *ExpressionStatement) TokenLiteral() string { return e.Expression.TokenLiteral() }
func (e *ExpressionStatement) String() string       { return e.Expression.String() }
func (e *ExpressionStatement) statementNode()       {}

type BlockStatement struct {
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return "" }
func (bs *BlockStatement) String() string {
	var out bytes.Buffer
	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

// SwitchStatement
type SwitchStatement struct {
	Discriminant Expression
	Cases        []*SwitchCase
}

func (ss *SwitchStatement) statementNode()       {}
func (ss *SwitchStatement) TokenLiteral() string { return "switch" }
func (ss *SwitchStatement) String() string       { return "switch" }

// SwitchCase
type SwitchCase struct {
	Tests      []Expression
	Consequent *BlockStatement
	IsDefault  bool
}

type FallthroughStatement struct{}

func (fs *FallthroughStatement) statementNode()       {}
func (fs *FallthroughStatement) TokenLiteral() string { return "fallthrough" }
func (fs *FallthroughStatement) String() string       { return "fallthrough" }

type IfStatement struct {
	Condition  Expression
	Consequent *BlockStatement
	Alternate  *BlockStatement
}

func (i *IfStatement) TokenLiteral() string { return "If" }
func (i *IfStatement) String() string       { return "If " + i.Condition.String() }
func (i *IfStatement) statementNode()       {}

type ReturnStatement struct {
	Value Expression
}

func (r *ReturnStatement) TokenLiteral() string { return "Return" }
func (r *ReturnStatement) String() string {
	if r.Value == nil {
		return "Return"
	}
	return "Return " + r.Value.String()
}
func (r *ReturnStatement) statementNode() {}

type BreakStatement struct{}

func (b *BreakStatement) TokenLiteral() string { return "Break" }
func (b *BreakStatement) String() string       { return "Break" }
func (b *BreakStatement) statementNode()       {}

type ForEachLoop struct {
	List Expression
	Body *BlockStatement
}

func (f *ForEachLoop) TokenLiteral() string { return "ForEach" }
func (f *ForEachLoop) String() string       { return "ForEach " + f.List.String() }
func (f *ForEachLoop) statementNode()       {}

type ForRangeStatement struct {
	Start   Expression
	End     Expression
	LoopVar string
	Body    *BlockStatement
}

func (f *ForRangeStatement) TokenLiteral() string { return "ForRange" }
func (f *ForRangeStatement) String() string {
	return "ForRange " + f.Start.String() + " to " + f.End.String()
}
func (f *ForRangeStatement) statementNode() {}

type WhileLoop struct {
	Condition Expression
	Body      *BlockStatement
}

func (w *WhileLoop) TokenLiteral() string { return "WhileLoop" }
func (w *WhileLoop) String() string       { return "While " + w.Condition.String() }
func (w *WhileLoop) statementNode()       {}

type ClassDeclaration struct {
	Name       *TypeReference
	BaseClass  *TypeReference
	Interfaces []*TypeReference
	Body       []Statement
	IsAbstract bool // declared with the abstract verb (밑설계하자); cannot be instantiated
}

func (c *ClassDeclaration) TokenLiteral() string { return "Class" }
func (c *ClassDeclaration) String() string {
	var out strings.Builder
	if c.BaseClass != nil {
		out.WriteString("class " + c.Name.Name + " extends " + c.BaseClass.Name + " {\n")
	} else {
		out.WriteString("class " + c.Name.Name + " {\n")
	}
	for _, s := range c.Body {
		out.WriteString("  " + s.String() + "\n")
	}
	out.WriteString("}")
	return out.String()
}
func (c *ClassDeclaration) statementNode() {}

// ImportItem is one name an import brings in, optionally under another name.
// As is "" unless the source used the `~로 가져오자` renaming form —
// 서로 다른 두 패키지가 같은 이름을 내보낼 때 로컬에서 다른 이름으로
// 바인딩하기 위한 형태입니다.
type ImportItem struct {
	Name string
	As   string
}

// BindName is the name the item is bound to in the importing scope.
func (i ImportItem) BindName() string {
	if i.As != "" {
		return i.As
	}
	return i.Name
}

// ImportStatement is `[모듈]에서 <이름>을 가져오자`, its list form
// (`<이름>와 <이름2>를 가져오자`), or the whole-module form (`[모듈]에서 전부
// 가져오자`, All). Only a single item can be renamed (`<이름>을 <별칭>으로`).
type ImportStatement struct {
	Module    string
	IsBuiltin bool
	All       bool
	Items     []ImportItem
}

func (i *ImportStatement) TokenLiteral() string { return "Import" }
func (i *ImportStatement) String() string {
	what := "*"
	if !i.All {
		names := make([]string, len(i.Items))
		for n, item := range i.Items {
			names[n] = item.Name
			if item.As != "" {
				names[n] += " as " + item.As
			}
		}
		what = strings.Join(names, ", ")
	}
	if i.IsBuiltin {
		return "Import " + what + " from builtin " + i.Module
	}
	return "Import " + what + " from " + i.Module
}
func (i *ImportStatement) statementNode() {}

type InterfaceDeclaration struct {
	Name *TypeReference
	Body []Statement
}

func (i *InterfaceDeclaration) TokenLiteral() string { return "Interface" }
func (i *InterfaceDeclaration) String() string {
	var out strings.Builder
	out.WriteString("interface " + i.Name.String() + " {\n")
	for _, s := range i.Body {
		out.WriteString("  " + s.String() + "\n")
	}
	out.WriteString("}")
	return out.String()
}
func (i *InterfaceDeclaration) statementNode() {}

type ListLiteral struct {
	Elements []Expression
}

func (l *ListLiteral) TokenLiteral() string { return "[" }
func (l *ListLiteral) String() string       { return "[List]" }
func (l *ListLiteral) expressionNode()      {}

type Property struct {
	Key   Expression
	Value Expression
}

type DictLiteral struct {
	Properties []*Property
}

func (d *DictLiteral) TokenLiteral() string { return "{" }
func (d *DictLiteral) String() string       { return "{Dict}" }
func (d *DictLiteral) expressionNode()      {}

type ListPushStatement struct {
	Target   Expression
	Value    Expression
	Position string // "front", "back"
}

func (s *ListPushStatement) TokenLiteral() string { return "ListPush" }
func (s *ListPushStatement) String() string       { return "ListPush" }
func (s *ListPushStatement) statementNode()       {}

type ListPopExpression struct {
	Target   Expression
	Position string // "front", "back"
}

func (s *ListPopExpression) TokenLiteral() string { return "ListPopExpr" }
func (s *ListPopExpression) String() string       { return "ListPopExpr" }
func (s *ListPopExpression) expressionNode()      {}

type ListPopStatement struct {
	Target   Expression
	Position string // "front", "back"
}

func (s *ListPopStatement) TokenLiteral() string { return "ListPop" }
func (s *ListPopStatement) String() string       { return "ListPop" }
func (s *ListPopStatement) statementNode()       {}

type TryStatement struct {
	Block     *BlockStatement
	Handlers  []*CatchClause
	Finalizer *BlockStatement
}

func (s *TryStatement) TokenLiteral() string { return "Try" }
func (s *TryStatement) String() string       { return "Try" }
func (s *TryStatement) statementNode()       {}

type CatchClause struct {
	Type  *TypeReference
	Param *Identifier
	Body  *BlockStatement
}

func (c *CatchClause) TokenLiteral() string { return "Catch" }
func (c *CatchClause) String() string       { return "Catch" }
func (c *CatchClause) statementNode()       {} // Though usually embedded, we implement it for simplicity

type ThrowStatement struct {
	Value Expression
}

func (s *ThrowStatement) TokenLiteral() string { return "Throw" }
func (s *ThrowStatement) String() string       { return "Throw" }
func (s *ThrowStatement) statementNode()       {}

type Parameter struct {
	Name           *Identifier
	TypeAnnotation *TypeReference
	Default        Expression
}

func (p *Parameter) TokenLiteral() string { return p.Name.Value }
func (p *Parameter) String() string       { return p.Name.String() }
func (p *Parameter) expressionNode()      {}

type FunctionDeclaration struct {
	Name           *Identifier
	Params         []*Parameter
	Body           *BlockStatement
	AccessModifier string // "public", "private", "protected"
	IsStatic       bool
	ReturnType     *TypeReference
}

func (f *FunctionDeclaration) TokenLiteral() string { return f.Name.Value }
func (f *FunctionDeclaration) String() string {
	var out strings.Builder
	out.WriteString("func " + f.Name.String() + "() {\n")
	for _, s := range f.Body.Statements {
		out.WriteString("  " + s.String() + "\n")
	}
	out.WriteString("}")
	return out.String()
}
func (f *FunctionDeclaration) statementNode() {}

type InterfaceMethod struct {
	Name *Identifier
}

func (i *InterfaceMethod) TokenLiteral() string { return i.Name.Value }
func (i *InterfaceMethod) String() string       { return i.Name.String() + "()" }
func (i *InterfaceMethod) statementNode()       {}

type ConstructorDeclaration struct {
	Id     *Identifier
	Params []*Parameter
	Body   []Statement
}

func (cd *ConstructorDeclaration) statementNode() {}

func (cd *ConstructorDeclaration) String() string { return "Constructor" }

func (cd *ConstructorDeclaration) TokenLiteral() string { return "Constructor" }

type TemplateLiteral struct {
	Value string
}

func (t *TemplateLiteral) expressionNode()      {}
func (t *TemplateLiteral) TokenLiteral() string { return t.Value }
func (t *TemplateLiteral) String() string       { return t.Value }

type BooleanLiteral struct {
	Value bool
}

func (b *BooleanLiteral) expressionNode() {}
func (b *BooleanLiteral) TokenLiteral() string {
	if b.Value {
		return "true"
	}
	return "false"
}
func (b *BooleanLiteral) String() string {
	if b.Value {
		return "true"
	}
	return "false"
}
