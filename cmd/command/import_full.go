package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/config"
	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/pokemontcg"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/importruns"
	"github.com/laiambryant/tui-cardman/internal/services/prices"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
	"github.com/spf13/cobra"
)

var importFullCmd = &cobra.Command{
	Use:   "import-full",
	Short: shortMessageImportAll,
	Long:  longMessageImportAll,
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

		// Initialize all services
		importRunService := importruns.NewImportRunService(database)
		setService := sets.NewSetService(database)
		cardService := card.NewCardService(database)
		tcgPlayerPriceService := prices.NewTCGPlayerPriceService(database)
		cardMarketPriceService := prices.NewCardMarketPriceService(database)

		importService := pokemontcg.NewImportService(
			database, client, logger,
			importRunService, setService, cardService,
			tcgPlayerPriceService, cardMarketPriceService,
		)

		logger.Info("Starting full Pokemon TCG import")
		if err := importService.ImportAllSets(ctx); err != nil {
			return fmt.Errorf("import failed: %w", err)
		}
		logger.Info("Full import completed successfully")
		return nil
	},
}

const shortMessageImportAll = "Import all Pokemon TCG sets and cards from the API"
const longMessageImportAll = `Performs a complete import of all Pokemon TCG sets and cards from the API.

This command will:
- Fetch all sets from the Pokemon TCG API
- For each set, import all cards with complete data (images, prices, etc.)
- Store all data in the local database

Note: This can take a significant amount of time and will use API quota.
Use 'import-updates' for incremental imports of new sets only.`

func init() {
	rootCmd.AddCommand(importFullCmd)
}
