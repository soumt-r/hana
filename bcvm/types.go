package bcvm

import (
	"github.com/soumt-r/hana/bytecode"
	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/symbol"
	"github.com/soumt-r/hana/typecheck"
)

// typeHost lets the engine-neutral type checker ask about user classes.
type typeHost struct{ vm *VM }

func (h typeHost) ClassOf(v interface{}) (string, bool) {
	if obj, ok := v.(*Object); ok {
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
		cls := h.vm.program.Classes[class]
		if cls == nil {
			return false
		}
		for _, iface := range cls.Interfaces {
			if iface == target {
				return true
			}
		}
		class = cls.BaseClass
	}
	return false
}

func (vm *VM) host() typecheck.Host { return typeHost{vm} }

// requireBool is a condition's value: only true and false are conditions.
func (vm *VM) requireBool(v interface{}) (bool, error) {
	if b, ok := v.(bool); ok {
		return b, nil
	}
	return false, errs.New(errs.ConditionNotBoolean, typecheck.Describe(&vm.Types, v, vm.host()))
}

// fieldType finds the [타입] a class (or an ancestor) declared for an instance field.
func (vm *VM) fieldType(className, prop string) string {
	for className != "" {
		cls := vm.program.Classes[className]
		if cls == nil {
			return ""
		}
		for _, f := range cls.Fields {
			if f.Name == prop {
				return f.Type
			}
		}
		className = cls.BaseClass
	}
	return ""
}

// checkField enforces a field's declared type on a write.
func (vm *VM) checkField(className, prop string, val interface{}) error {
	if annotation := vm.fieldType(className, prop); annotation != "" {
		return typecheck.Check(&vm.Types, annotation, prop, val, vm.host())
	}
	return nil
}

// bindParam binds one argument to its parameter in fr, enforcing the
// parameter's declared type (which then keeps constraining the body's writes).
func (vm *VM) bindParam(fr *frame, sym symbol.Symbol, p *bytecode.Param, val interface{}) error {
	if p.Type != "" && !typecheck.QuickAccepts(&vm.Types, p.Type, val) {
		if err := typecheck.CheckArgument(&vm.Types, p.Type, p.Name, val, vm.host()); err != nil {
			return err
		}
	}
	s := fr.put(sym, val)
	s.typ = p.Type
	return nil
}
