package engine

import (
	"strings"
	"testing"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func TestOptimizeSuggestedBySortsMatchingChecks(t *testing.T) {
	task := domain.MaintenanceTask{
		ID:                "maintenance.refresh",
		SuggestedByChecks: []string{"check.swap_pressure", "check.disk_pressure", "check.unknown"},
	}
	active := map[string]domain.CheckItem{
		"check.swap_pressure": {ID: "check.swap_pressure", Name: "Swap pressure"},
		"check.disk_pressure": {ID: "check.disk_pressure", Name: "Disk pressure"},
	}

	got := optimizeSuggestedBy(task, active)
	if len(got) != 2 {
		t.Fatalf("expected 2 suggested checks, got %v", got)
	}
	if got[0] != "Disk pressure" || got[1] != "Swap pressure" {
		t.Fatalf("expected sorted suggested checks, got %v", got)
	}
}

func TestOptimizePreflightSummaryTruncatesLongLists(t *testing.T) {
	active := map[string]domain.CheckItem{
		"a": {ID: "a", Name: "Alpha"},
		"b": {ID: "b", Name: "Beta"},
		"c": {ID: "c", Name: "Gamma"},
		"d": {ID: "d", Name: "Delta"},
		"e": {ID: "e", Name: "Epsilon"},
	}

	got := optimizePreflightSummary(domain.CheckReport{}, active)
	if !strings.Contains(got, "Preflight found 5 actionable checks:") {
		t.Fatalf("unexpected summary prefix: %q", got)
	}
	if !strings.Contains(got, "+1 more") {
		t.Fatalf("expected truncated remainder marker, got %q", got)
	}
	if !strings.Contains(got, "Alpha") || !strings.Contains(got, "Epsilon") {
		t.Fatalf("expected named checks in summary, got %q", got)
	}
}

func TestExpandMaintenanceTaskTargetsReportsInvalidGlob(t *testing.T) {
	task := domain.MaintenanceTask{
		Title:     "Bad glob",
		PathGlobs: []string{"["},
	}

	paths, warnings := expandMaintenanceTaskTargets(task)
	if len(paths) != 0 {
		t.Fatalf("expected no expanded paths, got %v", paths)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "invalid optimize glob") {
		t.Fatalf("expected invalid glob warning, got %v", warnings)
	}
}
