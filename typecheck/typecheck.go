// Package typecheck is hana's dynamic type checker (Runtime spec 2.2): does a
// runtime value satisfy a declared type such as [숫자], [(숫자)목록] or [자동차]?
// It is engine-neutral — the tree-walker and the bytecode VM both hand it plain
// Go values (float64, string, bool, []interface{}, map[interface{}]interface{},
// nil) and, for user classes, a Host that knows the class hierarchy. The browser
// engines mirror the same rules by hand (compare_tests.ts keeps them honest).
package typecheck

import (
	"strings"

	"github.com/soumt-r/hana/errs"
)

// Names are a language's names for the built-in types (하자: 숫자, 문자열, 논리,
// 아무거나, 목록, 사전, 비어있음; 카나데: 数字, 文字列, 論理, 何でも, リスト, 辞書, 空っぽ).
type Names struct {
	Number, String, Boolean, Any, List, Dict, Null string
}

// Spec is a parsed type annotation: a base name and, for generics like
// `(문자열, 숫자)사전`, its type arguments.
type Spec struct {
	Name string
	Args []string
}

// Parse reads an annotation as the parser stores it: "숫자", "(숫자)목록",
// "(문자열, 숫자)사전". The empty string (no annotation) is the Any type.
func Parse(annotation string) Spec {
	annotation = strings.TrimSpace(annotation)
	if strings.HasPrefix(annotation, "(") {
		if end := strings.Index(annotation, ")"); end > 0 {
			var args []string
			for _, a := range strings.Split(annotation[1:end], ",") {
				if a = strings.TrimSpace(a); a != "" {
					args = append(args, a)
				}
			}
			return Spec{Name: strings.TrimSpace(annotation[end+1:]), Args: args}
		}
	}
	return Spec{Name: annotation}
}

// Host answers the questions only an engine can: which class an object is an
// instance of, and whether one class or interface is (or extends/implements)
// another.
type Host interface {
	ClassOf(v interface{}) (class string, isObject bool)
	IsSubtype(class, target string) bool
}

// Accepts reports whether v satisfies s. Null (비어있음) is accepted by every type
// (spec 2.2: Nullable by default), and an omitted or [아무거나] type accepts anything.
func (s Spec) Accepts(n Names, v interface{}, h Host) bool {
	if v == nil || s.Name == "" || s.Name == n.Any {
		return true
	}
	switch s.Name {
	case n.Number:
		_, ok := v.(float64)
		return ok
	case n.String:
		_, ok := v.(string)
		return ok
	case n.Boolean:
		_, ok := v.(bool)
		return ok
	case n.Null:
		return false // v is not null here
	case n.List:
		list, ok := v.([]interface{})
		if !ok {
			return false
		}
		if len(s.Args) > 0 {
			elem := Spec{Name: s.Args[0]}
			for _, e := range list {
				if !elem.Accepts(n, e, h) {
					return false
				}
			}
		}
		return true
	case n.Dict:
		dict, ok := v.(map[interface{}]interface{})
		if !ok {
			return false
		}
		if len(s.Args) > 0 {
			keySpec, valSpec := Spec{}, Spec{}
			if len(s.Args) == 1 {
				valSpec = Spec{Name: s.Args[0]}
			} else {
				keySpec, valSpec = Spec{Name: s.Args[0]}, Spec{Name: s.Args[1]}
			}
			for k, e := range dict {
				if !keySpec.Accepts(n, k, h) || !valSpec.Accepts(n, e, h) {
					return false
				}
			}
		}
		return true
	}
	if h == nil {
		return false
	}
	class, isObject := h.ClassOf(v)
	return isObject && (class == s.Name || h.IsSubtype(class, s.Name))
}

// Describe names v's type for an error message, in the language's own words.
func Describe(n Names, v interface{}, h Host) string {
	switch t := v.(type) {
	case nil:
		return n.Null
	case float64:
		return n.Number
	case string:
		return n.String
	case bool:
		return n.Boolean
	case []interface{}:
		return n.List
	case map[interface{}]interface{}:
		return n.Dict
	default:
		if h != nil {
			if class, ok := h.ClassOf(t); ok {
				return class
			}
		}
	}
	return "?"
}

// quickAccepts answers the overwhelmingly common cases (a null, or a plain
// number/string/boolean against its own type name) without parsing the
// annotation. False only means "ask Accepts".
func quickAccepts(n Names, annotation string, v interface{}) bool {
	switch v.(type) {
	case nil:
		return true
	case float64:
		return annotation == n.Number
	case string:
		return annotation == n.String
	case bool:
		return annotation == n.Boolean
	}
	return false
}

// Check is Accepts as an error: a VariableTypeMismatch naming the variable (or
// property), the declared type as written, and what actually arrived.
func Check(n Names, annotation, name string, v interface{}, h Host) error {
	if quickAccepts(n, annotation, v) || Parse(annotation).Accepts(n, v, h) {
		return nil
	}
	return errs.New(errs.VariableTypeMismatch, name, annotation, Describe(n, v, h))
}

// CheckReturn is Check for the value a function returns; a function that ends
// without returning anything returns null, which every type accepts.
func CheckReturn(n Names, annotation, function string, v interface{}, h Host) error {
	if quickAccepts(n, annotation, v) || Parse(annotation).Accepts(n, v, h) {
		return nil
	}
	return errs.New(errs.ReturnTypeMismatch, function, annotation, Describe(n, v, h))
}

// CheckArgument is Check for a function parameter.
func CheckArgument(n Names, annotation, name string, v interface{}, h Host) error {
	if quickAccepts(n, annotation, v) || Parse(annotation).Accepts(n, v, h) {
		return nil
	}
	return errs.New(errs.ArgumentTypeMismatch, name, annotation, Describe(n, v, h))
}
