package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m cleanFlowModel) View() string {
	width, height := effectiveSize(m.width, m.height)
	compact := width < 118 || height < 28
	bodyWidth := width - 4
	leftWidth, rightWidth := splitColumns(bodyWidth, 0.72, 70, 28)
	panelLines := bodyLineBudget(height, 12, 8)
	left := renderPanel("Sweep lanes", cleanFlowSubtitle(m), renderCleanLedger(m, leftWidth-4, panelLines), leftWidth, true)
	right := renderPanel("Sweep deck", cleanFlowGateLine(m), renderCleanFocusPanel(m, rightWidth-4, panelLines), rightWidth, false)
	body := left
	if compact {
		ledgerLines, deckLines := cleanFlowCompactPanelLines(m, panelLines)
		body = strings.Join([]string{
			renderPanel("Sweep lanes", cleanFlowSubtitle(m), renderCleanLedger(m, width-8, ledgerLines), width-4, true),
			renderPanel("Sweep deck", cleanFlowGateLine(m), renderCleanFocusPanel(m, width-8, deckLines), width-4, false),
		}, "\n")
	} else {
		body = joinPanels(left, right, width-4)
	}
	stats := []string(nil)
	if compact {
		stats = cleanFlowStats(m, width)
	}
	return renderChrome("SIFT / Clean", "forge sweep rail", stats, body, nil, width, false, m.height)
}

func cleanFlowCompactPanelLines(m cleanFlowModel, panelLines int) (int, int) {
	ledgerLines := max(panelLines/2, 6)
	deckLines := max(panelLines/2, 6)
	if !m.autoFollow && m.scrollOffset > 0 {
		ledgerLines = max(panelLines-3, 8)
		deckLines = max(panelLines/3, 4)
	}
	return ledgerLines, deckLines
}

func cleanFlowSubtitle(m cleanFlowModel) string {
	hold := ""
	if !m.autoFollow && m.scrollOffset > 0 {
		hold = " • scroll hold"
	}
	switch m.phase {
	case cleanFlowScanning:
		return "scan first" + hold
	case cleanFlowReviewReady:
		return "frozen review" + hold
	case cleanFlowPermissions:
		return "access check" + hold
	case cleanFlowReclaiming:
		return "reclaiming now" + hold
	case cleanFlowResult:
		return "settled result" + hold
	default:
		return "controller ready" + hold
	}
}

func renderCleanLedger(m cleanFlowModel, width int, maxLines int) string {
	lines := []string{}
	lines = append(lines, renderRoutePinnedBlock("clean", cleanFlowSignalLabel(m), cleanFlowSignalMotion(m), width, cleanFlowFreezeBanner(m), cleanFlowHistoryBanner(m))...)
	lines = append(lines, cleanFlowPhaseTrackLines(m, width)...)
	for _, lane := range cleanFlowLanes(m) {
		lines = append(lines, cleanFlowLaneHeaderLines(lane, width)...)
		for _, row := range lane.Rows {
			size := "pending"
			if row.Bytes > 0 {
				size = cleanFlowHumanBytes(row.Bytes)
			}
			lines = append(lines, cleanFlowLedgerLine(m, row, width, size))
			if detail := cleanFlowLedgerDetail(row, width); detail != "" {
				lines = append(lines, detail)
			}
		}
		lines = append(lines, "")
	}
	if len(lines) == 0 {
		lines = append(lines, mutedStyle.Render("No cleanup lanes available."))
	}
	return strings.Join(cleanFlowViewportLines(m, lines, maxLines), "\n")
}

func cleanFlowLaneHeaderLines(lane cleanFlowLane, width int) []string {
	label, tone := cleanFlowLaneTelemetry(lane)
	head := []string{
		tone.Render(label),
		headerStyle.Render(lane.Label),
	}
	lines := []string{singleLine(strings.Join(head, "  "), max(width, 18))}
	if summary := cleanFlowLaneSummaryLine(lane, width); summary != "" {
		lines = append(lines, summary)
	}
	return lines
}

func cleanFlowLaneSummaryLine(lane cleanFlowLane, width int) string {
	parts := []string{fmt.Sprintf("%d %s", len(lane.Rows), pl(len(lane.Rows), "item", "items"))}
	if lane.Bytes > 0 {
		parts = append(parts, cleanFlowHumanBytes(lane.Bytes))
	}
	parts = append(parts, cleanFlowLaneMixSummary(lane)...)
	return panelMetaStyle.Render(singleLine(strings.Join(parts, " • "), max(width, 18)))
}

func cleanFlowLaneMixSummary(lane cleanFlowLane) []string {
	live := 0
	queued := 0
	watch := 0
	settled := 0
	for _, row := range lane.Rows {
		switch row.State {
		case "scanning", "reclaiming", "verifying":
			live++
		case "ready", "focus", "review", "queued":
			queued++
		case "protected", "failed":
			watch++
		case "settled":
			settled++
		}
	}
	parts := []string{}
	if live > 0 {
		parts = append(parts, fmt.Sprintf("%d live", live))
	}
	if queued > 0 {
		parts = append(parts, fmt.Sprintf("%d queued", queued))
	}
	if watch > 0 {
		parts = append(parts, fmt.Sprintf("%d watch", watch))
	}
	if settled > 0 {
		parts = append(parts, fmt.Sprintf("%d settled", settled))
	}
	return parts
}

func cleanFlowPhaseTrackLines(m cleanFlowModel, width int) []string {
	segments := []string{}
	current := cleanFlowSignalLabel(m)
	for _, phase := range []string{"SCAN RAIL", "REVIEW RAIL", "ACCESS RAIL", "RECLAIM RAIL", "SETTLED RAIL"} {
		label := routeStickyPhaseMeter("clean", phase)
		if label == "" {
			continue
		}
		style := panelMetaStyle
		if phase == current {
			style = headerStyle
		}
		segments = append(segments, style.Render(label))
	}
	return cleanFlowSegmentLines(segments, width)
}

func cleanFlowSegmentLines(segments []string, width int) []string {
	if len(segments) == 0 {
		return nil
	}
	const sep = "  •  "
	maxWidth := max(width-2, 24)
	lines := []string{}
	current := ""
	currentWidth := 0
	for _, segment := range segments {
		segmentWidth := ansi.StringWidth(ansi.Strip(segment))
		if current == "" {
			current = segment
			currentWidth = segmentWidth
			continue
		}
		if currentWidth+len(sep)+segmentWidth > maxWidth {
			lines = append(lines, current)
			current = segment
			currentWidth = segmentWidth
			continue
		}
		current += sep + segment
		currentWidth += len(sep) + segmentWidth
	}
	if current != "" {
		lines = append(lines, current)
	}
	lines = append(lines, "")
	return lines
}

func cleanFlowHistoryBanner(m cleanFlowModel) string {
	if m.autoFollow || m.scrollOffset <= 0 {
		return ""
	}
	parts := []string{
		reviewTokenStyle.Render("HISTORY HOLD"),
		panelMetaStyle.Render("manual review of recent sweep rows"),
		panelMetaStyle.Render("End returns live"),
	}
	return strings.Join(parts, "  •  ")
}

func cleanFlowFreezeBanner(m cleanFlowModel) string {
	switch m.phase {
	case cleanFlowReviewReady:
		parts := []string{
			reviewTokenStyle.Render("REVIEW FREEZE"),
			panelMetaStyle.Render("scan carry locked for gate review"),
		}
		return strings.Join(parts, "  •  ")
	case cleanFlowPermissions:
		parts := []string{
			reviewTokenStyle.Render("ACCESS HOLD"),
			panelMetaStyle.Render("scan carry held while access checks settle"),
		}
		return strings.Join(parts, "  •  ")
	default:
		return ""
	}
}

func cleanFlowLaneTelemetry(lane cleanFlowLane) (string, lipgloss.Style) {
	for _, row := range lane.Rows {
		switch row.State {
		case "reclaiming", "verifying":
			return "RECLAIM LANE", reviewTokenStyle
		case "scanning":
			return "SCAN LANE", reviewTokenStyle
		case "ready", "focus", "review":
			return "REVIEW LANE", safeTokenStyle
		case "protected", "failed":
			return "WATCH LANE", highTokenStyle
		}
	}
	return "SETTLED LANE", panelTitleStyle
}

func cleanFlowLedgerLine(m cleanFlowModel, row cleanFlowRow, width int, size string) string {
	labelWidth := max(width-34, 12)
	marker, lineStyle, stateStyle := cleanFlowRowStyles(m, row.State)
	tag, tagStyle := cleanFlowRowChrome(row.State)
	line := fmt.Sprintf("%s %-8s %-*s %10s  %s", marker, tagStyle.Render(tag), labelWidth, truncateText(row.Label, labelWidth), size, stateStyle.Render(cleanFlowLedgerStateLabel(row.State)))
	return lineStyle.Render(line)
}

func cleanFlowRowChrome(state string) (string, lipgloss.Style) {
	switch state {
	case "scanning", "reclaiming":
		return "LIVE", safeTokenStyle
	case "verifying":
		return "VERIFY", panelTitleStyle
	case "ready", "focus", "review", "queued":
		return "GATE", reviewTokenStyle
	case "protected", "failed":
		return "WATCH", highTokenStyle
	case "settled", "skipped":
		return "ARCHIVE", mutedStyle
	default:
		return "TRACE", mutedStyle
	}
}

func cleanFlowLedgerDetail(row cleanFlowRow, width int) string {
	if strings.TrimSpace(row.Path) == "" {
		return ""
	}
	switch row.State {
	case "scanning", "reclaiming", "verifying", "ready", "focus", "review":
	default:
		return ""
	}
	labelWidth := max(width-8, 12)
	return mutedStyle.Render("   path " + truncateText(row.Path, labelWidth))
}

func cleanFlowLedgerStateLabel(state string) string {
	switch state {
	case "scanning":
		return "ACTIVE SCAN"
	case "reclaiming":
		return "RECLAIM LIVE"
	case "verifying":
		return "VERIFY PASS"
	case "ready", "focus", "review":
		return "REVIEW READY"
	case "settled":
		return "SETTLED HOLD"
	case "queued":
		return "QUEUED HOLD"
	case "protected":
		return "PROTECTED"
	case "failed":
		return "FAILED"
	case "skipped":
		return "SKIPPED"
	default:
		return strings.ToUpper(state)
	}
}

func cleanFlowRowStyles(m cleanFlowModel, state string) (string, lipgloss.Style, lipgloss.Style) {
	switch state {
	case "scanning":
		return cleanFlowActiveMarker(m, safeTokenStyle), lipgloss.NewStyle().Bold(true), safeTokenStyle
	case "reclaiming":
		return cleanFlowActiveMarker(m, reviewTokenStyle), lipgloss.NewStyle().Bold(true), reviewTokenStyle
	case "verifying":
		return cleanFlowVerifyMarker(m), lipgloss.NewStyle().Bold(true), panelTitleStyle
	case "ready", "focus":
		return reviewStyle.Render("*"), lipgloss.NewStyle().Bold(true), reviewTokenStyle
	case "review":
		return reviewStyle.Render("-"), lipgloss.NewStyle(), reviewTokenStyle
	case "protected":
		return highStyle.Render("-"), lipgloss.NewStyle(), highTokenStyle
	case "failed":
		return highStyle.Render("x"), lipgloss.NewStyle().Bold(true), highTokenStyle
	case "skipped":
		return mutedStyle.Render("~"), mutedStyle, mutedStyle
	case "settled":
		return safeStyle.Render("+"), mutedStyle, safeTokenStyle
	default:
		return mutedStyle.Render("•"), lipgloss.NewStyle(), mutedStyle
	}
}

func cleanFlowActiveMarker(m cleanFlowModel, style lipgloss.Style) string {
	if m.reducedMotion || len(spinnerFrames) == 0 {
		return style.Render("*")
	}
	frame := spinnerFrames[m.spinnerFrame%len(spinnerFrames)]
	return style.Render(frame)
}

func cleanFlowVerifyMarker(m cleanFlowModel) string {
	if m.reducedMotion {
		return panelTitleStyle.Render("-")
	}
	if m.pulse {
		return panelTitleStyle.Render("+")
	}
	return panelTitleStyle.Render("-")
}

func renderCleanFocusPanel(m cleanFlowModel, width int, maxLines int) string {
	safeBytes, reviewBytes, totalBytes := cleanFlowByteMix(m)
	action, _ := m.selectedAction()
	accessLine := cleanFlowAccessLine(m)
	outcomeLine := cleanFlowOutcomeLine(m)
	lines := []string{
		renderCleanSummaryRail(m),
	}
	if accessLine != "" {
		lines = append(lines, mutedStyle.Render(singleLine("Access   "+accessLine, width)))
	}
	if outcomeLine != "" {
		lines = append(lines, mutedStyle.Render(singleLine("Outcome  "+outcomeLine, width)))
	}
	lines = append(lines,
		mutedStyle.Render(singleLine("Status   "+cleanFlowStatusLine(m), width)),
		mutedStyle.Render(singleLine("Next     "+cleanFlowNextLine(m), width)),
	)
	lines = append(lines,
		renderSectionRule(width),
		renderToneBadge(action.Tone)+" "+headerStyle.Render(cleanFlowFocusTitle(m)),
		wrapText(cleanFlowFocusCopy(m), width),
		"",
		renderSectionRule(width),
		headerStyle.Render("Sweep signal"),
		renderCleanSweepSignal(m),
		"",
	)
	if current, ok := cleanFlowCurrentSweepEntry(m); ok {
		carry := routeDeckHint("clean", "HOT PATH")
		if lane := cleanFlowDeckLane(current.Meta); lane != "" {
			carry = lane + "  •  " + carry
		}
		lines = append(lines,
			renderSectionRule(width),
			headerStyle.Render("Current sweep"),
			renderCleanDeckTag("HOT PATH", current.Meta),
			wrapText(current.Label, width),
			mutedStyle.Render(carry),
		)
	}
	if next, ok := cleanFlowNextReclaimEntry(m); ok {
		carry := routeDeckHint("clean", "NEXT PASS")
		if lane := cleanFlowDeckLane(next.Meta); lane != "" {
			carry = lane + "  •  " + carry
		}
		lines = append(lines,
			renderSectionRule(width),
			headerStyle.Render("Next reclaim"),
			renderCleanDeckTag("NEXT PASS", next.Meta),
			wrapText(next.Label, width),
			mutedStyle.Render(carry),
		)
	}
	lines = append(lines, "",
		mutedStyle.Render("Sweep    ")+"item-first scan and reclaim surface",
		mutedStyle.Render("Gate     ")+cleanFlowGateLine(m),
	)
	if totalBytes > 0 {
		lines = append(lines, mutedStyle.Render("Yield    ")+cleanFlowHumanBytes(totalBytes))
	}
	if safeBytes > 0 {
		lines = append(lines, mutedStyle.Render("Clear    ")+cleanFlowHumanBytes(safeBytes))
	}
	if reviewBytes > 0 {
		lines = append(lines, mutedStyle.Render("Watch    ")+cleanFlowHumanBytes(reviewBytes))
	}
	if action.Command != "" {
		lines = append(lines, mutedStyle.Render("Next     ")+menuNextActionLine(action))
	}
	return strings.Join(viewportLines(lines, 0, maxLines), "\n")
}

func renderCleanSummaryRail(m cleanFlowModel) string {
	safeBytes, reviewBytes, totalBytes := cleanFlowByteMix(m)
	parts := []string{
		renderCleanSummaryToken("SWEEP", cleanFlowPhaseCardValue(m)),
		renderCleanSummaryToken("CLEAR", cleanFlowHumanBytes(safeBytes)),
		renderCleanSummaryToken("WATCH", cleanFlowHumanBytes(reviewBytes)),
		renderCleanSummaryToken("YIELD", cleanFlowHumanBytes(totalBytes)),
	}
	return strings.Join(parts, "   ")
}

func renderCleanSummaryToken(label string, value string) string {
	return headerStyle.Render(label) + " " + mutedStyle.Render(value)
}

func renderCleanSweepSignal(m cleanFlowModel) string {
	summary := cleanFlowSignalLabel(m)
	extras := []string{}
	if current, ok := cleanFlowCurrentSweepEntry(m); ok && current.Meta != "" {
		extras = append(extras, current.Meta)
	}
	return renderRouteSignalBlock("clean", cleanFlowSignalMotion(m), cleanFlowSignalLoad(m), summary, extras...)
}

func cleanFlowSignalLabel(m cleanFlowModel) string {
	switch m.phase {
	case cleanFlowScanning:
		return "SCAN RAIL"
	case cleanFlowReviewReady:
		return "REVIEW RAIL"
	case cleanFlowPermissions:
		return "ACCESS RAIL"
	case cleanFlowReclaiming:
		return "RECLAIM RAIL"
	case cleanFlowResult:
		return "SETTLED RAIL"
	default:
		return "CLEAN RAIL"
	}
}

func cleanFlowSignalMotion(m cleanFlowModel) motionState {
	mode := motionModeLoading
	phase := "scan"
	switch m.phase {
	case cleanFlowScanning:
		mode = motionModeProgress
		phase = "scan"
	case cleanFlowReviewReady:
		mode = motionModeReview
		phase = "review"
	case cleanFlowPermissions:
		mode = motionModeLoading
		phase = "access"
	case cleanFlowReclaiming:
		mode = motionModeProgress
		phase = "reclaim"
	case cleanFlowResult:
		mode = motionModeReview
		phase = "settle"
	default:
		mode = motionModeIdle
		phase = "steady"
	}
	motion := newMotionState(m.spinnerFrame, m.pulse, mode, phase, "sweep")
	if m.reducedMotion {
		return reducedMotionState(motion)
	}
	return motion
}

func cleanFlowSignalLoad(m cleanFlowModel) float64 {
	_, _, totalBytes := cleanFlowByteMix(m)
	if totalBytes <= 0 {
		return 24
	}
	mb := float64(totalBytes) / float64(1<<20)
	if mb < 18 {
		return 18
	}
	if mb > 92 {
		return 92
	}
	return mb
}

func renderCleanDeckTag(label string, meta string) string {
	status := cleanFlowDeckStatus(meta)
	return renderRouteDeckTag("clean", label, status)
}

func cleanFlowDeckStatus(meta string) string {
	parts := strings.Split(meta, "•")
	if len(parts) == 0 {
		return ""
	}
	state := strings.ToLower(strings.TrimSpace(parts[len(parts)-1]))
	switch state {
	case "scanning":
		return "ACTIVE SCAN"
	case "ready", "focus", "review":
		return "REVIEW READY"
	case "reclaiming":
		return "RECLAIM LIVE"
	case "verifying":
		return "VERIFY PASS"
	case "settled":
		return "SETTLED HOLD"
	case "queued":
		return "QUEUED HOLD"
	case "protected":
		return "PROTECTED"
	case "failed":
		return "FAILED"
	default:
		return strings.ToUpper(state)
	}
}

func cleanFlowDeckLane(meta string) string {
	parts := strings.Split(meta, "•")
	if len(parts) == 0 {
		return strings.TrimSpace(meta)
	}
	return strings.TrimSpace(parts[0])
}

func cleanFlowViewportLines(m cleanFlowModel, lines []string, maxLines int) []string {
	if maxLines <= 0 || len(lines) <= maxLines {
		return lines
	}
	if m.autoFollow || m.scrollOffset <= 0 {
		return viewportLines(lines, len(lines)-1, maxLines)
	}
	end := len(lines) - m.scrollOffset
	if end < maxLines {
		end = maxLines
	}
	if end > len(lines) {
		end = len(lines)
	}
	start := end - maxLines
	if start < 0 {
		start = 0
	}
	window := append([]string{}, lines[start:end]...)
	if start > 0 {
		window[0] = mutedStyle.Render("…") + " " + strings.TrimLeft(window[0], " ")
	}
	if end < len(lines) {
		window[len(window)-1] = strings.TrimRight(window[len(window)-1], " ") + " " + mutedStyle.Render("…")
	}
	return window
}
