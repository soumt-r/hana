package vm

import (
	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/typecheck"
	"github.com/soumt-r/hana/value"
)

// typeHost lets the engine-neutral type checker ask about user classes.
type typeHost struct{ i *Interpreter }

func (h typeHost) ClassOf(v interface{}) (string, bool) {
	if obj, ok := v.(*HajaObject); ok {
		return obj.ClassName, true
	}
	return "", false
}

// IsSubtype is true when class is target, extends it, or (at any level of the
// chain) declares that it follows it as an interface.
func (h typeHost) IsSubtype(class, target string) bool {
	for class != "" {
		if class == target {
			return true
		}
		cls := h.i.Classes[class]
		if cls == nil {
			return false
		}
		for _, iface := range cls.Interfaces {
			if iface != nil && iface.Name == target {
				return true
			}
		}
		if cls.BaseClass == nil {
			return false
		}
		class = cls.BaseClass.Name
	}
	return false
}

func (i *Interpreter) host() typecheck.Host { return typeHost{i} }

// requireBool is a condition's value: only true and false are conditions.
func (i *Interpreter) requireBool(v interface{}) (bool, error) {
	if b, ok := v.(bool); ok {
		return b, nil
	}
	return false, errs.New(errs.ConditionNotBoolean, typecheck.Describe(&i.Config.Types, v, i.host()))
}

// fieldAnnotation finds the type a class (or an ancestor) declared for a field.
func (i *Interpreter) fieldAnnotation(className, prop string) (string, bool) {
	for className != "" {
		cls := i.Classes[className]
		if cls == nil {
			return "", false
		}
		for _, stmt := range cls.Body {
			if v, ok := stmt.(*ast.VariableDeclaration); ok && !v.IsStatic && v.Name.Value == prop {
				if v.TypeRef != nil {
					return v.TypeRef.Name, true
				}
				return "", false
			}
		}
		if cls.BaseClass == nil {
			return "", false
		}
		className = cls.BaseClass.Name
	}
	return "", false
}

// declaredTypeOf resolves name the way Environment.Assign does (this scope,
// then the object's fields, then outward) and returns the type it was declared
// with, if any.
func (i *Interpreter) declaredTypeOf(env *Environment, id *ast.Identifier) (string, bool) {
	sym := id.Symbol()
	for e := env; e != nil; e = e.parent {
		if t, found := e.typeOf(sym); found {
			return t, t != ""
		}
		if e.this != nil {
			if _, ok := e.this.Props[id.Value]; ok {
				return i.fieldAnnotation(e.this.ClassName, id.Value)
			}
		}
	}
	return "", false
}

// checkDeclaredType enforces the type an existing variable or field was
// declared with (Runtime spec 2.2: the constraint outlives the declaration).
func (i *Interpreter) checkDeclaredType(env *Environment, id *ast.Identifier, val interface{}) error {
	if declared, ok := i.declaredTypeOf(env, id); ok {
		return typecheck.Check(&i.Config.Types, declared, id.Value, val, i.host())
	}
	return nil
}

// checkListWrite is checkDeclaredType for a list that got a value at one end.
func (i *Interpreter) checkListWrite(env *Environment, id *ast.Identifier, list *value.List, front bool) error {
	if declared, ok := i.declaredTypeOf(env, id); ok {
		return typecheck.CheckAppended(&i.Config.Types, declared, id.Value, list, front, i.host())
	}
	return nil
}

// checkFieldListWrite is checkField for a list that got a value at one end.
func (i *Interpreter) checkFieldListWrite(obj *HajaObject, prop string, list *value.List, front bool) error {
	if annotation, ok := i.fieldAnnotation(obj.ClassName, prop); ok {
		return typecheck.CheckAppended(&i.Config.Types, annotation, prop, list, front, i.host())
	}
	return nil
}

// checkField enforces a class field's declared type on a write.
func (i *Interpreter) checkField(obj *HajaObject, prop string, val interface{}) error {
	if annotation, ok := i.fieldAnnotation(obj.ClassName, prop); ok {
		return typecheck.Check(&i.Config.Types, annotation, prop, val, i.host())
	}
	return nil
}

// assignVariable is where a `정하자` statement writes a variable. It checks the
// value against this statement's annotation and against the type an earlier
// declaration gave the variable, then assigns — or, for a new variable, declares
// it (remembering the annotation so it keeps constraining later writes).
func (i *Interpreter) assignVariable(env *Environment, id *ast.Identifier, val interface{}, annotation string, isConst bool) error {
	if annotation != "" {
		if err := typecheck.Check(&i.Config.Types, annotation, id.Value, val, i.host()); err != nil {
			return err
		}
	}
	if declared, ok := i.declaredTypeOf(env, id); ok && declared != annotation {
		if err := typecheck.Check(&i.Config.Types, declared, id.Value, val, i.host()); err != nil {
			return err
		}
	}
	sym := id.Symbol()
	ok, err := env.AssignSym(sym, val)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	if isConst {
		env.DeclareConstSym(sym, val)
	} else {
		env.DeclareSym(sym, val)
	}
	if annotation != "" {
		env.DeclareTypeSym(sym, annotation)
	}
	return nil
}

// declareParam binds one argument to its parameter, enforcing the parameter's
// declared type; the type then keeps constraining assignments inside the body.
func (i *Interpreter) declareParam(env *Environment, param *ast.Parameter, val interface{}) error {
	if param.TypeAnnotation != nil {
		if err := typecheck.CheckArgument(&i.Config.Types, param.TypeAnnotation.Name, param.Name.Value, val, i.host()); err != nil {
			return err
		}
		sym := param.Name.Symbol()
		env.DeclareSym(sym, val)
		env.DeclareTypeSym(sym, param.TypeAnnotation.Name)
		return nil
	}
	env.DeclareSym(param.Name.Symbol(), val)
	return nil
}
