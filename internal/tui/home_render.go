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
		marker := "> "
		if !action.Enabled {
			marker = "- "
		}

		// Bold, uppercase title with tone-based color
		toneStyle := getToneStyle(action.Tone)
		title := strings.ToUpper(truncateText(action.Title, 20))
		titleRendered := toneStyle.Bold(true).Render(title)

		// Marker + title
		line := fmt.Sprintf("%s%s", marker, titleRendered)

		// Add tone badge if enabled
		if action.Enabled {
			badgeText := strings.ToUpper(action.Tone)
			badge := toneStyle.Render(fmt.Sprintf(" [%s]", badgeText))
			line += badge
		} else {
			line += "  " + highStyle.Render("[SETUP]")
		}

		line = singleLine(line, width)

		// Colored left border when selected
		if i == cursor {
			leftBorder := toneStyle.Render("# ")
			line = leftBorder + line
		} else {
			line = "  " + line
		}

		lines = append(lines, line)
	}
	lines = viewportLines(lines, cursor, maxLines)
	return strings.Join(lines, "\n")
}

// Helper function to get tone-based style
func getToneStyle(tone string) lipgloss.Style {
	switch tone {
	case "safe":
		return safeTokenStyle
	case "review":
		return reviewTokenStyle
	case "high":
		return highTokenStyle
	default:
		return mutedStyle
	}
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
	lines = append(lines, mutedStyle.Render("Status   ")+wrapText(homeDetailStatusLine(action, live, diagnostics, cfg), width))
	if next := homeDetailNextLine(action); next != "" {
		lines = append(lines, mutedStyle.Render("Next     ")+wrapText(next, width))
	}
	if action.Safety != "" {
		lines = append(lines, mutedStyle.Render("Gate     ")+wrapText(truncateText(action.Safety, width), width))
	}
	if action.ID == "status" && live != nil {
		lines = append(lines, mutedStyle.Render(homeHealthLine(live)))
	}
	if action.ID == "optimize" {
		lines = append(lines, mutedStyle.Render(homeDetailStateLine(cfg, diagnostics)))
	}
	if (action.ID == "clean" || action.ID == "optimize") && lastExecution != nil {
		lines = append(lines, mutedStyle.Render("Carry    ")+homeExecutionLine(lastExecution))
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
	lines = append(lines, mutedStyle.Render("Status  ")+truncateText(homeDetailStatusLine(action, live, diagnostics, cfg), width))
	if next := homeDetailNextLine(action); next != "" {
		lines = append(lines, mutedStyle.Render("Next    ")+truncateText(next, width))
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
	showMascot := width >= 120 && maxLines >= 6
	textWidth := width
	if showMascot {
		textWidth = width - 11
	}
	signature := routeSignalSignatureForRoute("home")
	lead := signature.Mascot
	if signature.Doctrine != "" {
		lead += "  •  " + signature.Doctrine
	}
	lines = append(lines, mutedStyle.Render(singleLine(lead+"  •  Command  "+signalRailLabelForMotion(motion)+"  •  "+homeSignalStateLabel(motion), textWidth)))
	lines = append(lines, mutedStyle.Render(singleLine("Status   "+homeSpotlightStatusLine(actions, cursor, live, diagnostics, cfg), textWidth)))
	if cursor >= 0 && cursor < len(actions) {
		action := actions[cursor]
		lines = append(lines, mutedStyle.Render(singleLine("Focus    "+homeFocusLine(actions, cursor)+"  "+renderToneToken(action.Tone), textWidth)))
	}
	lines = append(lines, mutedStyle.Render(singleLine(homeWatchLine(live, diagnostics, update), textWidth)))
	lines = append(lines, mutedStyle.Render(singleLine(homeStateSummaryLine(lastExecution, cfg, diagnostics), textWidth)))
	lines = append(lines, mutedStyle.Render(singleLine("Next     "+homeNextLine(actions, cursor, live, lastExecution, diagnostics, update), textWidth)))
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
	// ASCII-friendly progress bar
	bar := strings.Repeat("#", filled) + strings.Repeat(".", barWidth-filled)

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
