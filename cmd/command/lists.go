package command

import (
	"fmt"
	"os"
	"strconv"

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
		return withDeps(func(d *cliDeps) error {
			game, err := d.resolveGame(args[0])
			if err != nil {
				return err
			}
			userLists, err := d.listSvc.GetListsByUserAndGame(d.userID, game.ID)
			if err != nil {
				return err
			}
			rows := make([][]string, len(userLists))
			for i, l := range userLists {
				rows[i] = []string{strconv.FormatInt(l.ID, 10), l.Name, l.Description, l.Color}
			}
			writeTabular(os.Stdout, []string{"ID", "NAME", "DESCRIPTION", "COLOR"}, rows)
			return nil
		})
	},
}

var listsCreateCmd = &cobra.Command{
	Use:   "create <game> <name>",
	Short: "Create a new list",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		desc, _ := cmd.Flags().GetString("description")
		color, _ := cmd.Flags().GetString("color")
		return withDeps(func(d *cliDeps) error {
			game, err := d.resolveGame(args[0])
			if err != nil {
				return err
			}
			l, err := d.listSvc.CreateList(cmd.Context(), d.userID, game.ID, args[1], desc, color)
			if err != nil {
				return err
			}
			fmt.Printf("Created list %q (ID: %d)\n", l.Name, l.ID)
			return nil
		})
	},
}

func init() {
	listsCreateCmd.Flags().String("description", "", "List description")
	listsCreateCmd.Flags().String("color", "#FFFFFF", "List color")
	listsCmd.AddCommand(listsListCmd, listsCreateCmd)
	listsCmd.AddCommand(listContainer.cardCommands()...)
	rootCmd.AddCommand(listsCmd)
}
