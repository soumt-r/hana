package bcvm

import (
	"github.com/soumt-r/hana/bytecode"
	"github.com/soumt-r/hana/symbol"
)

// Object is a class instance. Mirrors vm.HariObject.
type Object struct {
	ClassName string
	Props     map[string]interface{}
	members   map[symbol.Symbol]*memberEntry // the class's shared member cache, set on first use
}

func newObject(className string) *Object {
	return &Object{ClassName: className, Props: make(map[string]interface{})}
}

// classRef is a class used as a value (static member access, instanceof).
// Mirrors vm.ClassReference.
type classRef struct {
	ClassName string
}

// superRef wraps 나 when reached through 부모의 — only meaningful as
// GET_MEMBER's object, where it produces a boundMethod with IsSuper set.
// Mirrors vm.SuperReference.
type superRef struct {
	Receiver *Object
}

// boundMethod is an instance method looked up but not yet called (produced
// by GET_MEMBER's isFunc branch, consumed by CALL_METHOD). Mirrors
// vm.BoundMethod.
type boundMethod struct {
	Receiver *Object
	Method   string
	IsSuper  bool
	fn       *bytecode.Function // the method GET_MEMBER already resolved; nil when it has to be looked up
}

// boundStaticMethod is a class ('우리' or an explicit [클래스]) method looked
// up but not yet called. Mirrors vm.BoundStaticMethod.
type boundStaticMethod struct {
	ClassName string
	Method    string
}

// boundStringMethod is one of the small set of builtin pseudo-methods a
// string exposes (자르기/바꾸기/분리하기/포함확인). Mirrors
// vm.BoundStringMethod — see callMethod for the dispatch table.
type boundStringMethod struct {
	Value  string
	Method string
}

// findMethod walks className's BaseClass chain (own Methods/Constructor
// only at each level, never merged) looking for methodName ("__init__"
// means the constructor). Mirrors vm.findInClassChain as used by
// CallExpression's BoundMethod branch.
func (vm *VM) findMethod(className, methodName string) (*bytecode.Function, bool) {
	if methodName != "__init__" {
		fn := vm.member(className, symbol.Intern(methodName)).method
		return fn, fn != nil
	}
	for className != "" {
		cls, ok := vm.program.Classes[className]
		if !ok {
			return nil, false
		}
		if methodName == "__init__" {
			if cls.Constructor != nil {
				return cls.Constructor, true
			}
		} else if fn, ok := cls.Methods[methodName]; ok {
			return fn, true
		}
		className = cls.BaseClass
	}
	return nil, false
}

// memberEntry memoizes what walking a class's BaseClass chain finds for one
// member name. Classes never change while a program runs, so the answer for
// (class, name) is fixed: the first class in the chain that declares a field
// of that name decides its access, getter and setter, and the first that
// declares a method of that name decides the method.
type memberEntry struct {
	fieldAccess, methodAccess string
	getter                    *bytecode.Chunk
	setter                    *bytecode.SetterInfo
	method                    *bytecode.Function
}

// classMembers is the member cache shared by every instance of className.
func (vm *VM) classMembers(className string) map[symbol.Symbol]*memberEntry {
	m, ok := vm.members[className]
	if !ok {
		if vm.members == nil {
			vm.members = make(map[string]map[symbol.Symbol]*memberEntry)
		}
		m = make(map[symbol.Symbol]*memberEntry)
		vm.members[className] = m
	}
	return m
}

func (vm *VM) member(className string, sym symbol.Symbol) *memberEntry {
	return vm.lookupMember(vm.classMembers(className), className, sym)
}

// memberOf is member for an instance, which keeps its class's cache at hand.
func (vm *VM) memberOf(o *Object, sym symbol.Symbol) *memberEntry {
	if o.members == nil {
		o.members = vm.classMembers(o.ClassName)
	}
	return vm.lookupMember(o.members, o.ClassName, sym)
}

func (vm *VM) lookupMember(cache map[symbol.Symbol]*memberEntry, className string, sym symbol.Symbol) *memberEntry {
	if e, ok := cache[sym]; ok {
		return e
	}
	prop := sym.String()
	e := &memberEntry{fieldAccess: "public", methodAccess: "public"}
	fieldDone := false
	for name := className; name != ""; {
		cls, ok := vm.program.Classes[name]
		if !ok {
			break
		}
		if !fieldDone {
			for _, f := range cls.Fields {
				if f.Name == prop {
					e.fieldAccess, e.getter, e.setter = f.Access, f.Getter, f.Setter
					fieldDone = true
					break
				}
			}
		}
		if e.method == nil {
			if fn, ok := cls.Methods[prop]; ok {
				e.method, e.methodAccess = fn, fn.Access
			}
		}
		if fieldDone && e.method != nil {
			break
		}
		name = cls.BaseClass
	}
	cache[sym] = e
	return e
}

// classIsOrExtends reports whether className is targetName itself or
// inherits from it (directly or transitively) via BaseClass. Mirrors
// vm.classIsOrExtends exactly, including its existing gap: interface
// membership is never checked here, only class inheritance.
func (vm *VM) classIsOrExtends(className, targetName string) bool {
	for className != "" {
		if className == targetName {
			return true
		}
		cls, ok := vm.program.Classes[className]
		if !ok || cls.BaseClass == "" {
			return false
		}
		className = cls.BaseClass
	}
	return false
}
