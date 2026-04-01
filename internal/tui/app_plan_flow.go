package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/batu3384/sift/internal/domain"
)

func (m appModel) executeCurrentReview() (tea.Model, tea.Cmd) {
	effectivePlan := m.review.effectivePlan()
	focusPath := ""
	if item, ok := m.review.selectedItem(); ok {
		focusPath = strings.TrimSpace(item.Path)
	}
	preflight := buildPermissionPreflight(effectivePlan, focusPath)
	if preflight.required() {
		if m.hasAcceptedPermissionProfile(preflight) {
			m.helpVisible = false
			m.errorMsg = ""
			m.clearNotice()
			preflight.width = m.width
			preflight.height = m.height
			m.preflight = preflight
			if preflight.needsAdmin && m.permissionWarmup != nil {
				m.loadingLabel = "warm admin access"
				return m, batchWithUITick(m.permissionWarmup(preflight))
			}
			return m.executePreparedPlan(effectivePlan, focusPath, preflight)
		}
		m.helpVisible = false
		m.errorMsg = ""
		m.clearNotice()
		preflight.width = m.width
		preflight.height = m.height
		m.preflight = preflight
		m.route = RoutePreflight
		return m, nil
	}
	return m.executePreparedPlan(effectivePlan, focusPath, preflight)
}

func (m appModel) executePreparedPreflight() (tea.Model, tea.Cmd) {
	return m.executePreparedPlan(m.preflight.plan, m.preflight.focusPath, m.preflight)
}

func (m appModel) executePreparedPlan(plan domain.ExecutionPlan, focusPath string, preflight permissionPreflightModel) (tea.Model, tea.Cmd) {
	m.helpVisible = false
	m.preflight = permissionPreflightModel{}
	if m.callbacks.ExecutePlanWithProgress == nil {
		m.loadingLabel = "execution"
		return m, batchWithUITick(executePlanCmd(m.callbacks.ExecutePlan, plan))
	}
	m.loadingLabel = ""
	m.errorMsg = ""
	m.route = RouteProgress
	m.setProgressPlan(plan, focusPath)
	stream := make(chan tea.Msg, 16)
	execCtx, cancel := context.WithCancel(context.Background())
	m.executionStream = stream
	m.executionCancel = cancel
	if m.permissionKeepalive != nil {
		m.permissionKeepalive(execCtx, preflight)
	}
	go func(plan domain.ExecutionPlan, executor func(context.Context, domain.ExecutionPlan, func(domain.ExecutionProgress)) (domain.ExecutionResult, error), out chan<- tea.Msg, execCtx context.Context) {
		result, err := executor(execCtx, plan, func(progress domain.ExecutionProgress) {
			out <- executionProgressMsg{progress: progress}
		})
		out <- executionFinishedMsg{result: result, err: err}
		close(out)
	}(plan, m.callbacks.ExecutePlanWithProgress, stream, execCtx)
	return m, batchWithUITick(waitForExecutionStreamCmd(stream))
}

func (m *appModel) noteAcceptedPermissionProfile(preflight permissionPreflightModel) {
	signature := preflight.profileSignature()
	if signature == "" {
		return
	}
	if m.acceptedPermissionProfiles == nil {
		m.acceptedPermissionProfiles = map[string]struct{}{}
	}
	m.acceptedPermissionProfiles[signature] = struct{}{}
}

func (m appModel) hasAcceptedPermissionProfile(preflight permissionPreflightModel) bool {
	signature := preflight.profileSignature()
	if signature == "" || len(m.acceptedPermissionProfiles) == 0 {
		return false
	}
	_, ok := m.acceptedPermissionProfiles[signature]
	return ok
}

func (m appModel) startPlanLoad(label string, target Route, reviewReturn Route, resultReturn Route, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.loadingLabel = label
	m.helpVisible = false
	m.nextPlanRequestID++
	m.activePlanRequestID = m.nextPlanRequestID
	m.planLoadTransitionVisible = false
	m.pendingTargetRoute = target
	m.pendingReviewReturn = reviewReturn
	m.pendingResultReturn = resultReturn
	return m, tea.Batch(
		tagPlanLoadCmd(cmd, m.activePlanRequestID),
		planLoadTransitionCmd(m.activePlanRequestID),
		uiTickCmd(),
	)
}

func (m appModel) startAnalyzeReload(label string, preserveCursor bool) (tea.Model, tea.Cmd) {
	if m.analyze.loading {
		return m, nil
	}
	target := analyzePrimaryTarget(m.analyze.plan)
	m.analyze.loading = true
	// Show target in loading label if available
	if target != "" && label == "analyze" {
		m.loadingLabel = "analyze: " + target
	} else {
		m.loadingLabel = label
	}
	m.pendingAnalyzePushHistory = false
	m.pendingAnalyzePreserveCursor = preserveCursor
	m.pendingAnalyzePane = m.analyze.activePane()
	m.pendingAnalyzeQueuePath = ""
	if m.pendingAnalyzePane == analyzePaneQueue {
		if item, ok := m.analyze.selectedQueuedItem(); ok {
			m.pendingAnalyzeQueuePath = strings.TrimSpace(item.Path)
		}
	}
	if preserveCursor {
		m.pendingAnalyzeSelectionPath = m.analyze.currentSelectionPath()
	} else if strings.TrimSpace(m.pendingAnalyzeSelectionPath) == "" {
		m.pendingAnalyzeSelectionPath = ""
	}
	if target == "" {
		return m.startPlanLoad(m.loadingLabel, RouteAnalyze, m.analyzeReturnRoute, RouteHomeOrExit(m), loadPlanCmd(m.callbacks.LoadAnalyzeHome))
	}
	return m.startPlanLoad(m.loadingLabel, RouteAnalyze, m.analyzeReturnRoute, RouteHomeOrExit(m), loadPlanCmd(func() (domain.ExecutionPlan, error) {
		return m.callbacks.LoadAnalyzeTarget(target)
	}))
}

func analyzeResultDeletedPaths(result domain.ExecutionResult) []string {
	paths := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		if item.Status != domain.StatusDeleted && item.Status != domain.StatusCompleted {
			continue
		}
		path := strings.TrimSpace(item.Path)
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

func (m *appModel) setAnalyzePlan(plan domain.ExecutionPlan) {
	m.analyze = analyzeBrowserModel{
		plan:          plan,
		search:        newAnalyzeSearchInput(),
		staged:        map[string]domain.Finding{},
		previewLoader: m.callbacks.LoadAnalyzePreviews,
		previewCache:  map[string]analyzeDirectoryPreview{},
		stageOrder:    nil,
		width:         m.width,
		height:        m.height,
	}
	m.analyze.syncPreviewWindow()
}

func (m *appModel) setReviewPlan(plan domain.ExecutionPlan, requiresDecision bool) {
	m.review = planModel{
		plan:             plan,
		excluded:         map[string]bool{},
		requiresDecision: requiresDecision,
		width:            m.width,
		height:           m.height,
	}
}

func (m *appModel) setProgressPlan(plan domain.ExecutionPlan, focusPath string) {
	progress := progressModel{
		plan:       plan,
		width:      m.width,
		height:     m.height,
		pulse:      m.livePulse,
		autoFollow: true,
	}
	if idx, ok := progress.cursorForPath(focusPath); ok {
		progress.cursor = idx
		item := progress.plan.Items[idx]
		progress.current = &item
	}
	m.progress = progress
}

func (m *appModel) cancelPendingPlanLoad() {
	m.loadingLabel = ""
	m.planLoadTransitionVisible = false
	m.pendingAnalyzePushHistory = false
	m.pendingAnalyzePreserveCursor = false
	m.pendingAnalyzeSelectionPath = ""
	m.pendingAnalyzePane = ""
	m.pendingAnalyzeQueuePath = ""
	m.pendingTargetRoute = ""
	m.pendingReviewReturn = ""
	m.pendingResultReturn = ""
	m.activePlanRequestID = 0
	m.analyze.loading = false
}

func analyzePrimaryTarget(plan domain.ExecutionPlan) string {
	if len(plan.Targets) == 0 {
		return ""
	}
	return strings.TrimSpace(plan.Targets[0])
}

func (m appModel) analyzeActionPaths() []string {
	if m.analyze.activePane() == analyzePaneQueue {
		return m.analyze.stagedPaths()
	}
	item, ok := m.analyze.selectedItem()
	if !ok || strings.TrimSpace(item.Path) == "" {
		return nil
	}
	return []string{item.Path}
}

func analyzeTrashLabel(paths []string) string {
	if len(paths) <= 1 {
		return "move to trash"
	}
	return fmt.Sprintf("trash %d paths", len(paths))
}

func analyzeActionSummary(result domain.ExecutionResult) string {
	completed, deleted, failed, skipped, protected := countResultStatuses(result)
	parts := make([]string, 0, 4)
	if deleted > 0 || completed > 0 {
		count := deleted + completed
		parts = append(parts, fmt.Sprintf("Trashed %d %s", count, pl(count, "path", "paths")))
	}
	if protected > 0 {
		parts = append(parts, fmt.Sprintf("%d blocked", protected))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}
	if len(parts) == 0 {
		return "No analyze actions were applied."
	}
	return strings.Join(parts, " • ")
}
