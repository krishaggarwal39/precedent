package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "precedent",
	Short: "Precedent - Benchmark coding AI agents on your own codebase",
	Long: `Precedent lets you benchmark agents like Claude Code, Aider, and Codex
against the real commit history of your private repositories using a fail-to-pass methodology.`,
}

// SetVersion is called from main to wire ldflags
func SetVersion(v string) {
	rootCmd.Version = v
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
