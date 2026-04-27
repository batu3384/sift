package tui

import (
	"strings"
	"testing"

	"github.com/batu3384/sift/internal/domain"
)

func TestProgressStatsAdjustForNarrowAndWideLayouts(t *testing.T) {
	t.Parallel()

	progress := progressModel{
		plan: domain.ExecutionPlan{
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a"},
				{ID: "b", Path: "/tmp/b"},
			},
		},
		items: []domain.OperationResult{
			{FindingID: "a", Path: "/tmp/a", Status: domain.StatusDeleted},
			{FindingID: "b", Path: "/tmp/b", Status: domain.StatusCompleted},
		},
		spinnerFrame: 1,
	}

	narrow := progressStats(progress, 100)
	if len(narrow) != 3 {
		t.Fatalf("expected 3 stat cards for narrow layout, got %d", len(narrow))
	}

	wide := progressStats(progress, 140)
	if len(wide) != 4 {
		t.Fatalf("expected 4 stat cards for wide layout, got %d", len(wide))
	}
	joined := strings.Join(wide, "\n")
	for _, needle := range []string{"PROGRESS", "100%", "SETTLED", "2 / 2", "FREED"} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected %q in wide stats, got %s", needle, joined)
		}
	}
}

func TestProgressStageValueSummaryAndMeterForUninstall(t *testing.T) {
	t.Parallel()

	progress := progressModel{plan: domain.ExecutionPlan{Command: "uninstall"}}
	stage := stageInfo{
		Category: domain.CategoryAppLeftovers,
		Group:    "Very Long Application Batch Label",
		Index:    2,
		Total:    4,
		Done:     1,
		Items:    3,
		Bytes:    1536,
	}

	value := progressStageCardValue(progress, stage)
	if !strings.Contains(value, "target 2/4") || !strings.Contains(value, "…") {
		t.Fatalf("expected uninstall target value with truncation, got %q", value)
	}

	meter := progressMeterLine(progressModel{
		plan:  domain.ExecutionPlan{Items: []domain.Finding{{ID: "a"}, {ID: "b"}}},
		items: []domain.OperationResult{{FindingID: "a", Status: domain.StatusDeleted}},
	})
	if !strings.Contains(meter, "50%") {
		t.Fatalf("expected partial progress meter, got %q", meter)
	}
}

func TestProgressDetailShowsStageAndTotalCounters(t *testing.T) {
	t.Parallel()

	progress := progressModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", DisplayPath: "/tmp/a", Category: domain.CategoryBrowserData, Source: "Chrome code cache", Action: domain.ActionTrash},
				{ID: "b", Path: "/tmp/b", DisplayPath: "/tmp/b", Category: domain.CategoryBrowserData, Source: "Chrome code cache", Action: domain.ActionTrash},
				{ID: "c", Path: "/tmp/c", DisplayPath: "/tmp/c", Category: domain.CategoryLogs, Source: "Application logs", Action: domain.ActionTrash},
			},
		},
		items: []domain.OperationResult{
			{FindingID: "a", Path: "/tmp/a", Status: domain.StatusDeleted},
		},
		current:      &domain.Finding{ID: "b", Path: "/tmp/b", DisplayPath: "/tmp/b", Category: domain.CategoryBrowserData, Source: "Chrome code cache", Action: domain.ActionTrash},
		currentPhase: domain.ProgressPhaseRunning,
		autoFollow:   true,
	}

	view := progressDetailView(progress, 120, 20)
	for _, needle := range []string{
		"Stage    1/2 lanes  •  1/2 in stage  •  1/3 total settled",
		"Current  moving item to trash",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in progress detail counters, got:\n%s", needle, view)
		}
	}
}

func TestProgressMeterLineHandlesIdlePlans(t *testing.T) {
	t.Parallel()

	line := progressMeterLine(progressModel{})
	if !strings.Contains(line, "0%") {
		t.Fatalf("expected idle meter to report 0%%, got %q", line)
	}
}

func TestProgressAndResultFreedBytesExcludeCompletedNonDeleteActions(t *testing.T) {
	t.Parallel()

	plan := domain.ExecutionPlan{
		Command: "uninstall",
		Items: []domain.Finding{
			{ID: "native", Path: "/Applications/Example.app", Action: domain.ActionNative, Bytes: 500 << 20},
			{ID: "remnant", Path: "/tmp/example", Action: domain.ActionTrash, Bytes: 12 << 20},
		},
	}
	results := []domain.OperationResult{
		{FindingID: "native", Path: "/Applications/Example.app", Status: domain.StatusCompleted},
		{FindingID: "remnant", Path: "/tmp/example", Status: domain.StatusDeleted},
	}

	progressFreed := progressFreedBytes(progressModel{plan: plan, items: results})
	resultFreed := resultFreedBytes(plan, domain.ExecutionResult{Items: results})
	want := int64(12 << 20)
	if progressFreed != want {
		t.Fatalf("expected progress freed bytes to exclude native completion, got %d want %d", progressFreed, want)
	}
	if resultFreed != want {
		t.Fatalf("expected result freed bytes to exclude native completion, got %d want %d", resultFreed, want)
	}
}

func TestResultStatsAndSummaryReflectExecutionOutcome(t *testing.T) {
	t.Parallel()

	plan := domain.ExecutionPlan{Command: "clean", Items: []domain.Finding{{ID: "a"}, {ID: "b"}}}
	result := domain.ExecutionResult{
		Items: []domain.OperationResult{
			{Status: domain.StatusCompleted},
			{Status: domain.StatusDeleted},
			{Status: domain.StatusFailed},
			{Status: domain.StatusProtected},
			{Status: domain.StatusSkipped},
		},
	}

	stats := resultStats(plan, result, 140)
	if len(stats) != 4 {
		t.Fatalf("expected 4 result stat cards, got %d", len(stats))
	}
	joined := strings.Join(stats, "\n")
	for _, needle := range []string{"reclaim", "done", "2"} {
		if !strings.Contains(strings.ToLower(joined), needle) {
			t.Fatalf("expected %q in result stats, got %s", needle, joined)
		}
	}

	whatChanged := resultWhatChangedLine(result)
	for _, needle := range []string{"What changed", "1 deleted", "1 completed", "1 protected", "1 failed", "1 skipped"} {
		if !strings.Contains(whatChanged, needle) {
			t.Fatalf("expected %q in what-changed line, got %q", needle, whatChanged)
		}
	}
}

func TestResultListViewUsesReadableRows(t *testing.T) {
	t.Parallel()

	view := resultListView(resultModel{
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{Path: "/tmp/a", Status: domain.StatusDeleted, Message: "moved to trash"},
				{Path: "/tmp/b", Status: domain.StatusProtected, Reason: domain.ProtectionProtectedPath, Message: "Protected by policy."},
			},
		},
		cursor: 1,
	}, 160, 6)

	for _, needle := range []string{"✓", "/tmp/a", "⊘", "/tmp/b", "protected_path"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in result list, got %q", needle, view)
		}
	}
}

func TestResultAndProgressListsUseAvailableWidthForLabels(t *testing.T) {
	t.Parallel()

	longPath := "/Users/example/Library/Application Support/Google/Chrome/Default/Code Cache/js/app-long-cache-entry"
	resultView := resultListView(resultModel{
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{{Path: longPath, Status: domain.StatusDeleted}},
		},
	}, 160, 6)
	if !strings.Contains(resultView, "app-long-cache-entry") {
		t.Fatalf("expected wide result list to preserve useful label suffix, got:\n%s", resultView)
	}

	progressView := progressListView(progressModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{{
				ID:          "long",
				Path:        longPath,
				DisplayPath: longPath,
				Category:    domain.CategoryBrowserData,
				Bytes:       42 << 20,
			}},
		},
		cursor:     0,
		autoFollow: true,
	}, 160, 8)
	if !strings.Contains(progressView, "app-long-cache-entry") {
		t.Fatalf("expected wide progress list to preserve useful label suffix, got:\n%s", progressView)
	}
}

func TestResultWarningAndCommandLinesCollapseSingleItems(t *testing.T) {
	t.Parallel()

	warnings := strings.Join(resultWarningLines([]string{"review cache policy"}, 120), "\n")
	if strings.Contains(warnings, "Warnings") || !strings.Contains(warnings, "Warning  review cache policy") {
		t.Fatalf("expected compact warning line, got %q", warnings)
	}

	commands := strings.Join(resultCommandLines([]string{"sift clean"}, 120), "\n")
	if strings.Contains(commands, "Commands") || strings.Contains(commands, "Run\n") || !strings.Contains(commands, "Run      sift clean") {
		t.Fatalf("expected compact command line, got %q", commands)
	}
}

func TestResultDetailKeepsFollowUpCommandsNearSummaryOnShortPanes(t *testing.T) {
	t.Parallel()

	view := resultDetailView(resultModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", DisplayPath: "/tmp/a", Category: domain.CategoryBrowserData, Source: "Chrome code cache"},
				{ID: "b", Path: "/tmp/b", DisplayPath: "/tmp/b", Category: domain.CategoryLogs, Source: "Application logs"},
			},
		},
		result: domain.ExecutionResult{
			Warnings:         []string{"review protected cache policy"},
			FollowUpCommands: []string{"sift report latest"},
			Items: []domain.OperationResult{
				{FindingID: "a", Path: "/tmp/a", Status: domain.StatusDeleted},
				{FindingID: "b", Path: "/tmp/b", Status: domain.StatusFailed, Message: "permission denied"},
			},
		},
		width:  120,
		height: 12,
	}, 120, 8)

	for _, needle := range []string{"Warning  review protected cache policy", "Run      sift report latest"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected short result detail to keep %q visible, got:\n%s", needle, view)
		}
	}
}

func TestResultDetailSummarizesNotTouchedAndPermissionLimited(t *testing.T) {
	t.Parallel()

	view := resultDetailView(resultModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", DisplayPath: "/tmp/a", Category: domain.CategoryBrowserData, Source: "Chrome code cache", Action: domain.ActionTrash},
				{ID: "b", Path: "/tmp/b", DisplayPath: "/tmp/b", Category: domain.CategoryLogs, Source: "Application logs", Action: domain.ActionTrash},
				{ID: "c", Path: "/tmp/c", DisplayPath: "/tmp/c", Category: domain.CategoryLogs, Source: "Application logs", Action: domain.ActionTrash},
			},
		},
		result: domain.ExecutionResult{
			Warnings:         []string{"review protected cache policy"},
			FollowUpCommands: []string{"sift clean --profile safe"},
			Items: []domain.OperationResult{
				{FindingID: "a", Path: "/tmp/a", Status: domain.StatusDeleted},
				{FindingID: "b", Path: "/tmp/b", Status: domain.StatusProtected, Reason: domain.ProtectionProtectedPath},
				{FindingID: "c", Path: "/tmp/c", Status: domain.StatusFailed, Message: "permission denied"},
				{Path: "/tmp/skipped", Status: domain.StatusSkipped},
			},
		},
	}, 130, 24)

	for _, needle := range []string{
		"Not touched  1 protected  •  1 skipped  •  1 permission-limited",
		"Warning  review protected cache policy",
		"Run      sift clean --profile safe",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in result trust summary, got:\n%s", needle, view)
		}
	}
}

func TestResultDetailSelectedBlockUsesReadablePrimaryLine(t *testing.T) {
	t.Parallel()

	view := resultDetailView(resultModel{
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{Path: "/tmp/a", Status: domain.StatusFailed, Message: "permission denied"},
			},
		},
		width:  120,
		height: 20,
	}, 120, 20)

	for _, needle := range []string{"Selected", "Failed • /tmp/a", "permission denied"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in selected detail block, got %q", needle, view)
		}
	}
}

func TestResultDetailUsesUninstallSpecificRecoveryLanguage(t *testing.T) {
	t.Parallel()

	view := resultDetailView(resultModel{
		plan: domain.ExecutionPlan{
			Command: "uninstall",
			Targets: []string{"Example"},
			Items: []domain.Finding{
				{ID: "native", Path: "/Applications/Example.app", DisplayPath: "Example", Action: domain.ActionNative, Source: "Example native uninstall"},
				{ID: "remnant", Path: "/tmp/example", DisplayPath: "/tmp/example", Action: domain.ActionTrash, Category: domain.CategoryAppLeftovers, Source: "Example remnants"},
			},
		},
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{FindingID: "native", Path: "/Applications/Example.app", Status: domain.StatusFailed, Message: "handoff failed"},
				{FindingID: "remnant", Path: "/tmp/example", Status: domain.StatusProtected, Reason: domain.ProtectionProtectedPath},
			},
		},
		width:  140,
		height: 36,
	}, 140, 36)

	for _, needle := range []string{
		"Remnant Review",
		"Current Target",
		"Retry Handoff",
		"2 issues across 2 targets",
		"1 issue across 1 target",
		"Next    r retries failed handoff  •  x opens remnant review",
		"x opens remnant review",
		"m opens current target",
		"r retries failed handoff",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in uninstall result detail, got %q", needle, view)
		}
	}
}

func TestResultRecoveryBatchLinesHandleEmptyAndGroupedCandidates(t *testing.T) {
	t.Parallel()

	empty := strings.Join(resultRecoveryBatchLines(nil, 120), "\n")
	if !strings.Contains(empty, "No recovery items.") {
		t.Fatalf("expected empty recovery batch message, got %q", empty)
	}

	candidates := []domain.Finding{
		{Path: "/tmp/a", Category: domain.CategoryBrowserData, Source: "Chrome code cache", Bytes: 1024},
		{Path: "/tmp/b", Category: domain.CategoryBrowserData, Source: "Chrome code cache", Bytes: 2048},
		{Path: "/tmp/c", Category: domain.CategoryLogs, Source: "Application logs", Bytes: 512},
	}
	lines := strings.Join(resultRecoveryBatchLines(candidates, 140), "\n")
	for _, needle := range []string{"3 issues • 2 modules", "Chrome code cache • 2 items • 3.0 KB", "Application logs • 1 item • 512 B"} {
		if !strings.Contains(lines, needle) {
			t.Fatalf("expected %q in recovery batch lines, got %s", needle, lines)
		}
	}
}

func TestProgressExecutionRailReflectsSettledStatuses(t *testing.T) {
	t.Parallel()

	line := progressExecutionRail(progressModel{
		items: []domain.OperationResult{
			{Status: domain.StatusCompleted},
			{Status: domain.StatusDeleted},
			{Status: domain.StatusProtected},
			{Status: domain.StatusFailed},
		},
	})

	for _, needle := range []string{"Settled", "1 completed", "1 deleted", "1 protected", "1 failed"} {
		if !strings.Contains(line, needle) {
			t.Fatalf("expected %q in execution rail, got %q", needle, line)
		}
	}
}

func TestProgressExecutionRailUsesCommandSpecificBuckets(t *testing.T) {
	t.Parallel()

	line := progressExecutionRail(progressModel{
		plan: domain.ExecutionPlan{
			Command: "uninstall",
			Items: []domain.Finding{
				{ID: "native", Action: domain.ActionNative},
				{ID: "remnant", Action: domain.ActionTrash},
				{ID: "aftercare", Action: domain.ActionCommand, TaskPhase: "aftercare"},
			},
		},
		items: []domain.OperationResult{
			{FindingID: "native", Status: domain.StatusCompleted},
			{FindingID: "remnant", Status: domain.StatusDeleted},
			{FindingID: "aftercare", Status: domain.StatusCompleted},
		},
	})

	for _, needle := range []string{"Settled", "1 native", "1 removed", "1 aftercare"} {
		if !strings.Contains(line, needle) {
			t.Fatalf("expected %q in uninstall execution rail, got %q", needle, line)
		}
	}
}

func TestProgressDetailViewShowsRouteSignalForCleanRun(t *testing.T) {
	t.Parallel()

	view := progressDetailView(progressModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{{
				ID:          "chrome-js",
				Name:        "Chrome Code Cache/js",
				DisplayPath: "/tmp/chrome",
				Path:        "/tmp/chrome",
				Category:    domain.CategoryBrowserData,
				Action:      domain.ActionTrash,
				Bytes:       84 << 20,
			}},
		},
		currentPhase: domain.ProgressPhaseRunning,
		current: &domain.Finding{
			ID:          "chrome-js",
			Name:        "Chrome Code Cache/js",
			DisplayPath: "/tmp/chrome",
			Path:        "/tmp/chrome",
			Category:    domain.CategoryBrowserData,
			Action:      domain.ActionTrash,
			Bytes:       84 << 20,
		},
	}, 120, 24)

	for _, needle := range []string{"FORGE RAIL", "RECLAIM RAIL"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean progress detail, got %q", needle, view)
		}
	}
}

func TestResultDetailViewShowsRouteSignalForUninstallRun(t *testing.T) {
	t.Parallel()

	view := resultDetailView(resultModel{
		plan: domain.ExecutionPlan{
			Command: "uninstall",
			Items: []domain.Finding{{
				ID:          "native",
				Name:        "Example App",
				DisplayPath: "Example App",
				Path:        "/Applications/Example App.app",
				Action:      domain.ActionNative,
			}},
		},
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{{
				FindingID: "native",
				Path:      "/Applications/Example App.app",
				Status:    domain.StatusCompleted,
			}},
		},
	}, 120, 24)

	for _, needle := range []string{"COURIER RAIL", "AFTERCARE RAIL"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in uninstall result detail, got %q", needle, view)
		}
	}
}

func TestProgressCurrentLineReflectsBatchIntent(t *testing.T) {
	t.Parallel()

	progress := progressModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Profile: "safe",
			Totals:  domain.Totals{Bytes: 3 * 1024 * 1024},
			Items: []domain.Finding{
				{ID: "a", DisplayPath: "/tmp/cache", Action: domain.ActionTrash},
			},
		},
		current:      &domain.Finding{DisplayPath: "/tmp/cache", Action: domain.ActionTrash},
		currentPhase: domain.ProgressPhaseStarting,
	}

	current := progressCurrentLine(progress)
	for _, needle := range []string{"moving item to trash", "/tmp/cache"} {
		if !strings.Contains(current, needle) {
			t.Fatalf("expected %q in progress current line, got %q", needle, current)
		}
	}
}

func TestProgressCurrentLineShowsQueuedSelectedItemBeforeFirstUpdate(t *testing.T) {
	t.Parallel()

	progress := progressModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{
				{ID: "a", DisplayPath: "/tmp/cache", Path: "/tmp/cache", Action: domain.ActionTrash},
			},
		},
		current: &domain.Finding{ID: "a", DisplayPath: "/tmp/cache", Path: "/tmp/cache", Action: domain.ActionTrash},
	}

	line := progressCurrentLine(progress)
	for _, needle := range []string{"queued first approved item", "/tmp/cache"} {
		if !strings.Contains(line, needle) {
			t.Fatalf("expected %q in queued progress line, got %q", needle, line)
		}
	}
}

func TestProgressSummaryStatusAndNextLinesGuideExecution(t *testing.T) {
	t.Parallel()

	progress := progressModel{
		plan: domain.ExecutionPlan{
			Command: "optimize",
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", DisplayPath: "/tmp/a", Category: domain.CategoryMaintenance, Action: domain.ActionCommand},
			},
		},
		current:      &domain.Finding{ID: "a", Path: "/tmp/a", DisplayPath: "/tmp/a", Category: domain.CategoryMaintenance, Action: domain.ActionCommand},
		currentPhase: domain.ProgressPhaseStarting,
	}

	summary := progressSummaryLine(progress, progressStageInfo(progress))
	for _, needle := range []string{"Progress", "0%", "0/1 settled"} {
		if !strings.Contains(summary, needle) {
			t.Fatalf("expected %q in progress summary line, got %q", needle, summary)
		}
	}

	phase := progressPhaseLine(progress, progressStageInfo(progress))
	for _, needle := range []string{"Phase", "TASK", "1/1"} {
		if !strings.Contains(phase, needle) {
			t.Fatalf("expected %q in progress phase line, got %q", needle, phase)
		}
	}

	current := progressStepLine(progress)
	for _, needle := range []string{"Current", "running task", "/tmp/a"} {
		if !strings.Contains(current, needle) {
			t.Fatalf("expected %q in progress current line, got %q", needle, current)
		}
	}

	next := progressNextLine(progress)
	for _, needle := range []string{"Next", "result review"} {
		if !strings.Contains(next, needle) {
			t.Fatalf("expected %q in progress next line, got %q", needle, next)
		}
	}

	status := progressStatusLine(progress)
	for _, needle := range []string{"Status", "no completed operations yet"} {
		if !strings.Contains(status, needle) {
			t.Fatalf("expected %q in progress status line, got %q", needle, status)
		}
	}

	subtitle := progressDetailSubtitle(progress)
	for _, needle := range []string{"maintenance", "1/1", "queued"} {
		if !strings.Contains(subtitle, needle) {
			t.Fatalf("expected %q in progress detail subtitle, got %q", needle, subtitle)
		}
	}
}

func TestProgressListViewShowsHistoryHoldWhenBrowsingPastLiveItem(t *testing.T) {
	t.Parallel()

	progress := progressModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", DisplayPath: "/tmp/a", Category: domain.CategoryBrowserData},
				{ID: "b", Path: "/tmp/b", DisplayPath: "/tmp/b", Category: domain.CategoryBrowserData},
			},
		},
		cursor:       0,
		autoFollow:   false,
		current:      &domain.Finding{ID: "b", Path: "/tmp/b", DisplayPath: "/tmp/b", Category: domain.CategoryBrowserData},
		currentPhase: domain.ProgressPhaseRunning,
	}

	view := progressListView(progress, 120, 12)
	for _, needle := range []string{"History hold", "End returns to live item"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in progress history hold view, got:\n%s", needle, view)
		}
	}
}

func TestProgressStepLineUsesHandoffAndRemnantLanguage(t *testing.T) {
	t.Parallel()

	native := progressModel{
		plan:          domain.ExecutionPlan{Command: "uninstall"},
		current:       &domain.Finding{DisplayPath: "Example", Action: domain.ActionNative},
		currentPhase:  domain.ProgressPhaseRunning,
		currentStep:   "launch",
		currentDetail: "opening native uninstall",
	}
	if line := progressStepLine(native); !strings.Contains(line, "Current") || !strings.Contains(line, "opening native uninstall") {
		t.Fatalf("unexpected native uninstall step line: %q", line)
	}

	remnant := progressModel{
		plan:          domain.ExecutionPlan{Command: "uninstall"},
		current:       &domain.Finding{DisplayPath: "/tmp/example", Action: domain.ActionTrash},
		currentPhase:  domain.ProgressPhaseRunning,
		currentStep:   "trash",
		currentDetail: "moving /tmp/example to trash",
	}
	if line := progressStepLine(remnant); !strings.Contains(line, "Current") || !strings.Contains(line, "moving /tmp/example to trash") {
		t.Fatalf("unexpected remnant step line: %q", line)
	}
}

func TestResultScopeAndStatusLinesGuideRecovery(t *testing.T) {
	t.Parallel()

	model := resultModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Profile: "safe",
			Totals:  domain.Totals{Bytes: 3 * 1024 * 1024},
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", DisplayPath: "/tmp/a", Category: domain.CategoryBrowserData},
			},
		},
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{FindingID: "a", Path: "/tmp/a", Status: domain.StatusFailed},
			},
		},
	}

	scope := resultScopeLine(model)
	for _, needle := range []string{"Scope", "Quick Clean", "3.0 MB"} {
		if !strings.Contains(scope, needle) {
			t.Fatalf("expected %q in result scope line, got %q", needle, scope)
		}
	}

	status := resultStatusLine(model)
	for _, needle := range []string{"Status", "1 issue", "lane needs review"} {
		if !strings.Contains(status, needle) {
			t.Fatalf("expected %q in result status line, got %q", needle, status)
		}
	}
}

func TestResultSummaryAndNextLinesGuideRecovery(t *testing.T) {
	t.Parallel()

	model := resultModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Profile: "safe",
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", DisplayPath: "/tmp/a", Category: domain.CategoryBrowserData},
			},
		},
		result: domain.ExecutionResult{
			Warnings:         []string{"review"},
			FollowUpCommands: []string{"sift clean"},
			Items: []domain.OperationResult{
				{FindingID: "a", Path: "/tmp/a", Status: domain.StatusFailed},
			},
		},
	}

	summary := resultSummaryLine(model)
	for _, needle := range []string{"Result", "0% changed", "0/1 changed", "0 B freed"} {
		if !strings.Contains(summary, needle) {
			t.Fatalf("expected %q in result summary line, got %q", needle, summary)
		}
	}

	status := resultStatusLine(model)
	for _, needle := range []string{"Status", "1 issue", "1 warning", "1 follow-up command"} {
		if !strings.Contains(status, needle) {
			t.Fatalf("expected %q in result status line, got %q", needle, status)
		}
	}

	next := resultNextLine(model)
	for _, needle := range []string{"Next", "r retries failed", "x reopens recovery batch"} {
		if !strings.Contains(next, needle) {
			t.Fatalf("expected %q in result next line, got %q", needle, next)
		}
	}

	track := resultTrackLine(model)
	for _, needle := range []string{"Rail", "0 sections", "0 reclaimed", "1 open"} {
		if !strings.Contains(track, needle) {
			t.Fatalf("expected %q in result track line, got %q", needle, track)
		}
	}
}

func TestProgressSummaryAndResultSummaryUseCommandSpecificLanguage(t *testing.T) {
	t.Parallel()

	uninstallProgress := progressModel{
		plan: domain.ExecutionPlan{
			Command: "uninstall",
			Targets: []string{"Example App"},
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", DisplayPath: "/tmp/a", Category: domain.CategoryAppLeftovers, Action: domain.ActionTrash},
			},
		},
	}
	summary := progressSummaryLine(uninstallProgress, progressStageInfo(uninstallProgress))
	for _, needle := range []string{"Progress", "0%", "0/1 settled"} {
		if !strings.Contains(summary, needle) {
			t.Fatalf("expected %q in uninstall progress summary, got %q", needle, summary)
		}
	}

	optimizeResult := resultModel{
		plan: domain.ExecutionPlan{
			Command: "optimize",
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", DisplayPath: "/tmp/a", Category: domain.CategoryMaintenance, Action: domain.ActionCommand},
			},
		},
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{FindingID: "a", Path: "/tmp/a", Status: domain.StatusCompleted},
			},
		},
	}
	line := resultSummaryLine(optimizeResult)
	for _, needle := range []string{"Result", "100% changed", "1/1 changed"} {
		if !strings.Contains(line, needle) {
			t.Fatalf("expected %q in optimize result summary, got %q", needle, line)
		}
	}
}

func TestProgressStageInfoUsesSectionEventsFromExecutionStream(t *testing.T) {
	t.Parallel()

	progress := progressModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/cache", DisplayPath: "/tmp/cache", Category: domain.CategoryTempFiles, Action: domain.ActionTrash},
			},
		},
	}

	progress.apply(domain.ExecutionProgress{
		Event:        domain.ProgressEventSection,
		Phase:        domain.ProgressPhaseStarting,
		Step:         "section",
		Detail:       "starting user cache reclaim",
		Current:      1,
		SectionLabel: "User cache",
		SectionIndex: 1,
		SectionTotal: 3,
		SectionDone:  0,
		SectionItems: 4,
		SectionBytes: 2048,
		Item:         domain.Finding{Path: "/tmp/cache", DisplayPath: "/tmp/cache", Category: domain.CategoryTempFiles, Action: domain.ActionTrash},
	})

	stage := progressStageInfo(progress)
	if stage.Group != "User cache" || stage.Index != 1 || stage.Total != 3 || stage.Items != 4 || stage.Bytes != 2048 {
		t.Fatalf("expected explicit section stage info, got %+v", stage)
	}
}

func TestProgressStageInfoUsesUninstallAndOptimizeSectionEvents(t *testing.T) {
	t.Parallel()

	uninstall := progressModel{
		plan: domain.ExecutionPlan{
			Command: "uninstall",
			Items: []domain.Finding{
				{ID: "native", DisplayPath: "Example", Action: domain.ActionNative},
			},
		},
	}
	uninstall.apply(domain.ExecutionProgress{
		Event:        domain.ProgressEventSection,
		Phase:        domain.ProgressPhaseStarting,
		Step:         "section",
		Detail:       "starting native handoff",
		Current:      1,
		SectionLabel: "Native handoff",
		SectionIndex: 1,
		SectionTotal: 3,
		SectionDone:  0,
		SectionItems: 1,
		Item:         domain.Finding{DisplayPath: "Example", Action: domain.ActionNative},
	})
	stage := progressStageInfo(uninstall)
	if stage.Group != "Native handoff" || stage.Index != 1 || stage.Total != 3 {
		t.Fatalf("expected uninstall section stage info, got %+v", stage)
	}

	optimize := progressModel{
		plan: domain.ExecutionPlan{
			Command: "optimize",
			Items: []domain.Finding{
				{ID: "repair", DisplayPath: "/tmp/task", Action: domain.ActionCommand, TaskPhase: "repair"},
			},
		},
	}
	optimize.apply(domain.ExecutionProgress{
		Event:        domain.ProgressEventSection,
		Phase:        domain.ProgressPhaseStarting,
		Step:         "section",
		Detail:       "starting repair phase",
		Current:      1,
		SectionLabel: "Repair",
		SectionIndex: 1,
		SectionTotal: 2,
		SectionDone:  0,
		SectionItems: 1,
		Item:         domain.Finding{DisplayPath: "/tmp/task", Action: domain.ActionCommand, TaskPhase: "repair"},
	})
	stage = progressStageInfo(optimize)
	if stage.Group != "Repair" || stage.Index != 1 || stage.Total != 2 {
		t.Fatalf("expected optimize section stage info, got %+v", stage)
	}
}

func TestResultTrackLineUsesCommandSpecificBuckets(t *testing.T) {
	t.Parallel()

	model := resultModel{
		plan: domain.ExecutionPlan{
			Command: "uninstall",
			Items: []domain.Finding{
				{ID: "native", Path: "/Applications/Example.app", Action: domain.ActionNative},
				{ID: "remnant", Path: "/tmp/example", Action: domain.ActionTrash},
				{ID: "aftercare", Path: "/tmp/launchctl", Action: domain.ActionCommand, TaskPhase: "aftercare"},
			},
		},
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{FindingID: "native", Path: "/Applications/Example.app", Status: domain.StatusCompleted},
				{FindingID: "remnant", Path: "/tmp/example", Status: domain.StatusDeleted},
				{FindingID: "aftercare", Path: "/tmp/launchctl", Status: domain.StatusCompleted},
			},
		},
	}

	line := resultTrackLine(model)
	for _, needle := range []string{"Rail", "1 native", "1 removed", "1 aftercare"} {
		if !strings.Contains(line, needle) {
			t.Fatalf("expected %q in uninstall result track line, got %q", needle, line)
		}
	}
	status := resultStatusLine(model)
	for _, needle := range []string{"Status", "lane settled cleanly"} {
		if !strings.Contains(status, needle) {
			t.Fatalf("expected %q in uninstall result status line, got %q", needle, status)
		}
	}
}

func TestResultTrackLineUsesSectionAndPhaseLabels(t *testing.T) {
	t.Parallel()

	cleanModel := resultModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", Category: domain.CategoryBrowserData, Source: "Chrome code cache"},
				{ID: "b", Path: "/tmp/b", Category: domain.CategoryLogs, Source: "Application logs"},
			},
		},
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{FindingID: "a", Path: "/tmp/a", Status: domain.StatusDeleted},
			},
		},
	}
	if line := resultTrackLine(cleanModel); !strings.Contains(line, "1 section") {
		t.Fatalf("expected clean result track to include settled sections, got %q", line)
	}

	optimizeModel := resultModel{
		plan: domain.ExecutionPlan{
			Command: "optimize",
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", Action: domain.ActionCommand, TaskPhase: "repair"},
				{ID: "b", Path: "/tmp/b", Action: domain.ActionCommand, TaskPhase: "refresh"},
			},
		},
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{FindingID: "a", Path: "/tmp/a", Status: domain.StatusCompleted},
				{FindingID: "b", Path: "/tmp/b", Status: domain.StatusCompleted},
			},
		},
	}
	if line := resultTrackLine(optimizeModel); !strings.Contains(line, "repair 1 • refresh 1") {
		t.Fatalf("expected optimize result track to include phase breakdown, got %q", line)
	}
	if line := resultStatusLine(optimizeModel); !strings.Contains(line, "Status") || !strings.Contains(line, "lane settled cleanly") {
		t.Fatalf("expected optimize result status to show settled summary, got %q", line)
	}
}

func TestResultNextRailGuidesRetryAndRecovery(t *testing.T) {
	t.Parallel()

	line := resultNextRail(resultModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", DisplayPath: "/tmp/a", Category: domain.CategoryBrowserData},
			},
		},
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{FindingID: "a", Path: "/tmp/a", Status: domain.StatusFailed},
			},
		},
	})

	for _, needle := range []string{"Next rail", "r retries failed", "x reopens recovery batch"} {
		if !strings.Contains(line, needle) {
			t.Fatalf("expected %q in result next rail, got %q", needle, line)
		}
	}
}
