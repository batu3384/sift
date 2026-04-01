package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/batuhanyuksel/sift/internal/domain"
)

// planActionCounts returns the number of ready (actionable, non-advisory) items
// and protected items in the plan. Used by both planStats and planSummary.
func planActionCounts(plan domain.ExecutionPlan) (actionable, protected int) {
	for _, item := range plan.Items {
		if item.Status == domain.StatusProtected {
			protected++
			continue
		}
		if item.Action != domain.ActionAdvisory {
			actionable++
		}
	}
	return actionable, protected
}

func planStats(plan domain.ExecutionPlan, width int) []string {
	cardWidth := 24
	if width < 110 {
		cardWidth = width - 8
	}
	actionable, protected := planActionCounts(plan)
	return []string{
		renderStatCard("reclaim", domain.HumanBytes(plan.Totals.Bytes), "safe", cardWidth),
		renderStatCard("ready", fmt.Sprintf("%d %s", actionable, pl(actionable, "item", "items")), "review", cardWidth),
		renderStatCard("protected", fmt.Sprintf("%d %s", protected, pl(protected, "item", "items")), "high", cardWidth),
	}
}

func planListView(plan domain.ExecutionPlan, cursor int, width int, maxLines int) string {
	if len(plan.Items) == 0 {
		return mutedStyle.Render("No findings in the current plan.")
	}
	lines := []string{mutedStyle.Render(planSummary(plan)), mutedStyle.Render(meterLine(plan.Totals))}
	focusLine := 0
	currentCategory := domain.Category("")
	currentGroup := ""
	for i, item := range plan.Items {
		if item.Category != currentCategory {
			currentCategory = item.Category
			currentGroup = ""
			lines = append(lines, "", headerStyle.Render(sectionTitle(plan, currentCategory)))
		}
		group := groupedItemLabel(item)
		if group != "" && group != currentGroup {
			currentGroup = group
			lines = append(lines, mutedStyle.Render("  "+group))
		}
		line := selectionPrefix(i == cursor) + planListLine(plan, item)
		line = singleLine(line, width)
		if i == cursor {
			line = selectedLine.Render(line)
			focusLine = len(lines)
		}
		lines = append(lines, line)
	}
	lines = viewportLines(lines, focusLine, maxLines)
	return strings.Join(lines, "\n")
}

func planListLine(plan domain.ExecutionPlan, item domain.Finding) string {
	icon, lineStyle := planStatusIcon(item)
	label := displayFindingLabel(item)
	bytesLabel := domain.HumanBytes(item.Bytes)
	core := lineStyle.Render(fmt.Sprintf("%s  %-28s  %8s", icon, truncateText(label, 28), bytesLabel))
	if item.Action == domain.ActionNative {
		core += mutedStyle.Render("  native")
	} else if item.Action == domain.ActionCommand {
		core += mutedStyle.Render("  task")
	} else if item.Action == domain.ActionSkip {
		core += mutedStyle.Render("  excluded")
	}
	if item.Status == domain.StatusProtected && item.Policy.Reason != "" {
		core += "  " + highStyle.Render(string(item.Policy.Reason))
	}
	return core
}

func planStatusIcon(item domain.Finding) (string, lipgloss.Style) {
	if item.Status == domain.StatusProtected {
		return "⊘", highStyle
	}
	if item.Action == domain.ActionSkip {
		return "⊘", mutedStyle
	}
	switch item.Risk {
	case domain.RiskSafe:
		return "✓", safeStyle
	case domain.RiskHigh:
		return "!", highStyle
	default:
		return "~", reviewStyle
	}
}

func planDetailSubtitle(plan domain.ExecutionPlan, cursor int) string {
	if cursor < 0 || cursor >= len(plan.Items) {
		return planSummary(plan)
	}
	item := plan.Items[cursor]
	return fmt.Sprintf("%s • %s", sectionTitle(plan, item.Category), item.Status)
}

func planDetailView(model planModel, width int, maxLines int) string {
	plan := model.effectivePlan()
	cursor := model.cursor
	lines := make([]string, 0, 16)
	if len(plan.Items) == 0 || cursor < 0 || cursor >= len(plan.Items) {
		lines = append(lines, mutedStyle.Render("No item selected."))
	} else {
		item := plan.Items[cursor]
		icon, iconStyle := planStatusIcon(item)
		label := displayFindingLabel(item)
		bytesLabel := domain.HumanBytes(item.Bytes)
		riskStr := strings.TrimSpace(string(item.Risk))
		riskPart := ""
		if riskStr != "" {
			riskPart = mutedStyle.Render("  •  ") + styleForRisk(item.Risk).Render(strings.ToUpper(riskStr))
		}
		lines = append(lines,
			iconStyle.Render(fmt.Sprintf("%s  %s", icon, label))+"  "+headerStyle.Render(bytesLabel),
			mutedStyle.Render(resultStatusLabel(item.Status))+riskPart+mutedStyle.Render("  •  "+planActionLabel(item.Action)),
		)
		if item.Action == domain.ActionSkip {
			lines = append(lines, reviewStyle.Render("Excluded from this run"))
		} else if canToggleReviewItem(item) {
			lines = append(lines, mutedStyle.Render("Included in this run"))
		}
		if item.Source != "" {
			lines = append(lines, mutedStyle.Render("From     "+trimAnalyzeSource(item.Source)))
		}
		if item.TaskPhase != "" || item.TaskImpact != "" {
			taskParts := make([]string, 0, 2)
			if item.TaskPhase != "" {
				taskParts = append(taskParts, strings.ToUpper(item.TaskPhase))
			}
			if item.TaskImpact != "" {
				taskParts = append(taskParts, item.TaskImpact)
			}
			lines = append(lines, mutedStyle.Render("Task    "+strings.Join(taskParts, "  •  ")))
		}
		if len(item.SuggestedBy) > 0 {
			lines = append(lines, mutedStyle.Render("Suggested  "+strings.Join(item.SuggestedBy, ", ")))
		}
		if !item.LastModified.IsZero() {
			lines = append(lines, mutedStyle.Render("Modified  "+item.LastModified.Local().Format("2006-01-02 15:04")))
		}
		if item.Policy.Reason != "" {
			lines = append(lines, highStyle.Render("Why      "+string(item.Policy.Reason)))
		}
		if item.Action == domain.ActionNative && item.NativeCommand != "" {
			lines = append(lines, mutedStyle.Render("Run      "+item.NativeCommand))
		}
		if item.Action == domain.ActionCommand {
			command := strings.TrimSpace(strings.Join(append([]string{item.CommandPath}, item.CommandArgs...), " "))
			if command != "" {
				lines = append(lines, mutedStyle.Render("Run      "+command))
			}
		}
		if len(item.TaskVerify) > 0 {
			lines = append(lines, renderSectionRule(width), headerStyle.Render("Verify"))
			for _, verify := range item.TaskVerify[:min(len(item.TaskVerify), 2)] {
				lines = append(lines, mutedStyle.Render("• "+verify))
			}
		}
		if boardLines := planCommandBoardLines(plan, width); len(boardLines) > 0 {
			lines = append(lines, renderSectionRule(width), headerStyle.Render(planCommandBoardTitle(plan.Command)))
			lines = append(lines, boardLines...)
		}
		if group := model.currentGroupSummary(); group.total > 0 {
			lines = append(lines, renderSectionRule(width), headerStyle.Render(planCurrentGroupTitle(plan.Command)))
			lines = append(lines, mutedStyle.Render(planCurrentGroupSummaryLine(plan.Command, group)))
			lines = append(lines, mutedStyle.Render(planCurrentGroupActionLine(plan.Command)))
		}
		if plan.Command == "uninstall" {
			lines = append(lines, renderSectionRule(width), headerStyle.Render("Target Batch"))
			lines = append(lines, uninstallTargetSummaryLines(plan, width)...)
		}
	}
	if len(plan.Warnings) > 0 {
		lines = append(lines, renderSectionRule(width), headerStyle.Render("Plan Warnings"))
		for _, warning := range plan.Warnings {
			lines = append(lines, wrapText(mutedStyle.Render("• "+warning), width))
		}
	}
	if plan.Command == "analyze" {
		lines = append(lines, renderSectionRule(width), headerStyle.Render("Insights"))
		lines = append(lines, analyzeSummaryLines(plan)...)
	}
	lines = viewportLines(lines, 0, maxLines)
	return strings.Join(lines, "\n")
}

func planActionLabel(action domain.Action) string {
	switch action {
	case domain.ActionTrash:
		return "trash"
	case domain.ActionPermanent:
		return "delete"
	case domain.ActionNative:
		return "native uninstall"
	case domain.ActionCommand:
		return "task"
	case domain.ActionSkip:
		return "excluded"
	default:
		return "advisory"
	}
}

func decisionSubtitle(plan domain.ExecutionPlan) string {
	if plan.Command == "optimize" || plan.Command == "autofix" {
		tasks := actionableCount(plan)
		phases := maintenancePhaseCount(plan)
		return fmt.Sprintf("%d %s • %d suggested • %d %s", tasks, pl(tasks, "task", "tasks"), suggestedTaskCount(plan), phases, pl(phases, "phase", "phases"))
	}
	scopeSingular, scopeLabel := "module", "modules"
	if plan.Command == "uninstall" {
		scopeSingular, scopeLabel = "target", "targets"
	}
	mods := planModuleCount(plan)
	return fmt.Sprintf("%d ready • %d %s • %s", actionableCount(plan), mods, pl(mods, scopeSingular, scopeLabel), domain.HumanBytes(plan.Totals.Bytes))
}

func decisionView(model planModel, width int) string {
	plan := model.effectivePlan()
	lines := []string{
		reviewStyle.Render("Keys    space toggle  •  m toggle module  •  y run  •  esc back"),
	}
	if scope := planReviewScopeLine(plan); scope != "" {
		lines = append(lines, wrapText(mutedStyle.Render("Scope   "+scope), width))
	}
	if outcome := planReviewOutcomeLine(plan); outcome != "" {
		lines = append(lines, wrapText(mutedStyle.Render("Run     "+outcome), width))
	}
	preflight := buildPermissionPreflight(plan, "")
	if preflight.required() {
		lines = append(lines, wrapText(mutedStyle.Render(preflight.accessSummaryLine()), width))
		if manifest := preflight.manifestSummaryLine(width); manifest != "" {
			lines = append(lines, mutedStyle.Render(manifest))
		}
	}
	if plan.Command == "uninstall" {
		lines = append(lines, renderSectionRule(width), headerStyle.Render("Target Batch"))
		lines = append(lines, uninstallTargetSummaryLines(plan, width)...)
	}
	if plan.Command != "analyze" {
		moduleLines := planModuleLines(plan, width, 4)
		if len(moduleLines) > 0 {
			lines = append(lines, renderSectionRule(width), headerStyle.Render(planGroupCollectionTitle(plan.Command)))
			lines = append(lines, moduleLines...)
		}
	}
	if boardLines := planCommandBoardLines(plan, width); len(boardLines) > 0 {
		lines = append(lines, renderSectionRule(width), headerStyle.Render(planCommandBoardTitle(plan.Command)))
		lines = append(lines, boardLines...)
	}
	lines = append(lines, planDecisionWarningLines(plan.Warnings, width)...)
	return strings.Join(lines, "\n")
}

func planDecisionWarningLines(warnings []string, width int) []string {
	if len(warnings) == 0 {
		return nil
	}
	if len(warnings) == 1 {
		return []string{"", wrapText(mutedStyle.Render("Warning  "+warnings[0]), width)}
	}
	const maxShown = 2
	lines := []string{"", headerStyle.Render("Warnings")}
	shown := warnings[:min(len(warnings), maxShown)]
	for _, warning := range shown {
		lines = append(lines, wrapText(mutedStyle.Render("• "+warning), width))
	}
	if len(warnings) > maxShown {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("  …and %d more", len(warnings)-maxShown)))
	}
	return lines
}

func actionableCount(plan domain.ExecutionPlan) int {
	count := 0
	for _, item := range plan.Items {
		if item.Status == domain.StatusProtected {
			continue
		}
		if item.Action != domain.ActionAdvisory && item.Action != domain.ActionSkip {
			count++
		}
	}
	return count
}

func describeAction(action domain.Action) string {
	switch action {
	case domain.ActionTrash:
		return "Move to trash / recycle bin"
	case domain.ActionPermanent:
		return "Permanent delete"
	case domain.ActionNative:
		return "Launch native uninstaller"
	case domain.ActionCommand:
		return "Run managed maintenance command"
	default:
		return "Advisory only"
	}
}

func planSummary(plan domain.ExecutionPlan) string {
	actionable, protected := planActionCounts(plan)
	parts := []string{
		fmt.Sprintf("Ready %d", actionable),
		fmt.Sprintf("Protected %d", protected),
		fmt.Sprintf("Safe %s", domain.HumanBytes(plan.Totals.SafeBytes)),
		fmt.Sprintf("Review %s", domain.HumanBytes(plan.Totals.ReviewBytes)),
	}
	if plan.Totals.HighBytes > 0 {
		parts = append(parts, fmt.Sprintf("High %s", domain.HumanBytes(plan.Totals.HighBytes)))
	}
	return strings.Join(parts, "  •  ")
}

func planModuleLines(plan domain.ExecutionPlan, width int, limit int) []string {
	if limit <= 0 {
		return nil
	}
	type bucket struct {
		label string
		items int
		bytes int64
	}
	order := make([]string, 0)
	buckets := map[string]*bucket{}
	for _, item := range plan.Items {
		if item.Action == domain.ActionAdvisory {
			continue
		}
		key := progressGroupKey(item)
		if _, ok := buckets[key]; !ok {
			label := domain.ExecutionGroupLabel(item)
			if label == "" {
				label = sectionTitle(plan, item.Category)
			}
			buckets[key] = &bucket{label: label}
			order = append(order, key)
		}
		buckets[key].items++
		buckets[key].bytes += item.Bytes
	}
	if len(order) == 0 {
		return nil
	}
	lines := make([]string, 0, min(len(order), limit)+1)
	lines = append(lines, mutedStyle.Render(fmt.Sprintf("%d %s  •  %s total", len(order), planGroupCollectionSummaryLabel(plan.Command), domain.HumanBytes(plan.Totals.Bytes))))
	for _, key := range order[:min(len(order), limit)] {
		b := buckets[key]
		lines = append(lines, wrapText(safeStyle.Render(fmt.Sprintf("✓ %-20s  %d %s  •  %s", b.label, b.items, pl(b.items, "item", "items"), domain.HumanBytes(b.bytes))), width))
	}
	if len(order) > limit {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("+%d more %s", len(order)-limit, planGroupCollectionOverflowLabel(plan.Command))))
	}
	return lines
}

func planCurrentGroupTitle(command string) string {
	switch command {
	case "uninstall":
		return "Target"
	case "optimize", "autofix":
		return "Phase"
	default:
		return "Module"
	}
}

func planCurrentGroupSummaryLine(command string, group reviewGroupSummary) string {
	switch command {
	case "optimize", "autofix":
		return fmt.Sprintf("%s • %d/%d ready • %s", group.label, group.included, group.total, domain.HumanBytes(group.bytes))
	default:
		return fmt.Sprintf("%s • %d/%d included • %s", group.label, group.included, group.total, domain.HumanBytes(group.bytes))
	}
}

func planCurrentGroupActionLine(command string) string {
	switch command {
	case "uninstall":
		return "m toggles current target"
	case "optimize", "autofix":
		return "m toggles current phase"
	default:
		return "m toggles current module"
	}
}

func planGroupCollectionTitle(command string) string {
	switch command {
	case "uninstall":
		return "Targets"
	case "optimize", "autofix":
		return "Phases"
	default:
		return "Modules"
	}
}

func planGroupCollectionSummaryLabel(command string) string {
	switch command {
	case "uninstall":
		return "targets"
	case "optimize", "autofix":
		return "phases"
	default:
		return "cleanup modules"
	}
}

func planGroupCollectionOverflowLabel(command string) string {
	switch command {
	case "uninstall":
		return "targets"
	case "optimize", "autofix":
		return "phases"
	default:
		return "modules"
	}
}

func suggestedTaskCount(plan domain.ExecutionPlan) int {
	count := 0
	for _, item := range plan.Items {
		if len(item.SuggestedBy) > 0 {
			count++
		}
	}
	return count
}

func maintenancePhaseCount(plan domain.ExecutionPlan) int {
	seen := map[string]struct{}{}
	for _, item := range plan.Items {
		phase := strings.TrimSpace(item.TaskPhase)
		if phase == "" {
			continue
		}
		seen[phase] = struct{}{}
	}
	return len(seen)
}

func meterLine(totals domain.Totals) string {
	total := totals.SafeBytes + totals.ReviewBytes + totals.HighBytes
	if total <= 0 {
		return mutedStyle.Render("[░░░░░░░░░░░░░░░░░░░░]")
	}
	width := 20
	safePortion := int(float64(totals.SafeBytes) / float64(total) * float64(width))
	reviewPortion := int(float64(totals.ReviewBytes) / float64(total) * float64(width))
	highPortion := width - safePortion - reviewPortion
	if highPortion < 0 {
		highPortion = 0
	}
	bar := safeStyle.Render(strings.Repeat("▓", safePortion)) +
		reviewStyle.Render(strings.Repeat("▓", reviewPortion)) +
		highStyle.Render(strings.Repeat("▓", highPortion))
	remaining := width - safePortion - reviewPortion - highPortion
	if remaining > 0 {
		bar += mutedStyle.Render(strings.Repeat("░", remaining))
	}
	legend := safeStyle.Render(domain.HumanBytes(totals.SafeBytes)) + mutedStyle.Render(" safe  ") +
		reviewStyle.Render(domain.HumanBytes(totals.ReviewBytes)) + mutedStyle.Render(" review")
	if totals.HighBytes > 0 {
		legend += "  " + highStyle.Render(domain.HumanBytes(totals.HighBytes)) + mutedStyle.Render(" high")
	}
	return fmt.Sprintf("[%s]  %s", bar, legend)
}
