package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/batu3384/sift/internal/domain"
)

// validateProtectedPath validates a path before adding to protected list
func validateProtectedPath(path string) (string, bool) {
	// Trim whitespace
	path = strings.TrimSpace(path)
	if path == "" {
		return "Path cannot be empty.", false
	}

	// Expand home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "Cannot determine home directory.", false
		}
		path = filepath.Join(home, path[2:])
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "Invalid path format.", false
	}

	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		// Allow non-existent paths but warn
		return absPath, true
	}

	return absPath, true
}

type protectModel struct {
	paths          []string
	activeFamilies []string
	commandScopes  []domain.ProtectionScope
	cursor         int
	width          int
	height         int
	input          textinput.Model
	inputActive    bool
	message        string
	messageTicks   int
	explanation    *domain.ProtectionExplanation
}

func newProtectModel(paths []string) protectModel {
	input := textinput.New()
	input.Prompt = "> "
	input.Placeholder = "~/Projects/keep-me"
	input.CharLimit = 512
	input.Width = 48
	return protectModel{paths: append([]string{}, paths...), input: input}
}

func (m *protectModel) syncPaths(paths []string) {
	m.paths = append([]string{}, paths...)
	if m.cursor >= len(m.paths) {
		if len(m.paths) == 0 {
			m.cursor = 0
		} else {
			m.cursor = len(m.paths) - 1
		}
	}
}

func (m *protectModel) syncFamilies(families []string) {
	m.activeFamilies = append([]string{}, families...)
	sort.Strings(m.activeFamilies)
}

func (m *protectModel) syncScopes(scopes map[string][]string) {
	m.commandScopes = m.commandScopes[:0]
	for command, paths := range scopes {
		if len(paths) == 0 {
			continue
		}
		m.commandScopes = append(m.commandScopes, domain.ProtectionScope{
			Command: command,
			Paths:   append([]string{}, paths...),
		})
	}
	sort.Slice(m.commandScopes, func(i, j int) bool {
		return m.commandScopes[i].Command < m.commandScopes[j].Command
	})
}

func (m *protectModel) setMessage(message string, ticks int) {
	m.message = message
	m.messageTicks = ticks
}

func (m *protectModel) startInput() {
	m.inputActive = true
	m.input.SetValue("")
	m.input.Focus()
	m.setMessage("Type a path and press enter.", 0)
}

func (m *protectModel) cancelInput() {
	m.inputActive = false
	m.input.Blur()
	m.setMessage("Add cancelled.", routeMessageTicks)
}

func (m protectModel) selectedPath() string {
	if m.cursor < 0 || m.cursor >= len(m.paths) {
		return ""
	}
	return m.paths[m.cursor]
}

func (m protectModel) View() string {
	width, height := effectiveSize(m.width, m.height)
	compact := width < 118 || height < 28
	leftWidth := 54
	if width < 124 {
		leftWidth = 46
	}
	rightWidth := width - leftWidth - 10
	if rightWidth < 30 {
		rightWidth = 30
	}
	panelLines := bodyLineBudget(height, 15, 7)
	lines := []string{}
	if m.message != "" {
		lines = append(lines, mutedStyle.Render(m.message), "")
	}
	lines = append(lines, railStyle.Render("USER PROTECTED PATHS"))
	lines[len(lines)-1] = railStyle.Render("USER RAILS")
	focusLine := len(lines)
	if len(m.paths) == 0 {
		lines = append(lines, mutedStyle.Render("No user-defined protected paths yet."))
	} else {
		for idx, path := range m.paths {
			line := selectionPrefix(idx == m.cursor) + truncateText(path, width-10)
			line = singleLine(line, leftWidth-4)
			if idx == m.cursor {
				line = selectedLine.Render(line)
				focusLine = len(lines)
			}
			lines = append(lines, line)
		}
	}
	if len(m.activeFamilies) > 0 {
		lines = append(lines, renderSectionRule(leftWidth-4), railStyle.Render("FAMILY WATCH"))
		for _, family := range m.activeFamilies {
			lines = append(lines, mutedStyle.Render("· "+family))
		}
	}
	if len(m.commandScopes) > 0 {
		lines = append(lines, renderSectionRule(leftWidth-4), railStyle.Render("SCOPE WATCH"))
		for _, scope := range m.commandScopes {
			lines = append(lines, mutedStyle.Render(fmt.Sprintf("· %s  %d %s", scope.Command, len(scope.Paths), pl(len(scope.Paths), "path", "paths"))))
		}
	}
	lines = viewportLines(lines, focusLine, panelLines)
	leftBody := strings.Join(lines, "\n")
	rightBody := m.detailView(rightWidth-4, panelLines)
	if m.inputActive {
		rightBody = strings.Join([]string{
			rightBody,
			"",
			railStyle.Render("ADD GUARD"),
			m.input.View(),
			mutedStyle.Render("enter save  •  esc cancel"),
		}, "\n")
	}
	var body string
	if compact {
		body = strings.Join([]string{
			renderPanel("GUARD RAIL", fmt.Sprintf("%d user %s", len(m.paths), pl(len(m.paths), "rule", "rules")), leftBody, width-4, true),
			renderPanel("POLICY DECK", protectSubtitle(m), rightBody, width-4, false),
		}, "\n")
	} else {
		body = joinPanels(
			renderPanel("GUARD RAIL", fmt.Sprintf("%d user %s", len(m.paths), pl(len(m.paths), "rule", "rules")), leftBody, leftWidth, true),
			renderPanel("POLICY DECK", protectSubtitle(m), rightBody, rightWidth, false),
			width-4,
		)
	}
	return renderChrome("SIFT / Guardrails", "preserve never-delete paths and policy scope", protectStats(m, width), body, nil, width, false, m.height)
}

func (m protectModel) detailView(width int, maxLines int) string {
	path := m.selectedPath()
	if path == "" {
		if m.inputActive {
			return mutedStyle.Render("Enter a guard path.")
		}
		lines := []string{mutedStyle.Render("Select a guard path or press a to add one.")}
		if matrix := protectFamilyMatrixLines(m.activeFamilies, width); len(matrix) > 0 {
			lines = append(lines, renderSectionRule(width), headerStyle.Render("Family Watch"))
			lines = append(lines, matrix...)
		}
		if scopeLines := protectScopeMatrixLines(m.commandScopes, width); len(scopeLines) > 0 {
			lines = append(lines, renderSectionRule(width), headerStyle.Render("Scope Watch"))
			lines = append(lines, scopeLines...)
		}
		lines = viewportLines(lines, 0, maxLines)
		return strings.Join(lines, "\n")
	}
	lines := []string{
		mutedStyle.Render("Guard     ") + wrapText(path, width),
	}
	if m.explanation == nil {
		lines = append(lines, mutedStyle.Render("Press e to trace the active guard decision."))
		if len(m.commandScopes) > 0 {
			lines = append(lines, mutedStyle.Render("Use `sift protect scope list` to review command-scoped exclusions."))
		}
		if matrix := protectFamilyMatrixLines(m.activeFamilies, width); len(matrix) > 0 {
			lines = append(lines, renderSectionRule(width), headerStyle.Render("Family Watch"))
			lines = append(lines, matrix...)
		}
		if scopeLines := protectScopeMatrixLines(m.commandScopes, width); len(scopeLines) > 0 {
			lines = append(lines, renderSectionRule(width), headerStyle.Render("Scope Watch"))
			lines = append(lines, scopeLines...)
		}
		lines = viewportLines(lines, 0, maxLines)
		return strings.Join(lines, "\n")
	}
	lines = append(lines,
		mutedStyle.Render("Gate      ")+headerStyle.Render(strings.ReplaceAll(string(m.explanation.State), "_", " ")),
		mutedStyle.Render("Watch     ")+wrapText(m.explanation.Message, width),
	)
	lines = append(lines, renderSectionRule(width), headerStyle.Render("Decision Trace"))
	lines = append(lines, protectDecisionPathLines(*m.explanation, width)...)
	if len(m.explanation.UserMatches) > 0 {
		lines = append(lines, mutedStyle.Render("User      ")+wrapText(strings.Join(m.explanation.UserMatches, ", "), width))
	}
	if len(m.explanation.SystemMatches) > 0 {
		lines = append(lines, mutedStyle.Render("Built-in  ")+wrapText(strings.Join(m.explanation.SystemMatches, ", "), width))
	}
	if len(m.explanation.FamilyMatches) > 0 {
		lines = append(lines, mutedStyle.Render("Family    ")+wrapText(strings.Join(m.explanation.FamilyMatches, ", "), width))
	}
	if len(m.explanation.ExceptionMatches) > 0 {
		lines = append(lines, mutedStyle.Render("Exception ")+wrapText(strings.Join(m.explanation.ExceptionMatches, ", "), width))
	}
	if matrix := protectFamilyMatrixLines(m.activeFamilies, width); len(matrix) > 0 {
		lines = append(lines, renderSectionRule(width), headerStyle.Render("Family Watch"))
		lines = append(lines, matrix...)
	}
	if scopeLines := protectScopeMatrixLines(m.commandScopes, width); len(scopeLines) > 0 {
		lines = append(lines, renderSectionRule(width), headerStyle.Render("Scope Watch"))
		lines = append(lines, scopeLines...)
	}
	lines = viewportLines(lines, 0, maxLines)
	return strings.Join(lines, "\n")
}

func protectSubtitle(m protectModel) string {
	if m.inputActive {
		return "add guard path"
	}
	if m.explanation != nil && m.explanation.State != "" {
		return "decision trace"
	}
	if m.selectedPath() != "" {
		return "selected guard"
	}
	return "manage never-delete guardrails"
}

func protectStats(m protectModel, width int) []string {
	cardWidth := 24
	if width < 110 {
		cardWidth = width - 8
	}
	mode := "review"
	tone := "review"
	if m.inputActive {
		mode = "adding"
		tone = "safe"
	} else if m.explanation != nil {
		mode = "trace"
		tone = "safe"
	}
	return []string{
		renderRouteStatCard("protect", "guard", fmt.Sprintf("%d %s", len(m.paths), pl(len(m.paths), "path", "paths")), "safe", cardWidth),
		renderRouteStatCard("protect", "families", fmt.Sprintf("%d active", len(m.activeFamilies)), "review", cardWidth),
		renderRouteStatCard("protect", "scopes", fmt.Sprintf("%d %s", len(m.commandScopes), pl(len(m.commandScopes), "command", "commands")), "review", cardWidth),
		renderRouteStatCard("protect", "gate", mode, tone, cardWidth),
	}
}

func protectFamilyMatrixLines(families []string, width int) []string {
	if len(families) == 0 {
		return []string{mutedStyle.Render("No protected families are active.")}
	}
	lines := make([]string, 0, len(families))
	for _, family := range families {
		lines = append(lines, wrapText(safeStyle.Render("✓ "+family), width))
	}
	return lines
}

func protectScopeMatrixLines(scopes []domain.ProtectionScope, width int) []string {
	if len(scopes) == 0 {
		return []string{mutedStyle.Render("No command-scoped exclusions configured.")}
	}
	lines := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		preview := ""
		if len(scope.Paths) > 0 {
			preview = truncateText(scope.Paths[0], max(width-24, 18))
			if len(scope.Paths) > 1 {
				preview += fmt.Sprintf("  •  +%d more", len(scope.Paths)-1)
			}
		}
		lines = append(lines, wrapText(mutedStyle.Render(fmt.Sprintf("• %-12s %2d %-5s  %s", scope.Command, len(scope.Paths), pl(len(scope.Paths), "path", "paths"), preview)), width))
	}
	return lines
}

func protectDecisionPathLines(explanation domain.ProtectionExplanation, width int) []string {
	lines := []string{
		wrapText(mutedStyle.Render("Guard normalized -> command scopes -> user paths -> system roots -> protected families -> safe exceptions"), width),
	}
	if explanation.Command != "" {
		lines = append(lines, wrapText(mutedStyle.Render("Command  "+explanation.Command), width))
	}
	lines = append(lines, wrapText(mutedStyle.Render("Gate     "+strings.ReplaceAll(string(explanation.State), "_", " ")), width))
	return lines
}
