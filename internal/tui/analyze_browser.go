package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func newAnalyzeSearchInput() textinput.Model {
	search := textinput.New()
	search.Prompt = "search> "
	search.Placeholder = "filter current findings"
	search.CharLimit = 128
	search.Width = 28
	return search
}

func (m analyzeBrowserModel) Init() tea.Cmd { return nil }

func (m analyzeBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc", "backspace", "left", "h":
			if len(m.history) == 0 {
				return m, tea.Quit
			}
			last := m.history[len(m.history)-1]
			m.history = m.history[:len(m.history)-1]
			m.plan = last.plan
			m.cursor = last.cursor
			m.errMsg = ""
			m.loading = false
			m.syncPreviewWindow()
			return m, nil
		case "j", "down":
			if len(m.plan.Items) > 0 && m.cursor < len(m.plan.Items)-1 {
				m.cursor++
			}
			m.syncPreviewWindow()
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
			m.syncPreviewWindow()
		case "enter", "right", "l":
			item, ok := m.selectedItem()
			if !ok || !canDescendInto(item) || m.loading {
				return m, nil
			}
			m.loading = true
			m.errMsg = ""
			target := item.Path
			return m, loadAnalyzeTarget(m.loader, target)
		case " ":
			item, ok := m.selectedItem()
			if !ok || !canStage(item) {
				return m, nil
			}
			m.toggleStage(item)
			m.syncPreviewWindow()
			return m, nil
		case "u":
			item, ok := m.selectedItem()
			if !ok {
				return m, nil
			}
			m.removeStage(item.Path)
			m.syncPreviewWindow()
			return m, nil
		case "o":
			item, ok := m.selectedItem()
			if !ok {
				return m, nil
			}
			if m.opener == nil {
				m.errMsg = "Open/reveal is not available in this environment."
				return m, nil
			}
			if err := m.opener(item.Path); err != nil {
				m.errMsg = err.Error()
				return m, nil
			}
			m.errMsg = "Opened " + item.Path
			return m, nil
		case "x":
			if m.loading {
				return m, nil
			}
			paths := m.stagedPaths()
			if len(paths) == 0 {
				m.errMsg = "Stage one or more items before opening the cleanup review."
				return m, nil
			}
			if m.reviewLoader == nil {
				m.errMsg = "Cleanup review is not available."
				return m, nil
			}
			m.loading = true
			m.errMsg = ""
			return m, loadAnalyzeReview(m.reviewLoader, paths)
		}
	case analyzeLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.history = append(m.history, analyzeHistoryEntry{
			plan:   m.plan,
			cursor: m.cursor,
		})
		m.plan = msg.plan
		m.cursor = 0
		m.errMsg = ""
		m.syncPreviewWindow()
		return m, nil
	case analyzeReviewLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		plan := msg.plan
		m.nextPlan = &plan
		return m, tea.Quit
	}
	return m, nil
}

func (m analyzeBrowserModel) View() string {
	width, height := effectiveSize(m.width, m.height)
	compact := width < 118 || height < 28
	contentWidth := width - 6
	leftWidth, rightWidth := splitColumns(contentWidth-2, 0.62, 48, 34)
	panelLines := bodyLineBudget(height, 18, 7)
	subtitle := analyzeBreadcrumb(m.history, m.plan)
	if subtitle == "" {
		subtitle = currentAnalyzeTarget(m.plan)
	}
	trail := analyzeTrailLine(m.history, m.plan, width-8)
	browseSubtitle := currentAnalyzeTarget(m.plan)
	if trail != "" && trail != browseSubtitle {
		browseSubtitle = trail
	}
	var body string
	if compact {
		browseBody := strings.Join([]string{
			analyzeTrailBlock(m.history, m.plan, width-8),
			analyzeListView(m, width-8, panelLines),
		}, "\n")
		detailLines := max(panelLines/2, 5)
		body = strings.Join([]string{
			renderPanel("FILES", browseSubtitle, browseBody, width-4, true),
			renderPanel("DETAIL", analyzeDetailSubtitle(m), analyzeDetailView(m, width-8, detailLines), width-4, false),
		}, "\n")
	} else {
		browseBody := strings.Join([]string{
			analyzeTrailBlock(m.history, m.plan, leftWidth-4),
			analyzeListView(m, leftWidth-4, max(panelLines-2, 5)),
		}, "\n")
		body = joinPanels(
			renderPanel("FILES", browseSubtitle, browseBody, leftWidth, true),
			renderPanel("DETAIL", analyzeDetailSubtitle(m), analyzeDetailView(m, rightWidth-4, panelLines), rightWidth, false),
			width-4,
		)
	}
	if m.loading {
		body = strings.Join([]string{
			renderPanel("LOADING", "refreshing", reviewStyle.Render("Refreshing analysis snapshot. Existing results stay visible until the new scan is ready."), width-4, false),
			body,
		}, "\n")
	}
	if len(m.stageOrder) > 1 {
		queueOrder := m.sortedStageOrder()
		body = strings.Join([]string{
			body,
			renderPanel("QUEUE", analyzeStageSummary(m.staged, queueOrder, m.queueSort), analyzeQueueView(m, queueOrder, width-10, m.queueSort), width-4, m.activePane() == analyzePaneQueue),
		}, "\n")
	}
	return renderChrome(
		"SIFT / Analyze",
		subtitle,
		analyzeStats(m.plan, m.stageOrder, m.loading, m.errMsg, width),
		body,
		nil,
		width,
		false,
		m.height,
	)
}
