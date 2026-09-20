// Package tests holds hana's end-to-end regression tests: lex + parse +
// run a .hj source string through the real interpreter and assert on its
// output/errors. These are black-box tests against the public API of
// lexer/haja, parser/haja, vm, and stdlib, kept in one place instead of
// scattered per-package so a bug fix and its regression test live next to
// every other one, regardless of which package the fix landed in.
package tests

import (
	"strings"
	"testing"

	lexer "github.com/soumt-r/hana/lexer/haja"
	parser "github.com/soumt-r/hana/parser/haja"
	"github.com/soumt-r/hana/stdlib"
	"github.com/soumt-r/hana/vm"
)

// runHaja lexes, parses, and runs a haja source string with the standard
// library registered, returning the interpreter (so tests can inspect
// Output/GlobalEnv) and any runtime error.
func runHaja(t *testing.T, code string) (*vm.Interpreter, error) {
	t.Helper()
	l := lexer.New(code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %v", p.Errors())
	}
	interp := vm.NewInterpreter(prog)
	stdlib.RegisterStandardLibrary(interp)
	err := interp.Run()
	return interp, err
}

// requireErrorContains fails the test unless err is non-nil and its message
// contains substr (typically a Haja error class name like "MissingArgumentError").
func requireErrorContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error containing %q, got nil", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("expected error containing %q, got: %v", substr, err)
	}
}
