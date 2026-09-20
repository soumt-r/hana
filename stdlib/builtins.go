package stdlib

import (
	"github.com/soumt-r/hana/errs"
	"strconv"
	"unicode/utf8"

	"github.com/soumt-r/hana/std"
	"github.com/soumt-r/hana/std/stdimpl"
	"github.com/soumt-r/hana/vm"
)

func RegisterStandardLibrary(i *vm.Interpreter) {
	cfg := i.Config
	i.Bootstrap = RegisterStandardLibrary

	i.RegisterBuiltin(cfg.BuiltinToString, &vm.BuiltinFunction{Name: cfg.BuiltinToString, Fn: func(interp *vm.Interpreter, e *vm.Environment, args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errs.New(errs.ArgCountExact, 1)
		}
		return interp.FormatValue(args[0]), nil
	}})

	i.RegisterBuiltin(cfg.BuiltinToNumber, &vm.BuiltinFunction{Name: cfg.BuiltinToNumber, Fn: func(interp *vm.Interpreter, e *vm.Environment, args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errs.New(errs.ArgCountExact, 1)
		}
		switch v := args[0].(type) {
		case float64:
			return v, nil
		case string:
			num, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, errs.New(errs.ConvertToNumberFailed, v)
			}
			return num, nil
		default:
			return nil, errs.New(errs.ConvertToNumberInvalid)
		}
	}})

	i.RegisterBuiltin(cfg.BuiltinToCode, &vm.BuiltinFunction{Name: cfg.BuiltinToCode, Fn: func(interp *vm.Interpreter, e *vm.Environment, args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errs.New(errs.ArgCountExact, 1)
		}
		s, ok := args[0].(string)
		if !ok || utf8.RuneCountInString(s) != 1 {
			return nil, errs.New(errs.ConvertToCodeNeedsOneChar, cfg.BuiltinToCode)
		}
		r, _ := utf8.DecodeRuneInString(s)
		return float64(r), nil
	}})

	i.RegisterBuiltin(cfg.BuiltinToText, &vm.BuiltinFunction{Name: cfg.BuiltinToText, Fn: func(interp *vm.Interpreter, e *vm.Environment, args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errs.New(errs.ArgCountExact, 1)
		}
		num, ok := args[0].(float64)
		if !ok {
			return nil, errs.New(errs.ConvertToTextNeedsNumber)
		}
		return string(rune(num)), nil
	}})

	registerNativeModules(i)
}

// registerNativeModules exposes every std module under the interpreter's
// language names. A declared function without an implementation is a
// programming error, caught by the tests that walk std.Modules.
func registerNativeModules(i *vm.Interpreter) {
	lang := i.Config.Name
	for _, m := range std.Modules {
		module := vm.NativeModule{}
		for _, id := range m.Functions {
			name := std.FunctionName(lang, id)
			if host, ok := stdimpl.HostImpls[id]; ok {
				module[name] = &vm.BuiltinFunction{Name: name, Fn: func(interp *vm.Interpreter, e *vm.Environment, args ...interface{}) (interface{}, error) {
					return host(interp.CallFunction, args)
				}}
				continue
			}
			impl, ok := stdimpl.Impls[id]
			if !ok {
				panic("stdlib: no implementation for native function " + id)
			}
			module[name] = &vm.BuiltinFunction{Name: name, Fn: func(interp *vm.Interpreter, e *vm.Environment, args ...interface{}) (interface{}, error) {
				return impl(args)
			}}
		}
		i.RegisterNativeModule(std.ModuleName(lang, m.ID), module)
	}
}
