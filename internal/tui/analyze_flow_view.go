package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/batu3384/sift/internal/domain"
)

func (m analyzeFlowModel) View(base analyzeBrowserModel) string {
	width, height := effectiveSize(m.width, m.height)
	compact := width < 118 || height < 28
	leftWidth := 60
	if width < 132 {
		leftWidth = 50
	}
	rightWidth := width - leftWidth - 10
	if rightWidth < 30 {
		rightWidth = 30
	}
	panelLines := bodyLineBudget(height, 15, 7)
	left := renderPanel("Trace lanes", analyzeFlowSubtitle(m), renderAnalyzeFlowTrace(m, base, leftWidth-4, panelLines), leftWidth, true)
	right := renderPanel("Inspect deck", analyzeFlowGateLine(m), renderAnalyzeFlowDeck(m, base, rightWidth-4, panelLines), rightWidth, false)
	body := left
	if compact {
		ledgerLines, deckLines := analyzeFlowCompactPanelLines(m, panelLines)
		body = strings.Join([]string{
			renderPanel("Trace lanes", analyzeFlowSubtitle(m), renderAnalyzeFlowTrace(m, base, width-8, ledgerLines), width-4, true),
			renderPanel("Inspect deck", analyzeFlowGateLine(m), renderAnalyzeFlowDeck(m, base, width-8, deckLines), width-4, false),
		}, "\n")
	} else {
		body = joinPanels(left, right, width-4)
	}
	return renderChrome("SIFT / Analyze", "oracle trace rail", analyzeFlowStats(m, base, width), body, nil, width, false, m.height)
}

func analyzeFlowCompactPanelLines(m analyzeFlowModel, panelLines int) (int, int) {
	ledgerLines := max(panelLines/2, 6)
	deckLines := max(panelLines/2, 6)
	if !m.autoFollow && m.scrollOffset > 0 {
		ledgerLines = max(panelLines-3, 8)
		deckLines = max(panelLines/3, 4)
	}
	return ledgerLines, deckLines
}

func analyzeFlowSubtitle(m analyzeFlowModel) string {
	hold := ""
	if !m.autoFollow && m.scrollOffset > 0 {
		hold = " • scroll hold"
	}
	switch m.phase {
	case analyzeFlowInspecting:
		return "inspect live" + hold
	case analyzeFlowReviewReady:
		return "review frozen" + hold
	case analyzeFlowPermissions:
		return "access check" + hold
	case analyzeFlowReclaiming:
		return "reclaim live" + hold
	case analyzeFlowResult:
		return "settled result" + hold
	default:
		return "browser ready" + hold
	}
}

func analyzeFlowGateLine(m analyzeFlowModel) string {
	switch m.phase {
	case analyzeFlowPermissions:
		return "access manifest"
	case analyzeFlowReclaiming:
		return "reclaim rail"
	case analyzeFlowResult:
		return "settled deck"
	default:
		return "review gate on"
	}
}

func renderAnalyzeFlowTrace(m analyzeFlowModel, base analyzeBrowserModel, width int, maxLines int) string {
	headerLines := []string{}
	headerLines = append(headerLines, renderRoutePinnedBlock("analyze", analyzeFlowSignalLabel(m), analyzeFlowSignalMotion(m), width, analyzeFlowFreezeBanner(m), analyzeFlowHistoryBanner(m))...)
	headerLines = append(headerLines, analyzeFlowPhaseTrackLines(m, width)...)
	headerLines = append(headerLines, analyzeFlowLaneHeaderLines(m, width)...)
	lines := []string{}
	if active := analyzeFlowActiveTraceLines(m, base, width); len(active) > 0 && (m.autoFollow || m.scrollOffset <= 0) {
		lines = append(lines, active...)
		lines = append(lines, "")
	}
	if len(base.stageOrder) > 0 {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("QUEUE LANE  %d staged  •  NEXT PASS", len(base.stageOrder))))
	}
	if trace := analyzeFlowTraceRows(m, width); len(trace) > 0 {
		lines = append(lines, trace...)
		lines = append(lines, "")
	}
	if len(m.traceRows) == 0 {
		if trail := analyzeTrailLine(base.history, base.plan, width); trail != "" {
			lines = append(lines, mutedStyle.Render(trail), "")
		}
		lines = append(lines, strings.Split(analyzeListView(base, width, max(maxLines-len(lines), 6)), "\n")...)
	}
	return strings.Join(append(headerLines, analyzeFlowViewportLines(m, lines, max(maxLines-len(headerLines), 1))...), "\n")
}

func analyzeFlowLaneHeaderLines(m analyzeFlowModel, width int) []string {
	label := ""
	switch m.phase {
	case analyzeFlowPermissions:
		label = "WATCH LANE"
	case analyzeFlowReviewReady:
		label = "REVIEW LANE"
	case analyzeFlowReclaiming:
		label = "RECLAIM LANE"
	case analyzeFlowResult:
		label = "SETTLED LANE"
	default:
		label = "FOCUS LANE"
	}
	style := safeTokenStyle
	switch label {
	case "WATCH LANE":
		style = highTokenStyle
	case "REVIEW LANE", "RECLAIM LANE":
		style = reviewTokenStyle
	case "SETTLED LANE":
		style = panelTitleStyle
	}
	if telemetry := analyzeFlowLaneTelemetry(m); telemetry != "" {
		return []string{
			singleLine(style.Render(label), max(width, 18)),
			panelMetaStyle.Render(singleLine(telemetry, max(width, 18))),
		}
	}
	return []string{singleLine(style.Render(label), max(width, 18))}
}

func analyzeFlowHistoryBanner(m analyzeFlowModel) string {
	if m.autoFollow || m.scrollOffset <= 0 {
		return ""
	}
	parts := []string{
		reviewTokenStyle.Render("HISTORY HOLD"),
		panelMetaStyle.Render("manual review of trace lane rows"),
		panelMetaStyle.Render("End returns live"),
	}
	return strings.Join(parts, "  •  ")
}

func analyzeFlowFreezeBanner(m analyzeFlowModel) string {
	switch m.phase {
	case analyzeFlowReviewReady:
		parts := []string{
			reviewTokenStyle.Render("REVIEW FREEZE"),
			panelMetaStyle.Render("trace carry locked for gate review"),
		}
		return strings.Join(parts, "  •  ")
	case analyzeFlowPermissions:
		parts := []string{
			reviewTokenStyle.Render("ACCESS HOLD"),
			panelMetaStyle.Render("trace carry held while access checks settle"),
		}
		return strings.Join(parts, "  •  ")
	default:
		return ""
	}
}

func analyzeFlowActiveTraceLines(m analyzeFlowModel, base analyzeBrowserModel, width int) []string {
	if row, ok := m.currentTraceRow(); ok {
		size := ""
		if row.Bytes > 0 {
			size = "  " + domain.HumanBytes(row.Bytes)
		}
		chrome, chromeStyle := analyzeFlowTraceChrome(row.State)
		headline := headerStyle.Render(analyzeFlowTraceStateLabel(row.State))
		if chrome != "" {
			headline = chromeStyle.Render(chrome) + "  " + headline
		}
		lines := []string{
			analyzeFlowTraceMarker(m, row.State) + " " + headline + "  " + truncateText(row.Label, max(width-30, 16)) + size,
		}
		if row.Path != "" {
			lines = append(lines, mutedStyle.Render("path "+truncateText(row.Path, max(width-5, 12))))
		}
		return lines
	}
	item, ok := base.selectedActiveItem()
	if !ok {
		return nil
	}
	size := ""
	if item.Bytes > 0 {
		size = "  " + domain.HumanBytes(item.Bytes)
	}
	chrome, chromeStyle := analyzeFlowPhaseChrome(m.phase)
	headline := headerStyle.Render(analyzeFlowStateLabel(m.phase))
	if chrome != "" {
		headline = chromeStyle.Render(chrome) + "  " + headline
	}
	lines := []string{
		analyzeFlowActiveMarker(m) + " " + headline + "  " + truncateText(analyzeQueueFocusLabel(item), max(width-30, 16)) + size,
	}
	if item.Path != "" {
		lines = append(lines, mutedStyle.Render("path "+truncateText(item.Path, max(width-5, 12))))
	}
	return lines
}

func analyzeFlowTraceRows(m analyzeFlowModel, width int) []string {
	if len(m.traceRows) == 0 {
		return nil
	}
	lines := make([]string, 0, len(m.traceRows)*2)
	for _, row := range m.traceRows {
		size := "pending"
		if row.Bytes > 0 {
			size = domain.HumanBytes(row.Bytes)
		}
		labelWidth := max(width-24, 16)
		marker, lineStyle, stateStyle := analyzeFlowTraceRowStyles(m, row.State)
		chrome, chromeStyle := analyzeFlowTraceChrome(row.State)
		line := fmt.Sprintf("%s %-*s %10s", marker, labelWidth, truncateText(row.Label, labelWidth), size)
		if chrome != "" {
			line += "  " + chromeStyle.Render(chrome)
		}
		line += "  " + stateStyle.Render(analyzeFlowTraceStateLabel(row.State))
		lines = append(lines, lineStyle.Render(line))
		if detail := analyzeFlowTraceDetail(row, width); detail != "" {
			lines = append(lines, detail)
		}
	}
	return lines
}

func analyzeFlowTraceDetail(row analyzeFlowTraceRow, width int) string {
	if strings.TrimSpace(row.Path) == "" {
		return ""
	}
	switch row.State {
	case "review", "queued", "reclaiming", "verifying":
	default:
		return ""
	}
	return mutedStyle.Render("   path " + truncateText(row.Path, max(width-8, 12)))
}

func analyzeFlowActiveMarker(m analyzeFlowModel) string {
	switch m.phase {
	case analyzeFlowInspecting, analyzeFlowReclaiming:
		if !m.reducedMotion && len(spinnerFrames) > 0 {
			return reviewStyle.Render(spinnerFrames[m.spinnerFrame%len(spinnerFrames)])
		}
		return reviewStyle.Render("*")
	case analyzeFlowReviewReady:
		return reviewStyle.Render("*")
	case analyzeFlowPermissions:
		return highStyle.Render("-")
	case analyzeFlowResult:
		return safeStyle.Render("+")
	default:
		return mutedStyle.Render("-")
	}
}

func analyzeFlowTraceMarker(m analyzeFlowModel, state string) string {
	switch state {
	case "reclaiming":
		if !m.reducedMotion && len(spinnerFrames) > 0 {
			return reviewStyle.Render(spinnerFrames[m.spinnerFrame%len(spinnerFrames)])
		}
		return reviewStyle.Render("*")
	case "verifying":
		if m.reducedMotion {
			return panelTitleStyle.Render("-")
		}
		if m.pulse {
			return panelTitleStyle.Render("+")
		}
		return panelTitleStyle.Render("-")
	case "review":
		return reviewStyle.Render("*")
	case "queued":
		return mutedStyle.Render("-")
	case "protected":
		return highStyle.Render("-")
	case "failed":
		return highStyle.Render("x")
	case "settled":
		return safeStyle.Render("+")
	case "skipped":
		return mutedStyle.Render("-")
	default:
		return mutedStyle.Render("•")
	}
}

func analyzeFlowTraceRowStyles(m analyzeFlowModel, state string) (string, lipgloss.Style, lipgloss.Style) {
	switch state {
	case "reclaiming":
		return analyzeFlowTraceMarker(m, state), lipgloss.NewStyle().Bold(true), reviewTokenStyle
	case "verifying":
		return analyzeFlowTraceMarker(m, state), lipgloss.NewStyle().Bold(true), panelTitleStyle
	case "review":
		return analyzeFlowTraceMarker(m, state), lipgloss.NewStyle().Bold(true), reviewTokenStyle
	case "queued":
		return analyzeFlowTraceMarker(m, state), lipgloss.NewStyle(), mutedStyle
	case "protected":
		return analyzeFlowTraceMarker(m, state), lipgloss.NewStyle(), highTokenStyle
	case "failed":
		return analyzeFlowTraceMarker(m, state), lipgloss.NewStyle().Bold(true), highTokenStyle
	case "settled":
		return analyzeFlowTraceMarker(m, state), mutedStyle, safeTokenStyle
	case "skipped":
		return analyzeFlowTraceMarker(m, state), mutedStyle, mutedStyle
	default:
		return analyzeFlowTraceMarker(m, state), lipgloss.NewStyle(), mutedStyle
	}
}

func analyzeFlowTraceChrome(state string) (string, lipgloss.Style) {
	switch state {
	case "review", "queued":
		return "GATE", reviewTokenStyle
	case "reclaiming", "verifying":
		return "LIVE", reviewTokenStyle
	case "protected", "failed":
		return "WATCH", highTokenStyle
	case "settled", "skipped":
		return "ARCHIVE", safeTokenStyle
	default:
		return "", mutedStyle
	}
}

func analyzeFlowPhaseChrome(phase analyzeFlowPhase) (string, lipgloss.Style) {
	switch phase {
	case analyzeFlowInspecting, analyzeFlowReclaiming:
		return "LIVE", reviewTokenStyle
	case analyzeFlowReviewReady:
		return "GATE", reviewTokenStyle
	case analyzeFlowPermissions:
		return "WATCH", highTokenStyle
	case analyzeFlowResult:
		return "ARCHIVE", safeTokenStyle
	default:
		return "", mutedStyle
	}
}

func analyzeFlowViewportLines(m analyzeFlowModel, lines []string, maxLines int) []string {
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

func analyzeFlowLaneTelemetry(m analyzeFlowModel) string {
	if len(m.traceRows) == 0 {
		return ""
	}
	counts := map[string]int{}
	var bytes int64
	for _, row := range m.traceRows {
		bytes += row.Bytes
		switch state, _ := analyzeFlowTraceChrome(row.State); state {
		case "LIVE":
			counts["live"]++
		case "GATE":
			counts["gate"]++
		case "WATCH":
			counts["watch"]++
		case "ARCHIVE":
			counts["archive"]++
		}
	}
	parts := []string{fmt.Sprintf("%d %s", len(m.traceRows), pl(len(m.traceRows), "trace", "traces"))}
	if bytes > 0 {
		parts = append(parts, cleanFlowHumanBytes(bytes))
	}
	for _, key := range []string{"live", "gate", "watch", "archive"} {
		if count := counts[key]; count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", count, key))
		}
	}
	return strings.Join(parts, "  •  ")
}

func analyzeFlowTraceStateLabel(state string) string {
	switch state {
	case "review":
		return "REVIEW READY"
	case "queued":
		return "QUEUED HOLD"
	case "reclaiming":
		return "RECLAIM LIVE"
	case "verifying":
		return "VERIFY PASS"
	case "settled":
		return "SETTLED HOLD"
	case "protected":
		return "GUARDED HOLD"
	case "failed":
		return "FAILED HOLD"
	case "skipped":
		return "SKIPPED"
	default:
		return strings.ToUpper(state)
	}
}

func renderAnalyzeFlowDeck(m analyzeFlowModel, base analyzeBrowserModel, width int, maxLines int) string {
	focus, ok := analyzeFlowCurrentFocus(m, base)
	if !ok {
		lines := []string{
			mutedStyle.Render(singleLine("Status   "+analyzeFlowStatusLine(m), width)),
			mutedStyle.Render(singleLine("Next     "+analyzeFlowNextLine(m), width)),
			"",
			mutedStyle.Render("Pick a finding to open the inspect deck."),
		}
		return strings.Join(viewportLines(lines, 0, maxLines), "\n")
	}
	lines := []string{
		mutedStyle.Render(singleLine("Status   "+analyzeFlowStatusLine(m), width)),
		mutedStyle.Render(singleLine("Next     "+analyzeFlowNextLine(m), width)),
		"",
		renderToneBadge(focus.tone) + " " + headerStyle.Render(focus.label),
		wrapText(analyzeFlowFocusCopy(m, focus.label), width),
		"",
		renderSectionRule(width),
		headerStyle.Render("Inspect signal"),
		renderAnalyzeFlowSignal(m, base, focus.bytes),
		"",
		renderSectionRule(width),
		headerStyle.Render("Current focus"),
		renderAnalyzeDeckTag(m, "HOT PATH", focus.state),
		mutedStyle.Render(wrapText(analyzeFlowCurrentDeckCopy(m)+"  •  "+routeDeckHint("analyze", "HOT PATH"), width)),
	}
	if focus.path != "" {
		lines = append(lines, mutedStyle.Render("path "+truncateText(focus.path, max(width-5, 12))))
	}
	if outcome := analyzeFlowOutcomeLine(m); outcome != "" {
		lines = append(lines, mutedStyle.Render("Outcome  ")+outcome)
	}
	if len(base.stageOrder) > 0 {
		lines = append(lines,
			"",
			renderSectionRule(width),
			headerStyle.Render("Batch lane"),
			renderAnalyzeDeckTag(m, "NEXT PASS", analyzeFlowBatchStateLabel(m.phase)),
			mutedStyle.Render(wrapText(analyzeFlowNextDeckCopy(m, len(base.stageOrder))+"  •  "+routeDeckHint("analyze", "NEXT PASS"), width)),
		)
	}
	if m.phase != analyzeFlowResult {
		if preview := analyzeReviewPreviewLines(base, width); len(preview) > 0 {
			lines = append(lines, "", renderSectionRule(width), headerStyle.Render("Review preview"))
			lines = append(lines, preview...)
		}
	}
	lines = append(lines, "",
		mutedStyle.Render("Gate     ")+analyzeFlowGateLine(m),
		mutedStyle.Render("Scope    ")+sectionTitle(base.plan, focus.category),
	)
	if focus.bytes > 0 {
		lines = append(lines, mutedStyle.Render("Yield    ")+domain.HumanBytes(focus.bytes))
	}
	if impact := analyzeImpactLine(focus.finding, base.plan); impact != "" {
		lines = append(lines, mutedStyle.Render("Watch    ")+impact)
	}
	lines = append(lines, mutedStyle.Render("Next     ")+analyzeActionRail(base))
	return strings.Join(viewportLines(lines, 0, maxLines), "\n")
}

func analyzeFlowStatusLine(m analyzeFlowModel) string {
	switch m.phase {
	case analyzeFlowInspecting:
		return "live trace running"
	case analyzeFlowReviewReady:
		return "waiting at review gate"
	case analyzeFlowPermissions:
		return "access checks running"
	case analyzeFlowReclaiming:
		return "reclaim running"
	case analyzeFlowResult:
		return "run settled"
	default:
		return "browser ready"
	}
}

func analyzeFlowNextLine(m analyzeFlowModel) string {
	switch m.phase {
	case analyzeFlowInspecting:
		return "review gate"
	case analyzeFlowReviewReady:
		return "access check if needed"
	case analyzeFlowPermissions:
		return "reclaim rail"
	case analyzeFlowReclaiming:
		return "settled rail"
	case analyzeFlowResult:
		return "inspect result or rerun"
	default:
		return "pick trace"
	}
}

func analyzeFlowTone(item domain.Finding) string {
	switch item.Risk {
	case domain.RiskHigh:
		return "high"
	case domain.RiskReview:
		return "review"
	default:
		return "safe"
	}
}

func analyzeFlowFocusCopy(m analyzeFlowModel, label string) string {
	switch m.phase {
	case analyzeFlowPermissions:
		return "Access hold stays on this trace."
	case analyzeFlowReclaiming:
		return fmt.Sprintf("%s is reclaiming live on this trace.", label)
	case analyzeFlowResult:
		return "Trace settled. Check what cleared and what still needs review."
	case analyzeFlowReviewReady:
		return "Review is frozen on this trace."
	default:
		return fmt.Sprintf("%s is active. Stage findings and keep the reclaim gate warm.", label)
	}
}

func analyzeFlowCurrentDeckCopy(m analyzeFlowModel) string {
	switch m.phase {
	case analyzeFlowPermissions:
		return "access lift pinned"
	case analyzeFlowResult:
		return "settled reclaim pinned"
	default:
		return "trace reclaim pinned"
	}
}

func analyzeFlowNextDeckCopy(m analyzeFlowModel, queued int) string {
	if queued <= 0 {
		return "no queued reclaim behind this trace"
	}
	return "queued findings behind this trace"
}

func renderAnalyzeFlowSignal(m analyzeFlowModel, base analyzeBrowserModel, bytes int64) string {
	summary := analyzeFlowSignalLabel(m)
	extras := []string{}
	if bytes > 0 {
		extras = append(extras, domain.HumanBytes(bytes))
	}
	return renderRouteSignalBlock("analyze", analyzeFlowSignalMotion(m), analyzeFlowSignalLoad(base, bytes), summary, extras...)
}

func analyzeFlowSignalLabel(m analyzeFlowModel) string {
	switch m.phase {
	case analyzeFlowPermissions:
		return "ACCESS RAIL"
	case analyzeFlowReclaiming:
		return "RECLAIM RAIL"
	case analyzeFlowResult:
		return "SETTLED RAIL"
	case analyzeFlowReviewReady:
		return "REVIEW RAIL"
	default:
		return "TRACE RAIL"
	}
}

func analyzeFlowPhaseTrackLines(m analyzeFlowModel, width int) []string {
	current := analyzeFlowSignalLabel(m)
	segments := []string{}
	for _, phase := range []string{"TRACE RAIL", "REVIEW RAIL", "ACCESS RAIL", "RECLAIM RAIL", "SETTLED RAIL"} {
		label := routeStickyPhaseMeter("analyze", phase)
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

func analyzeFlowSignalMotion(m analyzeFlowModel) motionState {
	mode := motionModeIdle
	phase := "inspect"
	switch m.phase {
	case analyzeFlowInspecting:
		mode = motionModeLoading
		phase = "inspect"
	case analyzeFlowReviewReady:
		mode = motionModeReview
		phase = "review"
	case analyzeFlowPermissions:
		mode = motionModeAlert
		phase = "access"
	case analyzeFlowReclaiming:
		mode = motionModeProgress
		phase = "reclaim"
	case analyzeFlowResult:
		mode = motionModeIdle
		phase = "settle"
	}
	motion := newMotionState(m.spinnerFrame, m.pulse, mode, phase, "analyze")
	if m.reducedMotion {
		return reducedMotionState(motion)
	}
	return motion
}

func analyzeFlowSignalLoad(base analyzeBrowserModel, bytes int64) float64 {
	if bytes > 0 {
		return float64(bytes)
	}
	if plan, ok := base.reviewPreviewPlan(); ok {
		return float64(planDisplayBytes(plan))
	}
	return float64(len(base.plan.Items))
}

func analyzeFlowStateLabel(phase analyzeFlowPhase) string {
	switch phase {
	case analyzeFlowPermissions:
		return "ACCESS CHECK"
	case analyzeFlowReclaiming:
		return "RECLAIM LIVE"
	case analyzeFlowResult:
		return "SETTLED HOLD"
	case analyzeFlowReviewReady:
		return "REVIEW READY"
	default:
		return "ACTIVE TRACE"
	}
}

func analyzeFlowBatchStateLabel(phase analyzeFlowPhase) string {
	switch phase {
	case analyzeFlowPermissions:
		return "ACCESS CHECK"
	case analyzeFlowReclaiming:
		return "RECLAIM QUEUE"
	case analyzeFlowResult:
		return "SETTLED HOLD"
	case analyzeFlowReviewReady:
		return "REVIEW READY"
	default:
		return "QUEUE READY"
	}
}

type analyzeFlowFocus struct {
	finding  domain.Finding
	label    string
	path     string
	category domain.Category
	bytes    int64
	tone     string
	state    string
}

func analyzeFlowCurrentFocus(m analyzeFlowModel, base analyzeBrowserModel) (analyzeFlowFocus, bool) {
	if row, ok := m.currentTraceRow(); ok {
		return analyzeFlowFocus{
			finding: domain.Finding{
				ID:       row.FindingID,
				Name:     row.Label,
				Path:     row.Path,
				Category: row.Category,
				Bytes:    row.Bytes,
			},
			label:    row.Label,
			path:     row.Path,
			category: row.Category,
			bytes:    row.Bytes,
			tone:     analyzeFlowToneForState(row.State),
			state:    analyzeFlowTraceStateLabel(row.State),
		}, true
	}
	item, ok := base.selectedActiveItem()
	if !ok {
		return analyzeFlowFocus{}, false
	}
	return analyzeFlowFocus{
		finding:  item,
		label:    analyzeQueueFocusLabel(item),
		path:     item.Path,
		category: item.Category,
		bytes:    item.Bytes,
		tone:     analyzeFlowTone(item),
		state:    analyzeFlowStateLabel(m.phase),
	}, true
}

func analyzeFlowToneForState(state string) string {
	switch state {
	case "protected", "failed":
		return "high"
	case "review", "queued", "reclaiming", "verifying":
		return "review"
	default:
		return "safe"
	}
}

func renderAnalyzeDeckTag(m analyzeFlowModel, label, meta string) string {
	tag := renderRouteDeckTag("analyze", label, meta)
	if strings.EqualFold(strings.TrimSpace(label), "HOT PATH") {
		return analyzeFlowActiveMarker(m) + " " + tag
	}
	return tag
}

func analyzeFlowOutcomeLine(m analyzeFlowModel) string {
	if !m.hasResult || len(m.result.Items) == 0 {
		return ""
	}
	completed, deleted, failed, _, protected := countResultStatuses(m.result)
	cleared := completed + deleted
	parts := []string{}
	if cleared > 0 {
		parts = append(parts, fmt.Sprintf("%d cleared", cleared))
	}
	if protected > 0 {
		parts = append(parts, fmt.Sprintf("%d guarded", protected))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	return strings.Join(parts, "  •  ")
}
