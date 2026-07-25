package command

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/laiambryant/tui-cardman/internal/tui"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the TUI application",
	Long:  `Launch the interactive terminal UI for card management.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := newCommandRuntime(true, true)
		if err != nil {
			return err
		}
		defer runtime.Close()

		if runtime.logPath != "" {
			fmt.Fprintf(cmd.ErrOrStderr(), "log file: %s\n", runtime.logPath)
		}

		model, err := tui.NewModel(runtime.db, false)
		if err != nil {
			return fmt.Errorf("create TUI model: %w", err)
		}
		if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
			return fmt.Errorf("run TUI: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
