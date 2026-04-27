package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/batu3384/sift/internal/domain"
)

func (m uninstallFlowModel) View(base uninstallModel) string {
	width, height := effectiveSize(m.width, m.height)
	compact := width < 118 || height < 28
	leftWidth := 58
	if width < 132 {
		leftWidth = 50
	}
	rightWidth := width - leftWidth - 10
	if rightWidth < 30 {
		rightWidth = 30
	}
	panelLines := bodyLineBudget(height, 15, 7)
	left := renderPanel("Target lanes", uninstallFlowSubtitle(m), renderUninstallFlowTargets(m, base, leftWidth-4, panelLines), leftWidth, true)
	right := renderPanel("Removal deck", uninstallFlowGateLine(m), renderUninstallFlowDeck(m, base, rightWidth-4, panelLines), rightWidth, false)
	body := left
	if compact {
		ledgerLines, deckLines := uninstallFlowCompactPanelLines(m, panelLines)
		body = strings.Join([]string{
			renderPanel("Target lanes", uninstallFlowSubtitle(m), renderUninstallFlowTargets(m, base, width-8, ledgerLines), width-4, true),
			renderPanel("Removal deck", uninstallFlowGateLine(m), renderUninstallFlowDeck(m, base, width-8, deckLines), width-4, false),
		}, "\n")
	} else {
		body = joinPanels(left, right, width-4)
	}
	return renderChrome("SIFT / Uninstall", "courier handoff rail", uninstallFlowStats(m, base, width), body, nil, width, false, m.height)
}

func uninstallFlowCompactPanelLines(m uninstallFlowModel, panelLines int) (int, int) {
	ledgerLines := max(panelLines/2, 6)
	deckLines := max(panelLines/2, 6)
	if !m.autoFollow && m.scrollOffset > 0 {
		ledgerLines = max(panelLines-3, 8)
		deckLines = max(panelLines/3, 4)
	}
	return ledgerLines, deckLines
}

func uninstallFlowSubtitle(m uninstallFlowModel) string {
	hold := ""
	if !m.autoFollow && m.scrollOffset > 0 {
		hold = " • scroll hold"
	}
	switch m.phase {
	case uninstallFlowInventory:
		return "inventory live" + hold
	case uninstallFlowReviewReady:
		return "review frozen" + hold
	case uninstallFlowPermissions:
		return "access check" + hold
	case uninstallFlowRemoving:
		return "handoff live" + hold
	case uninstallFlowResult:
		return "settled result" + hold
	default:
		return "controller ready" + hold
	}
}

func renderUninstallFlowTargets(flow uninstallFlowModel, base uninstallModel, width int, maxLines int) string {
	headerLines := []string{}
	railPhase := "TARGET RAIL"
	if selected, ok := base.selected(); ok {
		railPhase = uninstallFlowSignalLabel(flow, selected)
	}
	headerLines = append(headerLines, renderRoutePinnedBlock("uninstall", railPhase, uninstallFlowSignalMotion(flow), width, uninstallFlowFreezeBanner(flow), uninstallFlowHistoryBanner(flow))...)
	headerLines = append(headerLines, uninstallFlowPhaseTrackLines(railPhase, width)...)
	lines := []string{}
	historyHold := !flow.autoFollow && flow.scrollOffset > 0
	if base.message != "" && !historyHold {
		lines = append(lines, mutedStyle.Render(base.message), "")
	}
	if !historyHold {
		lines = append(lines, railStyle.Render("TARGET FILTER"), base.search.View(), "")
	}
	if len(base.filtered) == 0 {
		lines = append(lines, mutedStyle.Render("No installed apps match the current filter."))
		return strings.Join(append(headerLines, uninstallFlowViewportLines(flow, lines, 0, max(maxLines-len(headerLines), 1))...), "\n")
	}
	focusLine := len(lines)
	for _, lane := range uninstallFlowTargetLanes(base) {
		lines = append(lines, uninstallFlowLaneHeaderLines(flow, base, lane, width)...)
		for _, idx := range lane.indices {
			item := base.items[idx]
			selected := base.filtered[base.cursor] == idx
			staged := base.isStaged(item)
			mode := reviewTokenStyle.Render("NATIVE")
			if !item.HasNative {
				mode = highTokenStyle.Render("REMNANTS")
			}
			parts := []string{
				fmt.Sprintf("%s%s", uninstallFlowSelectionPrefix(flow, item, selected), truncateText(item.Name, 22)),
				mode,
			}
			if staged {
				parts = append(parts, safeTokenStyle.Render("QUEUED"))
			}
			if chrome := uninstallFlowRowChrome(flow, item, selected, staged); chrome != "" {
				parts = append(parts, chrome)
			}
			extras := []string{}
			if item.Sensitive {
				extras = append(extras, highTokenStyle.Render("SENSITIVE"))
			} else if item.RequiresAdmin {
				extras = append(extras, reviewTokenStyle.Render("ADMIN"))
			}
			if item.SizeLabel != "" && width >= 42 {
				extras = append(extras, mutedStyle.Render(item.SizeLabel))
			}
			if len(extras) > 0 {
				parts = append(parts, extras...)
			}
			line := strings.Join(parts, "  ")
			line = singleLine(line, width)
			if selected {
				line = selectedLine.Render(line)
				focusLine = len(lines)
			}
			lines = append(lines, line)
			if detail := uninstallFlowTargetDetail(item, selected, width); detail != "" {
				lines = append(lines, detail)
			}
		}
		lines = append(lines, "")
	}
	return strings.Join(append(headerLines, uninstallFlowViewportLines(flow, lines, focusLine, max(maxLines-len(headerLines), 1))...), "\n")
}

func renderUninstallFlowDeck(m uninstallFlowModel, base uninstallModel, width int, maxLines int) string {
	selected, ok := base.selected()
	if !ok {
		lines := []string{
			mutedStyle.Render(singleLine("Status   "+uninstallFlowStatusLine(m), width)),
			mutedStyle.Render(singleLine("Next     "+uninstallFlowNextDeckLine(m), width)),
			"",
			mutedStyle.Render("Pick an app to open the removal deck."),
		}
		return strings.Join(viewportLines(lines, 0, maxLines), "\n")
	}
	lines := []string{
		mutedStyle.Render(singleLine("Status   "+uninstallFlowStatusLine(m), width)),
		mutedStyle.Render(singleLine("Next     "+uninstallFlowNextDeckLine(m), width)),
		"",
		renderToneBadge(toneForUninstall(selected.HasNative)) + " " + headerStyle.Render(selected.Name),
		wrapText(uninstallFlowFocusCopy(m, selected), width),
		"",
		renderSectionRule(width),
		headerStyle.Render("Removal signal"),
		renderUninstallRemovalSignal(m, selected),
		"",
		renderSectionRule(width),
		headerStyle.Render("Current target"),
		renderUninstallDeckTag(m, "HOT PATH", uninstallFlowStateLabel(m.phase, selected, true)),
		mutedStyle.Render(wrapText(uninstallFlowCurrentDeckCopy(m, selected)+"  •  "+routeDeckHint("uninstall", "HOT PATH"), width)),
	}
	if selected.Location != "" {
		lines = append(lines, mutedStyle.Render("path "+truncateText(selected.Location, max(width-5, 12))))
	}
	if previewLines := uninstallPreviewLines(base, selected, width); len(previewLines) > 0 {
		lines = append(lines, "", renderSectionRule(width), headerStyle.Render("Review preview"))
		lines = append(lines, previewLines...)
	}
	if base.stageCount() > 0 {
		lines = append(lines,
			"",
			renderSectionRule(width),
			headerStyle.Render("Batch lane"),
			renderUninstallDeckTag(m, "NEXT PASS", uninstallFlowStateLabel(m.phase, selected, false)),
			mutedStyle.Render(wrapText(uninstallFlowNextDeckCopy(m, selected)+"  •  "+routeDeckHint("uninstall", "NEXT PASS"), width)),
		)
	}
	lines = append(lines,
		"",
		mutedStyle.Render("Gate     ")+uninstallFlowGateLine(m),
		mutedStyle.Render("Scope    ")+uninstallScopeText(selected),
	)
	if outcome := uninstallFlowOutcomeLine(m); outcome != "" {
		lines = append(lines, mutedStyle.Render("Outcome  ")+outcome)
	}
	if size := coalesceText(selected.SizeLabel, "unknown"); size != "" {
		lines = append(lines, mutedStyle.Render("Yield    ")+size)
	}
	if note := uninstallNote(selected); note != "" {
		lines = append(lines, mutedStyle.Render("Watch    ")+note)
	}
	lines = append(lines, mutedStyle.Render("Next     ")+uninstallNextLine(base, selected, ok))
	return strings.Join(viewportLines(lines, 0, maxLines), "\n")
}

func uninstallFlowStatusLine(m uninstallFlowModel) string {
	switch m.phase {
	case uninstallFlowInventory:
		return "live inventory running"
	case uninstallFlowReviewReady:
		return "waiting at review gate"
	case uninstallFlowPermissions:
		return "access checks running"
	case uninstallFlowRemoving:
		return "handoff running"
	case uninstallFlowResult:
		return "aftercare settled"
	default:
		return "controller ready"
	}
}

func uninstallFlowNextDeckLine(m uninstallFlowModel) string {
	switch m.phase {
	case uninstallFlowInventory:
		return "review gate"
	case uninstallFlowReviewReady:
		return "access check if needed"
	case uninstallFlowPermissions:
		return "handoff rail"
	case uninstallFlowRemoving:
		return "aftercare rail"
	case uninstallFlowResult:
		return "inspect result or rerun"
	default:
		return "open target"
	}
}

func uninstallFlowFocusCopy(m uninstallFlowModel, item uninstallItem) string {
	switch m.phase {
	case uninstallFlowPermissions:
		return "Access hold keeps handoff on this rail."
	case uninstallFlowRemoving:
		return "Live handoff is moving through native and remnant passes."
	case uninstallFlowResult:
		return "Aftercare settled. Check what cleared and what stayed guarded."
	case uninstallFlowReviewReady:
		return "Review is frozen on the same handoff rail."
	case uninstallFlowInventory:
		return fmt.Sprintf("%s is warming on the handoff rail.", item.Name)
	default:
		return "Targets, gate, handoff, and aftercare stay on one rail."
	}
}

func uninstallFlowCurrentDeckCopy(m uninstallFlowModel, item uninstallItem) string {
	switch m.phase {
	case uninstallFlowPermissions:
		return "access lift pinned"
	case uninstallFlowResult:
		return "aftercare pinned"
	default:
		if item.HasNative {
			return "native handoff pinned"
		}
		return "remnant pass pinned"
	}
}

func uninstallFlowNextDeckCopy(m uninstallFlowModel, item uninstallItem) string {
	switch m.phase {
	case uninstallFlowResult:
		return "queued remnants behind this target"
	default:
		if item.HasNative {
			return "queued remnants behind this target"
		}
		return "aftercare follow-up behind this target"
	}
}

func uninstallFlowGateLine(m uninstallFlowModel) string {
	switch m.phase {
	case uninstallFlowPermissions:
		return "access manifest"
	case uninstallFlowRemoving:
		return "handoff live"
	case uninstallFlowResult:
		return "settled deck"
	default:
		return "review gate on"
	}
}

func renderUninstallRemovalSignal(m uninstallFlowModel, item uninstallItem) string {
	summary := uninstallFlowSignalLabel(m, item)
	extras := []string{}
	if phase := uninstallFlowStateLabel(m.phase, item, true); phase != "" {
		extras = append(extras, phase)
	}
	return renderRouteSignalBlock("uninstall", uninstallFlowSignalMotion(m), uninstallFlowSignalLoad(item), summary, extras...)
}

func uninstallFlowStats(flow uninstallFlowModel, base uninstallModel, width int) []string {
	cardWidth := 24
	if width < 110 {
		cardWidth = width - 8
	}
	native := 0
	watch := 0
	for _, item := range base.items {
		if item.HasNative {
			native++
		}
		if item.Sensitive || item.RequiresAdmin {
			watch++
		}
	}
	targetValue := fmt.Sprintf("%d %s", len(base.items), pl(len(base.items), "app", "apps"))
	if meter := routeStickyPhaseCount("uninstall", uninstallFlowPrimaryRail(flow)); meter != "" {
		targetValue += " " + meter
	}
	return []string{
		renderRouteStatCard("uninstall", "targets", targetValue, "review", cardWidth),
		renderRouteStatCard("uninstall", "native", fmt.Sprintf("%d native", native), "safe", cardWidth),
		renderRouteStatCard("uninstall", "watch", fmt.Sprintf("%d gated", watch), "high", cardWidth),
		renderRouteStatCard("uninstall", "queue", fmt.Sprintf("%d staged", base.stageCount()), "review", cardWidth),
	}
}

func uninstallFlowPrimaryRail(flow uninstallFlowModel) string {
	switch flow.phase {
	case uninstallFlowReviewReady:
		return "REVIEW RAIL"
	case uninstallFlowPermissions:
		return "ACCESS RAIL"
	case uninstallFlowRemoving:
		return "HANDOFF RAIL"
	case uninstallFlowResult:
		return "AFTERCARE RAIL"
	default:
		return "TARGET RAIL"
	}
}

func uninstallFlowStateLabel(phase uninstallFlowPhase, item uninstallItem, primary bool) string {
	switch phase {
	case uninstallFlowInventory:
		return "ACTIVE INVENTORY"
	case uninstallFlowReviewReady:
		if item.HasNative && primary {
			return "HANDOFF READY"
		}
		return "REVIEW READY"
	case uninstallFlowPermissions:
		return "ACCESS CHECK"
	case uninstallFlowRemoving:
		if item.HasNative && primary {
			return "NATIVE HANDOFF"
		}
		return "REMNANT PASS"
	case uninstallFlowResult:
		return "AFTERCARE HOLD"
	default:
		return "TARGET READY"
	}
}

func renderUninstallDeckTag(m uninstallFlowModel, label, meta string) string {
	tag := renderRouteDeckTag("uninstall", label, meta)
	if strings.EqualFold(strings.TrimSpace(label), "HOT PATH") {
		if marker := uninstallFlowDeckMarker(m); marker != "" {
			return marker + " " + tag
		}
	}
	return tag
}

func uninstallFlowDeckMarker(m uninstallFlowModel) string {
	switch m.phase {
	case uninstallFlowInventory, uninstallFlowRemoving:
		if !m.reducedMotion && len(spinnerFrames) > 0 {
			return reviewStyle.Render(spinnerFrames[m.spinnerFrame%len(spinnerFrames)])
		}
		return reviewStyle.Render("*")
	case uninstallFlowReviewReady:
		return reviewStyle.Render("*")
	case uninstallFlowPermissions:
		return highStyle.Render("-")
	case uninstallFlowResult:
		return safeStyle.Render("+")
	default:
		return ""
	}
}

func uninstallFlowSelectionPrefix(flow uninstallFlowModel, item uninstallItem, selected bool) string {
	if !selected {
		return selectionPrefix(false)
	}
	style := reviewStyle
	if !item.HasNative {
		style = highStyle
	}
	if item.RequiresAdmin || item.Sensitive {
		style = highStyle
	}
	if flow.reducedMotion || len(spinnerFrames) == 0 {
		return style.Render("* ")
	}
	return style.Render(spinnerFrames[flow.spinnerFrame%len(spinnerFrames)] + " ")
}

func uninstallFlowTargetDetail(item uninstallItem, selected bool, width int) string {
	if !selected || strings.TrimSpace(item.Location) == "" {
		return ""
	}
	return mutedStyle.Render("   path " + truncateText(item.Location, max(width-8, 12)))
}

func uninstallFlowRowChrome(flow uninstallFlowModel, item uninstallItem, selected bool, staged bool) string {
	switch uninstallFlowRowState(flow, item, selected, staged) {
	case "live":
		return reviewTokenStyle.Render("LIVE")
	case "gate":
		return reviewTokenStyle.Render("GATE")
	case "watch":
		return highTokenStyle.Render("WATCH")
	case "archive":
		return safeTokenStyle.Render("ARCHIVE")
	default:
		return ""
	}
}

func uninstallFlowRowState(flow uninstallFlowModel, item uninstallItem, selected bool, staged bool) string {
	switch {
	case item.Sensitive || item.RequiresAdmin:
		return "watch"
	case flow.phase == uninstallFlowResult:
		return "archive"
	case flow.phase == uninstallFlowReviewReady || flow.phase == uninstallFlowPermissions:
		if staged || selected {
			return "gate"
		}
	case flow.phase == uninstallFlowInventory || flow.phase == uninstallFlowRemoving:
		if selected {
			return "live"
		}
	}
	return ""
}

func uninstallFlowOutcomeLine(m uninstallFlowModel) string {
	if !m.hasResult || len(m.result.Items) == 0 {
		return ""
	}
	removed := 0
	guarded := 0
	failed := 0
	for _, item := range m.result.Items {
		switch item.Status {
		case domain.StatusDeleted:
			removed++
		case domain.StatusProtected:
			guarded++
		case domain.StatusFailed:
			failed++
		}
	}
	parts := []string{}
	if removed > 0 {
		parts = append(parts, fmt.Sprintf("%d removed", removed))
	}
	if guarded > 0 {
		parts = append(parts, fmt.Sprintf("%d guarded", guarded))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	return strings.Join(parts, "  •  ")
}

func uninstallFlowHistoryBanner(m uninstallFlowModel) string {
	if m.autoFollow || m.scrollOffset <= 0 {
		return ""
	}
	parts := []string{
		reviewTokenStyle.Render("HISTORY HOLD"),
		panelMetaStyle.Render("manual review of target lane rows"),
		panelMetaStyle.Render("End returns live"),
	}
	return strings.Join(parts, "  •  ")
}

func uninstallFlowFreezeBanner(m uninstallFlowModel) string {
	switch m.phase {
	case uninstallFlowReviewReady:
		parts := []string{
			reviewTokenStyle.Render("REVIEW FREEZE"),
			panelMetaStyle.Render("target carry locked for gate review"),
		}
		return strings.Join(parts, "  •  ")
	case uninstallFlowPermissions:
		parts := []string{
			reviewTokenStyle.Render("ACCESS HOLD"),
			panelMetaStyle.Render("target carry held while access checks settle"),
		}
		return strings.Join(parts, "  •  ")
	default:
		return ""
	}
}

type uninstallFlowTargetLane struct {
	label   string
	indices []int
}

func uninstallFlowTargetLanes(base uninstallModel) []uninstallFlowTargetLane {
	selectedIdx := -1
	if base.cursor >= 0 && base.cursor < len(base.filtered) {
		selectedIdx = base.filtered[base.cursor]
	}
	groups := map[string][]int{}
	lastSeen := map[string]int{}
	selectedLane := ""
	for pos, idx := range base.filtered {
		if idx < 0 || idx >= len(base.items) {
			continue
		}
		label := uninstallFlowTargetLaneLabel(base.items[idx])
		groups[label] = append(groups[label], idx)
		lastSeen[label] = pos
		if idx == selectedIdx {
			selectedLane = label
		}
	}
	order := make([]string, 0, len(groups))
	for label := range groups {
		order = append(order, label)
	}
	sort.SliceStable(order, func(i, j int) bool {
		if order[i] == selectedLane {
			return true
		}
		if order[j] == selectedLane {
			return false
		}
		return uninstallFlowLaneRank(order[i], lastSeen[order[i]]) < uninstallFlowLaneRank(order[j], lastSeen[order[j]])
	})
	lanes := make([]uninstallFlowTargetLane, 0, len(order))
	for _, label := range order {
		lanes = append(lanes, uninstallFlowTargetLane{label: label, indices: groups[label]})
	}
	return lanes
}

func uninstallFlowTargetLaneLabel(item uninstallItem) string {
	if item.Sensitive || item.RequiresAdmin {
		return "WATCH LANE"
	}
	if item.HasNative {
		return "NATIVE LANE"
	}
	return "REMNANT LANE"
}

func uninstallFlowLaneRank(label string, fallback int) int {
	switch label {
	case "NATIVE LANE":
		return 10
	case "REMNANT LANE":
		return 20
	case "WATCH LANE":
		return 30
	default:
		return 100 + fallback
	}
}

func uninstallFlowLaneHeaderLines(flow uninstallFlowModel, base uninstallModel, lane uninstallFlowTargetLane, width int) []string {
	style := panelTitleStyle
	switch lane.label {
	case "NATIVE LANE":
		style = reviewTokenStyle
	case "REMNANT LANE":
		style = highTokenStyle
	case "WATCH LANE":
		style = highTokenStyle
	}
	stateCounts := map[string]int{}
	for _, idx := range lane.indices {
		if idx < 0 || idx >= len(base.items) {
			continue
		}
		selected := base.cursor >= 0 && base.cursor < len(base.filtered) && base.filtered[base.cursor] == idx
		staged := base.isStaged(base.items[idx])
		if state := uninstallFlowRowState(flow, base.items[idx], selected, staged); state != "" {
			stateCounts[state]++
		}
	}
	head := []string{style.Render(lane.label)}
	lines := []string{singleLine(strings.Join(head, "  "), max(width, 18))}
	if summary := uninstallFlowLaneSummaryLine(base, lane, stateCounts, width); summary != "" {
		lines = append(lines, summary)
	}
	return lines
}

func uninstallFlowLaneSummaryLine(base uninstallModel, lane uninstallFlowTargetLane, stateCounts map[string]int, width int) string {
	parts := []string{fmt.Sprintf("%d %s", len(lane.indices), pl(len(lane.indices), "target", "targets"))}
	if bytes := uninstallFlowLaneBytes(base, lane); bytes > 0 {
		parts = append(parts, cleanFlowHumanBytes(bytes))
	}
	for _, key := range []string{"live", "gate", "watch", "archive"} {
		if count := stateCounts[key]; count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", count, key))
		}
	}
	return panelMetaStyle.Render(singleLine(strings.Join(parts, " • "), max(width, 18)))
}

func uninstallFlowLaneBytes(base uninstallModel, lane uninstallFlowTargetLane) int64 {
	var total int64
	for _, idx := range lane.indices {
		if idx < 0 || idx >= len(base.items) {
			continue
		}
		total += uninstallFlowApproxBytes(base.items[idx])
	}
	return total
}

func uninstallFlowApproxBytes(item uninstallItem) int64 {
	if item.ApproxBytes > 0 {
		return item.ApproxBytes
	}
	parts := strings.Fields(strings.TrimSpace(item.SizeLabel))
	if len(parts) < 2 {
		return 0
	}
	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}
	switch strings.ToUpper(parts[1]) {
	case "KB":
		return int64(value * 1024)
	case "MB":
		return int64(value * 1024 * 1024)
	case "GB":
		return int64(value * 1024 * 1024 * 1024)
	case "TB":
		return int64(value * 1024 * 1024 * 1024 * 1024)
	default:
		return 0
	}
}

func uninstallFlowPhaseTrackLines(current string, width int) []string {
	segments := []string{}
	for _, phase := range []string{"TARGET RAIL", "REVIEW RAIL", "ACCESS RAIL", "HANDOFF RAIL", "AFTERCARE RAIL"} {
		label := routeStickyPhaseMeter("uninstall", phase)
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

func uninstallFlowSignalLabel(m uninstallFlowModel, item uninstallItem) string {
	switch m.phase {
	case uninstallFlowInventory:
		return "TARGET RAIL"
	case uninstallFlowReviewReady:
		return "REVIEW RAIL"
	case uninstallFlowPermissions:
		return "ACCESS RAIL"
	case uninstallFlowRemoving:
		if item.HasNative {
			return "HANDOFF RAIL"
		}
		return "REMNANT RAIL"
	case uninstallFlowResult:
		return "AFTERCARE RAIL"
	default:
		return "TARGET RAIL"
	}
}

func uninstallFlowViewportLines(m uninstallFlowModel, lines []string, focusLine int, maxLines int) []string {
	if maxLines <= 0 || len(lines) <= maxLines {
		return lines
	}
	if m.autoFollow || m.scrollOffset <= 0 {
		return viewportLines(lines, focusLine, maxLines)
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

func uninstallFlowSignalMotion(m uninstallFlowModel) motionState {
	mode := motionModeIdle
	phase := "target"
	switch m.phase {
	case uninstallFlowInventory:
		mode = motionModeLoading
		phase = "inventory"
	case uninstallFlowReviewReady:
		mode = motionModeReview
		phase = "review"
	case uninstallFlowPermissions:
		mode = motionModeAlert
		phase = "access"
	case uninstallFlowRemoving:
		mode = motionModeProgress
		phase = "handoff"
	case uninstallFlowResult:
		mode = motionModeIdle
		phase = "aftercare"
	}
	motion := newMotionState(m.spinnerFrame, m.pulse, mode, phase, "uninstall")
	if m.reducedMotion {
		return reducedMotionState(motion)
	}
	return motion
}

func uninstallFlowSignalLoad(item uninstallItem) float64 {
	if item.ApproxBytes <= 0 {
		return 28
	}
	load := float64(item.ApproxBytes) / float64(128<<20) * 100
	if load < 18 {
		load = 18
	}
	if load > 96 {
		load = 96
	}
	return load
}
