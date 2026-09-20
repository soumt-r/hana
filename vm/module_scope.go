package vm

import (
	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/typecheck"
)

// Module scope: code that was imported keeps the scope of the module it came
// from. An imported function runs on the importing interpreter (with the
// importer's variables), but while its body runs, the names of functions are looked
// up in its own module first — the module's other functions, and what the module
// itself imported — so a package can use its helpers and the packages it depends
// on without the importer bringing them in. Each imported declaration remembers
// its module (Module on the AST node); i.scope is the module of the body that runs.
//
// Classes, interfaces and helper functions of a module that were not asked for
// are not visible to the importer by name, except that a class or interface the
// importer does not already have is registered under its own name, so the
// module's functions can create it.

// markModule records sub as the module of everything prog declares.
func markModule(prog *ast.Program, sub *Interpreter) {
	for _, stmt := range prog.Statements {
		switch d := stmt.(type) {
		case *ast.FunctionDeclaration:
			d.Module = sub
		case *ast.ClassDeclaration:
			d.Module = sub
			for _, member := range d.Body {
				switch m := member.(type) {
				case *ast.FunctionDeclaration:
					m.Module = sub
				case *ast.ConstructorDeclaration:
					m.Module = sub
				}
			}
		}
	}
}

// enterModule makes m (the Module of a declaration; nil for the program's own
// code) the module in scope and returns the one that was, to be put back in
// i.scope when the code is done.
func (i *Interpreter) enterModule(m interface{}) *Interpreter {
	prev := i.scope
	i.scope, _ = m.(*Interpreter)
	return prev
}

// scopedFunction finds a function called name in the module whose code is
// running: a function it declares, or one it imported (a function or a native
// library function bound in its own environment).
func (i *Interpreter) scopedFunction(name string) (interface{}, bool) {
	m := i.scope
	if m == nil {
		return nil, false
	}
	for _, stmt := range m.ast.Statements {
		if f, ok := stmt.(*ast.FunctionDeclaration); ok && f.Name.Value == name {
			return f, true
		}
	}
	if v, ok := m.GlobalEnv.Get(name); ok {
		switch v.(type) {
		case *ast.FunctionDeclaration, *BuiltinFunction:
			return v, true
		}
	}
	return nil, false
}

// bringModuleTypes registers the classes and interfaces a module declares under
// their own names when the importer has none of that name, so the module's
// functions can create them and check against them.
func (i *Interpreter) bringModuleTypes(sub *Interpreter, prog *ast.Program) {
	for _, stmt := range prog.Statements {
		switch d := stmt.(type) {
		case *ast.ClassDeclaration:
			if _, exists := i.Classes[d.Name.Name]; !exists {
				if cls, ok := sub.Classes[d.Name.Name]; ok {
					i.registerClass(d.Name.Name, cls)
				}
			}
		case *ast.InterfaceDeclaration:
			if _, exists := i.Interfaces[d.Name.Name]; !exists {
				if iface, ok := sub.Interfaces[d.Name.Name]; ok {
					i.Interfaces[d.Name.Name] = iface
				}
			}
		}
	}
}

// initFields evaluates the field initializers of a new object, in the scope of
// the module its class came from.
func (i *Interpreter) initFields(cls *ast.ClassDeclaration, obj *HajaObject, env *Environment) error {
	prev := i.enterModule(cls.Module)
	defer func() { i.scope = prev }()
	for _, stmt := range cls.Body {
		if vdecl, ok := stmt.(*ast.VariableDeclaration); ok && !vdecl.IsStatic {
			val, err := i.Evaluate(vdecl.Value, env)
			if err != nil {
				return err
			}
			if vdecl.TypeRef != nil {
				if err := typecheck.Check(i.Config.Types, vdecl.TypeRef.Name, vdecl.Name.Value, val, i.host()); err != nil {
					return err
				}
			}
			obj.Props[vdecl.Name.Value] = val
		} else if assign, ok := stmt.(*ast.Assignment); ok {
			if id, ok := assign.Target.(*ast.Identifier); ok {
				val, err := i.Evaluate(assign.Value, env)
				if err != nil {
					return err
				}
				obj.Props[id.Value] = val
			}
		}
	}
	return nil
}

// runConstructor binds the arguments and runs a constructor's body, in the scope
// of the module the constructor came from.
func (i *Interpreter) runConstructor(ctor *ast.ConstructorDeclaration, obj *HajaObject, clsName string, args []interface{}) error {
	ctorEnv := NewEnvironment(i.GlobalEnv)
	ctorEnv.this = obj
	ctorEnv.DeclareSym(selfClassSym, clsName)
	if err := i.bindParams(ctor.Params, args, ctorEnv); err != nil {
		return err
	}
	prev := i.enterModule(ctor.Module)
	defer func() { i.scope = prev }()
	for _, bs := range ctor.Body {
		_, err := i.Execute(bs, ctorEnv)
		if err != nil {
			if _, ok := err.(*ReturnValue); ok {
				break
			}
			return err
		}
	}
	return nil
}
