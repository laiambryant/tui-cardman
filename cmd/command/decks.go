package command

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
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
		return withDeps(func(d *cliDeps) error {
			game, err := d.resolveGame(args[0])
			if err != nil {
				return err
			}
			decks, err := d.deckSvc.GetDecksByUserAndGame(d.userID, game.ID)
			if err != nil {
				return err
			}
			rows := make([][]string, len(decks))
			for i, dk := range decks {
				rows[i] = []string{strconv.FormatInt(dk.ID, 10), dk.Name}
			}
			writeTabular(os.Stdout, []string{"ID", "NAME"}, rows)
			return nil
		})
	},
}

var decksCreateCmd = &cobra.Command{
	Use:   "create <game> <name>",
	Short: "Create a new deck",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDeps(func(d *cliDeps) error {
			game, err := d.resolveGame(args[0])
			if err != nil {
				return err
			}
			dk, err := d.deckSvc.CreateDeck(cmd.Context(), d.userID, game.ID, args[1])
			if err != nil {
				return err
			}
			fmt.Printf("Created deck %q (ID: %d)\n", dk.Name, dk.ID)
			return nil
		})
	},
}

func init() {
	decksCmd.AddCommand(decksListCmd, decksCreateCmd)
	decksCmd.AddCommand(deckContainer.cardCommands()...)
	rootCmd.AddCommand(decksCmd)
}
