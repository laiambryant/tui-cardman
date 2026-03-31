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

var collectionCmd = &cobra.Command{
	Use:   "collection",
	Short: "Manage your card collection",
}

var collectionListCmd = &cobra.Command{
	Use:   "list <game>",
	Short: "List your collection for a game",
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
		items, err := deps.collSvc.GetUserCollectionByGameID(deps.userID, game.ID)
		if err != nil {
			return err
		}
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			if item.Card == nil {
				continue
			}
			setCode := ""
			if item.Card.Set != nil {
				setCode = item.Card.Set.Code
			}
			rows = append(rows, []string{item.Card.Name, setCode, item.Card.Number, item.Card.Rarity, strconv.Itoa(item.Quantity)})
		}
		writeTabular(os.Stdout, []string{"NAME", "SET", "NUMBER", "RARITY", "QTY"}, rows)
		return nil
	},
}

var collectionAddCmd = &cobra.Command{
	Use:   "add <game> <set-code> <number> <quantity>",
	Short: "Add cards to your collection",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
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
		existing, err := deps.collSvc.GetCardQuantity(deps.userID, c.ID)
		if err != nil {
			return err
		}
		if err := deps.collSvc.UpsertCollectionBatch(context.Background(), deps.userID, map[int64]int{c.ID: existing + qty}); err != nil {
			return err
		}
		fmt.Printf("Added %d × %s to collection (total: %d)\n", qty, c.Name, existing+qty)
		return nil
	},
}

var collectionRemoveCmd = &cobra.Command{
	Use:   "remove <game> <set-code> <number> <quantity>",
	Short: "Remove cards from your collection",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
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
		existing, err := deps.collSvc.GetCardQuantity(deps.userID, c.ID)
		if err != nil {
			return err
		}
		newQty := existing - qty
		if newQty < 0 {
			newQty = 0
		}
		if err := deps.collSvc.UpsertCollectionBatch(context.Background(), deps.userID, map[int64]int{c.ID: newQty}); err != nil {
			return err
		}
		fmt.Printf("Removed %d × %s from collection (total: %d)\n", qty, c.Name, newQty)
		return nil
	},
}

var collectionImportCmd = &cobra.Command{
	Use:   "import <game> <csv-file>",
	Short: "Import collection from a CSV file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		deps, err := buildCLIDeps()
		if err != nil {
			return err
		}
		defer deps.db.Close()
		rows, err := export.FromCSV(args[1])
		if err != nil {
			return err
		}
		lookup := func(setCode, number string) (int64, error) {
			c, err := deps.cardSvc.GetCardBySetCodeAndNumber(setCode, number)
			if err != nil {
				return 0, nil
			}
			return c.ID, nil
		}
		result, quantities, err := export.ResolveCSVToCardQuantities(rows, lookup)
		if err != nil {
			return err
		}
		if err := deps.collSvc.UpsertCollectionBatch(context.Background(), deps.userID, quantities); err != nil {
			return err
		}
		fmt.Println(result.Summary())
		return nil
	},
}

var collectionExportCmd = &cobra.Command{
	Use:   "export <game>",
	Short: "Export your collection to a file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		outputFlag, _ := cmd.Flags().GetString("output")
		deps, err := buildCLIDeps()
		if err != nil {
			return err
		}
		defer deps.db.Close()
		game, err := resolveGame(deps.gameSvc, args[0])
		if err != nil {
			return err
		}
		items, err := deps.collSvc.GetUserCollectionByGameID(deps.userID, game.ID)
		if err != nil {
			return err
		}
		cardRows := collectionToCardRows(items)
		outFile := resolveOutputFile(outputFlag, "collection", game.Name, format)
		switch format {
		case "text":
			err = export.ToText(cardRows, outFile)
		case "ptcgo":
			err = export.ToPTCGO(game.Name+" Collection", cardRows, outFile)
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

func collectionToCardRows(items []cardmodel.UserCollection) []export.CardRow {
	rows := make([]export.CardRow, 0, len(items))
	for _, item := range items {
		if item.Card == nil {
			continue
		}
		setName, setCode := "", ""
		if item.Card.Set != nil {
			setName = item.Card.Set.Name
			setCode = item.Card.Set.Code
		}
		rows = append(rows, export.CardRow{
			Name:     item.Card.Name,
			SetName:  setName,
			SetCode:  setCode,
			Number:   item.Card.Number,
			Rarity:   item.Card.Rarity,
			Quantity: item.Quantity,
		})
	}
	return rows
}

func init() {
	collectionExportCmd.Flags().String("format", "csv", "Export format: csv, text, ptcgo")
	collectionExportCmd.Flags().String("output", "", "Output file path (auto-generated if omitted)")
	collectionCmd.AddCommand(collectionListCmd, collectionAddCmd, collectionRemoveCmd, collectionImportCmd, collectionExportCmd)
	rootCmd.AddCommand(collectionCmd)
}
