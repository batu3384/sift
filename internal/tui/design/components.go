// Package design provides the modern component system for SIFT TUI.
// Professional UI components with consistent styling and behavior.
package design

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Component styles - reusable UI building blocks
type ComponentStyles struct {
	// Surface styles
	Surface      lipgloss.Style
	SurfaceAlt   lipgloss.Style
	Overlay      lipgloss.Style

	// Text styles
	Title       lipgloss.Style
	Heading     lipgloss.Style
	Body        lipgloss.Style
	Caption     lipgloss.Style
	Code        lipgloss.Style

	// Semantic tone styles
	Safe        lipgloss.Style
	Review      lipgloss.Style
	High        lipgloss.Style
	Muted       lipgloss.Style

	// Badge styles
	SafeBadge   lipgloss.Style
	ReviewBadge lipgloss.Style
	HighBadge   lipgloss.Style

	// Interactive styles
	Button      lipgloss.Style
	ButtonActive lipgloss.Style
	Key         lipgloss.Style

	// Panel styles
	Panel       lipgloss.Style
	PanelActive lipgloss.Style
	Card        lipgloss.Style
}

// NewComponentStyles creates the complete component style system
func NewComponentStyles() ComponentStyles {
	return ComponentStyles{
		// Surface styles
		Surface: lipgloss.NewStyle().
			Background(ColorSurface).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorderDefault).
			Padding(1, 2),

		SurfaceAlt: lipgloss.NewStyle().
			Background(ColorOverlay).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorderMuted).
			Padding(1, 2),

		Overlay: lipgloss.NewStyle().
			Background(ColorOverlay).
			Padding(1, 2),

		// Text styles
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorTextPrimary),

		Heading: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorTextSecondary),

		Body: lipgloss.NewStyle().
			Foreground(ColorTextPrimary),

		Caption: lipgloss.NewStyle().
			Foreground(ColorTextMuted),

		Code: lipgloss.NewStyle().
			Foreground(ColorAccentPrimary),

		// Semantic tone styles - bold variants for tokens
		Safe: lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true),

		Review: lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true),

		High: lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true),

		Muted: lipgloss.NewStyle().
			Foreground(ColorTextMuted),

		// Badge styles
		SafeBadge: lipgloss.NewStyle().
			Foreground(ColorSurface).
			Background(ColorSuccess).
			Bold(true).
			Padding(0, 1),

		ReviewBadge: lipgloss.NewStyle().
			Foreground(ColorSurface).
			Background(ColorWarning).
			Bold(true).
			Padding(0, 1),

		HighBadge: lipgloss.NewStyle().
			Foreground(ColorSurface).
			Background(ColorDanger).
			Bold(true).
			Padding(0, 1),

		// Interactive styles
		Button: lipgloss.NewStyle().
			Foreground(ColorTextPrimary).
			Background(ColorSurface).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorderDefault).
			Padding(0, 2).
			Bold(true),

		ButtonActive: lipgloss.NewStyle().
			Foreground(ColorSelectionText).
			Background(ColorAccentPrimary).
			Border(lipgloss.RoundedBorder()).
			Padding(0, 2).
			Bold(true),

		Key: lipgloss.NewStyle().
			Foreground(ColorSurface).
			Background(ColorAccentPrimary).
			Bold(true).
			Padding(0, 1),

		// Panel styles
		Panel: lipgloss.NewStyle().
			Background(ColorSurface).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorderDefault).
			Padding(0, 1),

		PanelActive: lipgloss.NewStyle().
			Background(ColorOverlay).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorderAccent).
			Padding(0, 1),

		Card: lipgloss.NewStyle().
			Background(ColorSurface).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorderMuted).
			Padding(0, 1),
	}
}

// Global component styles instance
var Components = NewComponentStyles()

// ToneFromStatus converts a status string to a tone
func ToneFromStatus(status string) string {
	switch status {
	case "safe", "completed", "deleted", "success":
		return "safe"
	case "review", "pending", "skipped", "protected":
		return "review"
	case "high", "error", "failed", "critical":
		return "high"
	default:
		return "review"
	}
}

// ToneStyle returns the appropriate style for a tone
func ToneStyle(tone string) lipgloss.Style {
	switch tone {
	case "safe":
		return Components.Safe
	case "review":
		return Components.Review
	case "high":
		return Components.High
	default:
		return Components.Muted
	}
}

// ToneBadge returns the appropriate badge style for a tone
func ToneBadge(tone string) lipgloss.Style {
	switch tone {
	case "safe":
		return Components.SafeBadge
	case "review":
		return Components.ReviewBadge
	case "high":
		return Components.HighBadge
	default:
		return Components.ReviewBadge
	}
}

// ProgressBar renders a character-based progress bar
func ProgressBar(current, total int, width int, tone string) string {
	if width <= 0 {
		width = 20
	}
	if total <= 0 {
		total = 1
	}

	filled := int(float64(current) / float64(total) * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	empty := width - filled

	bar := ToneStyle(tone).Render(string(makeRepeated('█', filled))) +
		Components.Muted.Render(string(makeRepeated('░', empty)))

	return "[" + bar + "]"
}

func makeRepeated(r rune, count int) []rune {
	result := make([]rune, count)
	for i := range result {
		result[i] = r
	}
	return result
}

// Spinner returns the current frame of an animated spinner
func Spinner(frameIndex int, reducedMotion bool) string {
	if reducedMotion {
		return "●"
	}
	frames := SpinnerDotsFrames
	return frames[frameIndex%len(frames)]
}

// PulseIndicator returns a pulsing animation indicator
func PulseIndicator(phase int, reducedMotion bool) string {
	if reducedMotion {
		return "-"
	}
	phases := []string{"<", "<<", "<>", ">>"}
	return phases[phase%len(phases)]
}

// LoadingDots returns animated loading dots
func LoadingDots(frameIndex int, reducedMotion bool) string {
	if reducedMotion {
		return "..."
	}
	dots := []string{"   ", ".  ", ".. ", "..."}
	return dots[frameIndex%len(dots)]
}

// StateIndicator returns a state indicator glyph
func StateIndicator(state string, reducedMotion bool) string {
	switch state {
	case "loading", "scanning":
		return Spinner(0, reducedMotion)
	case "success", "completed", "done":
		return Components.Safe.Render("✓")
	case "error", "failed":
		return Components.High.Render("✗")
	case "warning", "review":
		return Components.Review.Render("!")
	case "pending", "idle":
		if reducedMotion {
			return "○"
		}
		return PulseIndicator(0, reducedMotion)
	default:
		return Components.Muted.Render("·")
	}
}

// TimerDisplay formats duration for display
func TimerDisplay(duration time.Duration) string {
	if duration < time.Second {
		return "<1s"
	}
	if duration < time.Minute {
		seconds := int(duration.Seconds())
		return formatInt(seconds) + "s"
	}
	minutes := int(duration.Minutes())
	seconds := int(duration.Seconds()) % 60
	return formatInt(minutes) + "m " + formatInt(seconds) + "s"
}

func formatInt(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

// SizeDisplay formats byte size for display
func SizeDisplay(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return formatInt(int(bytes)) + "B"
	}
	value := float64(bytes) / float64(unit)

	if value < unit {
		return formatFloat(value, 1) + "K"
	}
	value /= float64(unit)
	if value < unit {
		return formatFloat(value, 1) + "M"
	}
	value /= float64(unit)
	return formatFloat(value, 1) + "G"
}

func formatFloat(f float64, decimals int) string {
	intPart := int(f)
	decPart := int((f - float64(intPart)) * 10)
	if decimals > 1 {
		decPart = decPart*10 + int((f-float64(intPart)-float64(decPart)/10)*100)
	}

	if decimals == 1 {
		return string(rune('0'+intPart)) + "." + string(rune('0'+decPart))
	}
	return string(rune('0'+intPart)) + ".0" + string(rune('0'+decPart))
}

// Route tokens for different app sections
var RouteTokens = map[string]string{
	"home":      "SCOUT",
	"status":    "OBS",
	"clean":     "FORGE",
	"analyze":   "ORACLE",
	"uninstall": "COURIER",
	"progress":  "ACTION",
	"result":    "SETTLED",
	"review":    "GATE",
	"preflight": "ACCESS",
	"tools":     "UTILITY",
	"doctor":    "DIAG",
	"protect":   "GUARD",
}

// RouteToken returns the token for a route
func RouteToken(route string) string {
	if token, ok := RouteTokens[route]; ok {
		return token
	}
	return strings.ToUpper(route)
}

// NormalizeRoute normalizes route names to standard form
func NormalizeRoute(route string) string {
	switch route {
	case "home", "command deck":
		return "home"
	case "status", "observatory":
		return "status"
	case "clean", "sweep":
		return "clean"
	case "uninstall", "removal":
		return "uninstall"
	case "analyze", "inspect":
		return "analyze"
	case "progress":
		return "progress"
	case "result", "settled":
		return "result"
	case "review":
		return "review"
	case "preflight", "access":
		return "preflight"
	case "tools", "menu":
		return "tools"
	case "doctor", "diagnosis":
		return "doctor"
	case "protect", "guard":
		return "protect"
	default:
		return route
	}
}
