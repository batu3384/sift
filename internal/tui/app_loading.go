package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const planLoadTransitionDelay = 350 * time.Millisecond

func (m appModel) currentLoadingLabel() string {
	switch m.route {
	case RouteHome:
		if m.home.loading {
			return m.home.loadingLabel
		}
	case RouteStatus:
		if m.status.loading {
			return m.status.loadingLabel
		}
	case RouteDoctor:
		if m.doctor.loading {
			return m.doctor.loadingLabel
		}
	case RouteUninstall:
		if m.uninstall.loading {
			return m.uninstall.loadingLabel
		}
	}
	return m.loadingLabel
}

func (m appModel) dashboardLoadingActive() bool {
	return m.home.loading || m.status.loading || m.doctor.loading
}

func (m appModel) planLoadPending() bool {
	return m.activePlanRequestID != 0 && strings.TrimSpace(m.currentLoadingLabel()) != ""
}

func (m appModel) planLoadActive() bool {
	return m.planLoadPending() && m.planLoadTransitionVisible
}

func (m *appModel) clearDashboardLoading() {
	m.setHomeLoading("")
	m.setStatusLoading("")
	m.setDoctorLoading("")
}

func (m *appModel) setHomeLoading(label string) {
	m.home.loading = strings.TrimSpace(label) != ""
	m.home.loadingLabel = label
}

func (m *appModel) setStatusLoading(label string) {
	m.status.loading = strings.TrimSpace(label) != ""
	m.status.loadingLabel = label
}

func (m *appModel) setDoctorLoading(label string) {
	m.doctor.loading = strings.TrimSpace(label) != ""
	m.doctor.loadingLabel = label
}

func (m *appModel) setUninstallLoading(label string) {
	m.uninstall.loading = strings.TrimSpace(label) != ""
	m.uninstall.loadingLabel = label
}

func (m *appModel) startUninstallInventoryLoad() tea.Cmd {
	m.nextInventoryRequestID++
	m.activeInventoryRequestID = m.nextInventoryRequestID
	m.setUninstallLoading("installed apps")
	if len(m.uninstall.items) == 0 {
		m.uninstall.setMessage("Loading apps...", routeMessageLongTicks)
	} else {
		m.uninstall.setMessage("Refreshing apps...", routeMessageLongTicks)
	}
	return batchWithUITick(loadUninstallInventoryCmd(m.callbacks.LoadInstalledApps, m.callbacks.LoadCachedInstalledApps, m.activeInventoryRequestID))
}

func planLoadTransitionCmd(requestID int) tea.Cmd {
	if requestID == 0 {
		return nil
	}
	return tea.Tick(planLoadTransitionDelay, func(time.Time) tea.Msg {
		return planLoadTransitionMsg{requestID: requestID}
	})
}

func (m appModel) transitionView() string {
	width, _ := effectiveSize(m.width, m.height)
	label := m.currentLoadingLabel()
	target := m.transitionTargetLabel()
	subtitle := m.transitionSubtitle()
	stats := []string{
		renderStatCard("target", target, "review", 28),
		renderStatCard("task", loadingDisplayLabel(label), "safe", 34),
	}
	if back := m.transitionReturnLabel(); back != "" {
		stats = append(stats, renderStatCard("return", back, "review", 28))
	}
	lines := []string{
		mutedStyle.Render(loadingPulseLine(label, appMotionState(m))),
		mutedStyle.Render("State   " + loadingStageFlow(label)),
		mutedStyle.Render("Target  " + target),
	}
	if back := m.transitionReturnLabel(); back != "" {
		lines = append(lines, mutedStyle.Render("Back    "+back))
	}
	lines = append(lines, "", reviewStyle.Render("esc cancels"))
	body := renderPanel("LOADING", subtitle, strings.Join(lines, "\n"), width-4, false)
	return renderChrome(
		m.transitionTitle(),
		subtitle,
		stats,
		body,
		nil,
		width,
		false,
		m.height,
	)
}

func (m appModel) transitionTitle() string {
	switch m.pendingTargetRoute {
	case RouteReview:
		return "SIFT / Review"
	case RouteAnalyze:
		return "SIFT / Analyze"
	default:
		return "SIFT / Loading"
	}
}

func (m appModel) transitionSubtitle() string {
	switch m.pendingTargetRoute {
	case RouteReview:
		return "opening review"
	case RouteAnalyze:
		return "opening analysis"
	default:
		return "loading"
	}
}

func (m appModel) transitionTargetLabel() string {
	switch m.pendingTargetRoute {
	case RouteReview:
		return "review"
	case RouteAnalyze:
		return "analysis"
	default:
		return "next screen"
	}
}

func (m appModel) transitionReturnLabel() string {
	switch m.pendingReviewReturn {
	case RouteHome:
		return "home"
	case RouteClean:
		return "clean"
	case RouteTools:
		return "tools"
	case RouteAnalyze:
		return "analyze"
	case RouteUninstall:
		return "uninstall"
	case RouteResult:
		return "result"
	default:
		return ""
	}
}

func (m appModel) handlePlanLoadKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.matchesQuit(msg), m.matchesBack(msg):
		label := loadingDisplayLabel(m.currentLoadingLabel())
		m.cancelPendingPlanLoad()
		m.errorMsg = ""
		m.setNotice("Cancelled " + label + ".")
		return m, nil
	default:
		return m, nil
	}
}
