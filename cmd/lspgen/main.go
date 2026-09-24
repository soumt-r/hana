// lspgen writes the language server's keyword tables as TypeScript modules, so
// the docs sites' browser editors suggest the same keywords as hana's language
// server. The Go tables stay the single source of truth; the generated files are
// never edited by hand.
//
//	go run ./cmd/lspgen -hari ../hari-docs/src/utils/hari/lspKeywords.ts -kanade ../kanade-docs/src/utils/kanade/lspKeywords.ts
//	go run ./cmd/lspgen -check <same flags>   # exit 1 if any file is stale
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/soumt-r/hana/lsp"
)

func main() {
	check := flag.Bool("check", false, "verify the files are up to date instead of writing them")
	hari := flag.String("hari", "", "output path for the Hari keywords")
	kanade := flag.String("kanade", "", "output path for the Kanade keywords")
	flag.Parse()

	targets := map[string]string{"hari": *hari, "kanade": *kanade}
	any, stale := false, false
	for _, lang := range lsp.LanguageKeys {
		path := targets[lang]
		if path == "" {
			continue
		}
		any = true
		want := lsp.TypeScriptKeywords(lang)
		if *check {
			got, err := os.ReadFile(path)
			if err != nil || strings.ReplaceAll(string(got), "\r\n", "\n") != want {
				fmt.Fprintf(os.Stderr, "stale: %s (run lspgen without -check)\n", path)
				stale = true
			}
			continue
		}
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("wrote", path)
	}
	if !any {
		fmt.Fprintln(os.Stderr, "usage: lspgen [-check] [-hari <out.ts>] [-kanade <out.ts>]")
		os.Exit(2)
	}
	if stale {
		os.Exit(1)
	}
}
