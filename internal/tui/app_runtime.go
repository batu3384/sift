package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/batu3384/sift/internal/domain"
)

func (m *appModel) applyWindowSize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	m.syncMotionSettings()
	m.home.width, m.home.height = msg.Width, msg.Height
	m.clean.width, m.clean.height = msg.Width, msg.Height
	m.cleanFlow.width, m.cleanFlow.height = msg.Width, msg.Height
	m.tools.width, m.tools.height = msg.Width, msg.Height
	m.protect.width, m.protect.height = msg.Width, msg.Height
	m.uninstall.width, m.uninstall.height = msg.Width, msg.Height
	m.uninstallFlow.width, m.uninstallFlow.height = msg.Width, msg.Height
	m.status.width, m.status.height = msg.Width, msg.Height
	m.doctor.width, m.doctor.height = msg.Width, msg.Height
	m.analyze.width, m.analyze.height = msg.Width, msg.Height
	m.analyzeFlow.width, m.analyzeFlow.height = msg.Width, msg.Height
	m.review.width, m.review.height = msg.Width, msg.Height
	m.preflight.width, m.preflight.height = msg.Width, msg.Height
	m.progress.width, m.progress.height = msg.Width, msg.Height
	m.result.width, m.result.height = msg.Width, msg.Height
}

func (m appModel) handleDashboardLoaded(msg dashboardLoadedMsg) (tea.Model, tea.Cmd) {
	m.clearDashboardLoading()
	if msg.err != nil {
		m.clearNotice()
		m.errorMsg = msg.err.Error()
		return m, nil
	}
	m.errorMsg = ""
	m.clearNotice()
	m.applyDashboard(msg.data)
	return m, nil
}

func (m appModel) handleDashboardTick() (tea.Model, tea.Cmd) {
	if m.dashboardLoadingActive() || m.loadingLabel != "" {
		return m, dashboardTickCmd()
	}
	if m.route == RouteHome || m.route == RouteStatus || m.route == RouteDoctor {
		return m, tea.Batch(loadDashboardCmd(m.callbacks.LoadDashboard), dashboardTickCmd())
	}
	return m, dashboardTickCmd()
}

func (m appModel) handleUITick() (tea.Model, tea.Cmd) {
	if m.reducedMotion {
		m.livePulse = false
		m.spinnerFrame = 0
	} else {
		m.livePulse = !m.livePulse
		m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
	}
	m.tickNotices()
	m.syncMotionSettings()
	if m.wantsUITick() {
		return m, uiTickCmd()
	}
	return m, nil
}

func (m appModel) handleScanProgress(msg scanProgressMsg) (tea.Model, tea.Cmd) {
	if msg.route == RouteClean {
		if msg.requestID == 0 || msg.requestID != m.activeCleanPreviewRequestID {
			return m, nil
		}
		m.cleanFlow.applyScanProgress(msg.ruleID, msg.ruleName, msg.itemsFound, msg.bytesFound)
	}
	if msg.ruleName != "" {
		m.loadingLabel = "scanning: " + msg.ruleName
	}
	if m.cleanPreviewStream != nil {
		return m, waitForPreviewStreamCmd(m.cleanPreviewStream)
	}
	return m, nil
}

func (m appModel) handleMenuPreviewLoaded(msg menuPreviewLoadedMsg) (tea.Model, tea.Cmd) {
	switch msg.route {
	case RouteClean:
		if msg.requestID == 0 || msg.requestID != m.activeCleanPreviewRequestID {
			return m, nil
		}
		m.activeCleanPreviewRequestID = 0
		m.cleanPreviewStream = nil
		m.clean.applyPreview(msg.key, msg.plan, msg.err)
		m.cleanFlow.applyPreview(msg.key, msg.plan, msg.err)
	case RouteUninstall:
		if msg.requestID == 0 || msg.requestID != m.activeUninstallPreviewRequestID {
			return m, nil
		}
		m.activeUninstallPreviewRequestID = 0
		m.uninstall.applyPreview(msg.key, msg.plan, msg.err)
		m.uninstallFlow.applyPreview(msg.key, msg.plan, msg.err)
	case RouteAnalyze:
		if msg.requestID == 0 || msg.requestID != m.activeAnalyzePreviewRequestID {
			return m, nil
		}
		m.activeAnalyzePreviewRequestID = 0
		m.analyze.applyReviewPreview(msg.key, msg.plan, msg.err)
		m.analyzeFlow.applyReviewPreview(m.analyze)
	}
	return m, nil
}

func (m appModel) handleScanFinding(msg scanFindingMsg) (tea.Model, tea.Cmd) {
	if msg.route == RouteClean {
		if msg.requestID == 0 || msg.requestID != m.activeCleanPreviewRequestID {
			return m, nil
		}
		m.cleanFlow.applyScanFinding(msg.ruleID, msg.ruleName, msg.item)
	}
	if m.cleanPreviewStream != nil {
		return m, waitForPreviewStreamCmd(m.cleanPreviewStream)
	}
	return m, nil
}

func (m appModel) handleResultLoaded(msg resultLoadedMsg) (tea.Model, tea.Cmd) {
	m.loadingLabel = ""
	if msg.err != nil {
		m.clearNotice()
		m.errorMsg = msg.err.Error()
		return m, nil
	}
	m.errorMsg = ""
	m.clearNotice()
	m.result = buildResultModel(msg.plan, msg.result, m.result, m.width, m.height)
	if m.activeExecutionSourceRoute == RouteAnalyze {
		m.analyzeFlow.markResult(msg.plan, msg.result)
	} else if msg.plan.Command == "clean" {
		m.cleanFlow.markResult(msg.plan, msg.result)
	} else if msg.plan.Command == "uninstall" {
		m.uninstallFlow.markResult(msg.plan, msg.result)
	}
	m.activeExecutionSourceRoute = ""
	m.route = RouteResult
	return m, nil
}

func (m appModel) handleExecutionProgress(msg executionProgressMsg) (tea.Model, tea.Cmd) {
	m.errorMsg = ""
	m.progress.apply(msg.progress)
	if m.activeExecutionSourceRoute == RouteAnalyze {
		m.analyzeFlow.applyExecutionProgress(msg.progress)
	} else if m.progress.plan.Command == "clean" {
		m.cleanFlow.applyExecutionProgress(msg.progress)
	}
	if m.executionStream != nil {
		return m, waitForExecutionStreamCmd(m.executionStream)
	}
	return m, nil
}

func (m appModel) handlePermissionWarmupFinished(msg permissionWarmupFinishedMsg) (tea.Model, tea.Cmd) {
	m.loadingLabel = ""
	if msg.err != nil {
		m.clearNotice()
		message := strings.TrimSpace(msg.err.Error())
		if message == "" {
			message = "Admin access was not confirmed."
		}
		m.errorMsg = message
		return m, nil
	}
	m.errorMsg = ""
	m.clearNotice()
	return m.executePreparedPreflight()
}

func (m appModel) handleExecutionFinished(msg executionFinishedMsg) (tea.Model, tea.Cmd) {
	m.executionStream = nil
	m.executionCancel = nil
	if msg.err != nil {
		if errorsIsCanceled(msg.err) {
			m.errorMsg = ""
			m.setNotice("Execution cancelled. Partial results preserved.")
		} else {
			m.clearNotice()
			m.errorMsg = msg.err.Error()
		}
	} else {
		m.errorMsg = ""
		m.clearNotice()
	}
	m.result = buildResultModel(m.progress.plan, msg.result, m.result, m.width, m.height)
	if m.activeExecutionSourceRoute == RouteAnalyze {
		m.analyzeFlow.markResult(m.progress.plan, msg.result)
	} else if m.progress.plan.Command == "clean" {
		m.cleanFlow.markResult(m.progress.plan, msg.result)
	} else if m.progress.plan.Command == "uninstall" {
		m.uninstallFlow.markResult(m.progress.plan, msg.result)
	}
	m.activeExecutionSourceRoute = ""
	m.route = RouteResult
	return m, nil
}

func (m appModel) handleAppsLoaded(msg appsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.requestID != 0 && msg.requestID != m.activeInventoryRequestID {
		return m, nil
	}
	if msg.err != nil {
		if msg.cached {
			return m, nil
		}
		m.activeInventoryRequestID = 0
		m.setUninstallLoading("")
		m.clearNotice()
		m.errorMsg = msg.err.Error()
		return m, nil
	}
	if !msg.cached {
		m.activeInventoryRequestID = 0
		m.setUninstallLoading("")
	}
	m.errorMsg = ""
	m.clearNotice()
	m.applyInstalledApps(msg.apps)
	m.uninstall.setMessage(uninstallInventoryMessage(msg.loadedAt, msg.cached), routeMessageLongTicks)
	m.route = RouteUninstall
	return m, m.startUninstallPreviewLoad()
}

func (m appModel) handleProtectAdded(msg protectAddedMsg) (tea.Model, tea.Cmd) {
	m.loadingLabel = ""
	if msg.err != nil {
		m.clearNotice()
		m.errorMsg = msg.err.Error()
		return m, nil
	}
	m.errorMsg = ""
	m.clearNotice()
	m.protect.inputActive = false
	m.protect.input.Blur()
	m.applyConfig(msg.cfg)
	m.protect.syncPaths(msg.cfg.ProtectedPaths)
	m.protect.syncScopes(msg.cfg.CommandExcludes)
	m.protect.setMessage("Protected path added: "+msg.path, routeMessageTicks)
	m.refreshProtectExplanation(msg.path)
	return m, nil
}

func (m appModel) handleProtectRemoved(msg protectRemovedMsg) (tea.Model, tea.Cmd) {
	m.loadingLabel = ""
	if msg.err != nil {
		m.clearNotice()
		m.errorMsg = msg.err.Error()
		return m, nil
	}
	m.errorMsg = ""
	m.clearNotice()
	m.applyConfig(msg.cfg)
	m.protect.syncPaths(msg.cfg.ProtectedPaths)
	m.protect.syncScopes(msg.cfg.CommandExcludes)
	if msg.removed {
		m.protect.setMessage("Protected path removed: "+msg.path, routeMessageTicks)
	} else {
		m.protect.setMessage("Protected path not found: "+msg.path, routeMessageLongTicks)
	}
	m.refreshProtectExplanation(m.protect.selectedPath())
	return m, nil
}

func (m appModel) handleAnalyzeActionFinished(msg analyzeActionFinishedMsg) (tea.Model, tea.Cmd) {
	m.loadingLabel = ""
	if msg.err != nil {
		m.clearNotice()
		m.errorMsg = msg.err.Error()
		return m, nil
	}
	completed, deleted, failed, skipped, protected := countResultStatuses(msg.result)
	m.pendingAnalyzeSelectionPath = m.analyze.fallbackSelectionPathAfterRemoval(analyzeResultDeletedPaths(msg.result))
	for _, item := range msg.result.Items {
		if item.Status == domain.StatusDeleted || item.Status == domain.StatusCompleted {
			m.analyze.removeStage(item.Path)
		}
	}
	if m.analyze.filter == analyzeFilterQueued && len(m.analyze.stageOrder) == 0 {
		m.analyze.filter = analyzeFilterAll
	}
	m.analyze.syncPreviewWindow()
	if deleted > 0 || completed > 0 {
		m.errorMsg = ""
		m.clearNotice()
		return m.startAnalyzeReload("analysis refresh", false)
	}
	if protected > 0 || failed > 0 || skipped > 0 {
		m.errorMsg = ""
		m.setNotice(analyzeActionSummary(msg.result))
		return m, nil
	}
	m.errorMsg = ""
	m.clearNotice()
	return m, nil
}
