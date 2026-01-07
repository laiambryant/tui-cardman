package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/laiambryant/tui-cardman/internal/config"
	"github.com/laiambryant/tui-cardman/internal/pokemontcg"
	"github.com/spf13/cobra"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	table table.Model
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m, tea.Batch(
				tea.Printf("Let's go to %s!", m.table.SelectedRow()[1]),
			)
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

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

		// Prepare table columns and rows
		columns := []table.Column{
			{Title: "SET ID", Width: 10},
			{Title: "NAME", Width: 20},
			{Title: "SERIES", Width: 8},
			{Title: "TOTAL CARDS", Width: 15},
		}

		rows := []table.Row{}
		for _, set := range sets {
			rows = append(rows, table.Row{
				set.ID,
				set.Name,
				set.Series,
				fmt.Sprintf("%d", set.Total),
			})
		}

		t := table.New(
			table.WithColumns(columns),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(7),
		)

		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(true).
			Align(lipgloss.Center)
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true)
		t.SetStyles(s)

		m := model{t}
		if _, err := tea.NewProgram(m).Run(); err != nil {
			fmt.Println("Error running program:", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listSetsCmd)
}
