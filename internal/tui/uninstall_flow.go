package tui

import (
	"strings"

	"github.com/batu3384/sift/internal/domain"
)

type uninstallFlowPhase string

const (
	uninstallFlowIdle        uninstallFlowPhase = "idle"
	uninstallFlowInventory   uninstallFlowPhase = "inventory"
	uninstallFlowReviewReady uninstallFlowPhase = "review_ready"
	uninstallFlowPermissions uninstallFlowPhase = "permissions"
	uninstallFlowRemoving    uninstallFlowPhase = "removing"
	uninstallFlowResult      uninstallFlowPhase = "result"
)

type uninstallFlowModel struct {
	width     int
	height    int
	scrollOffset int
	autoFollow   bool
	pulse     bool
	reducedMotion bool
	spinnerFrame  int
	phase     uninstallFlowPhase
	preview   menuPreviewState
	preflight permissionPreflightModel
	result    domain.ExecutionResult
	hasResult bool
}

func newUninstallFlowModel() uninstallFlowModel {
	return uninstallFlowModel{phase: uninstallFlowIdle, autoFollow: true}
}

func (m *uninstallFlowModel) setPreviewLoading(key string) {
	loading := strings.TrimSpace(key) != ""
	m.preview = menuPreviewState{key: key, loading: loading}
	m.preflight = permissionPreflightModel{}
	m.result = domain.ExecutionResult{}
	m.hasResult = false
	m.scrollOffset = 0
	m.autoFollow = true
	if loading {
		m.phase = uninstallFlowInventory
		return
	}
	m.phase = uninstallFlowIdle
}

func (m *uninstallFlowModel) applyPreview(key string, plan domain.ExecutionPlan, err error) {
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
	m.scrollOffset = 0
	m.autoFollow = true
	m.phase = uninstallFlowReviewReady
}

func (m *uninstallFlowModel) markReviewReady(plan domain.ExecutionPlan) {
	m.syncPlan(plan)
	m.preflight = permissionPreflightModel{}
	m.scrollOffset = 0
	m.autoFollow = true
	m.phase = uninstallFlowReviewReady
}

func (m *uninstallFlowModel) markPermissions(preflight permissionPreflightModel) {
	m.syncPlan(preflight.plan)
	m.preflight = preflight
	m.scrollOffset = 0
	m.autoFollow = true
	m.phase = uninstallFlowPermissions
}

func (m *uninstallFlowModel) markRemoving(plan domain.ExecutionPlan, preflight permissionPreflightModel) {
	m.syncPlan(plan)
	m.preflight = preflight
	m.scrollOffset = 0
	m.autoFollow = true
	m.phase = uninstallFlowRemoving
}

func (m *uninstallFlowModel) markResult(plan domain.ExecutionPlan, result domain.ExecutionResult) {
	m.syncPlan(plan)
	m.preflight = permissionPreflightModel{}
	m.result = result
	m.hasResult = true
	m.scrollOffset = 0
	m.autoFollow = true
	m.phase = uninstallFlowResult
}

func (m *uninstallFlowModel) syncPlan(plan domain.ExecutionPlan) {
	if strings.TrimSpace(plan.Command) != "uninstall" {
		return
	}
	key := strings.TrimSpace(strings.Join(plan.Targets, ","))
	if key != "" {
		m.preview.key = key
	}
	m.preview.plan = plan
	m.preview.loaded = len(plan.Items) > 0 || strings.TrimSpace(plan.Command) != ""
	m.preview.loading = false
	m.preview.err = ""
}

func (m uninstallFlowModel) wantsAnimation() bool {
	if m.reducedMotion {
		return false
	}
	switch m.phase {
	case uninstallFlowInventory, uninstallFlowRemoving:
		return true
	default:
		return false
	}
}

func (m uninstallFlowModel) shouldUseLedgerScroll(base uninstallModel) bool {
	if len(base.filtered) == 0 {
		return false
	}
	switch m.phase {
	case uninstallFlowInventory, uninstallFlowReviewReady, uninstallFlowPermissions, uninstallFlowRemoving, uninstallFlowResult:
		return true
	default:
		return false
	}
}

func (m *uninstallFlowModel) scrollLedgerUp(base uninstallModel, lines int) bool {
	if !m.shouldUseLedgerScroll(base) || lines <= 0 {
		return false
	}
	m.autoFollow = false
	m.scrollOffset += lines
	return true
}

func (m *uninstallFlowModel) scrollLedgerDown(base uninstallModel, lines int) bool {
	if !m.shouldUseLedgerScroll(base) || lines <= 0 {
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

func (m uninstallFlowModel) ledgerScrollPageSize() int {
	width, height := effectiveSize(m.width, m.height)
	lines := bodyLineBudget(height, 15, 7)
	if width < 118 || height < 28 {
		lines = max(lines/2, 6)
	}
	return max(lines-2, 6)
}

func (m *uninstallFlowModel) scrollLedgerToLatest() {
	m.scrollOffset = 0
	m.autoFollow = true
}

func (m *uninstallFlowModel) scrollLedgerToOldest(base uninstallModel) bool {
	if !m.shouldUseLedgerScroll(base) {
		return false
	}
	m.autoFollow = false
	m.scrollOffset = max(len(base.filtered)*4, 12)
	return true
}
