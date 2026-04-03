package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/batu3384/sift/internal/domain"
)

func analyzeActionRail(model analyzeBrowserModel) string {
	parts := []string{"Next"}
	switch {
	case model.loading:
		parts = append(parts, "wait for scan", "esc back")
	case model.errMsg != "":
		parts = append(parts, "esc back", analyzeReviewAction(model))
	default:
		item, ok := model.selectedActiveItem()
		if len(model.stageOrder) > 0 || (ok && canStage(item)) {
			parts = append(parts, analyzeReviewAction(model))
		}
		if model.activePane() == analyzePaneQueue {
			if ok {
				parts = append(parts, "u remove")
			}
			break
		}
		if ok && canDescendInto(item) {
			parts = append(parts, "enter open")
		}
		if ok && canStage(item) {
			if _, staged := model.staged[item.Path]; staged {
				parts = append(parts, "u unstage")
			} else {
				parts = append(parts, "space add")
			}
		}
		if len(parts) == 1 {
			parts = append(parts, "browse items", "change filter")
		}
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return parts[0] + "  " + strings.Join(parts[1:], "  •  ")
}

func analyzeAtmosphereLine(model analyzeBrowserModel) string {
	switch {
	case model.loading:
		return "State   scanning"
	case model.errMsg != "":
		return "State   navigation error  •  see detail panel"
	default:
		filter := model.filter
		if filter == "" {
			filter = analyzeFilterAll
		}
		parts := []string{
			"State   " + strings.ToUpper(string(filter)),
			fmt.Sprintf("%d visible", len(model.visibleIndices())),
		}
		if len(model.stageOrder) > 0 {
			parts = append(parts, fmt.Sprintf("%d staged", len(model.stageOrder)))
		}
		if analyzePreviewReady(model) {
			parts = append(parts, "preview")
		}
		return strings.Join(parts, "  •  ")
	}
}

func analyzeTrailBlock(history []analyzeHistoryEntry, plan domain.ExecutionPlan, width int) string {
	lines := []string{mutedStyle.Render(analyzeTrailLine(history, plan, width))}
	if len(history) > 0 {
		lines = append(lines, mutedStyle.Render("Back  esc  •  History "+analyzeHistoryCountLabel(history)))
	}
	return strings.Join(lines, "\n")
}

func analyzeFilterMatch(filter analyzeFilter, item domain.Finding, staged map[string]domain.Finding) bool {
	switch filter {
	case analyzeFilterQueued:
		_, ok := staged[item.Path]
		return ok
	case analyzeFilterHigh:
		return item.Risk == domain.RiskHigh || item.Risk == domain.RiskReview
	default:
		return true
	}
}

func analyzeFilterLabel(model analyzeBrowserModel) string {
	filter := model.filter
	if filter == "" {
		filter = analyzeFilterAll
	}
	label := fmt.Sprintf("Filter %s", strings.ToUpper(string(filter)))
	if query := strings.TrimSpace(model.search.Value()); query != "" {
		label += fmt.Sprintf("  •  Search %q", query)
	}
	return fmt.Sprintf("%s  •  %d visible  •  %d queued", label, len(model.visibleIndices()), len(model.stageOrder))
}

func analyzeBrowseRail(model analyzeBrowserModel) string {
	return "Filter  " + analyzeFilterLabel(model)
}

func analyzeQueueRail(model analyzeBrowserModel, order []string) string {
	if len(order) == 0 {
		return "Review  nothing staged"
	}
	var bytes int64
	for _, path := range order {
		if item, ok := model.staged[path]; ok {
			bytes += item.Bytes
		}
	}
	count := len(order)
	modules, _ := analyzeStageBuckets(model.staged, order)
	label := "Queue  "
	if count == 1 {
		label = "Review "
	}
	return fmt.Sprintf("%s %d %s  •  %s  •  %d %s  •  %s", label, count, pl(count, "item", "items"), domain.HumanBytes(bytes), len(modules), pl(len(modules), "module", "modules"), analyzeQueueSortText(model.queueSort))
}

func analyzePreviewReady(model analyzeBrowserModel) bool {
	item, ok := model.selectedActiveItem()
	if !ok {
		return false
	}
	if strings.TrimSpace(analyzePreviewText(item, model.plan)) != "" {
		return true
	}
	return len(analyzeDirectoryPreviewLines(model.selectedPreview(), 120)) > 0
}

func analyzeSelectionLine(model analyzeBrowserModel) string {
	focus := strings.ToUpper(string(model.activePane()))
	if focus == "" {
		focus = "BROWSE"
	}
	total := len(model.visibleIndices())
	index := model.cursor + 1
	item, ok := model.selectedItem()
	if model.activePane() == analyzePaneQueue {
		total = len(model.sortedStageOrder())
		index = model.queueCursor + 1
		item, ok = model.selectedQueuedItem()
	}
	if !ok || total == 0 {
		return "Focus   none  •  pick a finding to inspect"
	}
	label := strings.TrimSpace(item.Name)
	if label == "" {
		label = filepath.Base(strings.TrimSpace(item.Path))
	}
	if label == "" {
		label = item.DisplayPath
	}
	parts := []string{fmt.Sprintf("Focus   %s %d/%d", focus, index, total), label}
	if item.Bytes > 0 {
		parts = append(parts, domain.HumanBytes(item.Bytes))
	}
	if _, staged := model.staged[item.Path]; staged {
		if len(model.stageOrder) > 1 {
			parts = append(parts, "queued")
		} else {
			parts = append(parts, "ready")
		}
	}
	return strings.Join(parts, "  •  ")
}

func analyzeStageSummary(staged map[string]domain.Finding, order []string, queueSort analyzeQueueSort) string {
	if len(order) == 0 {
		return "Cleanup queue empty"
	}
	var bytes int64
	count := 0
	categories := map[domain.Category]int{}
	modules, _ := analyzeStageBuckets(staged, order)
	for _, path := range order {
		if item, ok := staged[path]; ok {
			count++
			bytes += item.Bytes
			categories[item.Category]++
		}
	}
	return fmt.Sprintf("Queued %d %s  •  %s  •  %d %s  •  sort %s  •  %s", count, pl(count, "item", "items"), domain.HumanBytes(bytes), len(modules), pl(len(modules), "module", "modules"), strings.ToUpper(string(coalesceAnalyzeQueueSort(queueSort))), summarizeAnalyzeCategories(categories))
}

func analyzeQueueView(model analyzeBrowserModel, order []string, width int, queueSort analyzeQueueSort) string {
	lines := make([]string, 0, len(order))
	categoryCounts := map[domain.Category]int{}
	modules, buckets := analyzeStageBuckets(model.staged, order)
	for _, path := range order {
		item, ok := model.staged[path]
		if !ok {
			continue
		}
		categoryCounts[item.Category]++
		label := item.Name
		if label == "" {
			label = item.DisplayPath
		}
		line := wrapText(fmt.Sprintf("%-10s %-7s %-16s %s", domain.HumanBytes(item.Bytes), analyzeAge(item.LastModified), queueCategoryLabel(item.Category), label), width)
		if model.activePane() == analyzePaneQueue && queueIndexForPath(order, path) == model.queueCursor {
			line = selectedLine.Render(line)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return mutedStyle.Render("No staged items.")
	}
	header := []string{
		mutedStyle.Render("Queue  " + analyzeStageSummary(model.staged, order, queueSort)),
		mutedStyle.Render("Focus  staged sweep  •  " + summarizeAnalyzeCategories(categoryCounts)),
		mutedStyle.Render("Sort  " + analyzeQueueSortText(queueSort)),
	}
	header = append(header, analyzeReviewPreviewLines(model, width)...)
	if len(modules) > 0 {
		header = append(header, mutedStyle.Render("Modules"))
		for _, key := range modules[:min(len(modules), 3)] {
			bucket := buckets[key]
			header = append(header, mutedStyle.Render(fmt.Sprintf("• %s  %d %s  %s", bucket.label, bucket.count, pl(bucket.count, "item", "items"), domain.HumanBytes(bucket.bytes))))
		}
	}
	header = append(header, "")
	lines = append(header, lines...)
	return strings.Join(lines, "\n")
}

type analyzeStageBucket struct {
	label string
	count int
	bytes int64
}

func analyzeStageBuckets(staged map[string]domain.Finding, order []string) ([]string, map[string]*analyzeStageBucket) {
	keys := make([]string, 0)
	buckets := map[string]*analyzeStageBucket{}
	for _, path := range order {
		item, ok := staged[path]
		if !ok {
			continue
		}
		label := domain.ExecutionGroupLabel(item)
		if strings.TrimSpace(label) == "" {
			label = sectionTitle(domain.ExecutionPlan{}, item.Category)
		}
		if _, ok := buckets[label]; !ok {
			keys = append(keys, label)
			buckets[label] = &analyzeStageBucket{label: label}
		}
		buckets[label].count++
		buckets[label].bytes += item.Bytes
	}
	sort.SliceStable(keys, func(i, j int) bool {
		left := buckets[keys[i]]
		right := buckets[keys[j]]
		if left.bytes == right.bytes {
			return left.label < right.label
		}
		return left.bytes > right.bytes
	})
	return keys, buckets
}

func analyzeBatchHighlightLines(staged map[string]domain.Finding, order []string, width int) []string {
	modules, buckets := analyzeStageBuckets(staged, order)
	if len(modules) == 0 {
		return []string{mutedStyle.Render("Stage one or more items to prepare a cleanup batch.")}
	}
	lines := make([]string, 0, min(len(modules)+1, 4))
	lead := buckets[modules[0]]
	lines = append(lines, mutedStyle.Render(fmt.Sprintf("Top module  %s  •  %d %s  •  %s", lead.label, lead.count, pl(lead.count, "item", "items"), domain.HumanBytes(lead.bytes))))
	for _, key := range modules[:min(len(modules), 3)] {
		bucket := buckets[key]
		lines = append(lines, wrapText(mutedStyle.Render(fmt.Sprintf("• %s  %d %s  %s", bucket.label, bucket.count, pl(bucket.count, "item", "items"), domain.HumanBytes(bucket.bytes))), width))
	}
	return lines
}

func analyzeImpactLine(item domain.Finding, plan domain.ExecutionPlan) string {
	if item.Bytes <= 0 || plan.Totals.Bytes <= 0 {
		return ""
	}
	share := float64(item.Bytes) / float64(plan.Totals.Bytes) * 100
	if share < 0.1 {
		return "<0.1% of current reclaim"
	}
	return fmt.Sprintf("%.0f%% of current reclaim", share)
}

func analyzeReviewPreviewLines(model analyzeBrowserModel, width int) []string {
	paths := model.reviewPreviewPaths()
	if len(paths) == 0 {
		return nil
	}
	label := "Selected"
	if len(paths) > 1 {
		label = "Batch"
	}
	switch {
	case model.reviewPreview.loading:
		return []string{mutedStyle.Render(label + "   loading plan")}
	case model.reviewPreview.err != "":
		return []string{highStyle.Render(label + "   preview unavailable")}
	case model.reviewPreview.loaded:
		lines := []string{
			mutedStyle.Render(func() string {
				mods := planModuleCount(model.reviewPreview.plan)
				return fmt.Sprintf("%-8s %d ready  •  %d %s  •  %s", label, actionableCount(model.reviewPreview.plan), mods, pl(mods, "module", "modules"), domain.HumanBytes(planDisplayBytes(model.reviewPreview.plan)))
			}()),
		}
		if len(paths) > 1 {
			if item, ok := model.selectedQueuedItem(); ok && model.activePane() == analyzePaneQueue {
				lines = append(lines, mutedStyle.Render("Focus   "+analyzeQueueFocusLabel(item)))
			}
		} else if item, ok := model.selectedItem(); ok {
			lines = append(lines, mutedStyle.Render("Focus   "+analyzeQueueFocusLabel(item)))
		}
		if len(model.reviewPreview.plan.Warnings) > 0 {
			lines = append(lines, mutedStyle.Render("Note    "+truncateText(model.reviewPreview.plan.Warnings[0], width)))
		}
		return lines
	default:
		return nil
	}
}

func analyzeReviewAction(model analyzeBrowserModel) string {
	if len(model.reviewPreviewPaths()) > 1 || model.hasQueuedBatch() {
		return "x review batch"
	}
	return "x review selected"
}

func summarizeAnalyzeCategories(counts map[domain.Category]int) string {
	if len(counts) == 0 {
		return "no categories"
	}
	order := []domain.Category{
		domain.CategoryDiskUsage,
		domain.CategoryLargeFiles,
		domain.CategoryTempFiles,
		domain.CategoryBrowserData,
		domain.CategoryDeveloperCaches,
		domain.CategoryPackageCaches,
	}
	parts := make([]string, 0, len(counts))
	seen := map[domain.Category]struct{}{}
	for _, category := range order {
		if count, ok := counts[category]; ok {
			parts = append(parts, fmt.Sprintf("%s %d", queueCategoryLabel(category), count))
			seen[category] = struct{}{}
		}
	}
	for category, count := range counts {
		if _, ok := seen[category]; ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %d", queueCategoryLabel(category), count))
	}
	return strings.Join(parts, "  •  ")
}

func queueCategoryLabel(category domain.Category) string {
	switch category {
	case domain.CategoryDiskUsage:
		return "children"
	case domain.CategoryLargeFiles:
		return "files"
	case domain.CategoryTempFiles:
		return "temp"
	case domain.CategoryBrowserData:
		return "browser"
	case domain.CategoryDeveloperCaches:
		return "dev"
	case domain.CategoryPackageCaches:
		return "pkg"
	default:
		return strings.ToLower(strings.ReplaceAll(string(category), "_", " "))
	}
}

func analyzeAge(modified time.Time) string {
	if modified.IsZero() {
		return "--"
	}
	age := time.Since(modified)
	switch {
	case age < time.Hour:
		return "<1h"
	case age < 24*time.Hour:
		return fmt.Sprintf("%dh", int(age.Hours()))
	case age < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(age.Hours()/24))
	default:
		return fmt.Sprintf("%dw", int(age.Hours()/(24*7)))
	}
}

func coalesceAnalyzeQueueSort(value analyzeQueueSort) analyzeQueueSort {
	if value == "" {
		return analyzeQueueSortOrder
	}
	return value
}

func analyzeQueueSortText(value analyzeQueueSort) string {
	switch coalesceAnalyzeQueueSort(value) {
	case analyzeQueueSortSize:
		return "largest first"
	case analyzeQueueSortAge:
		return "oldest first"
	default:
		return "staged order"
	}
}

func analyzeSearchMatch(item domain.Finding, query string) bool {
	if query == "" {
		return true
	}
	fields := []string{
		item.Name,
		item.DisplayPath,
		item.Source,
		string(item.Category),
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func analyzeBreadcrumb(history []analyzeHistoryEntry, plan domain.ExecutionPlan) string {
	parts := make([]string, 0, len(history)+1)
	for _, entry := range history {
		parts = append(parts, breadcrumbLabel(entry.plan))
	}
	parts = append(parts, breadcrumbLabel(plan))
	return "Path " + strings.Join(parts, " > ")
}

func analyzeTrailLine(history []analyzeHistoryEntry, plan domain.ExecutionPlan, width int) string {
	sep := accentFrameStyle.Render(" → ")
	current := headerStyle.Render(breadcrumbLabel(plan))
	if len(history) == 0 {
		return singleLine("Path  "+current, width)
	}
	parent := mutedStyle.Render(breadcrumbLabel(history[len(history)-1].plan))
	line := "Path  " + parent + sep + current
	if len(history) > 1 {
		root := mutedStyle.Render(breadcrumbLabel(history[0].plan))
		suffix := mutedStyle.Render(fmt.Sprintf("(+%d)", len(history)-1))
		line = "Path  " + root + sep + current + "  " + suffix
	}
	return singleLine(line, width)
}

func analyzeHistoryCountLabel(history []analyzeHistoryEntry) string {
	if len(history) == 0 {
		return "root"
	}
	return fmt.Sprintf("%d %s", len(history), pl(len(history), "level", "levels"))
}

func breadcrumbLabel(plan domain.ExecutionPlan) string {
	if len(plan.Targets) == 0 {
		return "."
	}
	if len(plan.Targets) == 1 {
		return filepath.Base(plan.Targets[0])
	}
	return filepath.Base(plan.Targets[0]) + "…"
}
