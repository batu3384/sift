package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/batu3384/sift/internal/domain"
)

func (m appModel) updateAnalyze(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.analyze.searchActive {
		switch msg.String() {
		case "esc":
			m.analyze.stopSearch(true)
			m.errorMsg = ""
			m.setNotice("Search cleared.")
			return m, m.startAnalyzeReviewPreviewLoad()
		case "enter":
			m.analyze.stopSearch(false)
			m.errorMsg = ""
			m.clearNotice()
			return m, m.startAnalyzeReviewPreviewLoad()
		}
		var cmd tea.Cmd
		m.analyze.search, cmd = m.analyze.search.Update(msg)
		m.analyze.clampCursor()
		m.analyze.syncPreviewWindow()
		return m, tea.Batch(cmd, m.startAnalyzeReviewPreviewLoad())
	}
	previewDirty := false
	switch {
	case m.matchesQuit(msg):
		return m.leaveAnalyze()
	case m.matchesBack(msg):
		if m.analyze.loading {
			m.cancelPendingPlanLoad()
			m.analyze.errMsg = "Refresh cancelled. Current analysis view remains active."
			return m, nil
		}
		if len(m.analyze.history) == 0 {
			return m.leaveAnalyze()
		}
		last := m.analyze.history[len(m.analyze.history)-1]
		m.analyze.history = m.analyze.history[:len(m.analyze.history)-1]
		m.analyze.plan = last.plan
		m.analyze.cursor = last.cursor
		m.analyze.errMsg = ""
		m.analyze.loading = false
		previewDirty = true
	case msg.Type == tea.KeyPgUp:
		if m.analyzeFlow.scrollLedgerUp(m.analyzeFlow.ledgerScrollPageSize()) {
			return m, nil
		}
	case msg.Type == tea.KeyPgDown:
		if m.analyzeFlow.scrollLedgerDown(m.analyzeFlow.ledgerScrollPageSize()) {
			return m, nil
		}
	case msg.Type == tea.KeyHome:
		if m.analyzeFlow.scrollLedgerToOldest() {
			return m, nil
		}
	case msg.Type == tea.KeyEnd:
		if m.analyzeFlow.scrollOffset > 0 || !m.analyzeFlow.autoFollow {
			m.analyzeFlow.scrollLedgerToLatest()
			return m, nil
		}
	case m.matchesFocus(msg):
		m.analyze.cyclePane()
		previewDirty = true
	case m.matchesUp(msg):
		if m.analyze.activePane() == analyzePaneQueue {
			if m.analyze.queueCursor > 0 {
				m.analyze.queueCursor--
			}
		} else if m.analyze.cursor > 0 {
			m.analyze.cursor--
		}
		previewDirty = true
	case m.matchesDown(msg):
		if m.analyze.activePane() == analyzePaneQueue {
			if m.analyze.queueCursor < len(m.analyze.sortedStageOrder())-1 {
				m.analyze.queueCursor++
			}
		} else if m.analyze.cursor < len(m.analyze.visibleIndices())-1 {
			m.analyze.cursor++
		}
		previewDirty = true
	case m.matchesFilter(msg):
		m.analyze.cycleFilter()
		previewDirty = true
	case m.matchesSearch(msg):
		m.analyze.startSearch()
		m.errorMsg = ""
		m.clearNotice()
	case m.matchesSort(msg):
		m.analyze.cycleQueueSort()
		previewDirty = true
	case m.matchesActivate(msg):
		item, ok := m.analyze.selectedActiveItem()
		if !ok || !canDescendInto(item) || m.analyze.loading {
			return m, nil
		}
		m.analyze.loading = true
		// Show target path in loading label
		targetPath := strings.TrimSpace(item.Path)
		if targetPath != "" {
			m.loadingLabel = "analyze: " + targetPath
		} else {
			m.loadingLabel = "open folder analysis"
		}
		m.pendingAnalyzePushHistory = true
		m.pendingAnalyzePreserveCursor = false
		m.pendingAnalyzeSelectionPath = ""
		return m.startPlanLoad(m.loadingLabel, RouteAnalyze, m.analyzeReturnRoute, RouteHomeOrExit(m), loadPlanCmd(func() (domain.ExecutionPlan, error) {
			return m.callbacks.LoadAnalyzeTarget(item.Path)
		}))
	case m.matchesRefresh(msg):
		return m.startAnalyzeReload("refresh analysis view", true)
	case m.matchesStage(msg):
		if m.analyze.activePane() == analyzePaneQueue {
			item, ok := m.analyze.selectedQueuedItem()
			if ok {
				m.analyze.removeStage(item.Path)
			}
		} else {
			item, ok := m.analyze.selectedItem()
			if !ok || !canStage(item) {
				return m, nil
			}
			m.analyze.toggleStage(item)
		}
		previewDirty = true
	case m.matchesUnstage(msg):
		item, ok := m.analyze.selectedActiveItem()
		if ok {
			m.analyze.removeStage(item.Path)
		}
		previewDirty = true
	case m.matchesOpen(msg):
		item, ok := m.analyze.selectedActiveItem()
		if !ok {
			return m, nil
		}
		if m.callbacks.OpenPath == nil {
			m.clearNotice()
			m.errorMsg = "Open is not available in this environment."
			return m, nil
		}
		if err := m.callbacks.OpenPath(item.Path); err != nil {
			m.clearNotice()
			m.errorMsg = err.Error()
			return m, nil
		}
		m.errorMsg = ""
		m.setNotice("Opened " + item.Path)
	case m.matchesReveal(msg):
		item, ok := m.analyze.selectedActiveItem()
		if !ok {
			return m, nil
		}
		if m.callbacks.RevealPath == nil {
			m.clearNotice()
			m.errorMsg = "Reveal is not available in this environment."
			return m, nil
		}
		if err := m.callbacks.RevealPath(item.Path); err != nil {
			m.clearNotice()
			m.errorMsg = err.Error()
			return m, nil
		}
		m.errorMsg = ""
		m.setNotice("Revealed " + item.Path)
	case m.matchesDelete(msg):
		if m.analyze.loading {
			return m, nil
		}
		if m.callbacks.TrashPaths == nil {
			m.clearNotice()
			m.errorMsg = "Trash is not available in this environment."
			return m, nil
		}
		paths := m.analyzeActionPaths()
		if len(paths) == 0 {
			m.clearNotice()
			m.errorMsg = "Select one or more paths before moving them to Trash."
			return m, nil
		}
		m.errorMsg = ""
		m.clearNotice()
		m.loadingLabel = analyzeTrashLabel(paths)
		return m, batchWithUITick(trashAnalyzePathsCmd(m.callbacks.TrashPaths, paths))
	case m.matchesReview(msg):
		if m.analyze.loading {
			return m, nil
		}
		paths := m.analyze.reviewPreviewPaths()
		if len(paths) == 0 {
			m.clearNotice()
			m.errorMsg = "Pick a cleanup item before opening review."
			return m, nil
		}
		m.errorMsg = ""
		m.clearNotice()
		if previewPlan, loaded := m.analyze.reviewPreviewPlan(); loaded {
			m.setReviewPlan(previewPlan, shouldExecutePlan(previewPlan))
			m.route = RouteReview
			m.reviewReturnRoute = RouteAnalyze
			m.resultReturnRoute = RouteHomeOrExit(m)
			return m, nil
		}
		return m.startPlanLoad("staged cleanup review", RouteReview, RouteAnalyze, RouteHomeOrExit(m), loadPlanCmd(func() (domain.ExecutionPlan, error) {
			return m.callbacks.LoadReviewForPaths(paths)
		}))
	}
	if previewDirty {
		m.analyze.syncPreviewWindow()
		return m, m.startAnalyzeReviewPreviewLoad()
	}
	return m, nil
}

func (m *appModel) startAnalyzeReviewPreviewLoad() tea.Cmd {
	if m.callbacks.LoadReviewForPaths == nil {
		return nil
	}
	key := m.analyze.stagedReviewKey()
	if key == "" {
		m.analyze.setReviewPreviewLoading("")
		m.activeAnalyzePreviewRequestID = 0
		return nil
	}
	if m.analyze.reviewPreview.loaded && strings.TrimSpace(m.analyze.reviewPreview.key) == key {
		return nil
	}
	if m.analyze.reviewPreview.loading && strings.TrimSpace(m.analyze.reviewPreview.key) == key {
		return nil
	}
	paths := m.analyze.reviewPreviewPaths()
	if len(paths) == 0 {
		return nil
	}
	m.nextMenuPreviewRequestID++
	m.activeAnalyzePreviewRequestID = m.nextMenuPreviewRequestID
	m.analyze.setReviewPreviewLoading(key)
	return batchWithUITick(loadMenuPreviewCmd(RouteAnalyze, key, m.activeAnalyzePreviewRequestID, func() (domain.ExecutionPlan, error) {
		return m.callbacks.LoadReviewForPaths(paths)
	}))
}

func (m appModel) updateReview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	effectivePlan := m.review.effectivePlan()
	executable := shouldExecutePlan(effectivePlan)
	switch {
	case m.matchesQuit(msg), m.matchesBack(msg):
		return m.leaveReview()
	case m.matchesUp(msg):
		if m.review.cursor > 0 {
			m.review.cursor--
		}
	case m.matchesDown(msg):
		if m.review.cursor < len(m.review.plan.Items)-1 {
			m.review.cursor++
		}
	case m.matchesStage(msg):
		next, cmd := m.review.Update(msg)
		m.review = next.(planModel)
		return m, cmd
	case m.matchesModule(msg):
		m.review.toggleCurrentGroup()
	case m.matchesCancel(msg):
		if m.review.requiresDecision {
			return m.leaveReview()
		}
	case m.matchesExecute(msg):
		if !executable {
			m.errorMsg = "No actionable items are selected. Include at least one item before executing."
			return m, nil
		}
		m.errorMsg = ""
		return m.executeCurrentReview()
	}
	return m, nil
}

func (m appModel) updatePreflight(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.matchesQuit(msg), m.matchesBack(msg), m.matchesCancel(msg):
		if m.reviewReturnRoute == RouteAnalyze {
			m.analyzeFlow.markReviewReady(m.preflight.plan)
		} else if m.preflight.plan.Command == "clean" {
			m.cleanFlow.markReviewReady(m.preflight.plan)
		} else if m.preflight.plan.Command == "uninstall" {
			m.uninstallFlow.markReviewReady(m.preflight.plan)
		}
		m.preflight = permissionPreflightModel{}
		m.route = RouteReview
		return m, nil
	case m.matchesExecute(msg):
		m.errorMsg = ""
		m.noteAcceptedPermissionProfile(m.preflight)
		if m.preflight.needsAdmin {
			if m.permissionWarmup != nil {
				m.loadingLabel = "warm admin access"
				return m, m.permissionWarmup(m.preflight)
			}
			return m.executePreparedPreflight()
		}
		return m.executePreparedPreflight()
	}
	return m, nil
}

func (m appModel) updateResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.matchesUp(msg):
		m.result.flash = false
		if m.result.cursor > 0 {
			m.result.cursor--
		}
	case m.matchesDown(msg):
		m.result.flash = false
		if m.result.cursor < len(m.result.visibleIndices())-1 {
			m.result.cursor++
		}
	case m.matchesFilter(msg):
		m.result.flash = false
		m.result.cycleFilter()
	case m.matchesRetry(msg):
		m.result.flash = false
		paths := resultRecoveryPathsForStatuses(m.result.plan, m.result.result, m.result.filter, domain.StatusFailed)
		if len(paths) == 0 {
			m.errorMsg = "No failed items available for retry review."
			return m, nil
		}
		m.errorMsg = ""
		return m.startPlanLoad("failed item review", RouteReview, RouteResult, m.resultReturnRoute, loadPlanCmd(func() (domain.ExecutionPlan, error) {
			return m.callbacks.LoadReviewForPaths(paths)
		}))
	case m.matchesModule(msg):
		m.result.flash = false
		group := resultCurrentGroupRecoveryCandidates(m.result)
		if len(group) == 0 {
			m.errorMsg = "No current module or target issue batch available for recovery review."
			return m, nil
		}
		paths := make([]string, 0, len(group))
		for _, item := range group {
			if path := strings.TrimSpace(item.Path); path != "" {
				paths = append(paths, path)
			}
		}
		if len(paths) == 0 {
			m.errorMsg = "No current module or target issue batch available for recovery review."
			return m, nil
		}
		m.errorMsg = ""
		return m.startPlanLoad("current module recovery", RouteReview, RouteResult, m.resultReturnRoute, loadPlanCmd(func() (domain.ExecutionPlan, error) {
			return m.callbacks.LoadReviewForPaths(paths)
		}))
	case m.matchesReview(msg):
		m.result.flash = false
		paths := resultRecoveryPaths(m.result.plan, m.result.result, m.result.filter)
		if len(paths) == 0 {
			m.errorMsg = "No failed or protected items available for recovery review."
			return m, nil
		}
		m.errorMsg = ""
		return m.startPlanLoad("issue recovery review", RouteReview, RouteResult, m.resultReturnRoute, loadPlanCmd(func() (domain.ExecutionPlan, error) {
			return m.callbacks.LoadReviewForPaths(paths)
		}))
	case m.matchesQuit(msg), m.matchesBack(msg), m.matchesActivate(msg):
		m.result.flash = false
		return m.navigate(m.resultReturnRoute, m.resultReturnRoute == RouteHome)
	}
	return m, nil
}

func (m appModel) updateProgress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.matchesUp(msg):
		if m.progress.cursor > 0 {
			m.progress.cursor--
		}
		m.progress.autoFollow = false
		if runningCursor, ok := m.progress.runningCursor(); ok && m.progress.cursor == runningCursor {
			m.progress.autoFollow = true
		}
	case m.matchesDown(msg):
		if m.progress.cursor < len(m.progress.plan.Items)-1 {
			m.progress.cursor++
		}
		m.progress.autoFollow = false
		if runningCursor, ok := m.progress.runningCursor(); ok && m.progress.cursor == runningCursor {
			m.progress.autoFollow = true
		}
	case m.matchesStop(msg):
		if m.executionCancel != nil {
			m.executionCancel()
			m.progress.cancelRequested = true
		}
	}
	return m, nil
}

func (m appModel) leaveAnalyze() (tea.Model, tea.Cmd) {
	m.cancelPendingPlanLoad()
	return m.navigate(m.analyzeReturnRoute, m.analyzeReturnRoute == RouteHome)
}

func (m appModel) leaveReview() (tea.Model, tea.Cmd) {
	if m.reviewReturnRoute == RouteAnalyze {
		m.analyzeFlow.markReviewReady(m.review.effectivePlan())
	} else if m.reviewReturnRoute == RouteClean && m.review.plan.Command == "clean" {
		m.cleanFlow.markReviewReady(m.review.effectivePlan())
	} else if m.reviewReturnRoute == RouteUninstall && m.review.plan.Command == "uninstall" {
		m.uninstallFlow.markReviewReady(m.review.effectivePlan())
	}
	return m.navigate(m.reviewReturnRoute, m.reviewReturnRoute == RouteHome)
}
