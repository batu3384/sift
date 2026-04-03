package tui

import (
	"fmt"
	"strings"

	"github.com/batu3384/sift/internal/domain"
)

func analyzeStats(plan domain.ExecutionPlan, stageOrder []string, loading bool, errMsg string, width int) []string {
	cardWidth := 24
	if width < 110 {
		cardWidth = width - 8
	}
	diskUsage := findingsByCategory(plan.Items, domain.CategoryDiskUsage)
	largeFiles := findingsByCategory(plan.Items, domain.CategoryLargeFiles)
	state := "Ready"
	tone := "safe"
	if loading {
		state = "Scanning"
		tone = "review"
	} else if errMsg != "" {
		state = "Needs review"
		tone = "high"
	}
	cards := []string{
		renderStatCard("children", fmt.Sprintf("%d mapped", len(diskUsage)), "safe", cardWidth),
		renderStatCard("files", fmt.Sprintf("%d large", len(largeFiles)), "review", cardWidth),
		renderStatCard("reclaim", domain.HumanBytes(plan.Totals.Bytes), "review", cardWidth),
		renderStatCard("state", state, tone, cardWidth),
	}
	if width >= 128 {
		cards = append(cards, renderStatCard("queued", fmt.Sprintf("%d staged", len(stageOrder)), "safe", cardWidth))
	}
	return cards
}

func analyzeListView(model analyzeBrowserModel, width int, maxLines int) string {
	if len(model.plan.Items) == 0 {
		return mutedStyle.Render("No analysis findings for this target.")
	}
	visible := model.visibleIndices()
	lines := []string{}
	if model.searchActive || strings.TrimSpace(model.search.Value()) != "" {
		lines = append(lines, railStyle.Render("SEARCH"), model.search.View(), "")
	}
	if width >= 54 && maxLines >= 9 {
		lines = append(lines, analyzeSummaryLines(model.plan)...)
		lines = append(lines, mutedStyle.Render(analyzeBrowseRail(model)))
		if len(model.plan.Warnings) > 0 {
			lines = append(lines, mutedStyle.Render(truncateText(strings.Join(model.plan.Warnings, " | "), width)))
		}
	}
	if len(visible) == 0 {
		lines = append(lines, "", mutedStyle.Render("No findings match the current analyze filter."))
		return strings.Join(lines, "\n")
	}
	focusLine := len(lines) // default to end of header block
	currentCategory := domain.Category("")
	for visibleIdx, itemIdx := range visible {
		item := model.plan.Items[itemIdx]
		if item.Category != currentCategory {
			currentCategory = item.Category
			if width >= 48 {
				lines = append(lines, "", headerStyle.Render(sectionTitle(model.plan, currentCategory)))
			}
		}
		staged := ""
		if _, ok := model.staged[item.Path]; ok {
			staged = safeStyle.Render("✓ ")
		}
		line := selectionPrefix(visibleIdx == model.cursor) + staged + analyzeLine(item)
		line = singleLine(line, width)
		if visibleIdx == model.cursor {
			line = selectedLine.Render(line)
			focusLine = len(lines)
		}
		lines = append(lines, line)
	}
	lines = viewportLines(lines, focusLine, maxLines)
	return strings.Join(lines, "\n")
}

func analyzeDetailSubtitle(model analyzeBrowserModel) string {
	item, ok := model.selectedActiveItem()
	if !ok {
		if query := strings.TrimSpace(model.search.Value()); query != "" {
			return fmt.Sprintf("search %q", query)
		}
		return currentAnalyzeTarget(model.plan)
	}
	return fmt.Sprintf("%s • %s", item.Name, sectionTitle(model.plan, item.Category))
}

func analyzeDetailView(m analyzeBrowserModel, width int, maxLines int) string {
	queueOrder := m.sortedStageOrder()
	lines := []string{}
	if width >= 48 && maxLines >= 8 {
		lines = append(lines, mutedStyle.Render(analyzeTrailLine(m.history, m.plan, width)))
	}
	lines = append(lines, mutedStyle.Render(analyzeQueueRail(m, queueOrder)))
	lines = append(lines, mutedStyle.Render(analyzeSelectionLine(m)))
	lines = append(lines, mutedStyle.Render(analyzeAtmosphereLine(m)))
	lines = append(lines, mutedStyle.Render(analyzeActionRail(m)))
	if preview := analyzeReviewPreviewLines(m, width); len(preview) > 0 {
		lines = append(lines, preview...)
	}
	if m.loading {
		lines = append(lines, "", reviewStyle.Render("Scanning..."))
	}
	if m.errMsg != "" {
		lines = append(lines, "", highStyle.Render("Navigation error: "+m.errMsg))
	}
	if item, ok := m.selectedActiveItem(); ok {
		detailLine := fmt.Sprintf("%s  %s", domain.HumanBytes(item.Bytes), item.DisplayPath)
		if risk := strings.TrimSpace(string(item.Risk)); risk != "" {
			detailLine += "  •  " + styleForRisk(item.Risk).Render(strings.ToUpper(risk))
		}
		lines = append(lines, renderSectionRule(width), detailLine)
		if impact := analyzeImpactLine(item, m.plan); impact != "" {
			lines = append(lines, mutedStyle.Render("Impact  "+impact))
		}
		if currentView := analyzeCurrentViewLines(item, m, width); len(currentView) > 0 {
			lines = append(lines, headerStyle.Render("Focus"))
			lines = append(lines, currentView...)
		}
		previewLines := analyzeDirectoryPreviewLines(m.selectedPreview(), width)
		if len(previewLines) == 0 {
			if previewText := strings.TrimSpace(analyzePreviewText(item, m.plan)); previewText != "" {
				previewLines = []string{mutedStyle.Render(wrapText(previewText, width))}
			}
		}
		if len(previewLines) > 0 {
			lines = append(lines, headerStyle.Render("Preview"))
			lines = append(lines, previewLines...)
		}
		if item.Source != "" && width >= 52 {
			lines = append(lines, mutedStyle.Render("From    ")+trimAnalyzeSource(item.Source))
		}
		if !item.LastModified.IsZero() && width >= 52 {
			lines = append(lines, mutedStyle.Render("Last    ")+item.LastModified.Local().Format("2006-01-02 15:04"))
		}
	}
	if len(m.stageOrder) > 1 && width >= 52 && maxLines >= 10 {
		lines = append(lines, renderSectionRule(width), headerStyle.Render("Queue"))
		lines = append(lines, analyzeBatchHighlightLines(m.staged, queueOrder, width)...)
		lines = append(lines, renderSectionRule(width), headerStyle.Render("Items"))
		lines = append(lines, mutedStyle.Render("Sort  "+analyzeQueueSortText(m.queueSort)))
		for _, path := range queueOrder {
			item, ok := m.staged[path]
			if !ok {
				continue
			}
			lines = append(lines, wrapText(fmt.Sprintf("%-10s %s", domain.HumanBytes(item.Bytes), item.DisplayPath), width))
		}
	}
	lines = viewportLines(lines, 0, maxLines)
	return strings.Join(lines, "\n")
}
