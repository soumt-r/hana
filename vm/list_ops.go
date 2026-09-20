package vm

import "github.com/soumt-r/hana/ast"

// popFromList removes and returns the front or back element of list,
// returning the popped value and the resulting (shorter) list. Callers must
// ensure list is non-empty.
func popFromList(list []interface{}, position string) (popped interface{}, rest []interface{}) {
	if position == "front" {
		return list[0], list[1:]
	}
	return list[len(list)-1], list[:len(list)-1]
}

// assignListBack writes newList back to wherever target refers to (a plain
// variable or an object property) after a list push/pop. Go's []interface{}
// isn't a stable reference across append/slice operations, so mutating list
// statements/expressions have to re-assign the resulting slice back to its
// original binding by hand instead of mutating in place.
func (i *Interpreter) assignListBack(target ast.Expression, newList []interface{}, env *Environment) error {
	if id, ok := target.(*ast.Identifier); ok {
		if err := i.checkDeclaredType(env, id, newList); err != nil {
			return err
		}
		_, err := env.AssignSym(id.Symbol(), newList)
		return err
	}
	if mem, ok := target.(*ast.MemberExpression); ok {
		obj, err := i.Evaluate(mem.Object, env)
		if err != nil {
			return err
		}
		if hajaObj, ok := obj.(*HajaObject); ok {
			if propId, ok := mem.Property.(*ast.Identifier); ok {
				if err := i.checkField(hajaObj, propId.Value, newList); err != nil {
					return err
				}
				hajaObj.Props[propId.Value] = newList
			}
		}
	}
	return nil
}
