package bytecode

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/soumt-r/hana/ast"
	haja_lexer "github.com/soumt-r/hana/lexer/haja"
	kanade_lexer "github.com/soumt-r/hana/lexer/kanade"
	haja_parser "github.com/soumt-r/hana/parser/haja"
	kanade_parser "github.com/soumt-r/hana/parser/kanade"
)

// Compiler turns an *ast.Program into bytecode. It covers variables,
// arithmetic/comparison/logical operators, if/while/for-range, functions with
// default parameters and calls, return/break, print, list
// literals/push/pop/index/length, and switch/for-each/classes/interfaces/
// try-catch/import.
// Not covered: dynamic reflection (<'변수'>()), operator-overloading magic
// methods (기호 같다), the string/list builtin pseudo-methods (자르기/
// 바꾸기/...) (compileStatement/compileExpression report an error for
// anything unsupported rather than silently miscompiling it).
type Compiler struct {
	program     *Program
	breakJumps  [][]int  // stack of pending break-JUMP indices, one slice per enclosing loop
	loopScopes  []string // per enclosing loop: the marker of its per-iteration scope, "" when the body declares nothing
	tempCounter int
	errors      []string
	tryScopes   []*tryScope // stack of enclosing try statements, innermost last

	// importStack tracks which local files are currently being compiled, to
	// reject a circular "파일"에서 ... 가져오자 chain with a clear error
	// instead of recursing forever. Shared (same map instance) across every
	// sub-Compiler compileLocalImport creates for the files it pulls in, so
	// the check covers the whole chain, not just one file's direct imports.
	importStack map[string]bool

	// lang carries the handful of things that differ by source language
	// but aren't reachable through the AST alone: the literal self/static
	// words a *ast.Identifier can be ("나"/"우리" vs "私"/"私たち" — same
	// parse-time-vs-runtime-string-check design as parser/haja's
	// listPosition and vm/eval_expr.go's Identifier case) and how to
	// lex+parse a template literal's {...} interpolation, which is real
	// source text compiled fresh, not something the parser already turned
	// into AST. See NewCompiler/NewKanadeCompiler.
	lang *bcLang

	// pkgBinds holds, per `[모듈]에서 …` statement, the names it binds at run time
	// (see package_import.go).
	pkgBinds map[*ast.ImportStatement][]nativeBind

	// libraries are the packages whose native library the program binds when it
	// runs, in first-use order (see LibraryModules).
	libraries []string
}

type bcLang struct {
	name              string // language key (haja/kanade), as in hana.pkg.json
	ext               string // source extension of its entry points
	nativePrefix      string // marks an import from a package's native library
	selfWord          string
	pluralSelfWord    string
	listClearMethod   string
	defaultItemName   string // loop variable when `각각` has no explicit name
	parseExpr         func(code string) ast.Expression
	builtinErrorClass *ast.ClassDeclaration
}

var hajaLang = &bcLang{
	name:            "haja",
	ext:             ".hj",
	nativePrefix:    "네이티브_",
	selfWord:        "나",
	pluralSelfWord:  "우리",
	listClearMethod: "비우기",
	defaultItemName: "아이템",
	parseExpr: func(code string) ast.Expression {
		l := haja_lexer.New(code)
		p := haja_parser.New(l)
		return p.ParseExpression()
	},
}

var kanadeLang = &bcLang{
	name:            "kanade",
	ext:             ".knd",
	nativePrefix:    "ネイティブ_",
	selfWord:        "私",
	pluralSelfWord:  "私たち",
	listClearMethod: "空にする",
	defaultItemName: "アイテム",
	parseExpr: func(code string) ast.Expression {
		l := kanade_lexer.New(code)
		p := kanade_parser.New(l)
		return p.ParseExpression()
	},
}

// tryScope tracks one enclosing try statement while compiling its protected
// block and catch handlers, so a ReturnStatement/BreakStatement compiled
// anywhere inside can emit the right TRY_POPs/RUN_FINALLYs on its way out —
// see emitTryExits.
type tryScope struct {
	finallyChunk     *Chunk // nil if this try has no 마무리는 항상
	inProtectedBlock bool   // false once compiling catch handlers (that
	// level's TRY_PUSH entry is already popped by then — see compileTry)
}

func NewCompiler() *Compiler {
	return newCompiler(hajaLang)
}

// NewKanadeCompiler is NewCompiler for a 카나데(Kanade) source program: same
// bytecode compiler, only the self/static words and the template-literal
// interpolation lexer/parser (and hence the builtin [에러] class's own
// source) differ. See Compiler.lang.
func NewKanadeCompiler() *Compiler {
	return newCompiler(kanadeLang)
}

func newCompiler(lang *bcLang) *Compiler {
	c := &Compiler{
		program: &Program{
			Functions:  map[string]*Function{},
			Classes:    map[string]*ClassInfo{},
			Interfaces: map[string]*InterfaceInfo{},
		},
		lang: lang,
	}
	c.registerClass(lang.builtinErrorClass)
	return c
}

// builtinErrorClassSource is the same [오류] class vm.newBuiltinErrorClass
// builds by hand (AST 스펙 4.5: 발생된 에러 객체는 내장된 [오류] 클래스의
// 인스턴스이며 기본적으로 '메시지' 속성을 가진다). Unlike the tree-walker,
// the bytecode compiler already depends on the lexer/parser (for template
// literals), so there's no reason not to just write it as real source and
// register it exactly like a user-declared class.
const builtinErrorClassSource = `
[오류]를 설계하자:
    '메시지'를 [문자열]인 ""로 정하자

    처음 만들어질 때 ([문자열]인 '초기메시지') 다음과 같이 하자:
        '나'의 '메시지'를 '초기메시지'로 정하자

    [문자열]을 돌려주는 <__toString__>을 만들자 ():
        '나'의 '메시지'를 돌려주자
`

// kanadeBuiltinErrorClassSource is builtinErrorClassSource's 카나데
// translation, kept a straight word-for-word mirror (same field/param
// names in Japanese) so anyone comparing the two doesn't have to guess
// which parts are semantically load-bearing.
const kanadeBuiltinErrorClassSource = `
【エラー】を設計しよう:
    『メッセージ』を【文字列】の「」にしよう

    最初に作られる時 (【文字列】の『初期メッセージ』) 次のようにしよう:
        『私』の『メッセージ』を『初期メッセージ』にしよう

    【文字列】返す〈__toString__〉を作ろう ():
        『私』の『メッセージ』を返そう
`

func init() {
	hajaLang.builtinErrorClass = parseBuiltinErrorClass(func() (*ast.Program, []string) {
		l := haja_lexer.New(builtinErrorClassSource)
		p := haja_parser.New(l)
		return p.ParseProgram(), p.Errors()
	})
	kanadeLang.builtinErrorClass = parseBuiltinErrorClass(func() (*ast.Program, []string) {
		l := kanade_lexer.New(kanadeBuiltinErrorClassSource)
		p := kanade_parser.New(l)
		return p.ParseProgram(), p.Errors()
	})
}

func parseBuiltinErrorClass(parse func() (*ast.Program, []string)) *ast.ClassDeclaration {
	prog, errs := parse()
	if len(errs) > 0 {
		panic(fmt.Sprintf("bytecode: builtin error class source failed to parse: %v", errs))
	}
	for _, stmt := range prog.Statements {
		if cls, ok := stmt.(*ast.ClassDeclaration); ok {
			return cls
		}
	}
	panic("bytecode: builtin error class source failed to parse into a ClassDeclaration")
}

func (c *Compiler) Errors() []string { return c.errors }

func (c *Compiler) errorf(format string, args ...interface{}) {
	c.errors = append(c.errors, fmt.Sprintf(format, args...))
}

// Compile compiles prog into a Program (a Main chunk plus every function/
// class/interface declared in it). Functions, classes, and interfaces are
// all registered in a first pass (so forward references — a function
// calling one declared later, a class extending one declared later — just
// work), then Main is compiled in source order. A ClassDeclaration's own
// static-field initializers are the one exception: they're compiled inline
// into Main at the class's source position in the *second* pass, not
// registered in the first — see compileClassStaticInit.
func (c *Compiler) Compile(prog *ast.Program) *Program {
	main := newChunk()

	for _, stmt := range prog.Statements {
		switch s := stmt.(type) {
		case *ast.FunctionDeclaration:
			c.compileFunction(s)
		case *ast.InterfaceDeclaration:
			c.registerInterface(s)
		case *ast.ClassDeclaration:
			c.registerClass(s)
		case *ast.ImportStatement:
			// "파일"에서 ... 가져오자 is resolved entirely here, at compile
			// time, by merging the imported file's declaration into this
			// Program — like a function/class, it becomes available
			// throughout the program, not just after its own source
			// position (a deliberate simplification vs. the tree-walker's
			// strictly in-order import; see compileLocalImport). [모듈]에서
			// ... 가져오자 can't be resolved here since native modules are
			// runtime data supplied by the embedding Go program, not
			// something this compiler can see — it's compiled inline into
			// Main below instead, at its actual source position.
			if s.IsBuiltin {
				c.resolveBuiltinImport(s)
			} else {
				c.compileLocalImport(s)
			}
		}
	}
	for _, stmt := range prog.Statements {
		switch st := stmt.(type) {
		case *ast.FunctionDeclaration, *ast.InterfaceDeclaration:
			continue
		case *ast.ImportStatement:
			if !st.IsBuiltin {
				continue // already resolved above
			}
		}
		c.compileStatement(main, stmt)
	}
	main.emit(HALT, nil)

	c.program.Main = main
	return c.program
}

// registerInterface records name interfaceInfo requires, read back at
// vm.Run() to pre-flight-check implementing classes.
func (c *Compiler) registerInterface(decl *ast.InterfaceDeclaration) {
	info := &InterfaceInfo{Name: decl.Name.Name}
	for _, stmt := range decl.Body {
		if m, ok := stmt.(*ast.InterfaceMethod); ok {
			info.Methods = append(info.Methods, m.Name.Value)
		}
	}
	c.program.Interfaces[decl.Name.Name] = info
}

// compileLocalImport resolves "파일"에서 <이름>을 가져오자 at compile time:
// the imported file is lexed/parsed/compiled in its own throwaway
// sub-Compiler (so its declarations don't leak into this Program wholesale —
// mirrors vm.bindImportTarget only ever binding the one requested name, not
// everything the sub-interpreter ran), and only the specifically-requested
// function/class/interface is copied into c.program under its own name.
//
// One deliberate difference from the tree-walker: vm.importLocalFile
// actually runs the whole imported file (subInterpreter.Run()) before
// extracting the target, so any top-level side-effecting statements in it
// (프린트, static class field initializers, ...) execute too. Replicating
// that would mean either executing the imported file's Main chunk at
// compile time (impossible — there's no VM yet) or deferring resolution to
// a runtime opcode that runs a whole nested Program, which is a lot of
// machinery for what both engines already treat as unusual to lean on. This
// only merges declarations; a target's own top-level statements outside
// function/class/interface bodies don't run.
func (c *Compiler) compileLocalImport(s *ast.ImportStatement) {
	if c.importStack == nil {
		c.importStack = map[string]bool{}
	}
	if c.importStack[s.Module] {
		c.errorf("ImportError: circular import involving '%s'.", s.Module)
		return
	}

	content, err := os.ReadFile(s.Module)
	if err != nil {
		c.errorf("ImportError: File '%s' not found.", s.Module)
		return
	}

	// The imported file's own extension decides its lexer/parser (and the
	// sub-Compiler's lang), independent of c's own language — a 하자 file
	// can import a 카나데 file and vice versa, since both produce the same
	// *ast.Program.
	var prog *ast.Program
	var parseErrors []string
	var sub *Compiler
	if strings.HasSuffix(s.Module, ".knd") {
		l := kanade_lexer.New(string(content))
		p := kanade_parser.New(l)
		prog = p.ParseProgram()
		parseErrors = p.Errors()
		sub = NewKanadeCompiler()
	} else {
		l := haja_lexer.New(string(content))
		p := haja_parser.New(l)
		prog = p.ParseProgram()
		parseErrors = p.Errors()
		sub = NewCompiler()
	}
	if len(parseErrors) > 0 {
		c.errorf("ImportError: Syntax error in file '%s'.", s.Module)
		return
	}

	c.importStack[s.Module] = true
	defer delete(c.importStack, s.Module)

	sub.importStack = c.importStack // share so a transitive cycle is caught too
	for _, stmt := range prog.Statements {
		switch st := stmt.(type) {
		case *ast.FunctionDeclaration:
			sub.compileFunction(st)
		case *ast.InterfaceDeclaration:
			sub.registerInterface(st)
		case *ast.ClassDeclaration:
			sub.registerClass(st)
		case *ast.ImportStatement:
			if !st.IsBuiltin {
				sub.compileLocalImport(st)
			}
		}
	}
	c.errors = append(c.errors, sub.errors...)

	c.exportModule(s.Module, prog, sub)
	c.mergeImported(s.Module, s.All, s.Items, prog, sub)
}

// registerClass compiles decl's own fields/methods/constructor into a
// ClassInfo (never merged with its BaseClass — lookups walk the chain at
// call time). Static field *initializers* are handled separately, inline
// in Main at the class's source position (compileClassStaticInit), since
// they're runtime side effects that must run in program order, not
// metadata.
func (c *Compiler) registerClass(decl *ast.ClassDeclaration) {
	info := &ClassInfo{
		Name:          decl.Name.Name,
		IsAbstract:    decl.IsAbstract,
		Methods:       map[string]*Function{},
		StaticMethods: map[string]*Function{},
	}
	if decl.BaseClass != nil {
		info.BaseClass = decl.BaseClass.Name
	}
	for _, iface := range decl.Interfaces {
		info.Interfaces = append(info.Interfaces, iface.Name)
	}

	for _, stmt := range decl.Body {
		switch s := stmt.(type) {
		case *ast.VariableDeclaration:
			field := FieldInfo{Name: s.Name.Value, Access: accessOrPublic(s.AccessModifier)}
			if s.TypeRef != nil {
				field.Type = s.TypeRef.Name
			}
			if s.Value != nil {
				dc := newChunk()
				c.compileExpression(dc, s.Value)
				dc.emit(RETURN, nil)
				field.Default = dc
			}
			if s.Getter != nil {
				gc := newChunk()
				c.compileStatements(gc, s.Getter)
				gc.emit(PUSH_NULL, nil) // 돌려주자 없이 끝나면 비어있음
				gc.emit(RETURN, nil)
				field.Getter = gc
			}
			if s.Setter != nil {
				sc := newChunk()
				c.compileStatements(sc, s.Setter.Body)
				sc.emit(PUSH_NULL, nil)
				sc.emit(RETURN, nil)
				paramName := ""
				if s.Setter.Param != nil {
					paramName = s.Setter.Param.Value
				}
				field.Setter = &SetterInfo{ParamName: paramName, Body: sc}
			}
			if s.IsStatic {
				info.StaticFields = append(info.StaticFields, field)
			} else {
				info.Fields = append(info.Fields, field)
			}

		case *ast.FunctionDeclaration:
			fn := c.compileMethod(s.Name.Value, s.Params, s.Body, accessOrPublic(s.AccessModifier))
			fn.ReturnType = returnTypeName(s.ReturnType)
			if s.IsStatic {
				info.StaticMethods[s.Name.Value] = fn
			} else {
				info.Methods[s.Name.Value] = fn
			}

		case *ast.ConstructorDeclaration:
			body := newChunk()
			c.compileStatements(body, s.Body)
			info.Constructor = &Function{
				Name:      "__init__",
				Params:    c.compileParams(s.Params),
				BodyChunk: body,
			}
		}
	}

	c.program.Classes[decl.Name.Name] = info
}

func accessOrPublic(access string) string {
	if access == "" {
		return "public"
	}
	return access
}

func (c *Compiler) compileParams(params []*ast.Parameter) []Param {
	out := make([]Param, len(params))
	for i, p := range params {
		param := Param{Name: p.Name.Value}
		if p.TypeAnnotation != nil {
			param.Type = p.TypeAnnotation.Name
		}
		if p.Default != nil {
			dc := newChunk()
			c.compileExpression(dc, p.Default)
			dc.emit(RETURN, nil)
			param.Default = dc
		}
		out[i] = param
	}
	return out
}

func (c *Compiler) compileMethod(name string, params []*ast.Parameter, body *ast.BlockStatement, access string) *Function {
	bc := newChunk()
	c.compileBlock(bc, body)
	bc.emit(PUSH_NULL, nil)
	bc.emit(RETURN, nil)
	return &Function{Name: name, Params: c.compileParams(params), BodyChunk: bc, Access: access}
}

// compileClassStaticInit compiles decl's static field initializers inline
// into chunk at decl's own source position, mirroring
// vm.Interpreter.Run()'s step 3 exactly: a ClassDeclaration among the
// top-level statements runs its '우리'의 X 정하자 / static VariableDeclaration
// initializers right there, in program order, not hoisted to the start.
func (c *Compiler) compileClassStaticInit(chunk *Chunk, decl *ast.ClassDeclaration) {
	clsNameIdx := chunk.addName(decl.Name.Name)
	for _, stmt := range decl.Body {
		switch s := stmt.(type) {
		case *ast.VariableDeclaration:
			if !s.IsStatic {
				continue
			}
			c.compileExpression(chunk, s.Value)
			chunk.emit(SET_STATIC_FIELD, &StaticFieldOperand{ClassNameIndex: clsNameIdx, FieldNameIndex: chunk.addName(s.Name.Value)})
		case *ast.Assignment:
			mem, ok := s.Target.(*ast.MemberExpression)
			if !ok {
				continue
			}
			id, ok := mem.Object.(*ast.Identifier)
			if !ok || id.Value != c.lang.pluralSelfWord {
				continue
			}
			propId, ok := mem.Property.(*ast.Identifier)
			if !ok {
				continue
			}
			c.compileExpression(chunk, s.Value)
			chunk.emit(SET_STATIC_FIELD, &StaticFieldOperand{ClassNameIndex: clsNameIdx, FieldNameIndex: chunk.addName(propId.Value)})
		}
	}
}

// returnTypeName is the annotation a function declares for what it returns.
func returnTypeName(t *ast.TypeReference) string {
	if t == nil {
		return ""
	}
	return t.Name
}

func (c *Compiler) compileFunction(fn *ast.FunctionDeclaration) {
	body := newChunk()
	params := make([]Param, len(fn.Params))
	for i, p := range fn.Params {
		param := Param{Name: p.Name.Value}
		if p.TypeAnnotation != nil {
			param.Type = p.TypeAnnotation.Name
		}
		if p.Default != nil {
			dc := newChunk()
			c.compileExpression(dc, p.Default)
			dc.emit(RETURN, nil)
			param.Default = dc
		}
		params[i] = param
	}
	c.compileBlock(body, fn.Body)
	// 명시적 돌려주자 없이 끝까지 실행되면 비어있음을 반환한다 (Runtime 스펙 2.3).
	body.emit(PUSH_NULL, nil)
	body.emit(RETURN, nil)

	c.program.Functions[fn.Name.Value] = &Function{Name: fn.Name.Value, Params: params, BodyChunk: body, ReturnType: returnTypeName(fn.ReturnType)}
}

func (c *Compiler) compileBlock(chunk *Chunk, block *ast.BlockStatement) {
	if block == nil {
		return
	}
	c.compileStatements(chunk, block.Statements)
}

func (c *Compiler) compileStatements(chunk *Chunk, stmts []ast.Statement) {
	for _, stmt := range stmts {
		c.compileStatement(chunk, stmt)
	}
}

func (c *Compiler) compileStatement(chunk *Chunk, stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.VariableDeclaration:
		if s.Value != nil {
			c.compileExpression(chunk, s.Value)
		} else {
			chunk.emit(PUSH_NULL, nil) // 준비하자: 선언만
		}
		if s.IsStatic {
			// `'우리'의 'X'를 ... 정하자` written inside a method body parses as
			// a static VariableDeclaration; vm/exec_stmt.go stores it into the
			// running class's static field (whatever class 우리 is at runtime),
			// not a local. A class-ref is exactly what SET_MEMBER's static
			// branch already writes through.
			chunk.emit(LOAD_STATIC_CLASS, nil)
			chunk.emit(SET_MEMBER, &MemberOperand{PropNameIndex: chunk.addName(s.Name.Value)})
		} else if s.TypeRef != nil {
			chunk.emit(SET_VAR_TYPED, &TypedSetOperand{NameIndex: chunk.addName(s.Name.Value), Type: s.TypeRef.Name, Const: s.IsConstant})
		} else if s.IsConstant {
			chunk.emit(SET_CONST, chunk.addName(s.Name.Value))
		} else {
			chunk.emit(SET_VAR, chunk.addName(s.Name.Value))
		}

	case *ast.Assignment:
		c.compileExpression(chunk, s.Value)
		c.compileAssignTarget(chunk, s.Target)

	case *ast.PrintStatement:
		c.compileExpression(chunk, s.Value)
		chunk.emit(TO_DISPLAY_STRING, nil)
		if s.NewLine {
			chunk.emit(PRINT, nil)
		} else {
			chunk.emit(PRINT_INLINE, nil)
		}

	case *ast.ExpressionStatement:
		c.compileExpression(chunk, s.Expression)
		chunk.emit(POP, nil) // 문장으로 쓰인 값은 버림

	case *ast.ReturnStatement:
		if s.Value != nil {
			c.compileExpression(chunk, s.Value)
		} else {
			chunk.emit(PUSH_NULL, nil)
		}
		c.emitTryExits(chunk)
		chunk.emit(RETURN, nil)

	case *ast.BreakStatement:
		c.emitTryExits(chunk)
		if n := len(c.loopScopes); n > 0 && c.loopScopes[n-1] != "" {
			chunk.emit(POP_SCOPE, chunk.addName(c.loopScopes[n-1]))
		}
		idx := chunk.emit(JUMP, nil)
		c.registerBreak(idx)

	case *ast.ThrowStatement:
		c.compileExpression(chunk, s.Value)
		chunk.emit(THROW, nil)

	case *ast.TryStatement:
		c.compileTry(chunk, s)

	case *ast.IfStatement:
		c.compileIf(chunk, s)

	case *ast.WhileLoop:
		c.compileWhile(chunk, s)

	case *ast.ForRangeStatement:
		c.compileForRange(chunk, s)

	case *ast.ForEachLoop:
		c.compileForEach(chunk, s)

	case *ast.SwitchStatement:
		c.compileSwitch(chunk, s)

	case *ast.ClassDeclaration:
		c.compileClassStaticInit(chunk, s)

	case *ast.ImportStatement:
		// 로컬 파일 임포트("파일"에서 ...)는 Compile()의 첫 패스에서 이미
		// 처리됐으니(compileLocalImport), 여기 도달하는 건 항상 [모듈]에서
		// ... 가져오자(IsBuiltin) 뿐이다.
		c.emitBuiltinImport(chunk, s)

	case *ast.ListPushStatement:
		c.compileExpression(chunk, s.Target) // 현재 목록 값
		c.compileExpression(chunk, s.Value)
		chunk.emit(LIST_PUSH, s.Position)
		c.compileAssignTarget(chunk, s.Target) // 새 목록을 원래 자리(변수든 객체 필드든)에 다시 씀

	case *ast.ListPopStatement:
		c.compileExpression(chunk, s.Target)   // 현재 목록 값
		chunk.emit(LIST_POP, s.Position)       // -> ..., 나머지, 꺼낸값
		chunk.emit(POP, nil)                   // 문장형은 꺼낸 값을 버림 (AST 스펙: 반환값 없음)
		c.compileAssignTarget(chunk, s.Target) // 나머지를 다시 씀

	case *ast.InputStatement:
		// vm/exec_stmt.go와 같은 규칙: 한 줄을 읽어 [타입]으로 변환(타입이 없으면
		// 문자열)한 값을 바인딩한다. 입력 소스가 없으면 빈 줄을 읽는다.
		typeName := ""
		if s.TypeRef != nil {
			typeName = s.TypeRef.Name
		}
		chunk.emit(INPUT, typeName)
		if s.Target != nil {
			chunk.emit(SET_VAR, chunk.addName(s.Target.Value))
		} else {
			chunk.emit(POP, nil)
		}

	default:
		c.errorf("compileStatement: unsupported statement %T", stmt)
	}
}

func (c *Compiler) compileAssignTarget(chunk *Chunk, target ast.Expression) {
	if mem, ok := target.(*ast.MemberExpression); ok {
		c.compileMemberAssignTarget(chunk, mem)
		return
	}
	nameIdx, ok := c.identifierNameIndex(chunk, target)
	if !ok {
		c.errorf("Assignment: unsupported target %T", target)
		return
	}
	chunk.emit(SET_VAR, nameIdx)
}

// compileMemberExpression compiles a MemberExpression read (field/method
// access, list/string index, static member) into a single GET_MEMBER,
// deferring which of those it actually is to runtime — exactly like
// vm/eval_expr.go's MemberExpression case, which switches on e.Object's
// evaluated *type*, not its static shape. See MemberOperand's doc comment.
func (c *Compiler) compileMemberExpression(chunk *Chunk, e *ast.MemberExpression) {
	c.compileExpression(chunk, e.Object)
	var propName string
	isFunc := false
	if fr, ok := e.Property.(*ast.FunctionReference); ok {
		propName = fr.Name
		isFunc = true
	} else if id, ok := e.Property.(*ast.Identifier); ok {
		propName = id.Value
	}
	var indexChunk *Chunk
	if !isFunc {
		ic := newChunk()
		c.compileExpression(ic, e.Property)
		ic.emit(RETURN, nil)
		indexChunk = ic
	}
	chunk.emit(GET_MEMBER, &MemberOperand{PropNameIndex: chunk.addName(propName), IsFunc: isFunc, IndexChunk: indexChunk})
}

// compileMemberAssignTarget compiles a MemberExpression assignment target
// (object field, list index, static field — decided at runtime by
// SET_MEMBER exactly like GET_MEMBER, since e.g. a bare Identifier property
// is ambiguous between "field name" and "index variable" until you know
// what Object evaluated to). Value must already be on the stack, per
// compileStatement's Assignment case.
func (c *Compiler) compileMemberAssignTarget(chunk *Chunk, mem *ast.MemberExpression) {
	c.compileExpression(chunk, mem.Object)
	var propName string
	if id, ok := mem.Property.(*ast.Identifier); ok {
		propName = id.Value
	}
	ic := newChunk()
	c.compileExpression(ic, mem.Property)
	ic.emit(RETURN, nil)
	chunk.emit(SET_MEMBER, &MemberOperand{PropNameIndex: chunk.addName(propName), IndexChunk: ic})
}

// identifierNameIndex validates that target is a plain variable reference
// and interns its name into chunk's name pool.
func (c *Compiler) identifierNameIndex(chunk *Chunk, target ast.Expression) (int, bool) {
	id, ok := target.(*ast.Identifier)
	if !ok {
		return 0, false
	}
	return chunk.addName(id.Value), true
}

func (c *Compiler) compileIf(chunk *Chunk, s *ast.IfStatement) {
	c.compileExpression(chunk, s.Condition)
	jumpToElse := chunk.emit(JUMP_IF_FALSE, nil)
	c.compileBlock(chunk, s.Consequent)
	jumpToEnd := chunk.emit(JUMP, nil)
	chunk.patchOperand(jumpToElse, chunk.nextIndex())
	if s.Alternate != nil {
		c.compileBlock(chunk, s.Alternate)
	}
	chunk.patchOperand(jumpToEnd, chunk.nextIndex())
}

func (c *Compiler) compileWhile(chunk *Chunk, s *ast.WhileLoop) {
	c.pushLoop(c.bodyScope(s.Body))
	loopStart := chunk.nextIndex()
	c.compileExpression(chunk, s.Condition)
	jumpToEnd := chunk.emit(JUMP_IF_FALSE, nil)
	c.compileScopedBody(chunk, s.Body)
	chunk.emit(JUMP, loopStart)
	chunk.patchOperand(jumpToEnd, chunk.nextIndex())
	c.popLoopAndPatchBreaks(chunk)
}

// compileForRange compiles `시작부터 끝까지 반복하자`. Runtime 스펙 5의
// ForRangeStatement 규칙: 끝값 포함(inclusive), 시작이 끝보다 크면 역순.
// start/end는 임의의 표현식이라 방향(step)은 컴파일 시점이 아니라 런타임에
// 한 번만 결정해야 한다. 매 반복 분기 대신 "(end-loopvar)*step >= 0" 하나의
// 산술식으로 계속 여부를 판단해 정방향/역방향을 같은 코드로 처리한다.
func (c *Compiler) compileForRange(chunk *Chunk, s *ast.ForRangeStatement) {
	loopVar := s.LoopVar
	if loopVar == "" {
		loopVar = "인덱스"
	}
	tmp := c.newTempName()
	endName := tmp + "_end"
	stepName := tmp + "_step"

	// The loop variable and the hidden end/step values live in a scope of their
	// own, so they are gone after the loop, as in the tree-walker.
	outerScope := tmp + "_outer"
	chunk.emit(PUSH_SCOPE, chunk.addName(outerScope))

	c.compileExpression(chunk, s.Start)
	chunk.emit(SET_VAR, chunk.addName(loopVar))

	c.compileExpression(chunk, s.End)
	chunk.emit(SET_VAR, chunk.addName(endName))

	// step = (end < start) ? -1 : 1
	chunk.emit(LOAD_VAR, chunk.addName(endName))
	chunk.emit(LOAD_VAR, chunk.addName(loopVar))
	chunk.emit(LT, nil)
	jPositive := chunk.emit(JUMP_IF_FALSE, nil)
	chunk.emit(PUSH_CONST, chunk.addConstant(-1.0))
	jStepDone := chunk.emit(JUMP, nil)
	chunk.patchOperand(jPositive, chunk.nextIndex())
	chunk.emit(PUSH_CONST, chunk.addConstant(1.0))
	chunk.patchOperand(jStepDone, chunk.nextIndex())
	chunk.emit(SET_VAR, chunk.addName(stepName))

	c.pushLoop(c.bodyScope(s.Body))
	loopStart := chunk.nextIndex()
	chunk.emit(LOAD_VAR, chunk.addName(endName))
	chunk.emit(LOAD_VAR, chunk.addName(loopVar))
	chunk.emit(SUB, nil)
	chunk.emit(LOAD_VAR, chunk.addName(stepName))
	chunk.emit(MUL, nil)
	chunk.emit(PUSH_CONST, chunk.addConstant(0.0))
	chunk.emit(GTE, nil)
	jEnd := chunk.emit(JUMP_IF_FALSE, nil)

	c.compileScopedBody(chunk, s.Body)

	chunk.emit(LOAD_VAR, chunk.addName(loopVar))
	chunk.emit(LOAD_VAR, chunk.addName(stepName))
	chunk.emit(ADD, nil)
	chunk.emit(SET_VAR, chunk.addName(loopVar))
	chunk.emit(JUMP, loopStart)

	chunk.patchOperand(jEnd, chunk.nextIndex())
	c.popLoopAndPatchBreaks(chunk)
	chunk.emit(POP_SCOPE, chunk.addName(outerScope))
}

// compileForEach compiles `각각에 대해 반복하자` / MemberExpression-List form.
// exec_stmt.go's ForEachLoop case: if List parses as a MemberExpression, its
// Object is the list and its Property (if an Identifier) names the loop
// variable; otherwise List is the list itself and the loop variable is the
// engine's default item name. We replicate that exact compile-time check —
// itemName is therefore always known statically, never resolved at runtime.
// The loop itself walks a 1-based index (matching GET_INDEX) from 1 to
// GET_LENGTH so it can reuse existing opcodes instead of needing a new
// iterator opcode.
func (c *Compiler) compileForEach(chunk *Chunk, s *ast.ForEachLoop) {
	itemName := c.lang.defaultItemName
	listExpr := s.List
	if memExpr, ok := s.List.(*ast.MemberExpression); ok {
		if id, ok := memExpr.Property.(*ast.Identifier); ok {
			itemName = id.Value
		}
		listExpr = memExpr.Object
	}

	tmp := c.newTempName()
	listName := tmp + "_list"
	idxName := tmp + "_idx"

	outerScope := tmp + "_outer" // the item and the hidden list/index: gone after the loop
	chunk.emit(PUSH_SCOPE, chunk.addName(outerScope))

	c.compileExpression(chunk, listExpr)
	chunk.emit(TO_ITERABLE, nil)
	chunk.emit(SET_VAR, chunk.addName(listName))
	chunk.emit(PUSH_CONST, chunk.addConstant(1.0))
	chunk.emit(SET_VAR, chunk.addName(idxName))

	c.pushLoop(c.bodyScope(s.Body))
	loopStart := chunk.nextIndex()
	chunk.emit(LOAD_VAR, chunk.addName(idxName))
	chunk.emit(LOAD_VAR, chunk.addName(listName))
	chunk.emit(GET_LENGTH, nil)
	chunk.emit(LTE, nil)
	jEnd := chunk.emit(JUMP_IF_FALSE, nil)

	chunk.emit(LOAD_VAR, chunk.addName(listName))
	chunk.emit(LOAD_VAR, chunk.addName(idxName))
	chunk.emit(GET_INDEX, nil)
	chunk.emit(SET_VAR, chunk.addName(itemName))

	c.compileScopedBody(chunk, s.Body)

	chunk.emit(LOAD_VAR, chunk.addName(idxName))
	chunk.emit(PUSH_CONST, chunk.addConstant(1.0))
	chunk.emit(ADD, nil)
	chunk.emit(SET_VAR, chunk.addName(idxName))
	chunk.emit(JUMP, loopStart)

	chunk.patchOperand(jEnd, chunk.nextIndex())
	c.popLoopAndPatchBreaks(chunk)
	chunk.emit(POP_SCOPE, chunk.addName(outerScope))
}

// compileSwitch compiles `따라 나누자` as a chain of test-then-body sections,
// mirroring exec_stmt.go's SwitchStatement exactly: cases (including
// default) are checked strictly in source order and the first one that
// matches — a test equality or an unconditional default — wins, even if a
// default appears before other cases. 다음으로 이어가자 (fallthrough) jumps
// straight into the next case's body, skipping its tests entirely, exactly
// like the tree-walker's fallthroughNext flag.
//
// Two backpatch lists carry jump targets across loop iterations, since a
// case's fail/fallthrough targets are only known once the next case is laid
// out: pendingTestFail (failed tests fall through to the next case's test
// section) and pendingFallthrough (폴스루 jumps straight to the next case's
// body).
func (c *Compiler) compileSwitch(chunk *Chunk, s *ast.SwitchStatement) {
	discName := c.newTempName()
	c.compileExpression(chunk, s.Discriminant)
	chunk.emit(SET_VAR, chunk.addName(discName))

	var endJumps []int
	var pendingTestFail []int
	var pendingFallthrough []int

	for _, cs := range s.Cases {
		testStart := chunk.nextIndex()
		for _, j := range pendingTestFail {
			chunk.patchOperand(j, testStart)
		}
		pendingTestFail = nil

		var bodyJumps []int
		if !cs.IsDefault {
			for _, t := range cs.Tests {
				chunk.emit(LOAD_VAR, chunk.addName(discName))
				c.compileExpression(chunk, t)
				chunk.emit(EQ, nil)
				bodyJumps = append(bodyJumps, chunk.emit(JUMP_IF_TRUE, nil))
			}
			pendingTestFail = append(pendingTestFail, chunk.emit(JUMP, nil))
		}

		bodyStart := chunk.nextIndex()
		for _, j := range bodyJumps {
			chunk.patchOperand(j, bodyStart)
		}
		for _, j := range pendingFallthrough {
			chunk.patchOperand(j, bodyStart)
		}
		pendingFallthrough = nil

		fellThrough := false
		for _, stmt := range cs.Consequent.Statements {
			if _, ok := stmt.(*ast.FallthroughStatement); ok {
				fellThrough = true
				break
			}
			c.compileStatement(chunk, stmt)
		}
		if fellThrough {
			pendingFallthrough = append(pendingFallthrough, chunk.emit(JUMP, nil))
		} else {
			endJumps = append(endJumps, chunk.emit(JUMP, nil))
		}
	}

	if len(pendingTestFail) > 0 {
		// 맞는 경우도 나머지는도 없이 끝까지 왔다: 오류
		for _, j := range pendingTestFail {
			chunk.patchOperand(j, chunk.nextIndex())
		}
		pendingTestFail = nil
		chunk.emit(SWITCH_NO_MATCH, chunk.addName(discName))
	}

	end := chunk.nextIndex()
	for _, j := range pendingTestFail {
		chunk.patchOperand(j, end)
	}
	for _, j := range pendingFallthrough {
		chunk.patchOperand(j, end) // 마지막 case의 폴스루는 tree-walker와 동일하게 no-op
	}
	for _, j := range endJumps {
		chunk.patchOperand(j, end)
	}
}

// compileTry compiles 일단 해보자/오류가 발생했다면/마무리는 항상, mirroring
// exec_stmt.go's TryStatement case: catch clauses are checked in source
// order, an untyped handler (h.Type == nil) always matches, a typed one
// only if the raised value is a class instance that is-or-extends it (an
// engine-raised error never matches a typed handler, only an untyped one).
// The protected block and catch bodies run in chunk's own scope (no new
// frame), same as the tree-walker running them via the same env.
//
// Runtime spec 4.2 requires 마무리는 항상 to run no matter how the try
// statement is left — normal completion, no handler matching, or even a
// ReturnStatement inside it — so RUN_FINALLY is compiled at every one of
// those exit points rather than relying on a single fallthrough path (see
// RUN_FINALLY's doc comment and emitTryExits).
func (c *Compiler) compileTry(chunk *Chunk, s *ast.TryStatement) {
	var financeChunk *Chunk
	if s.Finalizer != nil {
		financeChunk = newChunk()
		c.compileStatements(financeChunk, s.Finalizer.Statements)
	}

	scope := &tryScope{finallyChunk: financeChunk, inProtectedBlock: true}
	c.tryScopes = append(c.tryScopes, scope)

	tryPushIdx := chunk.emit(TRY_PUSH, nil)
	c.compileStatements(chunk, s.Block.Statements)
	scope.inProtectedBlock = false // dispatchError already pops this level before jumping into a catch handler
	chunk.emit(TRY_POP, nil)
	if financeChunk != nil {
		chunk.emit(RUN_FINALLY, financeChunk)
	}
	jToEnd := chunk.emit(JUMP, nil)

	var catches []CatchInfo
	var handlerEndJumps []int
	for _, h := range s.Handlers {
		handlerStart := chunk.nextIndex()
		handlerScope := c.newTempName() + "_catch" // the error variable and what the handler declares end with it
		chunk.emit(PUSH_SCOPE, chunk.addName(handlerScope))
		chunk.emit(SET_VAR, chunk.addName(h.Param.Value))
		c.compileStatements(chunk, h.Body.Statements)
		chunk.emit(POP_SCOPE, chunk.addName(handlerScope))
		if financeChunk != nil {
			chunk.emit(RUN_FINALLY, financeChunk)
		}
		handlerEndJumps = append(handlerEndJumps, chunk.emit(JUMP, nil))
		typeName := ""
		if h.Type != nil {
			typeName = h.Type.Name
		}
		catches = append(catches, CatchInfo{TypeName: typeName, HandlerPc: handlerStart})
	}

	c.tryScopes = c.tryScopes[:len(c.tryScopes)-1]

	end := chunk.nextIndex()
	chunk.patchOperand(tryPushIdx, &TryOperand{Catches: catches, FinallyChunk: financeChunk})
	chunk.patchOperand(jToEnd, end)
	for _, j := range handlerEndJumps {
		chunk.patchOperand(j, end)
	}
}

// emitTryExits emits whatever TRY_POP/RUN_FINALLY instructions are needed
// to leave every currently-enclosing try statement cleanly, innermost
// first — called before a ReturnStatement/BreakStatement, both of which
// bypass catch matching entirely (Runtime spec 4.2 names ReturnStatement
// explicitly) but must still (a) keep the try stack accurate for whatever
// runs after a break, since a stale TRY_PUSH entry would let a later,
// unrelated error be wrongly caught by a handler that's no longer active,
// and (b) run every enclosing 마무리는 항상.
func (c *Compiler) emitTryExits(chunk *Chunk) {
	for i := len(c.tryScopes) - 1; i >= 0; i-- {
		scope := c.tryScopes[i]
		if scope.inProtectedBlock {
			chunk.emit(TRY_POP, nil)
		}
		if scope.finallyChunk != nil {
			chunk.emit(RUN_FINALLY, scope.finallyChunk)
		}
	}
}

func (c *Compiler) newTempName() string {
	c.tempCounter++
	return fmt.Sprintf("__tmp%d", c.tempCounter)
}

func (c *Compiler) pushLoop(scope string) {
	c.breakJumps = append(c.breakJumps, nil)
	c.loopScopes = append(c.loopScopes, scope)
}

// bodyScope names the per-iteration scope marker of a loop body, or "" when
// nothing in the body can declare a variable (then no scope opcodes are emitted).
func (c *Compiler) bodyScope(body *ast.BlockStatement) string {
	if !mayDeclare(reflect.ValueOf(body), map[uintptr]bool{}) {
		return ""
	}
	return c.newTempName() + "_scope"
}

// compileScopedBody compiles a loop body, inside the scope pushLoop was given.
func (c *Compiler) compileScopedBody(chunk *Chunk, body *ast.BlockStatement) {
	scope := c.loopScopes[len(c.loopScopes)-1]
	if scope == "" {
		c.compileBlock(chunk, body)
		return
	}
	chunk.emit(PUSH_SCOPE, chunk.addName(scope))
	c.compileBlock(chunk, body)
	chunk.emit(POP_SCOPE, chunk.addName(scope))
}

// mayDeclare reports whether the tree under v contains a node that can declare a
// variable in the surrounding scope. It looks at every field, so a new node type
// with children is covered without listing it here; being too eager only costs a
// scope that turns out to be empty.
func mayDeclare(v reflect.Value, seen map[uintptr]bool) bool {
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return false
		}
		return mayDeclare(v.Elem(), seen)
	case reflect.Ptr:
		if v.IsNil() || seen[v.Pointer()] {
			return false
		}
		seen[v.Pointer()] = true
		switch v.Interface().(type) {
		case *ast.VariableDeclaration, *ast.InputStatement, *ast.ImportStatement,
			*ast.ForEachLoop, *ast.ForRangeStatement, *ast.TryStatement,
			*ast.FunctionDeclaration, *ast.ClassDeclaration, *ast.InterfaceDeclaration:
			return true
		}
		return mayDeclare(v.Elem(), seen)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() && mayDeclare(v.Field(i), seen) {
				return true
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if mayDeclare(v.Index(i), seen) {
				return true
			}
		}
	}
	return false
}

func (c *Compiler) registerBreak(instrIndex int) {
	if len(c.breakJumps) == 0 {
		c.errorf("반복을 끝내자: 반복문 밖에서는 쓸 수 없어요 (IllegalBreakError)")
		return
	}
	top := len(c.breakJumps) - 1
	c.breakJumps[top] = append(c.breakJumps[top], instrIndex)
}

func (c *Compiler) popLoopAndPatchBreaks(chunk *Chunk) {
	top := len(c.breakJumps) - 1
	pending := c.breakJumps[top]
	c.breakJumps = c.breakJumps[:top]
	c.loopScopes = c.loopScopes[:top]
	target := chunk.nextIndex()
	for _, idx := range pending {
		chunk.patchOperand(idx, target)
	}
}

func (c *Compiler) compileExpression(chunk *Chunk, expr ast.Expression) {
	switch e := expr.(type) {
	case *ast.NumberLiteral:
		chunk.emit(PUSH_CONST, chunk.addConstant(e.Value))

	case *ast.StringLiteral:
		chunk.emit(PUSH_CONST, chunk.addConstant(unescapeHajaString(e.Value)))

	case *ast.BooleanLiteral:
		chunk.emit(PUSH_BOOL, e.Value)

	case *ast.NullLiteral:
		chunk.emit(PUSH_NULL, nil)

	case *ast.TemplateLiteral:
		c.compileTemplateLiteral(chunk, e)

	case *ast.Identifier:
		// '나'/'우리' are ordinary quoted identifiers lexically (VAR tokens,
		// not the KW_SELF/StaticReference path — see compileMemberExpression
		// and eval_expr.go's own Identifier case, which special-cases these
		// same two literal values instead of relying on SelfReference).
		//
		// Unlike the bare SelfReference/StaticReference AST nodes, the quoted
		// form is only *self* when a 나/우리 is actually bound: outside a
		// method the tree-walker falls through to an ordinary variable lookup
		// (하자's own tutorials name a plain variable '나'). So these carry the
		// name as a fallback operand.
		switch {
		case e.Value == c.lang.selfWord:
			chunk.emit(LOAD_SELF, chunk.addName(e.Value))
		case e.Value == c.lang.pluralSelfWord:
			chunk.emit(LOAD_STATIC_CLASS, chunk.addName(e.Value))
		default:
			chunk.emit(LOAD_VAR, chunk.addName(e.Value))
		}

	case *ast.SelfReference:
		chunk.emit(LOAD_SELF, nil)

	case *ast.SuperReference:
		chunk.emit(LOAD_SUPER, nil)

	case *ast.StaticReference:
		chunk.emit(LOAD_STATIC_CLASS, nil)

	case *ast.TypeReference:
		chunk.emit(PUSH_CLASS_REF, chunk.addName(e.Name))

	case *ast.NewExpression:
		for _, arg := range e.Arguments {
			c.compileExpression(chunk, arg)
		}
		chunk.emit(NEW_OBJECT, &NewObjectOperand{ClassNameIndex: chunk.addName(e.Class.Name), Argc: len(e.Arguments)})

	case *ast.BinaryExpression:
		if e.Operator == "instanceof" {
			c.compileExpression(chunk, e.Left)
			c.compileExpression(chunk, e.Right)
			chunk.emit(INSTANCE_OF, nil)
			return
		}
		c.compileExpression(chunk, e.Left)
		c.compileExpression(chunk, e.Right)
		switch e.Operator {
		case "+":
			chunk.emit(ADD, nil)
		case "-":
			chunk.emit(SUB, nil)
		case "*":
			chunk.emit(MUL, nil)
		case "/":
			chunk.emit(DIV, nil)
		case "%":
			chunk.emit(MOD, nil)
		case "==":
			chunk.emit(EQ, nil)
		case "!=":
			chunk.emit(NEQ, nil)
		case "<":
			chunk.emit(LT, nil)
		case "<=":
			chunk.emit(LTE, nil)
		case ">":
			chunk.emit(GT, nil)
		case ">=":
			chunk.emit(GTE, nil)
		default:
			c.errorf("BinaryExpression: 연산자 %q는 지원 범위 밖입니다", e.Operator)
		}

	case *ast.LogicalExpression:
		c.compileExpression(chunk, e.Left)
		if e.Operator == "그리고" {
			jShort := chunk.emit(JUMP_IF_FALSE, nil)
			c.compileExpression(chunk, e.Right)
			chunk.emit(CHECK_BOOL, nil)
			jEnd := chunk.emit(JUMP, nil)
			chunk.patchOperand(jShort, chunk.nextIndex())
			chunk.emit(PUSH_BOOL, false)
			chunk.patchOperand(jEnd, chunk.nextIndex())
		} else {
			jShort := chunk.emit(JUMP_IF_TRUE, nil)
			c.compileExpression(chunk, e.Right)
			chunk.emit(CHECK_BOOL, nil)
			jEnd := chunk.emit(JUMP, nil)
			chunk.patchOperand(jShort, chunk.nextIndex())
			chunk.emit(PUSH_BOOL, true)
			chunk.patchOperand(jEnd, chunk.nextIndex())
		}

	case *ast.ListLiteral:
		for _, el := range e.Elements {
			c.compileExpression(chunk, el)
		}
		chunk.emit(NEW_LIST, len(e.Elements))

	case *ast.DictLiteral:
		for _, prop := range e.Properties {
			c.compileExpression(chunk, prop.Key)
			c.compileExpression(chunk, prop.Value)
		}
		chunk.emit(NEW_DICT, len(e.Properties))

	case *ast.ListPopExpression:
		// 꺼낸 값 자체가 이 식의 결과여야 하는데, LIST_POP은 나머지 목록과
		// 꺼낸 값을 둘 다 스택에 남긴다. 나머지를 compileAssignTarget으로
		// 되쓰는 동안(대상이 MemberExpression이면 추가로 명령을 더 내보냄)
		// 꺼낸 값이 스택 맨 위 자리를 지키게 하기 위해 임시 변수에 잠깐
		// 옮겨뒀다가 마지막에 다시 불러온다.
		tmp := c.newTempName()
		c.compileExpression(chunk, e.Target) // 현재 목록 값
		chunk.emit(LIST_POP, e.Position)     // -> ..., 나머지, 꺼낸값
		chunk.emit(SET_VAR, chunk.addName(tmp))
		c.compileAssignTarget(chunk, e.Target) // 나머지를 다시 씀
		chunk.emit(LOAD_VAR, chunk.addName(tmp))

	case *ast.MemberExpression:
		c.compileMemberExpression(chunk, e)

	case *ast.CallExpression:
		callCallee := e.Callee
		// "TYPE의 〈함수〉()": only a real registered class means "static
		// method call" — a built-in type name used as decorative packaging
		// (kanade-docs' "【文字列】の〈文字列に〉(123)") means "ignore the
		// TYPE, call the function plainly". c.program.Classes is already
		// fully populated by Compile()'s first pass by the time any
		// CallExpression gets compiled, so this can be a compile-time
		// check — see the matching runtime check in vm/eval_expr.go's
		// CallExpression case for why it can't be decided at parse time.
		if mem, ok := callCallee.(*ast.MemberExpression); ok {
			if typeRef, ok := mem.Object.(*ast.TypeReference); ok {
				if _, isClass := c.program.Classes[typeRef.Name]; !isClass {
					if fr, ok := mem.Property.(*ast.FunctionReference); ok {
						callCallee = fr
					}
				}
			}
		}
		switch callee := callCallee.(type) {
		case *ast.FunctionReference:
			for _, arg := range e.Arguments {
				c.compileExpression(chunk, arg)
			}
			chunk.emit(CALL, &CallOperand{NameIndex: chunk.addName(callee.Name), Argc: len(e.Arguments)})
		case *ast.MemberExpression:
			// X의 <비우기>(): the one *mutating* list pseudo-method (Runtime
			// spec — lists are otherwise manipulated via native syntax, not
			// method calls). Needs its own compiled shape, not the generic
			// GET_MEMBER/CALL_METHOD path below: by the time a CALL_METHOD
			// result comes back, the bytecode has no memory of "which
			// variable/field did this list value come from", so nothing
			// could write the cleared list back — same reason LIST_PUSH/POP
			// aren't compiled through a generic method call either. Detected
			// by property name at compile time, but LIST_CLEAR itself
			// re-checks the runtime value's type, so this can never silently
			// misfire on some other callee that merely evaluates to a list.
			if fr, ok := callee.Property.(*ast.FunctionReference); ok && fr.Name == c.lang.listClearMethod && len(e.Arguments) == 0 {
				c.compileExpression(chunk, callee.Object) // 현재 목록 값
				chunk.emit(LIST_CLEAR, nil)               // -> 빈 목록
				c.compileAssignTarget(chunk, callee.Object)
				chunk.emit(PUSH_NULL, nil) // 이 식 자체의 결과값 (비우기는 반환값 없음)
				return
			}
			// X의 <메서드>(...): compiling the MemberExpression itself (its
			// Property is a FunctionReference) produces a bound method value
			// via GET_MEMBER — see compileMemberExpression's isFunc branch.
			c.compileExpression(chunk, callee)
			for _, arg := range e.Arguments {
				c.compileExpression(chunk, arg)
			}
			chunk.emit(CALL_METHOD, len(e.Arguments))
		default:
			c.errorf("CallExpression: unsupported callee %T (동적 참조 <'변수'>()는 지원 범위 밖입니다)", e.Callee)
		}

	case *ast.FunctionReference:
		// A function used as a value (passed to a native function or a callback
		// registration), not called: resolved by name when it is finally called.
		chunk.emit(PUSH_FUNC_REF, chunk.addName(e.Name))

	default:
		c.errorf("compileExpression: unsupported expression %T", expr)
	}
}

func (c *Compiler) compileTemplateLiteral(chunk *Chunk, e *ast.TemplateLiteral) {
	raw := unescapeHajaString(e.Value)
	first := true

	push := func(s string) {
		if s == "" {
			return
		}
		chunk.emit(PUSH_CONST, chunk.addConstant(s))
		if first {
			first = false
		} else {
			chunk.emit(ADD, nil)
		}
	}

	for len(raw) > 0 {
		open := strings.Index(raw, "{")
		if open == -1 {
			push(raw)
			break
		}
		if open > 0 {
			push(raw[:open])
		}
		raw = raw[open+1:]
		closeIdx := strings.Index(raw, "}")
		if closeIdx == -1 {
			push("{" + raw)
			break
		}
		innerCode := raw[:closeIdx]
		raw = raw[closeIdx+1:]

		innerExpr := c.lang.parseExpr(innerCode)
		if innerExpr == nil {
			continue
		}
		c.compileExpression(chunk, innerExpr)
		chunk.emit(TO_DISPLAY_STRING, nil)
		if first {
			first = false
		} else {
			chunk.emit(ADD, nil)
		}
	}

	if first {
		chunk.emit(PUSH_CONST, chunk.addConstant(""))
	}
}

// unescapeHajaString applies the same escape processing the tree-walking
// interpreter does at evaluation time (eval_expr.go), but once, at compile
// time, since string/template literals are constant.
func unescapeHajaString(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\t`, "\t")
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}
