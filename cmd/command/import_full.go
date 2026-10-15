package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
)

var importFullCmd = &cobra.Command{
	Use:   "import-full",
	Short: shortMessageImportAll,
	Long:  longMessageImportAll,
	RunE: func(cmd *cobra.Command, args []string) error {
		return withImporter("pokemon", func(importer gameimporter.GameImporter) error {
			if err := importer.ImportAll(cmd.Context()); err != nil {
				return fmt.Errorf("import failed: %w", err)
			}
			return nil
		})
	},
}

const (
	shortMessageImportAll = "Import all Pokemon TCG sets and cards from the API"
	longMessageImportAll  = `Performs a complete import of all Pokemon TCG sets and cards from the API.

This command will:
- Fetch all sets from the Pokemon TCG API
- For each set, import all cards with complete data (images, prices, etc.)
- Store all data in the local database

Note: This can take a significant amount of time and will use API quota.
Use 'import-updates' for incremental imports of new sets only.`
)

func init() {
	rootCmd.AddCommand(importFullCmd)
}
