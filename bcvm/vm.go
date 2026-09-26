// Package bcvm is the stack-based bytecode VM that executes a
// *bytecode.Program compiled by the bytecode package. It is a separate
// execution path alongside the tree-walking vm package, not a replacement
// for it.
package bcvm

import (
	"fmt"
	"github.com/soumt-r/hana/console"
	"github.com/soumt-r/hana/conv"
	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/native"
	"github.com/soumt-r/hana/num"
	"github.com/soumt-r/hana/pkg"
	"github.com/soumt-r/hana/strcat"
	"github.com/soumt-r/hana/typecheck"
	"github.com/soumt-r/hana/value"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/soumt-r/hana/bytecode"
	"github.com/soumt-r/hana/symbol"
)

// NativeFunction is a Go-implemented callable an embedding program (e.g.
// bcstdlib) exposes to compiled Hari code under a fixed name. Mirrors
// vm.BuiltinFunction.
type NativeFunction struct {
	Name string
	Fn   func(vm *VM, args []interface{}) (interface{}, error)
}

// VM runs a compiled Program. Output mirrors vm.Interpreter.Output — every
// printed line, in order — so callers (tests, doctest-equivalents) can
// assert on it without scraping stdout.
type VM struct {
	program *bytecode.Program
	depth   int // nested exec() calls (see maxExecDepth)

	// runLock is held while the program runs; native libraries' callbacks wait
	// for it (see native_host.go).
	runLock native.ExecLock

	// ReadLine supplies the next line for INPUT (`입력받자`); the CLI wires it to
	// stdin. nil reads an empty line, so tests never block. InputTypes names the
	// types INPUT can read, per language (UseJapaneseWords switches them).
	ReadLine     func() string
	Types        typecheck.Names
	globals      *frame
	modules      map[string]*frame      // the top-level variables of each imported module, by module name
	moduleReady  map[string]bool        // the modules whose top-level code has run
	staticFields map[string]interface{} // "ClassName.field" -> value
	framePool    []*frame
	formatDepth  int                                       // how deep FormatValue is inside lists (a list may contain itself)
	members      map[string]map[symbol.Symbol]*memberEntry // see object.go: class member lookups, memoized
	Output       []string

	natives       map[string]*NativeFunction            // always callable by name, no import needed (e.g. 숫자로)
	nativeModules map[string]map[string]*NativeFunction // [모듈] -> name -> fn; only callable after IMPORT_NATIVE

	// LengthWord is the property name getMember compares against for a
	// list/string's length (하자: "길이"). Unlike LIST_CLEAR's method name
	// (known at compile time, so the compiler can bucket it into its own
	// opcode — see LIST_CLEAR's doc comment), a plain GET_MEMBER's property
	// name is only known at runtime as a string in chunk.Names, so this has
	// to be a VM-level setting instead. Defaults to 하자's word; callers
	// running Kanade bytecode set it to "長さ" (see cmd/run.go).
	LengthWord string

	// StringSliceMethod/StringReplaceMethod/StringSplitMethod/
	// StringContainsMethod name the four string pseudo-methods
	// (자르기/바꾸기/분리하기/포함확인), same reasoning as LengthWord —
	// callStringMethod compares a runtime *boundStringMethod.Method string
	// against these.
	StringSliceMethod    string
	StringReplaceMethod  string
	StringSplitMethod    string
	StringContainsMethod string

	// NullString/TrueString/FalseString/ObjectFormat are formatValue's
	// display strings for nil/bool/*Object (하자: "비어있음"/"참"/"거짓"/
	// "[%s 객체]" — mirrors vm.LangConfig's same-named fields).
	NullString   string
	TrueString   string
	FalseString  string
	ObjectFormat string

	// ErrorClass/ErrorMessage name the built-in error class and its message
	// property a `발생했다면` handler binds an engine-raised error to (하자:
	// "오류"/"메시지" — mirrors vm.LangConfig.BuiltinErrorClass/
	// BuiltinErrorMessage). Locale picks the wording errs.Localize renders
	// that message in (mirrors vm.LangConfig.Locale).
	ErrorClass   string
	ErrorMessage string
	Locale       errs.Locale

	// EqualsMethod is the magic method `==`/`!=` calls on a class instance's
	// left operand (operator overloading; mirrors vm.LangConfig.EqualsMethodName).
	// VarQuoteOpen/VarQuoteClose are the VAR token's quote characters, needed
	// for dynamic reflection `<'변수'>()` — a <...> name that is itself a
	// quoted variable means "the method named by that variable's value"
	// (mirrors vm.LangConfig.VarQuoteOpen/Close).
	EqualsMethod  string
	VarQuoteOpen  string
	VarQuoteClose string
}

// UseJapaneseWords switches every 하자-default runtime word (length,
// string-pseudo-methods, nil/bool/object display strings) to 카나데's
// (kanade-docs' actual wording — see vm.JapaneseConfig, which this
// mirrors). One call instead of setting several fields at every bcvm.New
// call site for a Kanade program, specifically because forgetting one of
// several scattered same-purpose fields is exactly how the
// PluralSelfWords/私達 bug happened (see hana/CLAUDE.md).
func (vm *VM) UseJapaneseWords() {
	vm.LengthWord = "長さ"
	vm.StringSliceMethod = "切り取り"
	vm.StringReplaceMethod = "入れ替え"
	vm.StringSplitMethod = "分割"
	vm.StringContainsMethod = "含むか確認"
	vm.NullString = "空っぽ"
	vm.TrueString = "真"
	vm.FalseString = "偽"
	vm.ObjectFormat = "[%s オブジェクト]"
	vm.ErrorClass = "エラー"
	vm.ErrorMessage = "メッセージ"
	vm.Locale = errs.Japanese
	vm.EqualsMethod = "記号 同じだ"
	vm.VarQuoteOpen = "『"
	vm.VarQuoteClose = "』"
	vm.Types = typecheck.Names{Number: "数字", String: "文字列", Boolean: "論理", Any: "何でも", List: "リスト", Dict: "辞書", Null: "空っぽ"}
}

func New(program *bytecode.Program) *VM {
	return &VM{
		program:              program,
		globals:              newFrame(),
		staticFields:         make(map[string]interface{}),
		natives:              make(map[string]*NativeFunction),
		nativeModules:        make(map[string]map[string]*NativeFunction),
		LengthWord:           "길이",
		Types:                typecheck.Names{Number: "숫자", String: "문자열", Boolean: "논리", Any: "아무거나", List: "목록", Dict: "사전", Null: "비어있음"},
		StringSliceMethod:    "자르기",
		StringReplaceMethod:  "바꾸기",
		StringSplitMethod:    "분리하기",
		StringContainsMethod: "포함확인",
		NullString:           "비어있음",
		TrueString:           "참",
		FalseString:          "거짓",
		ObjectFormat:         "[%s 객체]",
		ErrorClass:           "오류",
		ErrorMessage:         "메시지",
		Locale:               errs.Korean,
		EqualsMethod:         "기호 같다",
		VarQuoteOpen:         "'",
		VarQuoteClose:        "'",
	}
}

// RegisterNative makes fn callable by name from any compiled Hari code, no
// import required — for core language tools like 숫자로/문자로 that Runtime
// spec 1章 (묵시적 형변환 금지) requires to always be available.
func (vm *VM) RegisterNative(name string, fn *NativeFunction) {
	vm.natives[name] = fn
}

// RegisterNativeModule registers a builtin module (e.g. [수학]) whose
// functions only become callable by name once a Hari program actually runs
// [모듈]에서 <이름>을 가져오자 (IMPORT_NATIVE) — mirrors
// vm.Interpreter.RegisterNativeModule/NativeModules exactly.
func (vm *VM) RegisterNativeModule(name string, module map[string]*NativeFunction) {
	vm.nativeModules[name] = module
}

// Run pre-flight-checks every class's declared interfaces (mirrors
// vm.Interpreter.Run()'s step 2: a class claiming to implement an
// interface must define every method that interface requires, checked
// against the class's own Methods only, not the inherited chain), then
// executes the program's Main chunk at global scope — which itself runs
// each class's static field initializers inline at their source position
// (see compileClassStaticInit).
func (vm *VM) Run() error {
	vm.runLock.Acquire()
	defer vm.runLock.Release()
	for clsName, cls := range vm.program.Classes {
		for _, ifaceName := range cls.Interfaces {
			iface, ok := vm.program.Interfaces[ifaceName]
			if !ok {
				continue
			}
			for _, methodName := range iface.Methods {
				if _, ok := cls.Methods[methodName]; !ok {
					return errs.New(errs.InterfaceNotImplemented, clsName, ifaceName, methodName)
				}
			}
		}
	}
	_, err := vm.exec(vm.program.Main, nil)
	return err
}

// exec runs chunk's instructions against a fresh operand stack. locals is
// the current call frame, or nil when executing at global scope (in which
// case variables resolve directly against vm.globals).
// maxExecDepth bounds nested chunk executions so runaway recursion becomes a
// catchable RecursionError instead of Go's fatal stack overflow. Same number as
// vm.MaxCallDepth in the tree-walker.
const maxExecDepth = 10000

// maxFormatDepth stops FormatValue on a list that contains itself.
const maxFormatDepth = 100

// evalIndex runs an index/property expression chunk (`<expr> RETURN`). The
// overwhelmingly common ones are a lone variable or constant, which are read
// directly instead of starting a whole exec call for them.
func (vm *VM) evalIndex(chunk *bytecode.Chunk, locals *frame) (interface{}, error) {
	if ins := chunk.Instructions; len(ins) == 2 && ins[1].Op == bytecode.RETURN {
		switch ins[0].Op {
		case bytecode.PUSH_CONST:
			return chunk.Constants[ins[0].Operand.(int)], nil
		case bytecode.LOAD_VAR:
			idx := ins[0].Operand.(int)
			val, ok := vm.lookupOrClass(locals, chunk.Symbols()[idx])
			if !ok {
				return nil, errs.New(errs.VariableNotFound, chunk.Names[idx])
			}
			return val, nil
		}
	}
	return vm.exec(chunk, locals)
}

// exec runs chunk. The depth counter is kept here rather than with a defer in run: run has
// dozens of return statements, which keeps Go from making a defer cheap, and every call
// of a hana function goes through here.
func (vm *VM) exec(chunk *bytecode.Chunk, locals *frame) (interface{}, error) {
	vm.depth++
	if vm.depth > maxExecDepth {
		vm.depth--
		return nil, errs.New(errs.CallTooDeep, maxExecDepth)
	}
	res, err := vm.run(chunk, locals)
	vm.depth--
	return res, err
}

func (vm *VM) run(chunk *bytecode.Chunk, locals *frame) (interface{}, error) {
	syms := chunk.Symbols()
	stack := make([]interface{}, 0, 16)
	push := func(v interface{}) { stack = append(stack, v) }
	pop := func() interface{} {
		v := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return v
	}

	// tryStack holds this exec() call's active TRY_PUSH frames. raise is the
	// common path every runtime error in this loop goes through instead of
	// returning directly — see dispatchError's doc comment. When it reports
	// handled==false, err has already been through it and may have been
	// overwritten by an intervening finally block's own error, so callers
	// must return err (the parameter shadows the caller's — use the
	// returned one), not whatever local error value they started with.
	var tryStack []*tryHandler
	raise := func(err error) (newPc int, handled bool, resultErr error) {
		return vm.dispatchError(err, &tryStack, &stack, push, locals)
	}

	pc := 0
	for pc < len(chunk.Instructions) {
		instr := &chunk.Instructions[pc]
		switch instr.Op {
		case bytecode.PUSH_CONST:
			push(chunk.Constants[instr.Operand.(int)])
		case bytecode.PUSH_NULL:
			push(nil)
		case bytecode.PUSH_BOOL:
			push(instr.Operand.(bool))

		case bytecode.LOAD_VAR:
			idx := instr.Operand.(int)
			val, ok := vm.loadVar(instr, locals, syms[idx], 0)
			if !ok {
				np, handled, rerr := raise(errs.New(errs.VariableNotFound, chunk.Names[idx]))
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			push(val)
		case bytecode.SET_VAR, bytecode.SET_CONST:
			if instr.Op == bytecode.SET_VAR && vm.storeVar(instr, locals, syms[instr.Operand.(int)], stack[len(stack)-1], 0) {
				stack = stack[:len(stack)-1]
				break
			}
			if err := vm.assignOrDeclare(locals, syms[instr.Operand.(int)], pop(), instr.Op == bytecode.SET_CONST, ""); err != nil {
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}

		case bytecode.SET_VAR_TYPED:
			op := instr.Operand.(*bytecode.TypedSetOperand)
			if err := vm.assignOrDeclare(locals, syms[op.NameIndex], pop(), op.Const, op.Type); err != nil {
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}

		case bytecode.DECLARE_VAR:
			vm.declaringFrame(locals).declare(syms[instr.Operand.(int)], pop())
		case bytecode.ILLEGAL_BREAK:
			np, handled, rerr := raise(errs.New(errs.IllegalBreak))
			if handled {
				pc = np
				continue
			}
			return nil, rerr
		case bytecode.CHECK_RANGE:
			_, startOK := stack[len(stack)-2].(float64)
			_, endOK := stack[len(stack)-1].(float64)
			if !startOK || !endOK {
				np, handled, rerr := raise(errs.New(errs.RangeMustBeNumbers))
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}

		case bytecode.PUSH_SCOPE:
			vm.declaringFrame(locals).openScope(syms[instr.Operand.(int)])
		case bytecode.POP_SCOPE:
			vm.declaringFrame(locals).closeScope(syms[instr.Operand.(int)])

		case bytecode.POP:
			pop()

		case bytecode.FOR_STEP:
			if next, ok := vm.forStep(instr, chunk, syms, locals); ok {
				pc = next
				continue
			}
			// Not a plain number loop: do Inc as the BIN it was and go on to the JUMP.
			fallthrough
		case bytecode.BIN:
			op, isBin := instr.Operand.(*bytecode.BinOperand)
			if !isBin {
				op = instr.Operand.(*bytecode.ForStepOperand).Inc
			}
			var left, right interface{}
			var missing string
			switch op.R.Kind {
			case bytecode.ArgStack:
				right = pop()
			case bytecode.ArgVar:
				v, ok := vm.loadVar(instr, locals, syms[op.R.Index], 2)
				if !ok {
					missing = chunk.Names[op.R.Index]
				}
				right = v
			default:
				right = chunk.Constants[op.R.Index]
			}
			switch op.L.Kind {
			case bytecode.ArgStack:
				left = pop()
			case bytecode.ArgVar:
				v, ok := vm.loadVar(instr, locals, syms[op.L.Index], 0)
				if !ok {
					missing = chunk.Names[op.L.Index]
				}
				left = v
			default:
				left = chunk.Constants[op.L.Index]
			}
			var result interface{}
			var err error
			if missing != "" {
				err = errs.New(errs.VariableNotFound, missing)
			} else if r, done := binaryFast(op.Op, left, right); done {
				result = r
			} else {
				result, err = vm.binaryWithEquals(op.Op, left, right)
			}
			if err == nil {
				switch {
				case op.Set >= 0:
					if !vm.storeVar(instr, locals, syms[op.Set], result, 4) {
						err = vm.assignOrDeclare(locals, syms[op.Set], result, false, "")
					}
				case op.Jump >= 0:
					var b bool
					if b, err = vm.requireBool(result); err == nil && !b {
						pc = op.Jump
						continue
					}
				default:
					push(result)
				}
			}
			if err != nil {
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}

		case bytecode.ADD, bytecode.SUB, bytecode.MUL, bytecode.DIV, bytecode.MOD,
			bytecode.EQ, bytecode.NEQ, bytecode.LT, bytecode.LTE, bytecode.GT, bytecode.GTE:
			right := pop()
			left := pop()
			if r, done := binaryFast(instr.Op, left, right); done {
				push(r)
				pc++
				continue
			}
			result, err := vm.binaryWithEquals(instr.Op, left, right)
			if err != nil {
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			push(result)
		case bytecode.NOT:
			b, err := vm.requireBool(pop())
			if err != nil {
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			push(!b)
		case bytecode.TO_ITERABLE:
			switch v := stack[len(stack)-1].(type) {
			case *value.List:
				// a loop walks the list as it was when the loop began
				stack[len(stack)-1] = value.NewList(append([]interface{}(nil), v.Items...))
			case string:
				chars := make([]interface{}, 0, len(v))
				for _, r := range v {
					chars = append(chars, string(r))
				}
				stack[len(stack)-1] = value.NewList(chars)
			default:
				np, handled, rerr := raise(errs.New(errs.NotIterable))
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
		case bytecode.INIT_MODULE:
			op := instr.Operand.(*bytecode.ModuleInit)
			if !vm.moduleReady[op.Name] {
				if vm.moduleReady == nil {
					vm.moduleReady = map[string]bool{}
				}
				vm.moduleReady[op.Name] = true
				if _, err := vm.exec(op.Body, vm.moduleFrame(op.Name)); err != nil {
					np, handled, rerr := raise(err)
					if handled {
						pc = np
						continue
					}
					return nil, rerr
				}
			}
		case bytecode.SET_LIST_VAR:
			op := instr.Operand.(*bytecode.ListSetOperand)
			list, _ := pop().(*value.List)
			if err := vm.checkListVar(locals, syms[op.NameIndex], list, op.Change); err != nil {
				list.Unpush(op.Change == bytecode.ListPushedFront)
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
		case bytecode.CHECK_CONST_VAR:
			if err := vm.requireMutable(locals, syms[instr.Operand.(int)]); err != nil {
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
		case bytecode.CHECK_LIST_FIELD:
			op := instr.Operand.(*bytecode.ListSetOperand)
			obj := pop()
			list, _ := pop().(*value.List)
			if err := vm.checkListField(obj, chunk.Names[op.NameIndex], list, op.Change); err != nil {
				list.Unpush(op.Change == bytecode.ListPushedFront)
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
		case bytecode.CHECK_BOOL:
			if _, err := vm.requireBool(stack[len(stack)-1]); err != nil {
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}

		case bytecode.JUMP:
			pc = instr.Operand.(int)
			continue
		case bytecode.JUMP_IF_FALSE:
			b, err := vm.requireBool(pop())
			if err != nil {
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			if !b {
				pc = instr.Operand.(int)
				continue
			}
		case bytecode.JUMP_IF_TRUE:
			b, err := vm.requireBool(pop())
			if err != nil {
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			if b {
				pc = instr.Operand.(int)
				continue
			}

		case bytecode.CALL:
			op := instr.Operand.(*bytecode.CallOperand)
			name := chunk.Names[op.NameIndex]
			fn := op.CachedFunction()
			if fn == nil {
				name = vm.resolveDynamicName(name, locals)
				if f, ok := vm.program.Functions[name]; ok {
					fn = f
					if name == chunk.Names[op.NameIndex] {
						op.CacheFunction(f) // a plain name always means this function
					}
				}
			}
			var result interface{}
			var err error
			if fn != nil {
				// a function of the program binds the arguments into its own frame, which
				// copies them, so they can be read where they lie on the stack
				// (the arguments are not cleared off: the next push overwrites them)
				result, err = vm.callFunction(fn, stack[len(stack)-op.Argc:])
				stack = stack[:len(stack)-op.Argc]
			} else if native, ok := vm.natives[name]; ok {
				args := make([]interface{}, op.Argc)
				for i := op.Argc - 1; i >= 0; i-- {
					args[i] = pop()
				}
				result, err = native.Fn(vm, args)
			} else {
				err = errs.New(errs.GlobalFunctionNotFound, name)
			}
			if err != nil {
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			push(result)
		case bytecode.RETURN:
			return pop(), nil

		case bytecode.PRINT:
			s, _ := pop().(string)
			vm.Output = append(vm.Output, s)
			console.Println(s)
		case bytecode.PRINT_INLINE:
			s, _ := pop().(string)
			vm.Output = append(vm.Output, s)
			console.Print(s)
		case bytecode.TO_DISPLAY_STRING:
			push(vm.FormatValue(pop()))

		case bytecode.NEW_LIST:
			n := instr.Operand.(int)
			elems := make([]interface{}, n)
			for i := n - 1; i >= 0; i-- {
				elems[i] = pop()
			}
			push(value.NewList(elems))

		case bytecode.NEW_DICT:
			n := instr.Operand.(int)
			pairs := make([]interface{}, 2*n)
			for i := 2*n - 1; i >= 0; i-- {
				pairs[i] = pop()
			}
			dict := make(map[interface{}]interface{}, n)
			for i := 0; i < n; i++ {
				dict[pairs[2*i]] = pairs[2*i+1]
			}
			push(dict)

		case bytecode.LIST_PUSH:
			position := instr.Operand.(string)
			val := pop()
			listVal := pop()
			list, ok := listVal.(*value.List)
			if !ok {
				np, handled, rerr := raise(errs.New(errs.NotAList))
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			list.Push(val, position == "front")
			push(list)

		case bytecode.LIST_POP:
			position := instr.Operand.(string)
			listVal := pop()
			list, ok := listVal.(*value.List)
			if !ok {
				np, handled, rerr := raise(errs.New(errs.NotAList))
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			if len(list.Items) == 0 {
				np, handled, rerr := raise(errs.New(errs.ListEmpty))
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			push(list.Pop(position))

		case bytecode.INPUT:
			typeName, _ := instr.Operand.(string)
			text := ""
			if vm.ReadLine != nil {
				text = vm.ReadLine()
			}
			val, ierr := conv.ParseInput(typeName, vm.Types, text, vm.TrueString, vm.FalseString)
			if ierr != nil {
				np, handled, rerr := raise(ierr)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			push(val)

		case bytecode.LIST_CLEAR:
			list, ok := pop().(*value.List)
			if !ok {
				np, handled, rerr := raise(errs.New(errs.ListOnlyMethod))
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			list.Items = []interface{}{}

		case bytecode.GET_INDEX:
			idxVal := pop()
			listVal := pop()
			list, ok := listVal.(*value.List)
			if !ok {
				np, handled, rerr := raise(errs.New(errs.NotAList))
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			num, ok := idxVal.(float64)
			if !ok {
				np, handled, rerr := raise(errs.New(errs.ListIndexMustBeNumber))
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			idx := int(num) - 1 // 1-based -> 0-based
			if idx < 0 || idx >= len(list.Items) {
				np, handled, rerr := raise(errs.New(errs.ListIndexOutOfRange))
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			push(list.Items[idx])
		case bytecode.GET_LENGTH:
			list, ok := pop().(*value.List)
			if !ok {
				np, handled, rerr := raise(errs.New(errs.NotAList))
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			push(num.Box(float64(len(list.Items))))

		case bytecode.NEW_OBJECT:
			op := instr.Operand.(*bytecode.NewObjectOperand)
			args := stack[len(stack)-op.Argc:]
			obj, err := vm.newObject(chunk.Names[op.ClassNameIndex], args, locals)
			clear(args)
			stack = stack[:len(stack)-op.Argc]
			if err != nil {
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			push(obj)

		case bytecode.GET_MEMBER:
			op := instr.Operand.(*bytecode.MemberOperand)
			objVal := pop()
			result, err := vm.getMember(objVal, chunk.Names[op.PropNameIndex], syms[op.PropNameIndex], op.IsFunc, op.IndexChunk, locals)
			if err != nil {
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			push(result)

		case bytecode.SET_MEMBER:
			op := instr.Operand.(*bytecode.MemberOperand)
			objVal := pop()
			val := pop()
			if err := vm.setMember(objVal, chunk.Names[op.PropNameIndex], syms[op.PropNameIndex], val, op.IndexChunk, locals); err != nil {
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}

		case bytecode.CALL_METHOD:
			argc := instr.Operand.(int)
			args := stack[len(stack)-argc:]
			calleeVal := stack[len(stack)-argc-1]
			result, err := vm.callMethod(calleeVal, args)
			clear(stack[len(stack)-argc-1:])
			stack = stack[:len(stack)-argc-1]
			if err != nil {
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			push(result)

		case bytecode.LOAD_SELF:
			if locals != nil && locals.this != nil {
				push(locals.this)
				break
			}
			if nameIdx, quoted := instr.Operand.(int); quoted {
				// A quoted '나' with no 나 bound is just a variable.
				if val, ok := vm.lookupOrClass(locals, syms[nameIdx]); ok {
					push(val)
					break
				}
				np, handled, rerr := raise(errs.New(errs.VariableNotFound, chunk.Names[nameIdx]))
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			np, handled, rerr := raise(errs.New(errs.ThisNotBound))
			if handled {
				pc = np
				continue
			}
			return nil, rerr
		case bytecode.LOAD_SUPER:
			if locals == nil || locals.this == nil {
				np, handled, rerr := raise(errs.New(errs.SuperOutsideMethod))
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			push(&superRef{Receiver: locals.this})
		case bytecode.LOAD_STATIC_CLASS:
			if locals != nil && locals.selfClass != "" {
				push(&classRef{ClassName: locals.selfClass})
				break
			}
			if nameIdx, quoted := instr.Operand.(int); quoted {
				if val, ok := vm.lookupOrClass(locals, syms[nameIdx]); ok {
					push(val)
					break
				}
				np, handled, rerr := raise(errs.New(errs.VariableNotFound, chunk.Names[nameIdx]))
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			np, handled, rerr := raise(errs.New(errs.StaticOutsideMethod))
			if handled {
				pc = np
				continue
			}
			return nil, rerr
		case bytecode.PUSH_CLASS_REF:
			push(&classRef{ClassName: chunk.Names[instr.Operand.(int)]})

		case bytecode.INSTANCE_OF:
			right := pop()
			left := pop()
			result := false
			if obj, ok := left.(*Object); ok {
				if cr, ok := right.(*classRef); ok {
					result = vm.classIsOrExtends(obj.ClassName, cr.ClassName)
				}
			}
			push(result)

		case bytecode.SET_STATIC_FIELD:
			op := instr.Operand.(*bytecode.StaticFieldOperand)
			val := pop()
			vm.staticFields[chunk.Names[op.ClassNameIndex]+"."+chunk.Names[op.FieldNameIndex]] = val

		case bytecode.THROW:
			np, handled, rerr := raise(&thrownValue{Value: pop()})
			if handled {
				pc = np
				continue
			}
			return nil, rerr

		case bytecode.TRY_PUSH:
			op := instr.Operand.(*bytecode.TryOperand)
			fr := vm.declaringFrame(locals)
			tryStack = append(tryStack, &tryHandler{catches: op.Catches, finally: op.FinallyChunk, stackDepth: len(stack), frame: fr, frameDepth: len(fr.slots)})
		case bytecode.TRY_POP:
			tryStack = tryStack[:len(tryStack)-1]
		case bytecode.RUN_FINALLY:
			fc := instr.Operand.(*bytecode.Chunk)
			if _, err := vm.exec(fc, locals); err != nil {
				np, handled, rerr := raise(err)
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}

		case bytecode.PUSH_FUNC_REF:
			push(&funcRef{Name: chunk.Names[instr.Operand.(int)]})

		case bytecode.IMPORT_NATIVE:
			op := instr.Operand.(*bytecode.ImportOperand)
			moduleName := chunk.Names[op.ModuleNameIndex]
			if op.Library {
				if err := vm.bindLibraryFunction(moduleName, chunk.Names[op.TargetNameIndex], chunk.Names[op.BindNameIndex]); err != nil {
					np, handled, rerr := raise(err)
					if handled {
						pc = np
						continue
					}
					return nil, rerr
				}
				pc++
				continue
			}
			module, ok := vm.nativeModules[moduleName]
			if ok && op.All {
				for name, fn := range module {
					vm.natives[name] = fn
				}
				pc++
				continue
			}
			var fn *NativeFunction
			moduleExists := ok
			if ok {
				fn, ok = module[chunk.Names[op.TargetNameIndex]]
			}
			if !ok {
				code, args := errs.ImportPackageNotFound, []interface{}{moduleName}
				if pkg.IsPackagePath(moduleName) {
					code = errs.ImportPackageNotInstalled
				} else if moduleExists {
					code, args = errs.ImportTargetNotFound, []interface{}{moduleName, chunk.Names[op.TargetNameIndex]}
				}
				np, handled, rerr := raise(errs.New(code, args...))
				if handled {
					pc = np
					continue
				}
				return nil, rerr
			}
			vm.natives[chunk.Names[op.BindNameIndex]] = fn

		case bytecode.HALT:
			return nil, nil

		default:
			return nil, fmt.Errorf("bcvm: unimplemented opcode %v", instr.Op)
		}
		pc++
	}
	return nil, nil
}

// callFunction binds args to fn's parameters (Runtime spec 1.2: missing a
// required argument is MissingArgumentError, too many is ArgumentError,
// a missing argument with a default evaluates it) and runs its body in a
// fresh frame.
func (vm *VM) callFunction(fn *bytecode.Function, args []interface{}) (interface{}, error) {
	callFrame := vm.takeFrame()
	callFrame.module = vm.moduleFrame(fn.Module)
	if err := vm.bindArgs(fn, args, callFrame); err != nil {
		return nil, err
	}
	res, err := vm.exec(fn.BodyChunk, callFrame)
	vm.giveFrame(callFrame)
	return vm.checkedReturn(fn, res, err)
}

// checkedReturn is what a call gives back: an error stays an error, and a value
// has to fit the type the function declared it returns (if it declared one).
func (vm *VM) checkedReturn(fn *bytecode.Function, res interface{}, err error) (interface{}, error) {
	if err != nil || fn.ReturnType == "" || typecheck.QuickAccepts(&vm.Types, fn.ReturnType, res) {
		return res, err
	}
	if err := typecheck.CheckReturn(&vm.Types, fn.ReturnType, fn.Name, res, vm.host()); err != nil {
		return nil, err
	}
	return res, nil
}

// bindArgs binds args to params into frame's own vars, implementing
// Runtime spec 1.2 exactly like vm.bindParams: too many arguments is a
// hard ArgumentError; a missing one falls back to its declared default
// (evaluated in frame, so it can see earlier-bound parameters and — for a
// method/constructor — 나) or, with no default, is a hard
// MissingArgumentError.
func (vm *VM) bindArgs(fn *bytecode.Function, args []interface{}, fr *frame) error {
	params, syms := fn.Params, fn.ParamSymbols()
	if len(args) > len(params) {
		return errs.New(errs.TooManyArguments, len(params), len(args))
	}
	if len(args) == len(params) {
		// the common case: every parameter has its argument
		for i := range params {
			if err := vm.bindParam(fr, syms[i], &params[i], args[i]); err != nil {
				return err
			}
		}
		return nil
	}
	for i := range params {
		p := &params[i]
		if i < len(args) {
			if err := vm.bindParam(fr, syms[i], p, args[i]); err != nil {
				return err
			}
			continue
		}
		if p.Default != nil {
			val, err := vm.exec(p.Default, fr)
			if err != nil {
				return err
			}
			if err := vm.bindParam(fr, syms[i], p, val); err != nil {
				return err
			}
			continue
		}
		return errs.New(errs.MissingArgument, p.Name)
	}
	return nil
}

// newObject constructs an instance of className: initializes that class's
// own declared fields (never the ancestors' — see ClassInfo's doc comment,
// this replicates vm/eval_expr.go's NewExpression exactly, defaults included
// evaluated in the *caller's* frame, before 나 exists), then finds a
// constructor by walking the BaseClass chain and runs it with 나 bound.
func (vm *VM) newObject(className string, args []interface{}, callerFrame *frame) (*Object, error) {
	cls, ok := vm.program.Classes[className]
	if !ok {
		if _, isIface := vm.program.Interfaces[className]; isIface {
			return nil, errs.New(errs.InstantiateInterface, className)
		}
		return nil, errs.New(errs.ClassNotFound, className)
	}
	if cls.IsAbstract {
		return nil, errs.New(errs.InstantiateAbstract, className)
	}
	obj := newObject(className)
	for _, f := range cls.Fields {
		if f.Default == nil {
			continue
		}
		val, err := vm.exec(f.Default, callerFrame)
		if err != nil {
			return nil, err
		}
		if f.Type != "" {
			if err := typecheck.Check(&vm.Types, f.Type, f.Name, val, vm.host()); err != nil {
				return nil, err
			}
		}
		obj.Props[f.Name] = val
	}

	ctor, found := vm.findMethod(className, "__init__")
	if found {
		ctorFrame := newFrame()
		ctorFrame.module = vm.moduleFrame(ctor.Module)
		ctorFrame.this = obj
		ctorFrame.selfClass = className
		if err := vm.bindArgs(ctor, args, ctorFrame); err != nil {
			return nil, err
		}
		if _, err := vm.exec(ctor.BodyChunk, ctorFrame); err != nil {
			return nil, err
		}
	}
	return obj, nil
}

// getMember resolves a MemberExpression read at runtime by objVal's actual
// type — see MemberOperand's doc comment for why this can't be decided at
// compile time.
func (vm *VM) getMember(objVal interface{}, propName string, sym symbol.Symbol, isFunc bool, indexChunk *bytecode.Chunk, locals *frame) (interface{}, error) {
	if isFunc {
		if resolved := vm.resolveDynamicName(propName, locals); resolved != propName {
			propName, sym = resolved, symbol.Intern(resolved)
		}
	}
	switch v := objVal.(type) {
	case *superRef:
		if !isFunc {
			return nil, errs.New(errs.SuperMemberMustBeMethod)
		}
		return &boundMethod{Receiver: v.Receiver, Method: propName, IsSuper: true}, nil

	case *Object:
		entry := vm.memberOf(v, sym)
		access := entry.fieldAccess
		if isFunc {
			access = entry.methodAccess
		}
		if access != "public" {
			if locals == nil || locals.this == nil {
				return nil, errs.AccessViolation(access, isFunc, propName)
			}
			if access == "private" && locals.this != v {
				return nil, errs.AccessViolation(access, isFunc, propName)
			}
		}
		if isFunc {
			return &boundMethod{Receiver: v, Method: propName, fn: entry.method}, nil
		}
		if getter := entry.getter; getter != nil {
			getterFrame := newFrame()
			getterFrame.module = vm.moduleFrame(vm.classModule(v.ClassName))
			getterFrame.this = v
			getterFrame.selfClass = v.ClassName
			return vm.exec(getter, getterFrame)
		}
		return v.Props[propName], nil

	case *classRef:
		if isFunc {
			cls, ok := vm.program.Classes[v.ClassName]
			if ok {
				if _, ok := cls.StaticMethods[propName]; ok {
					return &boundStaticMethod{ClassName: v.ClassName, Method: propName}, nil
				}
			}
			return nil, errs.New(errs.StaticMemberNotFound, propName)
		}
		if val, ok := vm.staticFields[v.ClassName+"."+propName]; ok {
			return val, nil
		}
		return nil, errs.New(errs.StaticMemberNotFound, propName)

	case *value.List:
		if isFunc {
			return nil, errs.New(errs.MethodNotFound, propName)
		}
		if propName == vm.LengthWord {
			return num.Box(float64(len(v.Items))), nil
		}
		idxVal, err := vm.evalIndex(indexChunk, locals)
		if err != nil {
			return nil, err
		}
		num, ok := idxVal.(float64)
		if !ok {
			return nil, errs.New(errs.ListIndexMustBeNumber)
		}
		idx := int(num) - 1
		if idx < 0 || idx >= len(v.Items) {
			return nil, errs.New(errs.ListIndexOutOfRange)
		}
		return v.Items[idx], nil

	case map[interface{}]interface{}:
		// The property is evaluated as an expression to get the key (a plain
		// <이름> reference is just its own name) — vm/eval_expr.go's dict branch.
		var key interface{} = propName
		if !isFunc {
			k, err := vm.evalIndex(indexChunk, locals)
			if err != nil {
				return nil, err
			}
			key = k
		}
		if val, ok := v[key]; ok {
			return val, nil
		}
		return nil, errs.New(errs.DictKeyNotFound, key)

	case string:
		if isFunc {
			return &boundStringMethod{Value: v, Method: propName}, nil
		}
		if propName == vm.LengthWord {
			return float64(utf8.RuneCountInString(v)), nil
		}
		idxVal, err := vm.evalIndex(indexChunk, locals)
		if err != nil {
			return nil, err
		}
		num, ok := idxVal.(float64)
		if !ok {
			return nil, errs.New(errs.MemberAccessOnString)
		}
		char, ok := conv.RuneAt(v, int(num)-1)
		if !ok {
			return nil, errs.New(errs.StringIndexOutOfRange)
		}
		return char, nil
	}
	return nil, errs.New(errs.MemberAccessUnsupported, errs.TypeNameOf(objVal))
}

// setMember resolves a MemberExpression assignment target at runtime,
// mirroring vm/exec_stmt.go's Assignment case — including that it, unlike
// getMember, applies no access-modifier check on the write path.
func (vm *VM) setMember(objVal interface{}, propName string, sym symbol.Symbol, val interface{}, indexChunk *bytecode.Chunk, locals *frame) error {
	switch v := objVal.(type) {
	case *Object:
		if setter := vm.memberOf(v, sym).setter; setter != nil {
			setterFrame := newFrame()
			setterFrame.module = vm.moduleFrame(vm.classModule(v.ClassName))
			setterFrame.this = v
			setterFrame.selfClass = v.ClassName
			if setter.ParamName != "" {
				setterFrame.put(symbol.Intern(setter.ParamName), val)
			}
			_, err := vm.exec(setter.Body, setterFrame)
			return err
		}
		if err := vm.checkField(v.ClassName, propName, val); err != nil {
			return err
		}
		v.Props[propName] = val
		return nil

	case *value.List:
		idxVal, err := vm.evalIndex(indexChunk, locals)
		if err != nil {
			return err
		}
		if num, ok := idxVal.(float64); ok {
			idx := int(num) - 1
			if idx >= 0 && idx < len(v.Items) {
				v.Items[idx] = val
			}
		}
		return nil

	case map[interface{}]interface{}:
		// vm/exec_stmt.go: use the evaluated property as the key; if it can't
		// be evaluated, a bare identifier property is the key's own name.
		if key, err := vm.evalIndex(indexChunk, locals); err == nil {
			v[key] = val
		} else if propName != "" {
			v[propName] = val
		}
		return nil

	case string:
		return errs.New(errs.StringIndexAssign)

	case *classRef:
		vm.staticFields[v.ClassName+"."+propName] = val
		return nil
	}
	return nil // 지원 안 하는 대상은 tree-walker와 동일하게 조용히 무시
}

// callMethod dispatches a bound (instance or static) method value produced
// by getMember. Mirrors vm/eval_expr.go's BoundMethod/BoundStaticMethod
// CallExpression branches.
func (vm *VM) callMethod(calleeVal interface{}, args []interface{}) (interface{}, error) {
	switch bm := calleeVal.(type) {
	case *boundMethod:
		startClass := bm.Receiver.ClassName
		if bm.IsSuper {
			cls, ok := vm.program.Classes[startClass]
			if !ok || cls.BaseClass == "" {
				return nil, errs.New(errs.MethodNotFound, bm.Method)
			}
			startClass = cls.BaseClass
		}
		fn := bm.fn
		if fn == nil || bm.IsSuper {
			var ok bool
			if fn, ok = vm.findMethod(startClass, bm.Method); !ok {
				return nil, errs.New(errs.MethodNotFound, bm.Method)
			}
		}
		methodFrame := vm.takeFrame()
		methodFrame.module = vm.moduleFrame(fn.Module)
		methodFrame.this = bm.Receiver
		methodFrame.selfClass = bm.Receiver.ClassName
		if err := vm.bindArgs(fn, args, methodFrame); err != nil {
			return nil, err
		}
		res, err := vm.exec(fn.BodyChunk, methodFrame)
		vm.giveFrame(methodFrame)
		return vm.checkedReturn(fn, res, err)

	case *boundStaticMethod:
		cls, ok := vm.program.Classes[bm.ClassName]
		if !ok {
			return nil, errs.New(errs.StaticMethodNotFound, bm.Method)
		}
		fn, ok := cls.StaticMethods[bm.Method]
		if !ok {
			return nil, errs.New(errs.StaticMethodNotFound, bm.Method)
		}
		methodFrame := newFrame()
		methodFrame.module = vm.moduleFrame(fn.Module)
		methodFrame.selfClass = bm.ClassName
		if err := vm.bindArgs(fn, args, methodFrame); err != nil {
			return nil, err
		}
		res, err := vm.exec(fn.BodyChunk, methodFrame)
		return vm.checkedReturn(fn, res, err)

	case *boundStringMethod:
		return vm.callStringMethod(bm, args)
	}
	return nil, errs.New(errs.NotCallable)
}

// callStringMethod implements the builtin pseudo-methods a string value
// exposes, mirroring vm/eval_expr.go's BoundStringMethod dispatch exactly
// for the success path. One deliberate improvement: the tree-walker
// indexes/type-asserts args without checking length or type first, which
// panics (crashing the whole process, uncatchable) on a wrong call instead
// of raising a Hari-level error — this version checks first and returns a
// normal ArgumentError/TypeError instead.
func (vm *VM) callStringMethod(bm *boundStringMethod, args []interface{}) (interface{}, error) {
	switch bm.Method {
	case vm.StringSliceMethod:
		if len(args) != 2 {
			return nil, errs.New(errs.ArgCountExact, 2)
		}
		startNum, ok1 := args[0].(float64)
		endNum, ok2 := args[1].(float64)
		if !ok1 || !ok2 {
			return nil, errs.New(errs.MethodArgMustBeNumber, bm.Method)
		}
		runes := []rune(bm.Value)
		start := int(startNum) - 1
		end := int(endNum)
		if start < 0 {
			start = 0
		}
		if end > len(runes) {
			end = len(runes)
		}
		if start > end {
			start = end
		}
		return string(runes[start:end]), nil

	case vm.StringReplaceMethod:
		if len(args) != 2 {
			return nil, errs.New(errs.ArgCountExact, 2)
		}
		oldStr, ok1 := args[0].(string)
		newStr, ok2 := args[1].(string)
		if !ok1 || !ok2 {
			return nil, errs.New(errs.MethodArgMustBeString, bm.Method)
		}
		return strings.ReplaceAll(bm.Value, oldStr, newStr), nil

	case vm.StringSplitMethod:
		if len(args) != 1 {
			return nil, errs.New(errs.ArgCountExact, 1)
		}
		sep, ok := args[0].(string)
		if !ok {
			return nil, errs.New(errs.MethodArgMustBeString, bm.Method)
		}
		parts := strings.Split(bm.Value, sep)
		res := make([]interface{}, len(parts))
		for i, p := range parts {
			res[i] = p
		}
		return value.NewList(res), nil

	case vm.StringContainsMethod:
		if len(args) != 1 {
			return nil, errs.New(errs.ArgCountExact, 1)
		}
		sub, ok := args[0].(string)
		if !ok {
			return nil, errs.New(errs.MethodArgMustBeString, bm.Method)
		}
		return strings.Contains(bm.Value, sub), nil
	}
	return nil, errs.New(errs.MethodNotFound, bm.Method)
}

// binaryFast is the operator applied to two numbers, the case that fills a loop. done is
// false for anything else (and for division, where zero has to be looked at), which the
// general path handles.
func binaryFast(op bytecode.Opcode, left, right interface{}) (result interface{}, done bool) {
	l, ok := left.(float64)
	if !ok {
		return nil, false
	}
	r, ok := right.(float64)
	if !ok {
		return nil, false
	}
	switch op {
	case bytecode.ADD:
		return num.Box(l + r), true
	case bytecode.SUB:
		return num.Box(l - r), true
	case bytecode.MUL:
		return num.Box(l * r), true
	case bytecode.DIV:
		if r != 0 {
			return num.Box(l / r), true
		}
	case bytecode.MOD:
		if ri := int64(r); ri != 0 {
			return num.Box(float64(int64(l) % ri)), true
		}
	case bytecode.LT:
		return l < r, true
	case bytecode.LTE:
		return l <= r, true
	case bytecode.GT:
		return l > r, true
	case bytecode.GTE:
		return l >= r, true
	case bytecode.EQ:
		return l == r, true
	case bytecode.NEQ:
		return l != r, true
	}
	return nil, false
}

// binaryWithEquals is binaryOp plus the equality magic method of an object on the left.
func (vm *VM) binaryWithEquals(op bytecode.Opcode, left, right interface{}) (interface{}, error) {
	if obj, isObj := left.(*Object); isObj && (op == bytecode.EQ || op == bytecode.NEQ) && vm.hasEqualsMethod(obj) {
		return vm.callEqualsMethod(obj, right, op == bytecode.NEQ)
	}
	return vm.binaryOp(op, left, right)
}

// forStep is FOR_STEP's fast path: when the counter and the end are numbers and the
// counter's variable takes the new number as it is, it adds the step, stores it,
// compares, and returns the instruction to go to. Otherwise it changes nothing and
// reports false, and the caller runs Inc as a BIN, then the JUMP and Test as before.
func (vm *VM) forStep(instr *bytecode.Instruction, chunk *bytecode.Chunk, syms []symbol.Symbol, locals *frame) (int, bool) {
	op := instr.Operand.(*bytecode.ForStepOperand)
	sym := syms[op.Inc.L.Index]
	var s *varSlot
	if locals != nil {
		s = probe(locals, sym, &instr.Hint[0])
	}
	if s == nil {
		s = probe(vm.globalsFor(locals), sym, &instr.Hint[1])
	}
	if s == nil || s.isConst {
		return 0, false
	}
	cur, ok := s.val.(float64)
	if !ok {
		return 0, false
	}
	// A step or end kept in the counter's own variable would change with it: leave
	// that to the plain instructions.
	step, ok := vm.forStepNumber(instr, chunk, syms, locals, op.Inc.R, sym, 4)
	if !ok {
		return 0, false
	}
	end, ok := vm.forStepNumber(instr, chunk, syms, locals, op.End, sym, 2)
	if !ok {
		return 0, false
	}
	next := cur + step
	boxed := num.Box(next)
	if s.typ != "" && !typecheck.QuickAccepts(&vm.Types, s.typ, boxed) {
		return 0, false
	}
	s.val = boxed
	var goOn bool
	switch {
	case op.Span:
		// the same arithmetic, in the same order, as the head's three BINs (the
		// conversion keeps each step rounded on its own, as the BINs are)
		goOn = float64(float64(end-next)*step) >= 0
	case op.Test.Op == bytecode.LT:
		goOn = next < end
	case op.Test.Op == bytecode.LTE:
		goOn = next <= end
	case op.Test.Op == bytecode.GT:
		goOn = next > end
	default: // GTE
		goOn = next >= end
	}
	if goOn {
		return op.Body, true
	}
	return op.Test.Jump, true
}

// forStepNumber reads a FOR_STEP operand (a constant or a variable other than the
// counter) as a number; false when it is not one or cannot be read.
func (vm *VM) forStepNumber(instr *bytecode.Instruction, chunk *bytecode.Chunk, syms []symbol.Symbol, locals *frame, a bytecode.BinArg, counter symbol.Symbol, hint int) (float64, bool) {
	var v interface{}
	if a.Kind == bytecode.ArgConst {
		v = chunk.Constants[a.Index]
	} else {
		sym := syms[a.Index]
		if sym == counter {
			return 0, false
		}
		var found bool
		if v, found = vm.loadVar(instr, locals, sym, hint); !found {
			return 0, false
		}
	}
	f, ok := v.(float64)
	return f, ok
}

// probe finds sym in f, trying the slot the instruction found it in last time (*hint)
// before searching, and remembers the slot it finds. nil if f does not have it.
func probe(f *frame, sym symbol.Symbol, hint *int32) *varSlot {
	if h := int(*hint); h < len(f.slots) && f.slots[h].sym == sym && !f.slots[h].hidden {
		return &f.slots[h]
	}
	if i := f.find(sym); i >= 0 {
		*hint = int32(i)
		return &f.slots[i]
	}
	return nil
}

// loadVar is lookupOrClass for the variable an instruction names, using the slots the
// instruction found it in before.
func (vm *VM) loadVar(in *bytecode.Instruction, locals *frame, sym symbol.Symbol, hint int) (interface{}, bool) {
	if locals != nil {
		if s := probe(locals, sym, &in.Hint[hint]); s != nil {
			return s.val, true
		}
	}
	if s := probe(vm.globalsFor(locals), sym, &in.Hint[hint+1]); s != nil {
		return s.val, true
	}
	if name := sym.String(); vm.program.Classes[name] != nil {
		return &classRef{ClassName: name}, true
	}
	return nil, false
}

// storeVar is the common case of SET_VAR: the variable exists, is not constant and was
// declared without a type. It reports whether it did the store; otherwise the general
// assignOrDeclare has to.
func (vm *VM) storeVar(in *bytecode.Instruction, locals *frame, sym symbol.Symbol, val interface{}, hint int) bool {
	var s *varSlot
	if locals != nil {
		s = probe(locals, sym, &in.Hint[hint])
	}
	if s == nil {
		s = probe(vm.globalsFor(locals), sym, &in.Hint[hint+1])
	}
	if s == nil || s.isConst {
		return false
	}
	if s.typ != "" && !typecheck.QuickAccepts(&vm.Types, s.typ, val) {
		return false
	}
	s.val = val
	return true
}

func (vm *VM) lookup(locals *frame, sym symbol.Symbol) (interface{}, bool) {
	if locals != nil {
		if v, ok := locals.get(sym); ok {
			return v, true
		}
	}
	if v, ok := vm.globalsFor(locals).get(sym); ok {
		return v, true
	}
	return nil, false
}

// globalsFor is the frame a name is looked up in after the call's own variables: the
// top-level variables of the module the running function came from, or the program's
// globals for the program's own code.
func (vm *VM) globalsFor(locals *frame) *frame {
	if locals != nil && locals.module != nil {
		return locals.module
	}
	return vm.globals
}

// moduleFrame is the frame that holds the top-level variables of the module called
// id, made when first asked for. "" (the program itself) has none.
func (vm *VM) moduleFrame(id string) *frame {
	if id == "" {
		return nil
	}
	if f, ok := vm.modules[id]; ok {
		return f
	}
	f := newFrame()
	f.module = f
	if vm.modules == nil {
		vm.modules = map[string]*frame{}
	}
	vm.modules[id] = f
	return f
}

// classModule is the module a class was imported from ("" for the program's own).
func (vm *VM) classModule(className string) string {
	if cls, ok := vm.program.Classes[className]; ok {
		return cls.Module
	}
	return ""
}

// resolveDynamicName implements dynamic reflection (spec 3.8,
// `<'변수'>()`): a <...> name still wrapped in the VAR quote characters names
// a variable, and the method actually meant is that variable's string value.
// Anything else — including a quoted name whose variable isn't a string — is
// the literal name, like vm/eval_expr.go's FunctionReference case.
func (vm *VM) resolveDynamicName(name string, locals *frame) string {
	if len(name) > len(vm.VarQuoteOpen)+len(vm.VarQuoteClose) && strings.HasPrefix(name, vm.VarQuoteOpen) && strings.HasSuffix(name, vm.VarQuoteClose) {
		idName := name[len(vm.VarQuoteOpen) : len(name)-len(vm.VarQuoteClose)]
		if v, ok := vm.lookup(locals, symbol.Intern(idName)); ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return name
}

// hasEqualsMethod reports whether obj's own class (not its ancestors —
// vm/eval_expr.go scans only cls.Body) defines the equality magic method.
func (vm *VM) hasEqualsMethod(obj *Object) bool {
	cls, ok := vm.program.Classes[obj.ClassName]
	if !ok {
		return false
	}
	_, ok = cls.Methods[vm.EqualsMethod]
	return ok
}

// callEqualsMethod runs obj's equality magic method against right, mirroring
// vm/eval_expr.go's BinaryExpression case: the first parameter is bound
// directly (no arity check), a returned bool is negated for `!=`, and a body
// that ends without returning means "not equal" (so `!=` is true).
func (vm *VM) callEqualsMethod(obj *Object, right interface{}, negate bool) (interface{}, error) {
	fn := vm.program.Classes[obj.ClassName].Methods[vm.EqualsMethod]
	fr := newFrame()
	fr.module = vm.moduleFrame(fn.Module)
	fr.this = obj
	fr.selfClass = obj.ClassName
	if len(fn.Params) > 0 {
		fr.put(fn.ParamSymbols()[0], right)
	}
	res, err := vm.exec(fn.BodyChunk, fr)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return negate, nil
	}
	if b, ok := res.(bool); ok && negate {
		return !b, nil
	}
	return res, nil
}

// lookupOrClass is lookup, then — if no variable has that name — a class of
// that name as a class-ref, so a class name can be used as a value
// (`'자동차'의 '총생산량'`). The variable wins over the class, matching
// vm/eval_expr.go's Identifier case.
func (vm *VM) lookupOrClass(locals *frame, sym symbol.Symbol) (interface{}, bool) {
	if v, ok := vm.lookup(locals, sym); ok {
		return v, true
	}
	if name := sym.String(); vm.program.Classes[name] != nil {
		return &classRef{ClassName: name}, true
	}
	return nil, false
}

// declaringFrame is the frame a new variable goes into: the call's locals, or the
// globals at the top level (where locals is nil).
func (vm *VM) declaringFrame(locals *frame) *frame {
	if locals != nil {
		return locals
	}
	return vm.globals
}

// assignOrDeclare mirrors vm.Environment's VariableDeclaration rule
// (Runtime spec 2.2): reassign wherever the name is already bound, else
// declare it fresh in the current scope (locals if inside a function call,
// globals at the top level).
//
// A name declared with isConst (고정하자) is remembered on the frame that
// holds it, and any later reassignment of that binding — through SET_VAR
// too, so plain 정하자 is covered — is a
// ConstantAssignmentError, exactly like vm.Environment.Assign.
func (vm *VM) assignOrDeclare(locals *frame, sym symbol.Symbol, val interface{}, isConst bool, annotation string) error {
	if annotation != "" {
		if err := typecheck.Check(&vm.Types, annotation, sym.String(), val, vm.host()); err != nil {
			return err
		}
	}
	var owner *frame
	idx := -1
	if locals != nil {
		if i := locals.find(sym); i >= 0 {
			owner, idx = locals, i
		}
	}
	if owner == nil {
		if g := vm.globalsFor(locals); g != nil {
			if i := g.find(sym); i >= 0 {
				owner, idx = g, i
			}
		}
	}
	if owner != nil {
		s := &owner.slots[idx]
		// The type an earlier declaration gave the variable keeps constraining it.
		if s.typ != "" && s.typ != annotation {
			if err := typecheck.Check(&vm.Types, s.typ, sym.String(), val, vm.host()); err != nil {
				return err
			}
		}
		if s.isConst {
			return errs.New(errs.ConstantAssignment, sym.String())
		}
		s.val = val
		return nil
	}
	target := vm.globals
	if locals != nil {
		target = locals
	}
	s := target.put(sym, val)
	s.typ = annotation
	s.isConst = isConst
	return nil
}

// requireMutable refuses to change the list a constant variable holds.
func (vm *VM) requireMutable(locals *frame, sym symbol.Symbol) error {
	if locals != nil {
		if i := locals.find(sym); i >= 0 {
			if locals.slots[i].isConst {
				return errs.New(errs.ConstantAssignment, sym.String())
			}
			return nil
		}
	}
	if g := vm.globalsFor(locals); g != nil {
		if i := g.find(sym); i >= 0 && g.slots[i].isConst {
			return errs.New(errs.ConstantAssignment, sym.String())
		}
	}
	return nil
}

// checkListVar tests the value just pushed on list against the type the variable it is
// kept in was declared with: only that value, because the rest of the list fitted before.
func (vm *VM) checkListVar(locals *frame, sym symbol.Symbol, list *value.List, change bytecode.ListChange) error {
	var s *varSlot
	if locals != nil {
		if i := locals.find(sym); i >= 0 {
			s = &locals.slots[i]
		}
	}
	if s == nil {
		if g := vm.globalsFor(locals); g != nil {
			if i := g.find(sym); i >= 0 {
				s = &g.slots[i]
			}
		}
	}
	if s == nil || s.typ == "" {
		return nil
	}
	return typecheck.CheckAppended(&vm.Types, s.typ, sym.String(), list, change == bytecode.ListPushedFront, vm.host())
}

// checkListField is checkListVar for a list kept in an object's field.
func (vm *VM) checkListField(obj interface{}, prop string, list *value.List, change bytecode.ListChange) error {
	o, ok := obj.(*Object)
	if !ok {
		return nil
	}
	if annotation := vm.fieldType(o.ClassName, prop); annotation != "" {
		return typecheck.CheckAppended(&vm.Types, annotation, prop, list, change == bytecode.ListPushedFront, vm.host())
	}
	return nil
}

// binaryOp implements ADD/SUB/MUL/DIV/MOD/EQ/NEQ/LT/LTE/GT/GTE exactly like
// vm/eval_expr.go's BinaryExpression case: numeric when both operands are
// float64, string concatenation for "+" otherwise, reference equality for
// ==/!=, and DivideByZeroError for / and % by zero (the tree-walker only
// actually checks this for `/`; `%` by zero panics there — this VM checks
// both, since Runtime spec 5.3 requires it for both).
func (vm *VM) binaryOp(op bytecode.Opcode, left, right interface{}) (interface{}, error) {
	switch op {
	case bytecode.EQ:
		return value.Equal(left, right), nil
	case bytecode.NEQ:
		return !value.Equal(left, right), nil
	}

	symbol := operatorSymbol(op)
	// Null-safe (Runtime spec 2.4): only the equality operators may see 비어있음.
	if left == nil || right == nil {
		return nil, errs.New(errs.NullOperand, symbol)
	}

	leftNum, leftIsNum := left.(float64)
	rightNum, rightIsNum := right.(float64)
	if leftIsNum && rightIsNum {
		switch op {
		case bytecode.ADD:
			return num.Box(leftNum + rightNum), nil
		case bytecode.SUB:
			return num.Box(leftNum - rightNum), nil
		case bytecode.MUL:
			return num.Box(leftNum * rightNum), nil
		case bytecode.DIV:
			if rightNum == 0 {
				return nil, errs.New(errs.DivideByZero)
			}
			return num.Box(leftNum / rightNum), nil
		case bytecode.MOD:
			if int64(rightNum) == 0 {
				return nil, errs.New(errs.DivideByZero)
			}
			return num.Box(float64(int64(leftNum) % int64(rightNum))), nil
		case bytecode.LT:
			return leftNum < rightNum, nil
		case bytecode.LTE:
			return leftNum <= rightNum, nil
		case bytecode.GT:
			return leftNum > rightNum, nil
		case bytecode.GTE:
			return leftNum >= rightNum, nil
		}
	}

	// String addition only joins strings: no implicit conversion (Runtime spec 2.2).
	if op == bytecode.ADD {
		if ls, ok := left.(string); ok {
			if rs, ok := right.(string); ok {
				return strcat.Join(ls, rs), nil
			}
		}
	}
	return nil, errs.New(errs.OperandTypeMismatch, symbol,
		typecheck.Describe(&vm.Types, left, vm.host()), typecheck.Describe(&vm.Types, right, vm.host()))
}

func operatorSymbol(op bytecode.Opcode) string {
	switch op {
	case bytecode.ADD:
		return "+"
	case bytecode.SUB:
		return "-"
	case bytecode.MUL:
		return "*"
	case bytecode.DIV:
		return "/"
	case bytecode.MOD:
		return "%"
	case bytecode.LT:
		return "<"
	case bytecode.LTE:
		return "<="
	case bytecode.GT:
		return ">"
	case bytecode.GTE:
		return ">="
	}
	return "?"
}

// FormatValue is formatValue exported for embedders (bcstdlib's 문자로).
// FormatValue mirrors vm.Interpreter.FormatValue.
func (vm *VM) FormatValue(val interface{}) string {
	if val == nil {
		return vm.NullString
	}
	switch v := val.(type) {
	case string:
		return v
	case bool:
		if v {
			return vm.TrueString
		}
		return vm.FalseString
	case *value.List:
		if vm.formatDepth > maxFormatDepth {
			return "[...]"
		}
		vm.formatDepth++
		elems := make([]string, len(v.Items))
		for i, el := range v.Items {
			elems[i] = vm.FormatValue(el)
		}
		vm.formatDepth--
		return "[" + strings.Join(elems, ", ") + "]"
	case map[interface{}]interface{}:
		// {키: 값, ...} — Go maps have no insertion order, so entries are
		// sorted by their displayed key to keep the output deterministic
		// (the TypeScript engines sort the same way).
		entries := make([]string, 0, len(v))
		for k, el := range v {
			entries = append(entries, vm.FormatValue(k)+": "+vm.FormatValue(el))
		}
		sort.Strings(entries)
		return "{" + strings.Join(entries, ", ") + "}"
	case float64:
		// See vm.Interpreter.FormatValue: 'f'/-1 avoids scientific notation
		// for large magnitudes (e.g. 3628800 -> "3.6288e+06" with %v/%g).
		return strconv.FormatFloat(v, 'f', -1, 64)
	case *Object:
		return fmt.Sprintf(vm.ObjectFormat, v.ClassName)
	}
	return fmt.Sprintf("%v", val)
}
