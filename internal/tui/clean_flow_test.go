package tui

import (
	"strings"
	"testing"

	"github.com/batu3384/sift/internal/domain"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestCleanFlowApplyPreviewTransitionsToReviewReady(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.setPreviewLoading("safe")
	if model.phase != cleanFlowScanning {
		t.Fatalf("expected scanning phase after preview warmup, got %s", model.phase)
	}

	plan := domain.ExecutionPlan{Command: "clean", Profile: "safe", DryRun: true}
	model.applyPreview("safe", plan, nil)

	if model.phase != cleanFlowReviewReady {
		t.Fatalf("expected review-ready phase after preview load, got %s", model.phase)
	}
	if !model.preview.loaded || model.preview.loading {
		t.Fatalf("expected loaded preview state, got %+v", model.preview)
	}
}

func TestCleanFlowViewRendersPlanBackedLedger(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.applyPreview("safe", domain.ExecutionPlan{
		Command: "clean",
		Profile: "safe",
		DryRun:  true,
		Totals:  domain.Totals{SafeBytes: 84 << 20, ReviewBytes: 28 << 20, ItemCount: 2, Bytes: 112 << 20},
		Items: []domain.Finding{
			{
				ID:          "a",
				Name:        "Chrome Code Cache/js",
				DisplayPath: "/tmp/chrome",
				Path:        "/tmp/chrome",
				Category:    domain.CategoryBrowserData,
				Bytes:       84 << 20,
				Risk:        domain.RiskSafe,
				Status:      domain.StatusPlanned,
			},
			{
				ID:          "b",
				Name:        "Discord leftovers",
				DisplayPath: "/tmp/discord",
				Path:        "/tmp/discord",
				Category:    domain.CategoryAppLeftovers,
				Bytes:       28 << 20,
				Risk:        domain.RiskReview,
				Status:      domain.StatusPlanned,
			},
		},
	}, nil)

	view := model.View()
	for _, needle := range []string{
		"Browser lane",
		"Residue lane",
		"Chrome Code Cache/js",
		"Discord leftovers",
		"84 MB",
		"28 MB",
		"frozen review",
		"FROZEN 02/05",
		"FORGE RAIL",
		"reclaim discipline",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean flow view, got %s", needle, view)
		}
	}
}

func TestStartCleanPreviewLoadResetsPreviewForNewProfile(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteClean)
	model.clean.applyPreview("safe", domain.ExecutionPlan{Command: "clean", Profile: "safe", DryRun: true}, nil)
	model.cleanFlow.applyPreview("safe", domain.ExecutionPlan{Command: "clean", Profile: "safe", DryRun: true}, nil)
	model.callbacks.LoadCleanProfile = func(profile string) (domain.ExecutionPlan, error) {
		return domain.ExecutionPlan{Command: "clean", Profile: profile, DryRun: true}, nil
	}

	model.clean.cursor = 1
	model.cleanFlow.cursor = 1
	cmd := model.startCleanPreviewLoad()
	if cmd == nil {
		t.Fatal("expected preview load command for changed clean profile")
	}
	if model.cleanFlow.phase != cleanFlowScanning {
		t.Fatalf("expected clean flow to return to scanning, got %s", model.cleanFlow.phase)
	}
	if !model.cleanFlow.preview.loading || model.cleanFlow.preview.key != "developer" {
		t.Fatalf("expected clean flow preview to warm developer profile, got %+v", model.cleanFlow.preview)
	}
}

func TestCleanFlowCachedPreviewDoesNotLeakAcrossProfiles(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteClean)
	model.clean.applyPreview("safe", domain.ExecutionPlan{Command: "clean", Profile: "safe", DryRun: true}, nil)
	model.cleanFlow.applyPreview("safe", domain.ExecutionPlan{Command: "clean", Profile: "safe", DryRun: true}, nil)
	model.clean.cursor = 1
	model.cleanFlow.cursor = 1
	model.callbacks.LoadCleanProfile = func(profile string) (domain.ExecutionPlan, error) {
		return domain.ExecutionPlan{Command: "clean", Profile: profile, DryRun: true}, nil
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if next.(appModel).route != RouteClean {
		t.Fatalf("expected to remain on clean route while new preview warms, got %s", next.(appModel).route)
	}
	if cmd == nil {
		t.Fatal("expected clean route to warm a new preview instead of opening the wrong cached review")
	}
}

func TestCleanFlowViewRendersAccessCheckPhase(t *testing.T) {
	t.Parallel()

	plan := domain.ExecutionPlan{
		Command:  "clean",
		Profile:  "safe",
		DryRun:   false,
		Platform: "darwin",
		Items: []domain.Finding{{
			ID:            "secure-clean",
			Name:          "Reset cache indexes",
			Path:          "/Library/Caches/system",
			DisplayPath:   "/usr/bin/sudo /usr/bin/true",
			Status:        domain.StatusPlanned,
			Action:        domain.ActionCommand,
			CommandPath:   "/usr/bin/sudo",
			CommandArgs:   []string{"/usr/bin/true"},
			RequiresAdmin: true,
			Category:      domain.CategoryMaintenance,
		}},
	}
	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.markPermissions(buildPermissionPreflight(plan, "/Library/Caches/system"))

	view := model.View()
	for _, needle := range []string{"access check", "access manifest", "Access", "1 admin"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean flow permissions view, got %s", needle, view)
		}
	}
}

func TestCleanFlowViewRendersSettledResultPhase(t *testing.T) {
	t.Parallel()

	plan := domain.ExecutionPlan{
		Command:  "clean",
		Profile:  "safe",
		DryRun:   false,
		Platform: "darwin",
		Totals:   domain.Totals{SafeBytes: 84 << 20, Bytes: 84 << 20, ItemCount: 1},
		Items: []domain.Finding{{
			ID:          "cache-a",
			Name:        "Chrome Code Cache/js",
			Path:        "/tmp/cache-a",
			DisplayPath: "/tmp/cache-a",
			Status:      domain.StatusPlanned,
			Action:      domain.ActionTrash,
			Category:    domain.CategoryBrowserData,
			Bytes:       84 << 20,
		}},
	}
	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.markResult(plan, domain.ExecutionResult{
		Items: []domain.OperationResult{{
			Path:   "/tmp/cache-a",
			Status: domain.StatusDeleted,
			Bytes:  84 << 20,
		}},
	})

	view := strings.Join(strings.Fields(ansi.Strip(model.View())), " ")
	for _, needle := range []string{"settled result", "run settled", "84 MB", "reclaimed", "Outcome"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean flow result view, got %s", needle, view)
		}
	}
}

func TestCleanFlowViewRendersLiveScanProgress(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.cursor = 1
	model.setPreviewLoading("developer")
	model.applyScanProgress("chrome_cache", "Chrome code cache", 12, 84<<20)

	view := model.View()
	for _, needle := range []string{"scan first", "Browser caches", "Chrome code cache", "84 MB", "12 items"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean flow scan view, got %s", needle, view)
		}
	}
}

func TestCleanFlowViewRendersLiveFindingScanProgress(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.cursor = 1
	model.setPreviewLoading("developer")
	model.applyScanFinding("chrome_cache", "Chrome code cache", domain.Finding{
		ID:          "chrome-js",
		Name:        "Chrome Code Cache/js",
		Path:        "/tmp/chrome",
		DisplayPath: "/tmp/chrome",
		Category:    domain.CategoryBrowserData,
		Bytes:       84 << 20,
	})

	view := model.View()
	for _, needle := range []string{"Chrome Code Cache/js", "Browser caches", "84 MB", "scanning"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean flow finding view, got %s", needle, view)
		}
	}
}

func TestCleanFlowViewFreezesRecentScanCarryIntoReviewReady(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.cursor = 1
	model.setPreviewLoading("developer")
	model.applyScanFinding("chrome_cache", "Chrome code cache", domain.Finding{
		ID:          "chrome-js",
		Name:        "Chrome Code Cache/js",
		Path:        "/tmp/chrome",
		DisplayPath: "/tmp/chrome",
		Category:    domain.CategoryBrowserData,
		Bytes:       84 << 20,
	})
	model.applyPreview("developer", domain.ExecutionPlan{
		Command: "clean",
		Profile: "developer",
		DryRun:  true,
		Totals:  domain.Totals{SafeBytes: 84 << 20, Bytes: 84 << 20, ItemCount: 1},
		Items: []domain.Finding{{
			ID:          "chrome-js",
			Name:        "Chrome Code Cache/js",
			Path:        "/tmp/chrome",
			DisplayPath: "/tmp/chrome",
			Category:    domain.CategoryBrowserData,
			Bytes:       84 << 20,
			Status:      domain.StatusPlanned,
		}},
	}, nil)

	view := model.View()
	for _, needle := range []string{"Chrome Code Cache/js", "frozen review", "ready", "REVIEW FREEZE", "scan carry locked for gate review", "▌ REVIEW FREEZE", "╺━━━ FORGE ━━━╸", "REVIEW 02/05"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean flow frozen review, got %s", needle, view)
		}
	}
	if strings.Contains(view, "scanning") {
		t.Fatalf("expected scan carry to freeze out of scanning state, got %s", view)
	}
}

func TestCleanFlowViewMovesActiveLaneToTop(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.cursor = 1
	model.setPreviewLoading("developer")
	model.applyScanFinding("chrome_cache", "Chrome code cache", domain.Finding{
		ID:          "chrome-js",
		Name:        "Chrome Code Cache/js",
		Path:        "/tmp/chrome",
		DisplayPath: "/tmp/chrome",
		Category:    domain.CategoryBrowserData,
		Bytes:       84 << 20,
	})
	model.applyScanFinding("xcode_cache", "Xcode DerivedData", domain.Finding{
		ID:          "derived-data",
		Name:        "DerivedData",
		Path:        "/tmp/derived",
		DisplayPath: "/tmp/derived",
		Category:    domain.CategoryDeveloperCaches,
		Bytes:       32 << 20,
	})

	view := model.View()
	devIdx := strings.Index(view, "Developer caches")
	browserIdx := strings.Index(view, "Browser caches")
	if devIdx < 0 || browserIdx < 0 {
		t.Fatalf("expected both lane labels in clean flow view, got %s", view)
	}
	if devIdx > browserIdx {
		t.Fatalf("expected active developer lane to render before browser lane, got %s", view)
	}
}

func TestCleanFlowViewUsesSpinnerFrameWhileScanning(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.cursor = 1
	model.spinnerFrame = 3
	model.pulse = true
	model.setPreviewLoading("developer")
	model.applyScanFinding("chrome_cache", "Chrome code cache", domain.Finding{
		ID:          "chrome-js",
		Name:        "Chrome Code Cache/js",
		Path:        "/tmp/chrome",
		DisplayPath: "/tmp/chrome",
		Category:    domain.CategoryBrowserData,
		Bytes:       84 << 20,
	})

	view := model.View()
	if !strings.Contains(view, spinnerFrames[3]) {
		t.Fatalf("expected active scanning row to use spinner frame %q, got %s", spinnerFrames[3], view)
	}
}

func TestCleanFlowViewSupportsHistoryScrollback(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 26
	model.cursor = 1
	model.spinnerFrame = 2
	model.pulse = true
	model.setPreviewLoading("developer")
	for i := 0; i < 10; i++ {
		model.applyScanFinding("cache", "Developer cache", domain.Finding{
			ID:          strings.Join([]string{"row", cleanFlowHumanBytes(int64(i + 1))}, "-"),
			Name:        strings.Join([]string{"Row", string(rune('A' + i))}, " "),
			Path:        strings.Join([]string{"/tmp/row", string(rune('a' + i))}, "-"),
			DisplayPath: strings.Join([]string{"/tmp/row", string(rune('a' + i))}, "-"),
			Category:    domain.CategoryDeveloperCaches,
			Bytes:       int64(i+1) << 20,
		})
	}

	view := strings.Join(strings.Fields(ansi.Strip(model.View())), " ")
	if strings.Contains(view, "Row A") {
		t.Fatalf("expected earliest row to be below the fold before scrollback, got %s", view)
	}

	model.scrollLedgerUp(40)
	view = strings.Join(strings.Fields(ansi.Strip(model.View())), " ")
	for _, needle := range []string{"Row A", "scroll hold", "HISTORY HOLD", "End returns live", "▌ HISTORY HOLD", "FORGE RAIL", "SCAN RAIL", "▌ >>> FORGE RAIL", "╺━━━ FORGE ━━━╸", "SCAN 01/05"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q after scrolling clean history, got %s", needle, view)
		}
	}

	model.scrollLedgerToLatest()
	view = ansi.Strip(model.View())
	if strings.Contains(view, "Row A") {
		t.Fatalf("expected latest-follow view after reset, got %s", view)
	}
}

func TestCleanFlowViewShowsLaneItemCounts(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.cursor = 1
	model.setPreviewLoading("developer")
	model.applyScanFinding("chrome_cache", "Chrome code cache", domain.Finding{
		ID:          "chrome-js",
		Name:        "Chrome Code Cache/js",
		Path:        "/tmp/chrome",
		DisplayPath: "/tmp/chrome",
		Category:    domain.CategoryBrowserData,
		Bytes:       84 << 20,
	})

	view := strings.Join(strings.Fields(ansi.Strip(model.View())), " ")
	if !strings.Contains(view, "1 item") {
		t.Fatalf("expected lane header to show item count, got %s", view)
	}
}

func TestCleanFlowViewShowsLaneLiveAndSettledCounts(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.cursor = 1
	model.setPreviewLoading("developer")
	model.applyScanFinding("chrome_cache", "Chrome code cache", domain.Finding{
		ID:          "chrome-js",
		Name:        "Chrome Code Cache/js",
		Path:        "/tmp/chrome",
		DisplayPath: "/tmp/chrome",
		Category:    domain.CategoryBrowserData,
		Bytes:       84 << 20,
	})
	model.applyScanFinding("chrome_cache", "Chrome code cache", domain.Finding{
		ID:          "chrome-gpu",
		Name:        "Chrome GPU Cache",
		Path:        "/tmp/chrome-gpu",
		DisplayPath: "/tmp/chrome-gpu",
		Category:    domain.CategoryBrowserData,
		Bytes:       12 << 20,
	})

	view := strings.Join(strings.Fields(ansi.Strip(model.View())), " ")
	for _, needle := range []string{"2 items", "1 live", "1 settled"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected lane telemetry %q, got %s", needle, view)
		}
	}
}

func TestCleanFlowReclaimingKeepsLedgerContinuity(t *testing.T) {
	t.Parallel()

	item := domain.Finding{
		ID:          "chrome-js",
		Name:        "Chrome Code Cache/js",
		Path:        "/tmp/chrome",
		DisplayPath: "/tmp/chrome",
		Category:    domain.CategoryBrowserData,
		Bytes:       84 << 20,
		Status:      domain.StatusPlanned,
		Action:      domain.ActionTrash,
	}
	plan := domain.ExecutionPlan{
		Command: "clean",
		Profile: "developer",
		DryRun:  false,
		Totals:  domain.Totals{SafeBytes: 84 << 20, Bytes: 84 << 20, ItemCount: 1},
		Items:   []domain.Finding{item},
	}

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.cursor = 1
	model.setPreviewLoading("developer")
	model.applyScanFinding("chrome_cache", "Chrome code cache", item)
	model.applyPreview("developer", plan, nil)
	model.markReclaiming(plan, permissionPreflightModel{})

	queuedView := model.View()
	for _, needle := range []string{"Chrome Code Cache/js", "queued", "reclaiming now"} {
		if !strings.Contains(queuedView, needle) {
			t.Fatalf("expected %q in queued reclaim view, got %s", needle, queuedView)
		}
	}

	model.applyExecutionProgress(domain.ExecutionProgress{
		Phase: domain.ProgressPhaseRunning,
		Item:  item,
	})
	runningView := model.View()
	if !strings.Contains(runningView, "reclaiming") {
		t.Fatalf("expected reclaiming state in running clean flow, got %s", runningView)
	}

	model.applyExecutionProgress(domain.ExecutionProgress{
		Phase: domain.ProgressPhaseFinished,
		Item:  item,
		Result: domain.OperationResult{
			FindingID: item.ID,
			Path:      item.Path,
			Status:    domain.StatusDeleted,
			Bytes:     item.Bytes,
		},
	})
	settledView := model.View()
	if !strings.Contains(settledView, "settled") {
		t.Fatalf("expected settled state in finished clean flow, got %s", settledView)
	}
}

func TestCleanFlowViewShowsCurrentSweepAndNextReclaimBlocks(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.cursor = 1
	model.setPreviewLoading("developer")
	model.applyScanFinding("chrome_cache", "Chrome code cache", domain.Finding{
		ID:          "chrome-js",
		Name:        "Chrome Code Cache/js",
		Path:        "/tmp/chrome",
		DisplayPath: "/tmp/chrome",
		Category:    domain.CategoryBrowserData,
		Bytes:       84 << 20,
	})
	model.applyScanFinding("xcode_cache", "Xcode DerivedData", domain.Finding{
		ID:          "derived-data",
		Name:        "DerivedData",
		Path:        "/tmp/derived",
		DisplayPath: "/tmp/derived",
		Category:    domain.CategoryDeveloperCaches,
		Bytes:       32 << 20,
	})

	view := model.View()
	for _, needle := range []string{
		"Current sweep",
		"HOT PATH",
		"ACTIVE SCAN",
		"DerivedData",
		"Developer caches",
		"Next reclaim",
		"NEXT PASS",
		"SETTLED HOLD",
		"Chrome Code Cache/js",
		"Browser caches",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean flow focus panel, got %s", needle, view)
		}
	}
}

func TestCleanFlowViewShowsSweepSignalAndLaneTelemetry(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.cursor = 1
	model.spinnerFrame = 2
	model.pulse = true
	model.setPreviewLoading("developer")
	model.applyScanFinding("chrome_cache", "Chrome code cache", domain.Finding{
		ID:          "chrome-js",
		Name:        "Chrome Code Cache/js",
		Path:        "/tmp/chrome",
		DisplayPath: "/tmp/chrome",
		Category:    domain.CategoryBrowserData,
		Bytes:       84 << 20,
	})
	model.applyScanFinding("xcode_cache", "Xcode DerivedData", domain.Finding{
		ID:          "derived-data",
		Name:        "DerivedData",
		Path:        "/tmp/derived",
		DisplayPath: "/tmp/derived",
		Category:    domain.CategoryDeveloperCaches,
		Bytes:       32 << 20,
	})

	view := ansi.Strip(model.View())
	for _, needle := range []string{
		"Sweep lanes",
		"Sweep deck",
		"Sweep signal",
		"SCAN RAIL",
		"╭───╮",
		"SCAN LANE",
		"SETTLED LANE",
		"Developer caches",
		"Browser caches",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean flow signal/telemetry view, got %s", needle, view)
		}
	}
}

func TestCleanFlowViewShowsFullPhaseTrack(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.cursor = 1
	model.setPreviewLoading("developer")
	model.applyScanFinding("chrome_cache", "Chrome code cache", domain.Finding{
		ID:          "chrome-js",
		Name:        "Chrome Code Cache/js",
		Path:        "/tmp/chrome",
		DisplayPath: "/tmp/chrome",
		Category:    domain.CategoryBrowserData,
		Bytes:       84 << 20,
	})

	view := strings.Join(strings.Fields(ansi.Strip(model.View())), " ")
	for _, needle := range []string{
		"SCAN 01/05",
		"REVIEW 02/05",
		"ACCESS 03/05",
		"RECLAIM 04/05",
		"SETTLED 05/05",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean phase track, got %s", needle, view)
		}
	}
}

func TestCleanFlowViewShowsLaneSummaryLine(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.cursor = 1
	model.setPreviewLoading("developer")
	model.applyScanFinding("chrome_cache", "Chrome code cache", domain.Finding{
		ID:          "chrome-js",
		Name:        "Chrome Code Cache/js",
		Path:        "/tmp/chrome",
		DisplayPath: "/tmp/chrome",
		Category:    domain.CategoryBrowserData,
		Bytes:       84 << 20,
	})

	view := strings.Join(strings.Fields(ansi.Strip(model.View())), " ")
	if !strings.Contains(view, "1 item • 84 MB • 1 live") {
		t.Fatalf("expected clean lane summary line, got %s", view)
	}
}

func TestCleanFlowViewShowsStatusAndNextLinesWhileScanning(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.cursor = 1
	model.setPreviewLoading("developer")
	model.applyScanFinding("chrome_cache", "Chrome code cache", domain.Finding{
		ID:          "chrome-js",
		Name:        "Chrome Code Cache/js",
		Path:        "/tmp/chrome",
		DisplayPath: "/tmp/chrome",
		Category:    domain.CategoryBrowserData,
		Bytes:       84 << 20,
	})

	view := strings.Join(strings.Fields(ansi.Strip(model.View())), " ")
	for _, needle := range []string{
		"Status live sweep running",
		"Next review gate",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean status deck, got %s", needle, view)
		}
	}
}

func TestCleanFlowViewShowsSettledStatusAndNextLines(t *testing.T) {
	t.Parallel()

	plan := domain.ExecutionPlan{
		Command:  "clean",
		Profile:  "safe",
		DryRun:   false,
		Platform: "darwin",
		Totals:   domain.Totals{SafeBytes: 84 << 20, Bytes: 84 << 20, ItemCount: 1},
		Items: []domain.Finding{{
			ID:          "cache-a",
			Name:        "Chrome Code Cache/js",
			Path:        "/tmp/cache-a",
			DisplayPath: "/tmp/cache-a",
			Status:      domain.StatusPlanned,
			Action:      domain.ActionTrash,
			Category:    domain.CategoryBrowserData,
			Bytes:       84 << 20,
		}},
	}
	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.markResult(plan, domain.ExecutionResult{
		Items: []domain.OperationResult{{
			Path:   "/tmp/cache-a",
			Status: domain.StatusDeleted,
			Bytes:  84 << 20,
		}},
	})

	view := strings.Join(strings.Fields(ansi.Strip(model.View())), " ")
	for _, needle := range []string{
		"Status run settled",
		"Next inspect watch or rerun",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean settled deck, got %s", needle, view)
		}
	}
}

func TestCleanFlowViewShowsReviewLaneTelemetryAfterFreeze(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.cursor = 1
	model.setPreviewLoading("developer")
	model.applyScanFinding("chrome_cache", "Chrome code cache", domain.Finding{
		ID:          "chrome-js",
		Name:        "Chrome Code Cache/js",
		Path:        "/tmp/chrome",
		DisplayPath: "/tmp/chrome",
		Category:    domain.CategoryBrowserData,
		Bytes:       84 << 20,
	})
	model.applyPreview("developer", domain.ExecutionPlan{
		Command: "clean",
		Profile: "developer",
		DryRun:  true,
		Totals:  domain.Totals{SafeBytes: 84 << 20, Bytes: 84 << 20, ItemCount: 1},
		Items: []domain.Finding{{
			ID:          "chrome-js",
			Name:        "Chrome Code Cache/js",
			Path:        "/tmp/chrome",
			DisplayPath: "/tmp/chrome",
			Category:    domain.CategoryBrowserData,
			Bytes:       84 << 20,
			Status:      domain.StatusPlanned,
		}},
	}, nil)

	view := ansi.Strip(model.View())
	if !strings.Contains(view, "REVIEW LANE") {
		t.Fatalf("expected review telemetry in frozen clean review, got %s", view)
	}
}

func TestCleanFlowViewUsesSweepStatsAndDeckLabels(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.cursor = 1
	model.setPreviewLoading("developer")
	model.applyScanFinding("chrome_cache", "Chrome code cache", domain.Finding{
		ID:          "chrome-js",
		Name:        "Chrome Code Cache/js",
		Path:        "/tmp/chrome",
		DisplayPath: "/tmp/chrome",
		Category:    domain.CategoryBrowserData,
		Bytes:       84 << 20,
	})

	view := strings.Join(strings.Fields(ansi.Strip(model.View())), " ")
	for _, needle := range []string{
		"SWEEP",
		"CLEAR",
		"WATCH",
		"YIELD",
		"forge sweep rail",
		"Sweep lanes",
		"FORGE HOT PATH",
		"residue stays",
		"gate",
		"YIELD",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean flow stat/deck language, got %s", needle, view)
		}
	}
}

func TestCleanFlowViewShowsPathCarryForActiveLedgerRows(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.cursor = 1
	model.setPreviewLoading("developer")
	model.applyScanFinding("xcode_cache", "Xcode DerivedData", domain.Finding{
		ID:          "derived-data",
		Name:        "DerivedData",
		Path:        "/tmp/derived",
		DisplayPath: "/tmp/derived",
		Category:    domain.CategoryDeveloperCaches,
		Bytes:       32 << 20,
	})

	view := ansi.Strip(model.View())
	for _, needle := range []string{
		"DerivedData",
		"LIVE",
		"path /tmp/derived",
		"ACTIVE SCAN",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean flow active row carry, got %s", needle, view)
		}
	}
}

func TestCleanFlowViewShowsWatchChromeForProtectedAndFailedRows(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.scanRows = []cleanFlowScanRow{
		{FindingID: "protected-a", Lane: "Browser caches", Label: "Chrome History", Path: "/tmp/history", Items: 1, Bytes: 8 << 20, State: "protected"},
		{FindingID: "failed-a", Lane: "Browser caches", Label: "Chrome Sessions", Path: "/tmp/sessions", Items: 1, Bytes: 4 << 20, State: "failed"},
	}
	model.phase = cleanFlowResult

	view := ansi.Strip(model.View())
	for _, needle := range []string{"WATCH", "PROTECTED", "FAILED"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean flow watch rows, got %s", needle, view)
		}
	}
}

func TestCleanFlowViewShowsArchiveChromeForSettledRows(t *testing.T) {
	t.Parallel()

	model := newCleanFlowModel()
	model.width = 132
	model.height = 34
	model.scanRows = []cleanFlowScanRow{
		{FindingID: "settled-a", Lane: "Browser caches", Label: "Chrome Code Cache/js", Path: "/tmp/cache", Items: 1, Bytes: 84 << 20, State: "settled"},
	}
	model.phase = cleanFlowResult

	view := ansi.Strip(model.View())
	for _, needle := range []string{"ARCHIVE", "SETTLED HOLD"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean flow settled rows, got %s", needle, view)
		}
	}
}
