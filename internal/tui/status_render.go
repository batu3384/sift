package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/engine"
	"github.com/batuhanyuksel/sift/internal/platform"
	"github.com/batuhanyuksel/sift/internal/store"
)

func statusOverviewSubtitle(live *engine.SystemSnapshot, lastExecution *store.ExecutionSummary, scans []store.RecentScan) string {
	if lastExecution != nil {
		return fmt.Sprintf("%s  •  live  •  alerts", statusExecutionLabel(lastExecution, scans))
	}
	if live != nil {
		return "live  •  alerts"
	}
	return "waiting for live data"
}

func statusAlertLine(model statusModel) string {
	issues := diagnosticIssueCount(model.diagnostics)
	parts := make([]string, 0, 4)
	if issues > 0 {
		parts = append(parts, fmt.Sprintf("%d doctor %s", issues, pl(issues, "issue", "issues")))
	}
	if model.updateNotice != nil && model.updateNotice.Available {
		parts = append(parts, strings.ToUpper(model.updateNotice.LatestVersion)+" ready")
	}
	if model.live != nil {
		if len(model.live.OperatorAlerts) > 0 {
			parts = append(parts, model.live.OperatorAlerts...)
		} else {
			if pressure := statusPressureLabel(model.live); pressure != "" && pressure != "steady" {
				parts = append(parts, "pressure "+pressure)
			}
			if model.live.ThermalState != "" && strings.ToLower(model.live.ThermalState) != "nominal" {
				parts = append(parts, "thermal "+strings.ToLower(model.live.ThermalState))
			}
		}
	}
	if len(parts) == 0 {
		return "Alerts none  •  no active operator issues"
	}
	return "Alerts " + strings.Join(parts, "  •  ")
}

func statusCompactCarryLine(model statusModel, width int) string {
	parts := []string{"Activity"}
	if sessionValue, _ := statusSessionCard(model.lastExecution, model.scans); sessionValue != "" {
		parts = append(parts, sessionValue)
	} else if len(model.scans) > 0 {
		parts = append(parts, statusCompactScanSummary(model.scans[0]))
	}
	if model.live != nil {
		if power := statusCompactBatteryPowerSummary(model.live); power != "" {
			parts = append(parts, power)
		}
		if btDevices := statusBluetoothDevicesSummary(model.live, width); btDevices != "" {
			parts = append(parts, btDevices)
		} else if bt := statusBluetoothSummary(model.live); bt != "" {
			parts = append(parts, bt)
		}
	}
	return strings.Join(parts, "  •  ")
}

func statusCompanionEnabled(model statusModel) bool {
	return strings.TrimSpace(strings.ToLower(model.companionMode)) != "off"
}

func statusCompanionMood(model statusModel) string {
	switch {
	case !statusCompanionEnabled(model):
		return "muted"
	case model.updateNotice != nil && model.updateNotice.Available:
		return "upgrade watch"
	case diagnosticIssueCount(model.diagnostics) > 0:
		return "guard watch"
	case model.live != nil && len(model.live.OperatorAlerts) > 0:
		return "heat watch"
	case model.live != nil && statusPressureLabel(model.live) != "steady":
		return "pressure watch"
	default:
		return "steady watch"
	}
}

func statusCompanionGlyph(model statusModel) string {
	frame := model.signalFrame % 4
	if frame < 0 {
		frame = 0
	}
	if !statusCompanionEnabled(model) {
		return []string{"◌", "◌", "◌", "◌"}[frame]
	}
	switch statusCompanionMood(model) {
	case "upgrade watch":
		return []string{"✦", "✧", "✦", "✧"}[frame]
	case "guard watch":
		return []string{"◉", "◎", "◉", "◎"}[frame]
	case "heat watch":
		return []string{"⬢", "⬡", "⬢", "⬡"}[frame]
	case "pressure watch":
		return []string{"◐", "◓", "◑", "◒"}[frame]
	default:
		return []string{"◒", "◐", "◓", "◑"}[frame]
	}
}

func statusCompanionLabel(model statusModel) string {
	if !statusCompanionEnabled(model) {
		return "companion muted (g wake)"
	}
	return fmt.Sprintf("companion %s %s (g mute)", statusCompanionGlyph(model), statusCompanionMood(model))
}

func statusTacticLine(model statusModel) string {
	parts := []string{"Recommended"}
	switch {
	case model.updateNotice != nil && model.updateNotice.Available:
		parts = append(parts, "apply update window", "rerun doctor after upgrade")
	case diagnosticIssueCount(model.diagnostics) > 0:
		parts = append(parts, "open doctor/check", "resolve posture drift")
	case model.live != nil && len(model.live.OperatorAlerts) > 0:
		parts = append(parts, "inspect power + thermal rails", "keep monitor live")
	case len(model.scans) > 0:
		parts = append(parts, "continue live monitor", "open analyze for deeper drill")
	default:
		parts = append(parts, "watch operator cadence", "open clean when reclaim spikes")
	}
	return strings.Join(parts, "  •  ")
}

func statusHealthLine(live *engine.SystemSnapshot) string {
	if live == nil {
		return "Now      waiting for telemetry"
	}
	parts := []string{fmt.Sprintf("%d / %s", live.HealthScore, strings.ToUpper(live.HealthLabel))}
	if pressure := statusPressureLabel(live); pressure != "" {
		parts = append(parts, "Pressure "+pressure)
	}
	if live.Battery != nil {
		parts = append(parts, fmt.Sprintf("Battery %.0f%% %s", live.Battery.Percent, strings.ToLower(live.Battery.State)))
	} else if live.PowerSource != "" {
		parts = append(parts, "Power "+strings.ToLower(live.PowerSource))
	}
	if live.SystemPowerWatts > 0 {
		parts = append(parts, fmt.Sprintf("%.0fW system", live.SystemPowerWatts))
	}
	if live.ThermalState != "" && strings.ToLower(live.ThermalState) != "nominal" {
		parts = append(parts, "Thermal "+strings.ToLower(live.ThermalState))
	}
	return "Now      " + strings.Join(parts, "  •  ")
}

func statusOverviewView(model statusModel, width int, maxLines int) string {
	live := model.live
	motion := statusMotionState(model)
	lines := make([]string, 0, 8)
	if live == nil {
		lines = append(lines, mutedStyle.Render("No live telemetry available yet."))
		if len(model.scans) == 0 {
			lines = append(lines, mutedStyle.Render("Run sift status, sift analyze, or sift clean to start the session rail."))
		}
		lines = viewportLines(lines, 0, maxLines)
		return strings.Join(lines, "\n")
	}
	if width < 110 || maxLines <= 4 {
		lines = append(lines, mutedStyle.Render(singleLine(fmt.Sprintf("Signal %s  •  %s", signalRailLabelForMotion(motion), statusCompanionLabel(model)), width)))
		lines = append(lines, mutedStyle.Render(singleLine(statusHealthLine(live), width)))
		lines = append(lines, mutedStyle.Render(singleLine(statusWatchLine(model, width), width)))
		if maxLines >= 5 {
			lines = append(lines, mutedStyle.Render(singleLine(statusRecentLine(model, width), width)))
		}
		if maxLines >= 6 {
			lines = append(lines, mutedStyle.Render(singleLine(statusNextLine(model), width)))
		}
		lines = viewportLines(lines, 0, maxLines)
		return strings.Join(lines, "\n")
	}
	// Reserve 11 chars for the 7-wide mascot column when there's enough room.
	textWidth := width
	showMascot := width >= 120 && maxLines >= 6
	if showMascot {
		textWidth = width - 11
	}
	lines = append(lines, mutedStyle.Render(singleLine("Signal   "+signalRailLabelForMotion(motion)+"  •  "+statusCompanionLabel(model), textWidth)))
	healthLine := statusHealthLine(live)
	lines = append(lines, statusOverviewHealthStyle(live).Render(singleLine(healthLine, textWidth)))
	watchLine := statusWatchLine(model, textWidth)
	lines = append(lines, statusOverviewAlertStyle(model).Render(singleLine(watchLine, textWidth)))
	lines = append(lines, mutedStyle.Render(singleLine(statusRecentLine(model, textWidth), textWidth)))
	if host := statusHostSummary(live); host != "" {
		hostParts := []string{host}
		if iface := statusInterfaceSummary(live); iface != "" {
			hostParts = append(hostParts, iface)
		}
		lines = append(lines, mutedStyle.Render(singleLine("Host     "+strings.Join(hostParts, "  •  "), textWidth)))
	}
	lines = append(lines, mutedStyle.Render(singleLine(statusNextLine(model), textWidth)))
	if trendLine := statusOverviewTrendLine(model, textWidth); trendLine != "" && maxLines > len(lines) {
		lines = append(lines, mutedStyle.Render(trendLine))
	}
	lines = viewportLines(lines, 0, maxLines)
	content := strings.Join(lines, "\n")
	if showMascot {
		return lipgloss.JoinHorizontal(lipgloss.Top, content, "  ", statusMascotFrame(model))
	}
	return content
}

func statusWatchLine(model statusModel, width int) string {
	parts := []string{}
	if alert := strings.TrimPrefix(statusAlertLine(model), "Alerts "); alert != "" {
		parts = append(parts, alert)
	}
	if vector := statusHeroVectorLabel(model.live, statusMotionState(model)); vector != "" {
		parts = append(parts, vector)
	}
	if model.live != nil {
		if model.live.CPUTempCelsius > 0 {
			parts = append(parts, fmt.Sprintf("%.1f°C", model.live.CPUTempCelsius))
		}
		if model.live.SystemPowerWatts > 0 {
			parts = append(parts, fmt.Sprintf("%.0fW system", model.live.SystemPowerWatts))
		}
	}
	if len(parts) == 0 {
		parts = append(parts, "no active operator issues")
	}
	return "Risk     " + strings.Join(parts, "  •  ")
}

func statusRecentLine(model statusModel, width int) string {
	parts := []string{}
	if sessionValue, _ := statusSessionCard(model.lastExecution, model.scans); sessionValue != "" {
		parts = append(parts, sessionValue)
	}
	if len(model.scans) > 0 {
		parts = append(parts, statusCompactScanSummary(model.scans[0]))
	}
	if len(parts) == 0 {
		parts = append(parts, "no recent activity")
	}
	return truncateText("Recent   "+strings.Join(parts, "  •  "), width)
}

func statusNextLine(model statusModel) string {
	return "Next     " + strings.TrimPrefix(statusTacticLine(model), "Recommended  •  ")
}

func statusStats(live *engine.SystemSnapshot, lastExecution *store.ExecutionSummary, scans []store.RecentScan, diagnostics []platform.Diagnostic, update *engine.UpdateNotice, width int) []string {
	cardWidth := 24
	if width < 110 {
		cardWidth = width - 8
	}
	healthValue := "loading"
	healthTone := "review"
	storageValue := "unknown"
	storageTone := "review"
	if live != nil {
		healthValue = fmt.Sprintf("%d / %s", live.HealthScore, strings.ToUpper(live.HealthLabel))
		healthTone = toneForHealth(live.HealthScore)
		storageValue = domain.HumanBytes(int64(live.DiskFreeBytes))
		storageTone = toneForPercent(live.DiskUsedPercent, 82, 92)
	}
	stats := []string{
		renderStatCard("health", healthValue, healthTone, cardWidth),
		renderStatCard("disk", storageValue, storageTone, cardWidth),
		renderStatCard("alerts", statusAlertCard(live, diagnostics, update), statusAlertTone(live, diagnostics, update), cardWidth),
	}
	if live != nil && live.Battery != nil {
		battValue := fmt.Sprintf("%.0f%% %s", live.Battery.Percent, live.Battery.State)
		battTone := toneForPercent(100-live.Battery.Percent, 80, 90)
		if strings.ToLower(live.Battery.State) == "charging" || strings.ToLower(live.Battery.State) == "charged" {
			battTone = "safe"
		}
		stats = append(stats, renderStatCard("battery", battValue, battTone, cardWidth))
	}
	if live != nil && live.GPUUsagePercent > 0 {
		gpuLabel := fmt.Sprintf("%.0f%%", live.GPUUsagePercent)
		if live.GPUModel != "" {
			parts := strings.Fields(live.GPUModel)
			if len(parts) > 0 {
				gpuLabel = fmt.Sprintf("%.0f%% %s", live.GPUUsagePercent, parts[len(parts)-1])
			}
		}
		gpuTone := toneForPercent(live.GPUUsagePercent, 60, 80)
		stats = append(stats, renderStatCard("gpu", gpuLabel, gpuTone, cardWidth))
	}
	if sessionValue, sessionTone := statusSessionCard(lastExecution, scans); sessionValue != "" {
		stats = append(stats, renderStatCard("activity", sessionValue, sessionTone, cardWidth))
	} else if len(scans) > 0 {
		stats = append(stats, renderStatCard("activity", fmt.Sprintf("%d %s", len(scans), pl(len(scans), "scan", "scans")), "review", cardWidth))
	}
	return stats
}

func statusAlertCard(live *engine.SystemSnapshot, diagnostics []platform.Diagnostic, update *engine.UpdateNotice) string {
	issues := diagnosticIssueCount(diagnostics)
	parts := make([]string, 0, 2)
	if issues > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", issues, pl(issues, "issue", "issues")))
	}
	if live != nil && len(live.OperatorAlerts) > 0 {
		parts = append(parts, live.OperatorAlerts[0])
	}
	if update != nil && update.Available {
		parts = append(parts, strings.ToUpper(update.LatestVersion))
	}
	if len(parts) == 0 {
		return "clear"
	}
	return strings.Join(parts, " • ")
}

func statusAlertTone(live *engine.SystemSnapshot, diagnostics []platform.Diagnostic, update *engine.UpdateNotice) string {
	for _, diagnostic := range diagnostics {
		if diagnostic.Status == "error" || diagnostic.Status == "warn" {
			return "high"
		}
	}
	if update != nil && update.Available {
		return "review"
	}
	if live != nil {
		if pressure := statusPressureLabel(live); pressure != "" && pressure != "steady" {
			return "review"
		}
		if live.ThermalState != "" && strings.ToLower(live.ThermalState) != "nominal" {
			return "review"
		}
	}
	return "safe"
}

func statusLiveSubtitle(live *engine.SystemSnapshot) string {
	if live == nil {
		return "waiting"
	}
	return "live"
}

func statusSystemView(live *engine.SystemSnapshot, width int, maxLines int) string {
	if live == nil {
		return mutedStyle.Render("No system metrics yet.")
	}
	if width < 48 || maxLines < 7 {
		lines := []string{
			mutedStyle.Render(fmt.Sprintf("CPU %.1f%%  •  MEM %.1f%%", live.CPUPercent, live.MemoryUsedPercent)),
			mutedStyle.Render(fmt.Sprintf("Free %s  •  Proc %d", domain.HumanBytes(int64(live.DiskFreeBytes)), live.ProcessCount)),
			mutedStyle.Render("Pressure " + statusPressureLabel(live)),
		}
		auxParts := make([]string, 0, 2)
		if summary := statusCompactBatteryPowerSummary(live); summary != "" {
			auxParts = append(auxParts, summary)
		}
		if btDevices := statusBluetoothDevicesSummary(live, width); btDevices != "" {
			auxParts = append(auxParts, btDevices)
		} else if bt := statusBluetoothSummary(live); bt != "" {
			auxParts = append(auxParts, bt)
		}
		if len(auxParts) > 0 {
			lines = append(lines, mutedStyle.Render(singleLine(strings.Join(auxParts, "  •  "), width)))
		}
		if live.Load1 > 0 {
			lines = append(lines, mutedStyle.Render(fmt.Sprintf("Load %.2f", live.Load1)))
		}
		lines = viewportLines(lines, 0, maxLines)
		return strings.Join(lines, "\n")
	}
	lines := []string{}
	hostLine := statusHostSummary(live)
	if hardware := statusHardwareSummary(live); hardware != "" {
		combined := hardware
		if hostLine != "" {
			combined = hostLine + "  •  " + hardware
		}
		if len(combined) <= width {
			lines = append(lines, mutedStyle.Render(singleLine(combined, width)))
		} else {
			if hostLine != "" {
				lines = append(lines, mutedStyle.Render(singleLine(hostLine, width)))
			}
			lines = append(lines, mutedStyle.Render(singleLine(hardware, width)))
		}
	} else if hostLine != "" {
		lines = append(lines, mutedStyle.Render(singleLine(hostLine, width)))
	}
	lines = append(lines,
		mutedStyle.Render(singleLine(fmt.Sprintf("CPU %.1f%%  •  Memory %.1f%%  •  Disk %.1f%%", live.CPUPercent, live.MemoryUsedPercent, live.DiskUsedPercent), width)),
		mutedStyle.Render(singleLine(fmt.Sprintf("Uptime %s  •  Processes %d  •  Users %d  •  Pressure %s", domain.HumanDuration(live.UptimeSeconds), live.ProcessCount, live.LoggedInUsers, statusPressureLabel(live)), width)),
	)
	if live.SwapTotalBytes > 0 {
		swapLine := fmt.Sprintf("Swap %.1f%%  •  used %s of %s", live.SwapUsedPercent, domain.HumanBytes(int64(live.SwapUsedBytes)), domain.HumanBytes(int64(live.SwapTotalBytes)))
		lines = append(lines, mutedStyle.Render(singleLine(swapLine, width)))
	}
	perfParts := make([]string, 0, 3)
	if live.Load1 > 0 {
		perfParts = append(perfParts, fmt.Sprintf("Load %.2f (%.2fx/core)", live.Load1, live.LoadPerCPU))
	}
	if live.PerformanceCores > 0 || live.EfficiencyCores > 0 {
		perfParts = append(perfParts, fmt.Sprintf("%dP+%dE cores", live.PerformanceCores, live.EfficiencyCores))
	}
	if len(live.CPUPerCore) > 0 {
		samples := make([]string, 0, min(len(live.CPUPerCore), 6))
		for _, value := range live.CPUPerCore[:min(len(live.CPUPerCore), 6)] {
			samples = append(samples, fmt.Sprintf("%.0f%%", value))
		}
		perfParts = append(perfParts, "Cores "+strings.Join(samples, " "))
	}
	if len(perfParts) > 0 {
		lines = append(lines, mutedStyle.Render(singleLine(strings.Join(perfParts, "  •  "), width)))
	}
	if len(live.TopProcesses) > 0 {
		lines = append(lines, headerStyle.Render("Top"))
		limit := 3
		if maxLines <= 9 {
			limit = 2
		}
		for _, proc := range live.TopProcesses[:min(len(live.TopProcesses), limit)] {
			lines = append(lines, singleLine(fmt.Sprintf("%s  •  cpu %.1f%%  •  mem %.1f%%  •  rss %s", proc.Name, proc.CPUPercent, proc.MemoryPercent, domain.HumanBytes(int64(proc.MemoryRSSBytes))), width))
		}
	}
	lines = viewportLines(lines, 0, maxLines)
	return strings.Join(lines, "\n")
}

func statusSystemViewWithTrends(model statusModel, width int, maxLines int) string {
	lines := []string{statusSystemView(model.live, width, maxLines)}
	if model.live != nil {
		lines = append(lines, mutedStyle.Render(statusHeroSceneLine(model, width)))
	}
	if len(model.cpuTrend) > 0 {
		lines = append(lines, mutedStyle.Render(statusTrendLine("CPU", model.cpuTrend, width)))
	}
	if len(model.memoryTrend) > 0 {
		lines = append(lines, mutedStyle.Render(statusTrendLine("Mem", model.memoryTrend, width)))
	}
	if len(model.diskTrend) > 0 {
		lines = append(lines, mutedStyle.Render(statusTrendLine("Disk I/O", model.diskTrend, width)))
	}
	lines = viewportLines(lines, 0, maxLines)
	return strings.Join(lines, "\n")
}

func statusStorageSubtitle(live *engine.SystemSnapshot) string {
	if live == nil {
		return "loading..."
	}
	return fmt.Sprintf("%.1f%% used  •  free %s", live.DiskUsedPercent, domain.HumanBytes(int64(live.DiskFreeBytes)))
}

func statusStorageView(model statusModel, width int, maxLines int) string {
	live := model.live
	if live == nil {
		return mutedStyle.Render("No storage telemetry.")
	}
	lines := []string{
		mutedStyle.Render(singleLine(fmt.Sprintf("Disk %.1f%% used  •  Free %s", live.DiskUsedPercent, domain.HumanBytes(int64(live.DiskFreeBytes))), width)),
	}
	if live.DiskIO != nil || model.diskReadRate > 0 || model.diskWriteRate > 0 {
		storageParts := make([]string, 0, 2)
		if live.DiskIO != nil {
			storageParts = append(storageParts, fmt.Sprintf("Disk I/O read %s  •  write %s", domain.HumanBytes(int64(live.DiskIO.ReadBytes)), domain.HumanBytes(int64(live.DiskIO.WriteBytes))))
		}
		if rate := statusRateLine("Disk rate", model.diskReadRate, model.diskWriteRate); rate != "" {
			storageParts = append(storageParts, rate)
		}
		lines = append(lines, mutedStyle.Render(singleLine(strings.Join(storageParts, "  •  "), width)))
	}
	if len(model.diskTrend) > 0 {
		lines = append(lines, mutedStyle.Render(statusTrendLine("Disk", model.diskTrend, width)))
	}
	lines = viewportLines(lines, 0, maxLines)
	return strings.Join(lines, "\n")
}

func statusPowerSubtitle(live *engine.SystemSnapshot) string {
	if live == nil {
		return "loading..."
	}
	parts := []string{}
	if live.Battery != nil {
		parts = append(parts, fmt.Sprintf("%.0f%% %s", live.Battery.Percent, strings.ToLower(live.Battery.State)))
	}
	if live.ThermalState != "" && !strings.EqualFold(live.ThermalState, "nominal") {
		parts = append(parts, "thermal "+strings.ToLower(live.ThermalState))
	}
	if live.SystemPowerWatts > 0 {
		parts = append(parts, fmt.Sprintf("%.0fW", live.SystemPowerWatts))
	}
	if len(parts) == 0 {
		return "nominal"
	}
	return strings.Join(parts, "  •  ")
}

func statusPowerView(model statusModel, width int, maxLines int) string {
	live := model.live
	if live == nil {
		return mutedStyle.Render("No power or network data.")
	}
	if width < 48 || maxLines < 6 {
		lines := []string{
			mutedStyle.Render(fmt.Sprintf("Net %s / %s", domain.HumanBytes(int64(live.NetworkRxBytes)), domain.HumanBytes(int64(live.NetworkTxBytes)))),
		}
		if live.Battery != nil {
			lines = append(lines, mutedStyle.Render(fmt.Sprintf("Battery %.0f%% %s", live.Battery.Percent, live.Battery.State)))
		}
		if summary := statusBatteryPowerSummary(live); summary != "" {
			lines = append(lines, mutedStyle.Render(summary))
		}
		if live.Proxy != nil {
			lines = append(lines, mutedStyle.Render(fmt.Sprintf("Proxy %t", live.Proxy.Enabled)))
		}
		lines = viewportLines(lines, 0, maxLines)
		return strings.Join(lines, "\n")
	}
	lines := []string{
		mutedStyle.Render(singleLine(statusNetworkOverviewLine(live, model.networkRxRate, model.networkTxRate), width)),
	}
	metaParts := make([]string, 0, 3)
	if ifaceSummary := statusInterfaceSummary(live); ifaceSummary != "" {
		metaParts = append(metaParts, ifaceSummary)
	}
	if live.VirtualizationSystem != "" {
		metaParts = append(metaParts, "Virtualization "+strings.TrimSpace(live.VirtualizationSystem+" "+live.VirtualizationRole))
	}
	if live.Proxy != nil {
		metaParts = append(metaParts, statusProxySummary(live.Proxy))
	}
	if len(metaParts) > 0 {
		lines = append(lines, mutedStyle.Render(singleLine(strings.Join(metaParts, "  •  "), width)))
	}
	powerParts := make([]string, 0, 2)
	if live.Battery != nil {
		battery := fmt.Sprintf("Battery %.0f%% %s", live.Battery.Percent, live.Battery.State)
		if live.Battery.RemainingMinutes > 0 {
			battery += fmt.Sprintf("  %d min remaining", live.Battery.RemainingMinutes)
		}
		if live.PowerSource != "" {
			battery += "  source " + live.PowerSource
		}
		powerParts = append(powerParts, battery)
	}
	if summary := statusBatteryPowerSummary(live); summary != "" {
		powerParts = append(powerParts, summary)
	}
	if len(powerParts) > 0 {
		lines = append(lines, mutedStyle.Render(singleLine(strings.Join(powerParts, "  •  "), width)))
	}
	btParts := make([]string, 0, 2)
	if bt := statusBluetoothSummary(live); bt != "" {
		btParts = append(btParts, bt)
	}
	if btDevices := statusBluetoothDevicesSummary(live, width); btDevices != "" {
		btParts = append(btParts, btDevices)
	}
	signalParts := make([]string, 0, 2)
	if gpu := statusGPUUsageSummary(live); gpu != "" {
		signalParts = append(signalParts, gpu)
	}
	if live.ThermalState != "" {
		thermal := "Thermal " + live.ThermalState
		if live.CPUTempCelsius > 0 {
			thermal += fmt.Sprintf("  •  %.1f°C", live.CPUTempCelsius)
		}
		signalParts = append(signalParts, thermal)
	}
	if len(signalParts) > 0 {
		lines = append(lines, mutedStyle.Render(singleLine(strings.Join(signalParts, "  •  "), width)))
	}
	if len(btParts) > 0 {
		lines = append(lines, mutedStyle.Render(singleLine(strings.Join(btParts, "  •  "), width)))
	}
	if live.PowerSource != "" && live.Battery == nil {
		lines = append(lines, mutedStyle.Render("Power source "+live.PowerSource))
	}
	tailParts := make([]string, 0, 2)
	if len(live.Highlights) > 0 {
		tailParts = append(tailParts, "Highlights "+live.Highlights[0])
	}
	if len(model.networkTrend) > 0 {
		tailParts = append(tailParts, statusTrendLine("Net", model.networkTrend, width))
	}
	if len(tailParts) > 0 {
		lines = append(lines, mutedStyle.Render(singleLine(strings.Join(tailParts, "  •  "), width)))
	}
	lines = viewportLines(lines, 0, maxLines)
	return strings.Join(lines, "\n")
}

func statusActivitySubtitle(scans []store.RecentScan, lastExecution *store.ExecutionSummary) string {
	if lastExecution == nil {
		if len(scans) == 0 {
			return "no scan history"
		}
		return fmt.Sprintf("%d %s", len(scans), pl(len(scans), "scan", "scans"))
	}
	return fmt.Sprintf("%d %s • %d deleted", len(scans), pl(len(scans), "scan", "scans"), lastExecution.Deleted)
}

func statusActivityView(scans []store.RecentScan, lastExecution *store.ExecutionSummary, width int, maxLines int) string {
	lines := []string{}
	if width < 48 || maxLines < 6 {
		if lastExecution != nil {
			lines = append(lines, mutedStyle.Render(fmt.Sprintf("Last run %s  •  %d deleted  •  %d failed", statusRelativeMoment(lastExecution.FinishedAt), lastExecution.Deleted, lastExecution.Failed)))
		}
		if len(scans) > 0 {
			lines = append(lines, mutedStyle.Render(statusCompactScanSummary(scans[0])))
		} else {
			lines = append(lines, mutedStyle.Render("No recent scans"))
		}
		lines = viewportLines(lines, 0, maxLines)
		return strings.Join(lines, "\n")
	}
	timelineLimit := 4
	includeRecentScans := true
	compactLayout := maxLines <= 8
	if compactLayout {
		timelineLimit = 1
		includeRecentScans = false
	}
	if timeline := statusTimelineLines(scans, lastExecution, width, timelineLimit); len(timeline) > 0 {
		lines = append(lines, headerStyle.Render("Recent"))
		lines = append(lines, timeline...)
	}
	if lastExecution != nil {
		if compactLayout {
			lines = append(lines, headerStyle.Render(singleLine(
				fmt.Sprintf("Last  •  Completed %d  Deleted %d  Failed %d", lastExecution.Completed, lastExecution.Deleted, lastExecution.Failed),
				width,
			)))
			for _, warning := range lastExecution.Warnings[:min(len(lastExecution.Warnings), 1)] {
				lines = append(lines, singleLine(mutedStyle.Render("Follow-up: "+warning), width))
			}
		} else {
			lines = append(lines, headerStyle.Render("Last"))
			summary := fmt.Sprintf("Completed %d  Deleted %d  Failed %d  Protected %d  Skipped %d", lastExecution.Completed, lastExecution.Deleted, lastExecution.Failed, lastExecution.Protected, lastExecution.Skipped)
			lines = append(lines, mutedStyle.Render(singleLine(summary, width)))
			followParts := make([]string, 0, 2)
			for _, warning := range lastExecution.Warnings[:min(len(lastExecution.Warnings), 1)] {
				followParts = append(followParts, "Follow-up: "+warning)
			}
			for _, followUp := range lastExecution.FollowUpCommands[:min(len(lastExecution.FollowUpCommands), 1)] {
				followParts = append(followParts, "Command: "+followUp)
			}
			if len(followParts) > 0 {
				lines = append(lines, singleLine(mutedStyle.Render(strings.Join(followParts, "  •  ")), width))
			}
		}
	}
	if len(scans) == 0 {
		lines = append(lines, mutedStyle.Render("No scan history yet. Run sift analyze or sift clean first."))
	} else if includeRecentScans {
		lines = append(lines, headerStyle.Render("History"))
		limit := 4
		if maxLines <= 10 {
			limit = 3
		}
		for _, scan := range scans[:min(len(scans), limit)] {
			profile := strings.TrimSpace(scan.Profile)
			if profile == "" {
				profile = "-"
			}
			label := strings.ToUpper(scan.Command)
			if profile != "-" {
				label += " / " + strings.ToUpper(profile)
			}
			lines = append(lines, singleLine(fmt.Sprintf("%s  •  %s  •  %d %s  •  %s", label, domain.HumanBytes(scan.Totals.Bytes), scan.Totals.ItemCount, pl(scan.Totals.ItemCount, "item", "items"), scan.CreatedAt.Local().Format("2006-01-02 15:04")), width))
		}
	}
	lines = viewportLines(lines, 0, maxLines)
	return strings.Join(lines, "\n")
}

func statusSessionCard(lastExecution *store.ExecutionSummary, scans []store.RecentScan) (string, string) {
	if lastExecution != nil {
		value := statusRelativeMoment(lastExecution.FinishedAt)
		if lastExecution.Deleted > 0 {
			value += fmt.Sprintf(" • %d deleted", lastExecution.Deleted)
		}
		tone := "safe"
		if lastExecution.Failed > 0 || lastExecution.Protected > 0 {
			tone = "high"
		} else if lastExecution.Deleted == 0 && lastExecution.Completed == 0 {
			tone = "review"
		}
		return value, tone
	}
	if len(scans) > 0 {
		return fmt.Sprintf("%s • %d %s", statusRelativeMoment(scans[0].CreatedAt), len(scans), pl(len(scans), "scan", "scans")), "review"
	}
	return "", ""
}

func statusRelativeMoment(at time.Time) string {
	if at.IsZero() {
		return "just now"
	}
	delta := time.Since(at)
	if delta < 0 {
		delta = 0
	}
	if delta < 48*time.Hour {
		return domain.HumanDuration(uint64(delta.Seconds())) + " ago"
	}
	return at.Local().Format("Jan 2 15:04")
}

func statusCompactScanSummary(scan store.RecentScan) string {
	label := strings.ToUpper(strings.TrimSpace(scan.Command))
	if profile := strings.TrimSpace(scan.Profile); profile != "" && profile != "-" {
		label += " / " + strings.ToUpper(profile)
	}
	return fmt.Sprintf("%s  •  %s  •  %s", label, domain.HumanBytes(scan.Totals.Bytes), statusRelativeMoment(scan.CreatedAt))
}

func statusTimelineLines(scans []store.RecentScan, lastExecution *store.ExecutionSummary, width int, limit int) []string {
	lines := []string{}
	if lastExecution != nil {
		label := statusExecutionLabel(lastExecution, scans)
		line := fmt.Sprintf("%s  %s  •  %d deleted  •  %d failed", statusTimestamp(lastExecution.FinishedAt), label, lastExecution.Deleted, lastExecution.Failed)
		if !lastExecution.StartedAt.IsZero() && !lastExecution.FinishedAt.IsZero() && lastExecution.FinishedAt.After(lastExecution.StartedAt) {
			line += "  •  " + domain.HumanDuration(uint64(lastExecution.FinishedAt.Sub(lastExecution.StartedAt).Seconds()))
		}
		lines = append(lines, singleLine(line, width))
	}
	for _, scan := range scans[:min(len(scans), limit)] {
		label := strings.ToUpper(strings.TrimSpace(scan.Command))
		if profile := strings.TrimSpace(scan.Profile); profile != "" && profile != "-" {
			label += " / " + strings.ToUpper(profile)
		}
		line := fmt.Sprintf("%s  %s  •  %s  •  %d %s", statusTimestamp(scan.CreatedAt), label, domain.HumanBytes(scan.Totals.Bytes), scan.Totals.ItemCount, pl(scan.Totals.ItemCount, "item", "items"))
		lines = append(lines, singleLine(line, width))
	}
	return lines
}

func statusExecutionLabel(lastExecution *store.ExecutionSummary, scans []store.RecentScan) string {
	label := "EXECUTION"
	if lastExecution == nil {
		return label
	}
	for _, scan := range scans {
		if scan.ScanID != "" && scan.ScanID == lastExecution.ScanID {
			label = strings.ToUpper(strings.TrimSpace(scan.Command))
			if profile := strings.TrimSpace(scan.Profile); profile != "" && profile != "-" {
				label += " / " + strings.ToUpper(profile)
			}
			return label
		}
	}
	return label
}

func statusTimestamp(at time.Time) string {
	if at.IsZero() {
		return "--:--"
	}
	return at.Local().Format("15:04")
}

// sparklineGlyph converts a 0-1 ratio to one of the 8 block-element glyphs.
func sparklineGlyph(ratio float64) rune {
	const glyphs = "▁▂▃▄▅▆▇█"
	r := []rune(glyphs)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return r[int(ratio*float64(len(r)-1))]
}

// renderSparkline builds a sparkline string from the last n values, normalised
// to the window maximum.
func renderSparkline(values []float64, n int) string {
	if len(values) == 0 {
		return ""
	}
	window := values
	if len(window) > n {
		window = window[len(window)-n:]
	}
	maxValue := 0.0
	for _, v := range window {
		if v > maxValue {
			maxValue = v
		}
	}
	if maxValue <= 0 {
		maxValue = 1
	}
	var b strings.Builder
	for _, v := range window {
		b.WriteRune(sparklineGlyph(v / maxValue))
	}
	return b.String()
}

func statusTrendLine(label string, values []float64, width int) string {
	if len(values) == 0 {
		return ""
	}
	window := values
	if len(window) > 14 {
		window = window[len(window)-14:]
	}
	line := fmt.Sprintf("%s %s  %.0f", label, renderSparkline(values, 14), window[len(window)-1])
	return truncateText(line, width)
}

func statusOverviewTrendLine(model statusModel, width int) string {
	parts := make([]string, 0, 4)
	if line := statusTrendGlyph("CPU", model.cpuTrend); line != "" {
		parts = append(parts, line)
	}
	if line := statusTrendGlyph("MEM", model.memoryTrend); line != "" {
		parts = append(parts, line)
	}
	if line := statusTrendGlyph("NET", model.networkTrend); line != "" {
		parts = append(parts, line)
	}
	if line := statusTrendGlyph("DISK", model.diskTrend); line != "" {
		parts = append(parts, line)
	}
	if len(parts) == 0 {
		return ""
	}
	return truncateText("Trends   "+strings.Join(parts, "  •  "), width)
}

func statusTrendGlyph(label string, values []float64) string {
	s := renderSparkline(values, 10)
	if s == "" {
		return ""
	}
	return label + " " + s
}

func statusHeroSceneLine(model statusModel, width int) string {
	live := model.live
	if live == nil {
		return truncateText("Sensors  waiting for telemetry", width)
	}
	motion := statusMotionState(model)
	parts := []string{
		"Sensors",
		statusHeroVectorLabel(live, motion),
		statusHeroVectorBands(live),
	}
	if summary := statusHeroSceneSummary(live); summary != "" {
		parts = append(parts, summary)
	}
	return truncateText(strings.Join(parts, "  •  "), width)
}

func statusHeroVectorLabel(live *engine.SystemSnapshot, motion motionState) string {
	switch {
	case motion.Mode == motionModeAlert:
		return "alert load"
	case statusPressureLabel(live) == "critical":
		return "heavy load"
	case live.Battery != nil && strings.EqualFold(live.PowerSource, "battery") && live.Battery.Percent > 0 && live.Battery.Percent < 30:
		return "battery drain"
	case live.GPUUsagePercent >= 50 || live.GPURendererPercent >= 50 || live.GPUTilerPercent >= 50:
		return "graphics load"
	default:
		return "steady load"
	}
}

func statusHeroVectorBands(live *engine.SystemSnapshot) string {
	if live == nil {
		return ""
	}
	bands := []string{
		statusHeroBand("C", live.CPUPercent/100),
		statusHeroBand("M", live.MemoryUsedPercent/100),
		statusHeroBand("G", statusHeroGPUPercent(live)/100),
		statusHeroBand("T", statusHeroThermalPercent(live)),
	}
	return strings.Join(bands, " ")
}

func statusHeroBand(label string, ratio float64) string {
	return label + string(sparklineGlyph(ratio))
}

func statusHeroGPUPercent(live *engine.SystemSnapshot) float64 {
	if live == nil {
		return 0
	}
	maxValue := live.GPUUsagePercent
	if live.GPURendererPercent > maxValue {
		maxValue = live.GPURendererPercent
	}
	if live.GPUTilerPercent > maxValue {
		maxValue = live.GPUTilerPercent
	}
	return maxValue
}

func statusHeroThermalPercent(live *engine.SystemSnapshot) float64 {
	if live == nil {
		return 0
	}
	if live.CPUTempCelsius > 0 {
		ratio := live.CPUTempCelsius / 100
		if ratio > 1 {
			ratio = 1
		}
		return ratio
	}
	switch strings.ToLower(strings.TrimSpace(live.ThermalState)) {
	case "critical":
		return 1
	case "serious", "heavy":
		return 0.8
	case "warm", "fair":
		return 0.6
	default:
		return 0.25
	}
}

func statusHeroSceneSummary(live *engine.SystemSnapshot) string {
	if live == nil {
		return ""
	}
	parts := make([]string, 0, 2)
	if live.ThermalState != "" && !strings.EqualFold(live.ThermalState, "nominal") {
		label := "thermal " + strings.ToLower(live.ThermalState)
		if live.CPUTempCelsius > 0 {
			label += fmt.Sprintf(" %.1f°C", live.CPUTempCelsius)
		}
		parts = append(parts, label)
	}
	if live.Battery != nil && strings.EqualFold(live.PowerSource, "battery") {
		parts = append(parts, fmt.Sprintf("battery %.0f%% %s", live.Battery.Percent, strings.ToLower(strings.TrimSpace(live.Battery.State))))
	} else if live.SystemPowerWatts > 0 {
		parts = append(parts, fmt.Sprintf("%.0fW system", live.SystemPowerWatts))
	}
	if len(parts) == 0 && len(live.OperatorAlerts) > 0 {
		return live.OperatorAlerts[0]
	}
	return strings.Join(parts, "  •  ")
}

func compactStatusAuxView(model statusModel, width int, maxLines int) string {
	live := model.live
	lines := []string{}
	if live == nil {
		return mutedStyle.Render("No storage or power data.")
	}
	lines = append(lines,
		mutedStyle.Render(fmt.Sprintf("Mem %.1f%%  •  Disk %.1f%%", live.MemoryUsedPercent, live.DiskUsedPercent)),
		mutedStyle.Render(fmt.Sprintf("Free %s  •  Net %s/%s", domain.HumanBytes(int64(live.DiskFreeBytes)), domain.HumanBytes(int64(live.NetworkRxBytes)), domain.HumanBytes(int64(live.NetworkTxBytes)))),
		mutedStyle.Render("Pressure "+statusPressureLabel(live)),
	)
	if live.DiskIO != nil {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("I/O %s read  •  %s write", domain.HumanBytes(int64(live.DiskIO.ReadBytes)), domain.HumanBytes(int64(live.DiskIO.WriteBytes)))))
	}
	if rate := statusRateLine("Rate", model.networkRxRate, model.networkTxRate); rate != "" {
		lines = append(lines, mutedStyle.Render(rate))
	}
	if len(model.cpuTrend) > 0 {
		lines = append(lines, mutedStyle.Render(statusTrendLine("CPU", model.cpuTrend, width)))
	}
	if len(model.networkTrend) > 0 {
		lines = append(lines, mutedStyle.Render(statusTrendLine("NET", model.networkTrend, width)))
	}
	batteryParts := make([]string, 0, 2)
	if live.Battery != nil {
		batteryParts = append(batteryParts, fmt.Sprintf("Battery %.0f%% %s", live.Battery.Percent, live.Battery.State))
	}
	if btDevices := statusBluetoothDevicesSummary(live, width); btDevices != "" {
		batteryParts = append(batteryParts, btDevices)
	} else if bt := statusBluetoothSummary(live); bt != "" {
		batteryParts = append(batteryParts, bt)
	}
	if len(batteryParts) > 0 {
		lines = append(lines, mutedStyle.Render(singleLine(strings.Join(batteryParts, "  •  "), width)))
	}
	if summary := statusCompactBatteryPowerSummary(live); summary != "" {
		lines = append(lines, mutedStyle.Render(summary))
	}
	if live.Proxy != nil && live.Proxy.Enabled {
		lines = append(lines, mutedStyle.Render("Proxy enabled"))
	}
	if len(live.Highlights) > 0 {
		lines = append(lines, mutedStyle.Render(truncateText(live.Highlights[0], width)))
	}
	lines = viewportLines(lines, 0, maxLines)
	return strings.Join(lines, "\n")
}

func statusRateLine(label string, readRate, writeRate float64) string {
	if readRate <= 0 && writeRate <= 0 {
		return ""
	}
	return fmt.Sprintf("%s ↓ %s/s  ↑ %s/s", label, domain.HumanBytes(int64(readRate)), domain.HumanBytes(int64(writeRate)))
}

func statusNetworkOverviewLine(live *engine.SystemSnapshot, rxRate, txRate float64) string {
	line := fmt.Sprintf("Network rx %s  •  tx %s", domain.HumanBytes(int64(live.NetworkRxBytes)), domain.HumanBytes(int64(live.NetworkTxBytes)))
	if rxRate > 0 || txRate > 0 {
		line += fmt.Sprintf("  •  Net rate ↓ %s/s  ↑ %s/s", domain.HumanBytes(int64(rxRate)), domain.HumanBytes(int64(txRate)))
	}
	return line
}

func statusProxySummary(proxy *engine.ProxySnapshot) string {
	if proxy == nil {
		return ""
	}
	line := fmt.Sprintf("Proxy enabled=%t", proxy.Enabled)
	if proxy.HTTP != "" {
		line += "  http " + proxy.HTTP
	}
	if proxy.HTTPS != "" {
		line += "  https " + proxy.HTTPS
	}
	if proxy.Bypass != "" {
		line += "  bypass " + proxy.Bypass
	}
	return line
}

func statusPowerMetricsSummary(live *engine.SystemSnapshot) string {
	if live == nil {
		return ""
	}
	parts := make([]string, 0, 5)
	if live.SystemPowerWatts > 0 {
		parts = append(parts, fmt.Sprintf("%.0fW system", live.SystemPowerWatts))
	}
	if live.AdapterPowerWatts > 0 {
		parts = append(parts, fmt.Sprintf("%.0fW adapter", live.AdapterPowerWatts))
	}
	if live.BatteryPowerWatts > 0 {
		parts = append(parts, fmt.Sprintf("%.0fW battery", live.BatteryPowerWatts))
	} else if live.BatteryPowerWatts < 0 {
		parts = append(parts, fmt.Sprintf("%.0fW charge", -live.BatteryPowerWatts))
	}
	if live.FanSpeedRPM > 0 {
		parts = append(parts, fmt.Sprintf("%d RPM fan", live.FanSpeedRPM))
	}
	if live.CPUTempCelsius > 0 {
		parts = append(parts, fmt.Sprintf("%.1f°C", live.CPUTempCelsius))
	}
	return strings.Join(parts, "  •  ")
}

func statusBatteryPowerSummary(live *engine.SystemSnapshot) string {
	if live == nil {
		return ""
	}
	parts := make([]string, 0, 6)
	if power := statusPowerMetricsSummary(live); power != "" {
		parts = append(parts, power)
	}
	if live.Battery != nil {
		if live.Battery.Condition != "" {
			parts = append(parts, "Health "+live.Battery.Condition)
		}
		if live.Battery.CycleCount > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", live.Battery.CycleCount, pl(live.Battery.CycleCount, "cycle", "cycles")))
		}
		if live.Battery.CapacityPercent > 0 {
			parts = append(parts, fmt.Sprintf("%d%% capacity", live.Battery.CapacityPercent))
		}
	}
	return strings.Join(parts, "  •  ")
}

func statusCompactBatteryPowerSummary(live *engine.SystemSnapshot) string {
	if live == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	if live.SystemPowerWatts > 0 {
		parts = append(parts, fmt.Sprintf("%.0fW system", live.SystemPowerWatts))
	}
	if live.BatteryPowerWatts > 0 {
		parts = append(parts, fmt.Sprintf("%.0fW battery", live.BatteryPowerWatts))
	} else if live.BatteryPowerWatts < 0 {
		parts = append(parts, fmt.Sprintf("%.0fW charge", -live.BatteryPowerWatts))
	} else if live.AdapterPowerWatts > 0 {
		parts = append(parts, fmt.Sprintf("%.0fW adapter", live.AdapterPowerWatts))
	}
	if live.Battery != nil {
		if live.Battery.CycleCount > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", live.Battery.CycleCount, pl(live.Battery.CycleCount, "cycle", "cycles")))
		} else if live.Battery.Condition != "" {
			parts = append(parts, "Health "+live.Battery.Condition)
		}
	}
	if live.GPUUsagePercent > 0 {
		parts = append(parts, fmt.Sprintf("GPU %.0f%%", live.GPUUsagePercent))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "  •  ")
}

func statusHostSummary(live *engine.SystemSnapshot) string {
	parts := []string{}
	if live.Platform != "" {
		parts = append(parts, strings.ToUpper(live.Platform))
	}
	if live.PlatformFamily != "" {
		parts = append(parts, live.PlatformFamily)
	}
	if live.Architecture != "" {
		parts = append(parts, strings.ToUpper(live.Architecture))
	}
	switch {
	case live.PerformanceCores > 0 || live.EfficiencyCores > 0:
		parts = append(parts, fmt.Sprintf("%dP+%dE cores", live.PerformanceCores, live.EfficiencyCores))
	case live.CPUPhysicalCores > 0 && live.CPUCores > 0:
		parts = append(parts, fmt.Sprintf("%dp/%dl cores", live.CPUPhysicalCores, live.CPUCores))
	case live.CPUCores > 0:
		parts = append(parts, fmt.Sprintf("%d %s", live.CPUCores, pl(live.CPUCores, "core", "cores")))
	}
	if live.VirtualizationSystem != "" {
		label := live.VirtualizationSystem
		if live.VirtualizationRole != "" {
			label += " " + live.VirtualizationRole
		}
		parts = append(parts, strings.TrimSpace(label))
	}
	return strings.Join(parts, "  •  ")
}

func statusHardwareSummary(live *engine.SystemSnapshot) string {
	parts := []string{}
	if live.HardwareModel != "" {
		parts = append(parts, live.HardwareModel)
	}
	if live.CPUModel != "" {
		parts = append(parts, live.CPUModel)
	}
	if live.GPUModel != "" && !strings.EqualFold(strings.TrimSpace(live.GPUModel), strings.TrimSpace(live.CPUModel)) {
		gpu := "gpu " + live.GPUModel
		if live.GPUUsagePercent > 0 {
			gpu += fmt.Sprintf(" %.0f%%", live.GPUUsagePercent)
		}
		parts = append(parts, gpu)
	}
	if live.DisplayResolution != "" || live.DisplayRefreshRate != "" {
		display := "display"
		if live.DisplayCount > 1 {
			display += fmt.Sprintf(" %dx", live.DisplayCount)
		}
		switch {
		case live.DisplayResolution != "" && live.DisplayRefreshRate != "":
			display += " " + live.DisplayResolution + " @" + live.DisplayRefreshRate
		case live.DisplayResolution != "":
			display += " " + live.DisplayResolution
		case live.DisplayRefreshRate != "":
			display += " " + live.DisplayRefreshRate
		}
		parts = append(parts, display)
	}
	if live.KernelVersion != "" {
		parts = append(parts, "kernel "+live.KernelVersion)
	}
	if live.BootTimeSeconds > 0 {
		parts = append(parts, "boot "+time.Unix(int64(live.BootTimeSeconds), 0).Local().Format("2006-01-02 15:04"))
	}
	return strings.Join(parts, "  •  ")
}

func statusInterfaceSummary(live *engine.SystemSnapshot) string {
	if live == nil {
		return ""
	}
	switch {
	case len(live.ActiveNetworkIfaces) > 0 && live.NetworkInterfaceCount > 0:
		return fmt.Sprintf("Interfaces %s  •  %d total", strings.Join(live.ActiveNetworkIfaces, ", "), live.NetworkInterfaceCount)
	case len(live.ActiveNetworkIfaces) > 0:
		return "Interfaces " + strings.Join(live.ActiveNetworkIfaces, ", ")
	case live.NetworkInterfaceCount > 0:
		return fmt.Sprintf("Interfaces %d total", live.NetworkInterfaceCount)
	default:
		return ""
	}
}

func statusBluetoothSummary(live *engine.SystemSnapshot) string {
	if live == nil {
		return ""
	}
	switch {
	case live.BluetoothPowered && live.BluetoothConnected > 0:
		return fmt.Sprintf("Bluetooth on  •  %d connected", live.BluetoothConnected)
	case live.BluetoothPowered:
		return "Bluetooth on"
	case live.BluetoothConnected > 0:
		return fmt.Sprintf("Bluetooth  •  %d connected", live.BluetoothConnected)
	default:
		return ""
	}
}

func statusBluetoothDevicesSummary(live *engine.SystemSnapshot, width int) string {
	if live == nil || len(live.BluetoothDevices) == 0 {
		return ""
	}
	parts := make([]string, 0, 2)
	for _, device := range live.BluetoothDevices {
		if !device.Connected {
			continue
		}
		label := strings.TrimSpace(device.Name)
		if label == "" {
			continue
		}
		if device.Battery != "" {
			label += " (" + device.Battery + ")"
		}
		parts = append(parts, label)
		if len(parts) == 2 {
			break
		}
	}
	if len(parts) == 0 {
		return ""
	}
	line := "Devices " + strings.Join(parts, ", ")
	if connected := live.BluetoothConnected - len(parts); connected > 0 {
		line += fmt.Sprintf("  •  +%d more", connected)
	}
	return truncateText(line, width)
}

func statusGPUUsageSummary(live *engine.SystemSnapshot) string {
	if live == nil || (live.GPUUsagePercent <= 0 && live.GPURendererPercent <= 0 && live.GPUTilerPercent <= 0) {
		return ""
	}
	parts := make([]string, 0, 3)
	if live.GPUUsagePercent > 0 {
		parts = append(parts, fmt.Sprintf("GPU %.0f%%", live.GPUUsagePercent))
	}
	if live.GPURendererPercent > 0 {
		parts = append(parts, fmt.Sprintf("render %.0f%%", live.GPURendererPercent))
	}
	if live.GPUTilerPercent > 0 {
		parts = append(parts, fmt.Sprintf("tiler %.0f%%", live.GPUTilerPercent))
	}
	return strings.Join(parts, "  •  ")
}

func statusPressureLabel(live *engine.SystemSnapshot) string {
	switch {
	case live == nil:
		return "unknown"
	case live.MemoryUsedPercent >= 85 || live.DiskUsedPercent >= 92 || live.LoadPerCPU >= 1.2:
		return "critical"
	case live.MemoryUsedPercent >= 70 || live.DiskUsedPercent >= 82 || live.LoadPerCPU >= 0.9 || live.SwapUsedPercent >= 10:
		return "watch"
	default:
		return "steady"
	}
}

func statusPressureBand(label string, value float64, watch float64, critical float64) string {
	state := "steady"
	if value >= critical {
		state = "critical"
	} else if value >= watch {
		state = "watch"
	}
	return fmt.Sprintf("%s %s", label, state)
}

func toneForPressure(label string) string {
	switch label {
	case "critical":
		return "high"
	case "watch":
		return "review"
	default:
		return "safe"
	}
}

func toneForPercent(value float64, watch float64, critical float64) string {
	if value >= critical {
		return "high"
	}
	if value >= watch {
		return "review"
	}
	return "safe"
}

// mascotFrameFromMotion delegates to the canonical mascotFrame renderer.
// Used by progress and home views that pass a generic motionState.
func mascotFrameFromMotion(motion motionState, cpuPercent float64) string {
	return mascotFrame(motion, cpuPercent)
}

// statusMascotFrame delegates to the canonical mascotFrame renderer using the
// status-model motion state and live CPU telemetry.
func statusMascotFrame(model statusModel) string {
	cpuPercent := 0.0
	if model.live != nil {
		cpuPercent = model.live.CPUPercent
	}
	return mascotFrame(statusMotionState(model), cpuPercent)
}

// statusOverviewHealthStyle returns a lipgloss style toned by the current
// health score so the "Now" line instantly signals system state at a glance.
func statusOverviewHealthStyle(live *engine.SystemSnapshot) lipgloss.Style {
	if live == nil {
		return mutedStyle
	}
	switch toneForHealth(live.HealthScore) {
	case "safe":
		return safeStyle
	case "high":
		return highStyle
	default:
		return reviewStyle
	}
}

// statusOverviewAlertStyle returns a style toned by alert severity so the
// "Risk" line immediately communicates whether action is needed.
func statusOverviewAlertStyle(model statusModel) lipgloss.Style {
	switch statusAlertTone(model.live, model.diagnostics, model.updateNotice) {
	case "high":
		return highStyle
	case "review":
		return reviewStyle
	default:
		return mutedStyle
	}
}
