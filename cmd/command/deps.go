package command

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/laiambryant/tui-cardman/internal/config"
	"github.com/laiambryant/tui-cardman/internal/export"
	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/importers"
	cardmodel "github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/services/cardgame"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/deck"
	"github.com/laiambryant/tui-cardman/internal/services/list"
	"github.com/laiambryant/tui-cardman/internal/services/user"
	"github.com/laiambryant/tui-cardman/internal/services/usercollection"
)

type cliDeps struct {
	runtime *commandRuntime
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
	runtime, err := newCommandRuntime(true, true)
	if err != nil {
		return nil, err
	}
	database := runtime.db
	userSvc := user.NewUserService(database)
	u, err := userSvc.GetFirstUser()
	if err != nil {
		runtime.Close()
		if errors.Is(err, user.ErrNoUsersFound) {
			return nil, fmt.Errorf("no local user found — run 'cardman serve' first")
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &cliDeps{
		runtime: runtime,
		db:      database,
		userID:  u.ID,
		cardSvc: card.NewCardService(database),
		collSvc: usercollection.NewUserCollectionService(database),
		deckSvc: deck.NewDeckService(database),
		listSvc: list.NewListService(database),
		gameSvc: cardgame.NewCardGameService(database),
	}, nil
}

func (d *cliDeps) Close() {
	if d == nil {
		return
	}
	if d.runtime != nil {
		d.runtime.Close()
		return
	}
	if d.db != nil {
		_ = d.db.Close()
	}
}

func normalizeGameName(name string) string {
	if canonical, ok := gameAliases[strings.ToLower(name)]; ok {
		return canonical
	}
	return name
}

func canonicalGameName(name string) string {
	normalized := strings.TrimSpace(strings.ToLower(name))
	switch normalized {
	case "pokemon", "pokémon tcg", "pokemontcg":
		return "Pokemon"
	case "mtg", "magic", "magic: the gathering":
		return "Magic: The Gathering"
	case "yugioh", "yu-gi-oh", "yu-gi-oh!":
		return "Yu-Gi-Oh!"
	case "onepiece", "one piece":
		return "One Piece"
	default:
		return normalizeGameName(name)
	}
}

func resolveGame(gameSvc cardgame.CardGameService, gameName string) (*cardmodel.CardGame, error) {
	canonical := canonicalGameName(gameName)
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

func buildImporterForGame(database *sql.DB, gameName string, cfg config.Config) (gameimporter.GameImporter, error) {
	importer, err := importers.New(database, canonicalGameName(gameName), cfg.GetAPIKey())
	if errors.Is(err, importers.ErrUnsupportedGame) {
		return nil, fmt.Errorf("unknown game: %q (try: pokemon, mtg, yugioh, onepiece)", gameName)
	}
	return importer, err
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
