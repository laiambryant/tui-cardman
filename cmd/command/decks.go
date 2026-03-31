package command

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/laiambryant/tui-cardman/internal/export"
	cardmodel "github.com/laiambryant/tui-cardman/internal/model"
)

var decksCmd = &cobra.Command{
	Use:   "decks",
	Short: "Manage your decks",
}

var decksListCmd = &cobra.Command{
	Use:   "list <game>",
	Short: "List all decks for a game",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		deps, err := buildCLIDeps()
		if err != nil {
			return err
		}
		defer deps.db.Close()
		game, err := resolveGame(deps.gameSvc, args[0])
		if err != nil {
			return err
		}
		decks, err := deps.deckSvc.GetDecksByUserAndGame(deps.userID, game.ID)
		if err != nil {
			return err
		}
		rows := make([][]string, len(decks))
		for i, d := range decks {
			rows[i] = []string{strconv.FormatInt(d.ID, 10), d.Name, d.Format}
		}
		writeTabular(os.Stdout, []string{"ID", "NAME", "FORMAT"}, rows)
		return nil
	},
}

var decksCreateCmd = &cobra.Command{
	Use:   "create <game> <name>",
	Short: "Create a new deck",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		deps, err := buildCLIDeps()
		if err != nil {
			return err
		}
		defer deps.db.Close()
		game, err := resolveGame(deps.gameSvc, args[0])
		if err != nil {
			return err
		}
		d, err := deps.deckSvc.CreateDeck(context.Background(), deps.userID, game.ID, args[1], format)
		if err != nil {
			return err
		}
		fmt.Printf("Created deck %q (ID: %d)\n", d.Name, d.ID)
		return nil
	},
}

var decksDeleteCmd = &cobra.Command{
	Use:   "delete <deck-id>",
	Short: "Delete a deck",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		deckID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid deck ID: %s", args[0])
		}
		deps, err := buildCLIDeps()
		if err != nil {
			return err
		}
		defer deps.db.Close()
		if err := deps.deckSvc.DeleteDeck(context.Background(), deckID); err != nil {
			return err
		}
		fmt.Printf("Deleted deck %d\n", deckID)
		return nil
	},
}

var decksShowCmd = &cobra.Command{
	Use:   "show <deck-id>",
	Short: "Show cards in a deck",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		deckID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid deck ID: %s", args[0])
		}
		deps, err := buildCLIDeps()
		if err != nil {
			return err
		}
		defer deps.db.Close()
		d, err := deps.deckSvc.GetDeckByID(deckID)
		if err != nil {
			return err
		}
		game, err := resolveGameByID(deps.gameSvc, d.CardGameID)
		if err != nil {
			return err
		}
		quantities, err := deps.deckSvc.GetAllQuantitiesForDeck(deckID)
		if err != nil {
			return err
		}
		allCards, err := deps.cardSvc.GetCardsByGameID(game.ID)
		if err != nil {
			return err
		}
		cardsByID := buildCardsByIDMap(allCards)
		rows := make([][]string, 0, len(quantities))
		for cardID, qty := range quantities {
			c, ok := cardsByID[cardID]
			if !ok {
				continue
			}
			setCode := ""
			if c.Set != nil {
				setCode = c.Set.Code
			}
			rows = append(rows, []string{c.Name, setCode, c.Number, c.Rarity, strconv.Itoa(qty)})
		}
		fmt.Printf("Deck: %s (Format: %s)\n", d.Name, d.Format)
		writeTabular(os.Stdout, []string{"NAME", "SET", "NUMBER", "RARITY", "QTY"}, rows)
		return nil
	},
}

var decksAddCardCmd = &cobra.Command{
	Use:   "add-card <deck-id> <set-code> <number> <quantity>",
	Short: "Add a card to a deck",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		deckID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid deck ID: %s", args[0])
		}
		qty, err := strconv.Atoi(args[3])
		if err != nil {
			return fmt.Errorf("invalid quantity: %s", args[3])
		}
		deps, err := buildCLIDeps()
		if err != nil {
			return err
		}
		defer deps.db.Close()
		c, err := deps.cardSvc.GetCardBySetCodeAndNumber(args[1], args[2])
		if err != nil {
			return fmt.Errorf("card not found: %w", err)
		}
		if err := deps.deckSvc.UpsertDeckCardBatch(context.Background(), deckID, map[int64]int{c.ID: qty}); err != nil {
			return err
		}
		fmt.Printf("Added %d × %s to deck\n", qty, c.Name)
		return nil
	},
}

var decksRemoveCardCmd = &cobra.Command{
	Use:   "remove-card <deck-id> <set-code> <number>",
	Short: "Remove a card from a deck",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		deckID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid deck ID: %s", args[0])
		}
		deps, err := buildCLIDeps()
		if err != nil {
			return err
		}
		defer deps.db.Close()
		c, err := deps.cardSvc.GetCardBySetCodeAndNumber(args[1], args[2])
		if err != nil {
			return fmt.Errorf("card not found: %w", err)
		}
		if err := deps.deckSvc.UpsertDeckCardBatch(context.Background(), deckID, map[int64]int{c.ID: 0}); err != nil {
			return err
		}
		fmt.Printf("Removed %s from deck\n", c.Name)
		return nil
	},
}

var decksValidateCmd = &cobra.Command{
	Use:   "validate <deck-id>",
	Short: "Validate a deck against game rules",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		deckID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid deck ID: %s", args[0])
		}
		deps, err := buildCLIDeps()
		if err != nil {
			return err
		}
		defer deps.db.Close()
		d, err := deps.deckSvc.GetDeckByID(deckID)
		if err != nil {
			return err
		}
		game, err := resolveGameByID(deps.gameSvc, d.CardGameID)
		if err != nil {
			return err
		}
		quantities, err := deps.deckSvc.GetAllQuantitiesForDeck(deckID)
		if err != nil {
			return err
		}
		allCards, err := deps.cardSvc.GetCardsByGameID(game.ID)
		if err != nil {
			return err
		}
		cardsByID := buildCardsByIDMap(allCards)
		deckCards := make([]cardmodel.Card, 0, len(quantities))
		for cardID, qty := range quantities {
			if c, ok := cardsByID[cardID]; ok && qty > 0 {
				deckCards = append(deckCards, c)
			}
		}
		errs := deps.deckSvc.ValidateDeck(deckCards, quantities, game.Name)
		if len(errs) == 0 {
			fmt.Println("Deck is valid.")
			return nil
		}
		for _, e := range errs {
			fmt.Printf("[%s] %s\n", e.Type, e.Message)
		}
		return fmt.Errorf("deck has %d validation error(s)", len(errs))
	},
}

var decksExportCmd = &cobra.Command{
	Use:   "export <deck-id>",
	Short: "Export a deck to a file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		deckID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid deck ID: %s", args[0])
		}
		format, _ := cmd.Flags().GetString("format")
		outputFlag, _ := cmd.Flags().GetString("output")
		deps, err := buildCLIDeps()
		if err != nil {
			return err
		}
		defer deps.db.Close()
		d, err := deps.deckSvc.GetDeckByID(deckID)
		if err != nil {
			return err
		}
		game, err := resolveGameByID(deps.gameSvc, d.CardGameID)
		if err != nil {
			return err
		}
		quantities, err := deps.deckSvc.GetAllQuantitiesForDeck(deckID)
		if err != nil {
			return err
		}
		allCards, err := deps.cardSvc.GetCardsByGameID(game.ID)
		if err != nil {
			return err
		}
		cardsByID := buildCardsByIDMap(allCards)
		cardRows := quantitiesToCardRows(quantities, cardsByID)
		outFile := resolveOutputFile(outputFlag, "deck", d.Name, format)
		switch format {
		case "text":
			err = export.ToText(cardRows, outFile)
		case "ptcgo":
			err = export.ToPTCGO(d.Name, cardRows, outFile)
		default:
			err = export.ToCSV(cardRows, outFile)
		}
		if err != nil {
			return err
		}
		fmt.Printf("Exported %d cards to %s\n", len(cardRows), outFile)
		return nil
	},
}

func init() {
	decksCreateCmd.Flags().String("format", "", "Deck format (e.g. standard)")
	decksExportCmd.Flags().String("format", "csv", "Export format: csv, text, ptcgo")
	decksExportCmd.Flags().String("output", "", "Output file path (auto-generated if omitted)")
	decksCmd.AddCommand(decksListCmd, decksCreateCmd, decksDeleteCmd, decksShowCmd, decksAddCardCmd, decksRemoveCardCmd, decksValidateCmd, decksExportCmd)
	rootCmd.AddCommand(decksCmd)
}
