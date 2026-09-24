package vm

import (
	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/errs"
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

// globalOf is the outermost environment of code that came from module m: the module's
// own, where its top-level variables live, so an imported function sees them (and not
// the importer's). Code of the program itself (nil) has the interpreter's own.
func (i *Interpreter) globalOf(m interface{}) *Environment {
	if mod, ok := m.(*Interpreter); ok {
		return mod.GlobalEnv
	}
	return i.GlobalEnv
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
// funcIndex finds the functions a program declares at its top level by name; the
// linear scan it replaces ran on every call.
type funcIndex struct {
	prog   *ast.Program
	count  int
	byName map[string]*ast.FunctionDeclaration
}

// topFunction is the first top-level function declaration of that name.
func (i *Interpreter) topFunction(name string) *ast.FunctionDeclaration {
	idx := i.funcIndex
	if idx == nil || idx.prog != i.ast || idx.count != len(i.ast.Statements) {
		idx = &funcIndex{prog: i.ast, count: len(i.ast.Statements), byName: map[string]*ast.FunctionDeclaration{}}
		for _, stmt := range i.ast.Statements {
			if f, ok := stmt.(*ast.FunctionDeclaration); ok {
				if _, dup := idx.byName[f.Name.Value]; !dup {
					idx.byName[f.Name.Value] = f
				}
			}
		}
		i.funcIndex = idx
	}
	return idx.byName[name]
}

func (i *Interpreter) scopedFunction(name string) (interface{}, bool) {
	m := i.scope
	if m == nil {
		return nil, false
	}
	if f := m.topFunction(name); f != nil {
		return f, true
	}
	if v, ok := m.GlobalEnv.Get(name); ok {
		switch v.(type) {
		case *ast.FunctionDeclaration, *BuiltinFunction:
			return v, true
		}
	}
	return nil, false
}

// bringModuleTypes registers the classes and interfaces a module declares (and the ones
// it brought in itself) under their own names, so the module's functions can create
// them and check against them. moduleID names the module for messages and ownership.
// A name the importer already has for a class of another module is an error, except
// for one the import statement gives another name (<이름>을 <별칭>으로): that is how the
// program says which one it means.
func (i *Interpreter) bringModuleTypes(sub *Interpreter, moduleID string, s *ast.ImportStatement) error {
	aliased := map[string]bool{}
	for _, item := range s.Items {
		if item.As != "" {
			aliased[item.Name] = true
		}
	}
	owner := func(m *Interpreter, name string) string {
		if o, ok := m.classOwner[name]; ok {
			return o
		}
		return moduleID
	}
	conflict := func(name, mine, theirs string) error {
		if aliased[name] {
			return nil
		}
		if theirs == "" {
			return errs.New(errs.ImportClassConflictOwn, moduleID, name)
		}
		return errs.New(errs.ImportClassConflict, moduleID, name, theirs)
	}
	for name, cls := range sub.Classes {
		if name == sub.Config.BuiltinErrorClass || name == i.Config.BuiltinErrorClass {
			continue
		}
		mine := owner(sub, name)
		if _, exists := i.Classes[name]; exists {
			if theirs := i.classOwner[name]; theirs != mine {
				if err := conflict(name, mine, theirs); err != nil {
					return err
				}
			}
			continue
		}
		i.registerClass(name, cls)
		i.rememberOwner(name, mine)
	}
	for name, iface := range sub.Interfaces {
		mine := owner(sub, name)
		if _, exists := i.Interfaces[name]; exists {
			if theirs := i.classOwner[name]; theirs != mine {
				if err := conflict(name, mine, theirs); err != nil {
					return err
				}
			}
			continue
		}
		i.Interfaces[name] = iface
		i.rememberOwner(name, mine)
	}
	return nil
}

func (i *Interpreter) rememberOwner(name, owner string) {
	if i.classOwner == nil {
		i.classOwner = map[string]string{}
	}
	i.classOwner[name] = owner
}

// initFields evaluates the field initializers of a new object, in the scope of
// the module its class came from.
func (i *Interpreter) initFields(cls *ast.ClassDeclaration, obj *HariObject, env *Environment) error {
	prev := i.enterModule(cls.Module)
	defer func() { i.scope = prev }()
	for _, stmt := range cls.Body {
		if vdecl, ok := stmt.(*ast.VariableDeclaration); ok && !vdecl.IsStatic {
			val, err := i.Evaluate(vdecl.Value, env)
			if err != nil {
				return err
			}
			if vdecl.TypeRef != nil {
				if err := typecheck.Check(&i.Config.Types, vdecl.TypeRef.Name, vdecl.Name.Value, val, i.host()); err != nil {
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
func (i *Interpreter) runConstructor(ctor *ast.ConstructorDeclaration, obj *HariObject, clsName string, args []interface{}) error {
	ctorEnv := NewEnvironment(i.globalOf(ctor.Module))
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

// A loaded module: the interpreter that ran it, and its program.
type loadedModule struct {
	sub  *Interpreter
	prog *ast.Program
}

type moduleCache struct{ loaded map[string]*loadedModule }

func (i *Interpreter) moduleCache() *moduleCache {
	if i.modules == nil {
		i.modules = &moduleCache{loaded: map[string]*loadedModule{}}
	}
	return i.modules
}

// loadModule returns the module called key, loading it first when it has not been:
// parse builds its program, and its top-level code runs once. The module is entered
// in the cache before it runs, so a module that imports itself, or one that imports it
// back, gets what has been loaded so far instead of loading forever (spec 4.3).
func (i *Interpreter) loadModule(key string, cfg LangConfig, parse func() (*ast.Program, error)) (*Interpreter, *ast.Program, error) {
	cache := i.moduleCache()
	if m, ok := cache.loaded[key]; ok {
		return m.sub, m.prog, nil
	}
	prog, err := parse()
	if err != nil {
		return nil, nil, err
	}
	sub := i.newSubInterpreter(prog, cfg)
	markModule(prog, sub)
	cache.loaded[key] = &loadedModule{sub: sub, prog: prog}
	if err := sub.Run(); err != nil {
		delete(cache.loaded, key)
		return nil, nil, err
	}
	return sub, prog, nil
}
