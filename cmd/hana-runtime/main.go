// Command hana-runtime is the small executable a packed program is built on:
// the bytecode VM and the standard library, without the parser, the tree-walker
// or the command line. `hana pack` appends a program to a copy of it.
package main

import (
	"fmt"
	"os"

	"github.com/soumt-r/hana/console"
	"github.com/soumt-r/hana/pack"
	"github.com/soumt-r/hana/runner"
)

func main() {
	app, err := pack.OpenSelf()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	defer app.Close()
	if err := app.Activate(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	prog, lang, err := app.Program()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	err = runner.NewVM(prog, lang).Run()
	console.Flush()
	if err != nil {
		runner.PrintRuntimeError(lang, err)
		os.Exit(1)
	}
}
