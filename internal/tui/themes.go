package tui

import "github.com/charmbracelet/lipgloss"

const DefaultThemeName = "aurora"

// ThemeNames returns the available theme names in display order.
func ThemeNames() []string {
	return []string{"aurora", "ocean", "daylight", "classic"}
}

var themes = map[string]colorScheme{
	"aurora": {
		Focused:                 lipgloss.Color("#FF5FD2"),
		Title:                   lipgloss.Color("#9D7CFF"),
		Secondary:               lipgloss.Color("#5EEAD4"),
		Blurred:                 lipgloss.Color("#8189B0"),
		Subtle:                  lipgloss.Color("#44486B"),
		Error:                   lipgloss.Color("#FF5C57"),
		Success:                 lipgloss.Color("#5AF78E"),
		Warning:                 lipgloss.Color("#F9C846"),
		Background:              lipgloss.Color(""),
		Foreground:              lipgloss.Color(""),
		TableSelectedForeground: lipgloss.Color("#1A1B26"),
		TableSelectedBackground: lipgloss.Color("#FF5FD2"),
		Disabled:                lipgloss.Color("#565A7D"),
	},
	"ocean": {
		Focused:                 lipgloss.Color("#38BDF8"),
		Title:                   lipgloss.Color("#7DD3FC"),
		Secondary:               lipgloss.Color("#34D399"),
		Blurred:                 lipgloss.Color("#7C8DA6"),
		Subtle:                  lipgloss.Color("#2E4A66"),
		Error:                   lipgloss.Color("#F87171"),
		Success:                 lipgloss.Color("#4ADE80"),
		Warning:                 lipgloss.Color("#FBBF24"),
		Background:              lipgloss.Color(""),
		Foreground:              lipgloss.Color(""),
		TableSelectedForeground: lipgloss.Color("#0B1B2B"),
		TableSelectedBackground: lipgloss.Color("#38BDF8"),
		Disabled:                lipgloss.Color("#4B617A"),
	},
	"daylight": {
		Focused:                 lipgloss.Color("#D33682"),
		Title:                   lipgloss.Color("#7C3AED"),
		Secondary:               lipgloss.Color("#0E7490"),
		Blurred:                 lipgloss.Color("#6B7280"),
		Subtle:                  lipgloss.Color("#B8B2A7"),
		Error:                   lipgloss.Color("#DC2626"),
		Success:                 lipgloss.Color("#15803D"),
		Warning:                 lipgloss.Color("#B45309"),
		Background:              lipgloss.Color(""),
		Foreground:              lipgloss.Color(""),
		TableSelectedForeground: lipgloss.Color("#FFFFFF"),
		TableSelectedBackground: lipgloss.Color("#D33682"),
		Disabled:                lipgloss.Color("#9CA3AF"),
	},
	"classic": {
		Focused:                 colorMagenta,
		Title:                   colorPurple,
		Secondary:               colorPurple,
		Blurred:                 colorBananaCream,
		Subtle:                  colorBananaCream,
		Error:                   colorRed,
		Success:                 lipgloss.Color("10"),
		Warning:                 lipgloss.Color("11"),
		Background:              lipgloss.Color(""),
		Foreground:              lipgloss.Color(""),
		TableSelectedForeground: lipgloss.Color(""),
		TableSelectedBackground: lipgloss.Color(""),
		Disabled:                colorMintCream,
	},
}

func themeByName(name string) (colorScheme, bool) {
	scheme, ok := themes[name]
	return scheme, ok
}

// ThemeSwatchColors returns representative accent colors used to preview a theme.
func ThemeSwatchColors(name string) []lipgloss.Color {
	scheme, ok := themeByName(name)
	if !ok {
		return nil
	}
	return []lipgloss.Color{scheme.Focused, scheme.Title, scheme.Secondary, scheme.Success, scheme.Warning, scheme.Error}
}
