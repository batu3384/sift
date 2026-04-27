package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"

	"github.com/batu3384/sift/internal/domain"
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
	for _, snippet := range []string{"CONTROL DECK", "Analyze trace rail • ? or esc closes", "MOVE", "MARK", "FIND", "FILE", "BACK"} {
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
	for _, snippet := range []string{"Trace:", "tab focus", "/ search", "f filter"} {
		if !strings.Contains(footer, snippet) {
			t.Fatalf("expected analyze footer to contain %q, got %q", snippet, footer)
		}
	}
	if strings.Contains(footer, "o open") {
		t.Fatalf("did not expect analyze footer to advertise lower-priority file actions, got %q", footer)
	}
}

func TestCleanFooterShowsHistoryHintsWhenLedgerScrollable(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteClean)
	model.width = 200
	model.height = 36
	model.cleanFlow.setPreviewLoading("safe")
	for i := 0; i < 4; i++ {
		model.cleanFlow.applyScanFinding("cache", "Browser cache", domain.Finding{
			ID:          "row-clean-" + string(rune('a'+i)),
			Name:        "Row " + string(rune('A'+i)),
			Path:        "/tmp/row-clean-" + string(rune('a'+i)),
			DisplayPath: "/tmp/row-clean-" + string(rune('a'+i)),
			Category:    domain.CategoryBrowserData,
			Bytes:       int64(i+1) << 20,
		})
	}

	footer := model.footerContent()
	for _, snippet := range []string{"History:", "pgup/pgdn page history", "home/end oldest/live"} {
		if !strings.Contains(footer, snippet) {
			t.Fatalf("expected clean footer to contain %q, got %q", snippet, footer)
		}
	}
}

func TestUninstallFooterShowsHistoryHintsWhenLedgerScrollable(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteUninstall)
	model.width = 200
	model.height = 36
	model.applyInstalledApps([]domain.AppEntry{
		{DisplayName: "Example App", UninstallCommand: "/Applications/Example Uninstaller.app"},
		{DisplayName: "Builder Tool", UninstallCommand: "/Applications/Builder Tool Uninstaller.app"},
	})
	model.uninstallFlow.phase = uninstallFlowInventory

	footer := model.footerContent()
	for _, snippet := range []string{"History:", "pgup/pgdn page history", "home/end oldest/live"} {
		if !strings.Contains(footer, snippet) {
			t.Fatalf("expected uninstall footer to contain %q, got %q", snippet, footer)
		}
	}
}

func TestAnalyzeHelpOverlayIncludesHistorySectionWhenTraceScrollable(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.width = 120
	model.height = 48
	model.analyzeFlow.phase = analyzeFlowInspecting
	model.analyzeFlow.traceRows = []analyzeFlowTraceRow{
		{FindingID: "a", Path: "/tmp/a", Label: "Row A", Category: domain.CategoryDiskUsage, Bytes: 8 << 20, State: "review"},
		{FindingID: "b", Path: "/tmp/b", Label: "Row B", Category: domain.CategoryDiskUsage, Bytes: 12 << 20, State: "review"},
	}

	view := strings.Join(strings.Fields(model.helpOverlayView()), " ")
	for _, snippet := range []string{"HISTORY", "pgup/pgdn page history", "home/end oldest/live"} {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected analyze help overlay to contain %q, got:\n%s", snippet, view)
		}
	}
}

func TestStatusFooterUsesConfiguredCompanionHint(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteStatus)
	model.width = 160
	model.height = 36
	model.keys.Companion = key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "companion"))

	footer := model.footerContent()
	if !strings.Contains(footer, "Watch: C companion") {
		t.Fatalf("expected status footer to include configured companion hint, got %q", footer)
	}
}

func TestReviewAndResultFooterUseRouteAwareSecondaryLabels(t *testing.T) {
	t.Parallel()

	review := newTestAppModel(RouteReview)
	review.width = 160
	review.height = 36
	review.review.plan = domain.ExecutionPlan{
		Command: "clean",
		Items: []domain.Finding{
			{ID: "a", Path: "/tmp/a", DisplayPath: "/tmp/a", Action: domain.ActionTrash, Status: domain.StatusPlanned, Category: domain.CategoryBrowserData, Source: "Chrome code cache"},
		},
	}
	if footer := review.footerContent(); !strings.Contains(footer, "Gate: m module") {
		t.Fatalf("expected review footer to use gate label, got %q", footer)
	}

	result := newTestAppModel(RouteResult)
	result.width = 160
	result.height = 36
	result.result.plan = domain.ExecutionPlan{
		Command: "clean",
		Items: []domain.Finding{
			{ID: "a", Path: "/tmp/a", DisplayPath: "/tmp/a", Action: domain.ActionTrash, Status: domain.StatusPlanned, Category: domain.CategoryBrowserData, Source: "Chrome code cache"},
		},
	}
	result.result.result = domain.ExecutionResult{
		Items: []domain.OperationResult{
			{FindingID: "a", Path: "/tmp/a", Status: domain.StatusFailed},
		},
	}
	if footer := result.footerContent(); !strings.Contains(footer, "Recovery:") || !strings.Contains(footer, "retry") {
		t.Fatalf("expected result footer to use recovery label, got %q", footer)
	}
}
