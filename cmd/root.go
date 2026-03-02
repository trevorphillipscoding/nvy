// Package cmd implements the nvy command-line interface.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set during build with -ldflags "-X github.com/trevorphillipscoding/nvy/cmd.Version=..."
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "nvy",
	Short: "nvy — minimalist runtime version manager",
	Long: `nvy is a minimalist, plugin-driven runtime version manager.

It downloads, verifies, and manages multiple versions of language runtimes
(Go, Node.js, and more) without relying on any external version manager.

Add ~/.nvy/shims to your PATH to use managed runtimes:
  export PATH="$HOME/.nvy/shims:$PATH"`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       Version,
}

// Execute is the entry point called by main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(globalCmd)
	rootCmd.AddCommand(localCmd)
	rootCmd.AddCommand(listCmd)
}
