package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Tab represents different tabs in the card game view
type Tab int

const (
	TabCollection Tab = iota
	TabCardSearch
	TabUserSearch
)

// CardGameTabsModel represents the state for the card game tabs view
type CardGameTabsModel struct {
	selectedGame       *CardGame
	currentTab         Tab
	searchInput        textinput.Model
	cards              []Card
	userCollections    []UserCollection
	filteredCards      []Card
	filteredCollection []UserCollection
	cursor             int
}

// NewCardGameTabsModel creates a new card game tabs model
func NewCardGameTabsModel(selectedGame *CardGame) CardGameTabsModel {
	searchInput := textinput.New()
	searchInput.Placeholder = "Search cards..."
	searchInput.Width = 30

	return CardGameTabsModel{
		selectedGame: selectedGame,
		currentTab:   TabCollection,
		searchInput:  searchInput,
		cursor:       0,
	}
}

func (m CardGameTabsModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m CardGameTabsModel) Update(msg tea.Msg) (CardGameTabsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// Full quit
			return m, tea.Quit
		case "q", "esc":
			// Return to main screen - parent will handle this
			return m, nil
		case "tab":
			// Move to next tab
			m.currentTab = (m.currentTab + 1) % 3
			if m.currentTab == TabCardSearch || m.currentTab == TabUserSearch {
				m.searchInput.Focus()
			} else {
				m.searchInput.Blur()
			}
			return m, nil
		case "shift+tab":
			// Move to previous tab
			if m.currentTab == 0 {
				m.currentTab = 2
			} else {
				m.currentTab--
			}
			if m.currentTab == TabCardSearch || m.currentTab == TabUserSearch {
				m.searchInput.Focus()
			} else {
				m.searchInput.Blur()
			}
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			maxItems := 0
			switch m.currentTab {
			case TabCollection:
				maxItems = len(m.filteredCollection)
			case TabCardSearch:
				maxItems = len(m.filteredCards)
			case TabUserSearch:
				maxItems = len(m.filteredCollection)
			}
			if m.cursor < maxItems-1 {
				m.cursor++
			}
			return m, nil
		}
	}

	// Update search input if in search tabs
	if m.currentTab == TabCardSearch || m.currentTab == TabUserSearch {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)

		// Filter results based on search
		if m.currentTab == TabCardSearch {
			m.filteredCards = m.filterCards(m.searchInput.Value())
		} else if m.currentTab == TabUserSearch {
			m.filteredCollection = m.filterUserCollection(m.searchInput.Value())
		}

		// Reset cursor when search changes
		m.cursor = 0
		return m, cmd
	}

	return m, nil
}

func (m CardGameTabsModel) View() string {
	var b strings.Builder

	// Header with game name
	if m.selectedGame != nil {
		b.WriteString(titleStyle.Render("📦 "+m.selectedGame.Name+" Collection Manager") + "\n\n")
	}

	// Tab navigation
	tabs := []string{"Collection", "Card Search", "User Search"}
	var tabStyles []string

	for i, tab := range tabs {
		if Tab(i) == m.currentTab {
			tabStyles = append(tabStyles, focusedStyle.Render("[ "+tab+" ]"))
		} else {
			tabStyles = append(tabStyles, blurredStyle.Render("  "+tab+"  "))
		}
	}

	b.WriteString(strings.Join(tabStyles, " ") + "\n\n")

	// Tab content
	switch m.currentTab {
	case TabCollection:
		b.WriteString(m.renderCollectionTab())
	case TabCardSearch:
		b.WriteString(m.renderCardSearchTab())
	case TabUserSearch:
		b.WriteString(m.renderUserSearchTab())
	}

	// Help text
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Tab/Shift+Tab: Switch tabs • ↑/↓: Navigate • Q/Esc: Back • Ctrl+C: Quit") + "\n")

	return b.String()
}

func (m CardGameTabsModel) renderCollectionTab() string {
	var b strings.Builder

	b.WriteString(focusedStyle.Render("Your Collection Summary") + "\n\n")

	if len(m.filteredCollection) == 0 {
		b.WriteString(blurredStyle.Render("No cards in your collection yet.") + "\n")
		b.WriteString(blurredStyle.Render("Use Card Search to discover cards to add!") + "\n")
	} else {
		// Summary stats
		totalCards := 0
		for _, collection := range m.filteredCollection {
			totalCards += collection.Quantity
		}

		b.WriteString(blurredStyle.Render("Total unique cards: ") +
			focusedStyle.Render(fmt.Sprintf("%d", len(m.filteredCollection))) + "\n")
		b.WriteString(blurredStyle.Render("Total cards: ") +
			focusedStyle.Render(fmt.Sprintf("%d", totalCards)) + "\n\n")

		// Collection list
		b.WriteString(focusedStyle.Render("Recent additions:") + "\n")
		for i, collection := range m.filteredCollection {
			if i >= 10 { // Show only first 10
				break
			}

			style := blurredStyle
			if i == m.cursor {
				style = focusedStyle
			}

			cardName := "Unknown Card"
			if collection.Card != nil {
				cardName = collection.Card.Name
			}

			line := style.Render(cardName + " x" + fmt.Sprintf("%d", collection.Quantity))
			if collection.Condition != "" {
				line += blurredStyle.Render(" (" + collection.Condition + ")")
			}
			b.WriteString(line + "\n")
		}
	}

	return b.String()
}

func (m CardGameTabsModel) renderCardSearchTab() string {
	var b strings.Builder

	b.WriteString(focusedStyle.Render("Search All Cards") + "\n\n")

	// Search input
	b.WriteString(blurredStyle.Render("Search: ") + m.searchInput.View() + "\n\n")

	if len(m.filteredCards) == 0 {
		if m.searchInput.Value() == "" {
			b.WriteString(blurredStyle.Render("Type to search for cards...") + "\n")
		} else {
			b.WriteString(blurredStyle.Render("No cards match your search.") + "\n")
		}
	} else {
		b.WriteString(focusedStyle.Render("Found cards:") + "\n")

		// Show up to 15 cards
		maxShow := 15
		if len(m.filteredCards) < maxShow {
			maxShow = len(m.filteredCards)
		}

		for i := 0; i < maxShow; i++ {
			card := m.filteredCards[i]

			style := blurredStyle
			if i == m.cursor {
				style = focusedStyle
			}

			line := style.Render(card.Name)
			if card.Expansion != "" {
				line += blurredStyle.Render(" - " + card.Expansion)
			}
			if card.Rarity != "" {
				line += blurredStyle.Render(" [" + card.Rarity + "]")
			}
			b.WriteString(line + "\n")
		}

		if len(m.filteredCards) > maxShow {
			b.WriteString(blurredStyle.Render(fmt.Sprintf("... and %d more", len(m.filteredCards)-maxShow)) + "\n")
		}
	}

	return b.String()
}

func (m CardGameTabsModel) renderUserSearchTab() string {
	var b strings.Builder

	b.WriteString(focusedStyle.Render("Search Your Collection") + "\n\n")

	// Search input
	b.WriteString(blurredStyle.Render("Search: ") + m.searchInput.View() + "\n\n")

	if len(m.filteredCollection) == 0 {
		if m.searchInput.Value() == "" {
			b.WriteString(blurredStyle.Render("Type to search your collection...") + "\n")
		} else {
			b.WriteString(blurredStyle.Render("No cards in your collection match your search.") + "\n")
		}
	} else {
		b.WriteString(focusedStyle.Render("Your matching cards:") + "\n")

		for i, collection := range m.filteredCollection {
			style := blurredStyle
			if i == m.cursor {
				style = focusedStyle
			}

			cardName := "Unknown Card"
			if collection.Card != nil {
				cardName = collection.Card.Name
			}

			line := style.Render(cardName + " x" + fmt.Sprintf("%d", collection.Quantity))
			if collection.Condition != "" {
				line += blurredStyle.Render(" (" + collection.Condition + ")")
			}
			if collection.Notes != "" {
				line += blurredStyle.Render(" - " + collection.Notes)
			}
			b.WriteString(line + "\n")
		}
	}

	return b.String()
}

// filterCards filters cards based on search query using fuzzy matching
func (m CardGameTabsModel) filterCards(query string) []Card {
	if query == "" {
		return m.cards
	}

	var filtered []Card
	query = strings.ToLower(query)

	for _, card := range m.cards {
		if strings.Contains(strings.ToLower(card.Name), query) ||
			strings.Contains(strings.ToLower(card.Expansion), query) ||
			strings.Contains(strings.ToLower(card.Rarity), query) {
			filtered = append(filtered, card)
		}
	}

	return filtered
}

// filterUserCollection filters user collection based on search query
func (m CardGameTabsModel) filterUserCollection(query string) []UserCollection {
	if query == "" {
		return m.userCollections
	}

	var filtered []UserCollection
	query = strings.ToLower(query)

	for _, collection := range m.userCollections {
		if collection.Card != nil {
			if strings.Contains(strings.ToLower(collection.Card.Name), query) ||
				strings.Contains(strings.ToLower(collection.Card.Expansion), query) ||
				strings.Contains(strings.ToLower(collection.Card.Rarity), query) ||
				strings.Contains(strings.ToLower(collection.Condition), query) ||
				strings.Contains(strings.ToLower(collection.Notes), query) {
				filtered = append(filtered, collection)
			}
		}
	}

	return filtered
}
