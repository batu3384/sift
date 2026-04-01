package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/platform"
)

type permissionPreflightModel struct {
	plan         domain.ExecutionPlan
	focusPath    string
	needsAdmin   bool
	needsDialogs bool
	needsNative  bool
	adminItems   int
	dialogItems  int
	nativeItems  int
	adminLabels  []string
	dialogLabels []string
	nativeLabels []string
	width        int
	height       int
}

func buildPermissionPreflight(plan domain.ExecutionPlan, focusPath string) permissionPreflightModel {
	model := permissionPreflightModel{
		plan:      plan,
		focusPath: strings.TrimSpace(focusPath),
	}
	for _, item := range plan.Items {
		if item.Action == domain.ActionAdvisory || item.Action == domain.ActionSkip || item.Status == domain.StatusProtected || item.Status == domain.StatusSkipped {
			continue
		}
		if item.RequiresAdmin || item.CommandPath == "/usr/bin/sudo" {
			model.needsAdmin = true
			model.adminItems++
			model.adminLabels = appendManifestLabel(model.adminLabels, preflightItemLabel(item))
		}
		if item.CommandPath == "/usr/bin/osascript" {
			model.needsDialogs = true
			model.dialogItems++
			model.dialogLabels = appendManifestLabel(model.dialogLabels, preflightItemLabel(item))
		}
		if item.Action == domain.ActionNative {
			model.needsNative = true
			model.nativeItems++
			model.nativeLabels = appendManifestLabel(model.nativeLabels, preflightItemLabel(item))
		}
	}
	return model
}

func (m permissionPreflightModel) required() bool {
	return m.needsAdmin || m.needsDialogs || m.needsNative
}

func (m permissionPreflightModel) scopeLabel() string {
	if scope := strings.TrimSpace(planReviewScopeLine(m.plan)); scope != "" {
		return scope
	}
	label := strings.TrimSpace(titleCase(m.plan.Command))
	if label == "" {
		label = "Run"
	}
	if count := preflightActionableCount(m.plan.Items); count > 0 {
		return fmt.Sprintf("%s • %d %s", label, count, pl(count, "item", "items"))
	}
	return label
}

func preflightActionableCount(items []domain.Finding) int {
	count := 0
	for _, item := range items {
		if item.Action == domain.ActionAdvisory || item.Action == domain.ActionSkip || item.Status == domain.StatusProtected || item.Status == domain.StatusSkipped {
			continue
		}
		count++
	}
	return count
}

func (m permissionPreflightModel) stats(width int) []string {
	stats := []string{renderStatCard("scope", m.scopeLabel(), "safe", 30)}
	if m.needsAdmin {
		stats = append(stats, renderStatCard("admin", formatPreflightCount(m.adminItems, "step"), "review", 24))
	}
	if m.needsDialogs {
		stats = append(stats, renderStatCard("dialogs", formatPreflightCount(m.dialogItems, "prompt"), "review", 24))
	}
	if m.needsNative {
		stats = append(stats, renderStatCard("native", formatPreflightCount(m.nativeItems, "handoff"), "review", 24))
	}
	return trimStatsForHeight(stats, m.height, false)
}

func formatPreflightCount(count int, noun string) string {
	if count <= 0 {
		count = 1
	}
	return fmt.Sprintf("%d %s", count, pl(count, noun, noun+"s"))
}

func appendManifestLabel(labels []string, label string) []string {
	label = strings.TrimSpace(label)
	if label == "" {
		return labels
	}
	for _, existing := range labels {
		if existing == label {
			return labels
		}
	}
	return append(labels, label)
}

func preflightItemLabel(item domain.Finding) string {
	if label := strings.TrimSpace(item.Name); label != "" && label != strings.TrimSpace(item.DisplayPath) {
		return label
	}
	if label := strings.TrimSpace(displayFindingLabel(item)); label != "" && label != "item" {
		return label
	}
	if path := strings.TrimSpace(item.CommandPath); path != "" {
		return path
	}
	if item.Action == domain.ActionNative {
		return "native handoff"
	}
	return "run item"
}

func (m permissionPreflightModel) accessLines(width int) string {
	lines := make([]string, 0, 10)
	if m.needsAdmin {
		lines = append(lines,
			wrapText("Admin   Touch ID, your sudo password, or a macOS password dialog may be requested before run.", width),
			wrapText(fmt.Sprintf("Why     %s need elevated access and an active admin session.", formatPreflightCount(m.adminItems, "step")), width),
		)
	}
	if m.needsDialogs {
		if len(lines) > 0 {
			lines = append(lines, renderSectionRule(width))
		}
		lines = append(lines,
			wrapText("Dialogs macOS may show one or more system prompts while this run continues.", width),
			wrapText(fmt.Sprintf("Why     %s may open while execution continues.", formatPreflightCount(m.dialogItems, "prompt")), width),
		)
	}
	if m.needsNative {
		if len(lines) > 0 {
			lines = append(lines, renderSectionRule(width))
		}
		lines = append(lines,
			wrapText("Native  An uninstaller or external app may open outside SIFT.", width),
			wrapText(fmt.Sprintf("Why     %s leave SIFT and return when the external app hands back control.", formatPreflightCount(m.nativeItems, "handoff")), width),
		)
	}
	if len(lines) == 0 {
		lines = append(lines, mutedStyle.Render("No extra access is needed for this run."))
	}
	return strings.Join(lines, "\n")
}

func (m permissionPreflightModel) flowLines(width int) string {
	lines := []string{
		"Run     " + m.scopeLabel(),
		"Keys    y run • esc back",
	}
	if m.focusPath != "" {
		lines = append(lines, wrapText("Focus   "+m.focusPath, width))
	}
	if m.needsAdmin {
		lines = append(lines, wrapText("Step 1   warm admin access", width))
		lines = append(lines, wrapText("Step 2   keep access alive while execution runs", width))
	}
	if m.needsDialogs {
		lines = append(lines, wrapText("Step     approve system prompts if macOS asks", width))
	}
	if m.needsNative {
		lines = append(lines, wrapText("Step     native app opens outside SIFT and returns to result tracking", width))
	}
	if !m.needsAdmin && !m.needsDialogs && !m.needsNative {
		lines = append(lines, wrapText("State   run starts immediately with no extra prompts.", width))
	}
	return strings.Join(lines, "\n")
}

func (m permissionPreflightModel) manifestLines(width int) string {
	lines := make([]string, 0, 16)
	appendSection := func(title string, labels []string) {
		if len(labels) == 0 {
			return
		}
		if len(lines) > 0 {
			lines = append(lines, renderSectionRule(width))
		}
		lines = append(lines, headerStyle.Render(title))
		for _, label := range labels[:min(len(labels), 4)] {
			lines = append(lines, wrapText(mutedStyle.Render("• "+label), width))
		}
		if len(labels) > 4 {
			lines = append(lines, wrapText(mutedStyle.Render(fmt.Sprintf("• %d more", len(labels)-4)), width))
		}
	}
	appendSection("ADMIN", m.adminLabels)
	appendSection("DIALOGS", m.dialogLabels)
	appendSection("NATIVE", m.nativeLabels)
	if len(lines) == 0 {
		lines = append(lines, mutedStyle.Render("No permission manifest items."))
	}
	return strings.Join(lines, "\n")
}

func (m permissionPreflightModel) trackLine() string {
	parts := []string{"Track"}
	if m.needsAdmin {
		parts = append(parts, fmt.Sprintf("%d admin", m.adminItems))
	}
	if m.needsDialogs {
		parts = append(parts, fmt.Sprintf("%d %s", m.dialogItems, pl(m.dialogItems, "dialog", "dialogs")))
	}
	if m.needsNative {
		parts = append(parts, fmt.Sprintf("%d native", m.nativeItems))
	}
	if len(parts) == 1 {
		parts = append(parts, "no extra access")
	}
	return strings.Join(parts, "  •  ")
}

func (m permissionPreflightModel) accessSummaryLine() string {
	parts := []string{"Access"}
	if m.needsAdmin {
		parts = append(parts, fmt.Sprintf("%d admin", m.adminItems))
	}
	if m.needsDialogs {
		parts = append(parts, fmt.Sprintf("%d %s", m.dialogItems, pl(m.dialogItems, "dialog", "dialogs")))
	}
	if m.needsNative {
		parts = append(parts, fmt.Sprintf("%d native", m.nativeItems))
	}
	if len(parts) == 1 {
		parts = append(parts, "none")
	}
	return strings.Join(parts, "  •  ")
}

func (m permissionPreflightModel) manifestSummaryLine(width int) string {
	labels := make([]string, 0, len(m.adminLabels)+len(m.dialogLabels)+len(m.nativeLabels))
	labels = append(labels, m.adminLabels...)
	labels = append(labels, m.dialogLabels...)
	labels = append(labels, m.nativeLabels...)
	if len(labels) == 0 {
		return ""
	}
	if len(labels) > 3 {
		labels = append(labels[:3], fmt.Sprintf("%d more", len(labels)-3))
	}
	return wrapText("Need    "+strings.Join(labels, "  •  "), width)
}

func (m permissionPreflightModel) profileSignature() string {
	if !m.required() {
		return ""
	}
	parts := []string{
		fmt.Sprintf("admin:%t", m.needsAdmin),
		fmt.Sprintf("dialogs:%t", m.needsDialogs),
		fmt.Sprintf("native:%t", m.needsNative),
	}
	appendLabels := func(kind string, labels []string) {
		if len(labels) == 0 {
			return
		}
		cloned := append([]string(nil), labels...)
		sort.Strings(cloned)
		parts = append(parts, kind+":"+strings.Join(cloned, "|"))
	}
	appendLabels("a", m.adminLabels)
	appendLabels("d", m.dialogLabels)
	appendLabels("n", m.nativeLabels)
	return strings.Join(parts, ";")
}

func (m permissionPreflightModel) View() string {
	width, height := effectiveSize(m.width, m.height)
	contentWidth := width - 6
	leftWidth, rightWidth := splitColumns(contentWidth-2, 0.52, 40, 34)
	body := joinPanels(
		renderPanel("ACCESS", "before run", m.accessLines(leftWidth-4), leftWidth, true),
		strings.Join([]string{
			renderPanel("MANIFEST", "items that need access", m.manifestLines(rightWidth-4), rightWidth, false),
			renderPanel("FLOW", "y run • esc back", m.flowLines(rightWidth-4), rightWidth, false),
		}, "\n"),
		width-4,
	)
	return renderChrome(
		"SIFT / Permissions",
		"access before run",
		m.stats(width),
		strings.Join([]string{
			wrapText(mutedStyle.Render("Scope   "+m.scopeLabel()), width-4),
			wrapText(mutedStyle.Render(m.trackLine()), width-4),
			body,
		}, "\n"),
		nil,
		width,
		false,
		height,
	)
}

func defaultPermissionWarmupCmd(model permissionPreflightModel) tea.Cmd {
	if !model.needsAdmin {
		return func() tea.Msg { return permissionWarmupFinishedMsg{} }
	}
	return func() tea.Msg {
		return permissionWarmupFinishedMsg{
			err: platform.WarmAdminSession(context.Background(), "SIFT needs admin access to continue this run."),
		}
	}
}

func defaultPermissionKeepalive(ctx context.Context, model permissionPreflightModel) {
	if !model.needsAdmin {
		return
	}
	platform.StartAdminKeepalive(ctx)
}
