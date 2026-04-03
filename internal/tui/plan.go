package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/platform"
	"github.com/batu3384/sift/internal/store"
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

func (m planModel) selectedItem() (domain.Finding, bool) {
	if m.cursor < 0 || m.cursor >= len(m.plan.Items) {
		return domain.Finding{}, false
	}
	return m.plan.Items[m.cursor], true
}
