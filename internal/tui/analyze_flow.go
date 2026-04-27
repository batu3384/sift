package tui

import (
	"strings"

	"github.com/batu3384/sift/internal/domain"
)

type analyzeFlowPhase string

const (
	analyzeFlowIdle        analyzeFlowPhase = "idle"
	analyzeFlowInspecting  analyzeFlowPhase = "inspecting"
	analyzeFlowReviewReady analyzeFlowPhase = "review_ready"
	analyzeFlowPermissions analyzeFlowPhase = "permissions"
	analyzeFlowReclaiming  analyzeFlowPhase = "reclaiming"
	analyzeFlowResult      analyzeFlowPhase = "result"
)

type analyzeFlowModel struct {
	width         int
	height        int
	scrollOffset  int
	autoFollow    bool
	pulse         bool
	reducedMotion bool
	spinnerFrame  int
	phase         analyzeFlowPhase
	preflight     permissionPreflightModel
	result        domain.ExecutionResult
	hasResult     bool
	traceRows     []analyzeFlowTraceRow
}

type analyzeFlowTraceRow struct {
	FindingID string
	Path      string
	Label     string
	Category  domain.Category
	Bytes     int64
	State     string
}

func newAnalyzeFlowModel() analyzeFlowModel {
	return analyzeFlowModel{phase: analyzeFlowIdle, autoFollow: true}
}

func (m *analyzeFlowModel) markInspecting(plan domain.ExecutionPlan) {
	if strings.TrimSpace(plan.Command) != "analyze" {
		return
	}
	m.preflight = permissionPreflightModel{}
	m.result = domain.ExecutionResult{}
	m.hasResult = false
	m.traceRows = nil
	m.scrollOffset = 0
	m.autoFollow = true
	m.phase = analyzeFlowInspecting
}

func (m *analyzeFlowModel) applyReviewPreview(base analyzeBrowserModel) {
	m.preflight = permissionPreflightModel{}
	m.result = domain.ExecutionResult{}
	m.hasResult = false
	if _, ok := base.reviewPreviewPlan(); ok {
		m.replaceTraceRowsForPlan(base.reviewPreview.plan, "review")
		m.scrollOffset = 0
		m.autoFollow = true
		m.phase = analyzeFlowReviewReady
		return
	}
	m.traceRows = nil
	m.scrollOffset = 0
	m.autoFollow = true
	m.phase = analyzeFlowInspecting
}

func (m *analyzeFlowModel) markReviewReady(plan domain.ExecutionPlan) {
	if strings.TrimSpace(plan.Command) == "" {
		return
	}
	m.preflight = permissionPreflightModel{}
	m.replaceTraceRowsForPlan(plan, "review")
	m.scrollOffset = 0
	m.autoFollow = true
	m.phase = analyzeFlowReviewReady
}

func (m *analyzeFlowModel) markPermissions(preflight permissionPreflightModel) {
	m.preflight = preflight
	if len(m.traceRows) == 0 {
		m.replaceTraceRowsForPlan(preflight.plan, "review")
	} else {
		m.freezeTraceRows("review")
	}
	m.scrollOffset = 0
	m.autoFollow = true
	m.phase = analyzeFlowPermissions
}

func (m *analyzeFlowModel) markReclaiming(plan domain.ExecutionPlan, preflight permissionPreflightModel) {
	if strings.TrimSpace(plan.Command) == "" {
		return
	}
	m.preflight = preflight
	m.replaceTraceRowsForPlan(plan, "queued")
	m.scrollOffset = 0
	m.autoFollow = true
	m.phase = analyzeFlowReclaiming
}

func (m *analyzeFlowModel) markResult(plan domain.ExecutionPlan, result domain.ExecutionResult) {
	if strings.TrimSpace(plan.Command) == "" {
		return
	}
	m.preflight = permissionPreflightModel{}
	m.result = result
	m.hasResult = true
	if len(m.traceRows) == 0 {
		m.replaceTraceRowsForPlan(plan, "queued")
	}
	for _, item := range result.Items {
		m.applyOperationResult(item, false)
	}
	m.scrollOffset = 0
	m.autoFollow = true
	m.phase = analyzeFlowResult
}

func (m *analyzeFlowModel) applyExecutionProgress(progress domain.ExecutionProgress) {
	if strings.TrimSpace(progress.Item.Path) == "" && strings.TrimSpace(progress.Result.Path) == "" {
		return
	}
	if len(m.traceRows) == 0 {
		m.ensureTraceRowForFinding(progress.Item)
	}
	state := analyzeFlowProgressState(progress)
	m.freezeTraceRows("queued")
	m.updateTraceRowState(progress.Item, progress.Result, state, true)
	if progress.Result.Status != "" {
		m.applyOperationResult(progress.Result, true)
	}
	if m.autoFollow {
		m.scrollOffset = 0
	}
	if m.phase != analyzeFlowResult {
		m.phase = analyzeFlowReclaiming
	}
}

func (m *analyzeFlowModel) replaceTraceRowsForPlan(plan domain.ExecutionPlan, state string) {
	if len(plan.Items) == 0 {
		m.traceRows = nil
		return
	}
	rows := make([]analyzeFlowTraceRow, 0, len(plan.Items))
	for _, item := range plan.Items {
		rows = append(rows, analyzeFlowTraceRow{
			FindingID: strings.TrimSpace(item.ID),
			Path:      strings.TrimSpace(item.Path),
			Label:     analyzeQueueFocusLabel(item),
			Category:  item.Category,
			Bytes:     item.Bytes,
			State:     state,
		})
	}
	m.traceRows = rows
}

func (m *analyzeFlowModel) freezeTraceRows(state string) {
	for idx := range m.traceRows {
		switch m.traceRows[idx].State {
		case "review", "queued", "reclaiming", "verifying":
			m.traceRows[idx].State = state
		}
	}
}

func (m *analyzeFlowModel) ensureTraceRowForFinding(item domain.Finding) {
	if strings.TrimSpace(item.Path) == "" && strings.TrimSpace(item.ID) == "" {
		return
	}
	if _, ok := m.traceRowIndexForFinding(item, domain.OperationResult{}); ok {
		return
	}
	m.traceRows = append(m.traceRows, analyzeFlowTraceRow{
		FindingID: strings.TrimSpace(item.ID),
		Path:      strings.TrimSpace(item.Path),
		Label:     analyzeQueueFocusLabel(item),
		Category:  item.Category,
		Bytes:     item.Bytes,
		State:     "queued",
	})
}

func (m *analyzeFlowModel) traceRowIndexForFinding(item domain.Finding, result domain.OperationResult) (int, bool) {
	id := strings.TrimSpace(item.ID)
	if id == "" {
		id = strings.TrimSpace(result.FindingID)
	}
	path := strings.TrimSpace(item.Path)
	if path == "" {
		path = strings.TrimSpace(result.Path)
	}
	for idx := range m.traceRows {
		switch {
		case id != "" && strings.TrimSpace(m.traceRows[idx].FindingID) == id:
			return idx, true
		case path != "" && strings.TrimSpace(m.traceRows[idx].Path) == path:
			return idx, true
		}
	}
	return 0, false
}

func (m *analyzeFlowModel) updateTraceRowState(item domain.Finding, result domain.OperationResult, state string, moveToEnd bool) {
	idx, ok := m.traceRowIndexForFinding(item, result)
	if !ok {
		m.ensureTraceRowForFinding(item)
		idx, ok = m.traceRowIndexForFinding(item, result)
		if !ok {
			return
		}
	}
	row := m.traceRows[idx]
	if id := strings.TrimSpace(item.ID); id != "" {
		row.FindingID = id
	} else if id := strings.TrimSpace(result.FindingID); id != "" {
		row.FindingID = id
	}
	if path := strings.TrimSpace(item.Path); path != "" {
		row.Path = path
	} else if path := strings.TrimSpace(result.Path); path != "" {
		row.Path = path
	}
	if label := analyzeQueueFocusLabel(item); strings.TrimSpace(label) != "" {
		row.Label = label
	}
	if item.Category != "" {
		row.Category = item.Category
	}
	if item.Bytes > 0 {
		row.Bytes = item.Bytes
	} else if result.Bytes > 0 {
		row.Bytes = result.Bytes
	}
	row.State = state
	if !moveToEnd || idx == len(m.traceRows)-1 {
		m.traceRows[idx] = row
		return
	}
	copy(m.traceRows[idx:], m.traceRows[idx+1:])
	m.traceRows[len(m.traceRows)-1] = row
}

func (m *analyzeFlowModel) applyOperationResult(result domain.OperationResult, moveToEnd bool) {
	item := domain.Finding{ID: strings.TrimSpace(result.FindingID), Path: strings.TrimSpace(result.Path)}
	m.updateTraceRowState(item, result, analyzeFlowResultState(result.Status), moveToEnd)
}

func (m analyzeFlowModel) currentTraceRow() (analyzeFlowTraceRow, bool) {
	for idx := len(m.traceRows) - 1; idx >= 0; idx-- {
		switch m.traceRows[idx].State {
		case "review", "queued", "reclaiming", "verifying":
			return m.traceRows[idx], true
		}
	}
	if len(m.traceRows) == 0 {
		return analyzeFlowTraceRow{}, false
	}
	return m.traceRows[len(m.traceRows)-1], true
}

func analyzeFlowResultState(status domain.FindingStatus) string {
	switch status {
	case domain.StatusDeleted, domain.StatusCompleted:
		return "settled"
	case domain.StatusProtected:
		return "protected"
	case domain.StatusFailed:
		return "failed"
	case domain.StatusSkipped, domain.StatusAdvisory:
		return "skipped"
	default:
		return "queued"
	}
}

func analyzeFlowProgressState(progress domain.ExecutionProgress) string {
	if progress.Result.Status != "" {
		return analyzeFlowResultState(progress.Result.Status)
	}
	switch progress.Phase {
	case domain.ProgressPhaseRunning:
		return "reclaiming"
	case domain.ProgressPhaseVerifying:
		return "verifying"
	case domain.ProgressPhaseFinished:
		return analyzeFlowResultState(progress.Result.Status)
	default:
		return "queued"
	}
}

func (m analyzeFlowModel) wantsAnimation() bool {
	if m.reducedMotion {
		return false
	}
	switch m.phase {
	case analyzeFlowInspecting, analyzeFlowReclaiming:
		return true
	default:
		return false
	}
}

func (m analyzeFlowModel) shouldUseLedgerScroll() bool {
	if len(m.traceRows) == 0 {
		return false
	}
	switch m.phase {
	case analyzeFlowInspecting, analyzeFlowReviewReady, analyzeFlowPermissions, analyzeFlowReclaiming, analyzeFlowResult:
		return true
	default:
		return false
	}
}

func (m *analyzeFlowModel) scrollLedgerUp(lines int) bool {
	if !m.shouldUseLedgerScroll() || lines <= 0 {
		return false
	}
	m.autoFollow = false
	m.scrollOffset += lines
	return true
}

func (m *analyzeFlowModel) scrollLedgerDown(lines int) bool {
	if !m.shouldUseLedgerScroll() || lines <= 0 {
		return false
	}
	if m.scrollOffset <= lines {
		m.scrollOffset = 0
		m.autoFollow = true
		return true
	}
	m.scrollOffset -= lines
	return true
}

func (m analyzeFlowModel) ledgerScrollPageSize() int {
	width, height := effectiveSize(m.width, m.height)
	lines := bodyLineBudget(height, 15, 7)
	if width < 118 || height < 28 {
		lines = max(lines/2, 6)
	}
	return max(lines-2, 6)
}

func (m *analyzeFlowModel) scrollLedgerToLatest() {
	m.scrollOffset = 0
	m.autoFollow = true
}

func (m *analyzeFlowModel) scrollLedgerToOldest() bool {
	if !m.shouldUseLedgerScroll() {
		return false
	}
	m.autoFollow = false
	m.scrollOffset = max(len(m.traceRows)*4, 12)
	return true
}
