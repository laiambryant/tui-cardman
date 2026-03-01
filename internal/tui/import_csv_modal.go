package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/laiambryant/tui-cardman/internal/export"
	cardservice "github.com/laiambryant/tui-cardman/internal/services/cards"
)

// importPhase tracks which step of the import flow is active.
type importPhase int

const (
	importPhaseInput   importPhase = iota // user is typing a file path
	importPhaseWorking                    // async CSV parse + card resolve in progress
	importPhaseResult                     // showing summary, awaiting confirm / cancel
)

// importReadyMsg is sent by the async worker once CSV parsing and card resolution finish.
type importReadyMsg struct {
	result     *export.ImportResult
	quantities map[int64]int
	err        error
}

// ImportApplyMsg is sent to the parent model when the user confirms an import.
// The parent is responsible for merging quantities into its tempQuantityChanges.
type ImportApplyMsg struct {
	Quantities map[int64]int
	Result     *export.ImportResult
}

// importCancelMsg is sent when the user cancels the import at any phase.
type importCancelMsg struct{}

// ImportState manages the full lifecycle of a CSV import action inside a TUI view.
type ImportState struct {
	active     bool
	phase      importPhase
	pathInput  textinput.Model
	result     *export.ImportResult
	quantities map[int64]int
	errMsg     string
	cardSvc    cardservice.CardService
}

// NewImportState creates a ready-to-use ImportState.
func NewImportState(cardSvc cardservice.CardService, sm *StyleManager) ImportState {
	ti := textinput.New()
	ti.Placeholder = "path/to/deck.csv"
	ti.Width = 50
	ti.CharLimit = 256
	sm.ApplyTextInputStyles(&ti)
	ti.Focus()
	return ImportState{
		active:    true,
		phase:     importPhaseInput,
		pathInput: ti,
		cardSvc:   cardSvc,
	}
}

// HandleKey processes a key event and returns any commands to run.
func (s *ImportState) HandleKey(key string) tea.Cmd {
	if !s.active {
		return nil
	}

	switch s.phase {
	case importPhaseInput:
		return s.handleInputPhaseKey(key)
	case importPhaseWorking:
		// only allow quit during loading
		if key == "esc" || key == "ctrl+c" {
			s.active = false
			return func() tea.Msg { return importCancelMsg{} }
		}
	case importPhaseResult:
		return s.handleResultPhaseKey(key)
	}
	return nil
}

func (s *ImportState) handleInputPhaseKey(key string) tea.Cmd {
	switch key {
	case "esc":
		s.active = false
		return func() tea.Msg { return importCancelMsg{} }
	case "enter":
		path := strings.TrimSpace(s.pathInput.Value())
		if path == "" {
			return nil
		}
		s.phase = importPhaseWorking
		cardSvc := s.cardSvc
		return func() tea.Msg {
			rows, err := export.FromCSV(path)
			if err != nil {
				return importReadyMsg{err: err}
			}
			lookup := func(setCode, number string) (int64, error) {
				card, err := cardSvc.GetCardBySetCodeAndNumber(setCode, number)
				if err != nil {
					return 0, err
				}
				if card == nil {
					return 0, nil
				}
				return card.ID, nil
			}
			result, quantities, err := export.ResolveCSVToCardQuantities(rows, lookup)
			if err != nil {
				return importReadyMsg{err: err}
			}
			return importReadyMsg{result: result, quantities: quantities}
		}
	default:
		var cmd tea.Cmd
		s.pathInput, cmd = s.pathInput.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		return cmd
	}
}

func (s *ImportState) handleResultPhaseKey(key string) tea.Cmd {
	switch key {
	case "esc", "n", "q":
		s.active = false
		return func() tea.Msg { return importCancelMsg{} }
	case "enter", "y":
		if s.result == nil || s.result.Imported == 0 {
			s.active = false
			return func() tea.Msg { return importCancelMsg{} }
		}
		result := s.result
		quantities := s.quantities
		s.active = false
		return func() tea.Msg {
			return ImportApplyMsg{Quantities: quantities, Result: result}
		}
	}
	return nil
}

// HandleTextInput passes a full tea.KeyMsg to the text input (for characters, backspace, etc.)
func (s *ImportState) HandleTextInput(msg tea.KeyMsg) tea.Cmd {
	if !s.active || s.phase != importPhaseInput {
		return nil
	}
	var cmd tea.Cmd
	s.pathInput, cmd = s.pathInput.Update(msg)
	return cmd
}

// HandleResult processes the async importReadyMsg.
func (s *ImportState) HandleResult(msg importReadyMsg) {
	if msg.err != nil {
		s.errMsg = FailureIcon + " " + msg.err.Error()
		s.phase = importPhaseResult
		s.result = &export.ImportResult{}
		return
	}
	s.result = msg.result
	s.quantities = msg.quantities
	s.phase = importPhaseResult
	s.errMsg = ""
}

// Render returns the string to display in the footer/overlay area.
func (s ImportState) Render(sm *StyleManager) string {
	if !s.active {
		return ""
	}

	var b strings.Builder

	switch s.phase {
	case importPhaseInput:
		b.WriteString(sm.GetTitleStyle().Render("Import CSV") + "\n")
		b.WriteString(sm.GetBlurredStyle().Render("File path: ") + s.pathInput.View() + "\n")
		b.WriteString(sm.GetBlurredStyle().Render("Enter: Load • Esc: Cancel"))

	case importPhaseWorking:
		b.WriteString(sm.GetBlurredStyle().Render(ImportIcon + " Reading CSV and matching cards..."))

	case importPhaseResult:
		if s.errMsg != "" {
			b.WriteString(sm.GetErrorStyle().Render(s.errMsg) + "\n")
			b.WriteString(sm.GetBlurredStyle().Render("Esc: Close"))
		} else {
			b.WriteString(sm.GetTitleStyle().Render("Import Summary") + "\n")
			b.WriteString(sm.GetNoStyle().Render(s.result.Summary()) + "\n")
			if len(s.result.SkippedRows) > 0 {
				b.WriteString(sm.GetBlurredStyle().Render("Skipped:") + "\n")
				limit := len(s.result.SkippedRows)
				if limit > 5 {
					limit = 5
				}
				for _, row := range s.result.SkippedRows[:limit] {
					b.WriteString(sm.GetBlurredStyle().Render("  • "+row.Name+" ("+row.SetCode+" "+row.Number+")") + "\n")
				}
				if len(s.result.SkippedRows) > 5 {
					remaining := len(s.result.SkippedRows) - 5
					b.WriteString(sm.GetBlurredStyle().Render("  ... and "+strconv.Itoa(remaining)+" more") + "\n")
				}
			}
			if s.result.Imported > 0 {
				b.WriteString(sm.GetBlurredStyle().Render("Enter/Y: Apply • Esc/N: Cancel"))
			} else {
				b.WriteString(sm.GetBlurredStyle().Render("No matching cards found. Esc: Close"))
			}
		}
	}

	return b.String()
}
