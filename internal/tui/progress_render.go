package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/batu3384/sift/internal/domain"
)

// progressStatsWithCategories returns the high-signal top cards for progress.
func progressStatsWithCategories(progress progressModel, width int) []string {
	cardWidth := 22
	if width < 110 {
		cardWidth = width - 8
	}
	stage := progressStageInfo(progress)
	freed := progressFreedBytes(progress)
	total := progressTotalBytes(progress.plan)

	cards := []string{
		renderRouteStatCard("progress", "progress", progressMeter(progress), "safe", cardWidth),
		renderRouteStatCard("progress", progressStageCardLabel(progress.plan), progressStageCardValue(progress, stage), "review", cardWidth+4),
		renderRouteStatCard("progress", progressSettledCardLabel(progress.plan), fmt.Sprintf("%d / %d", len(progress.items), len(progress.plan.Items)), "review", cardWidth),
		renderRouteStatCard("progress", "freed", fmt.Sprintf("%s / %s", domain.HumanBytes(freed), domain.HumanBytes(total)), "safe", cardWidth+2),
	}

	if width < 110 {
		return cards[:min(len(cards), 3)]
	}
	return cards
}

// progressStats is kept for backward compatibility
func progressStats(progress progressModel, width int) []string {
	return progressStatsWithCategories(progress, width)
}

// progressListViewMoleStyle returns a Mole-inspired progress view with live MB counter
func progressListViewMoleStyle(progress progressModel, width int, maxLines int) string {
	if len(progress.plan.Items) == 0 {
		return mutedStyle.Render("No execution items.")
	}
	stage := progressStageInfo(progress)
	freed := progressFreedBytes(progress)
	total := progressTotalBytes(progress.plan)
	done := len(progress.items)
	all := len(progress.plan.Items)
	lines := make([]string, 0, all+all/3+4)

	lines = append(lines, safeStyle.Render(fmt.Sprintf("Progress %s  •  %d/%d settled  •  %s / %s", progressMeter(progress), done, all, domain.HumanBytes(freed), domain.HumanBytes(total))))
	lines = append(lines, mutedStyle.Render(progressPhaseLine(progress, stage)))

	lines = append(lines, renderSectionRule(width))

	focusLine := 0
	currentCategory := domain.Category("")
	currentGroup := ""
	for idx, item := range progress.plan.Items {
		if item.Category != currentCategory {
			currentCategory = item.Category
			currentGroup = ""
			if len(lines) > 1 {
				lines = append(lines, renderSectionRule(width))
			}
			header := headerStyle.Render(sectionTitle(progress.plan, currentCategory))
			if stage.Category == currentCategory {
				header += "  " + reviewStyle.Render(fmt.Sprintf("STAGE %d/%d", stage.Index, stage.Total))
			}
			lines = append(lines, header)
			if idx == progress.cursor {
				focusLine = len(lines) - 1
			}
		}
		group := groupedItemLabel(item)
		if group != "" && group != currentGroup {
			currentGroup = group
			groupHeader := mutedStyle.Render("  " + group)
			if stage.Group == group {
				groupHeader = reviewStyle.Render("  " + group)
			}
			lines = append(lines, groupHeader)
		}
		icon := "-"
		lineStyle := mutedStyle
		isActive := progress.current != nil && idx == progress.cursor && progressPhaseActive(progress.currentPhase)
		if idx < len(progress.items) {
			switch progress.items[idx].Status {
			case domain.StatusDeleted, domain.StatusCompleted:
				icon = "+"
				lineStyle = safeStyle
			case domain.StatusFailed:
				icon = "x"
				lineStyle = highStyle
			case domain.StatusProtected, domain.StatusSkipped:
				icon = "s"
				lineStyle = reviewStyle
			default:
				icon = "-"
				lineStyle = mutedStyle
			}
		} else if isActive {
			icon = "~"
			lineStyle = reviewStyle
		}
		label := displayFindingLabel(item)
		bytesLabel := domain.HumanBytes(item.Bytes)
		// Mole-style: ✓ item name  size (compact format)
		line := selectionPrefix(idx == progress.cursor) + lineStyle.Render(fmt.Sprintf("%s %s  %s", icon, truncateText(label, 32), bytesLabel))
		if idx < len(progress.items) && progress.items[idx].Message != "" {
			line = fmt.Sprintf("%s  %s", line, mutedStyle.Render(truncateText(progress.items[idx].Message, max(width-48, 12))))
		} else if isActive && idx >= len(progress.items) {
			phaseLabel := progressPhaseSubtitle(progress.currentPhase)
			if phaseLabel == "" {
				phaseLabel = "queued…"
			} else {
				phaseLabel += "…"
			}
			line = fmt.Sprintf("%s  %s", line, reviewStyle.Render(phaseLabel))
		}
		line = singleLine(line, width)
		if idx == progress.cursor {
			line = selectedLine.Render(line)
			focusLine = len(lines)
		}
		lines = append(lines, line)
	}

	lines = progressFeedViewport(lines, focusLine, maxLines)
	return strings.Join(lines, "\n")
}

// progressListView is kept for backward compatibility
func progressListView(progress progressModel, width int, maxLines int) string {
	return progressListViewMoleStyle(progress, width, maxLines)
}

// progressFeedViewport is a tail-biased viewport: the cursor sits at the top
// third of the window so that completed items accumulate visibly above it,
// giving a flowing-log feel similar to Mole's downward-scrolling output.
func progressFeedViewport(lines []string, cursor int, maxLines int) []string {
	if maxLines <= 0 || len(lines) <= maxLines {
		return lines
	}
	// Place cursor at roughly top 1/3 so 2/3 of visible lines show history.
	start := cursor - (maxLines * 2 / 3)
	if start < 0 {
		start = 0
	}
	end := start + maxLines
	if end > len(lines) {
		end = len(lines)
		start = end - maxLines
		if start < 0 {
			start = 0
		}
	}
	window := append([]string{}, lines[start:end]...)
	if start > 0 {
		window[0] = mutedStyle.Render("…") + " " + strings.TrimLeft(window[0], " ")
	}
	if end < len(lines) {
		last := len(window) - 1
		window[last] = strings.TrimRight(window[last], " ") + " " + mutedStyle.Render("…")
	}
	return window
}

func progressDetailSubtitle(progress progressModel) string {
	if len(progress.plan.Items) == 0 {
		return "idle"
	}
	stage := progressStageInfo(progress)
	stageLabel := progressStageDetailLabel(progress)
	if progress.current != nil && progress.currentPhase == domain.ProgressPhaseStarting {
		return fmt.Sprintf("%s %d/%d • %s", stageLabel, stage.Index, stage.Total, progressPhaseSubtitle(progress.currentPhase))
	}
	if progress.current != nil && progressPhaseActive(progress.currentPhase) {
		return fmt.Sprintf("%s %d/%d • %s", stageLabel, stage.Index, stage.Total, progressPhaseSubtitle(progress.currentPhase))
	}
	if progress.cursor < len(progress.items) {
		return fmt.Sprintf("%s %d/%d • %s", stageLabel, stage.Index, stage.Total, strings.ToLower(string(progress.items[progress.cursor].Status)))
	}
	return fmt.Sprintf("%s %d/%d • queued", stageLabel, stage.Index, stage.Total)
}

func progressDetailView(progress progressModel, width int, maxLines int) string {
	stage := progressStageInfo(progress)
	motion := progressMotionState(progress)
	lines := []string{}
	if signal := progressRouteSignalLine(progress); signal != "" {
		lines = append(lines, signal)
	}
	stepStyle := mutedStyle
	if progressPhaseActive(progress.currentPhase) {
		stepStyle = reviewStyle
	}
	lines = append(lines,
		wrapText(mutedStyle.Render(progressSummaryLine(progress, stage)), width),
		mutedStyle.Render("Meter     "+progressMeterLine(progress)),
		wrapText(mutedStyle.Render(progressPhaseLine(progress, stage)), width),
		wrapText(stepStyle.Render(progressStepLine(progress)), width),
		wrapText(mutedStyle.Render(progressNextLine(progress)), width),
		wrapText(mutedStyle.Render(progressStatusLine(progress)), width),
	)
	if !progress.startedAt.IsZero() {
		lines = append(lines, mutedStyle.Render("Time      "+progressElapsed(progress.startedAt)))
	}
	if len(progress.plan.Items) == 0 {
		content := strings.Join(lines, "\n")
		if width >= 120 && maxLines >= 6 {
			cpu := float64(len(progress.items)) / float64(max(len(progress.plan.Items), 1)) * 100
			return lipgloss.JoinHorizontal(lipgloss.Top, content, "  ", mascotFrameFromMotion(motion, cpu))
		}
		return content
	}
	lines = append(lines, renderSectionRule(width), headerStyle.Render("Flow"))
	lines = append(lines, progressModuleFlowLines(progress, width)...)
	idx := progress.cursor
	if idx < 0 {
		idx = 0
	}
	if idx >= len(progress.plan.Items) {
		idx = len(progress.plan.Items) - 1
	}
	item := progress.plan.Items[idx]
	lines = append(lines, renderSectionRule(width),
		fmt.Sprintf("%s  %s", domain.HumanBytes(item.Bytes), item.DisplayPath),
		mutedStyle.Render("Action   "+describeAction(item.Action)),
	)
	if item.Action == domain.ActionNative {
		lines = append(lines, reviewStyle.Render("Native step  vendor uninstaller will be launched"))
	} else if item.Action == domain.ActionCommand {
		lines = append(lines, reviewStyle.Render("Managed task  system maintenance command will be executed"))
	}
	if progress.current != nil && progressPhaseActive(progress.currentPhase) && idx == progress.cursor {
		live := progress.currentDetail
		if live == "" {
			live = fmt.Sprintf("%s now", strings.ToLower(progressPhaseDisplay(progress.currentPhase, motion.Phase)))
		}
		lines = append(lines, reviewStyle.Render(fmt.Sprintf("%s %s", spinnerGlyph(motion), live)))
	}
	if idx < len(progress.items) {
		result := progress.items[idx]
		lines = append(lines,
			mutedStyle.Render("Result   "+string(result.Status)),
		)
		if result.Reason != "" {
			lines = append(lines, highStyle.Render("Reason  "+string(result.Reason)))
		}
		if result.Message != "" {
			lines = append(lines, wrapText(mutedStyle.Render(result.Message), width))
		}
	} else {
		lines = append(lines, reviewStyle.Render("Waiting"))
	}
	lines = append(lines, renderSectionRule(width), headerStyle.Render("Items"))
	lines = append(lines, progressCategoryLines(progress, width)...)
	lines = viewportLines(lines, 0, maxLines)
	content := strings.Join(lines, "\n")
	if width >= 120 && maxLines >= 6 {
		cpu := float64(len(progress.items)) / float64(max(len(progress.plan.Items), 1)) * 100
		return lipgloss.JoinHorizontal(lipgloss.Top, content, "  ", mascotFrameFromMotion(motion, cpu))
	}
	return content
}

func progressRouteSignalLine(progress progressModel) string {
	if !routeHasNamedSignal(progress.plan.Command) {
		return ""
	}
	signature := routeSignalSignatureForRoute(progress.plan.Command)
	if strings.TrimSpace(signature.Mascot) == "" {
		return ""
	}
	parts := []string{
		railStyle.Render(signature.Mascot),
		panelMetaStyle.Render(progressRouteSignalLabel(progress)),
	}
	if signature.Doctrine != "" {
		parts = append(parts, mutedStyle.Render(signature.Doctrine))
	}
	return strings.Join(parts, "  ")
}

// progressLiveStageBanner returns a prominent single-line banner showing the
// active cleanup category, item progress, and bytes for this stage — the sift
// equivalent of Mole's animated "正在清理" display. Shown only when a stage is
// actively running so the user always knows what category is being cleaned.
func progressRouteSignalLabel(progress progressModel) string {
	switch progress.plan.Command {
	case "uninstall":
		if item, ok := progressCurrentItem(progress); ok {
			if item.TaskPhase == "aftercare" || progress.currentPhase == domain.ProgressPhaseFinished {
				return "AFTERCARE RAIL"
			}
			if item.Action == domain.ActionNative {
				return "HANDOFF RAIL"
			}
			return "REMNANT RAIL"
		}
		if progress.currentPhase == domain.ProgressPhaseFinished {
			return "AFTERCARE RAIL"
		}
		return "TARGET RAIL"
	case "optimize":
		if progress.currentPhase == domain.ProgressPhaseFinished {
			return "SETTLED RAIL"
		}
		return "TASK RAIL"
	case "autofix":
		if progress.currentPhase == domain.ProgressPhaseFinished {
			return "SETTLED RAIL"
		}
		return "FIX RAIL"
	default:
		switch progress.currentPhase {
		case domain.ProgressPhasePreparing:
			return "ACCESS RAIL"
		case domain.ProgressPhaseVerifying:
			return "VERIFY RAIL"
		case domain.ProgressPhaseFinished:
			return "SETTLED RAIL"
		default:
			return "RECLAIM RAIL"
		}
	}
}

func progressCurrentItem(progress progressModel) (domain.Finding, bool) {
	if progress.current != nil {
		return *progress.current, true
	}
	if progress.cursor >= 0 && progress.cursor < len(progress.plan.Items) {
		return progress.plan.Items[progress.cursor], true
	}
	return domain.Finding{}, false
}
