package vm

import "github.com/soumt-r/hana/ast"

// listChange is how a list being written back differs from the one it replaces.
type listChange int

const (
	listShrunk      listChange = iota // elements were taken away (or all of them): still fits its type
	listPushedBack                    // one element was added at the back
	listPushedFront                   // one element was added at the front
)

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
//
// change says how the list differs from the one it replaces, because that decides how
// much of it has to be checked against the type it was declared with: a list that only
// got shorter still fits, and one that got a value at an end needs only that value
// tested (the rest fitted before). Checking every element on every push made building
// a list of n values take time proportional to n squared.
func (i *Interpreter) assignListBack(target ast.Expression, newList []interface{}, env *Environment, change listChange) error {
	if id, ok := target.(*ast.Identifier); ok {
		if err := i.checkListWrite(env, id, newList, change); err != nil {
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
				if err := i.checkFieldListWrite(hajaObj, propId.Value, newList, change); err != nil {
					return err
				}
				hajaObj.Props[propId.Value] = newList
			}
		}
	}
	return nil
}
