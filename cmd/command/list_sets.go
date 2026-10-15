package command

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/laiambryant/tui-cardman/internal/pokemontcg"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	shortMessageListSets = "List all available Pokemon TCG sets from the API"
	longMessageListSets  = `Fetch and display all available Pokemon TCG sets from the API.

This command queries the API for all sets and displays them in a table format
showing the set ID, name, series, and total card count.

Use this to discover set IDs for use with the 'import-sets' command.
For plain, scriptable output use 'cardman import list-sets pokemon'.
`
)

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
		runtime, err := newCommandRuntime(false, false)
		if err != nil {
			return err
		}
		defer runtime.Close()

		sets, err := pokemontcg.NewClient(runtime.config.GetAPIKey()).GetSets(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to fetch sets: %w", err)
		}

		// Prepare table columns and rows
		columns := []table.Column{
			{Title: "SET ID", Width: 10},
			{Title: "NAME", Width: 20},
			{Title: "TOTAL CARDS", Width: 15},
		}

		rows := make([]table.Row, 0, len(sets))
		for _, set := range sets {
			rows = append(rows, table.Row{set.ID, set.Name, strconv.Itoa(set.Total)})
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

		if _, err := tea.NewProgram(model{t}).Run(); err != nil {
			return fmt.Errorf("run sets table: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listSetsCmd)
}
