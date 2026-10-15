package command

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import card data from external APIs",
}

var importListSetsCmd = &cobra.Command{
	Use:   "list-sets <game>",
	Short: "List available sets for a game from the API",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withImporter(args[0], func(importer gameimporter.GameImporter) error {
			gameSets, err := importer.FetchSets(cmd.Context())
			if err != nil {
				return fmt.Errorf("fetch sets: %w", err)
			}
			rows := make([][]string, len(gameSets))
			for i, s := range gameSets {
				rows[i] = []string{s.APIID, s.Name, s.Code, strconv.Itoa(s.Total)}
			}
			writeTabular(os.Stdout, []string{"API ID", "NAME", "CODE", "TOTAL"}, rows)
			return nil
		})
	},
}

// importAction builds a RunE that runs an import for the game named in args[0].
func importAction(message string, run func(cmd *cobra.Command, args []string, importer gameimporter.GameImporter) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return withImporter(args[0], func(importer gameimporter.GameImporter) error {
			fmt.Printf(message+"\n", args[0])
			if err := run(cmd, args, importer); err != nil {
				return err
			}
			fmt.Println("Done.")
			return nil
		})
	}
}

var importGroupSetsCmd = &cobra.Command{
	Use:   "sets <game> <set-id...>",
	Short: "Import specific sets for a game",
	Args:  cobra.MinimumNArgs(2),
	RunE: importAction("Importing selected sets for %s...", func(cmd *cobra.Command, args []string, importer gameimporter.GameImporter) error {
		return importer.ImportSpecific(cmd.Context(), args[1:])
	}),
}

var importAllCmd = &cobra.Command{
	Use:   "all <game>",
	Short: "Import all sets for a game",
	Args:  cobra.ExactArgs(1),
	RunE: importAction("Importing all sets for %s...", func(cmd *cobra.Command, _ []string, importer gameimporter.GameImporter) error {
		return importer.ImportAll(cmd.Context())
	}),
}

var importNewCmd = &cobra.Command{
	Use:   "new <game>",
	Short: "Import only new sets for a game",
	Args:  cobra.ExactArgs(1),
	RunE: importAction("Importing new sets for %s...", func(cmd *cobra.Command, _ []string, importer gameimporter.GameImporter) error {
		return importer.ImportNew(cmd.Context())
	}),
}

func init() {
	importCmd.AddCommand(importListSetsCmd, importGroupSetsCmd, importAllCmd, importNewCmd)
	rootCmd.AddCommand(importCmd)
}
