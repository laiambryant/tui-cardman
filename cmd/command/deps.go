package command

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

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

// cliDeps holds the services used by the data commands, scoped to the local user.
type cliDeps struct {
	runtime *commandRuntime
	userID  int64
	cardSvc card.CardService
	collSvc usercollection.UserCollectionService
	deckSvc deck.DeckService
	listSvc list.ListService
	gameSvc cardgame.CardGameService
}

// withDeps opens the database (applying migrations), resolves the local user
// and runs fn.
func withDeps(fn func(d *cliDeps) error) error {
	runtime, err := newCommandRuntime(true, true)
	if err != nil {
		return err
	}
	defer runtime.Close()
	u, err := user.NewUserService(runtime.db).GetFirstUser()
	if errors.Is(err, user.ErrNoUsersFound) {
		return errors.New("no local user found — run 'cardman serve' first")
	}
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	database := runtime.db
	return fn(&cliDeps{
		runtime: runtime,
		userID:  u.ID,
		cardSvc: card.NewCardService(database),
		collSvc: usercollection.NewUserCollectionService(database),
		deckSvc: deck.NewDeckService(database),
		listSvc: list.NewListService(database),
		gameSvc: cardgame.NewCardGameService(database),
	})
}

// withImporter opens the database (applying migrations) and runs fn with the
// importer for the named game.
func withImporter(gameName string, fn func(importer gameimporter.GameImporter) error) error {
	runtime, err := newCommandRuntime(true, true)
	if err != nil {
		return err
	}
	defer runtime.Close()
	importer, err := importers.New(runtime.db, canonicalGameName(gameName), runtime.config.GetAPIKey())
	if errors.Is(err, importers.ErrUnsupportedGame) {
		return fmt.Errorf("unknown game: %q (try: pokemon, mtg, yugioh, onepiece)", gameName)
	}
	if err != nil {
		return err
	}
	return fn(importer)
}

var gameAliases = map[string]string{
	"pokemon":              "Pokemon",
	"pokémon tcg":          "Pokemon",
	"pokemontcg":           "Pokemon",
	"mtg":                  "Magic: The Gathering",
	"magic":                "Magic: The Gathering",
	"magic: the gathering": "Magic: The Gathering",
	"yugioh":               "Yu-Gi-Oh!",
	"yu-gi-oh":             "Yu-Gi-Oh!",
	"yu-gi-oh!":            "Yu-Gi-Oh!",
	"onepiece":             "One Piece",
	"one piece":            "One Piece",
}

// canonicalGameName maps a user-supplied game alias to its card_games name.
func canonicalGameName(name string) string {
	if canonical, ok := gameAliases[strings.ToLower(strings.TrimSpace(name))]; ok {
		return canonical
	}
	return name
}

func (d *cliDeps) resolveGame(gameName string) (*cardmodel.CardGame, error) {
	game, err := d.gameSvc.GetCardGameByName(canonicalGameName(gameName))
	if err != nil {
		return nil, fmt.Errorf("game %q not found (try: pokemon, mtg, yugioh, onepiece): %w", gameName, err)
	}
	return game, nil
}

// findCard looks up a card of the given game by set code and number.
func (d *cliDeps) findCard(gameID int64, setCode, number string) (*cardmodel.Card, error) {
	c, err := d.cardSvc.GetCardByGameAndSetCodeAndNumber(gameID, setCode, number)
	if err != nil {
		return nil, fmt.Errorf("card not found: %w", err)
	}
	if c == nil {
		return nil, fmt.Errorf("card not found: %s %s", setCode, number)
	}
	return c, nil
}

// cardRows returns export rows for the cards of a game with a positive
// quantity, in the card listing order (by name).
func (d *cliDeps) cardRows(gameID int64, quantities map[int64]int) ([]export.CardRow, error) {
	cards, err := d.cardSvc.GetCardsByGameID(gameID)
	if err != nil {
		return nil, err
	}
	rows := make([]export.CardRow, 0, len(quantities))
	for _, c := range cards {
		if qty := quantities[c.ID]; qty > 0 {
			rows = append(rows, toCardRow(c, qty))
		}
	}
	return rows, nil
}

func toCardRow(c cardmodel.Card, quantity int) export.CardRow {
	row := export.CardRow{Name: c.Name, Number: c.Number, Rarity: c.Rarity, Quantity: quantity, Category: c.Category}
	if c.Set != nil {
		row.SetName = c.Set.Name
		row.SetCode = c.Set.Code
	}
	return row
}

func parseID(kind, s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s ID: %s", kind, s)
	}
	return id, nil
}

func parseQuantity(s string) (int, error) {
	qty, err := strconv.Atoi(s)
	if err != nil || qty <= 0 {
		return 0, fmt.Errorf("invalid quantity: %s (must be a positive integer)", s)
	}
	return qty, nil
}

func writeTabular(w io.Writer, headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	tw.Flush()
}

// printCardRows prints card rows as a NAME/SET/NUMBER/RARITY/QTY table.
func printCardRows(rows []export.CardRow) {
	table := make([][]string, len(rows))
	for i, r := range rows {
		table[i] = []string{r.Name, r.SetCode, r.Number, r.Rarity, strconv.Itoa(r.Quantity)}
	}
	writeTabular(os.Stdout, []string{"NAME", "SET", "NUMBER", "RARITY", "QTY"}, table)
}

var exportExtensions = map[string]string{"csv": "csv", "text": "txt", "ptcgo": "ptcgo"}

// exportRows writes rows in the given format (csv, text or ptcgo) to outputFlag,
// or to a generated file name when outputFlag is empty.
func exportRows(rows []export.CardRow, format, outputFlag, exportType, name, title string) error {
	ext, ok := exportExtensions[format]
	if !ok {
		return fmt.Errorf("unknown export format %q (use csv, text or ptcgo)", format)
	}
	outFile := outputFlag
	if outFile == "" {
		outFile = export.GenerateFilename(exportType, name, ext)
	}
	var err error
	switch format {
	case "text":
		err = export.ToText(rows, outFile)
	case "ptcgo":
		err = export.ToPTCGO(title, rows, outFile)
	default:
		err = export.ToCSV(rows, outFile)
	}
	if err != nil {
		return err
	}
	fmt.Printf("Exported %d cards to %s\n", len(rows), outFile)
	return nil
}

func addExportFlags(cmd *cobra.Command) {
	cmd.Flags().String("format", "csv", "Export format: csv, text, ptcgo")
	cmd.Flags().String("output", "", "Output file path (auto-generated if omitted)")
}
