// stdgen writes hana/std's name tables as TypeScript modules, so the docs
// sites' browser engines expose the same native modules under the same names
// as hana. The Go tables stay the single source of truth; the generated files
// are never edited by hand.
//
//	go run ./cmd/stdgen -hari ../hari-docs/src/utils/hari/stdNames.ts -kanade ../kanade-docs/src/utils/kanade/stdNames.ts
//	go run ./cmd/stdgen -check <same flags>   # exit 1 if any file is stale
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/soumt-r/hana/std"
)

func main() {
	check := flag.Bool("check", false, "verify the files are up to date instead of writing them")
	hari := flag.String("hari", "", "output path for the Hari name table")
	kanade := flag.String("kanade", "", "output path for the Kanade name table")
	flag.Parse()

	targets := map[string]string{std.Hari: *hari, std.Kanade: *kanade}
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
			if err != nil || strings.ReplaceAll(string(got), "\r\n", "\n") != want {
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
		fmt.Fprintln(os.Stderr, "usage: stdgen [-check] [-hari <out.ts>] [-kanade <out.ts>]")
		os.Exit(2)
	}
	if stale {
		os.Exit(1)
	}
}
