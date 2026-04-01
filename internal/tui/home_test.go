package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/platform"
	"github.com/batu3384/sift/internal/store"
)

func TestHomeViewIncludesSummary(t *testing.T) {
	t.Parallel()
	model := homeModel{
		actions: []homeAction{
			{ID: "analyze_home", Title: "Analyze Home", Description: "Analyze your home directory.", Enabled: true},
		},
		live: &engine.SystemSnapshot{
			HealthScore:       87,
			HealthLabel:       "healthy",
			CPUPercent:        14.2,
			MemoryUsedPercent: 61.4,
			DiskFreeBytes:     1024 * 1024 * 1024,
			OperatorAlerts:    []string{"thermal warm 61.5°C"},
		},
		lastExecution: &store.ExecutionSummary{Completed: 2, Deleted: 1, Protected: 1, Skipped: 3},
		updateNotice:  &engine.UpdateNotice{Available: true, LatestVersion: "v9.9.9", Message: "Update v9.9.9 available. Run `sift update` to review the upgrade path."},
		diagnostics: []platform.Diagnostic{
			{Name: "report_cache", Status: "ok", Message: "/tmp/reports"},
			{Name: "purge_search_paths", Status: "warn", Message: "none configured"},
		},
		cfg:        config.Default(),
		executable: true,
		width:      120,
		height:     28,
	}
	view := model.View()
	for _, needle := range []string{"██████  ██ ███████ ████████", "review mode", "87 / HEALTHY", "LAST", "ALERTS", "HOME", "MENU", "DETAIL", "▸ Analyze Home", "focus analyze home", "Signal", "Focus", "Alerts", "Activity", "Next", "V9.9.9 ready", "thermal warm 61.5°C", "t opens check"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in home view, got %s", needle, view)
		}
	}
	if got := len(strings.Split(view, "\n")); got > model.height {
		t.Fatalf("expected home view to stay within %d lines, got %d", model.height, got)
	}
}

func TestHomeSelectsAction(t *testing.T) {
	t.Parallel()
	model := homeModel{
		actions: []homeAction{
			{ID: "analyze_home", Title: "Analyze Home", Description: "Analyze your home directory.", Enabled: true},
			{ID: "quit", Title: "Quit", Description: "Exit.", Enabled: true},
		},
	}
	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	selected, cmd := next.(homeModel).Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected quit command after selecting an action")
	}
	if selected.(homeModel).selected != "quit" {
		t.Fatalf("expected quit selection, got %+v", selected)
	}
}

func TestHomeViewCompactsWithinSmallTerminal(t *testing.T) {
	t.Parallel()
	model := homeModel{
		actions: []homeAction{
			{ID: "analyze", Title: "Analyze", Description: "Map large folders before cleaning.", Enabled: true, Tone: "review"},
			{ID: "clean", Title: "Clean", Description: "Choose what to clean, then review.", Enabled: true, Tone: "safe"},
		},
		live: &engine.SystemSnapshot{
			HealthScore:       87,
			HealthLabel:       "healthy",
			CPUPercent:        14.2,
			MemoryUsedPercent: 61.4,
			DiskFreeBytes:     1024 * 1024 * 1024,
			OperatorAlerts:    []string{"thermal warm 61.5°C"},
		},
		width:  100,
		height: 24,
	}
	view := model.View()
	for _, needle := range []string{"SIFT", "HOME", "MENU", "DETAIL"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in compact home view, got %s", needle, view)
		}
	}
	if got := len(strings.Split(view, "\n")); got > model.height {
		t.Fatalf("expected compact home view to stay within %d lines, got %d", model.height, got)
	}
}

func TestHomeViewRenderMatrix(t *testing.T) {
	t.Parallel()

	base := homeModel{
		actions: []homeAction{
			{ID: "analyze", Title: "Analyze", Description: "Map large folders before cleaning.", Command: "sift analyze", Enabled: true, Tone: "review", Safety: "Read-only drill-down.", When: "Use when you need a reclaim plan."},
			{ID: "clean", Title: "Clean", Description: "Choose what to clean, then review.", Command: "clean workspace", Enabled: true, Tone: "safe", Safety: "Review-first cleanup flow.", When: "Use for routine cleanup and reclaim work."},
		},
		live: &engine.SystemSnapshot{
			HealthScore:       87,
			HealthLabel:       "healthy",
			CPUPercent:        14.2,
			MemoryUsedPercent: 61.4,
			DiskFreeBytes:     1024 * 1024 * 1024,
			OperatorAlerts:    []string{"thermal warm 61.5°C"},
		},
		lastExecution: &store.ExecutionSummary{Completed: 2, Deleted: 1, Protected: 1, Skipped: 3},
		updateNotice:  &engine.UpdateNotice{Available: true, LatestVersion: "v9.9.9", Message: "Update v9.9.9 available. Run `sift update` to review the upgrade path."},
		diagnostics: []platform.Diagnostic{
			{Name: "report_cache", Status: "ok", Message: "/tmp/reports"},
			{Name: "purge_search_paths", Status: "warn", Message: "none configured"},
		},
		cfg:        config.Default(),
		executable: true,
	}

	for _, tc := range []struct {
		name    string
		width   int
		height  int
		needles []string
	}{
		{name: "80x24", width: 80, height: 24, needles: []string{"SIFT", "HOME", "MENU", "DETAIL"}},
		{name: "100x30", width: 100, height: 30, needles: []string{"HOME", "MENU", "DETAIL", "Signal", "Focus", "Alerts", "Activity", "Next"}},
		{name: "144x40", width: 144, height: 40, needles: []string{"██████", "HOME", "MENU", "DETAIL", "Signal", "Alerts", "Activity", "Live", "Next", "t opens check"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := base
			model.width = tc.width
			model.height = tc.height
			view := model.View()
			for _, needle := range tc.needles {
				if !strings.Contains(view, needle) {
					t.Fatalf("expected %q in %s home view, got %s", needle, tc.name, view)
				}
			}
			if got := len(strings.Split(view, "\n")); got > model.height {
				t.Fatalf("expected %s home view to stay within %d lines, got %d", tc.name, model.height, got)
			}
		})
	}
}
