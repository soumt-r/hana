package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// The texts (Short, Long, flag descriptions) are set by applyLocale in the
// language the command line speaks; see i18n.go.
var rootCmd = &cobra.Command{
	Use: "hana",
}

func init() {
	registerLocaleFlag(rootCmd)
}

func Execute() {
	uiLocale = detectUILocale(os.Args[1:])
	// cobra's own `completion` command has English-only texts of its own; nothing
	// here documents it.
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	applyLocale()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
