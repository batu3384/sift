package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/batu3384/sift/internal/domain"
)

func (m statusModel) Init() tea.Cmd { return nil }

func (m statusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c", "enter":
			return m, tea.Quit
		case "g":
			if m.companionMode == "off" {
				m.companionMode = "full"
			} else {
				m.companionMode = "off"
			}
		}
	}
	return m, nil
}

func (m resultModel) Init() tea.Cmd   { return nil }
func (m progressModel) Init() tea.Cmd { return nil }

func (m resultModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.result.Items)-1 {
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

func buildResultModel(plan domain.ExecutionPlan, result domain.ExecutionResult, previous resultModel, width int, height int) resultModel {
	model := resultModel{
		plan:   plan,
		result: result,
		width:  width,
		height: height,
		flash:  true,
		filter: preferredResultFilter(result, previous.filter),
	}
	if !model.restoreSelectionPath(previous.currentSelectionPath()) {
		model.cursor = model.preferredCursor()
	}
	model.clampCursor()
	return model
}

func preferredResultFilter(result domain.ExecutionResult, previous resultFilter) resultFilter {
	if hasRecoveryCandidates(result) {
		return resultFilterIssues
	}
	previous = coalesceResultFilter(previous)
	if resultHasVisibleItems(result, previous) {
		return previous
	}
	if resultHasVisibleItems(result, resultFilterClean) {
		return resultFilterClean
	}
	return resultFilterAll
}

func resultHasVisibleItems(result domain.ExecutionResult, filter resultFilter) bool {
	for _, item := range result.Items {
		if resultFilterMatch(filter, item) {
			return true
		}
	}
	return false
}

func (m resultModel) selectedItem() (domain.OperationResult, bool) {
	visible := m.visibleIndices()
	if len(visible) == 0 || m.cursor < 0 || m.cursor >= len(visible) {
		return domain.OperationResult{}, false
	}
	return m.result.Items[visible[m.cursor]], true
}

func (m resultModel) currentSelectionPath() string {
	item, ok := m.selectedItem()
	if !ok {
		return ""
	}
	return strings.TrimSpace(item.Path)
}

func (m *resultModel) restoreSelectionPath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	visible := m.visibleIndices()
	for visibleIdx, resultIdx := range visible {
		if strings.TrimSpace(m.result.Items[resultIdx].Path) == path {
			m.cursor = visibleIdx
			return true
		}
	}
	return false
}

func (m resultModel) preferredCursor() int {
	visible := m.visibleIndices()
	if len(visible) == 0 {
		return 0
	}
	if m.filter == resultFilterIssues {
		for visibleIdx, resultIdx := range visible {
			item := m.result.Items[resultIdx]
			if item.Status == domain.StatusFailed || item.Status == domain.StatusProtected {
				return visibleIdx
			}
		}
	}
	return 0
}

func (m *resultModel) cycleFilter() {
	currentPath := m.currentSelectionPath()
	switch m.filter {
	case resultFilterIssues:
		m.filter = resultFilterClean
	case resultFilterClean:
		m.filter = resultFilterAll
	default:
		m.filter = resultFilterIssues
	}
	if !m.restoreSelectionPath(currentPath) {
		m.cursor = m.preferredCursor()
	}
	m.clampCursor()
}

func (m *resultModel) clampCursor() {
	visible := m.visibleIndices()
	if len(visible) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(visible) {
		m.cursor = len(visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m resultModel) visibleIndices() []int {
	indices := make([]int, 0, len(m.result.Items))
	for idx, item := range m.result.Items {
		if resultFilterMatch(m.filter, item) {
			indices = append(indices, idx)
		}
	}
	return indices
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "pgup":
			m.browseProgressBy(-m.browsePageSize())
		case "pgdown":
			m.browseProgressBy(m.browsePageSize())
		case "home":
			m.browseOldestProgress()
		case "end":
			m.returnToLiveProgress()
		case "j", "down":
			m.browseProgressBy(1)
		case "k", "up":
			m.browseProgressBy(-1)
		}
	}
	return m, nil
}

func (m progressModel) browsePageSize() int {
	if m.height <= 0 {
		return 5
	}
	return max(m.height/3, 5)
}

func (m progressModel) shouldUseHistoryNavigation() bool {
	return len(m.plan.Items) > 1
}

func (m *progressModel) browseProgressBy(delta int) {
	if len(m.plan.Items) == 0 || delta == 0 {
		return
	}
	m.cursor = max(0, min(m.cursor+delta, len(m.plan.Items)-1))
	m.syncProgressAutoFollow()
}

func (m *progressModel) browseOldestProgress() {
	if len(m.plan.Items) == 0 {
		return
	}
	m.cursor = 0
	m.autoFollow = false
}

func (m *progressModel) returnToLiveProgress() {
	if len(m.plan.Items) == 0 {
		return
	}
	if runningCursor, ok := m.runningCursor(); ok {
		m.cursor = runningCursor
	} else {
		m.cursor = len(m.plan.Items) - 1
	}
	m.autoFollow = true
}

func (m *progressModel) syncProgressAutoFollow() {
	m.autoFollow = false
	if runningCursor, ok := m.runningCursor(); ok && m.cursor == runningCursor {
		m.autoFollow = true
	}
}

func (m *progressModel) apply(progress domain.ExecutionProgress) {
	if !progress.StartedAt.IsZero() {
		m.startedAt = progress.StartedAt
	}
	item := progress.Item
	m.current = &item
	m.currentPhase = progress.Phase
	m.currentStep = strings.TrimSpace(progress.Step)
	m.currentDetail = strings.TrimSpace(progress.Detail)
	if progress.Event == domain.ProgressEventSection {
		m.currentSectionKey = strings.TrimSpace(progress.SectionKey)
		m.currentSection = stageInfo{
			Category: item.Category,
			Group:    strings.TrimSpace(progress.SectionLabel),
			Index:    progress.SectionIndex,
			Total:    progress.SectionTotal,
			Done:     progress.SectionDone,
			Items:    progress.SectionItems,
			Bytes:    progress.SectionBytes,
		}
		if m.autoFollow {
			m.cursor = max(progress.Current-1, 0)
			if idx, ok := m.cursorForPath(item.Path); ok {
				m.cursor = idx
			}
		}
		return
	}
	if m.autoFollow {
		m.cursor = max(progress.Current-1, 0)
		if idx, ok := m.cursorForPath(item.Path); ok {
			m.cursor = idx
		}
	}
	if progress.Phase == domain.ProgressPhaseFinished {
		m.items = append(m.items, progress.Result)
		if m.currentSectionKey != "" && progressGroupKey(item) == m.currentSectionKey {
			m.currentSection.Done++
			if progress.Result.Status == domain.StatusDeleted {
				m.currentSection.Freed += item.Bytes
			}
		}
	}
}

func (m progressModel) cursorForPath(path string) (int, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return 0, false
	}
	for idx, item := range m.plan.Items {
		if strings.TrimSpace(item.Path) == path {
			return idx, true
		}
	}
	return 0, false
}

func (m progressModel) runningCursor() (int, bool) {
	if m.current == nil {
		return 0, false
	}
	return m.cursorForPath(m.current.Path)
}

func (m progressModel) View() string {
	width, height := effectiveSize(m.width, m.height)
	contentWidth := width - 6
	leftWidth, rightWidth := splitColumns(contentWidth-2, 0.58, 46, 32)
	panelLines := bodyLineBudget(height, 18, 7)
	body := joinPanels(
		renderPanel("PROGRESS RAIL", progressSummary(m), progressListView(m, leftWidth-4, panelLines), leftWidth, true),
		renderPanel("ACTION DECK", progressDetailSubtitle(m), progressDetailView(m, rightWidth-4, panelLines), rightWidth, false),
		width-4,
	)
	return renderChrome(
		"SIFT / Progress",
		"running",
		progressStats(m, width),
		body,
		nil,
		width,
		false,
		m.height,
	)
}

func (m resultModel) View() string {
	width, height := effectiveSize(m.width, m.height)
	contentWidth := width - 6
	leftWidth, rightWidth := splitColumns(contentWidth-2, 0.56, 46, 32)
	panelLines := bodyLineBudget(height, 19, 7)
	body := joinPanels(
		renderPanel("SETTLED RAIL", resultListSubtitle(m), resultListView(m, leftWidth-4, panelLines), leftWidth, true),
		renderPanel("OUTCOME DECK", resultDetailSubtitle(m), resultDetailView(m, rightWidth-4, panelLines), rightWidth, false),
		width-4,
	)
	if m.flash {
		body = strings.Join([]string{
			renderPanel("SETTLED GATE", "execution stream settled", safeStyle.Render("Operations finished. Review warnings and suggested commands before leaving this screen."), width-4, false),
			body,
		}, "\n")
	}
	return renderChrome(
		"SIFT / Result",
		"results",
		resultStats(m.plan, m.result, width),
		body,
		nil,
		width,
		false,
		m.height,
	)
}

func (m statusModel) View() string {
	width, height := effectiveSize(m.width, m.height)
	compact := width < 118 || height < 28
	contentWidth := width - 6
	leftWidth, rightWidth := splitColumns(contentWidth-2, 0.56, 40, 36)
	var panelLines int
	overviewLines := 6
	if compact {
		overviewLines = 5
		if height >= 26 || width >= 96 {
			overviewLines = 6
		}
		panelLines = max(bodyLineBudget(height, 20, 4)-2, 4)
	} else {
		overviewLines = 10
		if height < 36 {
			overviewLines = 8
		}
		panelLines = max((height-26)/2, 6)
	}
	overview := renderPanel("OBSERVATORY", statusOverviewSubtitle(m.live, m.lastExecution, m.scans), statusOverviewView(m, width-8, overviewLines), width-4, false)
	if compact {
		body := strings.Join([]string{
			overview,
			renderPanel("LIVE RAIL", statusLiveSubtitle(m.live), statusSystemViewWithTrends(m, width-8, panelLines), width-4, true),
			renderPanel("SESSION RAIL", statusActivitySubtitle(m.scans, m.lastExecution), statusActivityView(m.scans, m.lastExecution, width-8, max(panelLines-1, 4)), width-4, false),
		}, "\n")
		return renderChrome(
			"SIFT / Status",
			"observatory",
			statusStats(m.live, m.lastExecution, m.scans, m.diagnostics, m.updateNotice, width),
			body,
			nil,
			width,
			false,
			m.height,
		)
	}
	if width < 112 || height < 24 {
		body := strings.Join([]string{
			overview,
			renderPanel("LIVE RAIL", statusLiveSubtitle(m.live), statusSystemViewWithTrends(m, width-8, panelLines), width-4, true),
			renderPanel("SESSION RAIL", statusActivitySubtitle(m.scans, m.lastExecution), statusActivityView(m.scans, m.lastExecution, width-8, panelLines), width-4, false),
		}, "\n")
		return renderChrome(
			"SIFT / Status",
			"observatory",
			statusStats(m.live, m.lastExecution, m.scans, m.diagnostics, m.updateNotice, width),
			body,
			nil,
			width,
			false,
			m.height,
		)
	}
	leftColumn := strings.Join([]string{
		renderPanel("LIVE RAIL", statusLiveSubtitle(m.live), statusSystemViewWithTrends(m, leftWidth-4, panelLines), leftWidth, true),
		renderPanel("STORAGE RAIL", statusStorageSubtitle(m.live), statusStorageView(m, leftWidth-4, panelLines), leftWidth, false),
	}, "\n")
	rightColumn := strings.Join([]string{
		renderPanel("POWER RAIL", statusPowerSubtitle(m.live), statusPowerView(m, rightWidth-4, panelLines), rightWidth, false),
		renderPanel("SESSION RAIL", statusActivitySubtitle(m.scans, m.lastExecution), statusActivityView(m.scans, m.lastExecution, rightWidth-4, panelLines), rightWidth, false),
	}, "\n")
	body := strings.Join([]string{overview, joinPanels(leftColumn, rightColumn, width-4)}, "\n")
	return renderChrome(
		"SIFT / Status",
		"observatory",
		statusStats(m.live, m.lastExecution, m.scans, m.diagnostics, m.updateNotice, width),
		body,
		nil,
		width,
		false,
		m.height,
	)
}
