package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/laiambryant/tui-cardman/internal/gameimporter"
)

const (
	shortMessageImportUpdates = "Import only new Pokemon TCG sets that don't exist in the database"
	longMessageImportUpdates  = `Performs an incremental import of new Pokemon TCG sets only.

This command will:
- Fetch all sets from the Pokemon TCG API
- Compare with existing sets in the database
- Import only sets that are NOT already in the database
- Skip all existing sets to save time and API quota

This is ideal for periodic updates to catch new set releases without
re-importing existing data. Run this daily or weekly via cron.

Note: This does NOT update existing cards or refresh prices. It only
adds net-new sets.`
)

var importUpdatesCmd = &cobra.Command{
	Use:   "import-updates",
	Short: shortMessageImportUpdates,
	Long:  longMessageImportUpdates,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPokemonImport(func(importer gameimporter.GameImporter) error {
			if err := importer.ImportNew(cmd.Context()); err != nil {
				return fmt.Errorf("import failed: %w", err)
			}
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(importUpdatesCmd)
}
