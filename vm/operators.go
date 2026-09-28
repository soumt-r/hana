package vm

import (
	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/symbol"
)

// callOperatorMethod runs the method op names (LangConfig.OperatorMethods,
// "==" for both == and !=) of an object on the left, as `A의 <기호 더하기>(B)`
// would (spec 3.5): found through the class's ancestors, arguments and result
// checked, counted in the call nesting. found is false when left is no object
// or no class in its chain has the method.
func (i *Interpreter) callOperatorMethod(left interface{}, op string, right interface{}) (val interface{}, found bool, err error) {
	obj, ok := left.(*HariObject)
	if !ok {
		return nil, false, nil
	}
	name, ok := i.Config.OperatorMethods[op]
	if !ok {
		return nil, false, nil
	}
	cls, ok := i.Classes[obj.ClassName]
	if !ok {
		return nil, false, nil
	}
	sym := symbol.Intern(name)
	if i.classMember(cls, sym).method == nil {
		return nil, false, nil
	}
	// A call, so it counts in the nesting as a call expression does.
	i.callDepth++
	if i.callDepth > MaxCallDepth {
		i.callDepth--
		return nil, true, errs.New(errs.CallTooDeep, MaxCallDepth)
	}
	val, err = i.CallFunction(&BoundMethod{Object: obj, FuncName: name, Sym: sym}, []interface{}{right})
	i.callDepth--
	return val, true, err
}
