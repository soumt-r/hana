// errsgen writes the errs catalogs as a TypeScript module, so the docs sites'
// browser engines (haja-docs, kanade-docs) print exactly the wording the Go
// engines do without anyone re-typing it. The Go catalogs in hana/errs stay
// the single source of truth; the generated file is never edited by hand.
//
//	go run ./cmd/errsgen ../haja-docs/src/utils/haja/errCatalog.ts ../kanade-docs/src/utils/kanade/errCatalog.ts
//	go run ./cmd/errsgen -check <same paths>   # exit 1 if any file is stale
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/soumt-r/hana/errs"
)

func main() {
	args := os.Args[1:]
	check := false
	if len(args) > 0 && args[0] == "-check" {
		check = true
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: errsgen [-check] <out.ts>...")
		os.Exit(2)
	}
	want := errs.TypeScriptCatalog()
	stale := false
	for _, path := range args {
		if check {
			got, err := os.ReadFile(path)
			if err != nil || strings.ReplaceAll(string(got), "\r\n", "\n") != want {
				fmt.Fprintf(os.Stderr, "stale: %s (run errsgen without -check)\n", path)
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
	if stale {
		os.Exit(1)
	}
}
