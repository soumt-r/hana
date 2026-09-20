package bytecode

import (
	"reflect"
	"strings"

	"github.com/soumt-r/hana/ast"
)

// A counting loop can keep its position in the variable the program names, which is
// faster than a hidden counter copied into it on every pass. That is only right while the
// body never assigns to the variable: the loop is meant to run one pass per number of the
// range whatever the body does to the variable of its own pass (the tree-walker and the
// browser engine keep the position apart). writesVariable says whether the body might
// assign it, counting every way to: a declaration or assignment, a target of input, of a
// list push or pop, of a list emptied by a method call, or a loop that names it.
// It answers true when in doubt (an identifier that merely contains the name, as in
// `바깥의 가`), which only costs the slower loop.
func writesVariable(body *ast.BlockStatement, name string) bool {
	return writes(reflect.ValueOf(body), name, map[uintptr]bool{})
}

func writes(v reflect.Value, name string, seen map[uintptr]bool) bool {
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return false
		}
		return writes(v.Elem(), name, seen)
	case reflect.Ptr:
		if v.IsNil() || seen[v.Pointer()] {
			return false
		}
		seen[v.Pointer()] = true
		switch n := v.Interface().(type) {
		case *ast.VariableDeclaration:
			if n.Name != nil && strings.Contains(n.Name.Value, name) {
				return true
			}
		case *ast.Assignment:
			if mentions(n.Target, name) {
				return true
			}
		case *ast.InputStatement:
			if n.Target != nil && strings.Contains(n.Target.Value, name) {
				return true
			}
		case *ast.ListPushStatement:
			if mentions(n.Target, name) {
				return true
			}
		case *ast.ListPopStatement:
			if mentions(n.Target, name) {
				return true
			}
		case *ast.ListPopExpression:
			if mentions(n.Target, name) {
				return true
			}
		case *ast.CallExpression:
			if mem, ok := n.Callee.(*ast.MemberExpression); ok && mentions(mem.Object, name) {
				return true
			}
		case *ast.ForEachLoop:
			if mentions(n.List, name) {
				return true
			}
		case *ast.ForRangeStatement:
			if strings.Contains(n.LoopVar, name) {
				return true
			}
		}
		return writes(v.Elem(), name, seen)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() && writes(v.Field(i), name, seen) {
				return true
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if writes(v.Index(i), name, seen) {
				return true
			}
		}
	}
	return false
}

// mentions reports whether an identifier in the expression contains name.
func mentions(e ast.Expression, name string) bool {
	return mentionsValue(reflect.ValueOf(e), name, map[uintptr]bool{})
}

func mentionsValue(v reflect.Value, name string, seen map[uintptr]bool) bool {
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return false
		}
		return mentionsValue(v.Elem(), name, seen)
	case reflect.Ptr:
		if v.IsNil() || seen[v.Pointer()] {
			return false
		}
		seen[v.Pointer()] = true
		if id, ok := v.Interface().(*ast.Identifier); ok {
			return strings.Contains(id.Value, name)
		}
		return mentionsValue(v.Elem(), name, seen)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() && mentionsValue(v.Field(i), name, seen) {
				return true
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if mentionsValue(v.Index(i), name, seen) {
				return true
			}
		}
	}
	return false
}
