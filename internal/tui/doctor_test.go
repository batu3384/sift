package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/batuhanyuksel/sift/internal/platform"
)

func TestDoctorViewIncludesSummary(t *testing.T) {
	t.Parallel()

	model := doctorModel{
		diagnostics: []platform.Diagnostic{
			{Name: "config", Status: "ok", Message: "ready"},
			{Name: "test_policy", Status: "ok", Message: "ci-safe guards active"},
			{Name: "reports", Status: "warn", Message: "missing cache dir"},
			{Name: "audit", Status: "error", Message: "permission denied"},
		},
		width:  140,
		height: 28,
	}

	view := model.View()
	lowerView := strings.ToLower(view)
	for _, needle := range []string{"DOCTOR", "OK 2  WARN 1", "ERROR 1", "▸ ✓ OK", "Checks", "Everything looks healthy", "permission", "ci-safe", "guards active", "security", "updates", "config", "health"} {
		target := needle
		if needle == "security" || needle == "updates" || needle == "config" || needle == "health" || needle == "permission" || needle == "ci-safe" || needle == "guards active" {
			target = strings.ToLower(needle)
			if !strings.Contains(lowerView, target) {
				t.Fatalf("expected %q in doctor view, got %s", needle, view)
			}
			continue
		}
		if !strings.Contains(view, target) {
			t.Fatalf("expected %q in doctor view, got %s", needle, view)
		}
	}
}

func TestDoctorDetailShowsLaneAndFixHint(t *testing.T) {
	t.Parallel()

	view := doctorDetailView([]platform.Diagnostic{
		{Name: "firewall", Status: "warn", Message: "disabled"},
	}, 0, 64, 16)

	for _, needle := range []string{"Lane", "security", "Next", "sift autofix"} {
		if !strings.Contains(strings.ToLower(view), strings.ToLower(needle)) {
			t.Fatalf("expected %q in doctor detail, got %s", needle, view)
		}
	}
}

func TestDoctorExitsOnEnter(t *testing.T) {
	t.Parallel()

	model := doctorModel{
		diagnostics: []platform.Diagnostic{{Name: "config", Status: "ok", Message: "ready"}},
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected enter to return quit command")
	}
	if next.(doctorModel).cursor != 0 {
		t.Fatal("expected cursor to stay in place")
	}
}

func TestDoctorViewRenderMatrix(t *testing.T) {
	t.Parallel()

	base := doctorModel{
		diagnostics: []platform.Diagnostic{
			{Name: "firewall", Status: "warn", Message: "disabled"},
			{Name: "brew_updates", Status: "ok", Message: "up to date"},
			{Name: "touchid", Status: "warn", Message: "migration needed"},
			{Name: "test_policy", Status: "ok", Message: "ci-safe guards active"},
			{Name: "store", Status: "ok", Message: "ready"},
		},
	}

	cases := []struct {
		width  int
		height int
	}{
		{80, 24},
		{100, 30},
		{144, 40},
	}

	for _, tc := range cases {
		model := base
		model.width = tc.width
		model.height = tc.height
		view := model.View()
		for _, needle := range []string{"DOCTOR", "Checks", "firewall"} {
			if !strings.Contains(strings.ToLower(view), strings.ToLower(needle)) {
				t.Fatalf("expected %q in doctor view for %dx%d, got %s", needle, tc.width, tc.height, view)
			}
		}
		if got := len(strings.Split(view, "\n")); got > tc.height {
			t.Fatalf("expected doctor view to fit within %d lines, got %d", tc.height, got)
		}
	}
}
