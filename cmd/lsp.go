package cmd

import (
	"os"

	"github.com/soumt-r/hana/lsp"
	"github.com/spf13/cobra"
)

var lspCmd = &cobra.Command{
	Use:  "lsp",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// stdout is the protocol channel: nothing else may print to it.
		cmd.SilenceUsage = true
		return lsp.NewServer(os.Stdin, os.Stdout).Serve()
	},
}

func init() {
	rootCmd.AddCommand(lspCmd)
}
