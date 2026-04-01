package tui

import (
	"fmt"
	"strings"

	"github.com/batuhanyuksel/sift/internal/config"
	"github.com/batuhanyuksel/sift/internal/domain"
)

type menuPreviewState struct {
	key     string
	plan    domain.ExecutionPlan
	loaded  bool
	loading bool
	err     string
}

type menuModel struct {
	title    string
	subtitle string
	actions  []homeAction
	cursor   int
	width    int
	height   int
	hint     string
	preview  menuPreviewState
}

func (m menuModel) View() string {
	width, height := effectiveSize(m.width, m.height)
	compact := width < 118 || height < 28
	leftWidth := 50
	if width < 124 {
		leftWidth = 44
	}
	rightWidth := width - leftWidth - 10
	if rightWidth < 30 {
		rightWidth = 30
	}
	panelLines := bodyLineBudget(height, 14, 7)
	leftBody := []string{}
	if m.hint != "" {
		leftBody = append(leftBody, mutedStyle.Render(m.hint), "")
	}
	leftBody = append(leftBody, homeMenuView(m.actions, m.cursor, leftWidth-4, panelLines))
	var body string
	if compact {
		body = strings.Join([]string{
			renderPanel("MENU", fmt.Sprintf("%d %s", len(m.actions), pl(len(m.actions), "choice", "choices")), strings.Join(leftBody, "\n"), width-4, true),
			renderPanel("DETAIL", menuDetailSubtitle(m.actions, m.cursor), menuDetailView(m, width-8, max(panelLines/2, 5)), width-4, false),
		}, "\n")
	} else {
		body = joinPanels(
			renderPanel("MENU", fmt.Sprintf("%d %s", len(m.actions), pl(len(m.actions), "choice", "choices")), strings.Join(leftBody, "\n"), leftWidth, true),
			renderPanel("DETAIL", menuDetailSubtitle(m.actions, m.cursor), menuDetailView(m, rightWidth-4, panelLines), rightWidth, false),
			width-4,
		)
	}
	stats := menuStats(m.actions, m.cursor, width)
	return renderChrome(
		"SIFT / "+m.title,
		m.subtitle,
		stats,
		body,
		nil,
		width,
		false,
		m.height,
	)
}

func menuDetailSubtitle(actions []homeAction, cursor int) string {
	if cursor < 0 || cursor >= len(actions) {
		return ""
	}
	action := actions[cursor]
	if action.Command != "" {
		return action.Command
	}
	return action.Title
}

func menuDetailView(m menuModel, width int, maxLines int) string {
	actions := m.actions
	cursor := m.cursor
	if cursor < 0 || cursor >= len(actions) {
		return mutedStyle.Render("No option selected.")
	}
	action := actions[cursor]
	if specialized, ok := menuSpecializedDetailView(actions, cursor, width, maxLines); ok {
		return specialized
	}
	lines := []string{
		renderToneBadge(action.Tone) + " " + headerStyle.Render(action.Title),
	}
	if width < 40 || maxLines < 8 {
		if state := menuStateLine(actions, cursor); state != "" {
			lines = append(lines, mutedStyle.Render("State   ")+truncateText(state, width))
		}
		if next := menuNextActionLine(action); next != "" {
			lines = append(lines, mutedStyle.Render("Next    ")+truncateText(next, width))
		}
		if preview := menuPreviewSummaryLines(action, m.preview); len(preview) > 0 {
			lines = append(lines, preview...)
		}
		if action.Description != "" {
			lines = append(lines, wrapText(truncateText(action.Description, width), width))
		}
		if len(action.Modules) > 0 {
			lines = append(lines, mutedStyle.Render(fmt.Sprintf("Modules %d", len(action.Modules))))
		}
		if action.Command != "" {
			lines = append(lines, mutedStyle.Render("Action  ")+headerStyle.Render(action.Command))
		}
		lines = viewportLines(lines, 0, maxLines)
		return strings.Join(lines, "\n")
	}
	if state := menuStateLine(actions, cursor); state != "" {
		lines = append(lines, mutedStyle.Render("State   ")+wrapText(state, width))
	}
	if next := menuNextActionLine(action); next != "" {
		lines = append(lines, mutedStyle.Render("Next    ")+wrapText(next, width))
	}
	if preview := menuPreviewSummaryLines(action, m.preview); len(preview) > 0 {
		lines = append(lines, preview...)
	}
	if action.Description != "" {
		lines = append(lines, wrapText(truncateText(action.Description, width), width))
	}
	if len(action.Modules) > 0 {
		lines = append(lines, mutedStyle.Render("Scope   ")+wrapText(strings.Join(action.Modules, "  •  "), width))
	}
	if action.Safety != "" {
		lines = append(lines, mutedStyle.Render("Guard   ")+wrapText(truncateText(action.Safety, width), width))
	}
	if !action.Enabled {
		lines = append(lines, highStyle.Render("Setup   ")+"not ready in this session")
	}
	lines = viewportLines(lines, 0, maxLines)
	return strings.Join(lines, "\n")
}

func (m *menuModel) setPreviewLoading(key string) {
	m.preview = menuPreviewState{key: key, loading: strings.TrimSpace(key) != ""}
}

func (m *menuModel) applyPreview(key string, plan domain.ExecutionPlan, err error) {
	preview := menuPreviewState{key: key}
	if err != nil {
		preview.err = err.Error()
		m.preview = preview
		return
	}
	preview.plan = plan
	preview.loaded = true
	m.preview = preview
}

func (m menuModel) previewPlanForSelected() (domain.ExecutionPlan, bool) {
	if m.cursor < 0 || m.cursor >= len(m.actions) {
		return domain.ExecutionPlan{}, false
	}
	action := m.actions[m.cursor]
	if action.ProfileKey == "" || strings.TrimSpace(action.ProfileKey) != strings.TrimSpace(m.preview.key) || !m.preview.loaded {
		return domain.ExecutionPlan{}, false
	}
	return m.preview.plan, true
}

func menuPreviewSummaryLines(action homeAction, preview menuPreviewState) []string {
	if action.ProfileKey == "" || strings.TrimSpace(action.ProfileKey) != strings.TrimSpace(preview.key) {
		return nil
	}
	switch {
	case preview.loading:
		return []string{mutedStyle.Render("Plan    loading preview")}
	case preview.err != "":
		return []string{highStyle.Render("Plan    preview unavailable")}
	case preview.loaded:
		lines := []string{
			mutedStyle.Render(func() string {
			mods := planModuleCount(preview.plan)
			return fmt.Sprintf("Plan    %d ready  •  %d %s  •  %s", actionableCount(preview.plan), mods, pl(mods, "module", "modules"), domain.HumanBytes(planDisplayBytes(preview.plan)))
		}()),
		}
		mix := []string{}
		if preview.plan.Totals.SafeBytes > 0 {
			mix = append(mix, domain.HumanBytes(preview.plan.Totals.SafeBytes)+" safe")
		}
		if preview.plan.Totals.ReviewBytes > 0 {
			mix = append(mix, domain.HumanBytes(preview.plan.Totals.ReviewBytes)+" review")
		}
		if preview.plan.Totals.HighBytes > 0 {
			mix = append(mix, domain.HumanBytes(preview.plan.Totals.HighBytes)+" high")
		}
		if len(mix) > 0 {
			lines = append(lines, mutedStyle.Render("Mix     "+strings.Join(mix, "  •  ")))
		}
		if len(preview.plan.Warnings) > 0 {
			lines = append(lines, mutedStyle.Render("Note    "+truncateText(preview.plan.Warnings[0], 72)))
		}
		return lines
	default:
		return nil
	}
}

func menuStateLine(actions []homeAction, cursor int) string {
	if cursor < 0 || cursor >= len(actions) {
		return ""
	}
	return menuStateText(actions[cursor], cursor, len(actions))
}

func menuStateText(action homeAction, cursor int, total int) string {
	if total <= 0 {
		total = 1
	}
	state := fmt.Sprintf("%d/%d selected", cursor+1, total)
	if action.Enabled {
		return state + "  •  ready"
	}
	return state + "  •  setup required"
}

func menuNextActionLine(action homeAction) string {
	if !action.Enabled {
		return "finish setup first"
	}
	switch action.ID {
	case "clean_quick", "clean_developer", "clean_deep":
		return "enter opens review"
	case "check", "doctor":
		return "enter opens checks"
	case "autofix":
		return "enter opens fixes"
	case "optimize":
		return "enter opens review"
	case "installer":
		return "enter opens review"
	case "protect":
		return "enter opens editor"
	case "purge_scan":
		return "enter opens review"
	default:
		if action.ProfileKey != "" {
			return "enter opens review"
		}
		return "enter opens item"
	}
}

func menuStats(actions []homeAction, cursor int, width int) []string {
	cardWidth := 24
	if width < 110 {
		cardWidth = width - 8
	}
	enabled := 0
	tone := "review"
	selected := "none"
	for _, action := range actions {
		if action.Enabled {
			enabled++
		}
	}
	if cursor >= 0 && cursor < len(actions) {
		selected = actions[cursor].Title
		tone = actions[cursor].Tone
	}
	stats := []string{
		renderStatCard("options", fmt.Sprintf("%d enabled", enabled), "safe", cardWidth),
		renderStatCard("selected", selected, tone, cardWidth+6),
	}
	if cursor >= 0 && cursor < len(actions) && len(actions[cursor].Modules) > 0 {
		stats = append(stats, renderStatCard("modules", fmt.Sprintf("%d covered", len(actions[cursor].Modules)), tone, cardWidth))
	}
	return stats
}

func buildCleanActions() []homeAction {
	return []homeAction{
		{
			ID:          "clean_quick",
			Title:       "Quick Clean",
			Description: "Daily temp files, logs, and low-risk clutter.",
			Command:     "review quick clean",
			Safety:      "Lowest-risk reclaim lane.",
			When:        "Use for routine maintenance and fast reclaim.",
			Tone:        "safe",
			ProfileKey:  "safe",
			Modules:     []string{"Temporary files", "Application logs", "Safe clutter"},
			Enabled:     true,
		},
		{
			ID:          "clean_developer",
			Title:       "Workstation Clean",
			Description: "Developer, browser, package, and app cache reclaim.",
			Command:     "review workstation clean",
			Safety:      "Broader than quick clean, still routine.",
			When:        "Use after build churn and heavier daily use.",
			Tone:        "review",
			ProfileKey:  "developer",
			Modules:     []string{"Developer caches", "Package caches", "Browser caches"},
			Enabled:     true,
		},
		{
			ID:          "clean_deep",
			Title:       "Deep Reclaim",
			Description: "System leftovers, installers, caches, and app remnants.",
			Command:     "review deep reclaim",
			Safety:      "Widest reclaim surface before review.",
			When:        "Use when reclaiming maximum space across the machine.",
			Tone:        "high",
			ProfileKey:  "deep",
			Modules:     []string{"Developer caches", "Package manager caches", "Browser caches", "Installer leftovers", "App leftovers"},
			Enabled:     true,
		},
	}
}

func buildToolsActions(cfg config.Config) []homeAction {
	return []homeAction{
		{ID: "check", Title: "Check", Description: "Audit system posture and actionable drift.", Command: "sift check", Safety: "Read-only operational checks.", When: "Use before maintenance or support sessions.", Tone: "safe", Modules: []string{"Security", "Updates", "Config drift", "Health pressure"}, Enabled: true},
		{ID: "autofix", Title: "Autofix", Description: "Review safe fixes for current check findings.", Command: "sift autofix", Safety: "Fixes stay review-gated.", When: "Use after check when posture drift is actionable.", Tone: "review", Modules: []string{"Firewall", "Gatekeeper", "Touch ID", "Rosetta", "Update fixes"}, Enabled: true},
		{ID: "optimize", Title: "Optimize", Description: "Run curated maintenance cache resets.", Command: "sift optimize", Safety: "Safe maintenance tasks with review.", When: "Use before deep cleanup or support sessions.", Tone: "safe", Modules: []string{"Preflight", "Caches", "Indexes", "Network", "Repair", "Verify"}, Enabled: true},
		{ID: "installer", Title: "Installer Cleanup", Description: "Review stale installers across common download roots.", Command: "sift installer", Safety: "Installer remnants only.", When: "Use after install waves.", Tone: "safe", Modules: []string{"Downloads", "Documents", "Shared roots", "DMG/PKG", "ZIP/XIP"}, Enabled: true},
		{ID: "protect", Title: "Protect Paths", Description: "Manage never-delete user paths.", Command: "review protected paths", Safety: "Changes cleanup guardrails.", When: "Use when pinning folders and projects.", Tone: "review", Modules: []string{"Protected paths", "Families", "Command scopes", "Safe exceptions"}, Enabled: true},
		{ID: "purge_scan", Title: "Purge Scan", Description: "Find build artifacts across repositories.", Command: "sift purge scan", Safety: "Discovery only until review.", When: "Use for repo cleanup sweeps.", Tone: "review", Modules: []string{"Workspace roots", "Artifact clusters", "Vendor guard", "Review batch"}, Enabled: len(cfg.PurgeSearchPaths) > 0},
		{ID: "doctor", Title: "Doctor", Description: "Inspect config, audit, report, and runtime paths.", Command: "sift doctor", Safety: "Read-only diagnostics.", When: "Use when validating setup.", Tone: "safe", Enabled: true},
	}
}
