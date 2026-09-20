package cmd

import (
	"fmt"
	"github.com/soumt-r/hana/conv"
	"io/ioutil"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/bytecode"
	"github.com/soumt-r/hana/console"
	"github.com/soumt-r/hana/errs"
	lexer "github.com/soumt-r/hana/lexer/haja"
	kanadeLexer "github.com/soumt-r/hana/lexer/kanade"
	parser "github.com/soumt-r/hana/parser/haja"
	kanadeParser "github.com/soumt-r/hana/parser/kanade"
	"github.com/soumt-r/hana/runner"
	"github.com/soumt-r/hana/std/stdimpl"
	"github.com/soumt-r/hana/stdlib"
	"github.com/soumt-r/hana/vm"

	"github.com/spf13/cobra"
)

// A Haja program allocates a lot of short-lived values; collecting less often
// (the Go default is 100) makes it about 10% faster for a few times the memory.
const runGCPercent = 400

var runCmd = &cobra.Command{
	Use:  "run",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		debug.SetGCPercent(runGCPercent)
		filename := args[0]
		timing, _ := cmd.Flags().GetBool("timing")
		useBytecode, _ := cmd.Flags().GetBool("bc")
		if allow, _ := cmd.Flags().GetBool("allow-file"); !allow {
			stdimpl.FileAccess = stdimpl.DenyFiles
		}
		if allow, _ := cmd.Flags().GetBool("allow-net"); !allow {
			stdimpl.NetAccess = stdimpl.DenyNet
		}

		if strings.HasSuffix(filename, ".hn") {
			runPrecompiled(filename, timing)
			return
		}

		isKanade := strings.HasSuffix(filename, ".knd")
		loc := localeFor(isKanade)

		content, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Printf("%s: %v\n", errs.LabelText(loc, errs.LabelFileRead), err)
			os.Exit(1)
		}

		// 파싱 단계
		parseStart := time.Now()
		var prog *ast.Program
		var parseErrors []string
		if isKanade {
			l := kanadeLexer.New(string(content))
			p := kanadeParser.New(l)
			prog = p.ParseProgram()
			parseErrors = p.LocalizedErrors(loc)
		} else {
			l := lexer.New(string(content))
			p := parser.New(l)
			prog = p.ParseProgram()
			parseErrors = p.LocalizedErrors(loc)
		}
		parseDuration := time.Since(parseStart)

		if len(parseErrors) > 0 {
			fmt.Println(errs.LabelText(loc, errs.LabelParseFailed) + ":")
			for _, msg := range parseErrors {
				fmt.Printf("  - %s\n", msg)
			}
			os.Exit(1)
		}

		var runDuration time.Duration
		if useBytecode {
			compileStart := time.Now()
			var compiler *bytecode.Compiler
			if isKanade {
				compiler = bytecode.NewKanadeCompiler()
			} else {
				compiler = bytecode.NewCompiler()
			}
			bcProg := compiler.Compile(prog)
			parseDuration += time.Since(compileStart)
			if len(compiler.Errors()) > 0 {
				fmt.Println(errs.LabelText(loc, errs.LabelCompileFailed) + ":")
				for _, msg := range compiler.Errors() {
					fmt.Printf("  - %s\n", msg)
				}
				os.Exit(1)
			}

			bcLang := bytecode.LangHaja
			if isKanade {
				bcLang = bytecode.LangKanade
			}
			bcv := runner.NewVM(bcProg, bcLang)

			runStart := time.Now()
			err = bcv.Run()
			runDuration = time.Since(runStart)
		} else {
			langConfig := vm.KoreanConfig
			if isKanade {
				langConfig = vm.JapaneseConfig
			}
			interpreter := vm.NewInterpreter(prog, langConfig)
			stdlib.RegisterStandardLibrary(interpreter)
			interpreter.ReadLine = stdinLines()

			runStart := time.Now()
			err = interpreter.Run()
			runDuration = time.Since(runStart)
		}

		console.Flush()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", errs.LabelText(loc, errs.LabelRuntime), errs.Localize(loc, err))
			os.Exit(1)
		}

		if timing {
			total := parseDuration + runDuration
			parseLabel := T("timing.parse")
			if useBytecode {
				parseLabel = T("timing.parseCompile")
			}
			fmt.Fprintf(os.Stderr, "\n%s\n", T("timing.title"))
			fmt.Fprintf(os.Stderr, "   %s:  %v\n", parseLabel, parseDuration)
			fmt.Fprintf(os.Stderr, "   %s:  %v\n", T("timing.run"), runDuration)
			fmt.Fprintf(os.Stderr, "   %s:  %v\n", T("timing.total"), total)
		}
	},
}

// runPrecompiled runs a .hn file directly, skipping lex/parse/compile
// entirely — always via bcvm, since a .hn file already *is* bytecode
// (--bc is meaningless here and ignored).
func runPrecompiled(filename string, timing bool) {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("%s: %v\n", errs.LabelText(errs.Korean, errs.LabelFileRead), err)
		os.Exit(1)
	}
	defer f.Close()

	loadStart := time.Now()
	bcProg, lang, err := bytecode.ReadProgram(f)
	loadDuration := time.Since(loadStart)
	if err != nil {
		fmt.Printf("%s: %v\n", errs.LabelText(errs.Korean, errs.LabelBytecodeFile), err)
		os.Exit(1)
	}

	bcv := runner.NewVM(bcProg, lang)

	runStart := time.Now()
	err = bcv.Run()
	runDuration := time.Since(runStart)
	console.Flush()

	if err != nil {
		runner.PrintRuntimeError(lang, err)
		os.Exit(1)
	}

	if timing {
		total := loadDuration + runDuration
		fmt.Fprintf(os.Stderr, "\n%s\n", T("timing.title"))
		fmt.Fprintf(os.Stderr, "   %s:  %v\n", T("timing.load"), loadDuration)
		fmt.Fprintf(os.Stderr, "   %s:  %v\n", T("timing.run"), runDuration)
		fmt.Fprintf(os.Stderr, "   %s:  %v\n", T("timing.total"), total)
	}
}

func init() {
	runCmd.Flags().Bool("allow-file", true, "")
	runCmd.Flags().Bool("allow-net", true, "")
	runCmd.Flags().BoolP("timing", "t", false, "")
	runCmd.Flags().Bool("bc", false, "")
	rootCmd.AddCommand(runCmd)
}

// localeFor is the wording every CLI-printed error uses for a script of the
// given language — the same Locale vm.LangConfig/bcvm.VM carry.
func localeFor(isKanade bool) errs.Locale {
	if isKanade {
		return errs.Japanese
	}
	return errs.Korean
}

// stdinLines is the input source `hana run` gives a program's `입력받자`: one
// line of standard input per call, "" once stdin is exhausted.
func stdinLines() func() string {
	return conv.NewLineReader(os.Stdin)
}
