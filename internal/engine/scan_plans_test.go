package engine

import (
	"testing"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func TestSortPlanItemsPrioritizesActionableBeforeAdvisory(t *testing.T) {
	t.Parallel()

	items := []domain.Finding{
		{
			Path:     "/tmp/advisory",
			Category: domain.CategoryTempFiles,
			Action:   domain.ActionAdvisory,
			Bytes:    10,
		},
		{
			Path:     "/tmp/logs",
			Category: domain.CategoryLogs,
			Action:   domain.ActionTrash,
			Bytes:    10,
		},
		{
			Path:     "/tmp/temp",
			Category: domain.CategoryTempFiles,
			Action:   domain.ActionTrash,
			Bytes:    20,
		},
	}

	sortPlanItems("clean", items)

	if items[0].Path != "/tmp/temp" {
		t.Fatalf("expected temp actionable item first, got %+v", items)
	}
	if items[1].Path != "/tmp/advisory" {
		t.Fatalf("expected advisory item to trail actionable peers within same category, got %+v", items)
	}
	if items[2].Path != "/tmp/logs" {
		t.Fatalf("expected later category to remain after temp-file items, got %+v", items)
	}
}

func TestSanitizeTargetsRejectsTraversalAndControlChars(t *testing.T) {
	t.Parallel()

	service := &Service{}
	targets, warnings := service.sanitizeTargets([]string{
		"/tmp/cache",
		"../bad",
		"/tmp/\x00bad",
	})

	if len(targets) != 1 || targets[0] != "/tmp/cache" {
		t.Fatalf("expected only valid target to remain, got %+v", targets)
	}
	if len(warnings) != 2 {
		t.Fatalf("expected two warnings, got %+v", warnings)
	}
}
