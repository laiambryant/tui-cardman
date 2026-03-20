package command

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"text/tabwriter"

	"github.com/laiambryant/tui-cardman/internal/config"
	dbpkg "github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/export"
	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	cardmodel "github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/mtg"
	"github.com/laiambryant/tui-cardman/internal/onepiece"
	"github.com/laiambryant/tui-cardman/internal/pokemontcg"
	"github.com/laiambryant/tui-cardman/internal/services/cardgame"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/deck"
	"github.com/laiambryant/tui-cardman/internal/services/importruns"
	"github.com/laiambryant/tui-cardman/internal/services/list"
	"github.com/laiambryant/tui-cardman/internal/services/mtgcard"
	"github.com/laiambryant/tui-cardman/internal/services/onepiececard"
	"github.com/laiambryant/tui-cardman/internal/services/pokemoncard"
	"github.com/laiambryant/tui-cardman/internal/services/prices"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
	"github.com/laiambryant/tui-cardman/internal/services/user"
	"github.com/laiambryant/tui-cardman/internal/services/usercollection"
	"github.com/laiambryant/tui-cardman/internal/services/yugiohcard"
	"github.com/laiambryant/tui-cardman/internal/yugioh"
)

type cliDeps struct {
	db      *sql.DB
	userID  int64
	cardSvc card.CardService
	collSvc usercollection.UserCollectionService
	deckSvc deck.DeckService
	listSvc list.ListService
	gameSvc cardgame.CardGameService
}

var gameAliases = map[string]string{
	"pokemon":  "Pokémon TCG",
	"mtg":      "Magic: The Gathering",
	"yugioh":   "Yu-Gi-Oh!",
	"onepiece": "One Piece",
}

func buildCLIDeps() (*cliDeps, error) {
	config.LoadConfig()
	database, err := dbpkg.OpenDB(config.Cfg.DBDSN)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := dbpkg.ApplyMigrations(database, "internal/db/migrations"); err != nil {
		database.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}
	userSvc := user.NewUserService(database)
	u, err := userSvc.GetFirstUser()
	if err != nil {
		database.Close()
		if errors.Is(err, user.ErrNoUsersFound) {
			return nil, fmt.Errorf("no local user found — run 'cardman serve' first")
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &cliDeps{
		db:      database,
		userID:  u.ID,
		cardSvc: card.NewCardService(database),
		collSvc: usercollection.NewUserCollectionService(database),
		deckSvc: deck.NewDeckService(database),
		listSvc: list.NewListService(database),
		gameSvc: cardgame.NewCardGameService(database),
	}, nil
}

func normalizeGameName(name string) string {
	if canonical, ok := gameAliases[strings.ToLower(name)]; ok {
		return canonical
	}
	return name
}

func resolveGame(gameSvc cardgame.CardGameService, gameName string) (*cardmodel.CardGame, error) {
	canonical := normalizeGameName(gameName)
	game, err := gameSvc.GetCardGameByName(canonical)
	if err != nil {
		return nil, fmt.Errorf("game %q not found (try: pokemon, mtg, yugioh, onepiece): %w", gameName, err)
	}
	return game, nil
}

func resolveGameByID(gameSvc cardgame.CardGameService, gameID int64) (*cardmodel.CardGame, error) {
	games, err := gameSvc.GetAllCardGames()
	if err != nil {
		return nil, err
	}
	for _, g := range games {
		if g.ID == gameID {
			return &g, nil
		}
	}
	return nil, fmt.Errorf("game with ID %d not found", gameID)
}

func buildImporterForGame(database *sql.DB, gameName string) (gameimporter.GameImporter, error) {
	normalized := normalizeGameName(gameName)
	logger := slog.Default()
	importRunSvc := importruns.NewImportRunService(database)
	setSvc := sets.NewSetService(database)
	cardSvc := card.NewCardService(database)
	switch normalized {
	case "Pokémon TCG":
		client := pokemontcg.NewClient(config.GetAPIKey())
		tcgSvc := prices.NewTCGPlayerPriceService(database)
		cmSvc := prices.NewCardMarketPriceService(database)
		pokSvc := pokemoncard.NewPokemonCardService(database)
		svc := pokemontcg.NewImportService(database, client, logger, importRunSvc, setSvc, cardSvc, tcgSvc, cmSvc, pokSvc)
		return pokemontcg.NewPokemonGameImporter(client, svc, setSvc), nil
	case "Magic: The Gathering":
		client := mtg.NewClient()
		mtgSvc := mtgcard.NewMTGCardService(database)
		svc := mtg.NewImportService(database, client, logger, importRunSvc, setSvc, cardSvc, mtgSvc)
		return mtg.NewMTGGameImporter(client, svc, setSvc), nil
	case "Yu-Gi-Oh!":
		client := yugioh.NewClient()
		ygoSvc := yugiohcard.NewYuGiOhCardService(database)
		svc := yugioh.NewImportService(database, client, logger, importRunSvc, setSvc, cardSvc, ygoSvc)
		return yugioh.NewYuGiOhGameImporter(client, svc, setSvc), nil
	case "One Piece":
		client := onepiece.NewClient()
		opSvc := onepiececard.NewOnePieceCardService(database)
		svc := onepiece.NewImportService(database, client, logger, importRunSvc, setSvc, cardSvc, opSvc)
		return onepiece.NewOnePieceGameImporter(client, svc, setSvc), nil
	default:
		return nil, fmt.Errorf("unknown game: %q (try: pokemon, mtg, yugioh, onepiece)", gameName)
	}
}

func writeTabular(w io.Writer, headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	tw.Flush()
}

func resolveOutputFile(outputFlag, exportType, name, format string) string {
	if outputFlag != "" {
		return outputFlag
	}
	return export.GenerateFilename(exportType, name, format)
}

func buildCardsByIDMap(cards []cardmodel.Card) map[int64]cardmodel.Card {
	m := make(map[int64]cardmodel.Card, len(cards))
	for _, c := range cards {
		m[c.ID] = c
	}
	return m
}

func quantitiesToCardRows(quantities map[int64]int, cardsByID map[int64]cardmodel.Card) []export.CardRow {
	rows := make([]export.CardRow, 0, len(quantities))
	for cardID, qty := range quantities {
		c, ok := cardsByID[cardID]
		if !ok || qty == 0 {
			continue
		}
		setName, setCode := "", ""
		if c.Set != nil {
			setName = c.Set.Name
			setCode = c.Set.Code
		}
		rows = append(rows, export.CardRow{
			Name:     c.Name,
			SetName:  setName,
			SetCode:  setCode,
			Number:   c.Number,
			Rarity:   c.Rarity,
			Quantity: qty,
		})
	}
	return rows
}
