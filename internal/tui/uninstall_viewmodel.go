package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/batu3384/sift/internal/domain"
)

func toneForUninstall(hasNative bool) string {
	if hasNative {
		return "review"
	}
	return "high"
}

func uninstallModeText(hasNative bool) string {
	if hasNative {
		return "native uninstall + remnants"
	}
	return "remnants only"
}

func uninstallNote(item uninstallItem) string {
	if item.Sensitive {
		return "Sensitive data detected. Review carefully."
	}
	if item.HasNative {
		return "Native uninstall stays explicit."
	}
	return "No native uninstall found."
}

func uninstallOriginText(item uninstallItem) string {
	if strings.TrimSpace(item.Origin) == "" {
		return "discovered install"
	}
	return item.Origin
}

func uninstallScopeText(item uninstallItem) string {
	if item.RequiresAdmin {
		return "admin boundary"
	}
	return "user space"
}

func uninstallRiskText(item uninstallItem) string {
	if item.Sensitive {
		return "sensitive review"
	}
	if item.RequiresAdmin {
		return "admin review"
	}
	if item.HasNative {
		return "review"
	}
	return "remnant only"
}

func uninstallSubtitle(item uninstallItem, ok bool) string {
	if !ok {
		return "select an app"
	}
	subtitle := uninstallModeText(item.HasNative)
	if item.Origin != "" {
		subtitle += " • " + item.Origin
	}
	if item.Sensitive {
		subtitle += " • sensitive"
	}
	return subtitle
}

func uninstallSelectionLine(m uninstallModel, item uninstallItem, ok bool) string {
	if !ok || len(m.filtered) == 0 {
		return "State   none  •  choose an app"
	}
	index := m.cursor + 1
	parts := []string{fmt.Sprintf("State   %d/%d", index, len(m.filtered)), item.Name}
	if item.HasNative {
		parts = append(parts, "native")
	} else {
		parts = append(parts, "remnants")
	}
	if m.isStaged(item) {
		parts = append(parts, "queued")
	}
	return strings.Join(parts, "  •  ")
}

func uninstallNextLine(m uninstallModel, item uninstallItem, ok bool) string {
	parts := []string{"Next"}
	switch {
	case m.loading:
		parts = append(parts, "wait for refresh")
	case m.searchActive:
		parts = append(parts, "type to filter", "enter apply", "esc clear")
	case !ok:
		parts = append(parts, "pick an app", "/ filter")
	default:
		if _, loaded := m.previewPlanForSelected(); loaded {
			parts = append(parts, "enter review")
		} else {
			parts = append(parts, "enter open")
		}
		if m.isStaged(item) {
			parts = append(parts, "u remove")
		} else {
			parts = append(parts, "space queue")
		}
		if m.stageCount() > 0 {
			parts = append(parts, "x batch")
		}
		if !item.HasNative {
			parts = append(parts, "check files")
		}
	}
	return strings.Join(parts, "  •  ")
}

func uninstallPreviewLines(m uninstallModel, item uninstallItem, width int) []string {
	if strings.TrimSpace(m.preview.key) != uninstallStageKey(item.Name) {
		return nil
	}
	switch {
	case m.preview.loading:
		return []string{mutedStyle.Render("Preview  loading review")}
	case m.preview.err != "":
		return []string{highStyle.Render("Preview  unavailable")}
	case m.preview.loaded:
		mods := planModuleCount(m.preview.plan)
		lines := []string{
			mutedStyle.Render("Preview   ") + fmt.Sprintf("%d ready  •  %d %s  •  %s", actionableCount(m.preview.plan), mods, pl(mods, "module", "modules"), domain.HumanBytes(planDisplayBytes(m.preview.plan))),
		}
		if len(m.preview.plan.Warnings) > 0 {
			lines = append(lines, mutedStyle.Render("Note      ")+truncateText(m.preview.plan.Warnings[0], width))
		}
		return lines
	default:
		return nil
	}
}

func uninstallStats(m uninstallModel, width int) []string {
	cardWidth := 24
	if width < 110 {
		cardWidth = width - 8
	}
	mode := "idle"
	tone := "review"
	if m.loading {
		mode = "loading"
	} else if m.searchActive {
		mode = "filtering"
		tone = "safe"
	}
	native := 0
	admin := 0
	sensitive := 0
	for _, item := range m.items {
		if item.HasNative {
			native++
		}
		if item.RequiresAdmin {
			admin++
		}
		if item.Sensitive {
			sensitive++
		}
	}
	remnants := len(m.items) - native
	return []string{
		renderStatCard("installed", fmt.Sprintf("%d %s", len(m.items), pl(len(m.items), "app", "apps")), "safe", cardWidth),
		renderStatCard("native", fmt.Sprintf("%d native", native), "review", cardWidth),
		renderStatCard("remnants", fmt.Sprintf("%d review", remnants), "high", cardWidth),
		renderStatCard("sensitive", fmt.Sprintf("%d guarded", sensitive), "high", cardWidth),
		renderStatCard("queued", fmt.Sprintf("%d staged", m.stageCount()), "safe", cardWidth),
		renderStatCard("matches", fmt.Sprintf("%d %s", len(m.filtered), pl(len(m.filtered), "result", "results")), "review", cardWidth),
		renderStatCard("admin", fmt.Sprintf("%d gated", admin), "high", cardWidth),
		renderStatCard("cache", mode, tone, cardWidth),
	}
}

func coalesceText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func uninstallInventoryMessage(at time.Time, cached bool) string {
	if at.IsZero() {
		if cached {
			return "Cached app inventory loaded. Refreshing live install data..."
		}
		return "Installed app inventory refreshed."
	}
	label := formatAppModified(at)
	if label == "" {
		label = "just now"
	}
	if cached {
		return "Cached app inventory loaded (" + label + "). Refreshing live install data..."
	}
	return "Installed app inventory refreshed (" + label + ")."
}

func uninstallMatchesQuery(item uninstallItem, query string) bool {
	fields := []string{
		item.Name,
		item.Origin,
		item.Location,
		item.SizeLabel,
		strings.Join(item.FamilyMatches, " "),
		uninstallScopeText(item),
		uninstallModeText(item.HasNative),
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func compareUninstallItems(left, right uninstallItem) bool {
	leftScore := uninstallPriority(left)
	rightScore := uninstallPriority(right)
	if leftScore != rightScore {
		return leftScore > rightScore
	}
	if !left.LastSeenAt.Equal(right.LastSeenAt) {
		if left.LastSeenAt.IsZero() {
			return false
		}
		if right.LastSeenAt.IsZero() {
			return true
		}
		return left.LastSeenAt.After(right.LastSeenAt)
	}
	return strings.ToLower(left.Name) < strings.ToLower(right.Name)
}

func uninstallPriority(item uninstallItem) int {
	score := 0
	if item.HasNative {
		score += 4
	}
	if item.Sensitive {
		score += 2
	}
	if !item.RequiresAdmin {
		score += 2
	}
	switch strings.ToLower(strings.TrimSpace(item.Origin)) {
	case "homebrew cask", "setapp", "user program", "user application":
		score += 2
	case "registry uninstall":
		score += 1
	}
	return score
}

func uninstallSizeLabel(bytes int64) string {
	if bytes <= 0 {
		return ""
	}
	return domain.HumanBytes(bytes)
}

func uninstallFamilyText(item uninstallItem) string {
	if len(item.FamilyMatches) == 0 {
		return "none"
	}
	labels := make([]string, 0, len(item.FamilyMatches))
	for _, family := range item.FamilyMatches {
		labels = append(labels, strings.ReplaceAll(family, "_", " "))
	}
	return strings.Join(labels, ", ")
}

func uninstallStageKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (m uninstallModel) batchSummary(limit int) string {
	names := m.stageNames()
	if len(names) == 0 {
		return "none"
	}
	if len(names) <= limit {
		return strings.Join(names, ", ")
	}
	return strings.Join(names[:limit], ", ") + fmt.Sprintf(" +%d", len(names)-limit)
}
