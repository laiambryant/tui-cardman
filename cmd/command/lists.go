package command

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/laiambryant/tui-cardman/internal/export"
	"github.com/spf13/cobra"
)

var listsCmd = &cobra.Command{
	Use:   "lists",
	Short: "Manage your card lists",
}

var listsListCmd = &cobra.Command{
	Use:   "list <game>",
	Short: "List all lists for a game",
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
		userLists, err := deps.listSvc.GetListsByUserAndGame(deps.userID, game.ID)
		if err != nil {
			return err
		}
		rows := make([][]string, len(userLists))
		for i, l := range userLists {
			rows[i] = []string{strconv.FormatInt(l.ID, 10), l.Name, l.Description, l.Color}
		}
		writeTabular(os.Stdout, []string{"ID", "NAME", "DESCRIPTION", "COLOR"}, rows)
		return nil
	},
}

var listsCreateCmd = &cobra.Command{
	Use:   "create <game> <name>",
	Short: "Create a new list",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		desc, _ := cmd.Flags().GetString("description")
		color, _ := cmd.Flags().GetString("color")
		deps, err := buildCLIDeps()
		if err != nil {
			return err
		}
		defer deps.db.Close()
		game, err := resolveGame(deps.gameSvc, args[0])
		if err != nil {
			return err
		}
		l, err := deps.listSvc.CreateList(context.Background(), deps.userID, game.ID, args[1], desc, color)
		if err != nil {
			return err
		}
		fmt.Printf("Created list %q (ID: %d)\n", l.Name, l.ID)
		return nil
	},
}

var listsDeleteCmd = &cobra.Command{
	Use:   "delete <list-id>",
	Short: "Delete a list",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		listID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid list ID: %s", args[0])
		}
		deps, err := buildCLIDeps()
		if err != nil {
			return err
		}
		defer deps.db.Close()
		if err := deps.listSvc.DeleteList(context.Background(), listID); err != nil {
			return err
		}
		fmt.Printf("Deleted list %d\n", listID)
		return nil
	},
}

var listsShowCmd = &cobra.Command{
	Use:   "show <list-id>",
	Short: "Show cards in a list",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		listID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid list ID: %s", args[0])
		}
		deps, err := buildCLIDeps()
		if err != nil {
			return err
		}
		defer deps.db.Close()
		l, err := deps.listSvc.GetListByID(listID)
		if err != nil {
			return err
		}
		game, err := resolveGameByID(deps.gameSvc, l.CardGameID)
		if err != nil {
			return err
		}
		quantities, err := deps.listSvc.GetAllQuantitiesForList(listID)
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
		fmt.Printf("List: %s\n", l.Name)
		writeTabular(os.Stdout, []string{"NAME", "SET", "NUMBER", "RARITY", "QTY"}, rows)
		return nil
	},
}

var listsAddCardCmd = &cobra.Command{
	Use:   "add-card <list-id> <set-code> <number> <quantity>",
	Short: "Add a card to a list",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		listID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid list ID: %s", args[0])
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
		if err := deps.listSvc.UpsertListCardBatch(context.Background(), listID, map[int64]int{c.ID: qty}); err != nil {
			return err
		}
		fmt.Printf("Added %d × %s to list\n", qty, c.Name)
		return nil
	},
}

var listsRemoveCardCmd = &cobra.Command{
	Use:   "remove-card <list-id> <set-code> <number>",
	Short: "Remove a card from a list",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		listID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid list ID: %s", args[0])
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
		if err := deps.listSvc.UpsertListCardBatch(context.Background(), listID, map[int64]int{c.ID: 0}); err != nil {
			return err
		}
		fmt.Printf("Removed %s from list\n", c.Name)
		return nil
	},
}

var listsExportCmd = &cobra.Command{
	Use:   "export <list-id>",
	Short: "Export a list to a file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		listID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid list ID: %s", args[0])
		}
		format, _ := cmd.Flags().GetString("format")
		outputFlag, _ := cmd.Flags().GetString("output")
		deps, err := buildCLIDeps()
		if err != nil {
			return err
		}
		defer deps.db.Close()
		l, err := deps.listSvc.GetListByID(listID)
		if err != nil {
			return err
		}
		game, err := resolveGameByID(deps.gameSvc, l.CardGameID)
		if err != nil {
			return err
		}
		quantities, err := deps.listSvc.GetAllQuantitiesForList(listID)
		if err != nil {
			return err
		}
		allCards, err := deps.cardSvc.GetCardsByGameID(game.ID)
		if err != nil {
			return err
		}
		cardsByID := buildCardsByIDMap(allCards)
		cardRows := quantitiesToCardRows(quantities, cardsByID)
		outFile := resolveOutputFile(outputFlag, "list", l.Name, format)
		switch format {
		case "text":
			err = export.ToText(cardRows, outFile)
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
	listsCreateCmd.Flags().String("description", "", "List description")
	listsCreateCmd.Flags().String("color", "", "List color")
	listsExportCmd.Flags().String("format", "csv", "Export format: csv, text")
	listsExportCmd.Flags().String("output", "", "Output file path (auto-generated if omitted)")
	listsCmd.AddCommand(listsListCmd, listsCreateCmd, listsDeleteCmd, listsShowCmd, listsAddCardCmd, listsRemoveCardCmd, listsExportCmd)
	rootCmd.AddCommand(listsCmd)
}
