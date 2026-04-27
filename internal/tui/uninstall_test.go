package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/batu3384/sift/internal/domain"
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

func TestUninstallFlowViewShowsRemovalSignalAndLaneTelemetry(t *testing.T) {
	t.Parallel()

	base := newUninstallModel()
	base.width = 132
	base.height = 34
	base.setItems([]uninstallItem{
		{Name: "Example App", HasNative: true, SizeLabel: "84 MB"},
		{Name: "Builder Tool", HasNative: false, SizeLabel: "32 MB"},
		{Name: "Secure Suite", HasNative: true, RequiresAdmin: true, SizeLabel: "128 MB"},
	})

	flow := newUninstallFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = uninstallFlowInventory

	view := flow.View(base)
	for _, snippet := range []string{"courier handoff rail", "Removal signal", "TARGET RAIL", "TARGET 01/05", "NATIVE LANE", "REMNANT LANE", "WATCH LANE", "COURIER RAIL", "handoff continuity"} {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected %q in uninstall flow view, got %s", snippet, view)
		}
	}
}

func TestUninstallFlowStatsIncludeStepCounter(t *testing.T) {
	t.Parallel()

	base := newUninstallModel()
	base.setItems([]uninstallItem{
		{Name: "Example App", HasNative: true},
		{Name: "Builder Tool", HasNative: false},
	})
	flow := newUninstallFlowModel()
	flow.phase = uninstallFlowInventory

	cards := uninstallFlowStats(flow, base, 132)
	joined := strings.Join(cards, " ")
	for _, snippet := range []string{"TARGETS", "2 apps", "01/05"} {
		if !strings.Contains(joined, snippet) {
			t.Fatalf("expected %q in uninstall stat cards, got %s", snippet, joined)
		}
	}
}

func TestUninstallFlowViewShowsHandoffRemnantAndAftercareStates(t *testing.T) {
	t.Parallel()

	base := newUninstallModel()
	base.width = 132
	base.height = 34
	base.setItems([]uninstallItem{
		{Name: "Example App", HasNative: true, SizeLabel: "84 MB"},
		{Name: "Builder Tool", HasNative: false, SizeLabel: "32 MB"},
	})
	base.toggleStage(uninstallItem{Name: "Builder Tool", HasNative: false, SizeLabel: "32 MB"})

	flow := newUninstallFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = uninstallFlowRemoving

	removing := flow.View(base)
	normalizedRemoving := strings.Join(strings.Fields(ansi.Strip(removing)), " ")
	for _, snippet := range []string{"COURIER HOT PATH", "NATIVE HANDOFF", "NEXT PASS", "REMNANT PASS", "Live handoff is moving through native and remnant passes.", "native handoff pinned", "queued remnants behind this target"} {
		if !strings.Contains(normalizedRemoving, snippet) {
			t.Fatalf("expected %q in uninstall removing view, got %s", snippet, removing)
		}
	}

	flow.phase = uninstallFlowResult
	flow.hasResult = true
	flow.result = domain.ExecutionResult{
		Items: []domain.OperationResult{{
			Path:   "/tmp/example-cache",
			Status: domain.StatusDeleted,
			Bytes:  32 << 20,
		}},
	}
	settled := flow.View(base)
	if !strings.Contains(settled, "AFTERCARE HOLD") {
		t.Fatalf("expected AFTERCARE HOLD in uninstall result view, got %s", settled)
	}
}

func TestUninstallFlowViewUsesSpinnerFrameAndPathCarryDuringInventory(t *testing.T) {
	t.Parallel()

	base := newUninstallModel()
	base.width = 132
	base.height = 34
	base.setItems([]uninstallItem{{
		Name:       "Example App",
		HasNative:  true,
		Location:   "/Applications/Example App.app",
		SizeLabel:  "84 MB",
		ApproxBytes: 84 << 20,
	}})

	flow := newUninstallFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = uninstallFlowInventory
	flow.spinnerFrame = 3

	view := flow.View(base)
	normalizedView := strings.Join(strings.Fields(ansi.Strip(view)), " ")
	for _, snippet := range []string{spinnerFrames[3], "path /Applications/Example App.app", spinnerFrames[3] + " COURIER HOT PATH"} {
		if !strings.Contains(normalizedView, snippet) {
			t.Fatalf("expected %q in uninstall inventory view, got %s", snippet, view)
		}
	}
}

func TestUninstallFlowViewShowsOutcomeSummaryInResultPhase(t *testing.T) {
	t.Parallel()

	base := newUninstallModel()
	base.width = 132
	base.height = 34
	base.setItems([]uninstallItem{{
		Name:       "Example App",
		HasNative:  true,
		Location:   "/Applications/Example App.app",
		SizeLabel:  "84 MB",
		ApproxBytes: 84 << 20,
	}})

	flow := newUninstallFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = uninstallFlowResult
	flow.hasResult = true
	flow.result = domain.ExecutionResult{
		Items: []domain.OperationResult{
			{Path: "/tmp/cache", Status: domain.StatusDeleted, Bytes: 32 << 20},
			{Path: "/tmp/protected", Status: domain.StatusProtected, Bytes: 12 << 20},
			{Path: "/tmp/failed", Status: domain.StatusFailed, Bytes: 4 << 20},
		},
	}

	view := flow.View(base)
	if !strings.Contains(view, "Outcome  1 removed  •  1 guarded  •  1 failed") {
		t.Fatalf("expected uninstall outcome summary in result view, got %s", view)
	}
}

func TestUninstallFlowViewShowsLiveGateWatchAndArchiveRowChrome(t *testing.T) {
	t.Parallel()

	base := newUninstallModel()
	base.width = 132
	base.height = 34
	base.setItems([]uninstallItem{
		{Name: "Example App", HasNative: true, SizeLabel: "84 MB"},
		{Name: "Builder Tool", HasNative: false, SizeLabel: "32 MB"},
		{Name: "Secure Suite", HasNative: true, RequiresAdmin: true, SizeLabel: "128 MB"},
	})
	base.toggleStage(uninstallItem{Name: "Builder Tool", HasNative: false, SizeLabel: "32 MB"})

	flow := newUninstallFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = uninstallFlowInventory

	inventory := flow.View(base)
	for _, snippet := range []string{"Example App NATIVE LIVE", "Secure Suite NATIVE WATCH"} {
		if !strings.Contains(inventory, snippet) {
			t.Fatalf("expected %q in uninstall inventory view, got %s", snippet, inventory)
		}
	}

	flow.phase = uninstallFlowReviewReady
	review := flow.View(base)
	if !strings.Contains(review, "Builder Tool REMNANTS QUEUED GATE") {
		t.Fatalf("expected gated staged row in uninstall review view, got %s", review)
	}

	flow.phase = uninstallFlowResult
	settled := flow.View(base)
	if !strings.Contains(settled, "Example App NATIVE ARCHIVE") {
		t.Fatalf("expected archive row chrome in uninstall result view, got %s", settled)
	}
}

func TestUninstallFlowViewShowsLaneStateTelemetry(t *testing.T) {
	t.Parallel()

	base := newUninstallModel()
	base.width = 132
	base.height = 34
	base.setItems([]uninstallItem{
		{Name: "Example App", HasNative: true, SizeLabel: "84 MB"},
		{Name: "Builder Tool", HasNative: false, SizeLabel: "32 MB"},
		{Name: "Secure Suite", HasNative: true, RequiresAdmin: true, SizeLabel: "128 MB"},
	})
	base.toggleStage(uninstallItem{Name: "Builder Tool", HasNative: false, SizeLabel: "32 MB"})

	flow := newUninstallFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = uninstallFlowReviewReady

	view := strings.Join(strings.Fields(ansi.Strip(flow.View(base))), " ")
	for _, snippet := range []string{
		"NATIVE LANE",
		"1 target • 84 MB • 1 gate",
		"REMNANT LANE",
		"1 target • 32 MB • 1 gate",
		"WATCH LANE",
		"1 target • 128 MB • 1 watch",
	} {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected %q in uninstall lane telemetry, got %s", snippet, view)
		}
	}
}

func TestUninstallFlowViewSupportsHistoryScrollback(t *testing.T) {
	t.Parallel()

	base := newUninstallModel()
	base.width = 132
	base.height = 26
	items := make([]uninstallItem, 0, 10)
	for i := 0; i < 10; i++ {
		items = append(items, uninstallItem{
			Name:      fmt.Sprintf("Row %c", 'A'+i),
			HasNative: true,
			SizeLabel: fmt.Sprintf("%d MB", i+1),
		})
	}
	base.setItems(items)
	base.cursor = len(base.filtered) - 1

	flow := newUninstallFlowModel()
	flow.width = 132
	flow.height = 26
	flow.phase = uninstallFlowInventory
	flow.spinnerFrame = 4

	view := flow.View(base)
	if strings.Contains(view, "Row A") {
		t.Fatalf("expected earliest uninstall row to be below the fold before scrollback, got %s", view)
	}

	flow.scrollLedgerUp(base, 40)
	view = strings.Join(strings.Fields(ansi.Strip(flow.View(base))), " ")
	for _, snippet := range []string{"Row A", "scroll hold", "HISTORY HOLD", "End returns live", "▌ HISTORY HOLD", "COURIER RAIL", "TARGET RAIL", "▌ " + spinnerFrames[4] + " COURIER RAIL", "╺━━━ COURIER ━━━╸", "TARGET 01/05"} {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected %q after scrolling uninstall history, got %s", snippet, view)
		}
	}
}

func TestUninstallFlowViewShowsReviewFreezeBanner(t *testing.T) {
	t.Parallel()

	base := newUninstallModel()
	base.width = 132
	base.height = 26
	base.setItems([]uninstallItem{{Name: "Example App", HasNative: true, SizeLabel: "84 MB"}})

	flow := newUninstallFlowModel()
	flow.width = 132
	flow.height = 26
	flow.phase = uninstallFlowReviewReady
	flow.spinnerFrame = 2
	flow.pulse = true

	view := strings.Join(strings.Fields(ansi.Strip(flow.View(base))), " ")
	for _, snippet := range []string{"REVIEW FREEZE", "target carry locked for gate review", "▌ REVIEW FREEZE", "COURIER RAIL", "REVIEW RAIL", "▌ ~~~ COURIER RAIL", "╺━━━ COURIER ━━━╸", "REVIEW 02/05"} {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected %q in uninstall review freeze view, got %s", snippet, view)
		}
	}
}

func TestUninstallFlowViewShowsFullPhaseTrack(t *testing.T) {
	t.Parallel()

	base := newUninstallModel()
	base.width = 132
	base.height = 34
	base.setItems([]uninstallItem{{Name: "Example App", HasNative: true, SizeLabel: "84 MB"}})

	flow := newUninstallFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = uninstallFlowInventory

	view := strings.Join(strings.Fields(ansi.Strip(flow.View(base))), " ")
	for _, snippet := range []string{
		"TARGET 01/05",
		"REVIEW 02/05",
		"ACCESS 03/05",
		"HANDOFF 04/05",
		"AFTERCARE 05/05",
	} {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected %q in uninstall phase track, got %s", snippet, view)
		}
	}
}

func TestUninstallFlowViewShowsLaneSummaryLine(t *testing.T) {
	t.Parallel()

	base := newUninstallModel()
	base.width = 132
	base.height = 34
	base.setItems([]uninstallItem{{Name: "Example App", HasNative: true, SizeLabel: "84 MB"}})

	flow := newUninstallFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = uninstallFlowInventory

	view := strings.Join(strings.Fields(ansi.Strip(flow.View(base))), " ")
	if !strings.Contains(view, "1 target • 84 MB • 1 live") {
		t.Fatalf("expected uninstall lane summary line, got %s", view)
	}
}

func TestUninstallFlowViewShowsStatusAndNextLines(t *testing.T) {
	t.Parallel()

	base := newUninstallModel()
	base.width = 132
	base.height = 34
	base.setItems([]uninstallItem{{Name: "Example App", HasNative: true, SizeLabel: "84 MB"}})

	flow := newUninstallFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = uninstallFlowInventory

	view := strings.Join(strings.Fields(ansi.Strip(flow.View(base))), " ")
	for _, snippet := range []string{
		"Status live inventory running",
		"Next review gate",
	} {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected %q in uninstall deck, got %s", snippet, view)
		}
	}
}
