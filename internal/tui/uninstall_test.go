package tui

import (
	"strings"
	"testing"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func TestUninstallSelectionLineIncludesIndexModeAndQueueState(t *testing.T) {
	t.Parallel()

	model := newUninstallModel()
	model.setItems([]uninstallItem{
		{Name: "Alpha", HasNative: true},
		{Name: "Beta", HasNative: false},
	})
	model.cursor = 1
	item, ok := model.selected()
	if !ok {
		t.Fatal("expected selected uninstall item")
	}
	model.toggleStage(item)

	line := uninstallSelectionLine(model, item, ok)
	for _, snippet := range []string{"State   2/2", "Beta", "remnants", "queued"} {
		if !strings.Contains(line, snippet) {
			t.Fatalf("expected %q in selection line, got %q", snippet, line)
		}
	}
}

func TestUninstallNextLineReflectsSearchAndBatchActions(t *testing.T) {
	t.Parallel()

	model := newUninstallModel()
	model.setItems([]uninstallItem{
		{Name: "Alpha", HasNative: true},
		{Name: "Beta", HasNative: false},
	})
	model.cursor = 1
	item, ok := model.selected()
	if !ok {
		t.Fatal("expected selected uninstall item")
	}

	model.searchActive = true
	searchLine := uninstallNextLine(model, item, ok)
	for _, snippet := range []string{"type to filter", "enter apply", "esc clear"} {
		if !strings.Contains(searchLine, snippet) {
			t.Fatalf("expected %q in search next line, got %q", snippet, searchLine)
		}
	}

	model.searchActive = false
	model.toggleStage(item)
	batchLine := uninstallNextLine(model, item, ok)
	for _, snippet := range []string{"enter open", "u remove", "x batch", "check files"} {
		if !strings.Contains(batchLine, snippet) {
			t.Fatalf("expected %q in batch next line, got %q", snippet, batchLine)
		}
	}
}

func TestUninstallPreviewLinesShowLoadedPlanSummary(t *testing.T) {
	t.Parallel()

	model := newUninstallModel()
	model.setItems([]uninstallItem{
		{Name: "Alpha", HasNative: true},
	})
	item, ok := model.selected()
	if !ok {
		t.Fatal("expected selected uninstall item")
	}
	model.applyPreview(uninstallStageKey("Alpha"), domain.ExecutionPlan{
		Command: "uninstall",
		Totals: domain.Totals{
			Bytes: 2 * 1024 * 1024,
		},
		Items: []domain.Finding{
			{Name: "Cache", Path: "/tmp/a", Status: domain.StatusPlanned, Action: domain.ActionTrash, Category: domain.CategoryAppLeftovers, Bytes: 2 * 1024 * 1024},
		},
	}, nil)

	lines := uninstallPreviewLines(model, item, 64)
	joined := strings.Join(lines, "\n")
	for _, snippet := range []string{"Preview", "1 ready", "1 module", "2.0 MB"} {
		if !strings.Contains(joined, snippet) {
			t.Fatalf("expected %q in uninstall preview lines, got %q", snippet, joined)
		}
	}
}
