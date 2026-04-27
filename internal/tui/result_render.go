package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/batu3384/sift/internal/domain"
)

func resultStats(plan domain.ExecutionPlan, result domain.ExecutionResult, width int) []string {
	cardWidth := 22
	if width < 110 {
		cardWidth = width - 8
	}
	completed, deleted, failed, _, protected := countResultStatuses(result)
	totalFreed := resultFreedBytes(plan, result)
	return []string{
		renderRouteStatCard("result", resultCompletedCardLabel(plan), fmt.Sprintf("%d", completed), "safe", cardWidth),
		renderRouteStatCard("result", resultChangedCardLabel(plan), fmt.Sprintf("%d", resultChangedCount(plan, completed, deleted)), "safe", cardWidth),
		renderRouteStatCard("result", "issues", fmt.Sprintf("%d", protected+failed), "review", cardWidth),
		renderRouteStatCard("result", "freed", domain.HumanBytes(totalFreed), "safe", cardWidth),
	}
}

func resultListView(model resultModel, width int, maxLines int) string {
	visible := model.visibleIndices()
	if len(model.result.Items) == 0 {
		return mutedStyle.Render("No execution items.")
	}
	if len(visible) == 0 {
		return mutedStyle.Render("No items match the current filter.")
	}
	lines := make([]string, 0, len(visible))
	for visibleIdx, resultIdx := range visible {
		item := model.result.Items[resultIdx]
		icon, lineStyle := resultStatusIcon(item.Status)
		planItem, hasPlan := resultPlanItem(model.plan, item)
		label := strings.TrimSpace(item.Path)
		if hasPlan {
			if dl := displayFindingLabel(planItem); dl != "" {
				label = dl
			}
		}
		bytesLabel := ""
		if hasPlan && planItem.Bytes > 0 {
			bytesLabel = domain.HumanBytes(planItem.Bytes)
		}
		labelWidth := rowLabelWidth(width, 28)
		line := selectionPrefix(visibleIdx == model.cursor) + lineStyle.Render(fmt.Sprintf("%s  %-*s  %8s", icon, labelWidth, truncateText(label, labelWidth), bytesLabel))
		if item.Reason != "" {
			line = fmt.Sprintf("%s  %s", line, mutedStyle.Render(truncateText(string(item.Reason), max(width-48, 12))))
		}
		line = singleLine(line, width)
		if visibleIdx == model.cursor {
			line = selectedLine.Render(line)
		}
		lines = append(lines, line)
	}
	lines = viewportLines(lines, model.cursor, maxLines)
	return strings.Join(lines, "\n")
}

func resultLineParts(item domain.OperationResult) []string {
	icon, _ := resultStatusIcon(item.Status)
	parts := []string{icon + " " + resultStatusLabel(item.Status)}
	if path := strings.TrimSpace(item.Path); path != "" {
		parts = append(parts, path)
	}
	return parts
}

func resultStatusIcon(status domain.FindingStatus) (string, lipgloss.Style) {
	switch status {
	case domain.StatusDeleted, domain.StatusCompleted:
		return "✓", safeStyle
	case domain.StatusFailed:
		return "✗", highStyle
	case domain.StatusProtected, domain.StatusSkipped:
		return "⊘", reviewStyle
	default:
		return "·", mutedStyle
	}
}

func resultStatusLabel(status domain.FindingStatus) string {
	value := strings.TrimSpace(string(status))
	if value == "" {
		return "Item"
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func resultDetailSubtitle(model resultModel) string {
	label := strings.ToUpper(string(coalesceResultFilter(model.filter)))
	candidates := resultRecoveryCandidates(model.plan, model.result, model.filter)
	failed := resultRecoveryCandidatesForStatuses(model.plan, model.result, model.filter, domain.StatusFailed)
	group := resultCurrentGroupRecoveryCandidates(model)
	track := resultTrackSubtitle(model)
	if len(failed) > 0 {
		return label + " • " + track + " • retry ready"
	}
	if len(candidates) > 0 {
		if len(group) > 0 {
			return label + " • " + track + " • current + recovery"
		}
		return label + " • " + track + " • recovery ready"
	}
	if len(model.result.FollowUpCommands) > 0 {
		return label + " • " + track + " • commands"
	}
	if len(model.result.Warnings) > 0 {
		return label + " • " + track + " • warnings"
	}
	return label + " • " + track + " • settled"
}

func resultDetailView(model resultModel, width int, maxLines int) string {
	result := model.result
	visible := model.visibleIndices()
	candidates := resultRecoveryCandidates(model.plan, result, model.filter)
	failed := resultRecoveryCandidatesForStatuses(model.plan, result, model.filter, domain.StatusFailed)
	group := resultCurrentGroupRecoveryCandidates(model)
	freed := resultFreedBytes(model.plan, result)
	totalBytes := progressTotalBytes(model.plan)
	lines := []string{}
	if signal := resultRouteSignalLine(model); signal != "" {
		lines = append(lines, signal)
	}
	if totalBytes > 0 || freed > 0 {
		lines = append(lines, safeStyle.Render(fmt.Sprintf("Space freed: %s / %s", domain.HumanBytes(freed), domain.HumanBytes(totalBytes))))
	}
	lines = append(lines, resultPriorityNoticeLines(result, width)...)
	lines = append(lines,
		wrapText(mutedStyle.Render(resultSummaryLine(model)), width),
		wrapText(mutedStyle.Render(resultStatusLine(model)), width),
		wrapText(mutedStyle.Render(resultNotTouchedLine(result)), width),
		wrapText(mutedStyle.Render(resultScopeLine(model)), width),
		wrapText(mutedStyle.Render(resultTrackLine(model)), width),
		wrapText(mutedStyle.Render(resultNextLine(model)), width),
	)
	if model.plan.Command == "uninstall" {
		lines = append(lines, renderSectionRule(width), headerStyle.Render("Target Batch"))
		lines = append(lines, uninstallTargetSummaryLines(model.plan, width)...)
	}
	if len(visible) > 0 && model.cursor >= 0 && model.cursor < len(visible) {
		item := result.Items[visible[model.cursor]]
		lines = append(lines, renderSectionRule(width), headerStyle.Render("Selected"), wrapText(mutedStyle.Render(strings.Join(resultLineParts(item), " • ")), width))
		if item.Reason != "" {
			lines = append(lines, highStyle.Render("Why      "+string(item.Reason)))
		}
		if item.Message != "" {
			lines = append(lines, mutedStyle.Render(item.Message))
		}
	}
	if catLines := resultCategorySummaryLines(model.plan, result, width); len(catLines) > 0 {
		lines = append(lines, renderSectionRule(width), headerStyle.Render("Sections"))
		lines = append(lines, catLines...)
	}
	if len(candidates) > 0 {
		lines = append(lines, renderSectionRule(width), headerStyle.Render(resultRecoveryTitle(model.plan)))
		lines = append(lines, resultRecoveryDetailLines(model.plan, candidates, width)...)
		lines = append(lines, mutedStyle.Render(resultRecoveryActionLine(model.plan)))
	}
	if len(group) > 0 {
		lines = append(lines, renderSectionRule(width), headerStyle.Render(resultCurrentTitle(model.plan)))
		lines = append(lines, resultRecoveryDetailLines(model.plan, group, width)...)
		lines = append(lines, mutedStyle.Render(resultCurrentActionLine(model.plan)))
	}
	if boardLines := planCommandBoardLines(model.plan, width); len(boardLines) > 0 {
		lines = append(lines, renderSectionRule(width), headerStyle.Render(planCommandBoardTitle(model.plan.Command)))
		lines = append(lines, boardLines...)
	}
	if len(failed) > 0 {
		lines = append(lines, renderSectionRule(width), headerStyle.Render(resultRetryTitle(model.plan)))
		lines = append(lines, resultRecoveryDetailLines(model.plan, failed, width)...)
		lines = append(lines, mutedStyle.Render(resultRetryActionLine(model.plan)))
	}
	lines = viewportLines(lines, 0, maxLines)
	return strings.Join(lines, "\n")
}

func resultRouteSignalLine(model resultModel) string {
	if !routeHasNamedSignal(model.plan.Command) {
		return ""
	}
	signature := routeSignalSignatureForRoute(model.plan.Command)
	if strings.TrimSpace(signature.Mascot) == "" {
		return ""
	}
	parts := []string{
		railStyle.Render(signature.Mascot),
		panelMetaStyle.Render(resultRouteSignalLabel(model.plan.Command)),
	}
	if signature.Doctrine != "" {
		parts = append(parts, mutedStyle.Render(signature.Doctrine))
	}
	return strings.Join(parts, "  ")
}

func resultRouteSignalLabel(command string) string {
	switch strings.TrimSpace(strings.ToLower(command)) {
	case "uninstall":
		return "AFTERCARE RAIL"
	default:
		return "SETTLED RAIL"
	}
}

// resultCategorySummaryLines builds a per-category breakdown of what was
// cleaned and how many bytes were reclaimed — shown in the result detail panel
// as a "Sections" subsection so users can see exactly what was freed per group.
func resultCategorySummaryLines(plan domain.ExecutionPlan, result domain.ExecutionResult, width int) []string {
	type bucket struct {
		label   string
		settled int
		failed  int
		bytes   int64
	}
	order := make([]string, 0)
	buckets := map[string]*bucket{}
	for _, item := range result.Items {
		planItem, ok := resultPlanItem(plan, item)
		if !ok {
			continue
		}
		key := progressGroupKey(planItem)
		if key == "" {
			key = sectionTitle(plan, planItem.Category)
		}
		label := sectionTitle(plan, planItem.Category)
		if gl := groupedItemLabel(planItem); gl != "" {
			label = gl
		}
		if _, ok := buckets[key]; !ok {
			order = append(order, key)
			buckets[key] = &bucket{label: label}
		}
		switch item.Status {
		case domain.StatusDeleted:
			buckets[key].settled++
			buckets[key].bytes += planItem.Bytes
		case domain.StatusCompleted:
			buckets[key].settled++
		default:
			buckets[key].failed++
		}
	}
	if len(order) == 0 {
		return nil
	}
	lines := make([]string, 0, len(order))
	for _, key := range order {
		b := buckets[key]
		if b.settled > 0 && b.failed == 0 {
			line := fmt.Sprintf("✓ %-20s %d settled  •  %s freed", b.label, b.settled, domain.HumanBytes(b.bytes))
			lines = append(lines, wrapText(safeStyle.Render(line), width))
		} else if b.settled > 0 {
			line := fmt.Sprintf("~ %-20s %d settled  •  %s freed  •  %d open", b.label, b.settled, domain.HumanBytes(b.bytes), b.failed)
			lines = append(lines, wrapText(reviewStyle.Render(line), width))
		} else {
			line := fmt.Sprintf("· %-20s %d open", b.label, b.failed)
			lines = append(lines, wrapText(mutedStyle.Render(line), width))
		}
	}
	return lines
}

func resultWarningLines(warnings []string, width int) []string {
	if len(warnings) == 0 {
		return nil
	}
	if len(warnings) == 1 {
		return []string{"", wrapText(mutedStyle.Render("Warning  "+warnings[0]), width)}
	}
	lines := []string{"", headerStyle.Render("Warnings")}
	for _, warning := range warnings {
		lines = append(lines, wrapText(mutedStyle.Render("• "+warning), width))
	}
	return lines
}

func resultPriorityNoticeLines(result domain.ExecutionResult, width int) []string {
	lines := make([]string, 0, len(result.Warnings)+len(result.FollowUpCommands))
	for _, warning := range result.Warnings {
		lines = append(lines, wrapText(mutedStyle.Render("Warning  "+warning), width))
	}
	for _, command := range result.FollowUpCommands {
		lines = append(lines, wrapText(mutedStyle.Render("Run      "+command), width))
	}
	return lines
}

func resultCommandLines(commands []string, width int) []string {
	if len(commands) == 0 {
		return nil
	}
	if len(commands) == 1 {
		return []string{"", wrapText(mutedStyle.Render("Run      "+commands[0]), width)}
	}
	lines := []string{"", headerStyle.Render("Run")}
	for _, command := range commands {
		lines = append(lines, wrapText(mutedStyle.Render("• "+command), width))
	}
	return lines
}

func styleForRisk(risk domain.Risk) lipgloss.Style {
	switch risk {
	case domain.RiskSafe:
		return safeStyle
	case domain.RiskReview:
		return reviewStyle
	default:
		return highStyle
	}
}
