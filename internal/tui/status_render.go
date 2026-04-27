package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/store"
)

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
		signature := routeSignalSignatureForRoute("status")
		lead := signature.Mascot
		if signature.Doctrine != "" {
			lead += "  •  " + signature.Doctrine
		}
		lines = append(lines, mutedStyle.Render(singleLine(fmt.Sprintf("%s  •  Observatory %s  •  %s", lead, signalRailLabelForMotion(motion), statusCompanionLabel(model)), width)))
		lines = append(lines, mutedStyle.Render(singleLine("Status   "+statusOverviewStatusLine(model), width)))
		lines = append(lines, mutedStyle.Render(singleLine(statusWatchLine(model, width), width)))
		if maxLines >= 5 {
			lines = append(lines, mutedStyle.Render(singleLine(statusRecentLine(model, width), width)))
		}
		if maxLines >= 6 {
			lines = append(lines, mutedStyle.Render(singleLine("Next     "+statusOverviewNextLine(model), width)))
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
	signature := routeSignalSignatureForRoute("status")
	lead := signature.Mascot
	if signature.Doctrine != "" {
		lead += "  •  " + signature.Doctrine
	}
	lines = append(lines, mutedStyle.Render(singleLine(lead+"  •  Observatory "+signalRailLabelForMotion(motion)+"  •  "+statusCompanionLabel(model), textWidth)))
	lines = append(lines, statusOverviewHealthStyle(live).Render(singleLine("Status   "+statusOverviewStatusLine(model), textWidth)))
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
	lines = append(lines, mutedStyle.Render(singleLine("Next     "+statusOverviewNextLine(model), textWidth)))
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

func statusSystemView(live *engine.SystemSnapshot, width int, maxLines int) string {
	if live == nil {
		return mutedStyle.Render("No system metrics yet.")
	}
	if width < 48 || maxLines < 7 {
		lines := []string{
			mutedStyle.Render(singleLine("Status   "+statusSystemStatusLine(live), width)),
			mutedStyle.Render(fmt.Sprintf("CPU %.1f%%  •  MEM %.1f%%", live.CPUPercent, live.MemoryUsedPercent)),
			mutedStyle.Render(fmt.Sprintf("Free %s  •  Proc %d  •  Pressure %s", domain.HumanBytes(int64(live.DiskFreeBytes)), live.ProcessCount, statusPressureLabel(live))),
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
		nextLine := "Next     " + statusSystemNextLine(live)
		if live.Load1 > 0 {
			nextLine += fmt.Sprintf("  •  Load %.2f", live.Load1)
		}
		lines = append(lines, mutedStyle.Render(singleLine(nextLine, width)))
		lines = viewportLines(lines, 0, maxLines)
		return strings.Join(lines, "\n")
	}
	lines := []string{
		mutedStyle.Render(singleLine("Status   "+statusSystemStatusLine(live), width)),
		mutedStyle.Render(singleLine("Next     "+statusSystemNextLine(live), width)),
	}
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
		if maxLines > 10 || len(live.TopProcesses) == 0 {
			lines = append(lines, mutedStyle.Render(singleLine(strings.Join(perfParts, "  •  "), width)))
		}
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
	if len(perfParts) > 0 && maxLines <= 10 && len(live.TopProcesses) > 0 {
		lines = append(lines, mutedStyle.Render(singleLine(strings.Join(perfParts, "  •  "), width)))
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

func statusStorageView(model statusModel, width int, maxLines int) string {
	live := model.live
	if live == nil {
		return mutedStyle.Render("No storage telemetry.")
	}
	lines := []string{
		mutedStyle.Render(singleLine("Status   "+statusStorageStatusLine(model)+"  •  Next "+statusStorageNextLine(model), width)),
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

func statusPowerView(model statusModel, width int, maxLines int) string {
	live := model.live
	if live == nil {
		return mutedStyle.Render("No power or network data.")
	}
	if width < 48 || maxLines < 6 {
		lines := []string{
			mutedStyle.Render(singleLine("Status   "+statusPowerStatusLine(live), width)),
			mutedStyle.Render(singleLine("Next     "+statusPowerNextLine(live), width)),
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
		mutedStyle.Render(singleLine("Status   "+statusPowerStatusLine(live), width)),
		mutedStyle.Render(singleLine("Next     "+statusPowerNextLine(live), width)),
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

func statusActivityView(scans []store.RecentScan, lastExecution *store.ExecutionSummary, width int, maxLines int) string {
	lines := []string{
		mutedStyle.Render(singleLine("Status   "+statusActivityStatusLine(scans, lastExecution), width)),
		mutedStyle.Render(singleLine("Next     "+statusActivityNextLine(scans, lastExecution), width)),
	}
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
