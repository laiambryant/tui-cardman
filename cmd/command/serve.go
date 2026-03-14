package command

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/laiambryant/tui-cardman/internal/config"
	dbpkg "github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/tui"
)

const logFileName = "output.log"

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the TUI application",
	Long:  `Launch the interactive terminal UI for card management.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config.LoadConfig()
		logPath, err := resolveLogPath(logFileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: could not resolve log path: %v\n", err)
			logPath = logFileName
		}
		fmt.Fprintf(os.Stderr, "log file: %s\n", logPath)
		log, err := os.Create(logPath)
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
		if err := dbpkg.ApplyMigrations(db, "internal/db/migrations"); err != nil {
			return fmt.Errorf("apply migrations: %w", err)
		}
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

func resolveLogPath(name string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return name, err
	}
	return filepath.Join(filepath.Dir(exe), name), nil
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
