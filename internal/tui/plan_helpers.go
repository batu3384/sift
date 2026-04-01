package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/batu3384/sift/internal/domain"
)

type reviewGroupSummary struct {
	label    string
	total    int
	included int
	bytes    int64
}

func (m *planModel) toggleCurrentGroup() {
	indexes := m.currentGroupIndexes()
	if len(indexes) == 0 {
		return
	}
	if m.excluded == nil {
		m.excluded = map[string]bool{}
	}
	allExcluded := true
	for _, idx := range indexes {
		item := m.plan.Items[idx]
		if !m.excluded[item.ID] {
			allExcluded = false
			break
		}
	}
	if allExcluded {
		for _, idx := range indexes {
			delete(m.excluded, m.plan.Items[idx].ID)
		}
		return
	}
	for _, idx := range indexes {
		m.excluded[m.plan.Items[idx].ID] = true
	}
}

func (m planModel) currentGroupSummary() reviewGroupSummary {
	indexes := m.currentGroupIndexes()
	if len(indexes) == 0 {
		return reviewGroupSummary{}
	}
	summary := reviewGroupSummary{label: reviewGroupLabel(m.plan.Items[indexes[0]])}
	for _, idx := range indexes {
		item := m.plan.Items[idx]
		summary.total++
		if !m.excluded[item.ID] {
			summary.included++
			summary.bytes += item.Bytes
		}
	}
	return summary
}

func (m planModel) currentGroupIndexes() []int {
	item, ok := m.selectedItem()
	if !ok {
		return nil
	}
	label := reviewGroupLabel(item)
	indexes := make([]int, 0)
	for idx, candidate := range m.plan.Items {
		if !canToggleReviewItem(candidate) {
			continue
		}
		if reviewGroupLabel(candidate) != label {
			continue
		}
		indexes = append(indexes, idx)
	}
	return indexes
}

func reviewGroupLabel(item domain.Finding) string {
	label := groupedItemLabel(item)
	if strings.TrimSpace(label) == "" {
		label = sectionTitle(domain.ExecutionPlan{}, item.Category)
	}
	return label
}

func (m planModel) effectivePlan() domain.ExecutionPlan {
	if len(m.excluded) == 0 {
		return m.plan
	}
	plan := m.plan
	plan.Items = make([]domain.Finding, len(m.plan.Items))
	copy(plan.Items, m.plan.Items)
	for idx := range plan.Items {
		if !m.excluded[plan.Items[idx].ID] {
			continue
		}
		plan.Items[idx].Action = domain.ActionSkip
		plan.Items[idx].Status = domain.StatusSkipped
		plan.Items[idx].Recovery.Message = "Excluded from this run."
	}
	plan.Totals = calculatePlanTotals(plan.Items)
	return plan
}

func canToggleReviewItem(item domain.Finding) bool {
	if item.Status == domain.StatusProtected {
		return false
	}
	return item.Action != domain.ActionAdvisory
}

func calculatePlanTotals(items []domain.Finding) domain.Totals {
	var totals domain.Totals
	for _, item := range items {
		totals.ItemCount++
		if item.Action == domain.ActionSkip {
			continue
		}
		totals.Bytes += item.Bytes
		switch item.Risk {
		case domain.RiskSafe:
			totals.SafeBytes += item.Bytes
		case domain.RiskReview:
			totals.ReviewBytes += item.Bytes
		case domain.RiskHigh:
			totals.HighBytes += item.Bytes
		}
	}
	return totals
}

func planModuleCount(plan domain.ExecutionPlan) int {
	seen := map[string]struct{}{}
	for _, item := range plan.Items {
		if item.Action == domain.ActionAdvisory {
			continue
		}
		key := progressGroupKey(item)
		if key == "" {
			continue
		}
		seen[key] = struct{}{}
	}
	return len(seen)
}

func sectionTitle(plan domain.ExecutionPlan, category domain.Category) string {
	if plan.Command == "analyze" {
		switch category {
		case domain.CategoryDiskUsage:
			return "LARGEST CHILDREN"
		case domain.CategoryLargeFiles:
			return "LARGE FILES"
		}
	}
	// Mole-style category headers: ➤ Category Name
	title := domain.CategoryTitle(category)
	return "➤ " + title
}

func groupedItemLabel(item domain.Finding) string {
	group := domain.ExecutionGroupLabel(item)
	if group == "" || group == domain.CategoryTitle(item.Category) {
		return ""
	}
	return group
}

func progressGroupKey(item domain.Finding) string {
	return domain.ExecutionGroupKey(item)
}

func analyzeSummaryLines(plan domain.ExecutionPlan) []string {
	var lines []string
	diskUsage := findingsByCategory(plan.Items, domain.CategoryDiskUsage)
	largeFiles := findingsByCategory(plan.Items, domain.CategoryLargeFiles)
	lines = append(lines, mutedStyle.Render(fmt.Sprintf("Summary  children %d  •  files %d  •  total %s", len(diskUsage), len(largeFiles), domain.HumanBytes(plan.Totals.Bytes))))
	topParts := make([]string, 0, 2)
	if len(diskUsage) > 0 {
		topParts = append(topParts, fmt.Sprintf("child %s (%s)", diskUsage[0].Name, domain.HumanBytes(diskUsage[0].Bytes)))
	}
	if len(largeFiles) > 0 {
		topParts = append(topParts, fmt.Sprintf("file %s (%s)", largeFiles[0].Name, domain.HumanBytes(largeFiles[0].Bytes)))
	}
	if len(topParts) > 0 {
		lines = append(lines, mutedStyle.Render("Top  "+strings.Join(topParts, "  •  ")))
	}
	return lines
}

func findingsByCategory(items []domain.Finding, category domain.Category) []domain.Finding {
	out := make([]domain.Finding, 0, len(items))
	for _, item := range items {
		if item.Category == category {
			out = append(out, item)
		}
	}
	return out
}

func trimAnalyzeSource(source string) string {
	source = strings.TrimPrefix(source, "Immediate child of ")
	source = strings.TrimSuffix(source, " • folded")
	return source
}

func analyzePreviewText(item domain.Finding, plan domain.ExecutionPlan) string {
	switch item.Category {
	case domain.CategoryDiskUsage:
		root := trimAnalyzeSource(item.Source)
		if root == "" || strings.EqualFold(root, item.DisplayPath) {
			root = currentAnalyzeTarget(plan)
		}
		if strings.Contains(item.Name, string(filepath.Separator)) {
			return "Folded chain. Drill in starts at the deepest mapped directory."
		}
		return "Drill in for child map and cleanup queue."
	case domain.CategoryLargeFiles:
		root := trimAnalyzeSource(item.Source)
		if root == "" {
			return "Large file candidate. Review before staging for cleanup."
		}
		return fmt.Sprintf("Large file discovered under %s. Stage it directly or return to parent context.", root)
	default:
		if item.Source != "" {
			return trimAnalyzeSource(item.Source)
		}
	}
	return ""
}

func analyzeDirectoryPreviewLines(preview analyzeDirectoryPreview, width int) []string {
	return renderAnalyzeDirectoryPreviewLines(preview, width)
}

func renderAnalyzeDirectoryPreviewLines(preview analyzeDirectoryPreview, width int) []string {
	if strings.TrimSpace(preview.Path) == "" {
		return nil
	}
	if preview.Unavailable {
		return []string{mutedStyle.Render("Children  unavailable")}
	}
	lines := []string{
		mutedStyle.Render(fmt.Sprintf("Children  %d total  •  %d %s  •  %d %s", preview.Total, preview.Dirs, pl(preview.Dirs, "dir", "dirs"), preview.Files, pl(preview.Files, "file", "files"))),
	}
	if len(preview.DirNames) > 0 {
		lines = append(lines, mutedStyle.Render(singleLine("Dirs  "+strings.Join(preview.DirNames, ", "), width)))
	}
	if len(preview.FileSamples) > 0 {
		limit := len(preview.FileSamples)
		if limit > 2 {
			limit = 2
		}
		labels := make([]string, 0, limit)
		for _, sample := range preview.FileSamples[:limit] {
			labels = append(labels, fmt.Sprintf("%s (%s)", sample.Name, domain.HumanBytes(sample.Size)))
		}
		lines = append(lines, mutedStyle.Render(singleLine("Files "+strings.Join(labels, ", "), width)))
	}
	if len(preview.Names) > 0 {
		lines = append(lines, mutedStyle.Render(singleLine("Next  "+strings.Join(preview.Names, ", "), width)))
	}
	return lines
}

func analyzeCurrentViewLines(item domain.Finding, model analyzeBrowserModel, width int) []string {
	visible := model.visibleIndices()
	relevant := make([]domain.Finding, 0, len(visible))
	for _, idx := range visible {
		if idx < 0 || idx >= len(model.plan.Items) {
			continue
		}
		candidate := model.plan.Items[idx]
		if candidate.Category == item.Category {
			relevant = append(relevant, candidate)
		}
	}
	if len(relevant) == 0 {
		return nil
	}
	index := -1
	var totalBytes int64
	for i, candidate := range relevant {
		totalBytes += candidate.Bytes
		if candidate.Path == item.Path {
			index = i
		}
	}
	if index < 0 {
		return nil
	}
	share := "0%"
	if totalBytes > 0 {
		pct := float64(item.Bytes) / float64(totalBytes) * 100
		switch {
		case pct >= 10:
			share = fmt.Sprintf("%.0f%%", pct)
		case pct >= 1:
			share = fmt.Sprintf("%.1f%%", pct)
		default:
			share = "<1%"
		}
	}
	lines := []string{
		mutedStyle.Render(fmt.Sprintf("Focus  rank %d/%d  •  share %s", index+1, len(relevant), share)),
	}
	peers := make([]string, 0, 3)
	for _, candidate := range relevant {
		if candidate.Path == item.Path {
			continue
		}
		label := candidate.Name
		if strings.TrimSpace(label) == "" {
			label = filepath.Base(candidate.DisplayPath)
		}
		peers = append(peers, fmt.Sprintf("%s (%s)", label, domain.HumanBytes(candidate.Bytes)))
		if len(peers) == 3 {
			break
		}
	}
	if len(peers) > 0 {
		lines = append(lines, mutedStyle.Render(singleLine("Peers  "+strings.Join(peers, "  •  "), width)))
	}
	return lines
}

func formatFloatSeries(values []float64, limit int) string {
	if len(values) == 0 {
		return ""
	}
	if limit <= 0 || limit > len(values) {
		limit = len(values)
	}
	parts := make([]string, 0, limit+1)
	for _, value := range values[:limit] {
		parts = append(parts, fmt.Sprintf("%.0f%%", value))
	}
	if len(values) > limit {
		parts = append(parts, fmt.Sprintf("+%d more", len(values)-limit))
	}
	return strings.Join(parts, " ")
}

func currentAnalyzeTarget(plan domain.ExecutionPlan) string {
	if len(plan.Targets) == 0 {
		return "Current target"
	}
	if len(plan.Targets) == 1 {
		return "Target " + plan.Targets[0]
	}
	return fmt.Sprintf("Targets %s", strings.Join(plan.Targets, ", "))
}

func uninstallTargetSummaryLines(plan domain.ExecutionPlan, width int) []string {
	if len(plan.Targets) == 0 {
		return []string{mutedStyle.Render("No uninstall targets.")}
	}
	native := 0
	remnants := 0
	protected := 0
	for _, item := range plan.Items {
		switch item.Action {
		case domain.ActionNative:
			native++
		case domain.ActionTrash:
			remnants++
		}
		if item.Status == domain.StatusProtected {
			protected++
		}
	}
	apps := len(plan.Targets)
	lines := []string{
		mutedStyle.Render(fmt.Sprintf("%d %s  •  %d native %s  •  %d %s", apps, pl(apps, "app", "apps"), native, pl(native, "step", "steps"), remnants, pl(remnants, "remnant", "remnants"))),
		mutedStyle.Render(fmt.Sprintf("%d protected %s  •  %s total", protected, pl(protected, "item", "items"), domain.HumanBytes(plan.Totals.Bytes))),
	}
	displayTargets := plan.Targets[:min(len(plan.Targets), 3)]
	for _, target := range displayTargets {
		lines = append(lines, wrapText(mutedStyle.Render("• "+target), width))
	}
	extra := len(plan.Targets) - len(displayTargets)
	if extra > 0 {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("+%d more %s", extra, pl(extra, "target", "targets"))))
	}
	return lines
}

func analyzeLine(item domain.Finding) string {
	name := strings.TrimSpace(item.Name)
	if name == "" {
		name = strings.TrimSpace(item.DisplayPath)
	}
	if name == "" {
		name = strings.TrimSpace(item.Path)
	}
	parts := []string{name, domain.HumanBytes(item.Bytes)}
	if age := analyzeAge(item.LastModified); age != "" {
		parts = append(parts, age)
	}
	if risk := strings.TrimSpace(string(item.Risk)); risk != "" {
		parts = append(parts, styleForRisk(item.Risk).Render(risk))
	}
	return strings.Join(parts, " • ")
}
