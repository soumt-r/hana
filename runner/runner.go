// Package runner sets up and reports on the bytecode VM the way the command
// line does. `hana run x.hn` and packed programs share it, so both behave alike.
package runner

import (
	"fmt"
	"os"

	"github.com/soumt-r/hana/bcstdlib"
	"github.com/soumt-r/hana/bcvm"
	"github.com/soumt-r/hana/bytecode"
	"github.com/soumt-r/hana/console"
	"github.com/soumt-r/hana/conv"
	"github.com/soumt-r/hana/errs"
)

// NewVM is a VM for prog with the standard library of the language it was
// compiled from, reading `입력받자` from standard input.
func NewVM(prog *bytecode.Program, lang bytecode.Lang) *bcvm.VM {
	v := bcvm.New(prog)
	config := bcstdlib.Korean
	if lang == bytecode.LangKanade {
		config = bcstdlib.Japanese
		v.UseJapaneseWords()
	}
	bcstdlib.RegisterStandardLibrary(v, config)
	v.ReadLine = conv.NewLineReader(os.Stdin)
	return v
}

// Locale is the wording errors are printed in for a program of the language.
func Locale(lang bytecode.Lang) errs.Locale {
	if lang == bytecode.LangKanade {
		return errs.Japanese
	}
	return errs.Korean
}

// PrintRuntimeError writes a run's error to standard error.
func PrintRuntimeError(lang bytecode.Lang, err error) {
	console.Flush()
	loc := Locale(lang)
	fmt.Fprintf(os.Stderr, "%s: %s\n", errs.LabelText(loc, errs.LabelRuntime), errs.Localize(loc, err))
}
