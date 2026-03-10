package command

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/config"
	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/pokemontcg"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/importruns"
	"github.com/laiambryant/tui-cardman/internal/services/pokemoncard"
	"github.com/laiambryant/tui-cardman/internal/services/prices"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
)

// importDeps holds the database connection and fully-wired import service.
// Callers are responsible for calling database.Close() when done.
type importDeps struct {
	database      *sql.DB
	importService *pokemontcg.ImportService
	logger        *slog.Logger
}

// buildImportService opens the database and wires up the shared import service
// used by import-full, import-sets, and import-updates.
func buildImportService() (*importDeps, error) {
	database, err := db.OpenDB(config.Cfg.DBDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	apiKey := config.GetAPIKey()
	client := pokemontcg.NewClient(apiKey)
	logger := slog.Default()

	importRunService := importruns.NewImportRunService(database)
	setService := sets.NewSetService(database)
	cardService := card.NewCardService(database)
	tcgPlayerPriceService := prices.NewTCGPlayerPriceService(database)
	cardMarketPriceService := prices.NewCardMarketPriceService(database)

	pokemonCardService := pokemoncard.NewPokemonCardService(database)
	importService := pokemontcg.NewImportService(
		database, client, logger,
		importRunService, setService, cardService,
		tcgPlayerPriceService, cardMarketPriceService,
		pokemonCardService,
	)

	return &importDeps{
		database:      database,
		importService: importService,
		logger:        logger,
	}, nil
}
