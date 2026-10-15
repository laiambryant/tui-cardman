package command

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/laiambryant/tui-cardman/internal/export"
)

// cardContainer describes a deck or a list: a named set of card quantities
// belonging to one game. Decks and lists share their card subcommands.
type cardContainer struct {
	kind       string // "deck" or "list"
	label      string // "Deck" or "List"
	get        func(d *cliDeps, id int64) (name string, gameID int64, err error)
	quantities func(d *cliDeps, id int64) (map[int64]int, error)
	upsert     func(d *cliDeps, ctx context.Context, id int64, updates map[int64]int) error
	remove     func(d *cliDeps, ctx context.Context, id int64) error
}

var deckContainer = cardContainer{
	kind:  "deck",
	label: "Deck",
	get: func(d *cliDeps, id int64) (string, int64, error) {
		dk, err := d.deckSvc.GetDeckByID(id)
		if err != nil {
			return "", 0, err
		}
		return dk.Name, dk.CardGameID, nil
	},
	quantities: func(d *cliDeps, id int64) (map[int64]int, error) { return d.deckSvc.GetAllQuantitiesForDeck(id) },
	upsert: func(d *cliDeps, ctx context.Context, id int64, updates map[int64]int) error {
		return d.deckSvc.UpsertDeckCardBatch(ctx, id, updates)
	},
	remove: func(d *cliDeps, ctx context.Context, id int64) error { return d.deckSvc.DeleteDeck(ctx, id) },
}

var listContainer = cardContainer{
	kind:  "list",
	label: "List",
	get: func(d *cliDeps, id int64) (string, int64, error) {
		l, err := d.listSvc.GetListByID(id)
		if err != nil {
			return "", 0, err
		}
		return l.Name, l.CardGameID, nil
	},
	quantities: func(d *cliDeps, id int64) (map[int64]int, error) { return d.listSvc.GetAllQuantitiesForList(id) },
	upsert: func(d *cliDeps, ctx context.Context, id int64, updates map[int64]int) error {
		return d.listSvc.UpsertListCardBatch(ctx, id, updates)
	},
	remove: func(d *cliDeps, ctx context.Context, id int64) error { return d.listSvc.DeleteList(ctx, id) },
}

// cardCommands returns the delete, show, add-card, remove-card and export
// subcommands for a container kind.
func (c cardContainer) cardCommands() []*cobra.Command {
	exportCmd := &cobra.Command{
		Use:   fmt.Sprintf("export <%s-id>", c.kind),
		Short: fmt.Sprintf("Export a %s to a file", c.kind),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			output, _ := cmd.Flags().GetString("output")
			return c.withRows(args[0], func(name string, rows []export.CardRow) error {
				return exportRows(rows, format, output, c.kind, name, name)
			})
		},
	}
	addExportFlags(exportCmd)
	return []*cobra.Command{
		{
			Use:   fmt.Sprintf("delete <%s-id>", c.kind),
			Short: "Delete a " + c.kind,
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				id, err := parseID(c.kind, args[0])
				if err != nil {
					return err
				}
				return withDeps(func(d *cliDeps) error {
					if err := c.remove(d, cmd.Context(), id); err != nil {
						return err
					}
					fmt.Printf("Deleted %s %d\n", c.kind, id)
					return nil
				})
			},
		},
		{
			Use:   fmt.Sprintf("show <%s-id>", c.kind),
			Short: "Show cards in a " + c.kind,
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return c.withRows(args[0], func(name string, rows []export.CardRow) error {
					fmt.Printf("%s: %s\n", c.label, name)
					printCardRows(rows)
					return nil
				})
			},
		},
		{
			Use:   fmt.Sprintf("add-card <%s-id> <set-code> <number> <quantity>", c.kind),
			Short: fmt.Sprintf("Add copies of a card to a %s", c.kind),
			Args:  cobra.ExactArgs(4),
			RunE: func(cmd *cobra.Command, args []string) error {
				qty, err := parseQuantity(args[3])
				if err != nil {
					return err
				}
				return c.adjustCard(cmd.Context(), args[0], args[1], args[2], qty)
			},
		},
		{
			Use:   fmt.Sprintf("remove-card <%s-id> <set-code> <number>", c.kind),
			Short: fmt.Sprintf("Remove a card from a %s", c.kind),
			Args:  cobra.ExactArgs(3),
			RunE: func(cmd *cobra.Command, args []string) error {
				return c.adjustCard(cmd.Context(), args[0], args[1], args[2], 0)
			},
		},
		exportCmd,
	}
}

// load returns the container's name and game, reporting unknown IDs clearly.
func (c cardContainer) load(d *cliDeps, id int64) (string, int64, error) {
	name, gameID, err := c.get(d, id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", 0, fmt.Errorf("%s %d not found", c.kind, id)
	}
	return name, gameID, err
}

// adjustCard adds qty copies of a card to the container, or removes the card
// entirely when qty is 0.
func (c cardContainer) adjustCard(ctx context.Context, idArg, setCode, number string, qty int) error {
	id, err := parseID(c.kind, idArg)
	if err != nil {
		return err
	}
	return withDeps(func(d *cliDeps) error {
		_, gameID, err := c.load(d, id)
		if err != nil {
			return err
		}
		card, err := d.findCard(gameID, setCode, number)
		if err != nil {
			return err
		}
		newQty := 0
		if qty > 0 {
			quantities, err := c.quantities(d, id)
			if err != nil {
				return err
			}
			newQty = quantities[card.ID] + qty
		}
		if err := c.upsert(d, ctx, id, map[int64]int{card.ID: newQty}); err != nil {
			return err
		}
		if qty == 0 {
			fmt.Printf("Removed %s from %s\n", card.Name, c.kind)
		} else {
			fmt.Printf("Added %d × %s to %s (total: %d)\n", qty, card.Name, c.kind, newQty)
		}
		return nil
	})
}

// withRows loads the container's cards as rows and calls fn.
func (c cardContainer) withRows(idArg string, fn func(name string, rows []export.CardRow) error) error {
	id, err := parseID(c.kind, idArg)
	if err != nil {
		return err
	}
	return withDeps(func(d *cliDeps) error {
		name, gameID, err := c.load(d, id)
		if err != nil {
			return err
		}
		quantities, err := c.quantities(d, id)
		if err != nil {
			return err
		}
		rows, err := d.cardRows(gameID, quantities)
		if err != nil {
			return err
		}
		return fn(name, rows)
	})
}
