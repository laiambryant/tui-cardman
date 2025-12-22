package command

import (
	"fmt"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"gihtub.com/laiambryant/tui-cardman/internal/config"
	dbpkg "gihtub.com/laiambryant/tui-cardman/internal/db"
	"gihtub.com/laiambryant/tui-cardman/internal/tui"
)

const LOG_FILE_NAME = "output.log"

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the TUI application",
	Long:  `Launch the interactive terminal UI for card management.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config.LoadConfig()
		log, err := os.Create(LOG_FILE_NAME)
		if err != nil {
			return fmt.Errorf("create log file: %w", err)
		}
		defer log.Close()
		slog.SetDefault(slog.New(slog.NewTextHandler(log, &slog.HandlerOptions{Level: config.GetLogLevel()})))
		db, err := dbpkg.OpenDB(config.Cfg.DBDSN)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer db.Close()
		model, err := tui.NewModel(db, false)
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
