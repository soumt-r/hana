// stdgen writes hana/std's name tables as TypeScript modules, so the docs
// sites' browser engines expose the same native modules under the same names
// as hana. The Go tables stay the single source of truth; the generated files
// are never edited by hand.
//
//	go run ./cmd/stdgen -haja ../haja-docs/src/utils/haja/stdNames.ts -kanade ../kanade-docs/src/utils/kanade/stdNames.ts
//	go run ./cmd/stdgen -check <same flags>   # exit 1 if any file is stale
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/soumt-r/hana/std"
)

func main() {
	check := flag.Bool("check", false, "verify the files are up to date instead of writing them")
	haja := flag.String("haja", "", "output path for the Haja name table")
	kanade := flag.String("kanade", "", "output path for the Kanade name table")
	flag.Parse()

	targets := map[string]string{std.Haja: *haja, std.Kanade: *kanade}
	any := false
	stale := false
	for _, lang := range std.Languages {
		path := targets[lang]
		if path == "" {
			continue
		}
		any = true
		want := std.TypeScriptNames(lang)
		if *check {
			got, err := os.ReadFile(path)
			if err != nil || string(got) != want {
				fmt.Fprintf(os.Stderr, "stale: %s (run stdgen without -check)\n", path)
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
		fmt.Fprintln(os.Stderr, "usage: stdgen [-check] [-haja <out.ts>] [-kanade <out.ts>]")
		os.Exit(2)
	}
	if stale {
		os.Exit(1)
	}
}
