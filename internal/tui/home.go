package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/batuhanyuksel/sift/internal/config"
	"github.com/batuhanyuksel/sift/internal/engine"
	"github.com/batuhanyuksel/sift/internal/platform"
	"github.com/batuhanyuksel/sift/internal/store"
)

type homeAction struct {
	ID          string
	Title       string
	Description string
	Command     string
	Safety      string
	When        string
	Tone        string
	ProfileKey  string
	Modules     []string
	Enabled     bool
}

type homeModel struct {
	actions       []homeAction
	cursor        int
	selected      string
	cancelled     bool
	live          *engine.SystemSnapshot
	lastExecution *store.ExecutionSummary
	diagnostics   []platform.Diagnostic
	updateNotice  *engine.UpdateNotice
	cfg           config.Config
	executable    bool
	width         int
	height        int
	loading       bool
	loadingLabel  string
	pulse         bool
	signalFrame   int
	reducedMotion bool
}

func buildHomeActions(_ config.Config) []homeAction {
	return []homeAction{
		{ID: "clean", Title: "🧹 Clean", Description: "Remove unnecessary files to free up disk space", Command: "Clean", Safety: "Safe", When: "To reclaim disk space", Tone: "safe", Enabled: true},
		{ID: "uninstall", Title: "🗑️ Uninstall", Description: "Remove applications and clean leftovers", Command: "Uninstall", Safety: "Requires confirmation", When: "To fully remove an application", Tone: "review", Enabled: true},
		{ID: "analyze", Title: "📊 Analyze", Description: "Analyze disk usage and find large files", Command: "Analyze", Safety: "Read-only", When: "To see what's taking up space", Tone: "review", Enabled: true},
		{ID: "status", Title: "💻 Status", Description: "Check system health and diagnostics", Command: "Status", Safety: "Read-only", When: "To check system health", Tone: "safe", Enabled: true},
		{ID: "optimize", Title: "⚡ Optimize", Description: "Optimize system performance", Command: "Optimize", Safety: "Safe", When: "To improve system performance", Tone: "safe", Enabled: true},
	}
}

func (m homeModel) Init() tea.Cmd { return nil }

func (m homeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.actions)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			action := m.actions[m.cursor]
			if !action.Enabled {
				return m, nil
			}
			m.selected = action.ID
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m homeModel) View() string {
	width, height := effectiveSize(m.width, m.height)
	compact := width < 118 || height < 28
	leftWidth := 50
	if width < 132 {
		leftWidth = 44
	}
	rightWidth := width - leftWidth - 10
	if rightWidth < 30 {
		rightWidth = 30
	}
	panelLines := bodyLineBudget(height, 19, 5)
	spotlightLines := 5
	if compact {
		spotlightLines = 5
		if height >= 26 || width >= 96 {
			spotlightLines = 6
		}
	} else {
		spotlightLines = 9
	}
	overview := renderPanel("HOME", homeSpotlightSubtitle(m.actions, m.cursor, m.live, m.lastExecution, m.diagnostics, m.updateNotice), homeSpotlightView(m.actions, m.cursor, m.live, m.lastExecution, m.diagnostics, m.updateNotice, m.cfg, homeMotionState(m), width-8, spotlightLines), width-4, false)
	var panels string
	if compact {
		detailLines := max(panelLines/2, 5)
		panels = strings.Join([]string{
			renderPanel("MENU", fmt.Sprintf("%d %s", len(m.actions), pl(len(m.actions), "action", "actions")), homeMenuView(m.actions, m.cursor, width-8, panelLines), width-4, true),
			renderPanel("DETAIL", homeDetailSubtitle(m.actions, m.cursor), homeDetailCompactView(m.actions, m.cursor, m.live, m.lastExecution, m.diagnostics, m.cfg, width-8, detailLines), width-4, false),
		}, "\n")
	} else {
		panels = joinPanels(
			renderPanel("MENU", fmt.Sprintf("%d %s", len(m.actions), pl(len(m.actions), "action", "actions")), homeMenuView(m.actions, m.cursor, leftWidth-4, panelLines), leftWidth, true),
			renderPanel("DETAIL", homeDetailSubtitle(m.actions, m.cursor), homeDetailView(m.actions, m.cursor, m.live, m.lastExecution, m.diagnostics, m.cfg, rightWidth-4, panelLines), rightWidth, false),
			width-4,
		)
	}
	return renderChrome(
		"SIFT / Home",
		homeSubtitle(m.executable, m.cfg),
		homeStats(m.live, m.lastExecution, m.diagnostics, m.updateNotice, width),
		strings.Join([]string{overview, panels}, "\n"),
		nil,
		width,
		true,
		m.height,
	)
}
