package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/config"
	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/pokemontcg"
	"github.com/spf13/cobra"
)

const shortMessageImportUpdates = "Import only new Pokemon TCG sets that don't exist in the database"
const longMessageImportUpdates = `Performs an incremental import of new Pokemon TCG sets only.

This command will:
- Fetch all sets from the Pokemon TCG API
- Compare with existing sets in the database
- Import only sets that are NOT already in the database
- Skip all existing sets to save time and API quota

This is ideal for periodic updates to catch new set releases without
re-importing existing data. Run this daily or weekly via cron.

Note: This does NOT update existing cards or refresh prices. It only
adds net-new sets.`

var importUpdatesCmd = &cobra.Command{
	Use:   "import-updates",
	Short: shortMessageImportUpdates,
	Long:  longMessageImportUpdates,
	RunE: func(cmd *cobra.Command, args []string) error {
		config.LoadConfig()
		ctx := context.Background()
		database, err := db.OpenDB(config.Cfg.DBDSN)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer database.Close()
		apiKey := config.GetAPIKey()
		client := pokemontcg.NewClient(apiKey)
		logger := slog.Default()
		importService := pokemontcg.NewImportService(database, client, logger)
		logger.Info("Starting incremental Pokemon TCG import (new sets only)")
		if err := importService.ImportNewSets(ctx); err != nil {
			return fmt.Errorf("import failed: %w", err)
		}
		logger.Info("Incremental import completed successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(importUpdatesCmd)
}
