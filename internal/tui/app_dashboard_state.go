package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/batuhanyuksel/sift/internal/config"
	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/engine"
)

func (m *appModel) applyDashboard(data DashboardData) {
	prevLive := m.status.live
	rxRate, txRate := dashboardNetworkRates(prevLive, data.Report.Live)
	readRate, writeRate := dashboardDiskIORates(prevLive, data.Report.Live)
	m.home.live = data.Report.Live
	m.home.lastExecution = data.Report.LastExecution
	m.home.diagnostics = data.Diagnostics
	m.home.updateNotice = data.Update
	m.home.cfg = m.cfg
	m.home.actions = buildHomeActions(m.cfg)
	m.clean.actions = buildCleanActions()
	m.tools.actions = buildToolsActions(m.cfg)
	m.status.live = data.Report.Live
	m.status.networkRxRate, m.status.networkTxRate = rxRate, txRate
	m.status.diskReadRate, m.status.diskWriteRate = readRate, writeRate
	if live := data.Report.Live; live != nil {
		m.status.cpuTrend = appendStatusTrend(m.status.cpuTrend, live.CPUPercent)
		m.status.memoryTrend = appendStatusTrend(m.status.memoryTrend, live.MemoryUsedPercent)
		m.status.networkTrend = appendStatusTrend(m.status.networkTrend, rxRate+txRate)
		m.status.diskTrend = appendStatusTrend(m.status.diskTrend, readRate+writeRate)
	}
	m.status.scans = data.Report.RecentScans
	m.status.lastExecution = data.Report.LastExecution
	m.status.diagnostics = data.Diagnostics
	m.status.updateNotice = data.Update
	m.doctor.diagnostics = data.Diagnostics
	m.lastDashboardSync = formatDashboardSync(data.Report.Live)
}

func (m *appModel) applyConfig(cfg config.Config) {
	m.cfg = config.Normalize(cfg)
	m.home.cfg = m.cfg
	m.home.actions = buildHomeActions(m.cfg)
	m.tools.actions = buildToolsActions(m.cfg)
	m.protect.syncPaths(m.cfg.ProtectedPaths)
	m.protect.syncFamilies(m.cfg.ProtectedFamilies)
	m.protect.syncScopes(m.cfg.CommandExcludes)
}

func (m *appModel) applyInstalledApps(apps []domain.AppEntry) {
	items := make([]uninstallItem, 0, len(apps))
	for _, app := range apps {
		name := strings.TrimSpace(app.DisplayName)
		if name == "" {
			name = strings.TrimSpace(app.Name)
		}
		if name == "" {
			continue
		}
		items = append(items, uninstallItem{
			Name:          name,
			HasNative:     strings.TrimSpace(app.UninstallCommand) != "" || strings.TrimSpace(app.QuietUninstallCommand) != "",
			Origin:        strings.TrimSpace(app.Origin),
			Location:      strings.TrimSpace(app.BundlePath),
			LastModified:  formatAppModified(app.LastModified),
			LastSeenAt:    app.LastModified,
			ApproxBytes:   app.ApproxBytes,
			SizeLabel:     uninstallSizeLabel(app.ApproxBytes),
			Sensitive:     app.Sensitive,
			FamilyMatches: append([]string{}, app.FamilyMatches...),
			RequiresAdmin: app.RequiresAdmin,
		})
	}
	m.uninstall.setItems(items)
	m.uninstall.setMessage("", 0)
}

func formatAppModified(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	age := time.Since(value)
	switch {
	case age < 48*time.Hour:
		return "recently"
	case age < 14*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(age.Hours()/24))
	case age < 60*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(age.Hours()/(24*7)))
	default:
		return value.Local().Format("2006-01-02")
	}
}

func dashboardNetworkRates(previous, current *engine.SystemSnapshot) (float64, float64) {
	seconds := dashboardDeltaSeconds(previous, current)
	if seconds <= 0 || previous == nil || current == nil {
		return 0, 0
	}
	rx := dashboardRateDelta(previous.NetworkRxBytes, current.NetworkRxBytes, seconds)
	tx := dashboardRateDelta(previous.NetworkTxBytes, current.NetworkTxBytes, seconds)
	return rx, tx
}

func dashboardDiskIORates(previous, current *engine.SystemSnapshot) (float64, float64) {
	seconds := dashboardDeltaSeconds(previous, current)
	if seconds <= 0 || previous == nil || current == nil || previous.DiskIO == nil || current.DiskIO == nil {
		return 0, 0
	}
	read := dashboardRateDelta(previous.DiskIO.ReadBytes, current.DiskIO.ReadBytes, seconds)
	write := dashboardRateDelta(previous.DiskIO.WriteBytes, current.DiskIO.WriteBytes, seconds)
	return read, write
}

func dashboardDeltaSeconds(previous, current *engine.SystemSnapshot) float64 {
	if previous == nil || current == nil {
		return 0
	}
	prevAt, err := time.Parse(time.RFC3339, strings.TrimSpace(previous.CollectedAt))
	if err != nil {
		return 0
	}
	currAt, err := time.Parse(time.RFC3339, strings.TrimSpace(current.CollectedAt))
	if err != nil {
		return 0
	}
	seconds := currAt.Sub(prevAt).Seconds()
	if seconds <= 0 {
		return 0
	}
	return seconds
}

func dashboardRateDelta(previous, current uint64, seconds float64) float64 {
	if seconds <= 0 || current < previous {
		return 0
	}
	return float64(current-previous) / seconds
}

func appendStatusTrend(history []float64, value float64) []float64 {
	const limit = 14
	history = append(history, value)
	if len(history) > limit {
		history = append([]float64{}, history[len(history)-limit:]...)
	}
	return history
}

func (m *appModel) refreshProtectExplanation(path string) {
	if strings.TrimSpace(path) == "" || m.callbacks.ExplainProtection == nil {
		m.protect.explanation = nil
		return
	}
	explanation := m.callbacks.ExplainProtection(path)
	m.protect.explanation = &explanation
}

func RouteHomeOrExit(m appModel) Route {
	if m.hasHome {
		return RouteHome
	}
	return ""
}

func (m appModel) navigate(route Route, refreshDashboard bool) (tea.Model, tea.Cmd) {
	if route == "" {
		return m, tea.Quit
	}
	m.route = route
	m.preflight = permissionPreflightModel{}
	m.helpVisible = false
	m.clearNotice()
	if refreshDashboard && route == RouteHome {
		m.activeInventoryRequestID = 0
		m.setUninstallLoading("")
		m.setStatusLoading("")
		m.setDoctorLoading("")
		m.setHomeLoading("dashboard")
		return m, tea.Batch(loadDashboardCmd(m.callbacks.LoadDashboard), dashboardTickCmd(), uiTickCmd())
	}
	m.loadingLabel = ""
	if route != RouteUninstall {
		m.activeInventoryRequestID = 0
		m.setUninstallLoading("")
	}
	if route != RouteHome {
		m.setHomeLoading("")
	}
	if route != RouteStatus {
		m.setStatusLoading("")
	}
	if route != RouteDoctor {
		m.setDoctorLoading("")
	}
	return m, nil
}
