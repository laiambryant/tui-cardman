package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/laiambryant/tui-cardman/internal/export"
)

type ExportFormat int

const (
	ExportCSV ExportFormat = iota
	ExportText
	ExportPTCGO
)

type ExportState struct {
	active      bool
	formatIndex int
	formats     []ExportFormat
	statusMsg   string
	exportType  string
	exportName  string
	buildRowsFn func() []export.CardRow
	isDeck      bool
	deckName    string
}

type exportDoneMsg struct {
	filepath string
	err      error
}

func NewExportState(exportType, exportName string, isDeck bool, deckName string, buildRowsFn func() []export.CardRow) ExportState {
	formats := []ExportFormat{ExportCSV, ExportText}
	if isDeck {
		formats = append(formats, ExportPTCGO)
	}
	return ExportState{
		active:      true,
		formats:     formats,
		exportType:  exportType,
		exportName:  exportName,
		buildRowsFn: buildRowsFn,
		isDeck:      isDeck,
		deckName:    deckName,
	}
}

func (e *ExportState) HandleKey(s string) tea.Cmd {
	if !e.active {
		return nil
	}
	if s == "esc" {
		e.active = false
		e.statusMsg = ""
		return nil
	}
	if s == "left" {
		if e.formatIndex > 0 {
			e.formatIndex--
		}
		return nil
	}
	if s == "right" {
		if e.formatIndex < len(e.formats)-1 {
			e.formatIndex++
		}
		return nil
	}
	if s == "enter" {
		rows := e.buildRowsFn()
		format := e.formats[e.formatIndex]
		return e.executeExport(rows, format)
	}
	return nil
}

func (e *ExportState) executeExport(rows []export.CardRow, format ExportFormat) tea.Cmd {
	return func() tea.Msg {
		var ext string
		switch format {
		case ExportCSV:
			ext = "csv"
		case ExportText:
			ext = "txt"
		case ExportPTCGO:
			ext = "ptcgo"
		}
		filepath := export.GenerateFilename(e.exportType, e.exportName, ext)
		var err error
		switch format {
		case ExportCSV:
			err = export.ToCSV(rows, filepath)
		case ExportText:
			err = export.ToText(rows, filepath)
		case ExportPTCGO:
			err = export.ToPTCGO(e.deckName, rows, filepath)
		}
		return exportDoneMsg{filepath: filepath, err: err}
	}
}

func (e *ExportState) HandleResult(msg exportDoneMsg) {
	e.active = false
	if msg.err != nil {
		e.statusMsg = FailureIcon + " Export failed: " + msg.err.Error()
	} else {
		e.statusMsg = SuccessIcon + " Exported to " + msg.filepath
	}
}

func (e ExportState) Render(sm *StyleManager) string {
	if !e.active {
		return ""
	}
	var s string
	s += sm.GetTitleStyle().Render("Export as:") + " "
	for i, f := range e.formats {
		label := formatLabel(f)
		if i == e.formatIndex {
			s += sm.GetFocusedStyle().Render("< " + label + " >")
		} else {
			s += sm.GetBlurredStyle().Render(" " + label + " ")
		}
	}
	s += "\n" + sm.GetBlurredStyle().Render("←/→ Select • Enter Confirm • Esc Cancel")
	return s
}

func formatLabel(f ExportFormat) string {
	switch f {
	case ExportCSV:
		return "CSV"
	case ExportText:
		return "Text"
	case ExportPTCGO:
		return "PTCGO"
	}
	return ""
}
