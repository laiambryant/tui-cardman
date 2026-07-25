package command

import (
	"database/sql"
	"fmt"

	"github.com/spf13/cobra"

	dbpkg "github.com/laiambryant/tui-cardman/internal/db"
)

var (
	migrateDSN string
	migrateDir string
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	Long:  `Apply pending database migrations from the migrations directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := newCommandRuntime(false, false)
		if err != nil {
			return err
		}
		defer runtime.Close()

		dsn := migrateDSN
		if !cmd.Flags().Changed("dsn") {
			dsn = runtime.config.DBDSN
		}

		sqlDB, err := dbpkg.OpenDB(dsn)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer sqlDB.Close()

		if err := applyMigrationsFromFlag(cmd, sqlDB); err != nil {
			return fmt.Errorf("apply migrations: %w", err)
		}
		fmt.Println("migrations applied")
		return nil
	},
}

func applyMigrationsFromFlag(cmd *cobra.Command, sqlDB *sql.DB) error {
	if cmd.Flags().Changed("dir") {
		return dbpkg.ApplyMigrations(sqlDB, migrateDir)
	}
	return dbpkg.ApplyEmbeddedMigrations(sqlDB)
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().StringVar(&migrateDSN, "dsn", "file:cardman.db?_fk=1", "Database DSN")
	migrateCmd.Flags().StringVar(&migrateDir, "dir", "", "Apply migrations from a directory instead of the embedded ones")
}
