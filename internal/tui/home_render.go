package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/platform"
	"github.com/batu3384/sift/internal/store"
)

func homeMenuView(actions []homeAction, cursor int, width int, maxLines int) string {
	lines := make([]string, 0, len(actions))
	for i, action := range actions {
		marker := selectionPrefix(i == cursor)
		if !action.Enabled {
			marker = "· "
		}
		title := truncateText(action.Title, 22)
		line := fmt.Sprintf("%s%s  %s", marker, title, renderToneToken(action.Tone))
		if width >= 70 {
			line += "  " + mutedStyle.Render(truncateText(menuListSummary(action), max(width-26, 18)))
		}
		if !action.Enabled {
			line += "  " + highStyle.Render("setup")
		}
		line = singleLine(line, width)
		if i == cursor {
			line = selectedLine.Render(line)
		}
		lines = append(lines, line)
	}
	lines = viewportLines(lines, cursor, maxLines)
	return strings.Join(lines, "\n")
}

func homeDetailView(actions []homeAction, cursor int, live *engine.SystemSnapshot, lastExecution *store.ExecutionSummary, diagnostics []platform.Diagnostic, cfg config.Config, width int, maxLines int) string {
	if cursor < 0 || cursor >= len(actions) {
		return "No action selected."
	}
	action := actions[cursor]
	lines := []string{
		renderToneBadge(action.Tone) + " " + headerStyle.Render(action.Title),
		wrapText(truncateText(action.Description, width), width),
	}
	if next := homeDetailNextLine(action); next != "" {
		lines = append(lines, mutedStyle.Render("Next     ")+wrapText(next, width))
	}
	if action.Safety != "" {
		lines = append(lines, mutedStyle.Render("Guard    ")+wrapText(truncateText(action.Safety, width), width))
	}
	if action.ID == "status" && live != nil {
		lines = append(lines, mutedStyle.Render(homeHealthLine(live)))
	}
	if action.ID == "optimize" {
		lines = append(lines, mutedStyle.Render(homeDetailStateLine(cfg, diagnostics)))
	}
	if (action.ID == "clean" || action.ID == "optimize") && lastExecution != nil {
		lines = append(lines, mutedStyle.Render("Last     ")+homeExecutionLine(lastExecution))
	}
	if action.ID == "optimize" && len(cfg.PurgeSearchPaths) == 0 {
		lines = append(lines, highStyle.Render("Setup    add purge_search_paths"))
	}
	lines = viewportLines(lines, 0, maxLines)
	return strings.Join(lines, "\n")
}

func homeDetailCompactView(actions []homeAction, cursor int, live *engine.SystemSnapshot, lastExecution *store.ExecutionSummary, diagnostics []platform.Diagnostic, cfg config.Config, width int, maxLines int) string {
	if cursor < 0 || cursor >= len(actions) {
		return mutedStyle.Render("No action selected.")
	}
	action := actions[cursor]
	lines := []string{
		renderToneBadge(action.Tone) + " " + headerStyle.Render(action.Title),
		mutedStyle.Render(truncateText(action.Description, width)),
	}
	if next := homeDetailNextLine(action); next != "" {
		lines = append(lines, mutedStyle.Render("Next  ")+truncateText(next, width))
	}
	switch action.ID {
	case "status":
		if live != nil {
			lines = append(lines, mutedStyle.Render(truncateText(homeHealthLine(live), width)))
		}
		lines = append(lines, mutedStyle.Render(truncateText(homeWatchLine(live, diagnostics, nil), width)))
	case "optimize":
		lines = append(lines, mutedStyle.Render(truncateText(homeDetailStateLine(cfg, diagnostics), width)))
		if len(diagnostics) > 0 {
			lines = append(lines, mutedStyle.Render(homeDiagnosticLine(diagnostics)))
		}
	case "clean":
		if lastExecution != nil {
			lines = append(lines, mutedStyle.Render(truncateText(homeSessionRailLine(live, lastExecution), width)))
		}
	}
	lines = viewportLines(lines, 0, maxLines)
	return strings.Join(lines, "\n")
}

func homeSpotlightView(actions []homeAction, cursor int, live *engine.SystemSnapshot, lastExecution *store.ExecutionSummary, diagnostics []platform.Diagnostic, update *engine.UpdateNotice, cfg config.Config, motion motionState, width int, maxLines int) string {
	lines := make([]string, 0, 8)
	compact := width < 96 || maxLines <= 5
	showMascot := width >= 120 && maxLines >= 6
	textWidth := width
	if showMascot {
		textWidth = width - 11
	}
	lines = append(lines, mutedStyle.Render(singleLine("Signal   "+signalRailLabelForMotion(motion)+"  •  "+homeSignalStateLabel(motion), textWidth)))
	if cursor >= 0 && cursor < len(actions) {
		action := actions[cursor]
		focusParts := []string{}
		focusParts = append(focusParts, action.Title)
		if command := strings.TrimSpace(action.Command); command != "" {
			focusParts = append(focusParts, command)
		}
		lines = append(lines, mutedStyle.Render(singleLine("Focus    "+strings.Join(focusParts, "  •  ")+"  "+renderToneToken(action.Tone), textWidth)))
	}
	lines = append(lines, mutedStyle.Render(singleLine(homeWatchLine(live, diagnostics, update), textWidth)))
	lines = append(lines, mutedStyle.Render(singleLine(homeStateSummaryLine(lastExecution, cfg, diagnostics), textWidth)))
	if !compact && live != nil {
		lines = append(lines, mutedStyle.Render(singleLine(homeHealthLine(live), textWidth)))
	}
	lines = append(lines, mutedStyle.Render(singleLine(homeNextLine(actions, cursor, live, lastExecution, diagnostics, update), textWidth)))
	lines = viewportLines(lines, 0, maxLines)
	content := strings.Join(lines, "\n")
	if showMascot {
		cpuPercent := 0.0
		if live != nil {
			cpuPercent = live.CPUPercent
		}
		return lipgloss.JoinHorizontal(lipgloss.Top, content, "  ", mascotFrameFromMotion(motion, cpuPercent))
	}
	return content
}

// renderStorageCard creates a compact storage info card
func renderStorageCard(stats *StorageStats, cardWidth int) string {
	if stats == nil {
		return ""
	}

	usedPercent := 0.0
	if stats.Total > 0 {
		usedPercent = float64(stats.Used) / float64(stats.Total) * 100
	}

	barWidth := 20
	filled := int(usedPercent / 100 * float64(barWidth))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	var tone string
	switch {
	case usedPercent < 70:
		tone = "safe"
	case usedPercent < 85:
		tone = "review"
	default:
		tone = "high"
	}

	// Apply tone-based styling
	cardStyle := panelStyle
	switch tone {
	case "safe":
		cardStyle = cardStyle.BorderForeground(lipgloss.Color("#76D39B"))
	case "review":
		cardStyle = cardStyle.BorderForeground(lipgloss.Color("#E7BE5B"))
	case "high":
		cardStyle = cardStyle.BorderForeground(lipgloss.Color("#F08B74"))
	}

	card := cardStyle.Width(cardWidth).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			panelTitleStyle.Render("Storage"),
			"",
			fmt.Sprintf("[%s] %.1f%%", bar, usedPercent),
			fmt.Sprintf("Used: %s / %s", domain.HumanBytes(stats.Used), domain.HumanBytes(stats.Total)),
			fmt.Sprintf("Free: %s", domain.HumanBytes(stats.Free)),
		),
	)

	return card
}
