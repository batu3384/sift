package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/batu3384/sift/internal/domain"
)

func TestAnalyzeFlowViewShowsInspectSignalAndLaneDeck(t *testing.T) {
	t.Parallel()

	base := analyzeBrowserModel{
		width:  132,
		height: 34,
		plan: domain.ExecutionPlan{
			Command: "analyze",
			Items: []domain.Finding{
				{
					ID:          "chrome-js",
					Name:        "Chrome Code Cache/js",
					Path:        "/tmp/chrome",
					DisplayPath: "/tmp/chrome",
					Category:    domain.CategoryDiskUsage,
					Status:      domain.StatusAdvisory,
					Bytes:       84 << 20,
				},
				{
					ID:          "xcode-cache",
					Name:        "Xcode derived data",
					Path:        "/tmp/xcode",
					DisplayPath: "/tmp/xcode",
					Category:    domain.CategoryDiskUsage,
					Status:      domain.StatusAdvisory,
					Bytes:       32 << 20,
				},
			},
		},
	}
	flow := newAnalyzeFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = analyzeFlowInspecting

	view := flow.View(base)
	for _, snippet := range []string{"oracle trace rail", "Trace lanes", "Inspect deck", "Inspect signal", "FOCUS LANE", "Chrome Code Cache/js", "ORACLE RAIL", "trace focus", "SCANNING 01/05"} {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected %q in analyze flow view, got %s", snippet, view)
		}
	}
}

func TestAnalyzeFlowViewShowsReviewAndResultStates(t *testing.T) {
	t.Parallel()

	base := analyzeBrowserModel{
		width:  132,
		height: 34,
		plan: domain.ExecutionPlan{
			Command: "analyze",
			Items: []domain.Finding{{
				ID:          "chrome-js",
				Name:        "Chrome Code Cache/js",
				Path:        "/tmp/chrome",
				DisplayPath: "/tmp/chrome",
				Category:    domain.CategoryDiskUsage,
				Status:      domain.StatusAdvisory,
				Bytes:       84 << 20,
			}},
		},
	}
	base.applyReviewPreview("/tmp/chrome", domain.ExecutionPlan{
		Command: "clean",
		Totals:  domain.Totals{Bytes: 84 << 20, ItemCount: 1},
		Items: []domain.Finding{{
			Path:        "/tmp/chrome",
			DisplayPath: "/tmp/chrome",
			Status:      domain.StatusPlanned,
			Action:      domain.ActionTrash,
			Category:    domain.CategoryBrowserData,
			Bytes:       84 << 20,
		}},
	}, nil)

	flow := newAnalyzeFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = analyzeFlowReviewReady
	review := flow.View(base)
	for _, snippet := range []string{"review frozen", "review gate on", "REVIEW READY"} {
		if !strings.Contains(review, snippet) {
			t.Fatalf("expected %q in analyze review-ready view, got %s", snippet, review)
		}
	}

	flow.phase = analyzeFlowResult
	flow.hasResult = true
	flow.result = domain.ExecutionResult{
		Items: []domain.OperationResult{
			{Path: "/tmp/chrome", Status: domain.StatusDeleted, Bytes: 84 << 20},
			{Path: "/tmp/protected", Status: domain.StatusProtected, Bytes: 12 << 20},
		},
	}
	settled := flow.View(base)
	if !strings.Contains(settled, "Outcome  1 cleared  •  1 guarded") {
		t.Fatalf("expected analyze result outcome summary, got %s", settled)
	}
}

func TestAnalyzeFlowViewUsesSpinnerFrameAndPathCarryWhileInspecting(t *testing.T) {
	t.Parallel()

	base := analyzeBrowserModel{
		width:  132,
		height: 34,
		plan: domain.ExecutionPlan{
			Command: "analyze",
			Items: []domain.Finding{{
				ID:          "chrome-js",
				Name:        "Chrome Code Cache/js",
				Path:        "/tmp/chrome",
				DisplayPath: "/tmp/chrome",
				Category:    domain.CategoryDiskUsage,
				Status:      domain.StatusAdvisory,
				Bytes:       84 << 20,
			}},
		},
	}
	flow := newAnalyzeFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = analyzeFlowInspecting
	flow.spinnerFrame = 2

	view := flow.View(base)
	normalizedView := strings.Join(strings.Fields(ansi.Strip(view)), " ")
	for _, snippet := range []string{spinnerFrames[2], "ACTIVE TRACE", "path /tmp/chrome", spinnerFrames[2] + " ORACLE HOT PATH"} {
		if !strings.Contains(normalizedView, snippet) {
			t.Fatalf("expected %q in analyze inspect view, got %s", snippet, view)
		}
	}
}

func TestAnalyzeFlowViewShowsQueueLaneTelemetryForStagedBatch(t *testing.T) {
	t.Parallel()

	base := analyzeBrowserModel{
		width:  132,
		height: 34,
		plan: domain.ExecutionPlan{
			Command: "analyze",
			Items: []domain.Finding{
				{
					ID:          "chrome-js",
					Name:        "Chrome Code Cache/js",
					Path:        "/tmp/chrome",
					DisplayPath: "/tmp/chrome",
					Category:    domain.CategoryDiskUsage,
					Status:      domain.StatusAdvisory,
					Bytes:       84 << 20,
				},
				{
					ID:          "xcode-cache",
					Name:        "Xcode derived data",
					Path:        "/tmp/xcode",
					DisplayPath: "/tmp/xcode",
					Category:    domain.CategoryDiskUsage,
					Status:      domain.StatusAdvisory,
					Bytes:       32 << 20,
				},
			},
			Totals: domain.Totals{Bytes: 116 << 20, ItemCount: 2},
		},
	}
	base.toggleStage(base.plan.Items[0])
	base.toggleStage(base.plan.Items[1])

	flow := newAnalyzeFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = analyzeFlowReviewReady

	view := flow.View(base)
	normalizedView := strings.Join(strings.Fields(ansi.Strip(view)), " ")
	for _, snippet := range []string{
		"ORACLE HOT PATH",
		"QUEUE LANE 2 staged",
		"NEXT PASS",
		"Batch lane",
		"Status waiting at review gate",
		"Next access check if needed",
		"trace reclaim pinned",
		"Review is frozen on this trace.",
	} {
		if !strings.Contains(normalizedView, snippet) {
			t.Fatalf("expected %q in analyze batch view, got %s", snippet, view)
		}
	}
}

func TestAnalyzeFlowReclaimingKeepsTraceContinuity(t *testing.T) {
	t.Parallel()

	analyzePlan := domain.ExecutionPlan{
		Command: "analyze",
		Items: []domain.Finding{{
			ID:          "chrome-js",
			Name:        "Chrome Code Cache/js",
			Path:        "/tmp/chrome",
			DisplayPath: "/tmp/chrome",
			Category:    domain.CategoryDiskUsage,
			Status:      domain.StatusAdvisory,
			Bytes:       84 << 20,
		}},
	}
	reviewPlan := domain.ExecutionPlan{
		Command: "clean",
		DryRun:  false,
		Totals:  domain.Totals{Bytes: 84 << 20, SafeBytes: 84 << 20, ItemCount: 1},
		Items: []domain.Finding{{
			ID:          "chrome-js",
			Name:        "Chrome Code Cache/js",
			Path:        "/tmp/chrome",
			DisplayPath: "/tmp/chrome",
			Category:    domain.CategoryBrowserData,
			Status:      domain.StatusPlanned,
			Action:      domain.ActionTrash,
			Bytes:       84 << 20,
		}},
	}
	base := analyzeBrowserModel{
		width:  132,
		height: 34,
		plan:   analyzePlan,
	}
	base.toggleStage(analyzePlan.Items[0])

	flow := newAnalyzeFlowModel()
	flow.width = 132
	flow.height = 34
	flow.markReviewReady(reviewPlan)
	flow.markReclaiming(reviewPlan, permissionPreflightModel{})

	queued := flow.View(base)
	for _, snippet := range []string{"Chrome Code Cache/js", "QUEUED HOLD", "path /tmp/chrome"} {
		if !strings.Contains(queued, snippet) {
			t.Fatalf("expected %q in queued analyze reclaim view, got %s", snippet, queued)
		}
	}

	flow.applyExecutionProgress(domain.ExecutionProgress{
		Phase: domain.ProgressPhaseRunning,
		Item:  reviewPlan.Items[0],
	})
	running := flow.View(base)
	if !strings.Contains(running, "RECLAIM LIVE") {
		t.Fatalf("expected reclaim-live state in analyze view, got %s", running)
	}

	flow.applyExecutionProgress(domain.ExecutionProgress{
		Phase: domain.ProgressPhaseFinished,
		Item:  reviewPlan.Items[0],
		Result: domain.OperationResult{
			FindingID: "chrome-js",
			Path:      "/tmp/chrome",
			Status:    domain.StatusDeleted,
			Bytes:     84 << 20,
		},
	})
	settled := flow.View(base)
	if !strings.Contains(settled, "SETTLED HOLD") {
		t.Fatalf("expected settled-hold state in analyze view, got %s", settled)
	}
}

func TestAnalyzeFlowDeckFollowsActiveTraceRowDuringReclaim(t *testing.T) {
	t.Parallel()

	analyzePlan := domain.ExecutionPlan{
		Command: "analyze",
		Items: []domain.Finding{
			{
				ID:          "chrome-js",
				Name:        "Chrome Code Cache/js",
				Path:        "/tmp/chrome",
				DisplayPath: "/tmp/chrome",
				Category:    domain.CategoryDiskUsage,
				Status:      domain.StatusAdvisory,
				Bytes:       84 << 20,
			},
			{
				ID:          "derived-data",
				Name:        "DerivedData",
				Path:        "/tmp/derived",
				DisplayPath: "/tmp/derived",
				Category:    domain.CategoryDiskUsage,
				Status:      domain.StatusAdvisory,
				Bytes:       32 << 20,
			},
		},
	}
	reviewPlan := domain.ExecutionPlan{
		Command: "clean",
		DryRun:  false,
		Totals:  domain.Totals{Bytes: 116 << 20, SafeBytes: 116 << 20, ItemCount: 2},
		Items: []domain.Finding{
			{
				ID:          "chrome-js",
				Name:        "Chrome Code Cache/js",
				Path:        "/tmp/chrome",
				DisplayPath: "/tmp/chrome",
				Category:    domain.CategoryBrowserData,
				Status:      domain.StatusPlanned,
				Action:      domain.ActionTrash,
				Bytes:       84 << 20,
			},
			{
				ID:          "derived-data",
				Name:        "DerivedData",
				Path:        "/tmp/derived",
				DisplayPath: "/tmp/derived",
				Category:    domain.CategoryDeveloperCaches,
				Status:      domain.StatusPlanned,
				Action:      domain.ActionTrash,
				Bytes:       32 << 20,
			},
		},
	}
	base := analyzeBrowserModel{
		width:  132,
		height: 34,
		plan:   analyzePlan,
	}
	base.toggleStage(analyzePlan.Items[0])
	base.toggleStage(analyzePlan.Items[1])

	flow := newAnalyzeFlowModel()
	flow.width = 132
	flow.height = 34
	flow.markReviewReady(reviewPlan)
	flow.markReclaiming(reviewPlan, permissionPreflightModel{})
	flow.applyExecutionProgress(domain.ExecutionProgress{
		Phase: domain.ProgressPhaseRunning,
		Item:  reviewPlan.Items[1],
	})

	view := flow.View(base)
	for _, snippet := range []string{"ORACLE HOT PATH", "RECLAIM LIVE", "DerivedData", "path /tmp/derived"} {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected %q in analyze deck for active reclaim row, got %s", snippet, view)
		}
	}
}

func TestAnalyzeFlowViewShowsGateWatchAndArchiveTraceChrome(t *testing.T) {
	t.Parallel()

	base := analyzeBrowserModel{
		width:  132,
		height: 34,
		plan: domain.ExecutionPlan{
			Command: "analyze",
			Items: []domain.Finding{
				{
					ID:          "chrome-js",
					Name:        "Chrome Code Cache/js",
					Path:        "/tmp/chrome",
					DisplayPath: "/tmp/chrome",
					Category:    domain.CategoryDiskUsage,
					Status:      domain.StatusAdvisory,
					Bytes:       84 << 20,
				},
			},
		},
	}

	flow := newAnalyzeFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = analyzeFlowResult
	flow.traceRows = []analyzeFlowTraceRow{
		{FindingID: "chrome-js", Path: "/tmp/chrome", Label: "Chrome Code Cache/js", Category: domain.CategoryDiskUsage, Bytes: 84 << 20, State: "review"},
		{FindingID: "guarded-cache", Path: "/tmp/guarded", Label: "Guarded Cache", Category: domain.CategoryDiskUsage, Bytes: 12 << 20, State: "protected"},
		{FindingID: "settled-cache", Path: "/tmp/settled", Label: "Settled Cache", Category: domain.CategoryDiskUsage, Bytes: 4 << 20, State: "settled"},
	}

	view := flow.View(base)
	for _, snippet := range []string{"GATE  REVIEW READY", "Guarded Cache", "WATCH", "GUARDED HOLD", "Settled Cache", "ARCHIVE", "SETTLED HOLD"} {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected %q in analyze trace view, got %s", snippet, view)
		}
	}
}

func TestAnalyzeFlowViewShowsTraceLaneTelemetry(t *testing.T) {
	t.Parallel()

	base := analyzeBrowserModel{
		width:  132,
		height: 34,
		plan: domain.ExecutionPlan{
			Command: "analyze",
			Items: []domain.Finding{{
				ID:          "chrome-js",
				Name:        "Chrome Code Cache/js",
				Path:        "/tmp/chrome",
				DisplayPath: "/tmp/chrome",
				Category:    domain.CategoryDiskUsage,
				Status:      domain.StatusAdvisory,
				Bytes:       84 << 20,
			}},
		},
	}

	flow := newAnalyzeFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = analyzeFlowResult
	flow.traceRows = []analyzeFlowTraceRow{
		{FindingID: "chrome-js", Path: "/tmp/chrome", Label: "Chrome Code Cache/js", Category: domain.CategoryDiskUsage, Bytes: 84 << 20, State: "review"},
		{FindingID: "guarded-cache", Path: "/tmp/guarded", Label: "Guarded Cache", Category: domain.CategoryDiskUsage, Bytes: 12 << 20, State: "protected"},
		{FindingID: "settled-cache", Path: "/tmp/settled", Label: "Settled Cache", Category: domain.CategoryDiskUsage, Bytes: 4 << 20, State: "settled"},
	}

	view := strings.Join(strings.Fields(ansi.Strip(flow.View(base))), " ")
	for _, snippet := range []string{
		"SETTLED LANE",
		"3 traces • 100 MB • 1 gate • 1 watch • 1 archive",
		"Status run settled",
		"Next inspect result or rerun",
	} {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected %q in analyze lane telemetry summary, got %s", snippet, view)
		}
	}
}

func TestAnalyzeFlowViewSupportsHistoryScrollback(t *testing.T) {
	t.Parallel()

	base := analyzeBrowserModel{width: 132, height: 26, plan: domain.ExecutionPlan{Command: "analyze"}}
	flow := newAnalyzeFlowModel()
	flow.width = 132
	flow.height = 26
	flow.phase = analyzeFlowInspecting
	flow.spinnerFrame = 1
	for i := 0; i < 10; i++ {
		flow.traceRows = append(flow.traceRows, analyzeFlowTraceRow{
			FindingID: fmt.Sprintf("row-%c", 'A'+i),
			Path:      fmt.Sprintf("/tmp/row-%c", 'a'+i),
			Label:     fmt.Sprintf("Row %c", 'A'+i),
			Category:  domain.CategoryDiskUsage,
			Bytes:     int64(i+1) << 20,
			State:     "review",
		})
	}

	view := flow.View(base)
	if strings.Contains(view, "Row A") {
		t.Fatalf("expected earliest analyze row to be below the fold before scrollback, got %s", view)
	}

	flow.scrollLedgerUp(40)
	view = strings.Join(strings.Fields(ansi.Strip(flow.View(base))), " ")
	for _, snippet := range []string{"Row A", "scroll hold", "HISTORY HOLD", "End returns live", "▌ HISTORY HOLD", "ORACLE RAIL", "TRACE RAIL", "▌ " + spinnerFrames[1] + " ORACLE RAIL", "╺━━━ ORACLE ━━━╸", "TRACE 01/05"} {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected %q after scrolling analyze history, got %s", snippet, view)
		}
	}
}

func TestAnalyzeFlowViewShowsReviewFreezeBanner(t *testing.T) {
	t.Parallel()

	base := analyzeBrowserModel{width: 132, height: 26, plan: domain.ExecutionPlan{Command: "analyze"}}
	flow := newAnalyzeFlowModel()
	flow.width = 132
	flow.height = 26
	flow.phase = analyzeFlowReviewReady
	flow.spinnerFrame = 2
	flow.pulse = true
	flow.traceRows = []analyzeFlowTraceRow{{
		FindingID: "chrome-js",
		Path:      "/tmp/chrome",
		Label:     "Chrome Code Cache/js",
		Category:  domain.CategoryDiskUsage,
		Bytes:     84 << 20,
		State:     "review",
	}}

	view := strings.Join(strings.Fields(ansi.Strip(flow.View(base))), " ")
	for _, snippet := range []string{"REVIEW FREEZE", "trace carry locked for gate review", "▌ REVIEW FREEZE", "ORACLE RAIL", "REVIEW RAIL", "▌ ~~~ ORACLE RAIL", "╺━━━ ORACLE ━━━╸", "REVIEW 02/05"} {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected %q in analyze review freeze view, got %s", snippet, view)
		}
	}
}

func TestAnalyzeFlowViewShowsFullPhaseTrack(t *testing.T) {
	t.Parallel()

	base := analyzeBrowserModel{width: 132, height: 34, plan: domain.ExecutionPlan{Command: "analyze"}}
	flow := newAnalyzeFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = analyzeFlowInspecting

	view := strings.Join(strings.Fields(ansi.Strip(flow.View(base))), " ")
	for _, snippet := range []string{
		"TRACE 01/05",
		"REVIEW 02/05",
		"ACCESS 03/05",
		"RECLAIM 04/05",
		"SETTLED 05/05",
	} {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected %q in analyze phase track, got %s", snippet, view)
		}
	}
}

func TestAnalyzeFlowViewShowsLaneSummaryLine(t *testing.T) {
	t.Parallel()

	base := analyzeBrowserModel{width: 132, height: 34, plan: domain.ExecutionPlan{Command: "analyze"}}
	flow := newAnalyzeFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = analyzeFlowInspecting
	flow.traceRows = []analyzeFlowTraceRow{{
		FindingID: "chrome-js",
		Path:      "/tmp/chrome",
		Label:     "Chrome Code Cache/js",
		Category:  domain.CategoryDiskUsage,
		Bytes:     84 << 20,
		State:     "review",
	}}

	view := strings.Join(strings.Fields(ansi.Strip(flow.View(base))), " ")
	if !strings.Contains(view, "1 trace • 84 MB • 1 gate") {
		t.Fatalf("expected analyze lane summary line, got %s", view)
	}
}

func TestAnalyzeFlowViewShowsStatusAndNextLines(t *testing.T) {
	t.Parallel()

	base := analyzeBrowserModel{width: 132, height: 34, plan: domain.ExecutionPlan{Command: "analyze"}}
	flow := newAnalyzeFlowModel()
	flow.width = 132
	flow.height = 34
	flow.phase = analyzeFlowInspecting

	view := strings.Join(strings.Fields(ansi.Strip(flow.View(base))), " ")
	for _, snippet := range []string{
		"Status live trace running",
		"Next review gate",
	} {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected %q in analyze deck, got %s", snippet, view)
		}
	}
}
