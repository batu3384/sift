package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestPanelThemesNormalizesNamesAndBrightensActiveBackground(t *testing.T) {
	inactive := PanelThemes("spotlight", false)
	active := PanelThemes("SPOTLIGHT", true)

	if inactive.BorderColor != lipgloss.Color("#44616C") {
		t.Fatalf("inactive spotlight border = %q, want #44616C", inactive.BorderColor)
	}
	if active.BorderColor != inactive.BorderColor {
		t.Fatalf("active spotlight border = %q, want normalized spotlight border %q", active.BorderColor, inactive.BorderColor)
	}
	if active.BackgroundColor == inactive.BackgroundColor {
		t.Fatalf("active spotlight background should be brightened from %q", inactive.BackgroundColor)
	}
}

func TestCardThemeForToneAppliesToneAndLabelOverrides(t *testing.T) {
	review := CardThemeForTone("update", "review")
	if review.BorderColor != lipgloss.Color("#8A734E") {
		t.Fatalf("review update border = %q, want #8A734E", review.BorderColor)
	}

	health := CardThemeForTone("health", "safe")
	if health.LabelColor != lipgloss.Color("#B7C9CF") {
		t.Fatalf("health label color = %q, want #B7C9CF", health.LabelColor)
	}
}

func TestRouteCardThemeNormalizesRouteAliases(t *testing.T) {
	clean := RouteCardTheme("clean", "cache", "safe")
	sweep := RouteCardTheme("SWEEP", "cache", "safe")
	if clean != sweep {
		t.Fatalf("clean route theme = %#v, sweep alias theme = %#v", clean, sweep)
	}

	unknown := RouteCardTheme("unknown", "cache", "safe")
	base := CardThemeForTone("cache", "safe")
	if unknown != base {
		t.Fatalf("unknown route theme = %#v, want base tone theme %#v", unknown, base)
	}
}

func TestRouteLabelTokenDocumentsKnownRouteTokens(t *testing.T) {
	tests := map[string]string{
		"home":      "SCOUT",
		"status":    "OBS",
		"clean":     "FORGE",
		"uninstall": "COURIER",
		"analyze":   "ORACLE",
		"progress":  "ACTION",
		"result":    "SETTLED",
		"review":    "GATE",
		"preflight": "ACCESS",
		"tools":     "UTILITY",
		"doctor":    "DIAG",
		"protect":   "GUARD",
		"missing":   "",
	}

	for route, want := range tests {
		if got := RouteLabelToken(route); got != want {
			t.Fatalf("RouteLabelToken(%q) = %q, want %q", route, got, want)
		}
	}
}
