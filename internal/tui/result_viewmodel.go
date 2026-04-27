package tui

import (
	"fmt"
	"strings"

	"github.com/batu3384/sift/internal/domain"
)

func resultSummaryLine(model resultModel) string {
	completed, deleted, _, _, _ := countResultStatuses(model.result)
	total := max(len(model.plan.Items), len(model.result.Items))
	changed := resultChangedCount(model.plan, completed, deleted)
	percent := 0
	if total > 0 {
		percent = changed * 100 / total
	}
	return fmt.Sprintf("Result   %d%% changed  •  %d/%d changed  •  %s freed", percent, changed, total, domain.HumanBytes(resultFreedBytes(model.plan, model.result)))
}

func resultNextLine(model resultModel) string {
	return "Next    " + strings.TrimPrefix(resultNextRail(model), "Next rail  •  ")
}

func resultStatusLine(model resultModel) string {
	issues := len(resultRecoveryCandidates(model.plan, model.result, model.filter))
	parts := make([]string, 0, 4)
	if issues > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", issues, pl(issues, "issue", "issues")))
	}
	if warnCount := len(model.result.Warnings); warnCount > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", warnCount, pl(warnCount, "warning", "warnings")))
	}
	if cmdCount := len(model.result.FollowUpCommands); cmdCount > 0 {
		parts = append(parts, fmt.Sprintf("%d follow-up %s", cmdCount, pl(cmdCount, "command", "commands")))
	}
	if len(parts) == 0 {
		parts = append(parts, "lane settled cleanly")
	} else {
		parts = append(parts, "lane needs review")
	}
	return "Status   " + strings.Join(parts, "  •  ")
}

func resultNotTouchedLine(result domain.ExecutionResult) string {
	_, _, _, skipped, protected := countResultStatuses(result)
	permissionLimited := permissionLimitedResultCount(result)
	parts := make([]string, 0, 3)
	if protected > 0 {
		parts = append(parts, fmt.Sprintf("%d protected", protected))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}
	if permissionLimited > 0 {
		parts = append(parts, fmt.Sprintf("%d permission-limited", permissionLimited))
	}
	if len(parts) == 0 {
		return "Not touched  none"
	}
	return "Not touched  " + strings.Join(parts, "  •  ")
}

func permissionLimitedResultCount(result domain.ExecutionResult) int {
	count := 0
	for _, item := range result.Items {
		if item.Status != domain.StatusFailed {
			continue
		}
		text := strings.ToLower(strings.TrimSpace(string(item.Reason) + " " + item.Message))
		if strings.Contains(text, "permission") || strings.Contains(text, "denied") {
			count++
		}
	}
	return count
}

func resultTrackLine(model resultModel) string {
	completed, deleted, failed, _, protected := countResultStatuses(model.result)
	switch model.plan.Command {
	case "clean":
		sections := resultSettledSections(model.plan, model.result)
		return fmt.Sprintf("Rail    %d %s  •  %d reclaimed  •  %d open", sections, pl(sections, "section", "sections"), deleted, failed+protected)
	case "uninstall":
		native, removed, aftercare := uninstallResultBuckets(model)
		parts := []string{"Rail", fmt.Sprintf("%d native", native), fmt.Sprintf("%d removed", removed)}
		if aftercare > 0 {
			parts = append(parts, fmt.Sprintf("%d aftercare", aftercare))
		}
		if failed+protected > 0 {
			parts = append(parts, fmt.Sprintf("%d open", failed+protected))
		}
		return strings.Join(parts, "  •  ")
	case "optimize":
		return fmt.Sprintf("Rail    %s  •  %d applied  •  %d open", resultPhaseTrackLabel(model.plan, model.result), completed, failed+protected)
	case "autofix":
		return fmt.Sprintf("Rail    %s  •  %d applied  •  %d open", resultPhaseTrackLabel(model.plan, model.result), completed, failed+protected)
	default:
		items := len(model.result.Items)
		return fmt.Sprintf("Rail    %d %s  •  %d open", items, pl(items, "item", "items"), failed+protected)
	}
}

func resultTrackSubtitle(model resultModel) string {
	switch model.plan.Command {
	case "clean":
		n := resultSettledSections(model.plan, model.result)
		return fmt.Sprintf("%d %s", n, pl(n, "section", "sections"))
	case "uninstall":
		native, removed, aftercare := uninstallResultBuckets(model)
		parts := []string{}
		if native > 0 {
			parts = append(parts, fmt.Sprintf("%d native", native))
		}
		if removed > 0 {
			parts = append(parts, fmt.Sprintf("%d removed", removed))
		}
		if aftercare > 0 {
			parts = append(parts, fmt.Sprintf("%d aftercare", aftercare))
		}
		if len(parts) == 0 {
			parts = append(parts, "targets")
		}
		return strings.Join(parts, " • ")
	case "optimize", "autofix":
		return resultPhaseTrackLabel(model.plan, model.result)
	default:
		return "items"
	}
}

func resultNextRail(model resultModel) string {
	candidates := resultRecoveryCandidates(model.plan, model.result, model.filter)
	failed := resultRecoveryCandidatesForStatuses(model.plan, model.result, model.filter, domain.StatusFailed)
	group := resultCurrentGroupRecoveryCandidates(model)
	parts := []string{"Next rail"}
	switch {
	case len(failed) > 0:
		parts = append(parts, resultNextRetryAction(model.plan), resultNextRecoveryAction(model.plan))
	case len(group) > 0:
		parts = append(parts, resultNextCurrentAction(model.plan), resultNextGroupRecoveryAction(model.plan))
	case len(candidates) > 0:
		parts = append(parts, resultNextRecoveryAction(model.plan), resultFollowUpHint(model.plan))
	case len(model.result.FollowUpCommands) > 0:
		parts = append(parts, "review suggested commands", "rerun when ready")
	default:
		parts = append(parts, "settled cleanly", "open another lane when ready")
	}
	return strings.Join(parts, "  •  ")
}

func resultScopeLine(model resultModel) string {
	scope := strings.TrimSpace(planReviewScopeLine(model.plan))
	if scope == "" {
		n := len(model.result.Items)
		scope = fmt.Sprintf("%s  •  %d %s  •  %s reclaimable", titleCase(model.plan.Command), n, pl(n, "item", "items"), domain.HumanBytes(planDisplayBytes(model.plan)))
	}
	return "Scope   " + scope
}

func resultListSubtitle(model resultModel) string {
	visible := model.visibleIndices()
	scopeSingular, scopeLabel := resultScopeLabelPair(model.plan)
	total := len(model.result.Items)
	scopeCount := resultScopeCount(model.plan)
	return fmt.Sprintf("%s • %d/%d %s • %d %s", strings.ToUpper(string(coalesceResultFilter(model.filter))), len(visible), total, pl(total, "item", "items"), scopeCount, pl(scopeCount, scopeSingular, scopeLabel))
}

func resultFilterMatch(filter resultFilter, item domain.OperationResult) bool {
	switch filter {
	case resultFilterIssues:
		return item.Status == domain.StatusFailed || item.Status == domain.StatusProtected
	case resultFilterClean:
		return item.Status == domain.StatusDeleted || item.Status == domain.StatusCompleted || item.Status == domain.StatusSkipped
	default:
		return true
	}
}

func coalesceResultFilter(filter resultFilter) resultFilter {
	if filter == "" {
		return resultFilterAll
	}
	return filter
}

func hasRecoveryCandidates(result domain.ExecutionResult) bool {
	for _, item := range result.Items {
		if item.Status == domain.StatusFailed || item.Status == domain.StatusProtected {
			return true
		}
	}
	return false
}

func resultRecoveryCandidates(plan domain.ExecutionPlan, result domain.ExecutionResult, filter resultFilter) []domain.Finding {
	return resultRecoveryCandidatesForStatuses(plan, result, filter, domain.StatusFailed, domain.StatusProtected)
}

func resultRecoveryCandidatesForStatuses(plan domain.ExecutionPlan, result domain.ExecutionResult, filter resultFilter, statuses ...domain.FindingStatus) []domain.Finding {
	if len(result.Items) == 0 {
		return nil
	}
	filter = coalesceResultFilter(filter)
	allowed := make(map[domain.FindingStatus]struct{}, len(statuses))
	for _, status := range statuses {
		allowed[status] = struct{}{}
	}
	visibleIssuePaths := map[string]struct{}{}
	if filter == resultFilterIssues {
		for _, item := range result.Items {
			if resultFilterMatch(filter, item) {
				path := strings.TrimSpace(item.Path)
				if path != "" {
					visibleIssuePaths[path] = struct{}{}
				}
			}
		}
	}
	byID := make(map[string]domain.Finding, len(plan.Items))
	byPath := make(map[string]domain.Finding, len(plan.Items))
	for _, item := range plan.Items {
		if strings.TrimSpace(item.ID) != "" {
			byID[item.ID] = item
		}
		if path := strings.TrimSpace(item.Path); path != "" {
			byPath[path] = item
		}
	}
	candidates := make([]domain.Finding, 0)
	seen := map[string]struct{}{}
	for _, item := range result.Items {
		if _, ok := allowed[item.Status]; !ok {
			continue
		}
		path := strings.TrimSpace(item.Path)
		if len(visibleIssuePaths) > 0 {
			if _, ok := visibleIssuePaths[path]; !ok {
				if resolved, ok := byID[item.FindingID]; ok {
					if _, ok := visibleIssuePaths[strings.TrimSpace(resolved.Path)]; !ok {
						continue
					}
				} else {
					continue
				}
			}
		}
		var candidate domain.Finding
		if resolved, ok := byID[item.FindingID]; ok {
			candidate = resolved
		} else if resolved, ok := byPath[path]; ok {
			candidate = resolved
		} else if path != "" {
			candidate = domain.Finding{
				Path:        path,
				DisplayPath: path,
				Status:      item.Status,
				Action:      domain.ActionSkip,
				Category:    domain.CategorySystemClutter,
			}
		} else {
			continue
		}
		key := strings.TrimSpace(candidate.Path)
		if key == "" {
			key = strings.TrimSpace(item.Path)
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func resultRecoveryBatchLines(candidates []domain.Finding, width int) []string {
	if len(candidates) == 0 {
		return []string{mutedStyle.Render("No recovery items.")}
	}
	type bucket struct {
		label string
		count int
		bytes int64
	}
	order := make([]string, 0)
	buckets := map[string]*bucket{}
	for _, item := range candidates {
		label := groupedItemLabel(item)
		if label == "" {
			label = sectionTitle(domain.ExecutionPlan{}, item.Category)
		}
		if _, ok := buckets[label]; !ok {
			order = append(order, label)
			buckets[label] = &bucket{label: label}
		}
		buckets[label].count++
		buckets[label].bytes += item.Bytes
	}
	lines := make([]string, 0, len(order)+1)
	lines = append(lines, mutedStyle.Render(fmt.Sprintf("%d %s • %d %s", len(candidates), pl(len(candidates), "issue", "issues"), len(order), pl(len(order), "module", "modules"))))
	for _, key := range order {
		bucket := buckets[key]
		parts := []string{bucket.label, fmt.Sprintf("%d %s", bucket.count, pl(bucket.count, "item", "items"))}
		if bucket.bytes > 0 {
			parts = append(parts, domain.HumanBytes(bucket.bytes))
		}
		lines = append(lines, wrapText(mutedStyle.Render(strings.Join(parts, " • ")), width))
	}
	return lines
}

func resultRecoveryDetailLines(plan domain.ExecutionPlan, candidates []domain.Finding, width int) []string {
	lines := resultRecoveryBatchLines(candidates, width)
	if len(lines) == 0 || len(candidates) == 0 {
		return lines
	}
	lines[0] = mutedStyle.Render(resultRecoveryScopeSummary(plan, candidates))
	return lines
}

func resultRecoveryTitle(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "uninstall":
		return "Remnant Review"
	case "optimize", "autofix":
		return "Fix Review"
	default:
		return "Recovery"
	}
}

func resultCurrentTitle(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "uninstall":
		return "Current Target"
	case "optimize", "autofix":
		return "Current Phase"
	default:
		return "Current"
	}
}

func resultRetryTitle(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "uninstall":
		return "Retry Handoff"
	case "optimize", "autofix":
		return "Retry Fixes"
	default:
		return "Retry"
	}
}

func resultRecoveryActionLine(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "uninstall":
		return "x opens remnant review"
	case "optimize", "autofix":
		return "x opens fix review"
	default:
		return "x opens recovery review"
	}
}

func resultCurrentActionLine(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "uninstall":
		return "m opens current target"
	case "optimize", "autofix":
		return "m opens current phase"
	default:
		return "m opens current module"
	}
}

func resultRetryActionLine(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "uninstall":
		return "r retries failed handoff"
	case "optimize", "autofix":
		return "r retries failed fixes"
	default:
		return "r retries failed items"
	}
}

func resultFollowUpHint(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "uninstall":
		return "review blocked targets"
	case "optimize", "autofix":
		return "review blocked fixes"
	default:
		return "review blocked paths"
	}
}

func resultNextRetryAction(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "uninstall", "optimize", "autofix":
		return resultRetryActionLine(plan)
	default:
		return "r retries failed"
	}
}

func resultNextRecoveryAction(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "uninstall", "optimize", "autofix":
		return resultRecoveryActionLine(plan)
	default:
		return "x reopens recovery batch"
	}
}

func resultNextCurrentAction(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "uninstall", "optimize", "autofix":
		return resultCurrentActionLine(plan)
	default:
		return "m reopens current module"
	}
}

func resultNextGroupRecoveryAction(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "uninstall", "optimize", "autofix":
		return resultRecoveryActionLine(plan)
	default:
		return "x reopens all issues"
	}
}

func resultCurrentGroupRecoveryCandidates(model resultModel) []domain.Finding {
	visible := model.visibleIndices()
	if len(visible) == 0 || model.cursor < 0 || model.cursor >= len(visible) {
		return nil
	}
	item := model.result.Items[visible[model.cursor]]
	group := resultGroupLabelForItem(model.plan, item)
	if group == "" {
		return nil
	}
	candidates := resultRecoveryCandidates(model.plan, model.result, model.filter)
	return filterRecoveryCandidatesByGroup(candidates, group)
}

func filterRecoveryCandidatesByGroup(candidates []domain.Finding, group string) []domain.Finding {
	if group == "" || len(candidates) == 0 {
		return nil
	}
	filtered := make([]domain.Finding, 0, len(candidates))
	for _, candidate := range candidates {
		if groupedItemLabel(candidate) == group {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func resultGroupLabelForItem(plan domain.ExecutionPlan, item domain.OperationResult) string {
	if resolved, ok := resultPlanItem(plan, item); ok {
		return groupedItemLabel(resolved)
	}
	return ""
}

func resultPlanItem(plan domain.ExecutionPlan, item domain.OperationResult) (domain.Finding, bool) {
	for _, candidate := range plan.Items {
		if item.FindingID != "" && candidate.ID == item.FindingID {
			return candidate, true
		}
		if strings.TrimSpace(candidate.Path) != "" && strings.TrimSpace(candidate.Path) == strings.TrimSpace(item.Path) {
			return candidate, true
		}
	}
	return domain.Finding{}, false
}

func countResultStatuses(result domain.ExecutionResult) (completed, deleted, failed, skipped, protected int) {
	for _, item := range result.Items {
		switch item.Status {
		case domain.StatusCompleted:
			completed++
		case domain.StatusDeleted:
			deleted++
		case domain.StatusFailed:
			failed++
		case domain.StatusSkipped:
			skipped++
		case domain.StatusProtected:
			protected++
		}
	}
	return
}

func resultRecoveryScopeSummary(plan domain.ExecutionPlan, candidates []domain.Finding) string {
	if len(candidates) == 0 {
		return "0 issues"
	}
	seen := map[string]struct{}{}
	for _, item := range candidates {
		key := progressGroupKey(item)
		if key == "" {
			key = strings.TrimSpace(item.Path)
		}
		if key == "" {
			continue
		}
		seen[key] = struct{}{}
	}
	scopeSingular, scopePlural := "module", "modules"
	if plan.Command == "uninstall" {
		scopeSingular, scopePlural = "target", "targets"
	}
	return fmt.Sprintf("%d %s across %d %s", len(candidates), pl(len(candidates), "issue", "issues"), len(seen), pl(len(seen), scopeSingular, scopePlural))
}

func resultWhatChangedLine(result domain.ExecutionResult) string {
	completed, deleted, failed, skipped, protected := countResultStatuses(result)
	parts := []string{"What changed"}
	if deleted > 0 {
		parts = append(parts, fmt.Sprintf("%d deleted", deleted))
	}
	if completed > 0 {
		parts = append(parts, fmt.Sprintf("%d completed", completed))
	}
	if protected > 0 {
		parts = append(parts, fmt.Sprintf("%d protected", protected))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}
	if len(parts) == 1 {
		parts = append(parts, "no changes recorded")
	}
	return strings.Join(parts, "  •  ")
}

func resultCompletedCardLabel(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "uninstall":
		return "native"
	case "optimize", "autofix":
		return "applied"
	default:
		return "done"
	}
}

func resultChangedCardLabel(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "clean":
		return "reclaim"
	case "uninstall":
		return "removed"
	case "optimize":
		return "changes"
	case "autofix":
		return "fixed"
	default:
		return "changed"
	}
}

func resultScopeLabelPair(plan domain.ExecutionPlan) (singular, plural string) {
	switch plan.Command {
	case "uninstall":
		return "app", "apps"
	case "optimize", "autofix":
		return "task", "tasks"
	default:
		return "module", "modules"
	}
}

func resultScopeCount(plan domain.ExecutionPlan) int {
	if plan.Command == "uninstall" {
		if targets := uninstallTargetCount(plan); targets > 0 {
			return targets
		}
	}
	return planModuleCount(plan)
}

func resultChangedCount(_ domain.ExecutionPlan, completed, deleted int) int {
	return completed + deleted
}

func resultFreedBytes(plan domain.ExecutionPlan, result domain.ExecutionResult) int64 {
	var freed int64
	for _, item := range result.Items {
		if item.Status != domain.StatusDeleted {
			continue
		}
		planItem, ok := resultPlanItem(plan, item)
		if ok {
			freed += planItem.Bytes
		}
	}
	return freed
}

func uninstallResultBuckets(model resultModel) (native, removed, aftercare int) {
	for _, item := range model.result.Items {
		planItem, ok := resultPlanItem(model.plan, item)
		switch {
		case ok && planItem.Action == domain.ActionNative && item.Status == domain.StatusCompleted:
			native++
		case item.Status == domain.StatusDeleted:
			removed++
		case ok && planItem.Action == domain.ActionCommand && strings.TrimSpace(planItem.TaskPhase) != "" && item.Status == domain.StatusCompleted:
			aftercare++
		}
	}
	return native, removed, aftercare
}

func resultSettledSections(plan domain.ExecutionPlan, result domain.ExecutionResult) int {
	seen := map[string]struct{}{}
	for _, item := range result.Items {
		if item.Status != domain.StatusCompleted && item.Status != domain.StatusDeleted {
			continue
		}
		planItem, ok := resultPlanItem(plan, item)
		if !ok {
			continue
		}
		key := progressGroupKey(planItem)
		if key == "" {
			key = strings.TrimSpace(planItem.Path)
		}
		if key == "" {
			continue
		}
		seen[key] = struct{}{}
	}
	return len(seen)
}

func resultPhaseTrackLabel(plan domain.ExecutionPlan, result domain.ExecutionResult) string {
	counts := map[string]int{}
	order := make([]string, 0)
	for _, item := range result.Items {
		if item.Status != domain.StatusCompleted && item.Status != domain.StatusDeleted {
			continue
		}
		planItem, ok := resultPlanItem(plan, item)
		if !ok {
			continue
		}
		phase := strings.TrimSpace(planItem.TaskPhase)
		if phase == "" {
			continue
		}
		if _, ok := counts[phase]; !ok {
			order = append(order, phase)
		}
		counts[phase]++
	}
	if len(order) == 0 {
		if plan.Command == "autofix" {
			n := actionableCount(plan)
			return fmt.Sprintf("%d %s", n, pl(n, "fix", "fixes"))
		}
		n := max(maintenancePhaseCount(plan), 1)
		return fmt.Sprintf("%d %s", n, pl(n, "phase", "phases"))
	}
	parts := make([]string, 0, len(order))
	for _, phase := range order {
		parts = append(parts, fmt.Sprintf("%s %d", phase, counts[phase]))
	}
	return strings.Join(parts, " • ")
}
