// hana-lsp is the language server as a standalone binary: the VS Code
// extension bundles it per platform. It imports only the lexers, parsers and
// the lsp package — not the VM — so it cross-compiles everywhere, unlike the
// full hana CLI (whose VM has Windows-only plugin loading). `hana lsp` runs
// the same server for anyone who already has hana installed.
package main

import (
	"fmt"
	"os"

	"github.com/soumt-r/hana/lsp"
)

func main() {
	if err := lsp.NewServer(os.Stdin, os.Stdout).Serve(); err != nil {
		fmt.Fprintln(os.Stderr, "hana-lsp:", err)
		os.Exit(1)
	}
}
