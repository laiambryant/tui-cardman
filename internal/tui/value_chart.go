package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/laiambryant/tui-cardman/internal/services/usercollection"
)

var barChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func renderValueChart(snapshots []usercollection.ValueSnapshot, width, height int, sm *StyleManager) string {
	if len(snapshots) == 0 {
		return sm.GetBlurredStyle().Render("No value history data yet.")
	}
	chartWidth := width - 10
	if chartWidth < 5 {
		chartWidth = 5
	}
	data := snapshots
	if len(data) > chartWidth {
		data = data[len(data)-chartWidth:]
	}
	minVal := math.MaxFloat64
	maxVal := 0.0
	for _, s := range data {
		if s.Value < minVal {
			minVal = s.Value
		}
		if s.Value > maxVal {
			maxVal = s.Value
		}
	}
	if minVal == maxVal {
		minVal = maxVal * 0.9
		if minVal == 0 {
			maxVal = 1
		}
	}
	valueRange := maxVal - minVal
	chartHeight := height - 2
	if chartHeight < 1 {
		chartHeight = 1
	}
	barStyle := sm.applyBGFG(lipgloss.NewStyle().Foreground(sm.scheme.Focused))
	var b strings.Builder
	for _, s := range data {
		normalized := (s.Value - minVal) / valueRange
		idx := int(normalized * float64(len(barChars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(barChars) {
			idx = len(barChars) - 1
		}
		b.WriteString(barStyle.Render(string(barChars[idx])))
	}
	b.WriteString("\n")
	ns := sm.GetNoStyle()
	b.WriteString(ns.Render(fmt.Sprintf("$%.0f", minVal)))
	gap := chartWidth - 10
	if gap < 0 {
		gap = 0
	}
	b.WriteString(strings.Repeat(" ", gap))
	b.WriteString(ns.Render(fmt.Sprintf("$%.0f", maxVal)))
	return b.String()
}

func calcValueChange(snapshots []usercollection.ValueSnapshot, days int) (float64, bool) {
	if len(snapshots) < 2 {
		return 0, false
	}
	latest := snapshots[len(snapshots)-1]
	target := len(snapshots) - days
	if target < 0 {
		target = 0
	}
	prev := snapshots[target]
	if prev.Value == 0 {
		return 0, false
	}
	change := ((latest.Value - prev.Value) / prev.Value) * 100
	return change, true
}
