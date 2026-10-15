package command

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	cardmodel "github.com/laiambryant/tui-cardman/internal/model"
)

var cardsCmd = &cobra.Command{
	Use:   "cards",
	Short: "Browse and inspect cards",
}

var cardsListCmd = &cobra.Command{
	Use:   "list <game>",
	Short: "List all cards for a game",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query, _ := cmd.Flags().GetString("search")
		asJSON, _ := cmd.Flags().GetBool("json")
		return withDeps(func(d *cliDeps) error {
			game, err := d.resolveGame(args[0])
			if err != nil {
				return err
			}
			cards, err := d.cardSvc.GetCardsByGameID(game.ID)
			if err != nil {
				return err
			}
			if query != "" {
				cards = filterCards(cards, query)
			}
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(cards)
			}
			rows := make([][]string, len(cards))
			for i, c := range cards {
				r := toCardRow(c, 0)
				rows[i] = []string{c.Name, r.SetCode, r.SetName, c.Number, c.Rarity}
			}
			writeTabular(os.Stdout, []string{"NAME", "SET CODE", "SET", "NUMBER", "RARITY"}, rows)
			return nil
		})
	},
}

var cardsInfoCmd = &cobra.Command{
	Use:   "info <game> <set-code> <number>",
	Short: "Show details for a specific card",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")
		return withDeps(func(d *cliDeps) error {
			game, err := d.resolveGame(args[0])
			if err != nil {
				return err
			}
			c, err := d.findCard(game.ID, args[1], args[2])
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(c)
			}
			r := toCardRow(*c, 0)
			fmt.Printf("Name:    %s\nSet:     %s (%s)\nNumber:  %s\nRarity:  %s\n", c.Name, r.SetName, r.SetCode, c.Number, c.Rarity)
			return nil
		})
	},
}

func filterCards(cards []cardmodel.Card, query string) []cardmodel.Card {
	q := strings.ToLower(query)
	var result []cardmodel.Card
	for _, c := range cards {
		if strings.Contains(strings.ToLower(c.Name), q) {
			result = append(result, c)
		}
	}
	return result
}

func init() {
	cardsListCmd.Flags().String("search", "", "Filter cards by name")
	cardsListCmd.Flags().Bool("json", false, "Output as JSON")
	cardsInfoCmd.Flags().Bool("json", false, "Output as JSON")
	cardsCmd.AddCommand(cardsListCmd, cardsInfoCmd)
	rootCmd.AddCommand(cardsCmd)
}
