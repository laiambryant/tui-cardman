package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
)

const (
	shortMessageImportSets = "Import specific Pokemon TCG sets by set ID"
	longMessageImportSets  = `Import one or more specific Pokemon TCG sets by their set IDs.

This command will:
- Fetch only the specified sets from the Pokemon TCG API
- Import all cards from each specified set
- Store all data (images, prices, etc.) in the local database

Examples:
  cardman import-sets base1
  cardman import-sets base1 jungle fossil
  cardman import-sets swsh1 swsh2 swsh3

Set IDs are typically the lowercase set code (e.g., 'base1', 'jungle', 'swsh1').
You can find set IDs by browsing the API or running import-full with verbose logging.`
)

var importSetsCmd = &cobra.Command{
	Use:   "import-sets [set-id...]",
	Short: shortMessageImportSets,
	Long:  longMessageImportSets,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withImporter("pokemon", func(importer gameimporter.GameImporter) error {
			if err := importer.ImportSpecific(cmd.Context(), args); err != nil {
				return fmt.Errorf("import failed: %w", err)
			}
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(importSetsCmd)
}
