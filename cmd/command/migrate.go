package command

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"

	"github.com/laiambryant/tui-cardman/internal/config"
	dbpkg "github.com/laiambryant/tui-cardman/internal/db"
)

var (
	migrateDriver string
	migrateDSN    string
	migrateDir    string
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	Long:  `Apply pending database migrations from the migrations directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config.LoadConfig()
		log, err := os.Create(logFileName)
		if err != nil {
			return fmt.Errorf("create log file: %w", err)
		}
		defer log.Close()
		slog.SetDefault(slog.New(slog.NewTextHandler(log, &slog.HandlerOptions{Level: config.GetLogLevel()})))
		if migrateDSN == "" {
			migrateDSN = "file:cardman.db?_fk=1"
		}
		sqlDB, err := sql.Open(migrateDriver, migrateDSN)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer sqlDB.Close()
		if err := dbpkg.ApplyMigrations(sqlDB, migrateDir); err != nil {
			return fmt.Errorf("apply migrations: %w", err)
		}
		fmt.Println("migrations applied")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().StringVar(&migrateDriver, "driver", "sqlite3", "SQL driver name")
	migrateCmd.Flags().StringVar(&migrateDSN, "dsn", "file:cardman.db?_fk=1", "Database DSN")
	migrateCmd.Flags().StringVar(&migrateDir, "dir", "internal/db/migrations", "Migrations directory")
}
