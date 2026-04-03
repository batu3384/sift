package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/batu3384/sift/internal/domain"
)

// progressStatsWithCategories returns stats cards with category breakdown (Mole-style)
func progressStatsWithCategories(progress progressModel, width int) []string {
	cardWidth := 22
	if width < 110 {
		cardWidth = width - 8
	}
	completed, _, failed, skipped, _ := countResultStatuses(domain.ExecutionResult{Items: progress.items})
	stage := progressStageInfo(progress)
	motion := progressMotionState(progress)
	state := fmt.Sprintf("%s %s", spinnerGlyph(motion), progressPhaseDisplay(progress.currentPhase, motion.Phase))
	if progress.cancelRequested {
		state = "Stopping"
	}
	if len(progress.items) == len(progress.plan.Items) && len(progress.plan.Items) > 0 {
		state = "Wrapping up"
	}
	freed := progressFreedBytes(progress)
	total := progressTotalBytes(progress.plan)

	// Calculate category progress
	categoryProgress := calculateCategoryProgress(progress)

	cards := []string{
		renderStatCard(progressStageCardLabel(progress.plan), progressStageCardValue(progress, stage), "review", cardWidth+4),
		renderStatCard(progressSettledCardLabel(progress.plan), fmt.Sprintf("%d / %d", len(progress.items), len(progress.plan.Items)), "review", cardWidth),
		renderStatCard("freed", fmt.Sprintf("%s / %s", domain.HumanBytes(freed), domain.HumanBytes(total)), "safe", cardWidth),
		renderStatCard("state", state, progressTone(completed, failed, skipped), cardWidth+6),
	}

	// Add category progress cards if we have multiple categories
	if len(categoryProgress) > 1 && width >= 140 {
		for cat, stats := range categoryProgress {
			progressPct := 0
			if stats.total > 0 {
				progressPct = int(float64(stats.completed) / float64(stats.total) * 100)
			}
			catLabel := string(cat)
			if len(catLabel) > 12 {
				catLabel = catLabel[:12] + ".."
			}
			catCard := renderStatCard(catLabel, fmt.Sprintf("%d%% %s", progressPct, domain.HumanBytes(stats.freed)), "safe", cardWidth)
			cards = append(cards, catCard)
		}
	}

	if width < 110 {
		return cards[:min(len(cards), 3)]
	}
	return cards
}

// CategoryProgress holds progress info for a single category
type CategoryProgress struct {
	total     int
	completed int
	freed     int64
}

// calculateCategoryProgress calculates progress per category
func calculateCategoryProgress(progress progressModel) map[domain.Category]CategoryProgress {
	result := make(map[domain.Category]CategoryProgress)

	// Count total items per category
	for _, item := range progress.plan.Items {
		stats := result[item.Category]
		stats.total++
		stats.freed += item.Bytes
		result[item.Category] = stats
	}

	// Count completed items per category
	for i := 0; i < len(progress.items) && i < len(progress.plan.Items); i++ {
		item := progress.plan.Items[i]
		status := progress.items[i].Status
		if status == domain.StatusDeleted || status == domain.StatusCompleted {
			if stats, ok := result[item.Category]; ok {
				stats.completed++
				result[item.Category] = stats
			}
		}
	}

	return result
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

	// Mole-style header with live freed bytes counter
	if freed > 0 {
		// Live counter animation
		progressPct := 0
		if total > 0 {
			progressPct = int(float64(freed) / float64(total) * 100)
		}
		lines = append(lines, safeStyle.Render(fmt.Sprintf("⬆ %s freed  [%d%%]  ● %d / %d", domain.HumanBytes(freed), progressPct, done, all)))
	} else {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("○ %d %s queued  ● %s total", all, pl(all, "item", "items"), domain.HumanBytes(total))))
	}

	// Add category progress bars
	categoryProgress := calculateCategoryProgress(progress)
	if len(categoryProgress) > 1 && width >= 100 {
		lines = append(lines, "")
		for cat, stats := range categoryProgress {
			catName := string(cat)
			if len(catName) > 15 {
				catName = catName[:15]
			}
			catPct := 0
			if stats.total > 0 {
				catPct = int(float64(stats.completed) / float64(stats.total) * 100)
			}
			// Draw progress bar
			barWidth := 20
			filled := (barWidth * catPct) / 100
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
			lines = append(lines, mutedStyle.Render(fmt.Sprintf("  %-15s [%s] %s", catName, bar, domain.HumanBytes(stats.freed))))
		}
	}

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
		icon := "·"
		lineStyle := mutedStyle
		isActive := progress.current != nil && idx == progress.cursor && progressPhaseActive(progress.currentPhase)
		if idx < len(progress.items) {
			switch progress.items[idx].Status {
			case domain.StatusDeleted, domain.StatusCompleted:
				icon = "✓"
				lineStyle = safeStyle
			case domain.StatusFailed:
				icon = "✗"
				lineStyle = highStyle
			case domain.StatusProtected, domain.StatusSkipped:
				icon = "⊘"
				lineStyle = reviewStyle
			default:
				icon = "·"
				lineStyle = mutedStyle
			}
		} else if isActive {
			icon = "⟳"
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
	if banner := progressLiveStageBanner(progress, stage, motion, width); banner != "" {
		lines = append(lines, banner)
	}
	stepStyle := mutedStyle
	if progressPhaseActive(progress.currentPhase) {
		stepStyle = reviewStyle
	}
	completed, deleted, failed, _, _ := countResultStatuses(domain.ExecutionResult{Items: progress.items})
	statusStyle := mutedStyle
	if failed > 0 {
		statusStyle = highStyle
	} else if completed+deleted > 0 {
		statusStyle = safeStyle
	}
	freed := progressFreedBytes(progress)
	total := progressTotalBytes(progress.plan)
	lines = append(lines,
		wrapText(mutedStyle.Render(progressSummaryLine(progress, stage)), width),
		wrapText(statusStyle.Render(progressStatusLine(progress)), width),
		wrapText(stepStyle.Render(progressStepLine(progress)), width),
		safeStyle.Render(fmt.Sprintf("Freed     %s / %s", domain.HumanBytes(freed), domain.HumanBytes(total))),
		mutedStyle.Render("Done      "+progressMeterLine(progress)),
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
		mutedStyle.Render("Run     "+describeAction(item.Action)),
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
			mutedStyle.Render("Status  "+string(result.Status)),
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

// progressLiveStageBanner returns a prominent single-line banner showing the
// active cleanup category, item progress, and bytes for this stage — the sift
// equivalent of Mole's animated "正在清理" display. Shown only when a stage is
// actively running so the user always knows what category is being cleaned.
func progressLiveStageBanner(progress progressModel, stage stageInfo, motion motionState, width int) string {
	if stage.Total == 0 || !progressPhaseActive(progress.currentPhase) {
		return ""
	}
	label := stage.Group
	if label == "" {
		label = sectionTitle(progress.plan, stage.Category)
	}
	spinner := spinnerGlyph(motion)
	pct := 0
	if stage.Items > 0 {
		pct = stage.Done * 100 / stage.Items
	}
	barWidth := 12
	filled := barWidth * pct / 100
	// Animated leading-edge: cycles ▒→▓→█→▓ to create a shimmer at the fill boundary.
	var bar string
	if filled <= 0 {
		bar = strings.Repeat("░", barWidth)
	} else if filled >= barWidth {
		bar = strings.Repeat("▓", barWidth)
	} else {
		edges := []string{"▒", "▓", "█", "▓"}
		edge := edges[motion.Frame%len(edges)]
		bar = strings.Repeat("▓", filled-1) + edge + strings.Repeat("░", barWidth-filled)
	}
	action := "CLEANING"
	switch progress.plan.Command {
	case "uninstall":
		action = "REMOVING"
	case "optimize":
		action = "APPLYING"
	case "autofix":
		action = "FIXING"
	}
	bytesLabel := fmt.Sprintf("%s / %s", domain.HumanBytes(stage.Freed), domain.HumanBytes(stage.Bytes))
	line := fmt.Sprintf("%s  %s  [%s]  %d/%d  •  %s  •  stage %d/%d",
		reviewStyle.Render(spinner+" "+action),
		headerStyle.Render(label),
		reviewStyle.Render(bar),
		stage.Done, stage.Items,
		safeStyle.Render(bytesLabel),
		stage.Index, stage.Total,
	)
	return wrapText(line, width)
}
