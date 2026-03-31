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
		deps, err := buildCLIDeps()
		if err != nil {
			return err
		}
		defer deps.db.Close()
		game, err := resolveGame(deps.gameSvc, args[0])
		if err != nil {
			return err
		}
		cards, err := deps.cardSvc.GetCardsByGameID(game.ID)
		if err != nil {
			return err
		}
		if q, _ := cmd.Flags().GetString("search"); q != "" {
			cards = filterCards(cards, q)
		}
		if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
			return json.NewEncoder(os.Stdout).Encode(cards)
		}
		rows := make([][]string, len(cards))
		for i, c := range cards {
			setCode, setName := "", ""
			if c.Set != nil {
				setCode = c.Set.Code
				setName = c.Set.Name
			}
			rows[i] = []string{c.Name, setCode, setName, c.Number, c.Rarity}
		}
		writeTabular(os.Stdout, []string{"NAME", "SET CODE", "SET", "NUMBER", "RARITY"}, rows)
		return nil
	},
}

var cardsInfoCmd = &cobra.Command{
	Use:   "info <game> <set-code> <number>",
	Short: "Show details for a specific card",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		deps, err := buildCLIDeps()
		if err != nil {
			return err
		}
		defer deps.db.Close()
		c, err := deps.cardSvc.GetCardBySetCodeAndNumber(args[1], args[2])
		if err != nil {
			return err
		}
		if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
			return json.NewEncoder(os.Stdout).Encode(c)
		}
		setCode, setName := "", ""
		if c.Set != nil {
			setCode = c.Set.Code
			setName = c.Set.Name
		}
		fmt.Printf("Name:    %s\nSet:     %s (%s)\nNumber:  %s\nRarity:  %s\n", c.Name, setName, setCode, c.Number, c.Rarity)
		return nil
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
