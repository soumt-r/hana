package vm

import (
	"fmt"
	"github.com/soumt-r/hana/errs"
	"sort"
	"strconv"
	"strings"

	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/native"
)

type Interpreter struct {
	exec          native.ExecLock // held while hana code runs; see native_host.go
	ast           *ast.Program
	GlobalEnv     *Environment
	Classes       map[string]*ast.ClassDeclaration
	memberCache   map[memberKey]*classMember // see class_lookup.go
	scopePool     []*Environment             // see newScope in env.go
	Interfaces    map[string]*ast.InterfaceDeclaration
	Output        []string
	Config        LangConfig
	NativeModules map[string]NativeModule

	// Bootstrap installs the standard library into an interpreter. The stdlib
	// package sets it when it is registered, so a module loaded by import
	// (which runs in its own sub-interpreter) gets the same builtins and native
	// modules in its own language.
	Bootstrap func(*Interpreter)

	callDepth int // how many calls/constructions are in flight (see MaxCallDepth)

	// scope is the module whose code is running: the function names of that
	// module (its own functions and what it imported) are found before the
	// importer's. nil while the program's own code runs. See module_scope.go.
	scope *Interpreter

	// ReadLine supplies the next line for `입력받자` (the CLI wires it to stdin).
	// nil means no input source: the statement reads an empty line, so tests
	// and tools never block waiting for a terminal.
	ReadLine func() string
}

// MaxCallDepth bounds nested calls so runaway recursion becomes a catchable
// RecursionError instead of Go's fatal, uncatchable stack overflow. bcvm and
// the browser engines use the same number.
const MaxCallDepth = 10000

// newSubInterpreter creates the interpreter an imported module runs in: its
// own language config, plus the standard library if this interpreter has one.
func (i *Interpreter) newSubInterpreter(prog *ast.Program, cfg LangConfig) *Interpreter {
	sub := NewInterpreter(prog, cfg)
	if i.Bootstrap != nil {
		sub.Bootstrap = i.Bootstrap
		i.Bootstrap(sub)
	}
	return sub
}

func NewInterpreter(prog *ast.Program, config ...LangConfig) *Interpreter {
	cfg := KoreanConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	cfg.resolveSymbols()
	env := NewEnvironment(nil)

	interp := &Interpreter{
		ast:           prog,
		GlobalEnv:     env,
		Classes:       make(map[string]*ast.ClassDeclaration),
		Interfaces:    make(map[string]*ast.InterfaceDeclaration),
		Output:        []string{},
		Config:        cfg,
		NativeModules: make(map[string]NativeModule),
	}

	interp.Classes[cfg.BuiltinErrorClass] = newBuiltinErrorClass(cfg)
	return interp
}

// newBuiltinErrorClass는 모든 인터프리터 인스턴스에 기본으로 존재하는 [오류] 클래스를
// 만듭니다. AST 스펙 4.5: "발생된 에러 객체는 내장된 [오류] 클래스의 인스턴스이며,
// 기본적으로 '메시지' 속성을 가진다." 소스를 파싱하지 않고 AST를 직접 구성하는 이유는
// 인터프리터 초기화 시점에 파서 패키지에 의존하고 싶지 않기 때문입니다.
func newBuiltinErrorClass(cfg LangConfig) *ast.ClassDeclaration {
	self := func(prop string) *ast.MemberExpression {
		return &ast.MemberExpression{
			Object:   &ast.Identifier{Value: cfg.SelfWords[0]},
			Property: &ast.Identifier{Value: prop},
		}
	}

	return &ast.ClassDeclaration{
		Name: &ast.TypeReference{Name: cfg.BuiltinErrorClass},
		Body: []ast.Statement{
			&ast.VariableDeclaration{
				Name:           &ast.Identifier{Value: cfg.BuiltinErrorMessage},
				Value:          &ast.StringLiteral{Value: ""},
				AccessModifier: "public",
			},
			&ast.ConstructorDeclaration{
				Params: []*ast.Parameter{
					{Name: &ast.Identifier{Value: cfg.BuiltinErrorCtorArg}},
				},
				Body: []ast.Statement{
					&ast.Assignment{
						Target: self(cfg.BuiltinErrorMessage),
						Value:  &ast.Identifier{Value: cfg.BuiltinErrorCtorArg},
					},
				},
			},
			&ast.FunctionDeclaration{
				Name:           &ast.Identifier{Value: "__toString__"},
				AccessModifier: "public",
				Body: &ast.BlockStatement{
					Statements: []ast.Statement{
						&ast.ReturnStatement{Value: self(cfg.BuiltinErrorMessage)},
					},
				},
			},
		},
	}
}

func (i *Interpreter) RegisterBuiltin(name string, fn *BuiltinFunction) {
	i.GlobalEnv.Declare(name, fn)
}

func (i *Interpreter) RegisterNativeModule(name string, module NativeModule) {
	i.NativeModules[name] = module
}

func (i *Interpreter) Run() error {
	i.exec.Acquire()
	defer i.exec.Release()
	// 1. 선언부 수집
	for _, stmt := range i.ast.Statements {
		if cls, ok := stmt.(*ast.ClassDeclaration); ok {
			name := cls.Name.Name
			i.registerClass(name, cls)
		}
		if iface, ok := stmt.(*ast.InterfaceDeclaration); ok {
			name := iface.Name.Name
			i.Interfaces[name] = iface
		}
	}

	// 2. 인터페이스 검증 (Pre-flight Validation)
	//
	// [인터페이스]를 따르는 [클래스]를 설계하자로 선언한 인터페이스는 cls.Interfaces에
	// 담기는데, 예전 코드는 실수로 cls.BaseClass를 대신 확인하고 있었다 — 그 결과
	// 일반적인 인터페이스 구현 선언은 이 조건에 걸릴 일이 없어 검증이 사실상 항상
	// 통과했다(스펙 3.5가 약속하는 "즉시 검사되어 오류로 처리"가 실제로는 동작하지
	// 않았음). bytecode.Compiler.registerClass/vm.Run 비교 중 발견해 고쳤다.
	for clsName, cls := range i.Classes {
		for _, ifaceRef := range cls.Interfaces {
			iface, ok := i.Interfaces[ifaceRef.Name]
			if !ok {
				continue
			}
			// 클래스가 인터페이스의 모든 메서드를 구현했는지 확인
			for _, reqStmt := range iface.Body {
				if reqMeth, ok := reqStmt.(*ast.InterfaceMethod); ok {
					implemented := false
					for _, clsStmt := range cls.Body {
						if clsMeth, ok := clsStmt.(*ast.FunctionDeclaration); ok {
							if clsMeth.Name.Value == reqMeth.Name.Value {
								implemented = true
								break
							}
						}
					}
					if !implemented {
						return errs.New(errs.InterfaceNotImplemented, clsName, ifaceRef.Name, reqMeth.Name.Value)
					}
				}
			}
		}
	}

	// 3. 실행
	for _, stmt := range i.ast.Statements {
		switch s := stmt.(type) {
		case *ast.ClassDeclaration:
			// Execute static properties
			for _, clsStmt := range s.Body {
				if vdecl, ok := clsStmt.(*ast.VariableDeclaration); ok && vdecl.IsStatic {
					val, err := i.Evaluate(vdecl.Value, i.GlobalEnv)
					if err != nil {
						return err
					}
					i.GlobalEnv.Declare(s.Name.Name+"."+vdecl.Name.Value, val)
				} else if assign, ok := clsStmt.(*ast.Assignment); ok {
					if mem, ok := assign.Target.(*ast.MemberExpression); ok {
						if id, ok := mem.Object.(*ast.Identifier); ok && i.Config.IsPluralSelfWord(id.Value) {
							if propId, ok := mem.Property.(*ast.Identifier); ok {
								val, err := i.Evaluate(assign.Value, i.GlobalEnv)
								if err != nil {
									return err
								}
								i.GlobalEnv.Declare(s.Name.Name+"."+propId.Value, val)
							}
						}
					}
				}
			}
		case *ast.InterfaceDeclaration:
			// 스킵
		default:
			_, err := i.Execute(stmt, i.GlobalEnv)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (i *Interpreter) FormatValue(val interface{}) string {
	if val == nil {
		return i.Config.NullString
	}
	switch v := val.(type) {
	case string:
		return v
	case *HajaObject:
		return fmt.Sprintf(i.Config.ObjectFormat, v.ClassName)
	case bool:
		if v {
			return i.Config.TrueString
		}
		return i.Config.FalseString
	case []interface{}:
		elements := make([]string, len(v))
		for idx, el := range v {
			elements[idx] = i.FormatValue(el)
		}
		return "[" + strings.Join(elements, ", ") + "]"
	case map[interface{}]interface{}:
		// {키: 값, ...} — Go maps have no insertion order, so entries are
		// sorted by their displayed key to keep the output deterministic
		// (the TypeScript engines sort the same way).
		entries := make([]string, 0, len(v))
		for k, el := range v {
			entries = append(entries, i.FormatValue(k)+": "+i.FormatValue(el))
		}
		sort.Strings(entries)
		return "{" + strings.Join(entries, ", ") + "}"
	case float64:
		// %v/%g falls back to scientific notation for large magnitudes
		// (e.g. 3628800 -> "3.6288e+06"); 'f' with precision -1 prints the
		// shortest exact decimal instead, which is what users expect.
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return fmt.Sprintf("%v", val)
}

type ReturnValue struct {
	Value interface{}
}

func (r *ReturnValue) Error() string { return "return" }

type BreakValue struct{}

func (b *BreakValue) Error() string { return "break" }

type ThrownError struct {
	Value interface{}
}

// Error is what the CLI prints for an uncaught throw: the message of a thrown
// error object (its 메시지/メッセージ property), else the value as text.
func (t *ThrownError) Error() string {
	if obj, ok := t.Value.(*HajaObject); ok {
		for _, key := range []string{"메시지", "メッセージ"} {
			if msg, ok := obj.Props[key].(string); ok {
				return msg
			}
		}
	}
	return fmt.Sprintf("%v", t.Value)
}
