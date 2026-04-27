// Package design provides the modern layout system for SIFT TUI.
// Responsive layouts with proper spacing and structure.
package design

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Layout constants
const (
	MinWidth        = 80
	MinHeight       = 24
	DefaultWidth    = 120
	DefaultHeight   = 36
	CompactWidth    = 100
	CompactHeight   = 26
)

// EffectiveSize ensures dimensions are within acceptable bounds
func EffectiveSize(width, height int) (int, int) {
	if width <= 0 {
		width = DefaultWidth
	}
	if height <= 0 {
		height = DefaultHeight
	}
	if width < MinWidth {
		width = MinWidth
	}
	if height < MinHeight {
		height = MinHeight
	}
	return width, height
}

// IsCompactWidth returns true if width indicates compact mode
func IsCompactWidth(width int) bool {
	return width < CompactWidth
}

// IsCompactHeight returns true if height indicates compact mode
func IsCompactHeight(height int) bool {
	return height < CompactHeight
}

// IsCompact returns true if either dimension indicates compact mode
func IsCompact(width, height int) bool {
	return IsCompactWidth(width) || IsCompactHeight(height)
}

// PanelTheme defines the visual theme for a panel
type PanelTheme struct {
	BorderColor lipgloss.Color
	Background  lipgloss.Color
	TitleColor  lipgloss.Color
	MarkerColor lipgloss.Color
}

// PanelThemes defines themes for different panel types
var PanelThemes = map[string]PanelTheme{
	"default": {
		BorderColor: ColorBorderDefault,
		Background:  ColorSurface,
		TitleColor: ColorTextSecondary,
		MarkerColor: ColorTextMuted,
	},
	"active": {
		BorderColor: ColorBorderAccent,
		Background:  ColorOverlay,
		TitleColor: ColorTextPrimary,
		MarkerColor: ColorAccentPrimary,
	},
	"spotlight": {
		BorderColor: lipgloss.Color("#44616C"),
		Background:  lipgloss.Color("#111A1E"),
		TitleColor:  lipgloss.Color("#D9E7EB"),
		MarkerColor: ColorAccentPrimary,
	},
	"command": {
		BorderColor: lipgloss.Color("#6A7553"),
		Background:  lipgloss.Color("#141812"),
		TitleColor:  lipgloss.Color("#E5E0C8"),
		MarkerColor: ColorWarning,
	},
	"route": {
		BorderColor: lipgloss.Color("#4A6A74"),
		Background:  lipgloss.Color("#10191D"),
		TitleColor:  lipgloss.Color("#DCE8EC"),
		MarkerColor: ColorAccentPrimary,
	},
	"storage": {
		BorderColor: lipgloss.Color("#566E60"),
		Background:  lipgloss.Color("#121814"),
		TitleColor:  lipgloss.Color("#DDE8DE"),
		MarkerColor: ColorSuccess,
	},
	"progress": {
		BorderColor: lipgloss.Color("#7B6742"),
		Background:  lipgloss.Color("#17130E"),
		TitleColor:  lipgloss.Color("#F0DEC0"),
		MarkerColor: ColorWarning,
	},
	"result": {
		BorderColor: lipgloss.Color("#58705F"),
		Background:  lipgloss.Color("#111712"),
		TitleColor:  lipgloss.Color("#DDE7DF"),
		MarkerColor: ColorSuccess,
	},
}

// GetPanelTheme returns the appropriate panel theme
func GetPanelTheme(name string, active bool) PanelTheme {
	themeName := "default"
	if active {
		themeName = "active"
	}

	upper := strings.ToUpper(name)
	switch upper {
	case "SPOTLIGHT", "OVERVIEW":
		themeName = "spotlight"
	case "COMMAND DECK":
		themeName = "command"
	case "ROUTE RAIL", "ROUTE DECK":
		themeName = "route"
	case "STORAGE RAIL":
		themeName = "storage"
	case "PROGRESS RAIL", "REVIEW RAIL", "CHECK RAIL":
		themeName = "progress"
	case "SETTLED RAIL", "SETTLED GATE":
		themeName = "result"
	}

	if theme, ok := PanelThemes[themeName]; ok {
		return theme
	}
	return PanelThemes["default"]
}

// SplitColumns calculates column widths for a split layout
func SplitColumns(totalWidth int, leftRatio float64, minLeft, minRight int) (int, int) {
	if totalWidth <= 0 {
		return minLeft, minRight
	}

	left := int(float64(totalWidth) * leftRatio)
	right := totalWidth - left

	if left < minLeft {
		left = minLeft
		right = totalWidth - left
	}
	if right < minRight {
		right = minRight
		left = totalWidth - right
	}
	if left < minLeft {
		left = minLeft
	}
	if right < minRight {
		right = minRight
	}
	return left, right
}

// JoinHorizontal joins strings horizontally with vertical alignment
func JoinHorizontal(parts ...string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// JoinVertical joins strings vertically with left alignment
func JoinVertical(parts ...string) string {
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// StackPanels stacks panels vertically with spacing
func StackPanels(panels []string, spacing int) string {
	if len(panels) == 0 {
		return ""
	}
	if len(panels) == 1 {
		return panels[0]
	}

	result := panels[0]
	for i := 1; i < len(panels); i++ {
		if spacing > 0 && panels[i] != "" {
			result += strings.Repeat("\n", spacing)
		}
		result += panels[i]
	}
	return result
}

// RenderPanel renders a panel with title and body
func RenderPanel(title, body string, width int, theme PanelTheme) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderColor).
		Background(theme.Background).
		Padding(0, 1).
		Width(width)

	titleStyle := lipgloss.NewStyle().
		Foreground(theme.TitleColor).
		Bold(true)

	markerStyle := lipgloss.NewStyle().
		Foreground(theme.MarkerColor).
		Bold(true)

	marker := markerStyle.Render("▸")
	header := marker + " " + titleStyle.Render(title)

	return style.Width(width).Render(header + "\n" + body)
}

// RenderPanelRule renders a decorative rule within a panel
func RenderPanelRule(borderColor lipgloss.Color, width int) string {
	ruleWidth := width - 10
	if ruleWidth < 8 {
		ruleWidth = 8
	}
	core := strings.Repeat("─", ruleWidth)
	style := lipgloss.NewStyle().Foreground(borderColor)
	return style.Render("╺" + core + "╸")
}

// RenderKeyBar renders a key binding bar
func RenderKeyBar(keys ...string) string {
	if len(keys) == 0 {
		return ""
	}

	parts := make([]string, 0, len(keys)*2-1)
	sep := Components.Muted.Render("  ·  ")

	for i, key := range keys {
		if i > 0 {
			parts = append(parts, sep)
		}

		chunks := strings.SplitN(key, " ", 2)
		if len(chunks) == 1 {
			parts = append(parts, Components.Key.Render(chunks[0]))
		} else {
			parts = append(parts, Components.Key.Render(chunks[0])+" "+Components.Muted.Render(chunks[1]))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

// ViewportLines returns a window of lines centered around cursor
func ViewportLines(lines []string, cursor, maxLines int) []string {
	if maxLines <= 0 || len(lines) == 0 {
		return lines
	}
	if len(lines) <= maxLines {
		return lines
	}

	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(lines) {
		cursor = len(lines) - 1
	}

	// Center cursor with bias toward top
	start := cursor - maxLines/2
	if start < 0 {
		start = 0
	}

	end := start + maxLines
	if end > len(lines) {
		end = len(lines)
		start = end - maxLines
		if start < 0 {
			start = 0
		}
	}

	window := make([]string, end-start)
	copy(window, lines[start:end])

	if start > 0 {
		window[0] = Components.Muted.Render("…") + " " + strings.TrimLeft(window[0], " ")
	}
	if end < len(lines) {
		last := len(window) - 1
		window[last] = strings.TrimRight(window[last], " ") + " " + Components.Muted.Render("…")
	}

	return window
}

// TruncateText truncates text to fit within max width
func TruncateText(text string, max int) string {
	if max <= 0 {
		return text
	}
	// Simple ASCII truncation for now
	if len(text) <= max {
		return text
	}
	if max <= 3 {
		return text[:max]
	}
	return strings.TrimSpace(text[:max-1]) + "…"
}

// SingleLine converts text to a single line
func SingleLine(text string, max int) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")
	return TruncateText(text, max)
}

// BodyLineBudget calculates available lines for body content
func BodyLineBudget(height, reserve, minLines int) int {
	_, effectiveHeight := EffectiveSize(DefaultWidth, height)
	lines := effectiveHeight - reserve
	if lines < minLines {
		return minLines
	}
	return lines
}

// ClipRendered clips content to fit within max lines
func ClipRendered(content string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}

	lines := strings.Split(content, "\n")
	if len(lines) <= maxLines {
		return content
	}

	clipped := make([]string, maxLines)
	copy(clipped, lines[:maxLines])

	last := strings.TrimRight(clipped[maxLines-1], " ")
	if last == "" {
		clipped[maxLines-1] = Components.Muted.Render("…")
	} else {
		clipped[maxLines-1] = TruncateText(last, DefaultWidth/2) + " " + Components.Muted.Render("…")
	}

	return strings.Join(clipped, "\n")
}
