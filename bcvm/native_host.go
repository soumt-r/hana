package bcvm

import (
	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/native"
)

// The VM is a native.Host: native libraries call back into the program through
// it (see the native package for the ABI and the rules).

// funcRef is a function used as a value (PUSH_FUNC_REF), found by name when it
// is called.
type funcRef struct{ Name string }

// Call implements native.Host: run a function value on this VM, waiting until
// the program is not executing.
func (vm *VM) Call(fn interface{}, args []interface{}) (interface{}, error) {
	vm.runLock.Acquire()
	defer vm.runLock.Release()
	return vm.callValue(fn, args)
}

// CallValue runs a function value for a native function that is already running
// inside the program (a callback of the standard library). Unlike Call it does
// not wait for the program, which is the caller.
func (vm *VM) CallValue(fn interface{}, args []interface{}) (interface{}, error) {
	return vm.callValue(fn, args)
}

// Identify implements native.Host.
func (vm *VM) Identify(v interface{}) (interface{}, bool) {
	switch f := v.(type) {
	case *funcRef:
		return "fn:" + f.Name, true
	case *boundMethod:
		return struct {
			receiver *Object
			method   string
		}{f.Receiver, f.Method}, true
	case *NativeFunction:
		return f, true
	}
	return nil, false
}

// BeginNative implements native.Host.
func (vm *VM) BeginNative() func() { return vm.runLock.Begin() }

// callValue runs a function value with already-evaluated arguments.
func (vm *VM) callValue(fn interface{}, args []interface{}) (interface{}, error) {
	switch f := fn.(type) {
	case *funcRef:
		if compiled, ok := vm.program.Functions[f.Name]; ok {
			return vm.callFunction(compiled, args)
		}
		if native, ok := vm.natives[f.Name]; ok {
			return native.Fn(vm, args)
		}
		return nil, errs.New(errs.GlobalFunctionNotFound, f.Name)
	case *boundMethod:
		return vm.callMethod(f, args)
	case *NativeFunction:
		return f.Fn(vm, args)
	}
	return nil, errs.New(errs.NotCallable)
}

// bindLibraryFunction binds a function of a package's native library under
// bind, so compiled code calls it by that name.
func (vm *VM) bindLibraryFunction(module, target, bind string) error {
	lib, err := native.OpenModule(module)
	if err != nil {
		return err
	}
	fn, ok := lib.Lookup(target)
	if !ok {
		return errs.New(errs.ImportNativeFunctionNotFound, module, target)
	}
	vm.natives[bind] = &NativeFunction{Name: bind, Fn: func(vm *VM, args []interface{}) (interface{}, error) {
		return fn(vm, args)
	}}
	return nil
}

var _ native.Host = (*VM)(nil)
