package vm

import (
	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/symbol"
)

type Environment struct {
	vars      []varEntry             // 대부분의 스코프는 변수가 몇 개뿐이라 작은 정수(Symbol)를 훑는다
	index     []int32                // Symbol → 위치+1 (0: 없음); 변수가 많아진 스코프(최상위 등)에서만 만든다
	constants map[symbol.Symbol]bool // 상수로 선언된 변수 추적 (처음 쓸 때 만듦)
	parent    *Environment
	this      *HariObject
}

type varEntry struct {
	sym   symbol.Symbol
	value interface{}
	typ   string // 선언 때 붙은 타입 표기 (Runtime spec 2.2), 없으면 ""
}

const envIndexThreshold = 12

// The names every method call declares in its scope.
var (
	thisSym      = symbol.Intern("this")
	selfClassSym = symbol.Intern("__selfClass__")
)

func NewEnvironment(parent *Environment) *Environment {
	return &Environment{parent: parent}
}

// newScope/freeScope hand out call and loop scopes from a small free list: a
// scope is dead once its body has run (nothing keeps a reference to it), and
// making one is otherwise an allocation per call and per loop pass.
func (i *Interpreter) newScope(parent *Environment) *Environment {
	if n := len(i.scopePool); n > 0 {
		e := i.scopePool[n-1]
		i.scopePool = i.scopePool[:n-1]
		e.parent = parent
		return e
	}
	return NewEnvironment(parent)
}

func (i *Interpreter) freeScope(e *Environment) {
	if cap(e.vars) > 64 || len(i.scopePool) >= 64 {
		return
	}
	clear(e.vars)
	*e = Environment{vars: e.vars[:0]}
	i.scopePool = append(i.scopePool, e)
}

func (e *Environment) find(sym symbol.Symbol) int {
	if e.index != nil {
		if int(sym) < len(e.index) {
			return int(e.index[sym]) - 1
		}
		return -1
	}
	for i := range e.vars {
		if e.vars[i].sym == sym {
			return i
		}
	}
	return -1
}

// typeOf returns the type sym was declared with in this scope itself (not a
// parent); found says whether the scope declares the variable at all.
func (e *Environment) typeOf(sym symbol.Symbol) (typ string, found bool) {
	if i := e.find(sym); i >= 0 {
		return e.vars[i].typ, true
	}
	return "", false
}

// The methods named for a variable take its Symbol; the ones that take a name
// intern it first, for code that has only the text (imports, builtins).

func (e *Environment) DeclareSym(sym symbol.Symbol, value interface{}) {
	if i := e.find(sym); i >= 0 {
		e.vars[i].value = value
		return
	}
	if e.vars == nil {
		e.vars = make([]varEntry, 0, 4)
	}
	e.vars = append(e.vars, varEntry{sym: sym, value: value})
	n := len(e.vars)
	if e.index != nil {
		e.setIndex(sym, n-1)
	} else if n > envIndexThreshold {
		for i := range e.vars {
			e.setIndex(e.vars[i].sym, i)
		}
	}
}

func (e *Environment) setIndex(sym symbol.Symbol, slot int) {
	if int(sym) >= len(e.index) {
		grown := make([]int32, symbol.Count())
		copy(grown, e.index)
		e.index = grown
	}
	e.index[sym] = int32(slot) + 1
}

func (e *Environment) Declare(name string, value interface{}) {
	e.DeclareSym(symbol.Intern(name), value)
}

func (e *Environment) DeclareConstSym(sym symbol.Symbol, value interface{}) {
	e.DeclareSym(sym, value)
	if e.constants == nil {
		e.constants = make(map[symbol.Symbol]bool)
	}
	e.constants[sym] = true
}

// DeclareTypeSym records the type a variable declared in this scope was
// annotated with; later assignments must keep honoring it.
func (e *Environment) DeclareTypeSym(sym symbol.Symbol, annotation string) {
	if i := e.find(sym); i >= 0 {
		e.vars[i].typ = annotation
	}
}

// isConst reports whether the variable sym names from here was declared 고정하자: the
// nearest scope that has sym decides, so a loop or handler variable that hides an outer
// constant of the same name is not a constant (Runtime spec 1.1).
func (e *Environment) isConst(sym symbol.Symbol) bool {
	for env := e; env != nil; env = env.parent {
		if env.find(sym) >= 0 {
			return env.constants != nil && env.constants[sym]
		}
	}
	return false
}

func (e *Environment) AssignSym(sym symbol.Symbol, value interface{}) (bool, error) {
	if i := e.find(sym); i >= 0 {
		if e.isConst(sym) {
			return false, errs.New(errs.ConstantAssignment, sym.String())
		}
		e.vars[i].value = value
		return true, nil
	}
	if e.this != nil {
		name := sym.String()
		if _, ok := e.this.Props[name]; ok {
			e.this.Props[name] = value
			return true, nil
		}
	}
	if e.parent != nil {
		return e.parent.AssignSym(sym, value)
	}
	return false, nil
}

func (e *Environment) Assign(name string, value interface{}) (bool, error) {
	return e.AssignSym(symbol.Intern(name), value)
}

func (e *Environment) GetSym(sym symbol.Symbol) (interface{}, bool) {
	if i := e.find(sym); i >= 0 {
		return e.vars[i].value, true
	}
	if e.this != nil {
		if val, ok := e.this.Props[sym.String()]; ok {
			return val, true
		}
	}
	if e.parent != nil {
		return e.parent.GetSym(sym)
	}
	return nil, false
}

func (e *Environment) Get(name string) (interface{}, bool) {
	return e.GetSym(symbol.Intern(name))
}

func (e *Environment) GetAll() map[string]interface{} {
	all := make(map[string]interface{}, len(e.vars))
	for _, v := range e.vars {
		all[v.sym.String()] = v.value
	}
	return all
}
