package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/spf13/cobra"

	"github.com/laiambryant/tui-cardman/internal/tui"
)

var serveSSHCmd = &cobra.Command{
	Use:   "serve-ssh",
	Short: "Start the SSH server for remote TUI access",
	Long:  `Launch an SSH server that provides authenticated access to the card management TUI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := newCommandRuntime(true, true)
		if err != nil {
			return err
		}
		defer runtime.Close()

		hostKeyPath, err := ensureHostKey(runtime.config.SSHHostKey)
		if err != nil {
			return fmt.Errorf("ensure host key: %w", err)
		}
		s, err := wish.NewServer(
			wish.WithAddress(fmt.Sprintf(":%d", runtime.config.SSHPort)),
			wish.WithHostKeyPath(hostKeyPath),
			wish.WithMiddleware(
				bubbletea.Middleware(func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
					_, _, active := s.Pty()
					if !active {
						wish.Fatalln(s, "cardman requires an interactive terminal (try ssh -t)")
						return nil, nil
					}
					model, err := tui.NewModel(runtime.db, true)
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
		signal.Notify(done, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(done)
		serverErr := make(chan error, 1)
		go func() {
			slog.Info("Starting SSH server", "port", runtime.config.SSHPort)
			fmt.Printf("SSH server starting on port %d\n", runtime.config.SSHPort)
			fmt.Printf("Host key: %s\n", hostKeyPath)
			fmt.Println("Press Ctrl+C to stop")
			err := s.ListenAndServe()
			if errors.Is(err, ssh.ErrServerClosed) {
				err = nil
			}
			serverErr <- err
		}()

		select {
		case err := <-serverErr:
			if err != nil {
				slog.Error("SSH server error", "error", err)
				return fmt.Errorf("serve SSH: %w", err)
			}
			return nil
		case <-done:
			slog.Info("Stopping SSH server")
			fmt.Println("\nShutting down SSH server...")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.Shutdown(ctx); err != nil {
				return fmt.Errorf("shutdown SSH server: %w", err)
			}
		}

		return nil
	},
}

// ensureHostKey expands a leading "~/" in keyPath and generates an ED25519
// host key there if none exists yet.
func ensureHostKey(keyPath string) (string, error) {
	if keyPath == "" {
		return "", fmt.Errorf("SSH_HOST_KEY must not be empty")
	}
	if rest, ok := strings.CutPrefix(keyPath, "~/"); ok {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home dir: %w", err)
		}
		keyPath = filepath.Join(home, rest)
	}
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		return keyPath, nil
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return "", fmt.Errorf("create key directory: %w", err)
	}
	fmt.Printf("Generating SSH host key at %s...\n", keyPath)
	// wish creates the key file when the configured host key path does not exist.
	if _, err := wish.NewServer(wish.WithHostKeyPath(keyPath)); err != nil {
		return "", fmt.Errorf("generate host key: %w", err)
	}
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("Host key generated successfully"))
	return keyPath, nil
}

func init() {
	rootCmd.AddCommand(serveSSHCmd)
}
