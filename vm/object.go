package vm

import (
	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/symbol"
	"github.com/soumt-r/hana/value"
)

type HariObject struct {
	ClassName string
	Props     map[string]interface{}
	class     *ast.ClassDeclaration // ClassName's declaration, found on first use (see classOf)
}

func NewHariObject(className string) *HariObject {
	return &HariObject{
		ClassName: className,
		Props:     make(map[string]interface{}),
	}
}

type BoundMethod struct {
	Object   *HariObject
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
	List     *value.List
	FuncName string
	// Target is the expression the list came from; a list held by a constant
	// variable cannot be emptied.
	Target ast.Expression
}

type BoundStaticMethod struct {
	ClassName string
	FuncName  string
}

type SuperReference struct {
	Object *HariObject
}

// ClassReference는 클래스 자체를 값으로 취급할 때 씁니다 (정적 멤버 접근, instanceof 등).
type ClassReference struct {
	ClassName string
}

// NativeModule은 Go로 구현된 내장 모듈(예: [수학])이 노출하는 함수 테이블입니다.
type NativeModule map[string]*BuiltinFunction
