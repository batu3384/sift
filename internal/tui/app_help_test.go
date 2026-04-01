package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func TestReviewBindingsUseConfiguredStageKey(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteReview)
	model.keys.Stage = key.NewBinding(key.WithKeys("z"), key.WithHelp("z", "stage"))
	model.review.plan = domain.ExecutionPlan{
		Command: "clean",
		Items: []domain.Finding{{
			ID:     "one",
			Path:   "/tmp/cache",
			Action: domain.ActionTrash,
			Status: domain.StatusPlanned,
		}},
	}

	bindings := model.reviewBindings(false)
	foundToggle := false
	for _, binding := range bindings {
		help := binding.Help()
		if help.Desc != "toggle" {
			continue
		}
		foundToggle = true
		if help.Key != "z" {
			t.Fatalf("expected toggle help to use configured stage key, got %q", help.Key)
		}
		if len(binding.Keys()) != 1 || binding.Keys()[0] != "z" {
			t.Fatalf("expected toggle binding keys to use configured stage key, got %+v", binding.Keys())
		}
	}
	if !foundToggle {
		t.Fatal("expected review bindings to include a toggle binding")
	}
}

func TestCompactProtectBindingsDoNotAdvertiseEnter(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteProtect)
	bindings := model.compactRouteBindings(model.routeBindings().short)

	for _, binding := range bindings {
		for _, keyName := range binding.Keys() {
			if keyName == "enter" {
				t.Fatalf("did not expect compact protect bindings to advertise enter: %+v", bindings)
			}
		}
	}
}

func TestRouteBindingsIncludeHelpShortcut(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	bindings := model.routeBindings().short
	for _, binding := range bindings {
		if binding.Help().Desc == "help" {
			return
		}
	}
	t.Fatal("expected route bindings to advertise help")
}

func TestHelpOverlayViewIncludesRouteSections(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.width = 120
	model.height = 48
	view := model.helpOverlayView()
	for _, snippet := range []string{"Analyze • ? or esc closes", "MOVE", "MARK", "FIND", "FILE", "BACK"} {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected help overlay to contain %q, got:\n%s", snippet, view)
		}
	}
}

func TestAnalyzeFooterShowsSecondaryActionsSummary(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.width = 160
	model.height = 40

	footer := model.footerContent()
	for _, snippet := range []string{"Also:", "tab focus", "/ search", "f filter"} {
		if !strings.Contains(footer, snippet) {
			t.Fatalf("expected analyze footer to contain %q, got %q", snippet, footer)
		}
	}
	if strings.Contains(footer, "o open") {
		t.Fatalf("did not expect analyze footer to advertise lower-priority file actions, got %q", footer)
	}
}

func TestStatusFooterUsesConfiguredCompanionHint(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteStatus)
	model.width = 160
	model.height = 36
	model.keys.Companion = key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "companion"))

	footer := model.footerContent()
	if !strings.Contains(footer, "Also: C companion") {
		t.Fatalf("expected status footer to include configured companion hint, got %q", footer)
	}
}
