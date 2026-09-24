// Package bcstdlib mirrors stdlib for the bytecode VM: the same core
// language tools (문자로/숫자로/코드로/글자로, always callable — Runtime
// spec 1장 forbids implicit type conversion, so a program needs these to do
// it explicitly) and the same [수학] builtin module (올림/버림), registered
// against a *bcvm.VM instead of a *vm.Interpreter. It is a separate package
// rather than something stdlib itself exposes: the two engines' callable/value
// representations aren't compatible enough to share the registration code,
// only the behavior.
package bcstdlib

import (
	"github.com/soumt-r/hana/errs"
	"strconv"
	"unicode/utf8"

	"github.com/soumt-r/hana/bcvm"
	"github.com/soumt-r/hana/std"
	"github.com/soumt-r/hana/std/stdimpl"
)

// LangConfig names the builtins bcstdlib registers — bcvm's own equivalent
// of vm.LangConfig. A separate type rather than reusing vm.LangConfig on
// purpose: bcstdlib's own doc comment explains why this package doesn't
// share registration code with stdlib (the two engines' callable/value
// representations aren't compatible), and reusing vm's config type here
// would pull the bytecode engine into depending on the tree-walker package
// for no reason beyond avoiding a handful of duplicated string fields.
type LangConfig struct {
	ToString string
	ToNumber string
	ToCode   string
	ToText   string
	// Name is the language key hana/std's name tables are indexed by.
	Name string
}

var Korean = LangConfig{
	ToString: "문자로",
	ToNumber: "숫자로",
	ToCode:   "코드로",
	ToText:   "글자로",
	Name:     std.Hari,
}

var Japanese = LangConfig{
	ToString: "文字列に",
	ToNumber: "数字に",
	ToCode:   "コードに",
	ToText:   "文字に",
	Name:     std.Kanade,
}

// RegisterStandardLibrary registers the same core language tools stdlib
// gives the tree-walker, against a *bcvm.VM. cfg defaults to Korean (하자)
// when omitted, matching every existing caller.
func RegisterStandardLibrary(vm *bcvm.VM, cfg ...LangConfig) {
	c := Korean
	if len(cfg) > 0 {
		c = cfg[0]
	}

	vm.RegisterNative(c.ToString, &bcvm.NativeFunction{Name: c.ToString, Fn: func(vm *bcvm.VM, args []interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errs.New(errs.ArgCountExact, 1)
		}
		return vm.FormatValue(args[0]), nil
	}})

	vm.RegisterNative(c.ToNumber, &bcvm.NativeFunction{Name: c.ToNumber, Fn: func(vm *bcvm.VM, args []interface{}) (interface{}, error) {
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

	vm.RegisterNative(c.ToCode, &bcvm.NativeFunction{Name: c.ToCode, Fn: func(vm *bcvm.VM, args []interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errs.New(errs.ArgCountExact, 1)
		}
		s, ok := args[0].(string)
		if !ok || utf8.RuneCountInString(s) != 1 {
			return nil, errs.New(errs.ConvertToCodeNeedsOneChar, c.ToCode)
		}
		r, _ := utf8.DecodeRuneInString(s)
		return float64(r), nil
	}})

	vm.RegisterNative(c.ToText, &bcvm.NativeFunction{Name: c.ToText, Fn: func(vm *bcvm.VM, args []interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errs.New(errs.ArgCountExact, 1)
		}
		num, ok := args[0].(float64)
		if !ok {
			return nil, errs.New(errs.ConvertToTextNeedsNumber)
		}
		return string(rune(num)), nil
	}})

	registerNativeModules(vm, c.Name)
}

func registerNativeModules(vm *bcvm.VM, lang string) {
	for _, m := range std.Modules {
		module := map[string]*bcvm.NativeFunction{}
		for _, id := range m.Functions {
			name := std.FunctionName(lang, id)
			if host, ok := stdimpl.HostImpls[id]; ok {
				module[name] = &bcvm.NativeFunction{Name: name, Fn: func(vm *bcvm.VM, args []interface{}) (interface{}, error) {
					return host(vm.CallValue, args)
				}}
				continue
			}
			impl, ok := stdimpl.Impls[id]
			if !ok {
				panic("bcstdlib: no implementation for native function " + id)
			}
			module[name] = &bcvm.NativeFunction{Name: name, Fn: func(vm *bcvm.VM, args []interface{}) (interface{}, error) {
				return impl(args)
			}}
		}
		vm.RegisterNativeModule(std.ModuleName(lang, m.ID), module)
	}
}
