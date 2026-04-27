package tui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/platform"
)

func deliverCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected command")
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, next := range batch {
			if next == nil {
				continue
			}
			batchMsg := next()
			switch batchMsg.(type) {
			case uiTickMsg:
				continue
			default:
				return batchMsg
			}
		}
		t.Fatal("expected actionable message in batch command")
	}
	return msg
}

func TestAppRouterHomeStatusHome(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	model.applyDashboard(testDashboardData())
	model.home.cursor = findHomeAction(t, model.home.actions, "status")

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	routed := next.(appModel)
	if routed.route != RouteStatus {
		t.Fatalf("expected status route, got %s", routed.route)
	}

	back, cmd := routed.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected dashboard refresh command on back from status")
	}
	refreshed, _ := back.(appModel).Update(deliverCmd(t, cmd))
	if refreshed.(appModel).route != RouteHome {
		t.Fatalf("expected home route after refresh, got %s", refreshed.(appModel).route)
	}
}

func TestAppRouterHomeToolsDoctorTools(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	model.applyDashboard(testDashboardData())
	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	routed := next.(appModel)
	if routed.route != RouteTools {
		t.Fatalf("expected tools route, got %s", routed.route)
	}

	routed.tools.cursor = findHomeAction(t, routed.tools.actions, "doctor")
	next, _ = routed.Update(tea.KeyMsg{Type: tea.KeyEnter})
	doctor := next.(appModel)
	if doctor.route != RouteDoctor {
		t.Fatalf("expected doctor route, got %s", doctor.route)
	}

	back, _ := doctor.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if back.(appModel).route != RouteTools {
		t.Fatalf("expected tools route after doctor back, got %s", back.(appModel).route)
	}
}

func TestHomeRouteUsesToolsBinding(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	model.keys.Tools = key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "tools"))

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if next.(appModel).route != RouteHome {
		t.Fatalf("expected old tools key to do nothing, got %s", next.(appModel).route)
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if next.(appModel).route != RouteTools {
		t.Fatalf("expected custom tools key to open tools, got %s", next.(appModel).route)
	}
}

func TestStatusRouteUsesRefreshBinding(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteStatus)
	model.keys.Refresh = key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "refresh"))
	called := 0
	model.callbacks.LoadDashboard = func() (DashboardData, error) {
		called++
		return testDashboardData(), nil
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd != nil || next.(appModel).status.loading {
		t.Fatal("expected old refresh key to do nothing")
	}

	next, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	if cmd == nil {
		t.Fatal("expected custom refresh binding to trigger command")
	}
	if !next.(appModel).status.loading || next.(appModel).currentLoadingLabel() != "dashboard" {
		t.Fatalf("expected status loading after custom refresh, got loading=%v label=%q", next.(appModel).status.loading, next.(appModel).currentLoadingLabel())
	}
	deliverCmd(t, cmd)
	if called != 1 {
		t.Fatalf("expected dashboard loader to run once, got %d", called)
	}
}

func TestAnalyzeRouteUsesFocusBinding(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.keys.Focus = key.NewBinding(key.WithKeys("F"), key.WithHelp("F", "focus"))
	model.analyze.plan = domain.ExecutionPlan{
		Command: "analyze",
		Items: []domain.Finding{{
			ID:       "one",
			Path:     "/tmp/cache",
			Category: domain.CategoryDiskUsage,
			Status:   domain.StatusAdvisory,
			Fingerprint: domain.Fingerprint{
				Mode: 0o040755,
			},
		}},
	}
	model.analyze.toggleStage(model.analyze.plan.Items[0])

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	if next.(appModel).analyze.activePane() != analyzePaneBrowse {
		t.Fatalf("expected old focus key to do nothing, got %s", next.(appModel).analyze.activePane())
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'F'}})
	if next.(appModel).analyze.activePane() != analyzePaneBrowse {
		t.Fatalf("expected single staged item to stay in browse focus, got %s", next.(appModel).analyze.activePane())
	}
}

func TestHelpToggleShowsOverlayAndBlocksRouteActions(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	model.width = 120
	model.height = 36
	startCursor := model.home.cursor

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	helped := next.(appModel)
	if !helped.helpVisible {
		t.Fatal("expected help overlay to open")
	}
	if !strings.Contains(helped.View(), "Home scout rail • ? or esc closes") {
		t.Fatalf("expected help overlay view, got:\n%s", helped.View())
	}

	next, _ = helped.Update(tea.KeyMsg{Type: tea.KeyDown})
	blocked := next.(appModel)
	if blocked.home.cursor != startCursor {
		t.Fatalf("expected help overlay to block route actions, got cursor=%d want=%d", blocked.home.cursor, startCursor)
	}

	next, _ = blocked.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if next.(appModel).helpVisible {
		t.Fatal("expected help overlay to close")
	}
}

func TestStatusRouteUsesCompanionBinding(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteStatus)
	model.keys.Companion = key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "companion"))
	model.status.companionMode = "off"
	initialMode := model.status.companionMode

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if next.(appModel).status.companionMode != initialMode {
		t.Fatalf("expected old companion key to do nothing, got %q want %q", next.(appModel).status.companionMode, initialMode)
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	if next.(appModel).status.companionMode != "full" {
		t.Fatalf("expected custom companion key to toggle mode, got %q", next.(appModel).status.companionMode)
	}
}

func TestAppRouterToolsAutofixLoadsReviewPlan(t *testing.T) {
	t.Parallel()

	autofixPlan := domain.ExecutionPlan{
		Command:  "autofix",
		DryRun:   true,
		Platform: "darwin",
		Items: []domain.Finding{{
			ID:          "fix-1",
			RuleID:      "autofix.firewall",
			Name:        "Enable macOS firewall",
			DisplayPath: "/usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate on",
			Status:      domain.StatusPlanned,
			Action:      domain.ActionCommand,
			Category:    domain.CategoryMaintenance,
		}},
	}

	model := newTestAppModel(RouteTools)
	model.callbacks.LoadAutofix = func() (domain.ExecutionPlan, error) { return autofixPlan, nil }
	model.tools.cursor = findHomeAction(t, model.tools.actions, "autofix")

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected autofix load command")
	}
	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	review := next.(appModel)
	if review.route != RouteReview {
		t.Fatalf("expected review route, got %s", review.route)
	}
	if review.review.plan.Command != "autofix" {
		t.Fatalf("expected autofix review plan, got %+v", review.review.plan)
	}
	if review.review.requiresDecision {
		t.Fatal("expected autofix review to stay in preview mode for dry-run plans")
	}
}

func TestAppRouterHomeAnalyzeReviewProgressResultHome(t *testing.T) {
	t.Parallel()

	analyzePlan := domain.ExecutionPlan{
		Command:  "analyze",
		DryRun:   true,
		Platform: "darwin",
		Items: []domain.Finding{{
			Path:        "/tmp/cache",
			DisplayPath: "/tmp/cache",
			Status:      domain.StatusAdvisory,
			Category:    domain.CategoryDiskUsage,
		}},
	}
	reviewPlan := domain.ExecutionPlan{
		Command:  "clean",
		DryRun:   false,
		Platform: "darwin",
		Items: []domain.Finding{{
			Path:        "/tmp/cache",
			DisplayPath: "/tmp/cache",
			Status:      domain.StatusPlanned,
			Action:      domain.ActionTrash,
			Category:    domain.CategorySystemClutter,
		}},
	}
	result := domain.ExecutionResult{
		Items: []domain.OperationResult{{Path: "/tmp/cache", Status: domain.StatusDeleted}},
	}

	model := newTestAppModel(RouteHome)
	model.applyDashboard(testDashboardData())
	model.callbacks.LoadAnalyzeHome = func() (domain.ExecutionPlan, error) { return analyzePlan, nil }
	model.callbacks.LoadReviewForPaths = func(paths []string) (domain.ExecutionPlan, error) {
		if len(paths) != 1 || paths[0] != "/tmp/cache" {
			t.Fatalf("unexpected staged paths: %+v", paths)
		}
		return reviewPlan, nil
	}
	model.callbacks.ExecutePlanWithProgress = func(_ context.Context, plan domain.ExecutionPlan, emit func(domain.ExecutionProgress)) (domain.ExecutionResult, error) {
		if plan.Command != "clean" {
			t.Fatalf("unexpected execution plan command: %s", plan.Command)
		}
		emit(domain.ExecutionProgress{
			ScanID:    plan.ScanID,
			Current:   1,
			Completed: 0,
			Total:     1,
			Phase:     domain.ProgressPhaseStarting,
			Item:      plan.Items[0],
		})
		emit(domain.ExecutionProgress{
			ScanID:    plan.ScanID,
			Current:   1,
			Completed: 1,
			Total:     1,
			Phase:     domain.ProgressPhaseFinished,
			Item:      plan.Items[0],
			Result:    result.Items[0],
		})
		return result, nil
	}
	model.home.cursor = findHomeAction(t, model.home.actions, "analyze")

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected analyze load command")
	}
	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	analyze := next.(appModel)
	if analyze.route != RouteAnalyze {
		t.Fatalf("expected analyze route, got %s", analyze.route)
	}

	next, _ = analyze.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	analyze = next.(appModel)
	next, cmd = analyze.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd == nil {
		t.Fatal("expected review load command")
	}
	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	review := next.(appModel)
	if review.route != RouteReview {
		t.Fatalf("expected review route, got %s", review.route)
	}

	next, cmd = review.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected execute command")
	}
	progress := next.(appModel)
	if progress.route != RouteProgress {
		t.Fatalf("expected progress route, got %s", progress.route)
	}

	next, cmd = progress.Update(deliverCmd(t, cmd))
	streaming := next.(appModel)
	if streaming.route != RouteProgress {
		t.Fatalf("expected progress route during stream, got %s", streaming.route)
	}
	if streaming.progress.currentPhase != domain.ProgressPhaseStarting {
		t.Fatalf("expected starting progress phase, got %+v", streaming.progress)
	}
	if streaming.progress.cursor != 0 {
		t.Fatalf("expected running cursor to point at current item, got %d", streaming.progress.cursor)
	}
	if cmd == nil {
		t.Fatal("expected progress stream to continue")
	}

	next, cmd = streaming.Update(deliverCmd(t, cmd))
	finishing := next.(appModel)
	if finishing.route != RouteProgress {
		t.Fatalf("expected progress route during finishing stream, got %s", finishing.route)
	}
	if cmd == nil {
		t.Fatal("expected final stream message")
	}

	next, _ = finishing.Update(deliverCmd(t, cmd))
	resultRoute := next.(appModel)
	if resultRoute.route != RouteResult {
		t.Fatalf("expected result route, got %s", resultRoute.route)
	}
	if !resultRoute.result.flash {
		t.Fatal("expected result flash state after execution finishes")
	}

	next, cmd = resultRoute.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected dashboard refresh command on result close")
	}
	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	if next.(appModel).route != RouteHome {
		t.Fatalf("expected home route after result, got %s", next.(appModel).route)
	}
}

func TestAppRouterProgressCanStopExecution(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteReview)
	model.setReviewPlan(domain.ExecutionPlan{
		Command:  "clean",
		DryRun:   false,
		Platform: "darwin",
		Items: []domain.Finding{{
			ID:          "1",
			Path:        "/tmp/cache",
			DisplayPath: "/tmp/cache",
			Status:      domain.StatusPlanned,
			Action:      domain.ActionTrash,
			Category:    domain.CategorySystemClutter,
		}},
	}, true)
	stopped := make(chan struct{}, 1)
	model.callbacks.ExecutePlanWithProgress = func(ctx context.Context, plan domain.ExecutionPlan, emit func(domain.ExecutionProgress)) (domain.ExecutionResult, error) {
		emit(domain.ExecutionProgress{
			ScanID:    plan.ScanID,
			Current:   1,
			Completed: 0,
			Total:     1,
			Phase:     domain.ProgressPhaseStarting,
			Item:      plan.Items[0],
		})
		<-ctx.Done()
		stopped <- struct{}{}
		return domain.ExecutionResult{
			ScanID: plan.ScanID,
			Items:  []domain.OperationResult{},
		}, ctx.Err()
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected progress execution command")
	}
	progress := next.(appModel)
	if progress.route != RouteProgress {
		t.Fatalf("expected progress route, got %s", progress.route)
	}

	next, cmd = progress.Update(deliverCmd(t, cmd))
	running := next.(appModel)
	if cmd == nil {
		t.Fatal("expected stream wait command")
	}
	next, _ = running.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	stopping := next.(appModel)
	if !stopping.progress.cancelRequested {
		t.Fatal("expected cancelRequested to be set")
	}

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("expected execution context to be cancelled")
	}

	next, _ = stopping.Update(deliverCmd(t, cmd))
	result := next.(appModel)
	if result.route != RouteResult {
		t.Fatalf("expected result route after cancellation, got %s", result.route)
	}
	if result.errorMsg != "" {
		t.Fatalf("expected cancellation to avoid error bar, got %q", result.errorMsg)
	}
	if !strings.Contains(result.noticeMsg, "cancelled") {
		t.Fatalf("expected cancellation notice, got %q", result.noticeMsg)
	}
}

func TestReviewExecutionCarriesSelectionIntoProgress(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteReview)
	model.setReviewPlan(domain.ExecutionPlan{
		Command:  "clean",
		DryRun:   false,
		Platform: "darwin",
		Items: []domain.Finding{
			{ID: "1", Path: "/tmp/a", DisplayPath: "/tmp/a", Action: domain.ActionTrash, Status: domain.StatusPlanned, Category: domain.CategorySystemClutter},
			{ID: "2", Path: "/tmp/b", DisplayPath: "/tmp/b", Action: domain.ActionTrash, Status: domain.StatusPlanned, Category: domain.CategoryLogs},
		},
	}, true)
	model.review.cursor = 1
	model.callbacks.ExecutePlanWithProgress = func(ctx context.Context, plan domain.ExecutionPlan, emit func(domain.ExecutionProgress)) (domain.ExecutionResult, error) {
		return domain.ExecutionResult{}, nil
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected execution command")
	}
	progress := next.(appModel)
	if progress.route != RouteProgress {
		t.Fatalf("expected progress route, got %s", progress.route)
	}
	if progress.progress.cursor != 1 {
		t.Fatalf("expected progress cursor to preserve review selection, got %d", progress.progress.cursor)
	}
	if progress.progress.current == nil || progress.progress.current.Path != "/tmp/b" {
		t.Fatalf("expected progress current preview to match selected item, got %+v", progress.progress.current)
	}
}

func TestReviewEnterDoesNotExecute(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteReview)
	model.setReviewPlan(domain.ExecutionPlan{
		Command:  "clean",
		DryRun:   false,
		Platform: "darwin",
		Items: []domain.Finding{{
			ID:          "1",
			Path:        "/tmp/cache",
			DisplayPath: "/tmp/cache",
			Status:      domain.StatusPlanned,
			Action:      domain.ActionTrash,
			Category:    domain.CategorySystemClutter,
		}},
	}, true)
	model.callbacks.ExecutePlanWithProgress = func(ctx context.Context, plan domain.ExecutionPlan, emit func(domain.ExecutionProgress)) (domain.ExecutionResult, error) {
		t.Fatal("enter should not start execution")
		return domain.ExecutionResult{}, nil
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("did not expect execution command on enter")
	}
	if next.(appModel).route != RouteReview {
		t.Fatalf("expected to remain on review, got %s", next.(appModel).route)
	}
}

func TestReviewSpaceExcludesItemFromExecutionPlan(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteReview)
	model.setReviewPlan(domain.ExecutionPlan{
		Command:  "clean",
		DryRun:   false,
		Platform: "darwin",
		Items: []domain.Finding{
			{
				ID:          "1",
				Path:        "/tmp/cache-a",
				DisplayPath: "/tmp/cache-a",
				Status:      domain.StatusPlanned,
				Action:      domain.ActionTrash,
				Category:    domain.CategorySystemClutter,
			},
			{
				ID:          "2",
				Path:        "/tmp/cache-b",
				DisplayPath: "/tmp/cache-b",
				Status:      domain.StatusPlanned,
				Action:      domain.ActionTrash,
				Category:    domain.CategorySystemClutter,
			},
		},
	}, true)
	model.callbacks.ExecutePlanWithProgress = func(ctx context.Context, plan domain.ExecutionPlan, emit func(domain.ExecutionProgress)) (domain.ExecutionResult, error) {
		if plan.Items[0].Action != domain.ActionSkip || plan.Items[0].Status != domain.StatusSkipped {
			t.Fatalf("expected first item to be excluded, got %+v", plan.Items[0])
		}
		if plan.Items[1].Action != domain.ActionTrash {
			t.Fatalf("expected second item to remain actionable, got %+v", plan.Items[1])
		}
		return domain.ExecutionResult{}, nil
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	review := next.(appModel)
	next, cmd := review.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected execution command after explicit y")
	}
	if next.(appModel).route != RouteProgress {
		t.Fatalf("expected progress route, got %s", next.(appModel).route)
	}
}

func TestReviewExecuteRoutesThroughPermissionsWhenAccessNeedsPreflight(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteReview)
	model.width = 132
	model.height = 30
	model.setReviewPlan(domain.ExecutionPlan{
		Command:  "optimize",
		DryRun:   false,
		Platform: "darwin",
		Items: []domain.Finding{{
			ID:            "secure-task",
			Name:          "Reset caches",
			Path:          "/Library/Caches/system",
			DisplayPath:   "/usr/bin/sudo /usr/bin/true",
			Status:        domain.StatusPlanned,
			Action:        domain.ActionCommand,
			CommandPath:   "/usr/bin/sudo",
			CommandArgs:   []string{"/usr/bin/true"},
			RequiresAdmin: true,
			Category:      domain.CategoryMaintenance,
		}},
	}, true)

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd != nil {
		t.Fatal("did not expect execution to start before permission preflight")
	}
	preflight := next.(appModel)
	if preflight.route != RoutePreflight {
		t.Fatalf("expected preflight route, got %s", preflight.route)
	}
	view := preflight.View()
	for _, needle := range []string{"SIFT / ACCESS CHECK", "macOS password", "MANIFEST DECK", "Reset caches", "y run • esc back"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in preflight view, got:\n%s", needle, view)
		}
	}
}

func TestPermissionPreflightCanReturnToReview(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteReview)
	model.setReviewPlan(domain.ExecutionPlan{
		Command:  "optimize",
		DryRun:   false,
		Platform: "darwin",
		Items: []domain.Finding{{
			ID:            "secure-task",
			Path:          "/Library/Caches/system",
			DisplayPath:   "/usr/bin/sudo /usr/bin/true",
			Status:        domain.StatusPlanned,
			Action:        domain.ActionCommand,
			CommandPath:   "/usr/bin/sudo",
			CommandArgs:   []string{"/usr/bin/true"},
			RequiresAdmin: true,
			Category:      domain.CategoryMaintenance,
		}},
	}, true)

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	preflight := next.(appModel)
	next, _ = preflight.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if next.(appModel).route != RouteReview {
		t.Fatalf("expected review route after preflight back, got %s", next.(appModel).route)
	}
}

func TestPermissionPreflightWarmsAccessThenStartsExecution(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteReview)
	model.setReviewPlan(domain.ExecutionPlan{
		Command:  "optimize",
		DryRun:   false,
		Platform: "darwin",
		Items: []domain.Finding{{
			ID:            "secure-task",
			Path:          "/Library/Caches/system",
			DisplayPath:   "/usr/bin/sudo /usr/bin/true",
			Status:        domain.StatusPlanned,
			Action:        domain.ActionCommand,
			CommandPath:   "/usr/bin/sudo",
			CommandArgs:   []string{"/usr/bin/true"},
			RequiresAdmin: true,
			Category:      domain.CategoryMaintenance,
		}},
	}, true)
	warmupCalled := false
	keepaliveCalled := false
	model.permissionWarmup = func(pre permissionPreflightModel) tea.Cmd {
		warmupCalled = true
		if !pre.needsAdmin {
			t.Fatal("expected admin warmup request")
		}
		return func() tea.Msg { return permissionWarmupFinishedMsg{} }
	}
	model.permissionKeepalive = func(ctx context.Context, pre permissionPreflightModel) {
		keepaliveCalled = true
		if !pre.needsAdmin {
			t.Fatal("expected admin keepalive")
		}
	}
	model.callbacks.ExecutePlanWithProgress = func(ctx context.Context, plan domain.ExecutionPlan, emit func(domain.ExecutionProgress)) (domain.ExecutionResult, error) {
		return domain.ExecutionResult{}, nil
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	preflight := next.(appModel)
	if preflight.route != RoutePreflight {
		t.Fatalf("expected preflight route, got %s", preflight.route)
	}

	next, cmd := preflight.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected warmup command")
	}
	if !warmupCalled {
		t.Fatal("expected permission warmup to run")
	}

	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	progress := next.(appModel)
	if progress.route != RouteProgress {
		t.Fatalf("expected progress route after warmup, got %s", progress.route)
	}
	if !keepaliveCalled {
		t.Fatal("expected permission keepalive to start")
	}
	if cmd == nil {
		t.Fatal("expected execution stream command")
	}
}

func TestAcceptedPermissionProfileSkipsRepeatPreflight(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteReview)
	plan := domain.ExecutionPlan{
		Command:  "optimize",
		DryRun:   false,
		Platform: "darwin",
		Items: []domain.Finding{{
			ID:            "secure-task",
			Path:          "/Library/Caches/system",
			DisplayPath:   "/usr/bin/sudo /usr/bin/true",
			Name:          "Reset caches",
			Status:        domain.StatusPlanned,
			Action:        domain.ActionCommand,
			CommandPath:   "/usr/bin/sudo",
			CommandArgs:   []string{"/usr/bin/true"},
			RequiresAdmin: true,
			Category:      domain.CategoryMaintenance,
		}},
	}
	model.setReviewPlan(plan, true)

	warmupCalled := 0
	model.permissionWarmup = func(pre permissionPreflightModel) tea.Cmd {
		warmupCalled++
		return func() tea.Msg { return permissionWarmupFinishedMsg{} }
	}
	model.permissionKeepalive = func(ctx context.Context, pre permissionPreflightModel) {}
	model.callbacks.ExecutePlanWithProgress = func(ctx context.Context, plan domain.ExecutionPlan, emit func(domain.ExecutionProgress)) (domain.ExecutionResult, error) {
		return domain.ExecutionResult{}, nil
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	preflight := next.(appModel)
	if preflight.route != RoutePreflight {
		t.Fatalf("expected preflight route, got %s", preflight.route)
	}
	next, cmd := preflight.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected first warmup command")
	}
	if warmupCalled != 1 {
		t.Fatalf("expected first warmup, got %d", warmupCalled)
	}
	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	progress := next.(appModel)
	if progress.route != RouteProgress {
		t.Fatalf("expected progress route after first warmup, got %s", progress.route)
	}

	progress.route = RouteReview
	progress.setReviewPlan(plan, true)
	next, cmd = progress.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	review := next.(appModel)
	if review.route == RoutePreflight {
		t.Fatal("did not expect repeat preflight for accepted permission profile")
	}
	if cmd == nil {
		t.Fatal("expected direct warmup command for accepted profile")
	}
	if warmupCalled != 2 {
		t.Fatalf("expected direct second warmup, got %d", warmupCalled)
	}
	next, cmd = review.Update(deliverCmd(t, cmd))
	if next.(appModel).route != RouteProgress {
		t.Fatalf("expected progress route after repeat warmup, got %s", next.(appModel).route)
	}
	if cmd == nil {
		t.Fatal("expected execution stream command after repeat warmup")
	}
}

func TestResultCanOpenRecoveryReview(t *testing.T) {
	t.Parallel()

	reviewPlan := domain.ExecutionPlan{
		Command:  "clean",
		DryRun:   false,
		Platform: "darwin",
		Items: []domain.Finding{
			{
				ID:          "failed-1",
				Path:        "/tmp/cache",
				DisplayPath: "/tmp/cache",
				Status:      domain.StatusPlanned,
				Action:      domain.ActionTrash,
				Category:    domain.CategorySystemClutter,
			},
		},
	}
	model := newTestAppModel(RouteResult)
	model.review.plan = reviewPlan
	model.result = resultModel{
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{FindingID: "failed-1", Path: "/tmp/cache", Status: domain.StatusFailed, Message: "permission denied"},
			},
		},
	}
	model.resultReturnRoute = RouteHome
	model.callbacks.LoadReviewForPaths = func(paths []string) (domain.ExecutionPlan, error) {
		if len(paths) != 1 || paths[0] != "/tmp/cache" {
			t.Fatalf("unexpected recovery paths: %+v", paths)
		}
		return reviewPlan, nil
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd == nil {
		t.Fatal("expected recovery review load command")
	}
	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	review := next.(appModel)
	if review.route != RouteReview {
		t.Fatalf("expected review route, got %s", review.route)
	}
	if review.reviewReturnRoute != RouteResult {
		t.Fatalf("expected result as review return route, got %s", review.reviewReturnRoute)
	}
}

func TestProgressManualBrowseStopsAutoFollow(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteProgress)
	model.progress = progressModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", DisplayPath: "/tmp/a"},
				{ID: "b", Path: "/tmp/b", DisplayPath: "/tmp/b"},
				{ID: "c", Path: "/tmp/c", DisplayPath: "/tmp/c"},
			},
		},
		cursor:     1,
		autoFollow: true,
		current:    &domain.Finding{ID: "b", Path: "/tmp/b", DisplayPath: "/tmp/b"},
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	browsing := next.(appModel)
	if browsing.progress.cursor != 2 {
		t.Fatalf("expected manual browse cursor, got %d", browsing.progress.cursor)
	}
	if browsing.progress.autoFollow {
		t.Fatal("expected manual browse to disable auto-follow")
	}

	next, _ = browsing.Update(executionProgressMsg{
		progress: domain.ExecutionProgress{
			Current: 2,
			Total:   3,
			Phase:   domain.ProgressPhaseStarting,
			Item:    domain.Finding{ID: "b", Path: "/tmp/b", DisplayPath: "/tmp/b"},
		},
	})
	updated := next.(appModel)
	if updated.progress.cursor != 2 {
		t.Fatalf("expected cursor to stay on browsed item, got %d", updated.progress.cursor)
	}
}

func TestProgressHistoryKeysPageBrowseAndReturnToLive(t *testing.T) {
	t.Parallel()

	items := make([]domain.Finding, 12)
	for idx := range items {
		path := fmt.Sprintf("/tmp/item-%02d", idx)
		items[idx] = domain.Finding{ID: fmt.Sprintf("item-%02d", idx), Path: path, DisplayPath: path}
	}
	current := items[9]
	model := newTestAppModel(RouteProgress)
	model.height = 24
	model.progress = progressModel{
		plan:       domain.ExecutionPlan{Command: "clean", Items: items},
		cursor:     9,
		autoFollow: true,
		current:    &current,
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	pagedUp := next.(appModel)
	if pagedUp.progress.cursor >= model.progress.cursor || pagedUp.progress.autoFollow {
		t.Fatalf("expected PageUp to browse older progress history, got cursor=%d autoFollow=%v", pagedUp.progress.cursor, pagedUp.progress.autoFollow)
	}

	next, _ = pagedUp.Update(tea.KeyMsg{Type: tea.KeyHome})
	oldest := next.(appModel)
	if oldest.progress.cursor != 0 || oldest.progress.autoFollow {
		t.Fatalf("expected Home to jump to oldest progress item, got cursor=%d autoFollow=%v", oldest.progress.cursor, oldest.progress.autoFollow)
	}

	next, _ = oldest.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	pagedDown := next.(appModel)
	if pagedDown.progress.cursor <= oldest.progress.cursor || pagedDown.progress.autoFollow {
		t.Fatalf("expected PageDown to browse toward live progress, got cursor=%d autoFollow=%v", pagedDown.progress.cursor, pagedDown.progress.autoFollow)
	}

	next, _ = pagedDown.Update(tea.KeyMsg{Type: tea.KeyEnd})
	live := next.(appModel)
	if live.progress.cursor != 9 || !live.progress.autoFollow {
		t.Fatalf("expected End to return to live progress item, got cursor=%d autoFollow=%v", live.progress.cursor, live.progress.autoFollow)
	}
}

func TestExecutionProgressKeepsWaitingForStream(t *testing.T) {
	t.Parallel()

	stream := make(chan tea.Msg, 1)
	stream <- executionFinishedMsg{result: domain.ExecutionResult{}}

	model := newTestAppModel(RouteProgress)
	model.executionStream = stream
	model.progress = progressModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{{
				ID:          "cache",
				Path:        "/tmp/cache",
				DisplayPath: "/tmp/cache",
			}},
		},
	}

	next, cmd := model.Update(executionProgressMsg{
		progress: domain.ExecutionProgress{
			Current: 1,
			Total:   1,
			Phase:   domain.ProgressPhaseRunning,
			Item: domain.Finding{
				ID:          "cache",
				Path:        "/tmp/cache",
				DisplayPath: "/tmp/cache",
			},
		},
	})
	updated := next.(appModel)
	if updated.progress.current == nil || updated.progress.current.Path != "/tmp/cache" {
		t.Fatalf("expected current progress item to update, got %+v", updated.progress.current)
	}
	if cmd == nil {
		t.Fatal("expected execution progress to keep waiting for stream")
	}
	if _, ok := deliverCmd(t, cmd).(executionFinishedMsg); !ok {
		t.Fatalf("expected execution stream wait command to return executionFinishedMsg")
	}
}

func TestResultCanOpenCurrentModuleRecoveryReview(t *testing.T) {
	t.Parallel()

	reviewPlan := domain.ExecutionPlan{
		Command:  "clean",
		DryRun:   false,
		Platform: "darwin",
		Items: []domain.Finding{
			{
				ID:          "failed-logs",
				Path:        "/tmp/logs",
				DisplayPath: "/tmp/logs",
				Status:      domain.StatusPlanned,
				Action:      domain.ActionTrash,
				Category:    domain.CategoryLogs,
				Source:      "Application logs",
			},
			{
				ID:          "protected-browser",
				Path:        "/tmp/browser",
				DisplayPath: "/tmp/browser",
				Status:      domain.StatusPlanned,
				Action:      domain.ActionTrash,
				Category:    domain.CategoryBrowserData,
				Source:      "Chrome code cache",
			},
		},
	}
	model := newTestAppModel(RouteResult)
	model.result = resultModel{
		plan:   reviewPlan,
		cursor: 1,
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{FindingID: "failed-logs", Path: "/tmp/logs", Status: domain.StatusFailed, Message: "permission denied"},
				{FindingID: "protected-browser", Path: "/tmp/browser", Status: domain.StatusProtected, Reason: domain.ProtectionProtectedPath},
			},
		},
	}
	model.resultReturnRoute = RouteHome
	model.callbacks.LoadReviewForPaths = func(paths []string) (domain.ExecutionPlan, error) {
		if len(paths) != 1 || paths[0] != "/tmp/browser" {
			t.Fatalf("unexpected module recovery paths: %+v", paths)
		}
		return reviewPlan, nil
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if cmd == nil {
		t.Fatal("expected module recovery review load command")
	}
	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	review := next.(appModel)
	if review.route != RouteReview {
		t.Fatalf("expected review route, got %s", review.route)
	}
	if review.reviewReturnRoute != RouteResult {
		t.Fatalf("expected result as review return route, got %s", review.reviewReturnRoute)
	}
}

func TestResultCanCycleFilters(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteResult)
	model.result = resultModel{
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{Path: "/tmp/deleted", Status: domain.StatusDeleted},
				{Path: "/tmp/protected", Status: domain.StatusProtected},
				{Path: "/tmp/failed", Status: domain.StatusFailed},
			},
		},
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	issues := next.(appModel)
	if issues.result.filter != resultFilterIssues {
		t.Fatalf("expected issues filter, got %s", issues.result.filter)
	}
	if len(issues.result.visibleIndices()) != 2 {
		t.Fatalf("expected 2 issue items, got %+v", issues.result.visibleIndices())
	}

	next, _ = issues.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	clean := next.(appModel)
	if clean.result.filter != resultFilterClean {
		t.Fatalf("expected clean filter, got %s", clean.result.filter)
	}
	if len(clean.result.visibleIndices()) != 1 {
		t.Fatalf("expected 1 clean item, got %+v", clean.result.visibleIndices())
	}

	next, _ = clean.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	all := next.(appModel)
	if all.result.filter != resultFilterAll {
		t.Fatalf("expected all filter, got %s", all.result.filter)
	}
	if len(all.result.visibleIndices()) != 3 {
		t.Fatalf("expected all items, got %+v", all.result.visibleIndices())
	}
}

func TestExecutionFinishedFocusesIssuesInResult(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteProgress)
	model.progress.plan = domain.ExecutionPlan{
		Command: "clean",
		Items: []domain.Finding{
			{ID: "deleted", Path: "/tmp/deleted"},
			{ID: "protected", Path: "/tmp/protected"},
			{ID: "failed", Path: "/tmp/failed"},
		},
	}
	model.result = resultModel{
		filter: resultFilterClean,
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{FindingID: "old", Path: "/tmp/old", Status: domain.StatusDeleted},
			},
		},
	}

	next, _ := model.Update(executionFinishedMsg{
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{FindingID: "deleted", Path: "/tmp/deleted", Status: domain.StatusDeleted},
				{FindingID: "protected", Path: "/tmp/protected", Status: domain.StatusProtected},
				{FindingID: "failed", Path: "/tmp/failed", Status: domain.StatusFailed},
			},
		},
	})
	result := next.(appModel)
	if result.route != RouteResult {
		t.Fatalf("expected result route, got %s", result.route)
	}
	if result.result.filter != resultFilterIssues {
		t.Fatalf("expected issues filter for blocking result, got %s", result.result.filter)
	}
	selected, ok := result.result.selectedItem()
	if !ok {
		t.Fatal("expected selected result item")
	}
	if selected.Path != "/tmp/protected" {
		t.Fatalf("expected first issue to be selected, got %+v", selected)
	}
}

func TestResultFilterCyclePreservesSelectionPathWhenStillVisible(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteResult)
	model.result = resultModel{
		filter: resultFilterAll,
		cursor: 2,
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{Path: "/tmp/deleted", Status: domain.StatusDeleted},
				{Path: "/tmp/protected", Status: domain.StatusProtected},
				{Path: "/tmp/failed", Status: domain.StatusFailed},
			},
		},
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	filtered := next.(appModel)
	if filtered.result.filter != resultFilterIssues {
		t.Fatalf("expected issues filter, got %s", filtered.result.filter)
	}
	selected, ok := filtered.result.selectedItem()
	if !ok {
		t.Fatal("expected selected issue item")
	}
	if selected.Path != "/tmp/failed" {
		t.Fatalf("expected current issue path to stay selected, got %+v", selected)
	}
	if filtered.result.cursor != 1 {
		t.Fatalf("expected issue-relative cursor to be preserved, got %d", filtered.result.cursor)
	}
}

func TestResultNavigationClearsFlash(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteResult)
	model.result = resultModel{
		cursor: 0,
		flash:  true,
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{Path: "/tmp/deleted", Status: domain.StatusDeleted},
				{Path: "/tmp/failed", Status: domain.StatusFailed},
			},
		},
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	result := next.(appModel)
	if result.result.flash {
		t.Fatal("expected first result navigation to clear flash state")
	}
}

func TestResultRecoveryReviewUsesIssueBatchEvenFromCleanFilter(t *testing.T) {
	t.Parallel()

	reviewPlan := domain.ExecutionPlan{
		Command:  "clean",
		DryRun:   false,
		Platform: "darwin",
		Items: []domain.Finding{
			{
				ID:          "deleted-1",
				Path:        "/tmp/deleted",
				DisplayPath: "/tmp/deleted",
				Status:      domain.StatusPlanned,
				Action:      domain.ActionTrash,
				Category:    domain.CategorySystemClutter,
			},
			{
				ID:          "failed-1",
				Path:        "/tmp/failed",
				DisplayPath: "/tmp/failed",
				Status:      domain.StatusPlanned,
				Action:      domain.ActionTrash,
				Category:    domain.CategoryLogs,
				Source:      "Application logs",
			},
			{
				ID:          "protected-1",
				Path:        "/tmp/protected",
				DisplayPath: "/tmp/protected",
				Status:      domain.StatusPlanned,
				Action:      domain.ActionTrash,
				Category:    domain.CategoryBrowserData,
				Source:      "Chrome code cache",
			},
		},
	}

	model := newTestAppModel(RouteResult)
	model.result = resultModel{
		plan:   reviewPlan,
		filter: resultFilterClean,
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{FindingID: "deleted-1", Path: "/tmp/deleted", Status: domain.StatusDeleted},
				{FindingID: "failed-1", Path: "/tmp/failed", Status: domain.StatusFailed, Message: "permission denied"},
				{FindingID: "protected-1", Path: "/tmp/protected", Status: domain.StatusProtected, Reason: domain.ProtectionProtectedPath},
			},
		},
	}
	model.resultReturnRoute = RouteHome
	model.callbacks.LoadReviewForPaths = func(paths []string) (domain.ExecutionPlan, error) {
		if len(paths) != 2 || paths[0] != "/tmp/failed" || paths[1] != "/tmp/protected" {
			t.Fatalf("unexpected recovery paths: %+v", paths)
		}
		return reviewPlan, nil
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd == nil {
		t.Fatal("expected recovery review load command")
	}
	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	review := next.(appModel)
	if review.route != RouteReview {
		t.Fatalf("expected review route, got %s", review.route)
	}
}

func TestResultRetryReviewUsesFailedItemsOnly(t *testing.T) {
	t.Parallel()

	reviewPlan := domain.ExecutionPlan{
		Command:  "clean",
		DryRun:   false,
		Platform: "darwin",
		Items: []domain.Finding{
			{
				ID:          "failed-1",
				Path:        "/tmp/failed",
				DisplayPath: "/tmp/failed",
				Status:      domain.StatusPlanned,
				Action:      domain.ActionTrash,
				Category:    domain.CategoryLogs,
			},
			{
				ID:          "protected-1",
				Path:        "/tmp/protected",
				DisplayPath: "/tmp/protected",
				Status:      domain.StatusPlanned,
				Action:      domain.ActionTrash,
				Category:    domain.CategoryBrowserData,
			},
		},
	}

	model := newTestAppModel(RouteResult)
	model.result = resultModel{
		plan: reviewPlan,
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{FindingID: "failed-1", Path: "/tmp/failed", Status: domain.StatusFailed, Message: "permission denied"},
				{FindingID: "protected-1", Path: "/tmp/protected", Status: domain.StatusProtected, Reason: domain.ProtectionProtectedPath},
			},
		},
	}
	model.resultReturnRoute = RouteHome
	model.callbacks.LoadReviewForPaths = func(paths []string) (domain.ExecutionPlan, error) {
		if len(paths) != 1 || paths[0] != "/tmp/failed" {
			t.Fatalf("unexpected retry paths: %+v", paths)
		}
		return reviewPlan, nil
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	if cmd == nil {
		t.Fatal("expected retry review load command")
	}
	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	review := next.(appModel)
	if review.route != RouteReview {
		t.Fatalf("expected review route, got %s", review.route)
	}
}

func TestAnalyzeCanCycleFilters(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.analyze = analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Items: []domain.Finding{
				{Path: "/tmp/child", DisplayPath: "/tmp/child", Name: "child", Status: domain.StatusAdvisory, Risk: domain.RiskReview, Category: domain.CategoryDiskUsage},
				{Path: "/tmp/high.log", DisplayPath: "/tmp/high.log", Name: "high.log", Status: domain.StatusAdvisory, Risk: domain.RiskHigh, Category: domain.CategoryLargeFiles},
				{Path: "/tmp/safe.bin", DisplayPath: "/tmp/safe.bin", Name: "safe.bin", Status: domain.StatusAdvisory, Risk: domain.RiskSafe, Category: domain.CategoryLargeFiles},
			},
		},
		staged: map[string]domain.Finding{
			"/tmp/safe.bin": {Path: "/tmp/safe.bin", DisplayPath: "/tmp/safe.bin", Name: "safe.bin", Status: domain.StatusAdvisory, Risk: domain.RiskSafe, Category: domain.CategoryLargeFiles},
		},
		stageOrder: []string{"/tmp/safe.bin"},
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	queued := next.(appModel)
	if queued.analyze.filter != analyzeFilterQueued {
		t.Fatalf("expected queued filter, got %s", queued.analyze.filter)
	}
	if len(queued.analyze.visibleIndices()) != 1 {
		t.Fatalf("expected one queued item, got %+v", queued.analyze.visibleIndices())
	}

	next, _ = queued.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	high := next.(appModel)
	if high.analyze.filter != analyzeFilterHigh {
		t.Fatalf("expected high filter, got %s", high.analyze.filter)
	}
	if len(high.analyze.visibleIndices()) != 2 {
		t.Fatalf("expected two high/review items, got %+v", high.analyze.visibleIndices())
	}

	next, _ = high.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	all := next.(appModel)
	if all.analyze.filter != analyzeFilterAll {
		t.Fatalf("expected all filter, got %s", all.analyze.filter)
	}
	if len(all.analyze.visibleIndices()) != 3 {
		t.Fatalf("expected three visible items, got %+v", all.analyze.visibleIndices())
	}
}

func TestAnalyzeCanStartAndClearSearch(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.setAnalyzePlan(domain.ExecutionPlan{
		Command: "analyze",
		Targets: []string{"/repo"},
		Items: []domain.Finding{
			{
				Name:        "chrome-cache",
				Path:        "/repo/chrome-cache",
				DisplayPath: "/repo/chrome-cache",
				Category:    domain.CategoryDiskUsage,
				Risk:        domain.RiskReview,
				Action:      domain.ActionAdvisory,
				Status:      domain.StatusAdvisory,
			},
			{
				Name:        "slack-cache",
				Path:        "/repo/slack-cache",
				DisplayPath: "/repo/slack-cache",
				Category:    domain.CategoryDiskUsage,
				Risk:        domain.RiskReview,
				Action:      domain.ActionAdvisory,
				Status:      domain.StatusAdvisory,
			},
		},
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	searching := next.(appModel)
	if !searching.analyze.searchActive {
		t.Fatal("expected analyze search to become active")
	}

	next, _ = searching.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s', 'l', 'a', 'c', 'k'}})
	filtered := next.(appModel)
	if got := filtered.analyze.search.Value(); got != "slack" {
		t.Fatalf("expected analyze search value slack, got %q", got)
	}
	if len(filtered.analyze.visibleIndices()) != 1 {
		t.Fatalf("expected one visible analyze item, got %d", len(filtered.analyze.visibleIndices()))
	}

	next, _ = filtered.Update(tea.KeyMsg{Type: tea.KeyEsc})
	cleared := next.(appModel)
	if cleared.analyze.searchActive {
		t.Fatal("expected analyze search to stop on esc")
	}
	if got := cleared.analyze.search.Value(); got != "" {
		t.Fatalf("expected analyze search to clear, got %q", got)
	}
}

func TestAnalyzeCanCycleQueueSort(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	now := time.Now()
	model.analyze = analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Items: []domain.Finding{
				{Path: "/tmp/a", DisplayPath: "/tmp/a", Name: "a", Status: domain.StatusAdvisory, Risk: domain.RiskReview, Category: domain.CategoryDiskUsage, Bytes: 10, LastModified: now.Add(-time.Hour)},
				{Path: "/tmp/b", DisplayPath: "/tmp/b", Name: "b", Status: domain.StatusAdvisory, Risk: domain.RiskReview, Category: domain.CategoryLargeFiles, Bytes: 100, LastModified: now.Add(-48 * time.Hour)},
			},
		},
		staged: map[string]domain.Finding{
			"/tmp/a": {Path: "/tmp/a", DisplayPath: "/tmp/a", Name: "a", Status: domain.StatusAdvisory, Risk: domain.RiskReview, Category: domain.CategoryDiskUsage, Bytes: 10, LastModified: now.Add(-time.Hour)},
			"/tmp/b": {Path: "/tmp/b", DisplayPath: "/tmp/b", Name: "b", Status: domain.StatusAdvisory, Risk: domain.RiskReview, Category: domain.CategoryLargeFiles, Bytes: 100, LastModified: now.Add(-48 * time.Hour)},
		},
		stageOrder: []string{"/tmp/a", "/tmp/b"},
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	sizeSorted := next.(appModel)
	if sizeSorted.analyze.queueSort != analyzeQueueSortSize {
		t.Fatalf("expected size sort, got %s", sizeSorted.analyze.queueSort)
	}
	order := sizeSorted.analyze.sortedStageOrder()
	if len(order) != 2 || order[0] != "/tmp/b" {
		t.Fatalf("expected largest item first, got %+v", order)
	}

	next, _ = sizeSorted.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	ageSorted := next.(appModel)
	if ageSorted.analyze.queueSort != analyzeQueueSortAge {
		t.Fatalf("expected age sort, got %s", ageSorted.analyze.queueSort)
	}
	order = ageSorted.analyze.sortedStageOrder()
	if len(order) != 2 || order[0] != "/tmp/b" {
		t.Fatalf("expected oldest item first, got %+v", order)
	}

	next, _ = ageSorted.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	stagedOrder := next.(appModel)
	if stagedOrder.analyze.queueSort != analyzeQueueSortOrder {
		t.Fatalf("expected staged order sort, got %s", stagedOrder.analyze.queueSort)
	}
	order = stagedOrder.analyze.sortedStageOrder()
	if len(order) != 2 || order[0] != "/tmp/a" {
		t.Fatalf("expected original staged order, got %+v", order)
	}
}

func TestAnalyzeCanFocusQueueAndRemoveSelectedQueuedItem(t *testing.T) {
	t.Parallel()

	now := time.Now()
	model := newTestAppModel(RouteAnalyze)
	model.analyze = analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Items: []domain.Finding{
				{Path: "/tmp/a", DisplayPath: "/tmp/a", Name: "a", Status: domain.StatusAdvisory, Risk: domain.RiskReview, Category: domain.CategoryDiskUsage, Bytes: 10, LastModified: now.Add(-time.Hour)},
				{Path: "/tmp/b", DisplayPath: "/tmp/b", Name: "b", Status: domain.StatusAdvisory, Risk: domain.RiskReview, Category: domain.CategoryLargeFiles, Bytes: 100, LastModified: now.Add(-48 * time.Hour)},
			},
		},
		staged: map[string]domain.Finding{
			"/tmp/a": {Path: "/tmp/a", DisplayPath: "/tmp/a", Name: "a", Status: domain.StatusAdvisory, Risk: domain.RiskReview, Category: domain.CategoryDiskUsage, Bytes: 10, LastModified: now.Add(-time.Hour)},
			"/tmp/b": {Path: "/tmp/b", DisplayPath: "/tmp/b", Name: "b", Status: domain.StatusAdvisory, Risk: domain.RiskReview, Category: domain.CategoryLargeFiles, Bytes: 100, LastModified: now.Add(-48 * time.Hour)},
		},
		stageOrder: []string{"/tmp/a", "/tmp/b"},
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	queueFocused := next.(appModel)
	if queueFocused.analyze.activePane() != analyzePaneQueue {
		t.Fatalf("expected queue pane focus, got %s", queueFocused.analyze.activePane())
	}

	next, _ = queueFocused.Update(tea.KeyMsg{Type: tea.KeyDown})
	secondQueued := next.(appModel)
	if secondQueued.analyze.queueCursor != 1 {
		t.Fatalf("expected queue cursor on second item, got %d", secondQueued.analyze.queueCursor)
	}

	next, _ = secondQueued.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	removed := next.(appModel)
	if len(removed.analyze.stageOrder) != 1 || removed.analyze.stageOrder[0] != "/tmp/a" {
		t.Fatalf("expected second staged item removed, got %+v", removed.analyze.stageOrder)
	}
	if removed.analyze.queueCursor != 0 {
		t.Fatalf("expected queue cursor to clamp to remaining item, got %d", removed.analyze.queueCursor)
	}
	if removed.analyze.activePane() != analyzePaneBrowse {
		t.Fatalf("expected single remaining staged item to return to browse focus, got %s", removed.analyze.activePane())
	}
}

func TestAppRouterAnalyzeRefreshPreservesSelectedPath(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.analyze = analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items: []domain.Finding{
				{Path: "/tmp/cache", DisplayPath: "/tmp/cache", Name: "cache", Status: domain.StatusAdvisory, Category: domain.CategoryDiskUsage},
				{Path: "/tmp/logs", DisplayPath: "/tmp/logs", Name: "logs", Status: domain.StatusAdvisory, Category: domain.CategoryDiskUsage},
			},
		},
		cursor: 1,
		staged: map[string]domain.Finding{},
	}
	model.analyzeReturnRoute = RouteHome
	model.callbacks.LoadAnalyzeTarget = func(target string) (domain.ExecutionPlan, error) {
		if target != "/tmp" {
			t.Fatalf("unexpected refresh target: %s", target)
		}
		return domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items: []domain.Finding{
				{Path: "/tmp/logs", DisplayPath: "/tmp/logs", Name: "logs", Status: domain.StatusAdvisory, Category: domain.CategoryDiskUsage},
				{Path: "/tmp/cache", DisplayPath: "/tmp/cache", Name: "cache", Status: domain.StatusAdvisory, Category: domain.CategoryDiskUsage},
			},
		}, nil
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("expected analyze refresh command")
	}
	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	refreshed := next.(appModel)
	if refreshed.route != RouteAnalyze {
		t.Fatalf("expected analyze route after refresh, got %s", refreshed.route)
	}
	selected, ok := refreshed.analyze.selectedItem()
	if !ok {
		t.Fatal("expected a selected analyze item after refresh")
	}
	if selected.Path != "/tmp/logs" {
		t.Fatalf("expected refresh to preserve selected path /tmp/logs, got %+v", selected)
	}
}

func TestAppRouterAnalyzeRefreshKeepsQueueFocusAndQueuedSelection(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.analyze = analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items: []domain.Finding{
				{Path: "/tmp/a", DisplayPath: "/tmp/a", Name: "a", Status: domain.StatusAdvisory, Category: domain.CategoryDiskUsage},
				{Path: "/tmp/b", DisplayPath: "/tmp/b", Name: "b", Status: domain.StatusAdvisory, Category: domain.CategoryDiskUsage},
			},
		},
		staged: map[string]domain.Finding{
			"/tmp/a": {Path: "/tmp/a", DisplayPath: "/tmp/a", Name: "a", Status: domain.StatusAdvisory, Category: domain.CategoryDiskUsage},
			"/tmp/b": {Path: "/tmp/b", DisplayPath: "/tmp/b", Name: "b", Status: domain.StatusAdvisory, Category: domain.CategoryDiskUsage},
		},
		stageOrder:  []string{"/tmp/a", "/tmp/b"},
		pane:        analyzePaneQueue,
		queueCursor: 1,
	}
	model.analyzeReturnRoute = RouteHome
	model.callbacks.LoadAnalyzeTarget = func(target string) (domain.ExecutionPlan, error) {
		return domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items: []domain.Finding{
				{Path: "/tmp/b", DisplayPath: "/tmp/b", Name: "b", Status: domain.StatusAdvisory, Category: domain.CategoryDiskUsage},
				{Path: "/tmp/a", DisplayPath: "/tmp/a", Name: "a", Status: domain.StatusAdvisory, Category: domain.CategoryDiskUsage},
			},
		}, nil
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("expected analyze refresh command")
	}
	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	refreshed := next.(appModel)
	if refreshed.route != RouteAnalyze {
		t.Fatalf("expected analyze route after refresh, got %s", refreshed.route)
	}
	if refreshed.analyze.activePane() != analyzePaneQueue {
		t.Fatalf("expected queue pane to stay active, got %s", refreshed.analyze.activePane())
	}
	selected, ok := refreshed.analyze.selectedQueuedItem()
	if !ok {
		t.Fatal("expected selected queued item after refresh")
	}
	if selected.Path != "/tmp/b" {
		t.Fatalf("expected queued selection /tmp/b to survive refresh, got %+v", selected)
	}
}

func TestAppRouterAnalyzeRefreshCanBeCancelled(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.analyze = analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items: []domain.Finding{
				{Path: "/tmp/cache", DisplayPath: "/tmp/cache", Name: "cache", Status: domain.StatusAdvisory, Category: domain.CategoryDiskUsage},
			},
		},
		staged: map[string]domain.Finding{},
	}
	model.analyzeReturnRoute = RouteHome
	model.callbacks.LoadAnalyzeTarget = func(target string) (domain.ExecutionPlan, error) {
		return domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items: []domain.Finding{
				{Path: "/tmp/new", DisplayPath: "/tmp/new", Name: "new", Status: domain.StatusAdvisory, Category: domain.CategoryDiskUsage},
			},
		}, nil
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("expected analyze refresh command")
	}
	refreshing := next.(appModel)
	if !refreshing.analyze.loading {
		t.Fatal("expected analyze loading state")
	}

	cancelled, _ := refreshing.Update(tea.KeyMsg{Type: tea.KeyEsc})
	cancelledModel := cancelled.(appModel)
	if cancelledModel.analyze.loading {
		t.Fatal("expected analyze loading to be cancelled")
	}
	if cancelledModel.activePlanRequestID != 0 {
		t.Fatal("expected pending plan request to be cleared")
	}

	staleMsg := deliverCmd(t, cmd)
	staleHandled, _ := cancelledModel.Update(staleMsg)
	final := staleHandled.(appModel)
	if len(final.analyze.plan.Items) != 1 || final.analyze.plan.Items[0].Name != "cache" {
		t.Fatalf("expected stale refresh result to be ignored, got %+v", final.analyze.plan.Items)
	}
}

func TestAnalyzeCanOpenRevealAndTrashSelection(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.analyze = analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items: []domain.Finding{
				{
					Path:        "/tmp/cache",
					DisplayPath: "/tmp/cache",
					Name:        "cache",
					Status:      domain.StatusAdvisory,
					Action:      domain.ActionAdvisory,
					Category:    domain.CategoryDiskUsage,
				},
			},
		},
		staged: map[string]domain.Finding{},
	}
	model.analyzeReturnRoute = RouteHome
	var opened, revealed []string
	var trashed []string
	model.callbacks.OpenPath = func(path string) error {
		opened = append(opened, path)
		return nil
	}
	model.callbacks.RevealPath = func(path string) error {
		revealed = append(revealed, path)
		return nil
	}
	model.callbacks.TrashPaths = func(paths []string) (domain.ExecutionResult, error) {
		trashed = append([]string{}, paths...)
		return domain.ExecutionResult{
			Items: []domain.OperationResult{{
				Path:   "/tmp/cache",
				Status: domain.StatusDeleted,
			}},
		}, nil
	}
	model.callbacks.LoadAnalyzeTarget = func(target string) (domain.ExecutionPlan, error) {
		if target != "/tmp" {
			t.Fatalf("unexpected refresh target: %s", target)
		}
		return domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items: []domain.Finding{
				{
					Path:        "/tmp/logs",
					DisplayPath: "/tmp/logs",
					Name:        "logs",
					Status:      domain.StatusAdvisory,
					Action:      domain.ActionAdvisory,
					Category:    domain.CategoryDiskUsage,
				},
			},
		}, nil
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	openedModel := next.(appModel)
	if len(opened) != 1 || opened[0] != "/tmp/cache" {
		t.Fatalf("expected open callback for selected path, got %+v", opened)
	}

	next, _ = openedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	revealedModel := next.(appModel)
	if len(revealed) != 1 || revealed[0] != "/tmp/cache" {
		t.Fatalf("expected reveal callback for selected path, got %+v", revealed)
	}

	next, cmd := revealedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Fatal("expected analyze trash command")
	}
	next, cmd = next.(appModel).Update(deliverCmd(t, cmd))
	if len(trashed) != 1 || trashed[0] != "/tmp/cache" {
		t.Fatalf("expected trash callback for selected path, got %+v", trashed)
	}
	if cmd == nil {
		t.Fatal("expected analyze refresh command after trash action")
	}
	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	refreshed := next.(appModel)
	if refreshed.route != RouteAnalyze {
		t.Fatalf("expected analyze route after trash refresh, got %s", refreshed.route)
	}
	if len(refreshed.analyze.plan.Items) != 1 || refreshed.analyze.plan.Items[0].Path != "/tmp/logs" {
		t.Fatalf("expected refreshed analyze plan, got %+v", refreshed.analyze.plan.Items)
	}
}

func TestAnalyzeStageStartsReviewPreviewLoad(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.analyze = analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items: []domain.Finding{{
				Path:        "/tmp/cache",
				DisplayPath: "/tmp/cache",
				Name:        "cache",
				Status:      domain.StatusAdvisory,
				Action:      domain.ActionAdvisory,
				Category:    domain.CategoryDiskUsage,
			}},
			Totals: domain.Totals{Bytes: 2 * 1024 * 1024},
		},
		staged: map[string]domain.Finding{},
	}
	model.callbacks.LoadReviewForPaths = func(paths []string) (domain.ExecutionPlan, error) {
		if len(paths) != 1 || paths[0] != "/tmp/cache" {
			t.Fatalf("unexpected review preview paths: %+v", paths)
		}
		return domain.ExecutionPlan{Command: "clean", Targets: []string{"/tmp/cache"}}, nil
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	if cmd == nil {
		t.Fatal("expected analyze stage to start review preview load")
	}
	staged := next.(appModel)
	if !staged.analyze.reviewPreview.loading || staged.analyze.reviewPreview.key != "/tmp/cache" {
		t.Fatalf("expected staged review preview loading, got %+v", staged.analyze.reviewPreview)
	}
}

func TestAnalyzeUsesCachedReviewPreviewForImmediateReview(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.analyze = analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items: []domain.Finding{{
				Path:        "/tmp/cache",
				DisplayPath: "/tmp/cache",
				Name:        "cache",
				Status:      domain.StatusAdvisory,
				Action:      domain.ActionAdvisory,
				Category:    domain.CategoryDiskUsage,
			}},
		},
		staged: map[string]domain.Finding{
			"/tmp/cache": {
				Path:        "/tmp/cache",
				DisplayPath: "/tmp/cache",
				Name:        "cache",
				Status:      domain.StatusAdvisory,
				Action:      domain.ActionAdvisory,
				Category:    domain.CategoryDiskUsage,
			},
		},
		stageOrder: []string{"/tmp/cache"},
	}
	model.analyze.applyReviewPreview("/tmp/cache", domain.ExecutionPlan{
		Command:  "clean",
		DryRun:   true,
		Platform: "darwin",
		Targets:  []string{"/tmp/cache"},
		Items: []domain.Finding{{
			Path:        "/tmp/cache",
			DisplayPath: "/tmp/cache",
			Status:      domain.StatusPlanned,
			Action:      domain.ActionTrash,
			Category:    domain.CategorySystemClutter,
		}},
	}, nil)

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Fatal("expected cached analyze review preview to open immediately")
	}
	review := next.(appModel)
	if review.route != RouteReview {
		t.Fatalf("expected review route, got %s", review.route)
	}
	if len(review.review.plan.Targets) != 1 || review.review.plan.Targets[0] != "/tmp/cache" {
		t.Fatalf("expected cached review preview plan, got %+v", review.review.plan)
	}
}

func TestAnalyzeUsesSelectedItemPreviewForImmediateReview(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.analyze = analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items: []domain.Finding{{
				Path:        "/tmp/cache",
				DisplayPath: "/tmp/cache",
				Name:        "cache",
				Status:      domain.StatusAdvisory,
				Action:      domain.ActionAdvisory,
				Category:    domain.CategoryDiskUsage,
			}},
		},
		staged: map[string]domain.Finding{},
	}
	model.analyze.applyReviewPreview("/tmp/cache", domain.ExecutionPlan{
		Command:  "clean",
		DryRun:   true,
		Platform: "darwin",
		Targets:  []string{"/tmp/cache"},
		Items: []domain.Finding{{
			Path:        "/tmp/cache",
			DisplayPath: "/tmp/cache",
			Status:      domain.StatusPlanned,
			Action:      domain.ActionTrash,
			Category:    domain.CategorySystemClutter,
		}},
	}, nil)

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Fatal("expected cached selected-item review preview to open immediately")
	}
	review := next.(appModel)
	if review.route != RouteReview {
		t.Fatalf("expected review route, got %s", review.route)
	}
	if len(review.review.plan.Targets) != 1 || review.review.plan.Targets[0] != "/tmp/cache" {
		t.Fatalf("expected selected-item review preview plan, got %+v", review.review.plan)
	}
}

func TestAnalyzeTrashRefreshSelectsNextSurvivingItem(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.analyze = analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items: []domain.Finding{
				{Path: "/tmp/a", DisplayPath: "/tmp/a", Name: "a", Status: domain.StatusAdvisory, Action: domain.ActionAdvisory, Category: domain.CategoryDiskUsage},
				{Path: "/tmp/b", DisplayPath: "/tmp/b", Name: "b", Status: domain.StatusAdvisory, Action: domain.ActionAdvisory, Category: domain.CategoryDiskUsage},
				{Path: "/tmp/c", DisplayPath: "/tmp/c", Name: "c", Status: domain.StatusAdvisory, Action: domain.ActionAdvisory, Category: domain.CategoryDiskUsage},
			},
		},
		cursor: 1,
		staged: map[string]domain.Finding{},
	}
	model.analyzeReturnRoute = RouteHome
	model.callbacks.TrashPaths = func(paths []string) (domain.ExecutionResult, error) {
		if len(paths) != 1 || paths[0] != "/tmp/b" {
			t.Fatalf("unexpected trash paths: %+v", paths)
		}
		return domain.ExecutionResult{
			Items: []domain.OperationResult{{Path: "/tmp/b", Status: domain.StatusDeleted}},
		}, nil
	}
	model.callbacks.LoadAnalyzeTarget = func(target string) (domain.ExecutionPlan, error) {
		return domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items: []domain.Finding{
				{Path: "/tmp/a", DisplayPath: "/tmp/a", Name: "a", Status: domain.StatusAdvisory, Action: domain.ActionAdvisory, Category: domain.CategoryDiskUsage},
				{Path: "/tmp/c", DisplayPath: "/tmp/c", Name: "c", Status: domain.StatusAdvisory, Action: domain.ActionAdvisory, Category: domain.CategoryDiskUsage},
			},
		}, nil
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Fatal("expected analyze trash command")
	}
	next, cmd = next.(appModel).Update(deliverCmd(t, cmd))
	if cmd == nil {
		t.Fatal("expected analyze refresh command")
	}
	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	refreshed := next.(appModel)
	selected, ok := refreshed.analyze.selectedItem()
	if !ok {
		t.Fatal("expected a selected analyze item after trash refresh")
	}
	if selected.Path != "/tmp/c" {
		t.Fatalf("expected next surviving item /tmp/c to be selected, got %+v", selected)
	}
}

func TestAnalyzeQueueTrashUsesBatchPaths(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.analyze = analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items: []domain.Finding{
				{Path: "/tmp/a", DisplayPath: "/tmp/a", Name: "a", Status: domain.StatusAdvisory, Action: domain.ActionAdvisory, Category: domain.CategoryDiskUsage},
				{Path: "/tmp/b", DisplayPath: "/tmp/b", Name: "b", Status: domain.StatusAdvisory, Action: domain.ActionAdvisory, Category: domain.CategoryDiskUsage},
			},
		},
		staged: map[string]domain.Finding{
			"/tmp/a": {Path: "/tmp/a", DisplayPath: "/tmp/a", Name: "a", Status: domain.StatusAdvisory, Action: domain.ActionAdvisory, Category: domain.CategoryDiskUsage},
			"/tmp/b": {Path: "/tmp/b", DisplayPath: "/tmp/b", Name: "b", Status: domain.StatusAdvisory, Action: domain.ActionAdvisory, Category: domain.CategoryDiskUsage},
		},
		stageOrder: []string{"/tmp/a", "/tmp/b"},
		filter:     analyzeFilterQueued,
	}
	model.analyzeReturnRoute = RouteHome
	var trashed []string
	model.callbacks.TrashPaths = func(paths []string) (domain.ExecutionResult, error) {
		trashed = append([]string{}, paths...)
		return domain.ExecutionResult{
			Items: []domain.OperationResult{
				{Path: "/tmp/a", Status: domain.StatusDeleted},
				{Path: "/tmp/b", Status: domain.StatusDeleted},
			},
		}, nil
	}
	model.callbacks.LoadAnalyzeTarget = func(target string) (domain.ExecutionPlan, error) {
		return domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items:    nil,
		}, nil
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	queueFocused := next.(appModel)
	if queueFocused.analyze.activePane() != analyzePaneQueue {
		t.Fatalf("expected queue focus, got %s", queueFocused.analyze.activePane())
	}

	next, cmd := queueFocused.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Fatal("expected analyze batch trash command")
	}
	next, cmd = next.(appModel).Update(deliverCmd(t, cmd))
	if len(trashed) != 2 || trashed[0] != "/tmp/a" || trashed[1] != "/tmp/b" {
		t.Fatalf("expected queued paths to be trashed together, got %+v", trashed)
	}
	if cmd == nil {
		t.Fatal("expected refresh command after batch trash")
	}
	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	refreshed := next.(appModel)
	if len(refreshed.analyze.stageOrder) != 0 {
		t.Fatalf("expected staged queue to clear after successful trash, got %+v", refreshed.analyze.stageOrder)
	}
	if refreshed.analyze.filter != analyzeFilterAll {
		t.Fatalf("expected queued-only analyze filter to reset to all, got %s", refreshed.analyze.filter)
	}
}

func TestAppRouterHomeCleanReviewClean(t *testing.T) {
	t.Parallel()

	reviewPlan := domain.ExecutionPlan{
		Command:  "clean",
		DryRun:   false,
		Platform: "darwin",
		Items: []domain.Finding{{
			Path:        "/tmp/cache",
			DisplayPath: "/tmp/cache",
			Status:      domain.StatusPlanned,
			Action:      domain.ActionTrash,
			Category:    domain.CategorySystemClutter,
		}},
	}

	model := newTestAppModel(RouteHome)
	model.applyDashboard(testDashboardData())
	model.callbacks.LoadCleanProfile = func(profile string) (domain.ExecutionPlan, error) {
		if profile != "safe" {
			t.Fatalf("expected safe profile for quick clean, got %s", profile)
		}
		return reviewPlan, nil
	}
	model.home.cursor = findHomeAction(t, model.home.actions, "clean")

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected clean preview warmup command")
	}
	cleanRoute := next.(appModel)
	if cleanRoute.route != RouteClean {
		t.Fatalf("expected clean route, got %s", cleanRoute.route)
	}
	next, _ = cleanRoute.Update(deliverCmd(t, cmd))
	cleanRoute = next.(appModel)

	next, cmd = cleanRoute.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected cached quick clean preview to open review without reload")
	}
	review := next.(appModel)
	if review.route != RouteReview {
		t.Fatalf("expected review route, got %s", review.route)
	}

	next, _ = review.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if next.(appModel).route != RouteClean {
		t.Fatalf("expected clean route after leaving review, got %s", next.(appModel).route)
	}
}

func TestCleanMenuShowsModuleCoverage(t *testing.T) {
	t.Parallel()

	model := menuModel{
		title:    "Clean",
		subtitle: "choose scope",
		hint:     "Quick for routine cleanup, workstation for cache-heavy days, deep for maximum reclaim.",
		actions:  buildCleanActions(),
		width:    132,
		height:   30,
	}
	view := model.View()
	for _, needle := range []string{"CLEAN", "QUICK CLEAN", "WORKSTATION CLEAN", "DEEP RECLAIM", "State", "Next", "Scope", "Temporary files", "Safe clutter"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean menu view, got %s", needle, view)
		}
	}
}

func TestHomeCleanEntryStartsScopePreviewLoad(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	model.applyDashboard(testDashboardData())
	model.callbacks.LoadCleanProfile = func(profile string) (domain.ExecutionPlan, error) {
		if profile != "safe" {
			t.Fatalf("expected safe profile preview, got %s", profile)
		}
		return domain.ExecutionPlan{Command: "clean", DryRun: true}, nil
	}
	model.home.cursor = findHomeAction(t, model.home.actions, "clean")

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected clean entry to start preview load")
	}
	routed := next.(appModel)
	if routed.route != RouteClean {
		t.Fatalf("expected clean route, got %s", routed.route)
	}
	if !routed.clean.preview.loading || routed.clean.preview.key != "safe" {
		t.Fatalf("expected quick clean preview loading, got %+v", routed.clean.preview)
	}
}

func TestCleanPreviewLoadStartsAnimatedBatch(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteClean)
	model.callbacks.LoadCleanProfile = func(profile string) (domain.ExecutionPlan, error) {
		return domain.ExecutionPlan{Command: "clean", DryRun: true}, nil
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected clean load command")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected batched clean load command, got %T", msg)
	}
	foundTick := false
	foundPreview := false
	for _, item := range batch {
		switch item().(type) {
		case uiTickMsg:
			foundTick = true
		case menuPreviewLoadedMsg:
			foundPreview = true
		}
	}
	if !foundTick || !foundPreview {
		t.Fatalf("expected ui tick and preview load in batch, tick=%v preview=%v", foundTick, foundPreview)
	}
	if !next.(appModel).clean.preview.loading || !next.(appModel).cleanFlow.preview.loading {
		t.Fatalf("expected clean preview loading state, got clean=%+v flow=%+v", next.(appModel).clean.preview, next.(appModel).cleanFlow.preview)
	}
}

func TestCleanUsesCachedPreviewPlanForImmediateReview(t *testing.T) {
	t.Parallel()

	cachedPlan := domain.ExecutionPlan{
		Command:  "clean",
		DryRun:   true,
		Platform: "darwin",
		Profile:  "safe",
		Items: []domain.Finding{{
			ID:          "quick-1",
			Name:        "Cache",
			Path:        "/tmp/cache",
			DisplayPath: "/tmp/cache",
			Status:      domain.StatusPlanned,
			Action:      domain.ActionTrash,
			Category:    domain.CategorySystemClutter,
		}},
	}

	model := newTestAppModel(RouteClean)
	model.clean.applyPreview("safe", cachedPlan, nil)
	model.cleanFlow.applyPreview("safe", cachedPlan, nil)

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected cached preview to open review without reloading")
	}
	review := next.(appModel)
	if review.route != RouteReview {
		t.Fatalf("expected review route, got %s", review.route)
	}
	if review.review.plan.Profile != "safe" {
		t.Fatalf("expected cached safe preview plan, got %+v", review.review.plan)
	}
}

func TestCleanRouteUsesUpDownForLedgerScrollDuringActiveScan(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteClean)
	model.width = 132
	model.height = 26
	model.clean.cursor = 1
	model.cleanFlow.cursor = 1
	model.cleanFlow.setPreviewLoading("developer")
	for i := 0; i < 10; i++ {
		model.cleanFlow.applyScanFinding("cache", "Developer cache", domain.Finding{
			ID:          strings.Join([]string{"row", string(rune('A' + i))}, "-"),
			Name:        strings.Join([]string{"Row", string(rune('A' + i))}, " "),
			Path:        strings.Join([]string{"/tmp/row", string(rune('a' + i))}, "-"),
			DisplayPath: strings.Join([]string{"/tmp/row", string(rune('a' + i))}, "-"),
			Category:    domain.CategoryDeveloperCaches,
			Bytes:       int64(i+1) << 20,
		})
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	if cmd != nil {
		t.Fatal("did not expect profile reload while scrolling clean history")
	}
	scrolled := next.(appModel)
	if scrolled.cleanFlow.cursor != 1 || scrolled.clean.cursor != 1 {
		t.Fatalf("expected clean profile selection to stay put while scrolling, got flow=%d clean=%d", scrolled.cleanFlow.cursor, scrolled.clean.cursor)
	}
	if scrolled.cleanFlow.scrollOffset == 0 {
		t.Fatalf("expected clean flow scroll offset to increase, got %+v", scrolled.cleanFlow)
	}
	if !strings.Contains(ansi.Strip(scrolled.View()), "scroll hold") {
		t.Fatalf("expected clean view to advertise held scroll position, got %s", ansi.Strip(scrolled.View()))
	}
}

func TestCleanRouteEndKeyReturnsLedgerToLiveTail(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteClean)
	model.width = 132
	model.height = 26
	model.clean.cursor = 1
	model.cleanFlow.cursor = 1
	model.cleanFlow.setPreviewLoading("developer")
	for i := 0; i < 10; i++ {
		model.cleanFlow.applyScanFinding("cache", "Developer cache", domain.Finding{
			ID:          strings.Join([]string{"row", string(rune('A' + i))}, "-"),
			Name:        strings.Join([]string{"Row", string(rune('A' + i))}, " "),
			Path:        strings.Join([]string{"/tmp/row", string(rune('a' + i))}, "-"),
			DisplayPath: strings.Join([]string{"/tmp/row", string(rune('a' + i))}, "-"),
			Category:    domain.CategoryDeveloperCaches,
			Bytes:       int64(i+1) << 20,
		})
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	scrolled := next.(appModel)
	if scrolled.cleanFlow.scrollOffset == 0 {
		t.Fatalf("expected clean flow scroll offset to increase, got %+v", scrolled.cleanFlow)
	}

	next, _ = scrolled.Update(tea.KeyMsg{Type: tea.KeyEnd})
	live := next.(appModel)
	if live.cleanFlow.scrollOffset != 0 || !live.cleanFlow.autoFollow {
		t.Fatalf("expected End to return clean ledger to live tail, got %+v", live.cleanFlow)
	}
	if strings.Contains(ansi.Strip(live.View()), "scroll hold") {
		t.Fatalf("expected live tail view after End key, got %s", ansi.Strip(live.View()))
	}
}

func TestCleanRouteHomeKeyPinsLedgerToOldestHistory(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteClean)
	model.width = 132
	model.height = 26
	model.clean.cursor = 1
	model.cleanFlow.cursor = 1
	model.cleanFlow.setPreviewLoading("developer")
	for i := 0; i < 10; i++ {
		model.cleanFlow.applyScanFinding("cache", "Developer cache", domain.Finding{
			ID:          strings.Join([]string{"row", string(rune('A' + i))}, "-"),
			Name:        strings.Join([]string{"Row", string(rune('A' + i))}, " "),
			Path:        strings.Join([]string{"/tmp/row", string(rune('a' + i))}, "-"),
			DisplayPath: strings.Join([]string{"/tmp/row", string(rune('a' + i))}, "-"),
			Category:    domain.CategoryDeveloperCaches,
			Bytes:       int64(i+1) << 20,
		})
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyHome})
	oldest := next.(appModel)
	if oldest.cleanFlow.scrollOffset == 0 || oldest.cleanFlow.autoFollow {
		t.Fatalf("expected Home to pin clean ledger to oldest visible history, got %+v", oldest.cleanFlow)
	}
	view := ansi.Strip(oldest.View())
	if !strings.Contains(view, "Row A") {
		t.Fatalf("expected Home key to reveal earliest clean scan rows, got %s", view)
	}
}

func TestCleanRoutePageKeysScrollLedgerHistory(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteClean)
	model.width = 132
	model.height = 26
	model.clean.cursor = 1
	model.cleanFlow.cursor = 1
	model.cleanFlow.setPreviewLoading("developer")
	for i := 0; i < 18; i++ {
		model.cleanFlow.applyScanFinding("cache", "Developer cache", domain.Finding{
			ID:          strings.Join([]string{"row", string(rune('A' + i))}, "-"),
			Name:        strings.Join([]string{"Row", string(rune('A' + i))}, " "),
			Path:        strings.Join([]string{"/tmp/row", string(rune('a' + i))}, "-"),
			DisplayPath: strings.Join([]string{"/tmp/row", string(rune('a' + i))}, "-"),
			Category:    domain.CategoryDeveloperCaches,
			Bytes:       int64(i+1) << 20,
		})
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	pagedUp := next.(appModel)
	if pagedUp.cleanFlow.scrollOffset == 0 || pagedUp.cleanFlow.autoFollow {
		t.Fatalf("expected PageUp to enter held clean history, got %+v", pagedUp.cleanFlow)
	}

	next, _ = pagedUp.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	pagedDown := next.(appModel)
	if pagedDown.cleanFlow.scrollOffset >= pagedUp.cleanFlow.scrollOffset {
		t.Fatalf("expected PageDown to move clean history toward live tail, before=%d after=%d", pagedUp.cleanFlow.scrollOffset, pagedDown.cleanFlow.scrollOffset)
	}
}

func TestUninstallRoutePageKeysScrollLedgerHistory(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteUninstall)
	model.width = 132
	model.height = 26
	entries := make([]domain.AppEntry, 0, 12)
	for i := 0; i < 12; i++ {
		entries = append(entries, domain.AppEntry{
			DisplayName:      fmt.Sprintf("Row %c", 'A'+i),
			UninstallCommand: "/Applications/Example Uninstaller.app",
		})
	}
	model.applyInstalledApps(entries)
	model.uninstall.cursor = len(model.uninstall.filtered) - 1
	model.uninstallFlow.phase = uninstallFlowInventory

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	pagedUp := next.(appModel)
	if pagedUp.uninstall.cursor != len(model.uninstall.filtered)-1 {
		t.Fatalf("expected uninstall selection to stay put while paging history, got %d", pagedUp.uninstall.cursor)
	}
	if pagedUp.uninstallFlow.scrollOffset == 0 || pagedUp.uninstallFlow.autoFollow {
		t.Fatalf("expected PageUp to hold uninstall history, got %+v", pagedUp.uninstallFlow)
	}

	next, _ = pagedUp.Update(tea.KeyMsg{Type: tea.KeyEnd})
	live := next.(appModel)
	if live.uninstallFlow.scrollOffset != 0 || !live.uninstallFlow.autoFollow {
		t.Fatalf("expected End to return uninstall history to live tail, got %+v", live.uninstallFlow)
	}
}

func TestAnalyzeRoutePageKeysScrollLedgerHistory(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.width = 132
	model.height = 26
	model.analyzeFlow.phase = analyzeFlowInspecting
	for i := 0; i < 12; i++ {
		model.analyzeFlow.traceRows = append(model.analyzeFlow.traceRows, analyzeFlowTraceRow{
			FindingID: fmt.Sprintf("row-%c", 'A'+i),
			Path:      fmt.Sprintf("/tmp/row-%c", 'a'+i),
			Label:     fmt.Sprintf("Row %c", 'A'+i),
			Category:  domain.CategoryDiskUsage,
			Bytes:     int64(i+1) << 20,
			State:     "review",
		})
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	pagedUp := next.(appModel)
	if pagedUp.analyzeFlow.scrollOffset == 0 || pagedUp.analyzeFlow.autoFollow {
		t.Fatalf("expected PageUp to hold analyze history, got %+v", pagedUp.analyzeFlow)
	}

	next, _ = pagedUp.Update(tea.KeyMsg{Type: tea.KeyEnd})
	live := next.(appModel)
	if live.analyzeFlow.scrollOffset != 0 || !live.analyzeFlow.autoFollow {
		t.Fatalf("expected End to return analyze history to live tail, got %+v", live.analyzeFlow)
	}
}

func TestPlanLoadStaysInlineBeforeTransitionDelayAndBlocksRouteInput(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	model.width = 132
	model.height = 30
	model.applyDashboard(testDashboardData())
	model.callbacks.LoadAnalyzeHome = func() (domain.ExecutionPlan, error) {
		return domain.ExecutionPlan{Command: "analyze", DryRun: true}, nil
	}
	model.home.cursor = findHomeAction(t, model.home.actions, "analyze")
	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	loading := next.(appModel)
	if !loading.planLoadPending() {
		t.Fatal("expected pending plan load to be active")
	}
	if loading.planLoadActive() {
		t.Fatal("expected transition screen to remain hidden for short loads")
	}
	view := loading.View()
	for _, needle := range []string{"Analyze disk usage", "loading next screen"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in inline loading view, got %s", needle, view)
		}
	}
	for _, needle := range []string{"SIFT / Analyze", "opening analysis", "esc cancels"} {
		if strings.Contains(view, needle) {
			t.Fatalf("did not expect %q before transition delay, got %s", needle, view)
		}
	}

	loading.home.cursor = findHomeAction(t, loading.home.actions, "clean")
	next, _ = loading.Update(tea.KeyMsg{Type: tea.KeyDown})
	blocked := next.(appModel)
	if blocked.home.cursor != findHomeAction(t, loading.home.actions, "clean") {
		t.Fatalf("expected pending load to block route cursor changes, got %d", blocked.home.cursor)
	}
}

func TestPlanLoadShowsTransitionScreenAfterDelay(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	model.width = 132
	model.height = 30
	model.applyDashboard(testDashboardData())
	model.callbacks.LoadAnalyzeHome = func() (domain.ExecutionPlan, error) {
		return domain.ExecutionPlan{Command: "analyze", DryRun: true}, nil
	}
	model.home.cursor = findHomeAction(t, model.home.actions, "analyze")
	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	loading := next.(appModel)
	if !loading.planLoadPending() {
		t.Fatal("expected pending plan load")
	}

	next, _ = loading.Update(planLoadTransitionMsg{requestID: loading.activePlanRequestID})
	transition := next.(appModel)
	if !transition.planLoadActive() {
		t.Fatal("expected delayed transition screen to become active")
	}
	view := transition.View()
	for _, needle := range []string{"SIFT / ANALYZE", "opening analysis", "esc cancels"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in transition view, got %s", needle, view)
		}
	}
}

func TestPlanLoadCanBeCancelledWithoutChangingScreen(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	model.applyDashboard(testDashboardData())
	model.callbacks.LoadAnalyzeHome = func() (domain.ExecutionPlan, error) {
		return domain.ExecutionPlan{Command: "analyze", DryRun: true}, nil
	}
	model.home.cursor = findHomeAction(t, model.home.actions, "analyze")
	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	loading := next.(appModel)
	if !loading.planLoadPending() {
		t.Fatal("expected pending plan load to be active")
	}

	next, _ = loading.Update(tea.KeyMsg{Type: tea.KeyEsc})
	cancelled := next.(appModel)
	if cancelled.route != RouteHome {
		t.Fatalf("expected home route to remain active, got %s", cancelled.route)
	}
	if cancelled.planLoadPending() || cancelled.currentLoadingLabel() != "" {
		t.Fatalf("expected plan load to be cancelled, got pending=%v label=%q", cancelled.planLoadPending(), cancelled.currentLoadingLabel())
	}
	if cancelled.errorMsg != "" {
		t.Fatalf("expected cancellation to avoid error bar, got %q", cancelled.errorMsg)
	}
	if !strings.Contains(cancelled.noticeMsg, "Cancelled scanning files.") {
		t.Fatalf("expected cancellation notice, got %q", cancelled.noticeMsg)
	}
}

func TestPlanLoadDoesNotOpenHelpWhilePending(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	model.applyDashboard(testDashboardData())
	model.callbacks.LoadAnalyzeHome = func() (domain.ExecutionPlan, error) {
		return domain.ExecutionPlan{Command: "analyze", DryRun: true}, nil
	}
	model.home.cursor = findHomeAction(t, model.home.actions, "analyze")
	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	loading := next.(appModel)
	if !loading.planLoadPending() {
		t.Fatal("expected pending plan load to be active")
	}

	next, _ = loading.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if next.(appModel).helpVisible {
		t.Fatal("expected help overlay to stay closed while plan load is pending")
	}
}

func TestAppRouterHomeUninstallLoadsApps(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	model.applyDashboard(testDashboardData())
	model.callbacks.LoadInstalledApps = func() ([]domain.AppEntry, error) {
		return []domain.AppEntry{{DisplayName: "Example", UninstallCommand: "/Applications/Example Uninstaller.app"}}, nil
	}
	model.callbacks.LoadUninstallPlan = func(app string) (domain.ExecutionPlan, error) {
		if app != "Example" {
			t.Fatalf("expected preview for Example, got %s", app)
		}
		return domain.ExecutionPlan{Command: "uninstall", Targets: []string{app}}, nil
	}
	model.home.cursor = findHomeAction(t, model.home.actions, "uninstall")

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected installed apps load command")
	}
	loading := next.(appModel)
	if loading.route != RouteUninstall {
		t.Fatalf("expected uninstall route during inventory load, got %s", loading.route)
	}
	if !loading.uninstall.loading || loading.currentLoadingLabel() != "installed apps" {
		t.Fatalf("expected uninstall loading state, got loading=%v label=%q", loading.uninstall.loading, loading.currentLoadingLabel())
	}
	next, cmd = loading.Update(deliverCmd(t, cmd))
	uninstall := next.(appModel)
	if uninstall.route != RouteUninstall {
		t.Fatalf("expected uninstall route, got %s", uninstall.route)
	}
	item, ok := uninstall.uninstall.selected()
	if !ok || item.Name != "Example" {
		t.Fatalf("expected uninstall selection to include Example, got %+v", uninstall.uninstall.items)
	}
	if cmd == nil || !uninstall.uninstall.preview.loading || uninstall.uninstall.preview.key != uninstallStageKey("Example") {
		t.Fatalf("expected uninstall preview load to start, got cmd=%v preview=%+v", cmd != nil, uninstall.uninstall.preview)
	}
}

func TestAppRouterHomeUninstallUsesCachedInventoryBeforeFreshRefresh(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	model.applyDashboard(testDashboardData())
	model.home.cursor = findHomeAction(t, model.home.actions, "uninstall")

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected installed apps load command")
	}
	loading := next.(appModel)
	if loading.currentLoadingLabel() != "installed apps" || !loading.uninstall.loading {
		t.Fatalf("expected uninstall loading to be set, got loading=%v label=%q", loading.uninstall.loading, loading.currentLoadingLabel())
	}

	cachedAt := time.Now().Add(-2 * time.Hour)
	next, _ = loading.Update(appsLoadedMsg{
		apps: []domain.AppEntry{{
			DisplayName:      "Cached App",
			UninstallCommand: "/Applications/Cached Uninstaller.app",
		}},
		cached:   true,
		loadedAt: cachedAt,
	})
	cached := next.(appModel)
	if cached.route != RouteUninstall {
		t.Fatalf("expected uninstall route after cached inventory, got %s", cached.route)
	}
	item, ok := cached.uninstall.selected()
	if !ok || item.Name != "Cached App" {
		t.Fatalf("expected cached uninstall selection, got %+v", cached.uninstall.items)
	}
	if !strings.Contains(cached.uninstall.message, "Cached app inventory loaded") {
		t.Fatalf("expected cached inventory message, got %q", cached.uninstall.message)
	}
	if cached.uninstall.messageTicks == 0 {
		t.Fatalf("expected cached inventory message to be transient, got ticks=%d", cached.uninstall.messageTicks)
	}

	next, _ = cached.Update(appsLoadedMsg{
		apps: []domain.AppEntry{{
			DisplayName:      "Fresh App",
			UninstallCommand: "/Applications/Fresh Uninstaller.app",
		}},
		cached:   false,
		loadedAt: time.Now(),
	})
	fresh := next.(appModel)
	item, ok = fresh.uninstall.selected()
	if !ok || item.Name != "Fresh App" {
		t.Fatalf("expected fresh uninstall selection to replace cache, got %+v", fresh.uninstall.items)
	}
	if !strings.Contains(fresh.uninstall.message, "Installed app inventory refreshed") {
		t.Fatalf("expected refreshed inventory message, got %q", fresh.uninstall.message)
	}
	if fresh.currentLoadingLabel() != "" || fresh.uninstall.loading {
		t.Fatalf("expected uninstall loading to clear after fresh refresh, got loading=%v label=%q", fresh.uninstall.loading, fresh.currentLoadingLabel())
	}
}

func TestTransientNoticesFadeOnUITick(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	model.noticeMsg = "Opened /tmp/cache"
	model.noticeTicks = 1
	model.uninstall.message = "Queued Example"
	model.uninstall.messageTicks = 1

	next, _ := model.Update(uiTickMsg{})
	ticked := next.(appModel)
	if ticked.noticeMsg != "" || ticked.noticeTicks != 0 {
		t.Fatalf("expected app notice to clear after final tick, got %q (%d)", ticked.noticeMsg, ticked.noticeTicks)
	}
	if ticked.uninstall.message != "" || ticked.uninstall.messageTicks != 0 {
		t.Fatalf("expected uninstall notice to clear after final tick, got %q (%d)", ticked.uninstall.message, ticked.uninstall.messageTicks)
	}
}

func TestReducedMotionSkipsAnimationButStillClearsNotices(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	model.reducedMotion = true
	model.noticeMsg = "Opened /tmp/cache"
	model.noticeTicks = 1
	model.spinnerFrame = 4
	model.livePulse = true

	next, cmd := model.Update(uiTickMsg{})
	ticked := next.(appModel)
	if cmd != nil {
		t.Fatal("expected reduced-motion tick to stop when no transient notices remain")
	}
	if ticked.spinnerFrame != 0 || ticked.livePulse {
		t.Fatalf("expected reduced-motion tick to keep animation static, got frame=%d pulse=%v", ticked.spinnerFrame, ticked.livePulse)
	}
	if ticked.noticeMsg != "" || ticked.noticeTicks != 0 {
		t.Fatalf("expected reduced-motion tick to still clear notices, got %q (%d)", ticked.noticeMsg, ticked.noticeTicks)
	}
}

func TestIdleDashboardRoutesDoNotKeepUITickAlive(t *testing.T) {
	t.Parallel()

	for _, route := range []Route{RouteHome, RouteStatus, RouteDoctor} {
		model := newTestAppModel(route)
		model.clearDashboardLoading()
		if model.wantsUITick() {
			t.Fatalf("expected idle %s route not to keep ui tick alive", route)
		}
	}
}

func TestDashboardLoadingKeepsUITickAlive(t *testing.T) {
	t.Parallel()

	for _, route := range []Route{RouteHome, RouteStatus, RouteDoctor} {
		model := newTestAppModel(route)
		switch route {
		case RouteHome:
			model.setHomeLoading("dashboard")
		case RouteStatus:
			model.setStatusLoading("dashboard")
		case RouteDoctor:
			model.setDoctorLoading("doctor")
		}
		if !model.wantsUITick() {
			t.Fatalf("expected loading %s route to keep ui tick alive", route)
		}
	}
}

func TestWindowSizePropagatesAcrossRoutes(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteAnalyze)
	next, _ := model.Update(tea.WindowSizeMsg{Width: 84, Height: 26})
	sized := next.(appModel)

	if sized.width != 84 || sized.height != 26 {
		t.Fatalf("expected app size to update, got %dx%d", sized.width, sized.height)
	}
	checks := []struct {
		name          string
		width, height int
	}{
		{"home", sized.home.width, sized.home.height},
		{"clean", sized.clean.width, sized.clean.height},
		{"tools", sized.tools.width, sized.tools.height},
		{"protect", sized.protect.width, sized.protect.height},
		{"uninstall", sized.uninstall.width, sized.uninstall.height},
		{"status", sized.status.width, sized.status.height},
		{"doctor", sized.doctor.width, sized.doctor.height},
		{"analyze", sized.analyze.width, sized.analyze.height},
		{"review", sized.review.width, sized.review.height},
		{"preflight", sized.preflight.width, sized.preflight.height},
		{"progress", sized.progress.width, sized.progress.height},
		{"result", sized.result.width, sized.result.height},
	}
	for _, check := range checks {
		if check.width != 84 || check.height != 26 {
			t.Fatalf("expected %s size to update, got %dx%d", check.name, check.width, check.height)
		}
	}
}

func TestCompactFooterAfterResizeUsesCompactBindings(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	model.applyDashboard(testDashboardData())
	next, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	sized := next.(appModel)

	footer := sized.footerContent()
	for _, snippet := range []string{"enter open", "t tools", "? help"} {
		if !strings.Contains(footer, snippet) {
			t.Fatalf("expected compact footer to contain %q, got %q", snippet, footer)
		}
	}
	if !strings.Contains(sized.View(), "SIFT") {
		t.Fatalf("expected resized home view to remain readable, got:\n%s", sized.View())
	}
}

func TestUninstallKeepsCachedInventoryWhenFreshRefreshFails(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteUninstall)
	model.setUninstallLoading("installed apps")

	next, _ := model.Update(appsLoadedMsg{
		apps: []domain.AppEntry{{
			DisplayName:      "Cached App",
			UninstallCommand: "/Applications/Cached Uninstaller.app",
		}},
		cached:   true,
		loadedAt: time.Now().Add(-30 * time.Minute),
	})
	cached := next.(appModel)
	if cached.route != RouteUninstall {
		t.Fatalf("expected uninstall route after cached inventory, got %s", cached.route)
	}

	next, _ = cached.Update(appsLoadedMsg{
		err: errors.New("refresh failed"),
	})
	failed := next.(appModel)
	item, ok := failed.uninstall.selected()
	if !ok || item.Name != "Cached App" {
		t.Fatalf("expected cached uninstall selection to survive refresh failure, got %+v", failed.uninstall.items)
	}
	if failed.errorMsg != "refresh failed" {
		t.Fatalf("expected refresh error to surface, got %q", failed.errorMsg)
	}
	if failed.currentLoadingLabel() != "" || failed.uninstall.loading {
		t.Fatalf("expected uninstall loading to clear after refresh failure, got loading=%v label=%q", failed.uninstall.loading, failed.currentLoadingLabel())
	}
}

func TestNewAppModelSeedsInitialRouteState(t *testing.T) {
	t.Parallel()

	plan := &domain.ExecutionPlan{
		Command:   "clean",
		PlanState: "preview",
		Items: []domain.Finding{{
			ID:     "item-1",
			Path:   "/tmp/cache",
			Status: domain.StatusPlanned,
			Action: domain.ActionTrash,
		}},
	}
	result := &domain.ExecutionResult{
		Items: []domain.OperationResult{{
			FindingID: "item-1",
			Path:      "/tmp/cache",
			Status:    domain.StatusDeleted,
		}},
	}

	model := newAppModel(AppOptions{
		Config:        config.Default(),
		Executable:    true,
		InitialRoute:  RouteReview,
		InitialPlan:   plan,
		InitialResult: result,
	}, AppCallbacks{})

	if model.route != RouteReview {
		t.Fatalf("expected review route, got %s", model.route)
	}
	if model.review.plan.Command != "clean" || len(model.review.plan.Items) != 1 {
		t.Fatalf("expected review plan to be seeded, got %+v", model.review.plan)
	}
	if model.result.result.Items[0].Status != domain.StatusDeleted {
		t.Fatalf("expected initial result to be seeded, got %+v", model.result.result)
	}
	if model.result.plan.Command != "clean" {
		t.Fatalf("expected result plan to follow initial plan, got %+v", model.result.plan)
	}
}

func TestUninstallLateInventoryLoadDoesNotPullUserBackAfterLeaving(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	model.applyDashboard(testDashboardData())
	model.home.cursor = findHomeAction(t, model.home.actions, "uninstall")

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected installed apps load command")
	}
	loading := next.(appModel)
	if loading.route != RouteUninstall {
		t.Fatalf("expected uninstall route during load, got %s", loading.route)
	}

	back, _ := loading.Update(tea.KeyMsg{Type: tea.KeyEsc})
	home := back.(appModel)
	if home.route != RouteHome {
		t.Fatalf("expected home route after leaving uninstall load, got %s", home.route)
	}

	late, _ := home.Update(appsLoadedMsg{
		apps: []domain.AppEntry{{
			DisplayName:      "Late App",
			UninstallCommand: "/Applications/Late Uninstaller.app",
		}},
		requestID: loading.activeInventoryRequestID,
	})
	if late.(appModel).route != RouteHome {
		t.Fatalf("expected late inventory result to be ignored, got %s", late.(appModel).route)
	}
}

func TestUninstallFilterNarrowsResults(t *testing.T) {
	t.Parallel()

	model := newUninstallModel()
	model.setItems([]uninstallItem{
		{Name: "Example App", HasNative: true},
		{Name: "Builder Tool", HasNative: false},
	})
	model.startSearch()
	model.search.SetValue("exam")
	model.applyFilter()

	item, ok := model.selected()
	if !ok || item.Name != "Example App" {
		t.Fatalf("expected filtered uninstall selection, got %+v", item)
	}
	if len(model.filtered) != 1 {
		t.Fatalf("expected one filtered result, got %d", len(model.filtered))
	}
}

func TestUninstallCanStageBatchAndOpenBatchReview(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteUninstall)
	model.applyInstalledApps([]domain.AppEntry{
		{DisplayName: "Example App", UninstallCommand: "/Applications/Example Uninstaller.app"},
		{DisplayName: "Builder Tool"},
	})
	var got []string
	model.callbacks.LoadUninstallBatchPlan = func(apps []string) (domain.ExecutionPlan, error) {
		got = append([]string{}, apps...)
		return domain.ExecutionPlan{
			Command:  "uninstall",
			DryRun:   false,
			Platform: "darwin",
			Targets:  append([]string{}, apps...),
			Items: []domain.Finding{{
				Path:        "/tmp/example",
				DisplayPath: "/tmp/example",
				Status:      domain.StatusPlanned,
				Action:      domain.ActionTrash,
				Category:    domain.CategoryAppLeftovers,
			}},
		}, nil
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	staged := next.(appModel)
	if staged.uninstall.stageCount() != 1 {
		t.Fatalf("expected one staged app, got %+v", staged.uninstall.stageNames())
	}

	next, _ = staged.Update(tea.KeyMsg{Type: tea.KeyDown})
	staged = next.(appModel)
	next, _ = staged.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	staged = next.(appModel)
	if staged.uninstall.stageCount() != 2 {
		t.Fatalf("expected two staged apps, got %+v", staged.uninstall.stageNames())
	}

	next, cmd := staged.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd == nil {
		t.Fatal("expected batch uninstall review load command")
	}
	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	review := next.(appModel)
	if review.route != RouteReview {
		t.Fatalf("expected review route, got %s", review.route)
	}
	if len(got) != 2 || got[0] != "Example App" || got[1] != "Builder Tool" {
		t.Fatalf("expected staged app names in callback order, got %+v", got)
	}
	if len(review.review.plan.Targets) != 2 {
		t.Fatalf("expected batch uninstall targets, got %+v", review.review.plan.Targets)
	}
}

func TestUninstallUsesCachedPreviewPlanForImmediateReview(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteUninstall)
	model.applyInstalledApps([]domain.AppEntry{{DisplayName: "Example App", UninstallCommand: "/Applications/Example Uninstaller.app"}})
	model.uninstall.applyPreview(uninstallStageKey("Example App"), domain.ExecutionPlan{
		Command:  "uninstall",
		DryRun:   false,
		Platform: "darwin",
		Targets:  []string{"Example App"},
		Items: []domain.Finding{{
			Path:        "/tmp/example",
			DisplayPath: "/tmp/example",
			Status:      domain.StatusPlanned,
			Action:      domain.ActionTrash,
			Category:    domain.CategoryAppLeftovers,
		}},
	}, nil)

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected cached uninstall preview to open review without reload")
	}
	review := next.(appModel)
	if review.route != RouteReview {
		t.Fatalf("expected review route, got %s", review.route)
	}
	if len(review.review.plan.Targets) != 1 || review.review.plan.Targets[0] != "Example App" {
		t.Fatalf("expected cached uninstall preview plan, got %+v", review.review.plan)
	}
}

func TestUninstallSetItemsPreservesQueuedAppsAcrossRefresh(t *testing.T) {
	t.Parallel()

	model := newUninstallModel()
	model.setItems([]uninstallItem{
		{Name: "Example App", HasNative: true, SizeLabel: "1 GB"},
		{Name: "Builder Tool", HasNative: false},
	})
	if _, ok, staged := model.toggleSelectedStage(); !ok || !staged {
		t.Fatalf("expected selected app to stage, got ok=%v staged=%v", ok, staged)
	}

	model.setItems([]uninstallItem{
		{Name: "Example App", HasNative: true, SizeLabel: "2 GB"},
		{Name: "Builder Tool", HasNative: false},
	})

	if model.stageCount() != 1 {
		t.Fatalf("expected staged app to survive refresh, got %+v", model.stageNames())
	}
	queued := model.staged[uninstallStageKey("Example App")]
	if queued.SizeLabel != "2 GB" {
		t.Fatalf("expected queued metadata to refresh, got %+v", queued)
	}
}

func TestUninstallSetItemsPreservesSelectedAppAcrossRefresh(t *testing.T) {
	t.Parallel()

	model := newUninstallModel()
	model.setItems([]uninstallItem{
		{Name: "Alpha App", HasNative: true, SizeLabel: "1 GB"},
		{Name: "Beta Tool", HasNative: false, SizeLabel: "512 MB"},
	})
	model.cursor = 1

	model.setItems([]uninstallItem{
		{Name: "Beta Tool", HasNative: false, SizeLabel: "768 MB"},
		{Name: "Alpha App", HasNative: true, SizeLabel: "1 GB"},
	})

	selected, ok := model.selected()
	if !ok {
		t.Fatal("expected uninstall selection after refresh")
	}
	if selected.Name != "Beta Tool" {
		t.Fatalf("expected selected app to stay on Beta Tool, got %+v", selected)
	}
	if selected.SizeLabel != "768 MB" {
		t.Fatalf("expected refreshed metadata for selected app, got %+v", selected)
	}
}

func TestApplyInstalledAppsCarriesSensitiveMetadata(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteUninstall)
	model.applyInstalledApps([]domain.AppEntry{{
		DisplayName:   "Vault",
		BundlePath:    "/Applications/Vault.app",
		ApproxBytes:   1024,
		Sensitive:     true,
		FamilyMatches: []string{"password_managers"},
	}})

	item, ok := model.uninstall.selected()
	if !ok {
		t.Fatal("expected uninstall selection")
	}
	if item.SizeLabel != "1.0 KB" {
		t.Fatalf("expected size label, got %+v", item)
	}
	if !item.Sensitive {
		t.Fatalf("expected sensitive flag, got %+v", item)
	}
	if len(item.FamilyMatches) != 1 || item.FamilyMatches[0] != "password_managers" {
		t.Fatalf("expected family matches, got %+v", item)
	}
}

func TestUninstallFilterMatchesOriginAndLocation(t *testing.T) {
	t.Parallel()

	model := newUninstallModel()
	model.setItems([]uninstallItem{
		{Name: "Example App", HasNative: true, Origin: "homebrew cask", Location: "/Applications/Example.app"},
		{Name: "Builder Tool", HasNative: false, Origin: "user program", Location: "/Users/test/AppData/Builder"},
	})
	model.startSearch()
	model.search.SetValue("brew")
	model.applyFilter()

	item, ok := model.selected()
	if !ok || item.Name != "Example App" {
		t.Fatalf("expected origin-based filter match, got %+v", item)
	}

	model.search.SetValue("appdata")
	model.applyFilter()
	item, ok = model.selected()
	if !ok || item.Name != "Builder Tool" {
		t.Fatalf("expected location-based filter match, got %+v", item)
	}
}

func TestUninstallFilterMatchesFamilyAndFootprint(t *testing.T) {
	t.Parallel()

	model := newUninstallModel()
	model.setItems([]uninstallItem{
		{Name: "Vault", HasNative: true, SizeLabel: "1.0 GB", Sensitive: true, FamilyMatches: []string{"password_managers"}},
		{Name: "Builder Tool", HasNative: false, SizeLabel: "12 MB"},
	})
	model.startSearch()
	model.search.SetValue("password")
	model.applyFilter()

	item, ok := model.selected()
	if !ok || item.Name != "Vault" {
		t.Fatalf("expected family-based filter match, got %+v", item)
	}

	model.search.SetValue("1.0 gb")
	model.applyFilter()
	item, ok = model.selected()
	if !ok || item.Name != "Vault" {
		t.Fatalf("expected footprint-based filter match, got %+v", item)
	}
}

func TestUninstallItemsPreferNativeRecentEntries(t *testing.T) {
	t.Parallel()

	model := newUninstallModel()
	now := time.Now()
	model.setItems([]uninstallItem{
		{Name: "Old Native", HasNative: true, LastSeenAt: now.Add(-48 * time.Hour)},
		{Name: "Recent Native", HasNative: true, LastSeenAt: now.Add(-2 * time.Hour)},
		{Name: "Recent Remnants", HasNative: false, LastSeenAt: now.Add(-time.Hour)},
	})

	if len(model.items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(model.items))
	}
	if model.items[0].Name != "Recent Native" {
		t.Fatalf("expected recent native app first, got %+v", model.items)
	}
	if model.items[1].Name != "Old Native" {
		t.Fatalf("expected older native app second, got %+v", model.items)
	}
	if model.items[2].Name != "Recent Remnants" {
		t.Fatalf("expected remnant-only app last, got %+v", model.items)
	}
}

func TestDashboardTickRefreshesDashboardRoutes(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	called := 0
	model.callbacks.LoadDashboard = func() (DashboardData, error) {
		called++
		return testDashboardData(), nil
	}

	_, cmd := model.Update(dashboardTickMsg{})
	if cmd == nil {
		t.Fatal("expected dashboard tick command")
	}
	if called != 0 {
		t.Fatalf("expected loader to run inside command, got %d", called)
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok || len(batch) == 0 {
		t.Fatalf("expected batched refresh command, got %T", msg)
	}
	loadMsg := batch[0]()
	if _, ok := loadMsg.(dashboardLoadedMsg); !ok {
		t.Fatalf("expected dashboardLoadedMsg from first batch command, got %T", loadMsg)
	}
}

func TestFooterContentIncludesLiveSync(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteStatus)
	model.applyDashboard(DashboardData{
		Report: engine.StatusReport{
			Live: &engine.SystemSnapshot{
				CollectedAt: time.Date(2026, 3, 13, 12, 34, 56, 0, time.UTC).Format(time.RFC3339),
			},
		},
	})

	footer := model.footerContent()
	if !strings.Contains(footer, "updated ") {
		t.Fatalf("unexpected footer content: %s", footer)
	}
}

func TestApplyDashboardComputesStatusRates(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteStatus)
	prevAt := time.Date(2026, 3, 14, 10, 0, 0, 0, time.UTC)
	currAt := prevAt.Add(10 * time.Second)
	model.status.live = &engine.SystemSnapshot{
		CollectedAt:    prevAt.Format(time.RFC3339),
		NetworkRxBytes: 1_000,
		NetworkTxBytes: 2_000,
		DiskIO:         &engine.DiskIOSnapshot{ReadBytes: 3_000, WriteBytes: 4_000},
	}

	model.applyDashboard(DashboardData{
		Report: engine.StatusReport{
			Live: &engine.SystemSnapshot{
				CollectedAt:       currAt.Format(time.RFC3339),
				NetworkRxBytes:    3_000,
				NetworkTxBytes:    5_000,
				DiskIO:            &engine.DiskIOSnapshot{ReadBytes: 7_000, WriteBytes: 9_000},
				CPUPercent:        42,
				MemoryUsedPercent: 64,
				DiskUsedPercent:   51,
				DiskFreeBytes:     8 << 30,
				MemoryUsedBytes:   4 << 30,
				MemoryTotalBytes:  8 << 30,
				HealthScore:       88,
				HealthLabel:       "healthy",
			},
		},
	})

	if model.status.networkRxRate != 200 {
		t.Fatalf("expected rx rate 200, got %f", model.status.networkRxRate)
	}
	if model.status.networkTxRate != 300 {
		t.Fatalf("expected tx rate 300, got %f", model.status.networkTxRate)
	}
	if model.status.diskReadRate != 400 {
		t.Fatalf("expected disk read rate 400, got %f", model.status.diskReadRate)
	}
	if model.status.diskWriteRate != 500 {
		t.Fatalf("expected disk write rate 500, got %f", model.status.diskWriteRate)
	}
	if len(model.status.cpuTrend) != 1 || model.status.cpuTrend[0] != 42 {
		t.Fatalf("expected cpu trend to be recorded, got %+v", model.status.cpuTrend)
	}
	if len(model.status.networkTrend) != 1 || model.status.networkTrend[0] != 500 {
		t.Fatalf("expected network trend to be recorded, got %+v", model.status.networkTrend)
	}
}

func TestAppRouterToolsProtectAddRemove(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	model := newTestAppModel(RouteHome)
	model.cfg = cfg
	model.home.cfg = cfg
	model.applyDashboard(testDashboardData())
	model.callbacks.AddProtectedPath = func(path string) (config.Config, string, error) {
		next, normalized, err := config.AddProtectedPath(cfg, path)
		if err == nil {
			cfg = next
		}
		return cfg, normalized, err
	}
	model.callbacks.RemoveProtectedPath = func(path string) (config.Config, string, bool, error) {
		next, normalized, removed, err := config.RemoveProtectedPath(cfg, path)
		if err == nil {
			cfg = next
		}
		return cfg, normalized, removed, err
	}
	model.callbacks.ExplainProtection = func(path string) domain.ProtectionExplanation {
		return domain.ProtectionExplanation{
			Path:    path,
			State:   domain.ProtectionStateUserProtected,
			Message: "Blocked by a user-configured protected path.",
		}
	}
	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	tools := next.(appModel)
	tools.tools.cursor = findHomeAction(t, tools.tools.actions, "protect")

	next, _ = tools.Update(tea.KeyMsg{Type: tea.KeyEnter})
	protect := next.(appModel)
	if protect.route != RouteProtect {
		t.Fatalf("expected protect route, got %s", protect.route)
	}

	next, _ = protect.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	editing := next.(appModel)
	if !editing.protect.inputActive {
		t.Fatal("expected protect input mode to start")
	}
	protectedPath := filepath.Join(t.TempDir(), "keep")
	next, _ = editing.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(protectedPath)})
	adding, cmd := next.(appModel).Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected add protected path command")
	}
	next, _ = adding.(appModel).Update(deliverCmd(t, cmd))
	afterAdd := next.(appModel)
	if len(afterAdd.protect.paths) != 1 || afterAdd.protect.paths[0] != protectedPath {
		t.Fatalf("expected protected path to be added, got %+v", afterAdd.protect.paths)
	}

	removing, cmd := afterAdd.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Fatal("expected remove protected path command")
	}
	next, _ = removing.(appModel).Update(deliverCmd(t, cmd))
	afterRemove := next.(appModel)
	if len(afterRemove.protect.paths) != 0 {
		t.Fatalf("expected protected path to be removed, got %+v", afterRemove.protect.paths)
	}
}

func TestAppRouterToolsOptimizeReview(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteHome)
	model.applyDashboard(testDashboardData())
	model.callbacks.LoadOptimize = func() (domain.ExecutionPlan, error) {
		return domain.ExecutionPlan{
			Command:  "optimize",
			DryRun:   true,
			Platform: "darwin",
			Items: []domain.Finding{{
				ID:          "maintenance-1",
				Path:        "task-id",
				DisplayPath: "Review startup items",
				Status:      domain.StatusAdvisory,
				Action:      domain.ActionAdvisory,
				Category:    domain.CategoryMaintenance,
			}},
		}, nil
	}
	model.home.cursor = findHomeAction(t, model.home.actions, "optimize")

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected optimize load command")
	}
	next, _ = next.(appModel).Update(deliverCmd(t, cmd))
	review := next.(appModel)
	if review.route != RouteReview {
		t.Fatalf("expected review route for optimize, got %s", review.route)
	}
	if review.review.plan.Command != "optimize" {
		t.Fatalf("expected optimize review plan, got %+v", review.review.plan)
	}
}

func TestAnalyzeReviewExecuteMovesControllerToPermissions(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteReview)
	plan := domain.ExecutionPlan{
		Command:  "clean",
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
	model.setReviewPlan(plan, true)
	model.reviewReturnRoute = RouteAnalyze
	model.analyzeFlow.markReviewReady(plan)

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	routed := next.(appModel)
	if routed.route != RoutePreflight {
		t.Fatalf("expected preflight route, got %s", routed.route)
	}
	if routed.analyzeFlow.phase != analyzeFlowPermissions {
		t.Fatalf("expected analyze flow permissions phase, got %s", routed.analyzeFlow.phase)
	}
}

func TestAnalyzePreflightBackRestoresReviewReadyPhase(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteReview)
	plan := domain.ExecutionPlan{
		Command:  "clean",
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
	model.setReviewPlan(plan, true)
	model.reviewReturnRoute = RouteAnalyze
	model.analyzeFlow.markReviewReady(plan)

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	preflight := next.(appModel)
	next, _ = preflight.Update(tea.KeyMsg{Type: tea.KeyEsc})
	review := next.(appModel)
	if review.route != RouteReview {
		t.Fatalf("expected review route after preflight back, got %s", review.route)
	}
	if review.analyzeFlow.phase != analyzeFlowReviewReady {
		t.Fatalf("expected analyze flow review-ready phase, got %s", review.analyzeFlow.phase)
	}
}

func TestAnalyzeReviewExecuteMovesControllerToReclaiming(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteReview)
	plan := domain.ExecutionPlan{
		Command:  "clean",
		DryRun:   false,
		Platform: "darwin",
		Items: []domain.Finding{{
			ID:          "cache-a",
			Name:        "Chrome Code Cache/js",
			Path:        "/tmp/cache-a",
			DisplayPath: "/tmp/cache-a",
			Status:      domain.StatusPlanned,
			Action:      domain.ActionTrash,
			Category:    domain.CategoryBrowserData,
			Bytes:       48 << 20,
		}},
	}
	model.setReviewPlan(plan, true)
	model.reviewReturnRoute = RouteAnalyze
	model.analyzeFlow.markReviewReady(plan)
	model.callbacks.ExecutePlanWithProgress = func(ctx context.Context, plan domain.ExecutionPlan, emit func(domain.ExecutionProgress)) (domain.ExecutionResult, error) {
		return domain.ExecutionResult{}, nil
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected execution stream command")
	}
	progress := next.(appModel)
	if progress.route != RouteProgress {
		t.Fatalf("expected progress route, got %s", progress.route)
	}
	if progress.analyzeFlow.phase != analyzeFlowReclaiming {
		t.Fatalf("expected analyze flow reclaiming phase, got %s", progress.analyzeFlow.phase)
	}
}

func TestAnalyzeExecutionFinishedMovesControllerToResult(t *testing.T) {
	t.Parallel()

	plan := domain.ExecutionPlan{
		Command:  "clean",
		DryRun:   false,
		Platform: "darwin",
		Totals:   domain.Totals{SafeBytes: 48 << 20, Bytes: 48 << 20, ItemCount: 1},
		Items: []domain.Finding{{
			ID:          "cache-a",
			Name:        "Chrome Code Cache/js",
			Path:        "/tmp/cache-a",
			DisplayPath: "/tmp/cache-a",
			Status:      domain.StatusPlanned,
			Action:      domain.ActionTrash,
			Category:    domain.CategoryBrowserData,
			Bytes:       48 << 20,
		}},
	}
	model := newTestAppModel(RouteProgress)
	model.setProgressPlan(plan, "/tmp/cache-a")
	model.activeExecutionSourceRoute = RouteAnalyze
	model.analyzeFlow.markReclaiming(plan, permissionPreflightModel{})

	next, _ := model.Update(executionFinishedMsg{
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{{
				Path:   "/tmp/cache-a",
				Status: domain.StatusDeleted,
				Bytes:  48 << 20,
			}},
		},
	})
	result := next.(appModel)
	if result.route != RouteResult {
		t.Fatalf("expected result route, got %s", result.route)
	}
	if result.analyzeFlow.phase != analyzeFlowResult {
		t.Fatalf("expected analyze flow result phase, got %s", result.analyzeFlow.phase)
	}
}

func TestAnalyzeExecutionProgressUpdatesControllerTraceRows(t *testing.T) {
	t.Parallel()

	plan := domain.ExecutionPlan{
		Command: "clean",
		DryRun:  false,
		Totals:  domain.Totals{Bytes: 48 << 20, SafeBytes: 48 << 20, ItemCount: 1},
		Items: []domain.Finding{{
			ID:          "cache-a",
			Name:        "Chrome Code Cache/js",
			Path:        "/tmp/cache-a",
			DisplayPath: "/tmp/cache-a",
			Status:      domain.StatusPlanned,
			Action:      domain.ActionTrash,
			Category:    domain.CategoryBrowserData,
			Bytes:       48 << 20,
		}},
	}
	model := newTestAppModel(RouteProgress)
	model.setProgressPlan(plan, "/tmp/cache-a")
	model.activeExecutionSourceRoute = RouteAnalyze
	model.analyzeFlow.markReclaiming(plan, permissionPreflightModel{})

	next, _ := model.Update(executionProgressMsg{
		progress: domain.ExecutionProgress{
			Phase: domain.ProgressPhaseRunning,
			Item:  plan.Items[0],
		},
	})
	progress := next.(appModel)
	if len(progress.analyzeFlow.traceRows) != 1 {
		t.Fatalf("expected analyze trace row seeded from execution progress, got %+v", progress.analyzeFlow.traceRows)
	}
	if progress.analyzeFlow.traceRows[0].State != "reclaiming" {
		t.Fatalf("expected analyze trace row state reclaiming, got %+v", progress.analyzeFlow.traceRows[0])
	}
}

func newTestAppModel(initial Route) appModel {
	return appModel{
		route:                      initial,
		cfg:                        config.Default(),
		executable:                 true,
		hasHome:                    initial == RouteHome,
		keys:                       defaultKeyMap(),
		help:                       newHelpModel(),
		permissionWarmup:           defaultPermissionWarmupCmd,
		permissionKeepalive:        defaultPermissionKeepalive,
		acceptedPermissionProfiles: map[string]struct{}{},
		callbacks: AppCallbacks{
			LoadDashboard:     func() (DashboardData, error) { return testDashboardData(), nil },
			LoadInstalledApps: func() ([]domain.AppEntry, error) { return nil, nil },
			ExplainProtection: func(path string) domain.ProtectionExplanation {
				return domain.ProtectionExplanation{Path: path}
			},
		},
		home: homeModel{
			actions:    buildHomeActions(config.Default()),
			executable: true,
			cfg:        config.Default(),
		},
		clean: menuModel{
			title:    "Clean",
			subtitle: "choose scope",
			actions:  buildCleanActions(),
		},
		cleanFlow: newCleanFlowModel(),
		tools: menuModel{
			title:    "More Tools",
			subtitle: "more tools",
			actions:  buildToolsActions(config.Default()),
		},
		protect:       newProtectModel(nil),
		uninstall:     newUninstallModel(),
		uninstallFlow: newUninstallFlowModel(),
		analyzeFlow:   newAnalyzeFlowModel(),
	}
}

func testDashboardData() DashboardData {
	return DashboardData{
		Report: engine.StatusReport{
			Live: &engine.SystemSnapshot{HealthScore: 88, HealthLabel: "healthy", Platform: "darwin", PlatformVersion: "14.0"},
		},
		Diagnostics: []platform.Diagnostic{{Name: "config", Status: "ok", Message: "ready"}},
	}
}

func findHomeAction(t *testing.T, actions []homeAction, id string) int {
	t.Helper()
	for idx, action := range actions {
		if action.ID == id {
			return idx
		}
	}
	t.Fatalf("home action %s not found", id)
	return 0
}
