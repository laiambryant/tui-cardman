// Package command provides Cobra CLI commands for the tui-cardman application.
package command

import (
	goversion "github.com/caarlos0/go-version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cardman",
	Short: "TUI card management application",
	Long:  `A terminal-based card management application with database migrations and interactive UI.`,
}

func Execute(version goversion.Info) error {
	rootCmd.Version = version.String()
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	// Runtime errors are not usage errors; don't print the help text for them.
	rootCmd.SilenceUsage = true
	rootCmd.SetVersionTemplate("{{.Version}}\n")
}
