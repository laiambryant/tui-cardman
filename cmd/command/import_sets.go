package command

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/laiambryant/tui-cardman/internal/config"
	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/pokemontcg"
	"github.com/laiambryant/tui-cardman/internal/services/cardimages"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/importruns"
	"github.com/laiambryant/tui-cardman/internal/services/prices"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
	"github.com/spf13/cobra"
)

const shortMessageImportSets = "Import specific Pokemon TCG sets by set ID"
const longMessageImportSets = `Import one or more specific Pokemon TCG sets by their set IDs.

This command will:
- Fetch only the specified sets from the Pokemon TCG API
- Import all cards from each specified set
- Store all data (images, prices, etc.) in the local database

Examples:
  cardman import-sets base1
  cardman import-sets base1 jungle fossil
  cardman import-sets swsh1 swsh2 swsh3

Set IDs are typically the lowercase set code (e.g., 'base1', 'jungle', 'swsh1').
You can find set IDs by browsing the API or running import-full with verbose logging.`

var importSetsCmd = &cobra.Command{
	Use:   "import-sets [set-id...]",
	Short: shortMessageImportSets,
	Long:  longMessageImportSets,
	Args:  cobra.MinimumNArgs(1),
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
		cardImageService := cardimages.NewCardImageService(database)
		tcgPlayerPriceService := prices.NewTCGPlayerPriceService(database)
		cardMarketPriceService := prices.NewCardMarketPriceService(database)

		importService := pokemontcg.NewImportService(
			database, client, logger,
			importRunService, setService, cardService,
			cardImageService, tcgPlayerPriceService, cardMarketPriceService,
		)

		setIDs := args
		logger.Info("Starting import of specific sets", "sets", strings.Join(setIDs, ", "))

		if err := importService.ImportSpecificSets(ctx, setIDs); err != nil {
			return fmt.Errorf("import failed: %w", err)
		}

		logger.Info("Import of specific sets completed successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(importSetsCmd)
}
