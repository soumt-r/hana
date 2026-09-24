package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/bytecode"
	"github.com/soumt-r/hana/errs"
	lexer "github.com/soumt-r/hana/lexer/hari"
	kanadeLexer "github.com/soumt-r/hana/lexer/kanade"
	parser "github.com/soumt-r/hana/parser/hari"
	kanadeParser "github.com/soumt-r/hana/parser/kanade"

	"github.com/spf13/cobra"
)

var disasmCmd = &cobra.Command{
	Use:  "disasm",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		loc := localeFor(strings.HasSuffix(filename, ".knd"))

		if strings.HasSuffix(filename, ".hn") {
			f, err := os.Open(filename)
			if err != nil {
				fmt.Printf("%s: %v\n", errs.LabelText(loc, errs.LabelFileRead), err)
				os.Exit(1)
			}
			defer f.Close()
			bcProg, _, err := bytecode.ReadProgram(f)
			if err != nil {
				fmt.Printf("%s: %v\n", errs.LabelText(loc, errs.LabelBytecodeFile), err)
				os.Exit(1)
			}
			fmt.Print(bytecode.DisassembleProgram(bcProg))
			return
		}

		content, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Printf("%s: %v\n", errs.LabelText(loc, errs.LabelFileRead), err)
			os.Exit(1)
		}

		var prog *ast.Program
		var parseErrors []string
		var compiler *bytecode.Compiler
		if strings.HasSuffix(filename, ".knd") {
			l := kanadeLexer.New(string(content))
			p := kanadeParser.New(l)
			prog = p.ParseProgram()
			parseErrors = p.Errors()
			compiler = bytecode.NewKanadeCompiler()
		} else {
			l := lexer.New(string(content))
			p := parser.New(l)
			prog = p.ParseProgram()
			parseErrors = p.Errors()
			compiler = bytecode.NewCompiler()
		}
		if len(parseErrors) > 0 {
			fmt.Println(errs.LabelText(loc, errs.LabelParseFailed) + ":")
			for _, msg := range parseErrors {
				fmt.Printf("  - %s\n", msg)
			}
			os.Exit(1)
		}

		bcProg := compiler.Compile(prog)
		if len(compiler.Errors()) > 0 {
			fmt.Println(errs.LabelText(loc, errs.LabelCompileFailed) + ":")
			for _, msg := range compiler.Errors() {
				fmt.Printf("  - %s\n", msg)
			}
			os.Exit(1)
		}

		fmt.Print(bytecode.DisassembleProgram(bcProg))
	},
}

func init() {
	rootCmd.AddCommand(disasmCmd)
}
