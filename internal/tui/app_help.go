package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"

	"github.com/batu3384/sift/internal/domain"
)

func (m appModel) routeBindings() routeHelp {
	toggle := relabeledBinding(m.keys.Stage, "toggle")
	queue := relabeledBinding(m.keys.Stage, "queue")
	switch m.route {
	case RouteHome:
		return routeHelp{
			short: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Tools, m.keys.Refresh, m.keys.Help, m.keys.Quit},
			sections: []helpSection{
				{title: "Move", bindings: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter}},
				{title: "Do", bindings: []key.Binding{m.keys.Tools, m.keys.Refresh}},
				{title: "Back", bindings: []key.Binding{m.keys.Help, m.keys.Quit}},
			},
		}
	case RouteClean:
		sections := []helpSection{
			{title: "Move", bindings: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter}},
		}
		if history := m.routeHistoryBindings(false); len(history) > 0 {
			sections = append(sections, helpSection{title: "History", bindings: history})
		}
		sections = append(sections, helpSection{title: "Back", bindings: []key.Binding{m.keys.Back, m.keys.Help, m.keys.Quit}})
		return routeHelp{
			short:    []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Back, m.keys.Help, m.keys.Quit},
			sections: sections,
		}
	case RouteTools:
		return routeHelp{
			short: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Back, m.keys.Help, m.keys.Quit},
			sections: []helpSection{
				{title: "Move", bindings: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter}},
				{title: "Back", bindings: []key.Binding{m.keys.Back, m.keys.Help, m.keys.Quit}},
			},
		}
	case RouteProtect:
		return routeHelp{
			short: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Add, m.keys.Delete, m.keys.Explain, m.keys.Back, m.keys.Help},
			sections: []helpSection{
				{title: "Move", bindings: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Back}},
				{title: "Do", bindings: []key.Binding{m.keys.Add, m.keys.Delete, m.keys.Explain}},
				{title: "Back", bindings: []key.Binding{m.keys.Help, m.keys.Quit}},
			},
		}
	case RouteUninstall:
		sections := []helpSection{
			{title: "Move", bindings: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Back}},
			{title: "Batch", bindings: []key.Binding{queue, m.keys.Review}},
			{title: "Find", bindings: []key.Binding{m.keys.Search, m.keys.Refresh}},
		}
		if history := m.routeHistoryBindings(false); len(history) > 0 {
			sections = append(sections, helpSection{title: "History", bindings: history})
		}
		sections = append(sections, helpSection{title: "Back", bindings: []key.Binding{m.keys.Help, m.keys.Quit}})
		return routeHelp{
			short:    []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Stage, m.keys.Review, m.keys.Help, m.keys.Back},
			sections: sections,
		}
	case RouteStatus:
		return routeHelp{
			short: []key.Binding{m.keys.Back, m.keys.Refresh, m.keys.Help, m.keys.Quit},
			sections: []helpSection{
				{title: "Move", bindings: []key.Binding{m.keys.Back}},
				{title: "Do", bindings: []key.Binding{m.keys.Refresh, m.keys.Companion}},
				{title: "Back", bindings: []key.Binding{m.keys.Help, m.keys.Quit}},
			},
		}
	case RouteDoctor:
		return routeHelp{
			short: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Back, m.keys.Refresh, m.keys.Help, m.keys.Quit},
			sections: []helpSection{
				{title: "Move", bindings: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Back}},
				{title: "Do", bindings: []key.Binding{m.keys.Refresh}},
				{title: "Back", bindings: []key.Binding{m.keys.Help, m.keys.Quit}},
			},
		}
	case RouteAnalyze:
		sections := []helpSection{
			{title: "Move", bindings: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Focus, m.keys.Enter, m.keys.Back}},
			{title: "Mark", bindings: []key.Binding{toggle, m.keys.Review}},
			{title: "Find", bindings: []key.Binding{m.keys.Search, m.keys.Filter, m.keys.Sort}},
			{title: "File", bindings: []key.Binding{m.keys.Open, m.keys.Reveal, m.keys.Delete}},
		}
		if history := m.routeHistoryBindings(false); len(history) > 0 {
			sections = append(sections, helpSection{title: "History", bindings: history})
		}
		sections = append(sections, helpSection{title: "Back", bindings: []key.Binding{m.keys.Help, m.keys.Quit}})
		return routeHelp{
			short:    []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Stage, m.keys.Review, m.keys.Help, m.keys.Back},
			sections: sections,
		}
	case RouteReview:
		sections := []helpSection{
			{title: "Move", bindings: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Module}},
			{title: "Mark", bindings: []key.Binding{toggle}},
		}
		if shouldExecutePlan(m.review.effectivePlan()) {
			sections = append(sections, helpSection{title: "Run", bindings: []key.Binding{m.keys.Execute, m.keys.Cancel}})
		}
		sections = append(sections, helpSection{title: "Back", bindings: []key.Binding{m.keys.Help, m.keys.Back, m.keys.Quit}})
		return routeHelp{short: append(m.reviewBindings(false), m.keys.Help), sections: sections}
	case RoutePreflight:
		return routeHelp{
			short: []key.Binding{m.keys.Execute, m.keys.Back, m.keys.Help},
			sections: []helpSection{
				{title: "Run", bindings: []key.Binding{m.keys.Execute}},
				{title: "Back", bindings: []key.Binding{m.keys.Back, m.keys.Help, m.keys.Quit}},
			},
		}
	case RouteProgress:
		sections := []helpSection{
			{title: "Move", bindings: []key.Binding{m.keys.Up, m.keys.Down}},
		}
		if history := m.routeHistoryBindings(false); len(history) > 0 {
			sections = append(sections, helpSection{title: "History", bindings: history})
		}
		sections = append(sections,
			helpSection{title: "Do", bindings: []key.Binding{m.keys.Stop}},
			helpSection{title: "Back", bindings: []key.Binding{m.keys.Help}},
		)
		return routeHelp{
			short:    []key.Binding{m.keys.Up, m.keys.Down, m.keys.Stop, m.keys.Help},
			sections: sections,
		}
	case RouteResult:
		sections := []helpSection{
			{title: "Move", bindings: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Filter}},
		}
		issuePaths := resultRecoveryPaths(m.result.plan, m.result.result, m.result.filter)
		failedPaths := resultRecoveryPathsForStatuses(m.result.plan, m.result.result, m.result.filter, domain.StatusFailed)
		group := resultCurrentGroupRecoveryCandidates(m.result)
		recovery := make([]key.Binding, 0, 3)
		if len(failedPaths) > 0 {
			recovery = append(recovery, m.keys.Retry)
		}
		if len(group) > 0 {
			recovery = append(recovery, m.keys.Module)
		}
		if len(issuePaths) > 0 {
			recovery = append(recovery, m.keys.Review)
		}
		if len(recovery) > 0 {
			sections = append(sections, helpSection{title: "Fix", bindings: recovery})
		}
		sections = append(sections, helpSection{title: "Back", bindings: []key.Binding{m.keys.Enter, m.keys.Back, m.keys.Help, m.keys.Quit}})
		return routeHelp{short: append(m.resultBindings(false), m.keys.Help), sections: sections}
	default:
		return routeHelp{
			short: []key.Binding{m.keys.Help, m.keys.Quit},
			sections: []helpSection{
				{title: "Back", bindings: []key.Binding{m.keys.Help, m.keys.Quit}},
			},
		}
	}
}

func (m appModel) footerContent() string {
	if m.helpVisible {
		closeHelp := relabeledBinding(m.keys.Help, "close help")
		closeBack := relabeledBinding(m.keys.Back, "close help")
		return strings.Join([]string{
			m.help.ShortHelpView([]key.Binding{closeHelp, closeBack}),
			"help for " + routeHelpLabel(m.route),
		}, "  •  ")
	}
	if m.planLoadPending() {
		return strings.Join([]string{
			m.help.ShortHelpView([]key.Binding{m.keys.Back, m.keys.Help, m.keys.Quit}),
			"loading next screen",
		}, "  •  ")
	}
	bindings := m.routeBindings().short
	compact := compactWidth(m.width) || compactHeight(m.height)
	if compact {
		bindings = m.compactRouteBindings(bindings)
	}
	parts := []string{m.help.ShortHelpView(bindings)}
	if extra := m.footerSecondaryHint(compact); extra != "" {
		parts = append(parts, extra)
	}
	if (m.route == RouteHome || m.route == RouteStatus || m.route == RouteDoctor) && m.lastDashboardSync != "" {
		parts = append(parts, "updated "+m.lastDashboardSync)
	}
	if m.route == RouteHome || m.route == RouteStatus {
		var motion motionState
		if m.route == RouteHome {
			motion = homeMotionState(m.home)
		} else {
			motion = statusMotionState(m.status)
		}
		parts = append(parts, footerMotionLabel(motion))
	}
	return strings.Join(parts, "  •  ")
}

func (m appModel) helpOverlayView() string {
	width, height := effectiveSize(m.width, m.height)
	panelLines := bodyLineBudget(height, 16, 8)
	body := renderHelpSections(m.routeBindings().sections, width-8, panelLines)
	subtitle := routeHelpSubtitle(m.route, m.keys.Help.Help().Key, m.keys.Back.Help().Key)
	return renderPanel("CONTROL DECK", subtitle, body, width-4, false)
}
