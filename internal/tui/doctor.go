package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/batu3384/sift/internal/platform"
)

type doctorModel struct {
	diagnostics  []platform.Diagnostic
	cursor       int
	width        int
	height       int
	loading      bool
	loadingLabel string
}

func (m doctorModel) Init() tea.Cmd { return nil }

func (m doctorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c", "enter":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.diagnostics)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		}
	}
	return m, nil
}

func (m doctorModel) View() string {
	width, height := effectiveSize(m.width, m.height)
	stats := []string{
		renderStatCard("doctor", doctorSummary(m.diagnostics), doctorOverallTone(m.diagnostics), 28),
		renderStatCard("checks", fmt.Sprintf("%d %s", len(m.diagnostics), pl(len(m.diagnostics), "item", "items")), "safe", 24),
	}
	if width >= 120 {
		for _, card := range doctorGroupStatCards(m.diagnostics, 22) {
			stats = append(stats, card)
		}
	}
	leftWidth := 42
	if width >= 132 {
		leftWidth = 46
	}
	rightWidth := width - leftWidth - 10
	if rightWidth < 34 {
		rightWidth = 34
	}
	panelLines := bodyLineBudget(height, 14, 7)
	body := joinPanels(
		renderPanel("Checks", "Runtime, config, report, and audit diagnostics", doctorListView(m.diagnostics, m.cursor, leftWidth-4, panelLines), leftWidth, true),
		renderPanel("Details", doctorDetailSubtitle(m.diagnostics, m.cursor), doctorDetailView(m.diagnostics, m.cursor, rightWidth-4, panelLines), rightWidth, false),
		width-4,
	)
	return renderChrome(
		"SIFT / Doctor",
		"Inspect runtime health and operational safety paths.",
		stats,
		body,
		nil,
		width,
		false,
		m.height,
	)
}

func doctorSummary(diagnostics []platform.Diagnostic) string {
	okCount := 0
	warnCount := 0
	errorCount := 0
	for _, diagnostic := range diagnostics {
		switch diagnostic.Status {
		case "ok":
			okCount++
		case "warn":
			warnCount++
		default:
			errorCount++
		}
	}
	return fmt.Sprintf("OK %d  WARN %d  ERROR %d", okCount, warnCount, errorCount)
}

// doctorOverallTone returns the card tone that best reflects the worst
// diagnostic found: "high" if any errors, "review" if any warnings, else "safe".
func doctorOverallTone(diagnostics []platform.Diagnostic) string {
	hasWarn := false
	for _, d := range diagnostics {
		switch d.Status {
		case "error":
			return "high"
		case "warn":
			hasWarn = true
		}
	}
	if hasWarn {
		return "review"
	}
	return "safe"
}

// doctorStatusIcon returns the visual icon and line style for a diagnostic
// status — consistent with the ✓/!/✗ icon set used in result and progress views.
func doctorStatusIcon(status string) (icon string, lineStyle lipgloss.Style) {
	switch status {
	case "ok":
		return "✓", safeStyle
	case "warn":
		return "!", reviewStyle
	default:
		return "✗", highStyle
	}
}

func doctorListView(diagnostics []platform.Diagnostic, cursor int, width int, maxLines int) string {
	if len(diagnostics) == 0 {
		return mutedStyle.Render("No diagnostics available.")
	}
	lines := make([]string, 0, len(diagnostics))
	for i, diagnostic := range diagnostics {
		icon, lineStyle := doctorStatusIcon(diagnostic.Status)
		status := strings.ToUpper(diagnostic.Status)
		line := selectionPrefix(i == cursor) + lineStyle.Render(fmt.Sprintf("%s %-5s %-22s %s", icon, status, diagnostic.Name, diagnostic.Message))
		if i == cursor {
			line = selectedLine.Render(singleLine(line, width))
		}
		lines = append(lines, wrapText(line, width))
	}
	lines = viewportLines(lines, cursor, maxLines)
	return strings.Join(lines, "\n")
}

func doctorDetailSubtitle(diagnostics []platform.Diagnostic, cursor int) string {
	if cursor < 0 || cursor >= len(diagnostics) {
		return ""
	}
	return strings.ToUpper(diagnostics[cursor].Status)
}

func doctorDetailView(diagnostics []platform.Diagnostic, cursor int, width int, maxLines int) string {
	if len(diagnostics) == 0 || cursor < 0 || cursor >= len(diagnostics) {
		return "No diagnostic selected."
	}
	diagnostic := diagnostics[cursor]
	statusLine := safeStyle.Render("OK")
	switch diagnostic.Status {
	case "warn":
		statusLine = reviewStyle.Render("WARN")
	case "error":
		statusLine = highStyle.Render("ERROR")
	}
	guidance := "Everything looks healthy."
	switch diagnostic.Status {
	case "warn":
		guidance = "Review this item before running destructive cleanup."
	case "error":
		guidance = "Fix this before relying on automated cleanup or reports."
	}
	lines := []string{
		headerStyle.Render(diagnostic.Name),
		"",
		statusLine,
		"",
		mutedStyle.Render("Lane   " + doctorLaneLabel(diagnostic.Name)),
	}
	lines = append(lines,
		wrapText(diagnostic.Message, width),
		"",
		mutedStyle.Render(guidance),
	)
	if fix := doctorFixHint(diagnostic.Name, diagnostic.Status); fix != "" {
		lines = append(lines, renderSectionRule(width), headerStyle.Render("Next"), mutedStyle.Render(fix))
	}
	lines = viewportLines(lines, 0, maxLines)
	return strings.Join(lines, "\n")
}

func doctorGroupStatCards(diagnostics []platform.Diagnostic, cardWidth int) []string {
	counts := map[string]int{
		"security": 0,
		"updates":  0,
		"config":   0,
		"health":   0,
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Status != "warn" && diagnostic.Status != "error" {
			continue
		}
		counts[doctorLaneLabel(diagnostic.Name)]++
	}
	cards := make([]string, 0, 4)
	for _, key := range []string{"security", "updates", "config", "health"} {
		tone := "safe"
		if counts[key] > 0 {
			tone = "review"
		}
		cards = append(cards, renderStatCard(key, fmt.Sprintf("%d active", counts[key]), tone, cardWidth))
	}
	return cards
}

func doctorLaneLabel(name string) string {
	switch name {
	case "filevault", "firewall", "gatekeeper", "sip", "touchid":
		return "security"
	case "macos_updates", "brew_updates", "sift_update":
		return "updates"
	case "rosetta2", "git_identity", "login_items", "test_policy", "live_integration", "upstream_baseline":
		return "config"
	default:
		return "health"
	}
}

func doctorFixHint(name string, status string) string {
	if status == "ok" {
		return ""
	}
	switch name {
	case "firewall", "gatekeeper", "touchid", "rosetta2":
		return "Run sift autofix to load the reviewable fix batch for this warning."
	case "macos_updates", "brew_updates", "sift_update":
		return "Use sift update or the linked package-manager flow, then rerun check."
	case "login_items":
		return "Review login items drift, then rerun check to confirm the posture is clean."
	case "git_identity":
		return "Set the missing git identity fields, then rerun check."
	default:
		return "Rerun doctor or status after remediation to confirm the warning is gone."
	}
}
