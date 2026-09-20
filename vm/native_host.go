package vm

import (
	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/native"
	"github.com/soumt-r/hana/typecheck"
)

// The Interpreter is a native.Host: native libraries call back into the program
// through it (see the native package for the ABI and the rules).

// Call implements native.Host: run a function value on this interpreter, waiting
// until the program is not executing.
func (i *Interpreter) Call(fn interface{}, args []interface{}) (interface{}, error) {
	i.exec.Acquire()
	defer i.exec.Release()
	return i.CallFunction(fn, args)
}

// Identify implements native.Host.
func (i *Interpreter) Identify(v interface{}) (interface{}, bool) {
	switch fn := v.(type) {
	case *ast.FunctionDeclaration, *BuiltinFunction:
		return fn, true
	case *BoundMethod:
		return struct {
			obj  *HajaObject
			name string
		}{fn.Object, fn.FuncName}, true
	}
	return nil, false
}

// BeginNative implements native.Host: let callbacks run while a native function
// works. Outside Run and Call (nothing holds the lock) it does nothing.
func (i *Interpreter) BeginNative() func() { return i.exec.Begin() }

// CallFunction runs a hana function value (a declared function, a bound
// method, or a builtin) with already-evaluated arguments.
func (i *Interpreter) CallFunction(callee interface{}, args []interface{}) (interface{}, error) {
	switch fn := callee.(type) {
	case *BuiltinFunction:
		return fn.Fn(i, i.GlobalEnv, args...)
	case *ast.FunctionDeclaration:
		return i.runFunctionBody(fn, args, NewEnvironment(i.globalOf(fn.Module)))
	case *BoundMethod:
		cls, ok := i.Classes[fn.Object.ClassName]
		if !ok {
			return nil, errs.New(errs.ClassNotFound, fn.Object.ClassName)
		}
		method := i.classMember(cls, fn.Sym).method
		if method == nil {
			return nil, errs.New(errs.MethodNotFound, fn.FuncName)
		}
		env := NewEnvironment(i.globalOf(method.Module))
		env.this = fn.Object
		env.DeclareSym(thisSym, fn.Object)
		env.DeclareSym(selfClassSym, fn.Object.ClassName)
		return i.runFunctionBody(method, args, env)
	}
	return nil, errs.New(errs.NotCallable)
}

// runFunctionBody runs a function's body and returns its value, which has to fit
// the type the function declared it returns (if it declared one).
func (i *Interpreter) runFunctionBody(fn *ast.FunctionDeclaration, args []interface{}, env *Environment) (interface{}, error) {
	if err := i.bindParams(fn.Params, args, env); err != nil {
		return nil, err
	}
	prev := i.enterModule(fn.Module)
	result, err := i.runStatements(fn.Body.Statements, env)
	i.scope = prev
	if err != nil || fn.ReturnType == nil {
		return result, err
	}
	if err := typecheck.CheckReturn(&i.Config.Types, fn.ReturnType.Name, fn.Name.Value, result, i.host()); err != nil {
		return nil, err
	}
	return result, nil
}

// runStatements runs a body; what it returns is the value of its 돌려주자, or null.
func (i *Interpreter) runStatements(statements []ast.Statement, env *Environment) (interface{}, error) {
	for _, stmt := range statements {
		if _, err := i.Execute(stmt, env); err != nil {
			if ret, ok := err.(*ReturnValue); ok {
				return ret.Value, nil
			}
			return nil, err
		}
	}
	return nil, nil
}

var _ native.Host = (*Interpreter)(nil)
