package command

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"text/tabwriter"

	"github.com/laiambryant/tui-cardman/internal/config"
	"github.com/laiambryant/tui-cardman/internal/pokemontcg"
	"github.com/spf13/cobra"
)

const shortMessageListSets = "List all available Pokemon TCG sets from the API"
const longMessageListSets = `Fetch and display all available Pokemon TCG sets from the API.

This command queries the API for all sets and displays them in a table format
showing the set ID, name, series, and total card count.

Use this to discover set IDs for use with the 'import-sets' command.

Examples:
  cardman list-sets
  cardman list-sets | grep -i "Sword"
`

var listSetsCmd = &cobra.Command{
	Use:   "list-sets",
	Short: shortMessageListSets,
	Long:  longMessageListSets,
	RunE: func(cmd *cobra.Command, args []string) error {
		config.LoadConfig()
		ctx := context.Background()

		apiKey := config.GetAPIKey()
		client := pokemontcg.NewClient(apiKey)
		logger := slog.Default()

		logger.Info("Fetching sets from API...")
		sets, err := client.GetSets(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch sets: %w", err)
		}

		// Create a tabwriter for aligned output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "SET ID\tNAME\tSERIES\tTOTAL CARDS")
		fmt.Fprintln(w, "------\t----\t------\t-----------")

		for _, set := range sets {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\n",
				set.ID,
				set.Name,
				set.Series,
				set.Total)
		}

		w.Flush()

		logger.Info("Sets retrieved", "count", len(sets))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listSetsCmd)
}
