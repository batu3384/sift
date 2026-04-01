package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"

	"github.com/batuhanyuksel/sift/internal/domain"
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
		return routeHelp{
			short: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Back, m.keys.Help, m.keys.Quit},
			sections: []helpSection{
				{title: "Move", bindings: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter}},
				{title: "Back", bindings: []key.Binding{m.keys.Back, m.keys.Help, m.keys.Quit}},
			},
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
		return routeHelp{
			short: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Stage, m.keys.Review, m.keys.Help, m.keys.Back},
			sections: []helpSection{
				{title: "Move", bindings: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Back}},
				{title: "Batch", bindings: []key.Binding{queue, m.keys.Review}},
				{title: "Find", bindings: []key.Binding{m.keys.Search, m.keys.Refresh}},
				{title: "Back", bindings: []key.Binding{m.keys.Help, m.keys.Quit}},
			},
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
		return routeHelp{
			short: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Stage, m.keys.Review, m.keys.Help, m.keys.Back},
			sections: []helpSection{
				{title: "Move", bindings: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Focus, m.keys.Enter, m.keys.Back}},
				{title: "Mark", bindings: []key.Binding{toggle, m.keys.Review}},
				{title: "Find", bindings: []key.Binding{m.keys.Search, m.keys.Filter, m.keys.Sort}},
				{title: "File", bindings: []key.Binding{m.keys.Open, m.keys.Reveal, m.keys.Delete}},
				{title: "Back", bindings: []key.Binding{m.keys.Help, m.keys.Quit}},
			},
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
		return routeHelp{
			short: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Stop, m.keys.Help},
			sections: []helpSection{
				{title: "Move", bindings: []key.Binding{m.keys.Up, m.keys.Down}},
				{title: "Do", bindings: []key.Binding{m.keys.Stop}},
				{title: "Back", bindings: []key.Binding{m.keys.Help}},
			},
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

func (m appModel) compactRouteBindings(bindings []key.Binding) []key.Binding {
	switch m.route {
	case RouteHome:
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Tools, m.keys.Help}
	case RouteClean, RouteTools:
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Back, m.keys.Help}
	case RouteProtect:
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Add, m.keys.Delete, m.keys.Explain, m.keys.Help}
	case RouteStatus, RouteDoctor:
		if m.route == RouteStatus {
			return []key.Binding{m.keys.Back, m.keys.Refresh, m.keys.Companion, m.keys.Help}
		}
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Refresh, m.keys.Back, m.keys.Help}
	case RouteUninstall:
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Stage, m.keys.Review, m.keys.Enter, m.keys.Help}
	case RouteAnalyze:
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Focus, m.keys.Stage, m.keys.Review, m.keys.Help}
	case RouteReview:
		return append(m.reviewBindings(true), m.keys.Help)
	case RoutePreflight:
		return []key.Binding{m.keys.Execute, m.keys.Back, m.keys.Help}
	case RouteProgress:
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Stop, m.keys.Help}
	case RouteResult:
		return append(m.resultBindings(true), m.keys.Help)
	default:
		if len(bindings) > 4 {
			return bindings[:4]
		}
		return bindings
	}
}

func (m appModel) footerSecondaryHint(compact bool) string {
	items := m.routeSecondaryBindings()
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, binding := range items {
		if !binding.Enabled() {
			continue
		}
		help := binding.Help()
		if strings.TrimSpace(help.Desc) == "" {
			continue
		}
		keyLabel := strings.TrimSpace(help.Key)
		if keyLabel == "" {
			keyLabel = strings.Join(binding.Keys(), "/")
		}
		parts = append(parts, keyLabel+" "+help.Desc)
	}
	if len(parts) == 0 {
		return ""
	}
	limit := m.secondaryHintLimit()
	if limit > 0 && len(parts) > limit {
		parts = parts[:limit]
	}
	label := "Also: " + strings.Join(parts, "  •  ")
	maxWidth := max(m.width-24, 24)
	if compact {
		maxWidth = max(m.width-36, 20)
	}
	return truncateText(label, maxWidth)
}

func (m appModel) secondaryHintLimit() int {
	switch m.route {
	case RouteStatus:
		return 1
	case RouteUninstall:
		return 2
	case RouteAnalyze:
		return 3
	case RouteReview, RouteResult:
		return 2
	default:
		return 0
	}
}

func (m appModel) routeSecondaryBindings() []key.Binding {
	switch m.route {
	case RouteStatus:
		return []key.Binding{m.keys.Companion}
	case RouteUninstall:
		return []key.Binding{m.keys.Search, m.keys.Refresh}
	case RouteAnalyze:
		return []key.Binding{m.keys.Focus, m.keys.Search, m.keys.Filter, m.keys.Open}
	case RouteReview:
		return []key.Binding{m.keys.Module}
	case RoutePreflight:
		return nil
	case RouteResult:
		secondary := []key.Binding{}
		if len(resultRecoveryPathsForStatuses(m.result.plan, m.result.result, m.result.filter, domain.StatusFailed)) > 0 {
			secondary = append(secondary, m.keys.Retry)
		}
		if len(resultCurrentGroupRecoveryCandidates(m.result)) > 0 {
			secondary = append(secondary, m.keys.Module)
		}
		return secondary
	default:
		return nil
	}
}

func (m appModel) helpOverlayView() string {
	width, height := effectiveSize(m.width, m.height)
	panelLines := bodyLineBudget(height, 16, 8)
	body := renderHelpSections(m.routeBindings().sections, width-8, panelLines)
	subtitle := fmt.Sprintf("%s • %s or %s closes", routeHelpLabel(m.route), m.keys.Help.Help().Key, m.keys.Back.Help().Key)
	return renderPanel("HELP", subtitle, body, width-4, false)
}

func renderHelpSections(sections []helpSection, width int, maxLines int) string {
	lines := make([]string, 0, len(sections)*4)
	for _, section := range sections {
		if len(section.bindings) == 0 {
			continue
		}
		if len(lines) > 0 {
			lines = append(lines, renderSectionRule(width))
		}
		lines = append(lines, headerStyle.Render(strings.ToUpper(section.title)))
		for _, binding := range section.bindings {
			help := binding.Help()
			if !binding.Enabled() || strings.TrimSpace(help.Desc) == "" {
				continue
			}
			keyLabel := help.Key
			if strings.TrimSpace(keyLabel) == "" {
				keyLabel = strings.Join(binding.Keys(), "/")
			}
			lines = append(lines, wrapText(fmt.Sprintf("%-12s %s", keyLabel, help.Desc), width))
		}
	}
	if len(lines) == 0 {
		lines = append(lines, mutedStyle.Render("No controls available for this route."))
	}
	return strings.Join(viewportLines(lines, 0, maxLines), "\n")
}

func routeHelpLabel(route Route) string {
	switch route {
	case RouteHome:
		return "Home"
	case RouteClean:
		return "Clean"
	case RouteTools:
		return "More Tools"
	case RouteProtect:
		return "Protect"
	case RouteUninstall:
		return "Uninstall"
	case RouteStatus:
		return "Status"
	case RouteDoctor:
		return "Doctor"
	case RouteAnalyze:
		return "Analyze"
	case RouteReview:
		return "Review"
	case RoutePreflight:
		return "Permissions"
	case RouteProgress:
		return "Progress"
	case RouteResult:
		return "Result"
	default:
		return "Current screen"
	}
}

func (m appModel) reviewBindings(compact bool) []key.Binding {
	toggle := relabeledBinding(m.keys.Stage, "toggle")
	if shouldExecutePlan(m.review.effectivePlan()) {
		if compact {
			return []key.Binding{m.keys.Up, m.keys.Down, toggle, m.keys.Module, m.keys.Execute, m.keys.Back}
		}
		return []key.Binding{m.keys.Up, m.keys.Down, toggle, m.keys.Module, m.keys.Execute, m.keys.Cancel, m.keys.Back}
	}
	if compact {
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Module, m.keys.Back}
	}
	return []key.Binding{m.keys.Up, m.keys.Down, toggle, m.keys.Module, m.keys.Back, m.keys.Quit}
}

func (m appModel) resultBindings(compact bool) []key.Binding {
	issuePaths := resultRecoveryPaths(m.result.plan, m.result.result, m.result.filter)
	failedPaths := resultRecoveryPathsForStatuses(m.result.plan, m.result.result, m.result.filter, domain.StatusFailed)
	group := resultCurrentGroupRecoveryCandidates(m.result)
	if len(issuePaths) > 0 && len(failedPaths) > 0 && len(group) > 0 {
		if compact {
			return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Filter, m.keys.Retry, m.keys.Module, m.keys.Review, m.keys.Back}
		}
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Filter, m.keys.Retry, m.keys.Module, m.keys.Review, m.keys.Back, m.keys.Quit}
	}
	if len(issuePaths) > 0 && len(failedPaths) > 0 {
		if compact {
			return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Filter, m.keys.Retry, m.keys.Review, m.keys.Back}
		}
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Filter, m.keys.Retry, m.keys.Review, m.keys.Back, m.keys.Quit}
	}
	if len(issuePaths) > 0 && len(group) > 0 {
		if compact {
			return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Filter, m.keys.Module, m.keys.Review, m.keys.Back}
		}
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Filter, m.keys.Module, m.keys.Review, m.keys.Back, m.keys.Quit}
	}
	if len(issuePaths) > 0 {
		if compact {
			return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Filter, m.keys.Review, m.keys.Back}
		}
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Filter, m.keys.Review, m.keys.Back, m.keys.Quit}
	}
	return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Filter, m.keys.Back}
}

func shouldExecutePlan(plan domain.ExecutionPlan) bool {
	if plan.DryRun || plan.Command == "analyze" || plan.PlanState == "empty" {
		return false
	}
	for _, item := range plan.Items {
		if item.Action == domain.ActionAdvisory || item.Action == domain.ActionSkip {
			continue
		}
		if item.Status == domain.StatusProtected {
			continue
		}
		return true
	}
	return false
}

func resultRecoveryPaths(plan domain.ExecutionPlan, result domain.ExecutionResult, filter resultFilter) []string {
	candidates := resultRecoveryCandidates(plan, result, filter)
	if len(candidates) == 0 {
		return nil
	}
	return recoveryCandidatePaths(candidates)
}

func resultRecoveryPathsForStatuses(plan domain.ExecutionPlan, result domain.ExecutionResult, filter resultFilter, statuses ...domain.FindingStatus) []string {
	candidates := resultRecoveryCandidatesForStatuses(plan, result, filter, statuses...)
	if len(candidates) == 0 {
		return nil
	}
	return recoveryCandidatePaths(candidates)
}

func recoveryCandidatePaths(candidates []domain.Finding) []string {
	paths := make([]string, 0)
	seen := map[string]struct{}{}
	for _, item := range candidates {
		path := strings.TrimSpace(item.Path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths
}

func relabeledBinding(binding key.Binding, desc string) key.Binding {
	keys := binding.Keys()
	if len(keys) == 0 {
		return key.NewBinding(key.WithHelp("", desc))
	}
	helpText := binding.Help().Key
	if strings.TrimSpace(helpText) == "" {
		helpText = strings.Join(keys, "/")
	}
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(helpText, desc))
}
