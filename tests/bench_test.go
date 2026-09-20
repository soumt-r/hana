package tests

// Speed benchmarks for the two engines: the programs in bench/ run on the tree-walking
// interpreter and on the bytecode VM. Parsing and compiling are outside the timing.
//
//	go test ./tests -run XXX -bench Programs -benchtime 3x
//	go test ./tests -run XXX -bench Programs/lists/bytecode -cpuprofile cpu.out
//
// (go tool pprof -top cpu.out shows where the time goes.)

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soumt-r/hana/bcstdlib"
	"github.com/soumt-r/hana/bcvm"
	"github.com/soumt-r/hana/bytecode"
	lexer "github.com/soumt-r/hana/lexer/haja"
	parser "github.com/soumt-r/hana/parser/haja"
	"github.com/soumt-r/hana/stdlib"
	"github.com/soumt-r/hana/vm"
)

func BenchmarkPrograms(b *testing.B) {
	files, _ := filepath.Glob("../bench/*.hj")
	if len(files) == 0 {
		b.Skip("no programs in ../bench")
	}
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = null
	defer func() { os.Stdout = stdout }()

	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".hj")
		data, err := os.ReadFile(file)
		if err != nil {
			b.Fatal(err)
		}
		source := string(data)
		parse := func() *parser.Parser { return parser.New(lexer.New(source)) }

		b.Run(name+"/tree", func(b *testing.B) {
			p := parse()
			prog := p.ParseProgram()
			if len(p.Errors()) > 0 {
				b.Fatal(p.Errors())
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				interp := vm.NewInterpreter(prog)
				stdlib.RegisterStandardLibrary(interp)
				if err := interp.Run(); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(name+"/bytecode", func(b *testing.B) {
			p := parse()
			prog := p.ParseProgram()
			if len(p.Errors()) > 0 {
				b.Fatal(p.Errors())
			}
			compiled := bytecode.NewCompiler().Compile(prog)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				machine := bcvm.New(compiled)
				bcstdlib.RegisterStandardLibrary(machine)
				if err := machine.Run(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
