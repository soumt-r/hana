package hari

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/lexer/hari"
	"github.com/soumt-r/hana/token"
)

type Parser struct {
	tokens []token.Token
	pos    int
	errors []string
	lang   *LangProfile
	diags  []Diagnostic

	// declaredType is the `[타입]인 값` annotation parsePrimary just consumed;
	// parseGenericSOV moves it onto the component so a declaration can keep it.
	declaredType *ast.TypeReference
}

func New(l *hari.Lexer) *Parser {
	return NewFromTokens(l.Tokens, hariProfile)
}

// NewFromTokens builds a Parser for a token stream produced by a
// different locale's lexer, driven by that locale's LangProfile. Every
// other-locale entry point (e.g. parser/kanade) goes through this.
func NewFromTokens(tokens []token.Token, lang *LangProfile) *Parser {
	return &Parser{
		tokens: tokens,
		pos:    0,
		errors: []string{},
		lang:   lang,
	}
}

func (p *Parser) peek(offset int) *token.Token {
	if p.pos+offset < len(p.tokens) {
		return &p.tokens[p.pos+offset]
	}
	return nil
}

func (p *Parser) consume() token.Token {
	tok := p.tokens[p.pos]
	p.pos++
	return tok
}

// Errors lists every problem the parse found, in English: recorded errors plus
// one line per Diagnostic. Callers that show them to a person use
// LocalizedErrors instead.
func (p *Parser) Errors() []string {
	out := append([]string{}, p.errors...)
	for _, d := range p.Diagnostics() {
		out = append(out, d.Err().Error())
	}
	return out
}

// LocalizedErrors is Errors in the reader's language.
func (p *Parser) LocalizedErrors(loc errs.Locale) []string {
	out := append([]string{}, p.errors...)
	for _, d := range p.Diagnostics() {
		out = append(out, errs.Localize(loc, d.Err()))
	}
	return out
}

// DiagKind identifies what a Diagnostic is about; editors (the LSP) render
// the wording in the document's language.
type DiagKind string

const DiagUnknownToken DiagKind = "UnknownToken"

// Diagnostic is a parse problem with its source position (1-based Line,
// 0-based Col in characters). The parser recovers and keeps going, so these
// are collected instead of aborting; Errors() stays for hard failures.
type Diagnostic struct {
	Kind    DiagKind
	Line    int
	Col     int
	Length  int
	Literal string
}

// Diagnostics lists what to report. Running off the end of the input is
// usually just fallout of an earlier bad token, so it is included only when
// nothing better explains it, and only once.
func (p *Parser) Diagnostics() []Diagnostic {
	var real []Diagnostic
	for _, d := range p.diags {
		if d.Literal != "" {
			real = append(real, d)
		}
	}
	if len(real) == 0 && len(p.diags) > 0 {
		return p.diags[:1]
	}
	return real
}

func (p *Parser) lastContentLine() int {
	for i := len(p.tokens) - 1; i >= 0; i-- {
		switch p.tokens[i].Type {
		case token.EOF, "INDENT", "DEDENT":
			continue
		}
		return p.tokens[i].Line
	}
	return 1
}

// Err is the typed, localizable error for this diagnostic: an unknown token,
// or (empty literal) the parser running off the end of the input.
func (d Diagnostic) Err() *errs.Error {
	if d.Literal == "" {
		return errs.New(errs.SyntaxUnexpectedEnd, d.Line)
	}
	return errs.New(errs.SyntaxUnexpectedToken, d.Line, d.Col+1, d.Literal)
}

func (p *Parser) ParseProgram() *ast.Program {
	prog := &ast.Program{Statements: []ast.Statement{}}
	for p.peek(0) != nil && p.peek(0).Type != token.EOF {
		if p.peek(0).Type == "DEDENT" || p.peek(0).Type == "INDENT" {
			p.consume()
			continue
		}
		stmt := p.parseStatement()
		if stmt != nil {
			prog.Statements = append(prog.Statements, stmt)
		}
	}
	return prog
}

func (p *Parser) parseBlock() *ast.BlockStatement {
	block := &ast.BlockStatement{Statements: []ast.Statement{}}
	tok := p.peek(0)
	if tok != nil && tok.Type == token.COLON {
		p.consume() // ':'
	}
	tok = p.peek(0)
	if tok != nil && tok.Type == "INDENT" {
		p.consume() // 'INDENT'
		for {
			t := p.peek(0)
			if t == nil || t.Type == "DEDENT" || t.Type == token.EOF {
				break
			}
			stmt := p.parseStatement()
			if stmt != nil {
				block.Statements = append(block.Statements, stmt)
			}
		}
		if p.peek(0) != nil && p.peek(0).Type == "DEDENT" {
			p.consume()
		}
	}
	return block
}

func (p *Parser) parseStatement() ast.Statement {
	tok := p.peek(0)
	if tok == nil || tok.Type == token.EOF {
		return nil
	}
	if tok.Type == "INDENT" || tok.Type == "DEDENT" {
		// DEDENT는 parseBlock 등에서 루프 탈출 조건으로 쓰이므로 여기서 소비하지 않음
		// INDENT만 불필요하게 남은 경우 처리
		if tok.Type == "INDENT" {
			p.consume()
		}
		return nil
	}

	if tok.Type == token.LBRACKET || tok.Type == "TYPE" {
		// 타입 파싱일 수 있음 (Class, Interface)
		// 토큰을 스캔해서 KW_CLASS나 KW_INTERFACE가 있는지 찾음
		savedPos := p.pos
		foundType := false
		isClass := false
		isIface := false
		for i := p.pos; i < len(p.tokens); i++ {
			if p.tokens[i].Line != tok.Line {
				break
			}
			if p.tokens[i].Type == token.COLON || p.tokens[i].Type == token.KW_FROM || p.tokens[i].Type == token.KW_IMPORT {
				break
			}
			if p.tokens[i].Type == token.KW_CLASS || p.tokens[i].Type == token.KW_IMPLEMENTS {
				foundType = true
				isClass = true
				break
			}
			if p.tokens[i].Type == token.KW_INTERFACE {
				foundType = true
				isIface = true
				break
			}
		}

		if foundType {
			if isClass {
				return p.parseClass()
			}
			if isIface {
				return p.parseInterface()
			}
		}
		p.pos = savedPos
	}

	isFuncDecl := false
	startLine := tok.Line
	for i := p.pos; i < len(p.tokens); i++ {
		t := p.tokens[i]
		if t.Line != startLine {
			break
		}
		if t.Type == token.COLON || t.Type == "DEDENT" || t.Type == "INDENT" {
			break
		}
		if t.Type == "FUNCTION" {
			if i+1 < len(p.tokens) {
				nextIdx := i + 1
				if p.tokens[nextIdx].Type == "PARTICLE" {
					nextIdx++
				}
				if nextIdx < len(p.tokens) && (p.tokens[nextIdx].Type == token.KW_MAKE || p.tokens[nextIdx].Type == token.KW_MUST_HAVE) {
					isFuncDecl = true
					break
				}
			}
		}
	}
	if isFuncDecl {
		return p.parseFunctionDecl()
	}

	if tok.Type == token.KW_IF {
		return p.parseIf()
	}

	if tok.Type == token.KW_CONSTRUCT {
		return p.parseConstructor()
	}

	if tok.Type == token.KW_BREAK {
		p.consume()
		return &ast.BreakStatement{}
	}

	if tok.Type == token.KW_RETURN {
		p.consume()
		return &ast.ReturnStatement{}
	}

	if tok.Type == token.KW_TRY {
		return p.parseTry()
	}

	// Property declaration with getter/setter?
	hasSetter := false
	for i := 0; p.peek(i) != nil && p.peek(i).Line == tok.Line && p.peek(i).Type != token.EOF; i++ {
		if p.peek(i).Type == token.COLON {
			if i > 0 && p.peek(i-1).Type == token.KW_MAKE {
				hasSetter = true
				break
			}
		}
	}
	if hasSetter {
		return p.parsePropertyDeclaration()
	}

	return p.parseGenericSOV()
}

func (p *Parser) parsePropertyDeclaration() *ast.VariableDeclaration {
	expr := p.parsePrimary()
	name, _ := expr.(*ast.Identifier)

	if p.peek(0) != nil && p.peek(0).Type == "PARTICLE" {
		p.consume()
	}

	typeRef := p.parseTypeRef()

	if p.peek(0) != nil && p.peek(0).Type == "PARTICLE" {
		p.consume()
	}

	p.consume() // 정할 때
	p.consume() // :

	if p.peek(0) != nil && p.peek(0).Type == "INDENT" {
		p.consume()
	}

	var getter []ast.Statement
	var setter *ast.SetterInfo

	for p.peek(0) != nil && p.peek(0).Type != "DEDENT" && p.peek(0).Type != token.EOF {
		tok := p.peek(0)
		if tok.Type == token.KW_GETTER {
			p.consume() // 가져올 때
			p.consume() // :
			getter = p.parseBlock().Statements
		} else if tok.Type == token.KW_SETTER {
			p.consume() // 정할 때

			// Optional (param) ?
			var param *ast.Identifier
			if p.peek(0) != nil && p.peek(0).Type == token.LPAREN {
				p.consume() // (
				param = p.parsePrimary().(*ast.Identifier)
				p.consume() // )
			}
			p.consume() // :
			setter = &ast.SetterInfo{Param: param, Body: p.parseBlock().Statements}
		} else {
			p.consume()
		}
	}

	if p.peek(0) != nil && p.peek(0).Type == "DEDENT" {
		p.consume()
	}

	return &ast.VariableDeclaration{Name: name, TypeRef: typeRef, Getter: getter, Setter: setter, AccessModifier: "public"}
}

func (p *Parser) parseTry() *ast.TryStatement {
	p.consume() // 일단 해보자
	for p.peek(0) != nil && p.peek(0).Type != token.COLON {
		p.consume()
	}
	block := p.parseBlock()

	var handlers []*ast.CatchClause

	for {
		if p.peek(0) == nil {
			break
		}
		// 이 반복이 실제로 오류가 발생했다면 절이 아니라고 판명되면(즉 아래
		// KW_CATCH 검사에서 실패하면), 여기서부터 앞으로 미리 읽은(consume한)
		// 토큰들을 이 지점으로 되돌린다. 안 그러면 try 블록 바로 다음에 오는,
		// 우연히 [타입]/오류로 시작하는 무관한 문장(예: 클래스 선언)의 앞부분
		// 토큰을 조용히 먹어버려 파싱이 깨진다.
		startPos := p.pos

		var errType *ast.TypeReference
		if p.peek(0).Type == "TYPE" {
			errType = p.parseTypeRef()
		} else if p.peek(0).Type == token.IDENT && literalIn(p.peek(0).Literal, p.lang.ErrorLiterals) {
			p.consume()
		} else if p.peek(0).Type != token.KW_CATCH {
			// 카나데는 마커 단어 없이 바로 "発生したら(...)"라고 쓸 수 있음
			// (하리의 "오류가 발생했다면"과 달리 앞에 아무 토큰도 없어도 됨).
			break
		}

		if p.peek(0) != nil && p.peek(0).Type == "PARTICLE" {
			p.consume()
		}

		if p.peek(0) == nil || p.peek(0).Type != token.KW_CATCH {
			p.pos = startPos
			break
		}

		p.consume() // 발생했다면
		var paramName string
		if p.peek(0) != nil && p.peek(0).Type == token.LPAREN {
			p.consume() // (
			if p.peek(0) != nil && (p.peek(0).Type == token.VAR || p.peek(0).Type == token.IDENT) {
				nameTok := p.consume()
				paramName = nameTok.Literal
				if nameTok.Type == token.VAR {
					paramName = paramName[p.lang.DelimLen : len(paramName)-p.lang.DelimLen]
				}
			}
			if p.peek(0) != nil && p.peek(0).Type == token.RPAREN {
				p.consume() // )
			}
		} else {
			paramName = "에러"
		}
		for p.peek(0) != nil && p.peek(0).Type != token.COLON {
			p.consume()
		}
		catchBlock := p.parseBlock()
		handlers = append(handlers, &ast.CatchClause{Type: errType, Param: &ast.Identifier{Value: paramName}, Body: catchBlock})
	}

	var finalizer *ast.BlockStatement
	if p.peek(0) != nil && p.peek(0).Type == token.KW_FINALLY {
		p.consume() // 마무리는 항상
		for p.peek(0) != nil && p.peek(0).Type != token.COLON {
			p.consume()
		}
		finalizer = p.parseBlock()
	}

	return &ast.TryStatement{Block: block, Handlers: handlers, Finalizer: finalizer}
}

func (p *Parser) parseTypeRef() *ast.TypeReference {
	if p.peek(0).Type == "TYPE" {
		t := p.consume()
		val := t.Literal
		if strings.HasPrefix(val, p.lang.TypeOpen) && strings.HasSuffix(val, p.lang.TypeClose) {
			val = val[p.lang.DelimLen : len(val)-p.lang.DelimLen]
		}
		return &ast.TypeReference{Name: val}
	}
	if p.peek(0).Type == token.LBRACKET {
		p.consume() // [
		name := p.consume().Literal
		p.consume() // ]
		return &ast.TypeReference{Name: name}
	}
	return nil
}

func (p *Parser) parseInterface() *ast.InterfaceDeclaration {
	nameNode := p.parseTypeRef()
	if p.peek(0).Type == "PARTICLE" {
		p.consume()
	}
	p.consume() // 규정하자
	block := p.parseBlock()
	return &ast.InterfaceDeclaration{Name: nameNode, Body: block.Statements}
}

func (p *Parser) parseClass() *ast.ClassDeclaration {
	var baseClass *ast.TypeReference
	var interfaces []*ast.TypeReference
	var nameNode *ast.TypeReference
	isAbstract := false

	for p.peek(0) != nil && p.peek(0).Type != token.COLON && p.peek(0).Type != "DEDENT" && p.peek(0).Type != "INDENT" {
		tok := p.peek(0)

		if tok.Type == "TYPE" || tok.Type == token.LBRACKET {
			tNode := p.parseTypeRef()

			if p.peek(0) != nil && p.peek(0).Type == "PARTICLE" {
				p.consume()
			}

			if p.peek(0) != nil && p.peek(0).Type == token.KW_BASE {
				p.consume() // 따라 하길 / 따라하고
				baseClass = tNode
			} else if p.peek(0) != nil && p.peek(0).Type == token.KW_IMPLEMENTS {
				p.consume() // 따르는 / 갖추는
				interfaces = append(interfaces, tNode)
			} else if p.peek(0) != nil && p.peek(0).Type == token.KW_CLASS {
				nameNode = tNode
				isAbstract = p.lang.IsAbstractClassVerb(p.consume().Literal) // 설계하자 / 밑설계하자
				break
			} else {
				nameNode = tNode
			}
		} else if tok.Type == token.KW_CLASS {
			isAbstract = p.lang.IsAbstractClassVerb(p.consume().Literal)
			break
		} else {
			p.consume()
		}
	}

	for p.peek(0) != nil && p.peek(0).Type != token.COLON {
		p.consume()
	}
	if p.peek(0) != nil && p.peek(0).Type == token.COLON {
		p.consume() // :
	}

	block := p.parseBlock()
	return &ast.ClassDeclaration{Name: nameNode, BaseClass: baseClass, Interfaces: interfaces, Body: block.Statements, IsAbstract: isAbstract}
}

func (p *Parser) parseFunctionDecl() ast.Statement {
	var returnType *ast.TypeReference
	isStatic := false

	for p.peek(0) != nil && p.peek(0).Type != "FUNCTION" && p.peek(0).Type != token.COLON {
		tok := p.peek(0)
		if tok.Type == "TYPE" || tok.Type == token.LBRACKET {
			returnType = p.parseTypeRef()
		} else if tok.Type == token.VAR || tok.Type == token.IDENT {
			if literalIn(tok.Literal, p.lang.PluralSelfWords) {
				isStatic = true
			}
			p.consume()
		} else {
			p.consume()
		}
	}

	nameTok := p.consume()
	name := nameTok.Literal
	name = name[p.lang.DelimLen : len(name)-p.lang.DelimLen] // FUNCTION 델리미터 제거
	if p.peek(0) != nil && p.peek(0).Type == "PARTICLE" {
		p.consume()
	}
	action := p.consume()

	// 파라미터 파싱
	if p.peek(0) != nil && p.peek(0).Type == token.LPAREN {
		p.consume() // (
	}

	var params []*ast.Parameter
	for p.peek(0) != nil && p.peek(0).Type != token.RPAREN {
		tok := p.peek(0)
		var typeAnn *ast.TypeReference
		if tok.Type == token.LBRACKET || tok.Type == "TYPE" {
			typeAnn = p.parseTypeRef()
			if p.peek(0) != nil && (p.peek(0).Type == "TYPE_IN" || p.peek(0).Literal == p.lang.TypeInWord) {
				p.consume() // 인
			}
		}

		var paramName *ast.Identifier
		if p.peek(0) != nil && (p.peek(0).Type == token.VAR || p.peek(0).Type == token.IDENT) {
			expr := p.parsePrimary()
			if id, ok := expr.(*ast.Identifier); ok {
				paramName = id
			}
		} else if p.peek(0) != nil && p.peek(0).Type == "PARTICLE" {
			p.consume()
			continue
		} else {
			p.consume() // unknown
		}

		if paramName != nil {
			var defaultVal ast.Expression
			if p.peek(0) != nil && p.peek(0).Type == token.ASSIGN {
				p.consume() // =
				defaultVal = p.parseExpression()
			}
			params = append(params, &ast.Parameter{Name: paramName, TypeAnnotation: typeAnn, Default: defaultVal})
		}

		if p.peek(0) != nil && (p.peek(0).Literal == "," || p.peek(0).Type == "PARTICLE") {
			p.consume()
		}
	}
	if p.peek(0) != nil && p.peek(0).Type == token.RPAREN {
		p.consume() // )
	}

	access := p.lang.AccessModifierFromVerb(action.Literal)

	if action.Type == token.KW_MUST_HAVE {
		return &ast.InterfaceMethod{Name: &ast.Identifier{Value: name}}
	}

	block := p.parseBlock()
	return &ast.FunctionDeclaration{Name: &ast.Identifier{Value: name}, Params: params, Body: block, AccessModifier: access, IsStatic: isStatic, ReturnType: returnType}
}

func (p *Parser) parseIf() *ast.IfStatement {
	p.consume() // 만약 / もし
	return p.parseIfBody()
}

// parseIfBody parses everything after the leading if-keyword: the
// condition, block, and any else/else-if tail. Split out from parseIf so
// KW_ELIF (카나데: "もしくは", a single word that means "else, and here's
// another if" in one token — unlike 하자's "그렇지 않고 만약", two separate
// tokens) can recurse into it directly without expecting a leading if-
// keyword to consume first.
func (p *Parser) parseIfBody() *ast.IfStatement {
	cond := p.parseCondition()
	for p.peek(0) != nil && p.peek(0).Type != token.COLON {
		p.consume()
	}
	block := p.parseBlock()

	var alt *ast.BlockStatement
	if p.peek(0) != nil && p.peek(0).Type == token.KW_ELSE {
		p.consume() // 그렇지 않다면 / 그렇지 않고
		if p.peek(0) != nil && p.peek(0).Type == token.KW_IF {
			alt = &ast.BlockStatement{Statements: []ast.Statement{p.parseIf()}}
		} else {
			for p.peek(0) != nil && p.peek(0).Type != token.COLON {
				p.consume()
			}
			alt = p.parseBlock()
		}
	} else if p.peek(0) != nil && p.peek(0).Type == token.KW_ELIF {
		p.consume() // もしくは
		alt = &ast.BlockStatement{Statements: []ast.Statement{p.parseIfBody()}}
	}

	return &ast.IfStatement{Condition: cond, Consequent: block, Alternate: alt}
}

// parseCondition parses a full condition expression: one comparison/operand,
// optionally chained with 그리고/또는 into a LogicalExpression (spec 2.4 —
// mixing them requires parens on each side, e.g. `(A) 그리고 (B) 또는 (C)`),
// followed by an optional trailing '라면'.
func (p *Parser) parseCondition() ast.Expression {
	cond := p.parseConditionOperand()

	for p.peek(0) != nil && (p.peek(0).Type == token.KW_AND || p.peek(0).Type == token.KW_OR) {
		opTok := p.consume()
		op := "그리고"
		if opTok.Type == token.KW_OR {
			op = "또는"
		}
		right := p.parseConditionOperand()
		cond = &ast.LogicalExpression{Left: cond, Operator: op, Right: right}
	}

	// '라면' 같은 식별자 무시
	if p.peek(0) != nil && p.peek(0).Type == token.IDENT && literalIn(p.peek(0).Literal, p.lang.ConditionThenWords) {
		p.consume()
	}

	return cond
}

// parseConditionOperand parses a single condition operand: either a
// parenthesized (possibly itself compound) condition, or one comparison in
// SVO (`A 가 B 보다 크다`) or SOV (`A B 크다`) word order.
func (p *Parser) parseConditionOperand() ast.Expression {
	var cond ast.Expression

	if p.peek(0) != nil && p.peek(0).Type == token.LPAREN {
		start, errorCount, diagCount := p.pos, len(p.errors), len(p.diags)
		p.consume()
		cond = p.parseCondition()
		if p.peek(0) != nil && p.peek(0).Type == token.RPAREN {
			p.consume()
		}
		if next := p.peek(0); next == nil || endsCondition(next.Type) {
			return cond
		}
		// The group was not a whole condition but the start of an expression:
		// `(('x' % 2) == 0)`, `(('x' * 2) + 1) >= 9`. Read it again as one.
		p.pos, p.errors, p.diags = start, p.errors[:errorCount], p.diags[:diagCount]
	}
	return p.finishComparison(p.parseExpression())
}

// endsCondition reports whether a token can follow a complete condition: the
// close of an enclosing group, the block's colon, 그리고/또는, or the trailing
// 라면.
func endsCondition(t token.TokenType) bool {
	return t == token.RPAREN || t == token.COLON || t == token.KW_AND || t == token.KW_OR || t == token.IDENT
}

// finishComparison completes a comparison whose left side has been parsed: the
// operator and right side after it, in SVO (`A 가 B 보다 크다`) or SOV word order.
func (p *Parser) finishComparison(cond ast.Expression) ast.Expression {
	if p.peek(0) != nil && p.peek(0).Type == "PARTICLE" {
		p.consume()
	}

	if p.peek(0) != nil && p.peek(0).Type != "COMPARE" && p.peek(0).Type != token.RPAREN && p.peek(0).Type != token.COLON && p.peek(0).Type != token.IDENT && p.peek(0).Type != token.KW_AND && p.peek(0).Type != token.KW_OR {
		// Possibly SOV: Left Right Compare
		right := p.parseExpression()
		if p.peek(0) != nil && p.peek(0).Type == "PARTICLE" {
			p.consume()
		}
		if p.peek(0) != nil && p.peek(0).Type == "COMPARE" {
			op := p.lang.NormalizeCompareOpSOV(p.consume().Literal)
			cond = &ast.BinaryExpression{Left: cond, Operator: op, Right: right}
		} else {
			// Fallback or error, but let's assume it was just part of something else
		}
	} else if p.peek(0) != nil && p.peek(0).Type == "COMPARE" {
		// SVO: Left Compare Right
		op := p.lang.NormalizeCompareOpSVO(p.consume().Literal)
		right := p.parseExpression()
		cond = &ast.BinaryExpression{Left: cond, Operator: op, Right: right}
	}

	return cond
}

// listPosition은 "앞에"/"뒤에서" 같은 조사를 보고 AST 스펙이 정한 값("front"/"back")으로
// 정규화한다. 원본 한국어 조사를 그대로 AST에 흘려보내지 않아야, kanade 같은 다른 로케일의
// 파서가 같은 AST 노드를 만들 때도 동일한 값을 쓸 수 있다.
func (p *Parser) listPosition(particles []string) string {
	if len(particles) > 0 && strings.Contains(particles[0], p.lang.FrontMarker) {
		return "front"
	}
	return "back"
}

type Component struct {
	Expr      ast.Expression
	Particles []string
	Type      *ast.TypeReference // `[타입]인 값`'s annotation, when the component had one
}

func (p *Parser) parseGenericSOV() ast.Statement {
	components := []Component{}
	for {
		tok := p.peek(0)
		if tok == nil {
			break
		}
		if tok.Type == token.KW_MAKE || tok.Type == token.KW_EXECUTE || tok.Type == token.KW_PRINT || tok.Type == token.KW_RETURN || tok.Type == token.KW_LOOP || tok.Type == token.KW_PUSH || tok.Type == token.KW_POP || tok.Type == token.KW_THROW || tok.Type == token.KW_MUST_HAVE || tok.Type == token.KW_IMPORT || tok.Type == token.KW_ADD || tok.Type == token.KW_SUB || tok.Type == token.KW_SWITCH || tok.Type == token.KW_FALLTHROUGH || tok.Type == token.KW_INPUT {
			break
		}
		p.declaredType = nil
		expr := p.parseExpression()
		declared := p.declaredType
		p.declaredType = nil
		var parts []string
		for p.peek(0) != nil && (p.peek(0).Type == token.COMMA || p.peek(0).Type == "PARTICLE" || p.peek(0).Type == token.KW_FROM || p.peek(0).Type == "TYPE_IN" || p.peek(0).Type == token.KW_FRONT || p.peek(0).Type == token.KW_BACK) {
			tok := p.consume()
			if tok.Type == "PARTICLE" || tok.Type == token.KW_FRONT || tok.Type == token.KW_BACK {
				parts = append(parts, tok.Literal)
			}
		}
		components = append(components, Component{Expr: expr, Particles: parts, Type: declared})
	}

	if p.peek(0) == nil || p.peek(0).Type == token.EOF {
		return nil
	}
	verb := p.consume()

	if verb.Type == token.KW_ADD || verb.Type == token.KW_SUB {
		target := components[0].Expr
		val := components[1].Expr
		op := "+"
		if verb.Type == token.KW_SUB {
			op = "-"
		}
		return &ast.Assignment{Target: target, Value: &ast.BinaryExpression{Left: target, Operator: op, Right: val}}
	}
	if verb.Type == token.KW_MAKE {
		target := components[0].Expr
		var val ast.Expression
		if len(components) > 1 {
			val = components[1].Expr
		}

		var declared *ast.TypeReference
		if len(components) > 1 {
			declared = components[1].Type
		}
		access := p.lang.AccessModifierFromVerb(verb.Literal)
		isConst := p.lang.IsConstVerb(verb.Literal)

		if id, ok := target.(*ast.Identifier); ok {
			return &ast.VariableDeclaration{Name: id, TypeRef: declared, Value: val, AccessModifier: access, IsConstant: isConst}
		} else if mem, ok := target.(*ast.MemberExpression); ok {
			// StaticReference only ever comes from a *bare* 우리/私たち
			// (parsePrimary's IDENT case) — the quoted form ('우리'/『私たち』,
			// what every real example actually uses) parses as a plain
			// Identifier by design (see vm/eval_expr.go and
			// bytecode.compileExpression's own Identifier-value checks), so
			// this has to check for that shape too or every quoted static
			// field declaration silently becomes a plain instance field.
			_, isStaticRef := mem.Object.(*ast.StaticReference)
			isQuotedStatic := false
			if id, ok := mem.Object.(*ast.Identifier); ok {
				isQuotedStatic = literalIn(id.Value, p.lang.PluralSelfWords)
			}
			if isStaticRef || isQuotedStatic {
				if propId, ok := mem.Property.(*ast.Identifier); ok {
					return &ast.VariableDeclaration{Name: propId, TypeRef: declared, Value: val, AccessModifier: access, IsStatic: true}
				}
			}
			return &ast.Assignment{Target: target, Value: val}
		} else {
			return &ast.Assignment{Target: target, Value: val}
		}
	}
	if verb.Type == token.KW_EXECUTE {
		return &ast.ExpressionStatement{Expression: components[0].Expr}
	}
	if verb.Type == token.KW_RETURN {
		if len(components) > 0 {
			return &ast.ReturnStatement{Value: components[0].Expr}
		}
		return &ast.ReturnStatement{}
	}
	if verb.Type == token.KW_PRINT {
		newLine := !p.lang.IsPrintInlineVerb(verb.Literal)
		return &ast.PrintStatement{Value: components[0].Expr, NewLine: newLine}
	}
	if verb.Type == token.KW_LOOP {
		var loopVar string
		if p.peek(0) != nil && p.peek(0).Type == token.LPAREN {
			p.consume() // (
			if p.peek(0) != nil && p.peek(0).Type == token.VAR {
				loopVar = p.consume().Literal
				loopVar = loopVar[p.lang.DelimLen : len(loopVar)-p.lang.DelimLen]
			}
			if p.peek(0) != nil && p.peek(0).Type == token.RPAREN {
				p.consume() // )
			}
		}
		if len(components) >= 1 {
			switch p.lang.ClassifyLoop(verb, components) {
			case LoopForEach:
				return &ast.ForEachLoop{List: components[0].Expr, Body: p.parseBlock()}
			case LoopWhile:
				return &ast.WhileLoop{Condition: components[0].Expr, Body: p.parseBlock()}
			default:
				if len(components) >= 2 {
					return &ast.ForRangeStatement{Start: components[0].Expr, End: components[1].Expr, LoopVar: loopVar, Body: p.parseBlock()}
				}
			}
		}
	}
	if verb.Type == token.KW_PUSH {
		// '목록' 뒤에 '값'을 추가하자
		if len(components) >= 2 {
			return &ast.ListPushStatement{Target: components[0].Expr, Value: components[1].Expr, Position: p.listPosition(components[0].Particles)}
		}
	}
	if verb.Type == token.KW_POP {
		if len(components) >= 1 {
			return &ast.ListPopStatement{Target: components[0].Expr, Position: p.listPosition(components[0].Particles)}
		}
	}
	if verb.Type == token.KW_INPUT {
		targetId, _ := components[0].Expr.(*ast.Identifier)
		var typeAnn *ast.TypeReference
		if len(components) > 1 {
			if t, ok := components[1].Expr.(*ast.TypeReference); ok {
				typeAnn = t
			}
		}
		return &ast.InputStatement{Target: targetId, TypeRef: typeAnn}
	}
	if verb.Type == token.KW_FALLTHROUGH {
		return &ast.FallthroughStatement{}
	}
	if verb.Type == token.KW_THROW {
		if len(components) > 0 {
			return &ast.ThrowStatement{Value: components[0].Expr}
		}
	}
	if verb.Type == token.KW_MUST_HAVE {
		if len(components) > 0 {
			// 인터페이스 메서드 규정 - V1 단순화 (아무것도 안 하거나 더미 반환)
			// 현재 AST에 InterfaceMethodNode가 없으므로 임시로 빈 Statement 반환
			return &ast.ExpressionStatement{Expression: components[0].Expr}
		}
	}
	if verb.Type == token.KW_SWITCH {
		// '회원등급'에 따라 나누자:
		if len(components) > 0 {
			stmt := &ast.SwitchStatement{Discriminant: components[0].Expr, Cases: []*ast.SwitchCase{}}
			if p.peek(0) != nil && p.peek(0).Type == token.COLON {
				p.consume()
			}
			if p.peek(0) != nil && p.peek(0).Type == "INDENT" {
				p.consume()
			}
			for p.peek(0) != nil && p.peek(0).Type != "DEDENT" && p.peek(0).Type != token.EOF {
				if p.peek(0).Type == "INDENT" {
					p.consume()
					continue
				}
				if p.peek(0).Type == token.KW_DEFAULT {
					p.consume() // 나머지는
					if p.peek(0) != nil && p.peek(0).Type == token.COLON {
						p.consume()
					}
					cBody := p.parseBlock()
					stmt.Cases = append(stmt.Cases, &ast.SwitchCase{IsDefault: true, Consequent: cBody})
				} else {
					// "VIP", "일반" 인 경우: (카나데: "인" 마커 없이 바로
					// "「VIP」の場合:"처럼 KW_CASE가 곧장 나올 수도 있음)
					tests := []ast.Expression{}
					for p.peek(0) != nil {
						if p.peek(0).Type == "TYPE_IN" {
							p.consume() // 인
							if p.peek(0) != nil && p.peek(0).Type == token.KW_CASE {
								p.consume() // 경우
							}
							break
						}
						if p.peek(0).Type == token.KW_CASE {
							p.consume() // 場合
							break
						}
						expr := p.parseExpression()
						tests = append(tests, expr)
						if p.peek(0) != nil && p.peek(0).Type == token.COMMA {
							p.consume()
						}
					}
					if p.peek(0) != nil && p.peek(0).Type == token.COLON {
						p.consume()
					}
					cBody := p.parseBlock()
					stmt.Cases = append(stmt.Cases, &ast.SwitchCase{Tests: tests, Consequent: cBody})
				}
			}
			if p.peek(0) != nil && p.peek(0).Type == "DEDENT" {
				p.consume()
			}
			return stmt
		}
	}
	if verb.Type == token.KW_IMPORT {
		if len(components) >= 2 {
			module := ""
			isBuiltin := false

			if id, ok := components[0].Expr.(*ast.Identifier); ok {
				module = id.Value
			} else if str, ok := components[0].Expr.(*ast.StringLiteral); ok {
				module = str.Value
			} else if typeRef, ok := components[0].Expr.(*ast.TypeReference); ok {
				module = typeRef.Name
				isBuiltin = true
			} else if list, ok := components[0].Expr.(*ast.ListLiteral); ok {
				if len(list.Elements) > 0 {
					if id, ok := list.Elements[0].(*ast.Identifier); ok {
						module = id.Value
						isBuiltin = true
					}
				}
			}

			rest := components[1:]

			// [모듈]에서 전부 가져오자: 조사 없는 "전부" 하나
			if len(rest) == 1 && len(rest[0].Particles) == 0 {
				if id, ok := rest[0].Expr.(*ast.Identifier); ok && id.Value == p.lang.ImportAllWord {
					return &ast.ImportStatement{Module: module, IsBuiltin: isBuiltin, All: true}
				}
			}

			// <이름>을 <별칭>으로 가져오자 (이름 충돌을 피하는 형태):
			// 두 번째 컴포넌트가 "로"/"으로" 조사를 달고 있으면 로컬
			// 바인딩 이름을 별칭으로 바꾼다. 항목이 여럿이면(<a>와 <b>를)
			// 별칭 없이 각각을 그대로 가져온다.
			if len(rest) == 2 && hasParticle(rest[1].Particles, p.lang.ImportAsParticles...) {
				return &ast.ImportStatement{Module: module, IsBuiltin: isBuiltin,
					Items: []ast.ImportItem{{Name: importNameFromExpr(rest[0].Expr), As: importNameFromExpr(rest[1].Expr)}}}
			}
			items := make([]ast.ImportItem, 0, len(rest))
			for _, comp := range rest {
				items = append(items, ast.ImportItem{Name: importNameFromExpr(comp.Expr)})
			}
			return &ast.ImportStatement{Module: module, IsBuiltin: isBuiltin, Items: items}
		}
	}

	return nil
}

// importNameFromExpr extracts a plain name from the shapes an import
// target/alias can take: a quoted identifier ('이름'), a plain string
// ("이름"), or a function reference (<이름>).
func importNameFromExpr(expr ast.Expression) string {
	if id, ok := expr.(*ast.Identifier); ok {
		return id.Value
	}
	if str, ok := expr.(*ast.StringLiteral); ok {
		return str.Value
	}
	if ref, ok := expr.(*ast.FunctionReference); ok {
		return ref.Name
	}
	return ""
}

func hasParticle(particles []string, want ...string) bool {
	for _, p := range particles {
		for _, w := range want {
			if p == w {
				return true
			}
		}
	}
	return false
}

func (p *Parser) ParseExpression() ast.Expression {
	return p.parseExpression()
}

func (p *Parser) parseExpression() ast.Expression {
	return p.parseBinary(1)
}

// binaryPrecedence is the usual arithmetic order: * / % bind tighter than + -.
func binaryPrecedence(op string) int {
	switch op {
	case "*", "/", "%":
		return 2
	}
	return 1
}

// parseBinary reads operators of at least minPrec, left to right within one level.
func (p *Parser) parseBinary(minPrec int) ast.Expression {
	expr := p.parseMemberAndCall()
	for {
		tok := p.peek(0)
		if tok == nil || tok.Type != "OP" {
			break
		}
		prec := binaryPrecedence(tok.Literal)
		if prec < minPrec {
			break
		}
		op := p.consume().Literal
		right := p.parseBinary(prec + 1)
		expr = &ast.BinaryExpression{Left: expr, Operator: op, Right: right}
	}
	return expr
}

func (p *Parser) parseMemberAndCall() ast.Expression {
	expr := p.parsePrimary()

	for {
		tok := p.peek(0)
		if tok == nil {
			break
		}
		if tok.Type == "PARTICLE" && tok.Literal == p.lang.MemberParticle &&
			p.peek(1) != nil && p.peek(1).Type == token.IDENT && p.peek(1).Literal == p.lang.PoppedValueWord {
			// 카나데는 인덱싱 뒤에 조사로 연결된 장식용 "の値"를 쓴다
			// ("『果物』の1番目の値" = "그 목록의 1번째의 값" = 그냥
			// "그 목록의 1번째"). 하리는 이 단어를 조사 없이 바로 붙여
			//써서(예: "1번째 값") 애초에 MemberExpression 체인에 들어오지
			// 않고 바깥 조사 수집 루프에서 그냥 버려지는데, 카나데는 "의"로
			// 연결하는 바람에 별도 멤버 접근처럼 보여 그냥 두면 인덱싱
			// 결과(문자열/숫자)에 없는 '값'이라는 필드/변수를 찾으려 든다.
			// 여기서 미리 감지해 그대로 건너뛴다(체인 값은 그대로 유지).
			p.consume() // の
			p.consume() // 값/値
		} else if tok.Type == "PARTICLE" && tok.Literal == p.lang.MemberParticle &&
			!(p.peek(1) != nil && (p.peek(1).Type == token.KW_FRONT || p.peek(1).Type == token.KW_BACK)) {
			p.consume()
			prop := p.parsePrimary()
			expr = &ast.MemberExpression{Object: expr, Property: prop}
		} else if tok.Type == "PARTICLE" && tok.Literal == p.lang.MemberParticle {
			// 카나데는 앞/뒤 마커 앞에도 조사를 붙인다("『果物』の前に..." —
			// 하자라면 "'과일' 앞에"처럼 조사 없이 바로 씀). member-access로
			// 잘못 삼키지 않도록 그냥 건너뛰고, 다음 루프에서 KW_FRONT/BACK
			// 분기(바로 아래, 또는 바깥 parseGenericSOV의 조사 수집 루프)가
			// 이어서 처리하게 한다.
			p.consume()
		} else if tok.Type == token.LPAREN {
			p.consume() // (
			args := []ast.Expression{}
			for p.peek(0) != nil && p.peek(0).Type != token.RPAREN {
				args = append(args, p.parseExpression())
				if p.peek(0) != nil && (p.peek(0).Type == token.COMMA || p.peek(0).Type == "PARTICLE") {
					p.consume()
				}
			}
			if p.peek(0) != nil && p.peek(0).Type == token.RPAREN {
				p.consume() // )
			}
			expr = &ast.CallExpression{Callee: expr, Arguments: args}
		} else if (tok.Type == token.KW_FRONT || tok.Type == token.KW_BACK) && p.peek(1) != nil && p.peek(1).Type == token.KW_POPPED {
			// '목록' 앞에서/뒤에서 꺼낸 (값) — ListPopStatement(문장형 꺼내자)와 달리
			// 값 자체가 필요한 표현식 형태.
			pos := "back"
			if tok.Type == token.KW_FRONT {
				pos = "front"
			}
			p.consume() // 앞에서/뒤에서
			p.consume() // 꺼낸
			if p.peek(0) != nil && p.peek(0).Type == token.IDENT && p.peek(0).Literal == p.lang.PoppedValueWord {
				p.consume() // 값 (가독성용 옵션 토큰)
			}
			expr = &ast.ListPopExpression{Target: expr, Position: pos}
		} else {
			break
		}
	}
	return expr
}

func (p *Parser) parsePrimary() ast.Expression {
	tok := p.consume()
	if tok.Type == "OP" && tok.Literal == "-" {
		next := p.consume()
		if next.Type == token.INT {
			val, _ := strconv.ParseFloat("-"+next.Literal, 64)
			return &ast.NumberLiteral{Value: val}
		}
		return &ast.StringLiteral{Value: "알수없음: -"}
	}
	if tok.Type == token.STRING {
		val := tok.Literal[p.lang.DelimLen : len(tok.Literal)-p.lang.DelimLen]
		return &ast.StringLiteral{Value: val}
	}
	if tok.Type == "TEMPLATE_STRING" {
		val := tok.Literal
		if strings.HasPrefix(val, p.lang.TemplatePrefix) {
			val = val[len(p.lang.TemplatePrefix):]
		}
		if strings.HasSuffix(val, p.lang.TemplateSuffix) {
			val = val[:len(val)-len(p.lang.TemplateSuffix)]
		}
		return &ast.TemplateLiteral{Value: val}
	}
	if tok.Type == token.VAR {
		val := tok.Literal[p.lang.DelimLen : len(tok.Literal)-p.lang.DelimLen]
		return &ast.Identifier{Value: val}
	}
	if tok.Type == "FUNCTION" {
		val := tok.Literal[p.lang.DelimLen : len(tok.Literal)-p.lang.DelimLen] // <공격> -> 공격
		if val == p.lang.ConstructorFunctionName {
			val = "__init__"
		}
		return &ast.FunctionReference{Name: val}
	}
	if tok.Type == token.KW_NULL {
		return &ast.NullLiteral{}
	}
	if tok.Type == token.KW_TRUE {
		return &ast.BooleanLiteral{Value: true}
	}
	if tok.Type == token.KW_FALSE {
		return &ast.BooleanLiteral{Value: false}
	}
	if tok.Type == token.INT {
		val, _ := strconv.ParseFloat(tok.Literal, 64)
		return &ast.NumberLiteral{Value: val}
	}
	if tok.Type == token.KW_SELF {
		return &ast.SelfReference{}
	}
	if tok.Type == token.KW_PARENT {
		return &ast.SuperReference{}
	}
	if tok.Type == token.IDENT {
		if literalIn(tok.Literal, p.lang.PluralSelfWords) {
			return &ast.StaticReference{}
		}
		return &ast.Identifier{Value: tok.Literal}
	}
	if tok.Type == token.KW_NEW {
		cls := p.parseTypeRef()
		var args []ast.Expression
		if p.peek(0) != nil && p.peek(0).Type == token.LPAREN {
			p.consume() // (
			for p.peek(0) != nil && p.peek(0).Type != token.RPAREN {
				args = append(args, p.parseExpression())
				if p.peek(0) != nil && (p.peek(0).Type == token.COMMA || p.peek(0).Type == "PARTICLE") {
					p.consume()
				}
			}
			if p.peek(0) != nil && p.peek(0).Type == token.RPAREN {
				p.consume() // )
			}
		}
		return &ast.NewExpression{Class: cls, Arguments: args}
	}
	if tok.Type == "TYPE" {
		val := tok.Literal
		if strings.HasPrefix(val, p.lang.TypeOpen) && strings.HasSuffix(val, p.lang.TypeClose) {
			val = val[p.lang.DelimLen : len(val)-p.lang.DelimLen]
		}
		typeRef := &ast.TypeReference{Name: val}

		if p.peek(0) != nil && (p.peek(0).Type == "TYPE_IN" || p.peek(0).Literal == p.lang.TypeInWord) {
			// 카나데는 TYPE_IN("[타입]인 값")과 멤버 접근 조사를 같은 단어
			// "の"로 겸용한다(kanade-docs 실측). 뒤에 오는 게 값처럼 보이면
			// (숫자/문자열/목록/새로운 등) 기존처럼 타입 주석으로 보고
			// 값만 반환하지만, FUNCTION이 오면 그건 값이 아니라
			// "TYPE의 〈정적메서드〉()" 같은 정적 멤버 접근이다 — 이 경우
			// TYPE 자체를 버리지 않고 MemberExpression의 Object로 남겨야
			// 한다. TYPE이 또 오는 경우(kanade-docs가 종종 같은 클래스명을
			// "TYPE의 TYPE의 〈메서드〉()"처럼 중복해서 씀)는 바깥 TYPE을
			// 버리고 안쪽을 그대로 재귀 파싱한다 — 그래야 Property가
			// MemberExpression 자체가 되는(평가 쪽이 다루지 못하는) 이중
			// 중첩을 피할 수 있다.
			if p.peek(1) != nil && p.peek(1).Type == "TYPE" {
				p.consume() // の
				return p.parsePrimary()
			}
			// 이 읽기는 TYPE_IN 낱말이 곧 멤버 접근 조사일 때(카나데의 "の")만 맞다.
			// 하리의 정적 호출은 조사 "의"로 쓰고("[상자]의 <만들기>()"), "인"은 언제나
			// 타입 표시라서 "[상자]인 <만들기>()"는 타입이 붙은 함수 호출 값이다.
			if p.peek(1) != nil && p.peek(1).Type == "FUNCTION" && p.peek(0).Literal == p.lang.MemberParticle {
				p.consume() // の
				prop := p.parsePrimary()
				return &ast.MemberExpression{Object: typeRef, Property: prop}
			}
			p.consume() // 인
			res := p.parsePrimary()
			p.declaredType = typeRef
			return res
		}
		return typeRef
	}
	if tok.Type == token.LBRACKET {
		// [] or [...]
		list := &ast.ListLiteral{Elements: []ast.Expression{}}
		for p.peek(0) != nil && p.peek(0).Type != token.RBRACKET {
			list.Elements = append(list.Elements, p.parseExpression())
			if p.peek(0) != nil && (p.peek(0).Type == token.COMMA || p.peek(0).Type == "PARTICLE") {
				p.consume()
			}
		}
		if p.peek(0) != nil && p.peek(0).Type == token.RBRACKET {
			p.consume()
		}
		return list
	}
	if tok.Type == token.LBRACE {
		dict := &ast.DictLiteral{Properties: []*ast.Property{}}
		for p.peek(0) != nil && p.peek(0).Type != token.RBRACE {
			key := p.parseExpression()
			if p.peek(0) != nil && p.peek(0).Type == token.COLON {
				p.consume()
			}
			val := p.parseExpression()
			dict.Properties = append(dict.Properties, &ast.Property{Key: key, Value: val})
			if p.peek(0) != nil && (p.peek(0).Type == token.COMMA || p.peek(0).Type == "PARTICLE") {
				p.consume()
			}
		}
		if p.peek(0) != nil && p.peek(0).Type == token.RBRACE {
			p.consume()
		}
		return dict
	}
	if tok.Type == token.LPAREN {
		expr := p.parseCondition()
		if p.peek(0) != nil && p.peek(0).Type == token.RPAREN {
			p.consume()
		}
		return expr
	}
	line := tok.Line
	if tok.Literal == "" {
		// Ran off the end: point at the last line that has code, not at the
		// empty line after it.
		line = p.lastContentLine()
	}
	p.diags = append(p.diags, Diagnostic{Kind: DiagUnknownToken, Line: line, Col: tok.Col, Length: utf8.RuneCountInString(tok.Literal), Literal: tok.Literal})
	return &ast.StringLiteral{Value: fmt.Sprintf("알수없음: %s", tok.Literal)}
}

func (p *Parser) parseConstructor() *ast.ConstructorDeclaration {
	p.consume()
	var params []*ast.Parameter
	if p.peek(0) != nil && p.peek(0).Type == token.LPAREN {
		p.consume()
		for p.peek(0) != nil && p.peek(0).Type != token.RPAREN {
			var typeAnn *ast.TypeReference
			if p.peek(0).Type == token.LBRACKET || p.peek(0).Type == "TYPE" {
				typeAnn = p.parseTypeRef()
				if p.peek(0) != nil && (p.peek(0).Type == "TYPE_IN" || p.peek(0).Literal == p.lang.TypeInWord) {
					p.consume()
				}
			}
			var paramName *ast.Identifier
			if p.peek(0) != nil && p.peek(0).Type == token.VAR {
				expr := p.parsePrimary()
				if id, ok := expr.(*ast.Identifier); ok {
					paramName = id
				}
			} else {
				p.consume()
			}
			if paramName != nil {
				var defaultVal ast.Expression
				if p.peek(0) != nil && p.peek(0).Type == token.ASSIGN {
					p.consume() // =
					defaultVal = p.parseExpression()
				}
				params = append(params, &ast.Parameter{Name: paramName, TypeAnnotation: typeAnn, Default: defaultVal})
			}
		}
		p.consume()
	}
	if p.peek(0) != nil && p.peek(0).Type == token.KW_DO_AS {
		p.consume()
	}
	for p.peek(0) != nil && p.peek(0).Type != token.COLON {
		p.consume()
	}
	block := p.parseBlock()
	return &ast.ConstructorDeclaration{Id: &ast.Identifier{Value: p.lang.ConstructorFunctionName}, Params: params, Body: block.Statements}
}
