package tui

import (
	"fmt"
	"strings"

	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/platform"
	"github.com/batu3384/sift/internal/store"
)

func homeSubtitle(executable bool, _ config.Config) string {
	mode := "preview mode"
	if executable {
		mode = "review mode"
	}
	return mode + "  •  t opens tools"
}

func homeHealthLine(live *engine.SystemSnapshot) string {
	return fmt.Sprintf("Live %d (%s)  •  CPU %.1f%%  •  Mem %.1f%%  •  Free %s",
		live.HealthScore,
		strings.ToUpper(live.HealthLabel),
		live.CPUPercent,
		live.MemoryUsedPercent,
		domain.HumanBytes(int64(live.DiskFreeBytes)),
	)
}

func homeExecutionLine(execution *store.ExecutionSummary) string {
	settled := execution.Completed + execution.Deleted
	parts := []string{fmt.Sprintf("%d settled", settled)}
	if execution.FreedBytes > 0 {
		parts = append(parts, domain.HumanBytes(execution.FreedBytes)+" freed")
	}
	if execution.Protected > 0 {
		parts = append(parts, fmt.Sprintf("%d protected", execution.Protected))
	}
	if execution.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", execution.Failed))
	}
	return "Last execution " + strings.Join(parts, "  •  ")
}

func homeDiagnosticLine(diagnostics []platform.Diagnostic) string {
	issues := diagnosticIssueCount(diagnostics)
	if issues == 0 {
		return "State  •  no current issues"
	}
	if issues == 1 {
		return "State  •  1 issue needs review"
	}
	return fmt.Sprintf("State  •  %d issues need review", issues)
}

func diagnosticIssueCount(diagnostics []platform.Diagnostic) int {
	count := 0
	for _, d := range diagnostics {
		if d.Status == "warn" || d.Status == "error" {
			count++
		}
	}
	return count
}

func homeRecommendedAction(actions []homeAction, cursor int, diagnostics []platform.Diagnostic, update *engine.UpdateNotice) string {
	switch {
	case update != nil && update.Available:
		return "run sift update"
	case diagnosticIssueCount(diagnostics) > 0:
		return "press t for sift check"
	case cursor >= 0 && cursor < len(actions):
		action := actions[cursor]
		if command := strings.TrimSpace(action.Command); command != "" {
			return command
		}
		return strings.ToLower(action.Title)
	default:
		return "open status"
	}
}

func homeSessionRailLine(live *engine.SystemSnapshot, lastExecution *store.ExecutionSummary) string {
	parts := []string{"Last"}
	if lastExecution != nil {
		settled := lastExecution.Completed + lastExecution.Deleted
		parts = append(parts, fmt.Sprintf("%d settled", settled))
		if lastExecution.FreedBytes > 0 {
			parts = append(parts, domain.HumanBytes(lastExecution.FreedBytes)+" freed")
		}
		if lastExecution.Protected > 0 {
			parts = append(parts, fmt.Sprintf("%d protected", lastExecution.Protected))
		}
		if lastExecution.Failed > 0 {
			parts = append(parts, fmt.Sprintf("%d failed", lastExecution.Failed))
		}
		return strings.Join(parts, "  •  ")
	}
	if live != nil {
		parts = append(parts, fmt.Sprintf("health %d", live.HealthScore), "monitor ready")
		return strings.Join(parts, "  •  ")
	}
	return "Last  •  no recent execution"
}

func menuListSummary(action homeAction) string {
	if len(action.Modules) > 0 {
		return strings.Join(action.Modules[:min(len(action.Modules), 2)], "  •  ")
	}
	return action.Description
}

func homeDetailSubtitle(actions []homeAction, cursor int) string {
	if cursor < 0 || cursor >= len(actions) {
		return ""
	}
	action := actions[cursor]
	if !action.Enabled {
		return "setup needed"
	}
	return action.Command
}

func homeDetailNextLine(action homeAction) string {
	if !action.Enabled {
		return "finish setup first"
	}
	switch action.ID {
	case "clean":
		return "enter picks clean scope"
	case "uninstall":
		return "enter opens apps"
	case "optimize":
		return "enter opens review"
	case "analyze":
		return "enter opens browser"
	case "status":
		return "enter opens status"
	default:
		if command := strings.TrimSpace(action.Command); command != "" {
			return command
		}
		return "enter opens item"
	}
}

func homeDetailStateLine(cfg config.Config, diagnostics []platform.Diagnostic) string {
	nFamilies := len(cfg.ProtectedFamilies)
	parts := []string{
		fmt.Sprintf("%d protected", len(cfg.ProtectedPaths)),
		fmt.Sprintf("%d %s", nFamilies, pl(nFamilies, "family", "families")),
	}
	scopes := len(cfg.CommandExcludes)
	parts = append(parts, fmt.Sprintf("%d %s", scopes, pl(scopes, "scope", "scopes")))
	if roots := len(cfg.PurgeSearchPaths); roots > 0 {
		parts = append(parts, fmt.Sprintf("%d purge %s", roots, pl(roots, "root", "roots")))
	}
	if issues := diagnosticIssueCount(diagnostics); issues > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", issues, pl(issues, "issue", "issues")))
	}
	return "State    " + strings.Join(parts, "  •  ")
}

func homeSpotlightSubtitle(actions []homeAction, cursor int, live *engine.SystemSnapshot, lastExecution *store.ExecutionSummary, diagnostics []platform.Diagnostic, update *engine.UpdateNotice) string {
	actionTitle := "ready"
	if cursor >= 0 && cursor < len(actions) {
		actionTitle = strings.ToLower(actions[cursor].Title)
	}
	parts := []string{"focus " + actionTitle}
	if issues := diagnosticIssueCount(diagnostics); issues > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", issues, pl(issues, "issue", "issues")))
	}
	if live != nil {
		parts = append(parts, fmt.Sprintf("health %d", live.HealthScore))
	}
	if lastExecution != nil {
		parts = append(parts, fmt.Sprintf("last run %d deleted", lastExecution.Deleted))
	}
	if update != nil && update.Available {
		parts = append(parts, "update "+strings.ToUpper(update.LatestVersion))
	}
	return strings.Join(parts, "  •  ")
}

func homeWatchLine(live *engine.SystemSnapshot, diagnostics []platform.Diagnostic, update *engine.UpdateNotice) string {
	parts := []string{}
	if issues := diagnosticIssueCount(diagnostics); issues > 0 {
		parts = append(parts, fmt.Sprintf("%d doctor %s", issues, pl(issues, "issue", "issues")))
	}
	if update != nil && update.Available {
		parts = append(parts, strings.ToUpper(update.LatestVersion)+" ready")
	}
	if live != nil && len(live.OperatorAlerts) > 0 {
		parts = append(parts, live.OperatorAlerts[0])
	}
	if len(parts) == 0 {
		parts = append(parts, "system steady")
	}
	return "Alerts   " + strings.Join(parts, "  •  ")
}

func homeStateSummaryLine(lastExecution *store.ExecutionSummary, cfg config.Config, _ []platform.Diagnostic) string {
	parts := []string{}
	if lastExecution != nil {
		parts = append(parts, fmt.Sprintf("%d completed", lastExecution.Completed), fmt.Sprintf("%d deleted", lastExecution.Deleted))
		if lastExecution.Protected > 0 {
			parts = append(parts, fmt.Sprintf("%d protected", lastExecution.Protected))
		}
	} else {
		parts = append(parts, "no recent run")
	}
	scopes := len(cfg.CommandExcludes)
	parts = append(parts, fmt.Sprintf("%d %s", scopes, pl(scopes, "scope", "scopes")))
	if roots := len(cfg.PurgeSearchPaths); roots > 0 {
		parts = append(parts, fmt.Sprintf("%d purge %s", roots, pl(roots, "root", "roots")))
	}
	return "Activity " + strings.Join(parts, "  •  ")
}

func homeNextLine(actions []homeAction, cursor int, live *engine.SystemSnapshot, lastExecution *store.ExecutionSummary, diagnostics []platform.Diagnostic, update *engine.UpdateNotice) string {
	parts := []string{homeRecommendedAction(actions, cursor, diagnostics, update)}
	switch {
	case update != nil && update.Available:
		parts = append(parts, "t opens check")
	case diagnosticIssueCount(diagnostics) > 0:
		parts = append(parts, "t opens check")
	case live != nil && len(live.OperatorAlerts) > 0:
		parts = append(parts, "open status")
	case lastExecution != nil && lastExecution.Deleted > 0:
		parts = append(parts, "open analyze")
	default:
		parts = append(parts, "open status")
	}
	return "Next     " + strings.Join(parts, "  •  ")
}

func homeSignalStateLabel(motion motionState) string {
	switch motion.Mode {
	case motionModeAlert:
		return "needs attention"
	case motionModeLoading:
		return "refreshing"
	case motionModeReview:
		return "reviewing"
	default:
		return "ready"
	}
}

func homeStats(live *engine.SystemSnapshot, lastExecution *store.ExecutionSummary, diagnostics []platform.Diagnostic, update *engine.UpdateNotice, width int) []string {
	cardWidth := 23
	if width < 110 {
		cardWidth = width - 6
	}
	stats := []string{}
	if live != nil && live.DiskTotalBytes > 0 {
		storageStats := &StorageStats{
			Total: int64(live.DiskTotalBytes),
			Used:  int64(live.DiskTotalBytes) - int64(live.DiskFreeBytes),
			Free:  int64(live.DiskFreeBytes),
		}
		stats = append(stats, renderStorageCard(storageStats, cardWidth))
	}

	if live != nil {
		stats = append(stats,
			renderStatCard("live", fmt.Sprintf("%d / %s", live.HealthScore, strings.ToUpper(live.HealthLabel)), toneForHealth(live.HealthScore), cardWidth),
			renderStatCard("disk", domain.HumanBytes(int64(live.DiskFreeBytes)), "safe", cardWidth),
		)
	} else {
		stats = append(stats,
			renderStatCard("live", "loading", "review", cardWidth),
			renderStatCard("disk", "unknown", "review", cardWidth),
		)
	}
	warnings := diagnosticIssueCount(diagnostics)
	recentValue := "no runs"
	recentTone := "review"
	if lastExecution != nil {
		recentValue = fmt.Sprintf("%d deleted", lastExecution.Deleted)
		recentTone = "safe"
	}
	stats = append(stats, renderStatCard("last", recentValue, recentTone, cardWidth))
	watchValue := fmt.Sprintf("%d active", warnings)
	watchTone := toneForIssues(warnings)
	if update != nil && update.Available {
		watchValue = strings.ToUpper(update.LatestVersion) + " ready"
		watchTone = "review"
	} else if live != nil && len(live.OperatorAlerts) > 0 {
		watchValue = live.OperatorAlerts[0]
		watchTone = "review"
	}
	stats = append(stats, renderStatCard("alerts", watchValue, watchTone, cardWidth))
	return stats
}

func toneForHealth(score int) string {
	switch {
	case score >= 85:
		return "safe"
	case score >= 70:
		return "review"
	default:
		return "high"
	}
}

func toneForIssues(count int) string {
	if count == 0 {
		return "safe"
	}
	return "review"
}

type StorageStats struct {
	Total     int64
	Used      int64
	Free      int64
	Apps      int64
	Documents int64
	Photos    int64
	System    int64
	Other     int64
}
