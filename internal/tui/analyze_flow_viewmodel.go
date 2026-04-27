package tui

import (
	"fmt"
	"strings"

	"github.com/batu3384/sift/internal/domain"
)

func analyzeFlowStats(flow analyzeFlowModel, base analyzeBrowserModel, width int) []string {
	cardWidth := 24
	if width < 110 {
		cardWidth = width - 8
	}
	diskUsage := findingsByCategory(base.plan.Items, domain.CategoryDiskUsage)
	largeFiles := findingsByCategory(base.plan.Items, domain.CategoryLargeFiles)
	state, tone := analyzeFlowStateCard(flow, base)
	cards := []string{
		renderRouteStatCard("analyze", "children", fmt.Sprintf("%d mapped", len(diskUsage)), "safe", cardWidth),
		renderRouteStatCard("analyze", "files", fmt.Sprintf("%d large", len(largeFiles)), "review", cardWidth),
		renderRouteStatCard("analyze", "reclaim", domain.HumanBytes(base.plan.Totals.Bytes), "review", cardWidth),
		renderRouteStatCard("analyze", "state", state, tone, cardWidth),
	}
	if width >= 128 {
		cards = append(cards, renderRouteStatCard("analyze", "queued", fmt.Sprintf("%d staged", len(base.stageOrder)), "safe", cardWidth))
	}
	return cards
}

func analyzeFlowStateCard(flow analyzeFlowModel, base analyzeBrowserModel) (string, string) {
	state := "READY"
	tone := "safe"
	switch flow.phase {
	case analyzeFlowInspecting:
		state = "SCANNING"
		tone = "review"
	case analyzeFlowReviewReady:
		state = "REVIEW"
		tone = "review"
	case analyzeFlowPermissions:
		state = "ACCESS"
		tone = "high"
	case analyzeFlowReclaiming:
		state = "RECLAIM"
		tone = "review"
	case analyzeFlowResult:
		state = "SETTLED"
		tone = "safe"
	default:
		if base.loading {
			state = "SCANNING"
			tone = "review"
		} else if strings.TrimSpace(base.errMsg) != "" {
			state = "NEEDS REVIEW"
			tone = "high"
		}
	}
	if meter := routeStickyPhaseCount("analyze", analyzeFlowSignalLabel(flow)); meter != "" {
		state += " " + meter
	}
	return state, tone
}
