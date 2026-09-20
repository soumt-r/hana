package vm

import (
	"github.com/soumt-r/hana/errs"

	"github.com/soumt-r/hana/ast"
)

// bindParams binds call arguments to a function/method/constructor's declared
// parameters into env. This implements Runtime spec 1.2:
//   - a missing argument whose parameter has a `default` expression gets that
//     default (evaluated in env, so it can see earlier-bound parameters and
//     the caller's scope);
//   - a missing argument with no default is a hard MissingArgumentError,
//     never a silently-unset variable;
//   - more arguments than declared parameters is a hard ArgumentError,
//     never silently-dropped extras.
func (i *Interpreter) bindParams(params []*ast.Parameter, args []interface{}, env *Environment) error {
	if len(args) > len(params) {
		return errs.New(errs.TooManyArguments, len(params), len(args))
	}
	for idx, param := range params {
		if idx < len(args) {
			if err := i.declareParam(env, param, args[idx]); err != nil {
				return err
			}
			continue
		}
		if param.Default != nil {
			val, err := i.Evaluate(param.Default, env)
			if err != nil {
				return err
			}
			if err := i.declareParam(env, param, val); err != nil {
				return err
			}
			continue
		}
		return errs.New(errs.MissingArgument, param.Name.Value)
	}
	return nil
}
