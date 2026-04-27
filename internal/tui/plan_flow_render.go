package tui

import (
	"fmt"
	"strings"

	"github.com/batu3384/sift/internal/domain"
)

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
			mutedStyle.Render("Status   "+planDetailStatusLine(plan, item)),
			mutedStyle.Render("Scope    "+planDetailScopeLine(model, item)),
			mutedStyle.Render("Next     "+planDetailNextLine(plan, item)),
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

func decisionView(model planModel, width int) string {
	plan := model.effectivePlan()
	lines := []string{
		mutedStyle.Render("Status   " + planDecisionStatusLine(plan)),
	}
	if scope := planReviewScopeLine(plan); scope != "" {
		lines = append(lines, wrapText(mutedStyle.Render("Scope    "+scope), width))
	}
	if outcome := planReviewOutcomeLine(plan); outcome != "" {
		lines = append(lines, wrapText(mutedStyle.Render("Next     "+outcome), width))
	}
	if gate := planDecisionGateLine(plan); gate != "" {
		lines = append(lines, wrapText(mutedStyle.Render("Gate     "+gate), width))
	}
	lines = append(lines, planTrustSummaryLines(plan, width)...)
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
