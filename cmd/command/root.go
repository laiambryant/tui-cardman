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
	rootCmd.SetVersionTemplate("{{.Version}}\n")
}
