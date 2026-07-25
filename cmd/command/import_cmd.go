package command

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
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
		runtime, err := newCommandRuntime(true, false)
		if err != nil {
			return err
		}
		defer runtime.Close()
		importer, err := buildImporterForGame(runtime.db, args[0], runtime.config)
		if err != nil {
			return err
		}
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
	},
}

var importGroupSetsCmd = &cobra.Command{
	Use:   "sets <game> <set-id...>",
	Short: "Import specific sets for a game",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := newCommandRuntime(true, false)
		if err != nil {
			return err
		}
		defer runtime.Close()
		importer, err := buildImporterForGame(runtime.db, args[0], runtime.config)
		if err != nil {
			return err
		}
		setIDs := args[1:]
		fmt.Printf("Importing %d set(s) for %s...\n", len(setIDs), args[0])
		if err := importer.ImportSpecific(cmd.Context(), setIDs); err != nil {
			return err
		}
		fmt.Println("Done.")
		return nil
	},
}

var importAllCmd = &cobra.Command{
	Use:   "all <game>",
	Short: "Import all sets for a game",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := newCommandRuntime(true, false)
		if err != nil {
			return err
		}
		defer runtime.Close()
		importer, err := buildImporterForGame(runtime.db, args[0], runtime.config)
		if err != nil {
			return err
		}
		fmt.Printf("Importing all sets for %s...\n", args[0])
		if err := importer.ImportAll(cmd.Context()); err != nil {
			return err
		}
		fmt.Println("Done.")
		return nil
	},
}

var importNewCmd = &cobra.Command{
	Use:   "new <game>",
	Short: "Import only new sets for a game",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := newCommandRuntime(true, false)
		if err != nil {
			return err
		}
		defer runtime.Close()
		importer, err := buildImporterForGame(runtime.db, args[0], runtime.config)
		if err != nil {
			return err
		}
		fmt.Printf("Importing new sets for %s...\n", args[0])
		if err := importer.ImportNew(cmd.Context()); err != nil {
			return err
		}
		fmt.Println("Done.")
		return nil
	},
}

func init() {
	importCmd.AddCommand(importListSetsCmd, importGroupSetsCmd, importAllCmd, importNewCmd)
	rootCmd.AddCommand(importCmd)
}
