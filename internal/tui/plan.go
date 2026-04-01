package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/engine"
	"github.com/batuhanyuksel/sift/internal/platform"
	"github.com/batuhanyuksel/sift/internal/store"
)

type planModel struct {
	plan             domain.ExecutionPlan
	cursor           int
	excluded         map[string]bool
	requiresDecision bool
	confirmed        bool
	decisionMade     bool
	width            int
	height           int
}

type resultModel struct {
	plan          domain.ExecutionPlan
	result        domain.ExecutionResult
	cursor        int
	width         int
	height        int
	flash         bool
	spinnerFrame  int
	filter        resultFilter
	reducedMotion bool
}

type resultFilter string

const (
	resultFilterAll    resultFilter = "all"
	resultFilterIssues resultFilter = "issues"
	resultFilterClean  resultFilter = "clean"
)

type progressModel struct {
	plan              domain.ExecutionPlan
	items             []domain.OperationResult
	cursor            int
	width             int
	height            int
	startedAt         time.Time
	spinnerFrame      int
	pulse             bool
	current           *domain.Finding
	currentPhase      domain.ProgressPhase
	currentStep       string
	currentDetail     string
	currentSection    stageInfo
	currentSectionKey string // engine section key for the active section
	cancelRequested   bool
	autoFollow        bool
	reducedMotion     bool
}

type analyzeLoader func(target string) (domain.ExecutionPlan, error)
type analyzeReviewLoader func(paths []string) (domain.ExecutionPlan, error)
type analyzeOpenFunc func(path string) error
type analyzePreviewLoader func(paths []string) map[string]domain.DirectoryPreview

type analyzeHistoryEntry struct {
	plan   domain.ExecutionPlan
	cursor int
}

type analyzeDirectoryPreview = domain.DirectoryPreview

type analyzeLoadedMsg struct {
	plan domain.ExecutionPlan
	err  error
}

type analyzeReviewLoadedMsg struct {
	plan domain.ExecutionPlan
	err  error
}

type analyzeBrowserModel struct {
	plan          domain.ExecutionPlan
	cursor        int
	queueCursor   int
	history       []analyzeHistoryEntry
	loader        analyzeLoader
	reviewLoader  analyzeReviewLoader
	opener        analyzeOpenFunc
	loading       bool
	errMsg        string
	search        textinput.Model
	searchActive  bool
	staged        map[string]domain.Finding
	stageOrder    []string
	nextPlan      *domain.ExecutionPlan
	reviewPreview menuPreviewState
	filter        analyzeFilter
	queueSort     analyzeQueueSort
	pane          analyzePane
	previewLoader analyzePreviewLoader
	previewCache  map[string]analyzeDirectoryPreview
	width         int
	height        int
}

type analyzeFilter string
type analyzeQueueSort string
type analyzePane string

const (
	analyzeFilterAll    analyzeFilter = "all"
	analyzeFilterQueued analyzeFilter = "queued"
	analyzeFilterHigh   analyzeFilter = "high"

	analyzeQueueSortOrder analyzeQueueSort = "order"
	analyzeQueueSortSize  analyzeQueueSort = "size"
	analyzeQueueSortAge   analyzeQueueSort = "age"

	analyzePaneBrowse analyzePane = "browse"
	analyzePaneQueue  analyzePane = "queue"

	// Max history entries to prevent memory leaks
	maxAnalyzeHistory = 50
)

func newAnalyzeSearchInput() textinput.Model {
	search := textinput.New()
	search.Prompt = "search> "
	search.Placeholder = "filter current findings"
	search.CharLimit = 128
	search.Width = 28
	return search
}

func (m planModel) Init() tea.Cmd { return nil }

func (m planModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			if m.requiresDecision {
				m.decisionMade = true
			}
			return m, tea.Quit
		case "y":
			if m.requiresDecision {
				m.confirmed = true
				m.decisionMade = true
			}
			return m, tea.Quit
		case "n":
			if m.requiresDecision {
				m.decisionMade = true
			}
			return m, tea.Quit
		case " ":
			item, ok := m.selectedItem()
			if !ok || !canToggleReviewItem(item) {
				return m, nil
			}
			if m.excluded == nil {
				m.excluded = map[string]bool{}
			}
			if m.excluded[item.ID] {
				delete(m.excluded, item.ID)
			} else {
				m.excluded[item.ID] = true
			}
		case "j", "down":
			if len(m.plan.Items) > 0 && m.cursor < len(m.plan.Items)-1 {
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

func (m planModel) View() string {
	plan := m.effectivePlan()
	width, height := effectiveSize(m.width, m.height)
	contentWidth := width - 6
	leftWidth, rightWidth := splitColumns(contentWidth-2, 0.58, 46, 32)
	panelLines := bodyLineBudget(height, 18, 7)
	body := joinPanels(
		renderPanel("ITEMS", fmt.Sprintf("%d %s • %s", plan.Totals.ItemCount, pl(plan.Totals.ItemCount, "item", "items"), domain.HumanBytes(plan.Totals.Bytes)), planListView(plan, m.cursor, leftWidth-4, panelLines), leftWidth, true),
		renderPanel("DETAIL", planDetailSubtitle(plan, m.cursor), planDetailView(m, rightWidth-4, panelLines), rightWidth, false),
		width-4,
	)
	if m.requiresDecision {
		body = strings.Join([]string{
			body,
			renderPanel("REVIEW", decisionSubtitle(plan), decisionView(m, width-8), width-4, false),
		}, "\n")
	}
	return renderChrome(
		"SIFT / "+strings.ToUpper(plan.Command),
		fmt.Sprintf("%s • review", plan.Platform),
		planStats(plan, width),
		body,
		nil,
		width,
		false,
		m.height,
	)
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

type statusModel struct {
	live          *engine.SystemSnapshot
	scans         []store.RecentScan
	lastExecution *store.ExecutionSummary
	diagnostics   []platform.Diagnostic
	updateNotice  *engine.UpdateNotice
	networkRxRate float64
	networkTxRate float64
	diskReadRate  float64
	diskWriteRate float64
	cpuTrend      []float64
	memoryTrend   []float64
	networkTrend  []float64
	diskTrend     []float64
	width         int
	height        int
	loading       bool
	loadingLabel  string
	pulse         bool
	signalFrame   int
	companionMode string
	reducedMotion bool
}

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
		case "j", "down":
			if len(m.plan.Items) > 0 && m.cursor < len(m.plan.Items)-1 {
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
		// Keep currentSection.Done and currentSection.Freed in sync so the
		// live stage banner shows accurate real-time progress between section events.
		if m.currentSectionKey != "" && progressGroupKey(item) == m.currentSectionKey {
			m.currentSection.Done++
			if progress.Result.Status == domain.StatusDeleted || progress.Result.Status == domain.StatusCompleted {
				m.currentSection.Freed += item.Bytes
			}
		}
	}
}

func (m progressModel) selectedItem() (domain.Finding, bool) {
	if m.cursor < 0 || m.cursor >= len(m.plan.Items) {
		return domain.Finding{}, false
	}
	return m.plan.Items[m.cursor], true
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
		renderPanel("ITEMS", progressSummary(m), progressListView(m, leftWidth-4, panelLines), leftWidth, true),
		renderPanel("DETAIL", progressDetailSubtitle(m), progressDetailView(m, rightWidth-4, panelLines), rightWidth, false),
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
		renderPanel("RESULTS", resultListSubtitle(m), resultListView(m, leftWidth-4, panelLines), leftWidth, true),
		renderPanel("DETAIL", resultDetailSubtitle(m), resultDetailView(m, rightWidth-4, panelLines), rightWidth, false),
		width-4,
	)
	if m.flash {
		body = strings.Join([]string{
			renderPanel("DONE", "execution stream settled", safeStyle.Render("Operations finished. Review warnings and suggested commands before leaving this screen."), width-4, false),
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
	panelLines := bodyLineBudget(height, 23, 4)
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
	overview := renderPanel("OVERVIEW", statusOverviewSubtitle(m.live, m.lastExecution, m.scans), statusOverviewView(m, width-8, overviewLines), width-4, false)
	if compact {
		body := strings.Join([]string{
			overview,
			renderPanel("SYSTEM", statusLiveSubtitle(m.live), statusSystemViewWithTrends(m, width-8, panelLines), width-4, true),
			renderPanel("ACTIVITY", statusActivitySubtitle(m.scans, m.lastExecution), statusActivityView(m.scans, m.lastExecution, width-8, max(panelLines-1, 4)), width-4, false),
		}, "\n")
		return renderChrome(
			"SIFT / Status",
			"live",
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
			renderPanel("SYSTEM", statusLiveSubtitle(m.live), statusSystemViewWithTrends(m, width-8, panelLines), width-4, true),
			renderPanel("ACTIVITY", statusActivitySubtitle(m.scans, m.lastExecution), statusActivityView(m.scans, m.lastExecution, width-8, panelLines), width-4, false),
		}, "\n")
		return renderChrome(
			"SIFT / Status",
			"live",
			statusStats(m.live, m.lastExecution, m.scans, m.diagnostics, m.updateNotice, width),
			body,
			nil,
			width,
			false,
			m.height,
		)
	}
	leftColumn := strings.Join([]string{
		renderPanel("SYSTEM", statusLiveSubtitle(m.live), statusSystemViewWithTrends(m, leftWidth-4, panelLines), leftWidth, true),
		renderPanel("STORAGE", statusStorageSubtitle(m.live), statusStorageView(m, leftWidth-4, panelLines), leftWidth, false),
	}, "\n")
	rightColumn := strings.Join([]string{
		renderPanel("POWER", statusPowerSubtitle(m.live), statusPowerView(m, rightWidth-4, panelLines), rightWidth, false),
		renderPanel("ACTIVITY", statusActivitySubtitle(m.scans, m.lastExecution), statusActivityView(m.scans, m.lastExecution, rightWidth-4, panelLines), rightWidth, false),
	}, "\n")
	body := strings.Join([]string{overview, joinPanels(leftColumn, rightColumn, width-4)}, "\n")
	return renderChrome(
		"SIFT / Status",
		"live",
		statusStats(m.live, m.lastExecution, m.scans, m.diagnostics, m.updateNotice, width),
		body,
		nil,
		width,
		false,
		m.height,
	)
}

func (m planModel) selectedItem() (domain.Finding, bool) {
	if m.cursor < 0 || m.cursor >= len(m.plan.Items) {
		return domain.Finding{}, false
	}
	return m.plan.Items[m.cursor], true
}
