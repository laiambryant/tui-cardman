package command

import (
	"github.com/laiambryant/tui-cardman/internal/gameimporter"
	"github.com/laiambryant/tui-cardman/internal/importers"
	"github.com/laiambryant/tui-cardman/internal/pokemontcg"
)

// runPokemonImport opens the database (applying migrations) and runs fn with
// the Pokemon importer. It backs import-full, import-sets and import-updates.
func runPokemonImport(fn func(importer gameimporter.GameImporter) error) error {
	runtime, err := newCommandRuntime(true, true)
	if err != nil {
		return err
	}
	defer runtime.Close()
	importer, err := importers.New(runtime.db, pokemontcg.GameName, runtime.config.GetAPIKey())
	if err != nil {
		return err
	}
	return fn(importer)
}
