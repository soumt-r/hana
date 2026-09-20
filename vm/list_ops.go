package vm

import (
	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/value"
)

// maxFormatDepth stops FormatValue on a list that contains itself.
const maxFormatDepth = 100

// requireMutable refuses to change the list a constant (고정하자) variable holds: Runtime
// spec 2.1 counts a push, a pop and an emptying among the operations that change a
// constant.
func (i *Interpreter) requireMutable(target ast.Expression, env *Environment) error {
	if id, ok := target.(*ast.Identifier); ok && env.isConst(id.Symbol()) {
		return errs.New(errs.ConstantAssignment, id.Value)
	}
	return nil
}

// checkListPush tests the value that was just pushed on list (at its front or back)
// against the type declared for where the list is kept: a variable or an object field.
// Only that value needs testing, since the rest fitted before; checking every element on
// every push made building a list of n values take time proportional to n squared.
// The caller takes the value back off when this fails.
func (i *Interpreter) checkListPush(target ast.Expression, list *value.List, env *Environment, front bool) error {
	if id, ok := target.(*ast.Identifier); ok {
		return i.checkListWrite(env, id, list, front)
	}
	if mem, ok := target.(*ast.MemberExpression); ok {
		obj, err := i.Evaluate(mem.Object, env)
		if err != nil {
			return err
		}
		if hajaObj, ok := obj.(*HajaObject); ok {
			if propId, ok := mem.Property.(*ast.Identifier); ok {
				return i.checkFieldListWrite(hajaObj, propId.Value, list, front)
			}
		}
	}
	return nil
}
