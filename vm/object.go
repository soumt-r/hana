package vm

import (
	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/symbol"
)

type HajaObject struct {
	ClassName string
	Props     map[string]interface{}
	class     *ast.ClassDeclaration // ClassName's declaration, found on first use (see classOf)
}

func NewHajaObject(className string) *HajaObject {
	return &HajaObject{
		ClassName: className,
		Props:     make(map[string]interface{}),
	}
}

type BoundMethod struct {
	Object   *HajaObject
	FuncName string
	Sym      symbol.Symbol // FuncName as a Symbol
	IsSuper  bool
}

type BuiltinFunction struct {
	Name string
	Fn   func(i *Interpreter, env *Environment, args ...interface{}) (interface{}, error)
}

type BoundStringMethod struct {
	Value    string
	FuncName string
}

type BoundListMethod struct {
	List     []interface{}
	FuncName string
	// Target is the MemberExpression's Object expression the list came
	// from (a plain variable or an object field) — needed because a
	// mutating list method (비우기) can't mutate List in place (Go slices
	// aren't stable references the way a JS array is) and has to write the
	// result back to wherever it came from instead, same as
	// ListPushStatement/ListPopStatement already do via assignListBack.
	Target ast.Expression
}

type BoundStaticMethod struct {
	ClassName string
	FuncName  string
}

type SuperReference struct {
	Object *HajaObject
}

// ClassReference는 클래스 자체를 값으로 취급할 때 씁니다 (정적 멤버 접근, instanceof 등).
type ClassReference struct {
	ClassName string
}

// NativeModule은 Go로 구현된 내장 모듈(예: [수학])이 노출하는 함수 테이블입니다.
type NativeModule map[string]*BuiltinFunction
