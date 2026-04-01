package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/batu3384/sift/internal/domain"
)

func batchWithUITick(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return uiTickCmd()
	}
	return tea.Batch(cmd, uiTickCmd())
}

func (m appModel) updateHome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.matchesQuit(msg):
		return m, tea.Quit
	case m.matchesUp(msg):
		if m.home.cursor > 0 {
			m.home.cursor--
		}
	case m.matchesDown(msg):
		if m.home.cursor < len(m.home.actions)-1 {
			m.home.cursor++
		}
	case m.matchesRefresh(msg):
		m.setHomeLoading("dashboard")
		return m, batchWithUITick(loadDashboardCmd(m.callbacks.LoadDashboard))
	case m.matchesTools(msg):
		m.route = RouteTools
		return m, nil
	case m.matchesActivate(msg):
		action := m.home.actions[m.home.cursor]
		if !action.Enabled {
			return m, nil
		}
		switch action.ID {
		case "status":
			m.route = RouteStatus
			return m, nil
		case "analyze":
			// Show "scanning home directory" in loading label
			return m.startPlanLoad("analyze", RouteAnalyze, RouteHome, RouteHome, loadPlanCmd(m.callbacks.LoadAnalyzeHome))
		case "clean":
			m.route = RouteClean
			return m, m.startCleanPreviewLoad()
		case "uninstall":
			m.route = RouteUninstall
			return m, m.startUninstallInventoryLoad()
		case "optimize":
			return m.startPlanLoad("optimize review", RouteReview, RouteHome, RouteHome, loadPlanCmd(m.callbacks.LoadOptimize))
		}
	}
	return m, nil
}

func (m appModel) updateClean(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.matchesQuit(msg), m.matchesBack(msg):
		return m.navigate(RouteHome, true)
	case m.matchesUp(msg):
		if m.clean.cursor > 0 {
			m.clean.cursor--
			return m, m.startCleanPreviewLoad()
		}
	case m.matchesDown(msg):
		if m.clean.cursor < len(m.clean.actions)-1 {
			m.clean.cursor++
			return m, m.startCleanPreviewLoad()
		}
	case m.matchesActivate(msg):
		action := m.clean.actions[m.clean.cursor]
		switch action.ID {
		case "back":
			return m.navigate(RouteHome, true)
		default:
			if action.ProfileKey != "" {
				if previewPlan, ok := m.clean.previewPlanForSelected(); ok {
					m.setReviewPlan(previewPlan, shouldExecutePlan(previewPlan))
					m.route = RouteReview
					m.reviewReturnRoute = RouteClean
					m.resultReturnRoute = RouteClean
					return m, nil
				}
				label := strings.ToLower(strings.TrimSpace(action.Title)) + " review"
				return m.startPlanLoad(label, RouteReview, RouteClean, RouteClean, loadPlanCmd(func() (domain.ExecutionPlan, error) {
					return m.callbacks.LoadCleanProfile(action.ProfileKey)
				}))
			}
		}
	}
	return m, nil
}

func (m *appModel) startCleanPreviewLoad() tea.Cmd {
	if m.callbacks.LoadCleanProfile == nil || m.clean.cursor < 0 || m.clean.cursor >= len(m.clean.actions) {
		return nil
	}
	action := m.clean.actions[m.clean.cursor]
	key := strings.TrimSpace(action.ProfileKey)
	if key == "" {
		m.clean.setPreviewLoading("")
		m.activeCleanPreviewRequestID = 0
		return nil
	}
	if m.clean.preview.loaded && strings.TrimSpace(m.clean.preview.key) == key {
		return nil
	}
	if m.clean.preview.loading && strings.TrimSpace(m.clean.preview.key) == key {
		return nil
	}
	m.nextMenuPreviewRequestID++
	m.activeCleanPreviewRequestID = m.nextMenuPreviewRequestID
	m.clean.setPreviewLoading(key)
	return batchWithUITick(loadMenuPreviewCmd(RouteClean, key, m.activeCleanPreviewRequestID, func() (domain.ExecutionPlan, error) {
		return m.callbacks.LoadCleanProfile(key)
	}))
}

func (m appModel) updateTools(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.matchesQuit(msg), m.matchesBack(msg):
		return m.navigate(RouteHome, true)
	case m.matchesUp(msg):
		if m.tools.cursor > 0 {
			m.tools.cursor--
		}
	case m.matchesDown(msg):
		if m.tools.cursor < len(m.tools.actions)-1 {
			m.tools.cursor++
		}
	case m.matchesActivate(msg):
		action := m.tools.actions[m.tools.cursor]
		if !action.Enabled {
			return m, nil
		}
		switch action.ID {
		case "back":
			return m.navigate(RouteHome, true)
		case "check":
			m.route = RouteDoctor
			return m, nil
		case "optimize":
			return m.startPlanLoad("optimize review", RouteReview, RouteTools, RouteTools, loadPlanCmd(m.callbacks.LoadOptimize))
		case "autofix":
			return m.startPlanLoad("autofix review", RouteReview, RouteTools, RouteTools, loadPlanCmd(m.callbacks.LoadAutofix))
		case "installer":
			return m.startPlanLoad("installer cleanup review", RouteReview, RouteTools, RouteTools, loadPlanCmd(m.callbacks.LoadInstaller))
		case "protect":
			m.route = RouteProtect
			m.protect.setMessage("", 0)
			m.refreshProtectExplanation(m.protect.selectedPath())
			return m, nil
		case "purge_scan":
			return m.startPlanLoad("purge scan", RouteReview, RouteTools, RouteTools, loadPlanCmd(m.callbacks.LoadPurgeScan))
		case "doctor":
			m.route = RouteDoctor
			return m, nil
		}
	}
	return m, nil
}

func (m appModel) updateUninstall(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.uninstall.searchActive {
		switch msg.String() {
		case "esc":
			m.uninstall.stopSearch()
			m.uninstall.setMessage("Search cleared.", routeMessageTicks)
			return m, m.startUninstallPreviewLoad()
		case "enter":
			m.uninstall.stopSearch()
			m.uninstall.setMessage("", 0)
			return m, m.startUninstallPreviewLoad()
		}
		var cmd tea.Cmd
		m.uninstall.search, cmd = m.uninstall.search.Update(msg)
		m.uninstall.applyFilter()
		return m, tea.Batch(cmd, m.startUninstallPreviewLoad())
	}
	switch {
	case m.matchesQuit(msg), m.matchesBack(msg):
		return m.navigate(RouteHome, true)
	case m.matchesSearch(msg):
		m.uninstall.startSearch()
		return m, nil
	case m.matchesStage(msg):
		item, ok, staged := m.uninstall.toggleSelectedStage()
		if !ok {
			m.uninstall.setMessage("Select an app before queueing it.", routeMessageTicks)
			return m, nil
		}
		if staged {
			m.uninstall.setMessage(item.Name+" queued for batch review.", routeMessageTicks)
		} else {
			m.uninstall.setMessage(item.Name+" removed from the queue.", routeMessageTicks)
		}
		return m, nil
	case m.matchesUnstage(msg):
		item, ok := m.uninstall.selected()
		if !ok || !m.uninstall.isStaged(item) {
			m.uninstall.setMessage("Selected app is not queued.", routeMessageTicks)
			return m, nil
		}
		m.uninstall.toggleStage(item)
		m.uninstall.setMessage(item.Name+" removed from the queue.", routeMessageTicks)
		return m, nil
	case m.matchesReview(msg):
		names := m.uninstall.stageNames()
		if len(names) == 0 {
			m.uninstall.setMessage("Queue apps before opening batch review.", routeMessageTicks)
			return m, nil
		}
		if m.callbacks.LoadUninstallBatchPlan == nil {
			m.uninstall.setMessage("Batch uninstall review is unavailable.", routeMessageTicks)
			return m, nil
		}
		return m.startPlanLoad("batch uninstall plan", RouteReview, RouteUninstall, RouteUninstall, loadPlanCmd(func() (domain.ExecutionPlan, error) {
			return m.callbacks.LoadUninstallBatchPlan(names)
		}))
	case m.matchesRefresh(msg):
		return m, m.startUninstallInventoryLoad()
	case m.matchesUp(msg):
		if m.uninstall.cursor > 0 {
			m.uninstall.cursor--
			return m, m.startUninstallPreviewLoad()
		}
	case m.matchesDown(msg):
		if m.uninstall.cursor < len(m.uninstall.filtered)-1 {
			m.uninstall.cursor++
			return m, m.startUninstallPreviewLoad()
		}
	case m.matchesActivate(msg):
		item, ok := m.uninstall.selected()
		if !ok {
			return m, nil
		}
		if previewPlan, loaded := m.uninstall.previewPlanForSelected(); loaded {
			m.setReviewPlan(previewPlan, shouldExecutePlan(previewPlan))
			m.route = RouteReview
			m.reviewReturnRoute = RouteUninstall
			m.resultReturnRoute = RouteUninstall
			return m, nil
		}
		return m.startPlanLoad("uninstall plan", RouteReview, RouteUninstall, RouteUninstall, loadPlanCmd(func() (domain.ExecutionPlan, error) {
			return m.callbacks.LoadUninstallPlan(item.Name)
		}))
	}
	return m, nil
}

func (m *appModel) startUninstallPreviewLoad() tea.Cmd {
	if m.callbacks.LoadUninstallPlan == nil {
		return nil
	}
	item, ok := m.uninstall.selected()
	if !ok {
		m.uninstall.setPreviewLoading("")
		m.activeUninstallPreviewRequestID = 0
		return nil
	}
	key := uninstallStageKey(item.Name)
	if key == "" {
		m.uninstall.setPreviewLoading("")
		m.activeUninstallPreviewRequestID = 0
		return nil
	}
	if m.uninstall.preview.loaded && strings.TrimSpace(m.uninstall.preview.key) == key {
		return nil
	}
	if m.uninstall.preview.loading && strings.TrimSpace(m.uninstall.preview.key) == key {
		return nil
	}
	m.nextMenuPreviewRequestID++
	m.activeUninstallPreviewRequestID = m.nextMenuPreviewRequestID
	m.uninstall.setPreviewLoading(key)
	return batchWithUITick(loadMenuPreviewCmd(RouteUninstall, key, m.activeUninstallPreviewRequestID, func() (domain.ExecutionPlan, error) {
		return m.callbacks.LoadUninstallPlan(item.Name)
	}))
}

func (m appModel) updateStatus(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.matchesQuit(msg):
		return m.navigate(RouteHomeOrExit(m), m.hasHome)
	case m.matchesBack(msg), m.matchesActivate(msg):
		return m.navigate(RouteHomeOrExit(m), m.hasHome)
	case m.matchesCompanion(msg):
		if m.status.companionMode == "off" {
			m.status.companionMode = "full"
		} else {
			m.status.companionMode = "off"
		}
	case m.matchesRefresh(msg):
		m.setStatusLoading("dashboard")
		return m, batchWithUITick(loadDashboardCmd(m.callbacks.LoadDashboard))
	}
	return m, nil
}

func (m appModel) updateDoctor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.matchesQuit(msg):
		if m.hasHome && m.route != RouteHome {
			return m.navigate(RouteTools, false)
		}
		return m.navigate(RouteHomeOrExit(m), m.hasHome)
	case m.matchesBack(msg), m.matchesActivate(msg):
		if m.hasHome && m.route != RouteHome {
			return m.navigate(RouteTools, false)
		}
		return m.navigate(RouteHomeOrExit(m), m.hasHome)
	case m.matchesUp(msg):
		if m.doctor.cursor > 0 {
			m.doctor.cursor--
		}
	case m.matchesDown(msg):
		if m.doctor.cursor < len(m.doctor.diagnostics)-1 {
			m.doctor.cursor++
		}
	case m.matchesRefresh(msg):
		m.setDoctorLoading("doctor")
		return m, batchWithUITick(loadDashboardCmd(m.callbacks.LoadDashboard))
	}
	return m, nil
}
