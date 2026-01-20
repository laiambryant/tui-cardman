package command

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/spf13/cobra"

	"github.com/laiambryant/tui-cardman/internal/config"
	dbpkg "github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/tui"
)

var serveSSHCmd = &cobra.Command{
	Use:   "serve-ssh",
	Short: "Start the SSH server for remote TUI access",
	Long:  `Launch an SSH server that provides authenticated access to the card management TUI.`,
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
		hostKeyPath, err := ensureHostKey(config.Cfg.SSHHostKey)
		if err != nil {
			return fmt.Errorf("ensure host key: %w", err)
		}
		s, err := wish.NewServer(
			wish.WithAddress(fmt.Sprintf(":%d", config.Cfg.SSHPort)),
			wish.WithHostKeyPath(hostKeyPath),
			wish.WithMiddleware(
				bubbletea.Middleware(func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
					_, _, active := s.Pty()
					if !active {
						fmt.Println("no active terminal, skipping")
						return nil, nil
					}
					model, err := tui.NewModel(db, true)
					if err != nil {
						slog.Error("failed to create TUI model", "error", err)
						return nil, nil
					}
					return model, []tea.ProgramOption{
						tea.WithAltScreen(),
						tea.WithInput(s),
						tea.WithOutput(s),
					}
				}),
				logging.Middleware(),
			),
		)
		if err != nil {
			return fmt.Errorf("create SSH server: %w", err)
		}
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			slog.Info("Starting SSH server", "port", config.Cfg.SSHPort)
			fmt.Printf("SSH server starting on port %d\n", config.Cfg.SSHPort)
			fmt.Printf("Host key: %s\n", hostKeyPath)
			fmt.Println("Press Ctrl+C to stop")
			if err = s.ListenAndServe(); err != nil {
				slog.Error("SSH server error", "error", err)
			}
		}()
		<-done
		slog.Info("Stopping SSH server")
		fmt.Println("\nShutting down SSH server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown SSH server: %w", err)
		}

		return nil
	},
}

// ensureHostKey generates SSH host key if it doesn't exist
func ensureHostKey(keyPath string) (string, error) {
	if keyPath[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home dir: %w", err)
		}
		keyPath = filepath.Join(home, keyPath[2:])
	}
	if err := checkIfKeyExists(keyPath); err != nil {
		return "", err
	}
	return keyPath, nil
}

func checkIfKeyExists(keyPath string) error {
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		dir := filepath.Dir(keyPath)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("create key directory: %w", err)
		}
		fmt.Printf("Generating SSH host key at %s...\n", keyPath)
		if err := generateHostKey(keyPath); err != nil {
			return fmt.Errorf("generate host key: %w", err)
		}
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✓ Host key generated successfully"))
	}
	return nil
}

// generateHostKey creates a new ED25519 SSH host key
func generateHostKey(path string) error {
	_, err := wish.NewServer(wish.WithHostKeyPath(path))
	return err
}

func init() {
	rootCmd.AddCommand(serveSSHCmd)
}
