package tui

import (
	"strings"

	"github.com/batu3384/sift/internal/domain"
)

type cleanFlowPhase string

const (
	cleanFlowIdle        cleanFlowPhase = "idle"
	cleanFlowScanning    cleanFlowPhase = "scanning"
	cleanFlowReviewReady cleanFlowPhase = "review_ready"
	cleanFlowPermissions cleanFlowPhase = "permissions"
	cleanFlowReclaiming  cleanFlowPhase = "reclaiming"
	cleanFlowResult      cleanFlowPhase = "result"
)

type cleanFlowModel struct {
	title         string
	subtitle      string
	actions       []homeAction
	cursor        int
	scrollOffset  int
	autoFollow    bool
	width         int
	height        int
	hint          string
	pulse         bool
	reducedMotion bool
	spinnerFrame  int
	phase         cleanFlowPhase
	preview       menuPreviewState
	preflight     permissionPreflightModel
	result        domain.ExecutionResult
	hasResult     bool
	scanRows      []cleanFlowScanRow
}

type cleanFlowScanRow struct {
	FindingID string
	RuleID    string
	Lane      string
	Label     string
	Path      string
	Items     int
	Bytes     int64
	State     string
}

func newCleanFlowModel() cleanFlowModel {
	return cleanFlowModel{
		title:    "Clean",
		subtitle: "forge sweep rail",
		hint:     "Scan first, review before reclaim.",
		actions:  buildCleanActions(),
		autoFollow: true,
		phase:    cleanFlowIdle,
	}
}

func (m cleanFlowModel) selectedAction() (homeAction, bool) {
	if m.cursor < 0 || m.cursor >= len(m.actions) {
		return homeAction{}, false
	}
	return m.actions[m.cursor], true
}

func (m *cleanFlowModel) setPreviewLoading(key string) {
	loading := strings.TrimSpace(key) != ""
	m.preview = menuPreviewState{key: key, loading: loading}
	m.preflight = permissionPreflightModel{}
	m.result = domain.ExecutionResult{}
	m.hasResult = false
	m.scanRows = nil
	m.scrollOffset = 0
	m.autoFollow = true
	if loading {
		m.phase = cleanFlowScanning
		return
	}
	m.phase = cleanFlowIdle
}

func (m *cleanFlowModel) applyPreview(key string, plan domain.ExecutionPlan, err error) {
	preview := menuPreviewState{key: key}
	if err != nil {
		preview.err = err.Error()
		m.preview = preview
		return
	}
	preview.plan = plan
	preview.loaded = true
	m.preview = preview
	m.preflight = permissionPreflightModel{}
	m.result = domain.ExecutionResult{}
	m.hasResult = false
	m.freezeScanRows("ready")
	if m.autoFollow {
		m.scrollOffset = 0
	}
	m.phase = cleanFlowReviewReady
}

func (m cleanFlowModel) previewPlanForSelected() (domain.ExecutionPlan, bool) {
	action, ok := m.selectedAction()
	if !ok {
		return domain.ExecutionPlan{}, false
	}
	if action.ProfileKey == "" || strings.TrimSpace(action.ProfileKey) != strings.TrimSpace(m.preview.key) || !m.preview.loaded {
		return domain.ExecutionPlan{}, false
	}
	return m.preview.plan, true
}

func (m *cleanFlowModel) markReviewReady(plan domain.ExecutionPlan) {
	m.syncPlan(plan)
	m.preflight = permissionPreflightModel{}
	m.freezeScanRows("ready")
	if m.autoFollow {
		m.scrollOffset = 0
	}
	m.phase = cleanFlowReviewReady
}

func (m *cleanFlowModel) markPermissions(preflight permissionPreflightModel) {
	m.syncPlan(preflight.plan)
	m.preflight = preflight
	m.freezeScanRows("ready")
	if m.autoFollow {
		m.scrollOffset = 0
	}
	m.phase = cleanFlowPermissions
}

func (m *cleanFlowModel) markReclaiming(plan domain.ExecutionPlan, preflight permissionPreflightModel) {
	m.syncPlan(plan)
	m.preflight = preflight
	m.ensureScanRowsForPlan(plan)
	m.freezePendingRows("queued")
	if m.autoFollow {
		m.scrollOffset = 0
	}
	m.phase = cleanFlowReclaiming
}

func (m *cleanFlowModel) markResult(plan domain.ExecutionPlan, result domain.ExecutionResult) {
	m.syncPlan(plan)
	m.preflight = permissionPreflightModel{}
	m.result = result
	m.hasResult = true
	m.ensureScanRowsForPlan(plan)
	for _, op := range result.Items {
		m.applyOperationResult(op, false)
	}
	if m.autoFollow {
		m.scrollOffset = 0
	}
	m.phase = cleanFlowResult
}

func (m *cleanFlowModel) applyScanProgress(ruleID string, ruleName string, itemsFound int, bytesFound int64) {
	if strings.TrimSpace(ruleName) == "" {
		return
	}
	for idx := range m.scanRows {
		if m.scanRows[idx].State == "scanning" {
			m.scanRows[idx].State = "settled"
		}
	}
	action, _ := m.selectedAction()
	row := cleanFlowScanRow{
		RuleID: strings.TrimSpace(ruleID),
		Lane:   cleanFlowScanLaneLabel(action, ruleID, ruleName),
		Label:  strings.TrimSpace(ruleName),
		Items:  itemsFound,
		Bytes:  bytesFound,
		State:  "scanning",
	}
	for idx := range m.scanRows {
		if m.scanRows[idx].RuleID == row.RuleID && row.RuleID != "" {
			m.scanRows[idx] = row
			if m.autoFollow {
				m.scrollOffset = 0
			}
			m.phase = cleanFlowScanning
			return
		}
	}
	m.scanRows = append(m.scanRows, row)
	if len(m.scanRows) > 72 {
		m.scanRows = m.scanRows[len(m.scanRows)-72:]
	}
	if m.autoFollow {
		m.scrollOffset = 0
	}
	m.phase = cleanFlowScanning
}

func (m *cleanFlowModel) applyScanFinding(ruleID string, ruleName string, item domain.Finding) {
	for idx := range m.scanRows {
		if m.scanRows[idx].State == "scanning" {
			m.scanRows[idx].State = "settled"
		}
	}
	action, _ := m.selectedAction()
	row := cleanFlowScanRow{
		FindingID: strings.TrimSpace(item.ID),
		RuleID:    strings.TrimSpace(ruleID),
		Lane:      cleanFlowScanLaneLabel(action, ruleID, ruleName),
		Label:     cleanFlowRowLabel(item),
		Path:      strings.TrimSpace(item.Path),
		Items:     1,
		Bytes:     item.Bytes,
		State:     "scanning",
	}
	if row.Lane == "" {
		row.Lane = cleanFlowLaneLabel(item)
	}
	m.scanRows = append(m.scanRows, row)
	if len(m.scanRows) > 72 {
		m.scanRows = m.scanRows[len(m.scanRows)-72:]
	}
	if m.autoFollow {
		m.scrollOffset = 0
	}
	m.phase = cleanFlowScanning
}

func (m *cleanFlowModel) applyExecutionProgress(progress domain.ExecutionProgress) {
	if strings.TrimSpace(progress.Item.Path) == "" && strings.TrimSpace(progress.Result.Path) == "" {
		return
	}
	m.ensureScanRowForFinding(progress.Item)
	state := cleanFlowProgressState(progress)
	m.freezePendingRows("queued")
	m.updateScanRowState(progress.Item, progress.Result, state, true)
	if progress.Result.Status != "" {
		m.applyOperationResult(progress.Result, true)
	}
	if m.autoFollow {
		m.scrollOffset = 0
	}
	if m.phase != cleanFlowResult {
		m.phase = cleanFlowReclaiming
	}
}

func (m *cleanFlowModel) syncPlan(plan domain.ExecutionPlan) {
	if strings.TrimSpace(plan.Command) != "clean" {
		return
	}
	if strings.TrimSpace(plan.Profile) != "" {
		m.preview.key = strings.TrimSpace(plan.Profile)
	}
	m.preview.plan = plan
	m.preview.loaded = len(plan.Items) > 0 || strings.TrimSpace(plan.Command) != ""
	m.preview.loading = false
	m.preview.err = ""
}

func (m *cleanFlowModel) freezeScanRows(activeState string) {
	state := strings.TrimSpace(activeState)
	if state == "" {
		state = "settled"
	}
	for idx := range m.scanRows {
		if m.scanRows[idx].State == "scanning" {
			m.scanRows[idx].State = state
		}
	}
}

func (m *cleanFlowModel) freezePendingRows(state string) {
	for idx := range m.scanRows {
		switch m.scanRows[idx].State {
		case "scanning", "ready", "reclaiming", "verifying":
			m.scanRows[idx].State = state
		}
	}
}

func (m *cleanFlowModel) ensureScanRowsForPlan(plan domain.ExecutionPlan) {
	for _, item := range plan.Items {
		m.ensureScanRowForFinding(item)
	}
}

func (m *cleanFlowModel) ensureScanRowForFinding(item domain.Finding) {
	if strings.TrimSpace(item.Path) == "" && strings.TrimSpace(item.ID) == "" {
		return
	}
	if _, ok := m.scanRowIndexForFinding(item); ok {
		return
	}
	row := cleanFlowScanRow{
		FindingID: strings.TrimSpace(item.ID),
		RuleID:    strings.TrimSpace(item.RuleID),
		Lane:      cleanFlowLaneLabel(item),
		Label:     cleanFlowRowLabel(item),
		Path:      strings.TrimSpace(item.Path),
		Items:     1,
		Bytes:     item.Bytes,
		State:     "queued",
	}
	m.scanRows = append(m.scanRows, row)
	if len(m.scanRows) > 72 {
		m.scanRows = m.scanRows[len(m.scanRows)-72:]
	}
}

func (m cleanFlowModel) shouldUseLedgerScroll() bool {
	if len(m.scanRows) == 0 {
		return false
	}
	switch m.phase {
	case cleanFlowScanning, cleanFlowReviewReady, cleanFlowPermissions, cleanFlowReclaiming, cleanFlowResult:
		return true
	default:
		return false
	}
}

func (m *cleanFlowModel) scrollLedgerUp(lines int) bool {
	if !m.shouldUseLedgerScroll() || lines <= 0 {
		return false
	}
	m.autoFollow = false
	m.scrollOffset += lines
	return true
}

func (m *cleanFlowModel) scrollLedgerDown(lines int) bool {
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

func (m cleanFlowModel) ledgerScrollPageSize() int {
	width, height := effectiveSize(m.width, m.height)
	lines := bodyLineBudget(height, 16, 6)
	if width < 118 || height < 28 {
		lines = max(lines/2, 6)
	}
	return max(lines-2, 6)
}

func (m *cleanFlowModel) scrollLedgerToLatest() {
	m.scrollOffset = 0
	m.autoFollow = true
}

func (m *cleanFlowModel) scrollLedgerToOldest() bool {
	if !m.shouldUseLedgerScroll() || len(m.scanRows) == 0 {
		return false
	}
	m.autoFollow = false
	m.scrollOffset = max(len(m.scanRows)*4, 12)
	return true
}

func (m *cleanFlowModel) scanRowIndexForFinding(item domain.Finding) (int, bool) {
	id := strings.TrimSpace(item.ID)
	path := strings.TrimSpace(item.Path)
	for idx := range m.scanRows {
		switch {
		case id != "" && strings.TrimSpace(m.scanRows[idx].FindingID) == id:
			return idx, true
		case path != "" && strings.TrimSpace(m.scanRows[idx].Path) == path:
			return idx, true
		}
	}
	return 0, false
}

func (m *cleanFlowModel) updateScanRowState(item domain.Finding, result domain.OperationResult, state string, moveToEnd bool) {
	idx, ok := m.scanRowIndexForFinding(item)
	if !ok {
		if strings.TrimSpace(item.Path) == "" && strings.TrimSpace(result.Path) == "" {
			return
		}
		m.ensureScanRowForFinding(item)
		idx, ok = m.scanRowIndexForFinding(item)
		if !ok {
			return
		}
	}
	row := m.scanRows[idx]
	if strings.TrimSpace(item.ID) != "" {
		row.FindingID = strings.TrimSpace(item.ID)
	}
	if strings.TrimSpace(item.RuleID) != "" {
		row.RuleID = strings.TrimSpace(item.RuleID)
	}
	if label := strings.TrimSpace(item.Name); label != "" {
		row.Label = label
	} else if label := strings.TrimSpace(item.DisplayPath); label != "" {
		row.Label = label
	} else if row.Label == "" {
		row.Label = cleanFlowRowLabel(item)
	}
	if item.Category != "" {
		if lane := cleanFlowLaneLabel(item); lane != "" {
			row.Lane = lane
		}
	} else if row.Lane == "" {
		if lane := cleanFlowLaneLabel(item); lane != "" {
			row.Lane = lane
		}
	}
	if path := strings.TrimSpace(item.Path); path != "" {
		row.Path = path
	} else if path := strings.TrimSpace(result.Path); path != "" {
		row.Path = path
	}
	if item.Bytes > 0 {
		row.Bytes = item.Bytes
	} else if result.Bytes > 0 {
		row.Bytes = result.Bytes
	}
	row.Items = 1
	row.State = state
	if !moveToEnd || idx == len(m.scanRows)-1 {
		m.scanRows[idx] = row
		return
	}
	copy(m.scanRows[idx:], m.scanRows[idx+1:])
	m.scanRows[len(m.scanRows)-1] = row
}

func (m *cleanFlowModel) applyOperationResult(result domain.OperationResult, moveToEnd bool) {
	item := domain.Finding{ID: strings.TrimSpace(result.FindingID), Path: strings.TrimSpace(result.Path)}
	m.updateScanRowState(item, result, cleanFlowResultState(result.Status), moveToEnd)
}

func cleanFlowResultState(status domain.FindingStatus) string {
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

func cleanFlowProgressState(progress domain.ExecutionProgress) string {
	if progress.Result.Status != "" {
		return cleanFlowResultState(progress.Result.Status)
	}
	switch progress.Phase {
	case domain.ProgressPhaseRunning:
		return "reclaiming"
	case domain.ProgressPhaseVerifying:
		return "verifying"
	case domain.ProgressPhaseFinished:
		return cleanFlowResultState(progress.Result.Status)
	default:
		return "queued"
	}
}

func (m cleanFlowModel) wantsAnimation() bool {
	if m.reducedMotion {
		return false
	}
	switch m.phase {
	case cleanFlowScanning, cleanFlowReclaiming:
		return true
	default:
		return false
	}
}
