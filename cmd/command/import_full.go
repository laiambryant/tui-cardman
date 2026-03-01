package command

import (
	"context"
	"fmt"

	"github.com/laiambryant/tui-cardman/internal/config"
	"github.com/spf13/cobra"
)

var importFullCmd = &cobra.Command{
	Use:   "import-full",
	Short: shortMessageImportAll,
	Long:  longMessageImportAll,
	RunE: func(cmd *cobra.Command, args []string) error {
		config.LoadConfig()
		ctx := context.Background()

		deps, err := buildImportService()
		if err != nil {
			return err
		}
		defer deps.database.Close()

		deps.logger.Info("Starting full Pokemon TCG import")
		if err := deps.importService.ImportAllSets(ctx); err != nil {
			return fmt.Errorf("import failed: %w", err)
		}
		deps.logger.Info("Full import completed successfully")
		return nil
	},
}

const shortMessageImportAll = "Import all Pokemon TCG sets and cards from the API"
const longMessageImportAll = `Performs a complete import of all Pokemon TCG sets and cards from the API.

This command will:
- Fetch all sets from the Pokemon TCG API
- For each set, import all cards with complete data (images, prices, etc.)
- Store all data in the local database

Note: This can take a significant amount of time and will use API quota.
Use 'import-updates' for incremental imports of new sets only.`

func init() {
	rootCmd.AddCommand(importFullCmd)
}
