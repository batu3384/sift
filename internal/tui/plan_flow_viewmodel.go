package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/batu3384/sift/internal/domain"
)

// planActionCounts returns the number of ready (actionable, non-advisory) items
// and protected items in the plan. Used by both planStats and planSummary.
func planActionCounts(plan domain.ExecutionPlan) (actionable, protected int) {
	for _, item := range plan.Items {
		if item.Status == domain.StatusProtected {
			protected++
			continue
		}
		if actionableDisplayItem(item) {
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
		renderRouteStatCard("review", "actionable", domain.HumanBytes(planDisplayBytes(plan)), "safe", cardWidth),
		renderRouteStatCard("review", "ready", fmt.Sprintf("%d %s", actionable, pl(actionable, "item", "items")), "review", cardWidth),
		renderRouteStatCard("review", "protected", fmt.Sprintf("%d %s", protected, pl(protected, "item", "items")), "high", cardWidth),
	}
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

func planDetailStatusLine(plan domain.ExecutionPlan, item domain.Finding) string {
	if item.Status == domain.StatusProtected {
		switch plan.Command {
		case "uninstall":
			return "target stays under watch"
		case "analyze":
			return "trace stays under watch"
		case "optimize", "autofix":
			return "task stays under watch"
		default:
			return "review item stays under watch"
		}
	}
	if item.Action == domain.ActionSkip {
		switch plan.Command {
		case "uninstall":
			return "target held outside this run"
		case "analyze":
			return "trace held outside this run"
		case "optimize", "autofix":
			return "task held outside this run"
		default:
			return "review item held outside this run"
		}
	}
	switch plan.Command {
	case "uninstall":
		return "target handoff ready"
	case "analyze":
		return "trace review ready"
	case "optimize", "autofix":
		return "task review ready"
	default:
		return "review item ready"
	}
}

func planDetailScopeLine(model planModel, item domain.Finding) string {
	if group := model.currentGroupSummary(); group.total > 0 {
		return planCurrentGroupSummaryLine(model.plan.Command, group)
	}
	label := groupedItemLabel(item)
	if strings.TrimSpace(label) == "" {
		label = sectionTitle(model.plan, item.Category)
	}
	return fmt.Sprintf("%s • %s", label, domain.HumanBytes(item.Bytes))
}

func planDetailNextLine(plan domain.ExecutionPlan, item domain.Finding) string {
	groupAction := planCurrentGroupActionLine(plan.Command)
	subject := planReviewSubject(plan.Command)
	if canToggleReviewItem(item) {
		return fmt.Sprintf("space toggles this %s • %s", subject, groupAction)
	}
	if item.Action == domain.ActionSkip {
		return fmt.Sprintf("space restores this %s • %s", subject, groupAction)
	}
	return fmt.Sprintf("watch policy keeps this %s • %s", subject, groupAction)
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
	return fmt.Sprintf("%d ready • %d %s • %s", actionableCount(plan), mods, pl(mods, scopeSingular, scopeLabel), domain.HumanBytes(planDisplayBytes(plan)))
}

func planDecisionStatusLine(plan domain.ExecutionPlan) string {
	switch strings.TrimSpace(plan.Command) {
	case "clean", "installer", "purge":
		return "review gate ready"
	case "uninstall":
		return "handoff gate ready"
	case "analyze":
		return "trace gate ready"
	case "optimize", "autofix":
		return "task gate ready"
	default:
		return "run gate ready"
	}
}

func planDecisionGateLine(plan domain.ExecutionPlan) string {
	subject := planReviewSubject(plan.Command)
	return fmt.Sprintf("space toggles this %s • %s", subject, planCurrentGroupActionLine(plan.Command))
}

func planTrustSummaryLines(plan domain.ExecutionPlan, width int) []string {
	actionable, protected, skipped := planTrustCounts(plan)
	lines := []string{
		wrapText(mutedStyle.Render(fmt.Sprintf("Will touch %d approved %s", actionable, pl(actionable, "item", "items"))), width),
	}
	if protected > 0 || skipped > 0 {
		lines = append(lines,
			wrapText(mutedStyle.Render("Safe    protected and excluded items stay out of this run"), width),
			wrapText(mutedStyle.Render("Not touched  "+planNotTouchedSummary(protected, skipped)), width),
		)
	}
	return lines
}

func planTrustCounts(plan domain.ExecutionPlan) (actionable, protected, skipped int) {
	for _, item := range plan.Items {
		if item.Status == domain.StatusProtected {
			protected++
			continue
		}
		if item.Action == domain.ActionSkip || item.Status == domain.StatusSkipped {
			skipped++
			continue
		}
		if item.Action != domain.ActionAdvisory {
			actionable++
		}
	}
	return actionable, protected, skipped
}

func planNotTouchedSummary(protected, skipped int) string {
	parts := make([]string, 0, 2)
	if protected > 0 {
		parts = append(parts, fmt.Sprintf("%d protected", protected))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d excluded", skipped))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, "  •  ")
}

func planReviewSubject(command string) string {
	switch strings.TrimSpace(command) {
	case "uninstall":
		return "target"
	case "analyze":
		return "trace"
	case "optimize", "autofix":
		return "task"
	default:
		return "item"
	}
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
		if actionableDisplayItem(item) {
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
		fmt.Sprintf("Safe %s", domain.HumanBytes(planActionableRiskBytes(plan, domain.RiskSafe))),
		fmt.Sprintf("Review %s", domain.HumanBytes(planActionableRiskBytes(plan, domain.RiskReview))),
	}
	if high := planActionableRiskBytes(plan, domain.RiskHigh); high > 0 {
		parts = append(parts, fmt.Sprintf("High %s", domain.HumanBytes(high)))
	}
	return strings.Join(parts, "  •  ")
}

func planActionableRiskBytes(plan domain.ExecutionPlan, risk domain.Risk) int64 {
	var total int64
	for _, item := range plan.Items {
		if actionableDisplayItem(item) && item.Risk == risk {
			total += item.Bytes
		}
	}
	return total
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
		if !actionableDisplayItem(item) {
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
	lines = append(lines, mutedStyle.Render(fmt.Sprintf("%d %s  •  %s actionable", len(order), planGroupCollectionSummaryLabel(plan.Command), domain.HumanBytes(planDisplayBytes(plan)))))
	for _, key := range order[:min(len(order), limit)] {
		b := buckets[key]
		lines = append(lines, wrapText(safeStyle.Render(fmt.Sprintf("✓ %-20s  %d %s  •  %s", b.label, b.items, pl(b.items, "item", "items"), domain.HumanBytes(b.bytes))), width))
	}
	if len(order) > limit {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("+%d more %s", len(order)-limit, planGroupCollectionOverflowLabel(plan.Command))))
	}
	return lines
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
	case "analyze":
		return "m toggles current trace"
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
