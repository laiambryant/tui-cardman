package command

import (
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

		if err := dbpkg.ApplyMigrations(sqlDB, migrateDir); err != nil {
			return fmt.Errorf("apply migrations: %w", err)
		}
		fmt.Println("migrations applied")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().StringVar(&migrateDSN, "dsn", "file:cardman.db?_fk=1", "Database DSN")
	migrateCmd.Flags().StringVar(&migrateDir, "dir", "internal/db/migrations", "Migrations directory")
}
