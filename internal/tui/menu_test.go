package tui

import (
	"strings"
	"testing"

	"github.com/batuhanyuksel/sift/internal/config"
	"github.com/batuhanyuksel/sift/internal/domain"
)

func TestCleanMenuDetailShowsSelectionAndNextAction(t *testing.T) {
	t.Parallel()

	model := menuModel{actions: buildCleanActions(), cursor: 1}
	view := menuDetailView(model, 72, 16)
	for _, needle := range []string{
		"State",
		"2/3 selected",
		"Next",
		"enter opens review",
		"Workstation Clean",
	} {
		if !strings.Contains(strings.ToLower(view), strings.ToLower(needle)) {
			t.Fatalf("expected %q in clean menu detail, got %s", needle, view)
		}
	}
}

func TestToolsMenuDetailUsesSpecializedBoard(t *testing.T) {
	t.Parallel()

	actions := buildToolsActions(config.Default())
	cursor := 0
	for i, action := range actions {
		if action.ID == "optimize" {
			cursor = i
			break
		}
	}

	view := menuDetailView(menuModel{actions: actions, cursor: cursor}, 72, 16)
	for _, needle := range []string{
		"Optimize",
		"State",
		"3/7 selected",
		"Next",
		"enter opens review",
		"Focus",
		"preflight",
		"Steps",
		"review tasks",
		"Scope",
		"Caches",
		"Risk",
	} {
		if !strings.Contains(strings.ToLower(view), strings.ToLower(needle)) {
			t.Fatalf("expected %q in specialized tools detail, got %s", needle, view)
		}
	}
}

func TestToolsCheckDetailShowsDiagnosticsFlow(t *testing.T) {
	t.Parallel()

	actions := buildToolsActions(config.Default())
	view := menuDetailView(menuModel{actions: actions, cursor: 0}, 72, 16)
	for _, needle := range []string{
		"Check",
		"1/7 selected",
		"enter opens checks",
	} {
		if !strings.Contains(strings.ToLower(view), strings.ToLower(needle)) {
			t.Fatalf("expected %q in tools check detail, got %s", needle, view)
		}
	}
}

func TestToolsMenuRenderMatrix(t *testing.T) {
	t.Parallel()

	actions := buildToolsActions(config.Default())
	cases := []struct {
		width  int
		height int
	}{
		{80, 24},
		{100, 30},
		{144, 40},
	}

	for _, tc := range cases {
		list := homeMenuView(actions, 0, tc.width/2, tc.height)
		detail := menuDetailView(menuModel{actions: actions, cursor: 0}, tc.width/2, tc.height)
		for _, view := range []string{list, detail} {
			if got := len(strings.Split(view, "\n")); got > tc.height {
				t.Fatalf("expected tools menu render to fit within %d lines, got %d", tc.height, got)
			}
		}
		for _, needle := range []string{"focus", "state", "next"} {
			if !strings.Contains(strings.ToLower(detail), needle) {
				t.Fatalf("expected %q in specialized tools detail in %dx%d, got %s", needle, tc.width, tc.height, detail)
			}
		}
	}
}

func TestCleanMenuDetailShowsLoadedPlanPreview(t *testing.T) {
	t.Parallel()

	model := menuModel{
		actions: buildCleanActions(),
		cursor:  0,
	}
	model.applyPreview("safe", domain.ExecutionPlan{
		Command: "clean",
		Totals: domain.Totals{
			Bytes:       3 * 1024 * 1024,
			SafeBytes:   2 * 1024 * 1024,
			ReviewBytes: 1 * 1024 * 1024,
		},
		Items: []domain.Finding{
			{Name: "Cache A", Path: "/tmp/a", Status: domain.StatusPlanned, Action: domain.ActionTrash, Category: domain.CategorySystemClutter, Bytes: 2 * 1024 * 1024},
			{Name: "Cache B", Path: "/tmp/b", Status: domain.StatusPlanned, Action: domain.ActionTrash, Category: domain.CategoryLogs, Bytes: 1 * 1024 * 1024},
		},
	}, nil)

	view := menuDetailView(model, 72, 16)
	for _, needle := range []string{"Plan", "2 ready", "2 modules", "3.0 MB", "Mix", "2.0 MB safe", "1.0 MB review"} {
		if !strings.Contains(strings.ToLower(view), strings.ToLower(needle)) {
			t.Fatalf("expected %q in clean preview detail, got %s", needle, view)
		}
	}
}
