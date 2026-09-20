package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/bytecode"
	"github.com/soumt-r/hana/errs"
	lexer "github.com/soumt-r/hana/lexer/haja"
	kanadeLexer "github.com/soumt-r/hana/lexer/kanade"
	parser "github.com/soumt-r/hana/parser/haja"
	kanadeParser "github.com/soumt-r/hana/parser/kanade"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:  "build",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		outPath, _ := cmd.Flags().GetString("output")
		if outPath == "" {
			outPath = strings.TrimSuffix(filename, filepath.Ext(filename)) + ".hn"
		}

		bcProg, lang, _ := compileFile(filename)

		out, err := os.Create(outPath)
		if err != nil {
			fmt.Println(T("build.createFail", err))
			os.Exit(1)
		}
		defer out.Close()

		if err := bcProg.Encode(out, lang); err != nil {
			fmt.Println(T("build.saveFail", err))
			os.Exit(1)
		}

		fmt.Printf("✅ %s\n", outPath)
	},
}

// compileFile compiles a .hj/.knd file to bytecode. It also returns the
// packages whose native library the program loads at run time. Problems are
// printed and end the command, like everywhere else on the command line.
func compileFile(filename string) (*bytecode.Program, bytecode.Lang, []string) {
	isKanade := strings.HasSuffix(filename, ".knd")
	loc := localeFor(isKanade)

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("%s: %v\n", errs.LabelText(loc, errs.LabelFileRead), err)
		os.Exit(1)
	}

	var prog *ast.Program
	var parseErrors []string
	var compiler *bytecode.Compiler
	lang := bytecode.LangHaja
	if isKanade {
		l := kanadeLexer.New(string(content))
		p := kanadeParser.New(l)
		prog = p.ParseProgram()
		parseErrors = p.LocalizedErrors(loc)
		compiler = bytecode.NewKanadeCompiler()
		lang = bytecode.LangKanade
	} else {
		l := lexer.New(string(content))
		p := parser.New(l)
		prog = p.ParseProgram()
		parseErrors = p.LocalizedErrors(loc)
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
	return bcProg, lang, compiler.LibraryModules()
}

func init() {
	buildCmd.Flags().StringP("output", "o", "", "")
	rootCmd.AddCommand(buildCmd)
}
