package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/laiambryant/tui-cardman/internal/export"
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
		return withCollectionRows(args[0], func(_ string, rows []export.CardRow) error {
			printCardRows(rows)
			return nil
		})
	},
}

var collectionAddCmd = &cobra.Command{
	Use:   "add <game> <set-code> <number> <quantity>",
	Short: "Add cards to your collection",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		return adjustCollection(cmd, args, 1)
	},
}

var collectionRemoveCmd = &cobra.Command{
	Use:   "remove <game> <set-code> <number> <quantity>",
	Short: "Remove cards from your collection",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		return adjustCollection(cmd, args, -1)
	},
}

// adjustCollection adds (sign 1) or removes (sign -1) copies of a card.
func adjustCollection(cmd *cobra.Command, args []string, sign int) error {
	qty, err := parseQuantity(args[3])
	if err != nil {
		return err
	}
	return withDeps(func(d *cliDeps) error {
		game, err := d.resolveGame(args[0])
		if err != nil {
			return err
		}
		c, err := d.findCard(game.ID, args[1], args[2])
		if err != nil {
			return err
		}
		existing, err := d.collSvc.GetCardQuantity(d.userID, c.ID)
		if err != nil {
			return err
		}
		newQty := max(existing+sign*qty, 0)
		if err := d.collSvc.UpsertCollectionBatch(cmd.Context(), d.userID, map[int64]int{c.ID: newQty}); err != nil {
			return err
		}
		verb, prep := "Added", "to"
		if sign < 0 {
			verb, prep = "Removed", "from"
		}
		fmt.Printf("%s %d × %s %s collection (total: %d)\n", verb, qty, c.Name, prep, newQty)
		return nil
	})
}

var collectionImportCmd = &cobra.Command{
	Use:   "import <game> <csv-file>",
	Short: "Import collection from a CSV file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		rows, err := export.FromCSV(args[1])
		if err != nil {
			return err
		}
		return withDeps(func(d *cliDeps) error {
			game, err := d.resolveGame(args[0])
			if err != nil {
				return err
			}
			lookup := func(setCode, number string) (int64, error) {
				c, err := d.cardSvc.GetCardByGameAndSetCodeAndNumber(game.ID, setCode, number)
				if err != nil || c == nil {
					return 0, err
				}
				return c.ID, nil
			}
			result, quantities, err := export.ResolveCSVToCardQuantities(rows, lookup)
			if err != nil {
				return err
			}
			if err := d.collSvc.UpsertCollectionBatch(cmd.Context(), d.userID, quantities); err != nil {
				return err
			}
			fmt.Println(result.Summary())
			return nil
		})
	},
}

var collectionExportCmd = &cobra.Command{
	Use:   "export <game>",
	Short: "Export your collection to a file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")
		return withCollectionRows(args[0], func(gameName string, rows []export.CardRow) error {
			return exportRows(rows, format, output, "collection", gameName, gameName+" Collection")
		})
	},
}

// withCollectionRows loads the user's collection for a game (most recently
// added first) as rows and calls fn.
func withCollectionRows(gameArg string, fn func(gameName string, rows []export.CardRow) error) error {
	return withDeps(func(d *cliDeps) error {
		game, err := d.resolveGame(gameArg)
		if err != nil {
			return err
		}
		items, err := d.collSvc.GetUserCollectionByGameID(d.userID, game.ID)
		if err != nil {
			return err
		}
		rows := make([]export.CardRow, 0, len(items))
		for _, item := range items {
			rows = append(rows, toCardRow(*item.Card, item.Quantity))
		}
		return fn(game.Name, rows)
	})
}

func init() {
	addExportFlags(collectionExportCmd)
	collectionCmd.AddCommand(collectionListCmd, collectionAddCmd, collectionRemoveCmd, collectionImportCmd, collectionExportCmd)
	rootCmd.AddCommand(collectionCmd)
}
