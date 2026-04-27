package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRouteCardTonePaletteVariesByRoute(t *testing.T) {
	t.Parallel()

	baseBorder, _, _, baseBackground := routeCardTonePalette("", "state", "review")
	cleanBorder, _, _, cleanBackground := routeCardTonePalette("clean", "sweep", "review")
	uninstallBorder, _, _, uninstallBackground := routeCardTonePalette("uninstall", "watch", "high")
	resultBorder, _, _, resultBackground := routeCardTonePalette("result", "freed", "safe")

	if cleanBorder == baseBorder && cleanBackground == baseBackground {
		t.Fatalf("expected clean route palette to differ from base palette, got border=%s background=%s", cleanBorder, cleanBackground)
	}
	if uninstallBorder == baseBorder && uninstallBackground == baseBackground {
		t.Fatalf("expected uninstall route palette to differ from base palette, got border=%s background=%s", uninstallBorder, uninstallBackground)
	}
	if resultBorder == lipgloss.Color("") || resultBackground == lipgloss.Color("") {
		t.Fatalf("expected result route palette to return concrete colors, got border=%s background=%s", resultBorder, resultBackground)
	}
	if cleanBackground == uninstallBackground {
		t.Fatalf("expected clean and uninstall route backgrounds to differ, got %s", cleanBackground)
	}
}

func TestRenderRouteStatCardKeepsReadableTextContract(t *testing.T) {
	t.Parallel()

	rendered := renderRouteStatCard("clean", "sweep", "SCANNING", "review", 108)
	for _, needle := range []string{"FORGE", "SWEEP", "SCANNING", "╺━", "╰─"} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("expected %q in rendered route stat card, got %q", needle, rendered)
		}
	}
}

func TestRenderRouteStatCardKeepsCompactCardsTight(t *testing.T) {
	t.Parallel()

	rendered := renderRouteStatCard("clean", "sweep", "SCANNING", "review", 28)
	for _, needle := range []string{"SWEEP", "SCANNING"} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("expected %q in compact rendered route stat card, got %q", needle, rendered)
		}
	}
	if strings.Contains(rendered, "╺━") {
		t.Fatalf("expected compact route stat card to avoid accent cap, got %q", rendered)
	}
}

func TestRouteCardLabelTokenVariesByRoute(t *testing.T) {
	t.Parallel()

	if got := routeCardLabelToken("home"); got != "SCOUT" {
		t.Fatalf("expected home token SCOUT, got %q", got)
	}
	if got := routeCardLabelToken("status"); got != "OBS" {
		t.Fatalf("expected status token OBS, got %q", got)
	}
	if got := routeCardLabelToken("clean"); got != "FORGE" {
		t.Fatalf("expected clean token FORGE, got %q", got)
	}
	if got := routeCardLabelToken(""); got != "" {
		t.Fatalf("expected empty token for default route, got %q", got)
	}
}

func TestPanelThemeVariesForHomeAndStatusRails(t *testing.T) {
	t.Parallel()

	baseBorder, baseBackground, _, _ := panelTheme("DETAIL", false)
	commandBorder, commandBackground, _, _ := panelTheme("COMMAND DECK", false)
	observatoryBorder, observatoryBackground, _, _ := panelTheme("OBSERVATORY", false)

	if commandBorder == baseBorder && commandBackground == baseBackground {
		t.Fatalf("expected command deck theme to differ from base theme, got border=%s background=%s", commandBorder, commandBackground)
	}
	if observatoryBorder == baseBorder && observatoryBackground == baseBackground {
		t.Fatalf("expected observatory theme to differ from base theme, got border=%s background=%s", observatoryBorder, observatoryBackground)
	}
	if commandBackground == observatoryBackground {
		t.Fatalf("expected command deck and observatory backgrounds to differ, got %s", commandBackground)
	}
}

func TestRenderPanelRuleUsesCappedChrome(t *testing.T) {
	t.Parallel()

	rule := renderPanelRule(lipgloss.Color("#4A6972"), 28)
	for _, needle := range []string{"╺", "╸", "─"} {
		if !strings.Contains(rule, needle) {
			t.Fatalf("expected %q in capped panel rule, got %q", needle, rule)
		}
	}
}
