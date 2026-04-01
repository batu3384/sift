package tui

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/batuhanyuksel/sift/internal/config"
	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/engine"
	"github.com/batuhanyuksel/sift/internal/platform"
)

type Route string

const (
	RouteHome      Route = "home"
	RouteClean     Route = "clean"
	RouteTools     Route = "tools"
	RouteProtect   Route = "protect"
	RouteUninstall Route = "uninstall"
	RouteStatus    Route = "status"
	RouteDoctor    Route = "doctor"
	RouteAnalyze   Route = "analyze"
	RouteReview    Route = "review"
	RoutePreflight Route = "preflight"
	RouteProgress  Route = "progress"
	RouteResult    Route = "result"
)

type DashboardData struct {
	Report      engine.StatusReport
	Diagnostics []platform.Diagnostic
	Update      *engine.UpdateNotice
}

type AppOptions struct {
	Config        config.Config
	Executable    bool
	InitialRoute  Route
	InitialPlan   *domain.ExecutionPlan
	InitialResult *domain.ExecutionResult
}

type AppCallbacks struct {
	LoadDashboard           func() (DashboardData, error)
	LoadCachedInstalledApps func() ([]domain.AppEntry, time.Time, error)
	LoadInstalledApps       func() ([]domain.AppEntry, error)
	LoadAnalyzeHome         func() (domain.ExecutionPlan, error)
	LoadAnalyzeTarget       func(target string) (domain.ExecutionPlan, error)
	LoadAnalyzePreviews     func(paths []string) map[string]domain.DirectoryPreview
	LoadCleanProfile        func(profile string) (domain.ExecutionPlan, error)
	LoadInstaller           func() (domain.ExecutionPlan, error)
	LoadOptimize            func() (domain.ExecutionPlan, error)
	LoadAutofix             func() (domain.ExecutionPlan, error)
	LoadPurgeScan           func() (domain.ExecutionPlan, error)
	LoadUninstallPlan       func(app string) (domain.ExecutionPlan, error)
	LoadUninstallBatchPlan  func(apps []string) (domain.ExecutionPlan, error)
	LoadReviewForPaths      func(paths []string) (domain.ExecutionPlan, error)
	AddProtectedPath        func(path string) (config.Config, string, error)
	RemoveProtectedPath     func(path string) (config.Config, string, bool, error)
	ExplainProtection       func(path string) domain.ProtectionExplanation
	ExecutePlan             func(plan domain.ExecutionPlan) (domain.ExecutionResult, error)
	ExecutePlanWithProgress func(ctx context.Context, plan domain.ExecutionPlan, emit func(domain.ExecutionProgress)) (domain.ExecutionResult, error)
	OpenPath                func(path string) error
	RevealPath              func(path string) error
	TrashPaths              func(paths []string) (domain.ExecutionResult, error)
	OnScanProgress          func(ruleID string, ruleName string, itemsFound int, bytesFound int64)
}

type dashboardLoadedMsg struct {
	data DashboardData
	err  error
}

type planLoadedMsg struct {
	plan      domain.ExecutionPlan
	err       error
	requestID int
}

type planLoadTransitionMsg struct {
	requestID int
}

type menuPreviewLoadedMsg struct {
	route     Route
	key       string
	plan      domain.ExecutionPlan
	err       error
	requestID int
}

type resultLoadedMsg struct {
	plan   domain.ExecutionPlan
	result domain.ExecutionResult
	err    error
}

type executionProgressMsg struct {
	progress domain.ExecutionProgress
}

type executionFinishedMsg struct {
	result domain.ExecutionResult
	err    error
}

type permissionWarmupFinishedMsg struct {
	err error
}

type appsLoadedMsg struct {
	apps      []domain.AppEntry
	err       error
	cached    bool
	loadedAt  time.Time
	requestID int
}

type protectAddedMsg struct {
	cfg  config.Config
	path string
	err  error
}

type protectRemovedMsg struct {
	cfg     config.Config
	path    string
	removed bool
	err     error
}

type analyzeActionFinishedMsg struct {
	result domain.ExecutionResult
	err    error
}

type scanProgressMsg struct {
	ruleID      string
	ruleName    string
	itemsFound  int
	bytesFound  int64
}

type dashboardTickMsg struct{}
type uiTickMsg struct{}

type appModel struct {
	route                           Route
	cfg                             config.Config
	executable                      bool
	hasHome                         bool
	width                           int
	height                          int
	keys                            appKeyMap
	help                            help.Model
	helpVisible                     bool
	callbacks                       AppCallbacks
	permissionWarmup                func(permissionPreflightModel) tea.Cmd
	permissionKeepalive             func(context.Context, permissionPreflightModel)
	home                            homeModel
	clean                           menuModel
	tools                           menuModel
	protect                         protectModel
	uninstall                       uninstallModel
	status                          statusModel
	doctor                          doctorModel
	analyze                         analyzeBrowserModel
	review                          planModel
	preflight                       permissionPreflightModel
	progress                        progressModel
	result                          resultModel
	errorMsg                        string
	noticeMsg                       string
	noticeTicks                     int
	loadingLabel                    string
	lastDashboardSync               string
	spinnerFrame                    int
	livePulse                       bool
	nextPlanRequestID               int
	activePlanRequestID             int
	nextMenuPreviewRequestID        int
	activeCleanPreviewRequestID     int
	activeUninstallPreviewRequestID int
	activeAnalyzePreviewRequestID   int
	nextInventoryRequestID          int
	activeInventoryRequestID        int
	reviewReturnRoute               Route
	resultReturnRoute               Route
	analyzeReturnRoute              Route
	pendingAnalyzePushHistory       bool
	pendingAnalyzePreserveCursor    bool
	pendingAnalyzeSelectionPath     string
	pendingAnalyzePane              analyzePane
	pendingAnalyzeQueuePath         string
	pendingTargetRoute              Route
	pendingReviewReturn             Route
	pendingResultReturn             Route
	planLoadTransitionVisible       bool
	scanCurrentRule                 string
	scanCurrentItems                int
	scanCurrentBytes                int64
	scanTotalBytes                  int64
	executionStream                 <-chan tea.Msg
	executionCancel                 context.CancelFunc
	acceptedPermissionProfiles      map[string]struct{}
}

const dashboardRefreshInterval = 15 * time.Second
const uiTickInterval = 250 * time.Millisecond

func RunApp(opts AppOptions, callbacks AppCallbacks) error {
	model := appModel{
		route:               opts.InitialRoute,
		cfg:                 config.Normalize(opts.Config),
		executable:          opts.Executable,
		hasHome:             opts.InitialRoute == RouteHome,
		keys:                defaultKeyMap(),
		help:                newHelpModel(),
		callbacks:           callbacks,
		permissionWarmup:    defaultPermissionWarmupCmd,
		permissionKeepalive: defaultPermissionKeepalive,
		acceptedPermissionProfiles: map[string]struct{}{},
		home: homeModel{
			actions:    buildHomeActions(opts.Config),
			executable: opts.Executable,
			cfg:        config.Normalize(opts.Config),
		},
		clean: menuModel{
			title:    "Clean",
			subtitle: "choose scope",
			hint:     "Quick for routine cleanup, workstation for cache-heavy days, deep for maximum reclaim.",
			actions:  buildCleanActions(),
		},
		tools: menuModel{
			title:    "More Tools",
			subtitle: "more tools",
			hint:     "Check, fixes, installer cleanup, protect, purge, and diagnostics live here.",
			actions:  buildToolsActions(config.Normalize(opts.Config)),
		},
		protect:   newProtectModel(config.Normalize(opts.Config).ProtectedPaths),
		uninstall: newUninstallModel(),
	}
	model.protect.syncFamilies(model.cfg.ProtectedFamilies)
	model.protect.syncScopes(model.cfg.CommandExcludes)
	switch opts.InitialRoute {
	case RouteHome:
		model.setHomeLoading("dashboard")
	case RouteStatus:
		model.setStatusLoading("dashboard")
	case RouteDoctor:
		model.setDoctorLoading("doctor")
	case RouteUninstall:
		model.setUninstallLoading("installed apps")
	}
	if opts.InitialPlan != nil {
		switch opts.InitialRoute {
		case RouteAnalyze:
			model.setAnalyzePlan(*opts.InitialPlan)
		case RouteReview:
			model.setReviewPlan(*opts.InitialPlan, shouldExecutePlan(*opts.InitialPlan))
		}
	}
	if opts.InitialResult != nil {
		model.result = resultModel{result: *opts.InitialResult}
		if opts.InitialPlan != nil {
			model.result.plan = *opts.InitialPlan
		}
	}
	_, err := newProgram(model).Run()
	return err
}

func (m appModel) Init() tea.Cmd {
	if m.route == RouteHome || m.route == RouteStatus || m.route == RouteDoctor {
		cmds := []tea.Cmd{loadDashboardCmd(m.callbacks.LoadDashboard), dashboardTickCmd()}
		if m.wantsUITick() {
			cmds = append(cmds, uiTickCmd())
		}
		return tea.Batch(cmds...)
	}
	if m.route == RouteUninstall {
		cmds := []tea.Cmd{loadUninstallInventoryCmd(m.callbacks.LoadInstalledApps, m.callbacks.LoadCachedInstalledApps, m.activeInventoryRequestID), dashboardTickCmd()}
		if m.wantsUITick() {
			cmds = append(cmds, uiTickCmd())
		}
		return tea.Batch(cmds...)
	}
	if m.wantsUITick() {
		return tea.Batch(dashboardTickCmd(), uiTickCmd())
	}
	return tea.Batch(dashboardTickCmd())
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.home.width, m.home.height = msg.Width, msg.Height
		m.clean.width, m.clean.height = msg.Width, msg.Height
		m.tools.width, m.tools.height = msg.Width, msg.Height
		m.protect.width, m.protect.height = msg.Width, msg.Height
		m.uninstall.width, m.uninstall.height = msg.Width, msg.Height
		m.status.width, m.status.height = msg.Width, msg.Height
		m.doctor.width, m.doctor.height = msg.Width, msg.Height
		m.analyze.width, m.analyze.height = msg.Width, msg.Height
		m.review.width, m.review.height = msg.Width, msg.Height
		m.preflight.width, m.preflight.height = msg.Width, msg.Height
		m.result.width, m.result.height = msg.Width, msg.Height
		return m, nil
	case dashboardLoadedMsg:
		m.clearDashboardLoading()
		if msg.err != nil {
			m.clearNotice()
			m.errorMsg = msg.err.Error()
			return m, nil
		}
		m.errorMsg = ""
		m.clearNotice()
		m.applyDashboard(msg.data)
		return m, nil
	case dashboardTickMsg:
		if m.dashboardLoadingActive() || m.loadingLabel != "" {
			return m, dashboardTickCmd()
		}
		if m.route == RouteHome || m.route == RouteStatus || m.route == RouteDoctor {
			return m, tea.Batch(loadDashboardCmd(m.callbacks.LoadDashboard), dashboardTickCmd())
		}
		return m, dashboardTickCmd()
	case uiTickMsg:
		m.livePulse = !m.livePulse
		m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
		m.tickNotices()
		m.home.pulse = m.livePulse
		m.home.signalFrame = m.spinnerFrame
		m.status.pulse = m.livePulse
		m.status.signalFrame = m.spinnerFrame
		if m.route == RouteProgress {
			m.progress.pulse = m.livePulse
		}
		if m.wantsUITick() {
			return m, uiTickCmd()
		}
		return m, nil
	case scanProgressMsg:
		// Update loading label with scan progress
		if msg.ruleName != "" {
			m.loadingLabel = "scanning: " + msg.ruleName
		}
		return m, nil
	case planLoadedMsg:
		if msg.requestID != 0 && msg.requestID != m.activePlanRequestID {
			return m, nil
		}
		m.loadingLabel = ""
		m.activePlanRequestID = 0
		m.planLoadTransitionVisible = false
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
			m.pendingAnalyzePushHistory = false
			m.pendingAnalyzePreserveCursor = false
			m.pendingAnalyzeSelectionPath = ""
			m.pendingAnalyzePane = ""
			m.pendingAnalyzeQueuePath = ""
			m.pendingTargetRoute = ""
			m.pendingReviewReturn = ""
			m.pendingResultReturn = ""
			return m, nil
		}
		m.errorMsg = ""
		switch m.pendingTargetRoute {
		case RouteAnalyze:
			if m.analyze.search.CharLimit == 0 {
				m.analyze.search = newAnalyzeSearchInput()
			}
			if m.analyze.previewLoader == nil {
				m.analyze.previewLoader = m.callbacks.LoadAnalyzePreviews
			}
			if m.pendingAnalyzePushHistory {
				m.analyze.history = append(m.analyze.history, analyzeHistoryEntry{
					plan:   m.analyze.plan,
					cursor: m.analyze.cursor,
				})
				// Limit history size to prevent memory leak
				if len(m.analyze.history) > maxAnalyzeHistory {
					m.analyze.history = m.analyze.history[len(m.analyze.history)-maxAnalyzeHistory:]
				}
			}
			cursor := 0
			if m.pendingAnalyzeSelectionPath != "" {
				restored := msg.plan
				m.analyze.plan = restored
				if matchedCursor, ok := m.analyze.cursorForPath(m.pendingAnalyzeSelectionPath); ok {
					cursor = matchedCursor
				} else if m.pendingAnalyzePreserveCursor {
					cursor = m.analyze.cursor
				}
			} else if m.pendingAnalyzePreserveCursor {
				cursor = m.analyze.cursor
			}
			m.analyze.plan = msg.plan
			m.analyze.cursor = cursor
			m.analyze.clampCursor()
			if m.pendingAnalyzePane == analyzePaneQueue && len(m.analyze.stageOrder) > 0 {
				m.analyze.pane = analyzePaneQueue
				if matchedQueueCursor, ok := m.analyze.queueCursorForPath(m.pendingAnalyzeQueuePath); ok {
					m.analyze.queueCursor = matchedQueueCursor
				}
			}
			m.analyze.clampQueueCursor()
			m.analyze.syncPreviewWindow()
			m.analyze.loading = false
			m.analyze.errMsg = ""
			m.route = RouteAnalyze
			m.analyzeReturnRoute = m.pendingReviewReturn
			m.pendingAnalyzePushHistory = false
			m.pendingAnalyzePreserveCursor = false
			m.pendingAnalyzeSelectionPath = ""
			m.pendingAnalyzePane = ""
			m.pendingAnalyzeQueuePath = ""
			m.pendingTargetRoute = ""
			m.pendingReviewReturn = ""
			m.pendingResultReturn = ""
			return m, m.startAnalyzeReviewPreviewLoad()
		case RouteReview:
			m.setReviewPlan(msg.plan, shouldExecutePlan(msg.plan))
			m.route = RouteReview
			m.reviewReturnRoute = m.pendingReviewReturn
			m.resultReturnRoute = m.pendingResultReturn
		}
		m.pendingAnalyzePushHistory = false
		m.pendingAnalyzePreserveCursor = false
		m.pendingAnalyzeSelectionPath = ""
		m.pendingAnalyzePane = ""
		m.pendingAnalyzeQueuePath = ""
		m.pendingTargetRoute = ""
		m.pendingReviewReturn = ""
		m.pendingResultReturn = ""
		return m, nil
	case planLoadTransitionMsg:
		if msg.requestID == 0 || msg.requestID != m.activePlanRequestID || !m.planLoadPending() {
			return m, nil
		}
		m.planLoadTransitionVisible = true
		return m, nil
	case menuPreviewLoadedMsg:
		switch msg.route {
		case RouteClean:
			if msg.requestID == 0 || msg.requestID != m.activeCleanPreviewRequestID {
				return m, nil
			}
			m.activeCleanPreviewRequestID = 0
			m.clean.applyPreview(msg.key, msg.plan, msg.err)
		case RouteUninstall:
			if msg.requestID == 0 || msg.requestID != m.activeUninstallPreviewRequestID {
				return m, nil
			}
			m.activeUninstallPreviewRequestID = 0
			m.uninstall.applyPreview(msg.key, msg.plan, msg.err)
		case RouteAnalyze:
			if msg.requestID == 0 || msg.requestID != m.activeAnalyzePreviewRequestID {
				return m, nil
			}
			m.activeAnalyzePreviewRequestID = 0
			m.analyze.applyReviewPreview(msg.key, msg.plan, msg.err)
		}
		return m, nil
	case resultLoadedMsg:
		m.loadingLabel = ""
		if msg.err != nil {
			m.clearNotice()
			m.errorMsg = msg.err.Error()
			return m, nil
		}
		m.errorMsg = ""
		m.clearNotice()
		m.result = buildResultModel(msg.plan, msg.result, m.result, m.width, m.height)
		m.route = RouteResult
		return m, nil
	case executionProgressMsg:
		m.errorMsg = ""
		m.progress.apply(msg.progress)
		if m.executionStream != nil {
			return m, waitForExecutionStreamCmd(m.executionStream)
		}
		return m, nil
	case permissionWarmupFinishedMsg:
		m.loadingLabel = ""
		if msg.err != nil {
			m.clearNotice()
			message := strings.TrimSpace(msg.err.Error())
			if message == "" {
				message = "Admin access was not confirmed."
			}
			m.errorMsg = message
			return m, nil
		}
		m.errorMsg = ""
		m.clearNotice()
		return m.executePreparedPreflight()
	case executionFinishedMsg:
		m.executionStream = nil
		m.executionCancel = nil
		if msg.err != nil {
			if errorsIsCanceled(msg.err) {
				m.errorMsg = ""
				m.setNotice("Execution cancelled. Partial results preserved.")
			} else {
				m.clearNotice()
				m.errorMsg = msg.err.Error()
			}
		} else {
			m.errorMsg = ""
			m.clearNotice()
		}
		m.result = buildResultModel(m.progress.plan, msg.result, m.result, m.width, m.height)
		m.route = RouteResult
		return m, nil
	case appsLoadedMsg:
		if msg.requestID != 0 && msg.requestID != m.activeInventoryRequestID {
			return m, nil
		}
		if msg.err != nil {
			if msg.cached {
				return m, nil
			}
			m.activeInventoryRequestID = 0
			m.setUninstallLoading("")
			m.clearNotice()
			m.errorMsg = msg.err.Error()
			return m, nil
		}
		if !msg.cached {
			m.activeInventoryRequestID = 0
			m.setUninstallLoading("")
		}
		m.errorMsg = ""
		m.clearNotice()
		m.applyInstalledApps(msg.apps)
		if msg.cached {
			m.uninstall.setMessage(uninstallInventoryMessage(msg.loadedAt, true), routeMessageLongTicks)
		} else {
			m.uninstall.setMessage(uninstallInventoryMessage(msg.loadedAt, false), routeMessageLongTicks)
		}
		m.route = RouteUninstall
		return m, m.startUninstallPreviewLoad()
	case protectAddedMsg:
		m.loadingLabel = ""
		if msg.err != nil {
			m.clearNotice()
			m.errorMsg = msg.err.Error()
			return m, nil
		}
		m.errorMsg = ""
		m.clearNotice()
		m.protect.inputActive = false
		m.protect.input.Blur()
		m.applyConfig(msg.cfg)
		m.protect.syncPaths(msg.cfg.ProtectedPaths)
		m.protect.syncScopes(msg.cfg.CommandExcludes)
		m.protect.setMessage("Protected path added: "+msg.path, routeMessageTicks)
		m.refreshProtectExplanation(msg.path)
		return m, nil
	case protectRemovedMsg:
		m.loadingLabel = ""
		if msg.err != nil {
			m.clearNotice()
			m.errorMsg = msg.err.Error()
			return m, nil
		}
		m.errorMsg = ""
		m.clearNotice()
		m.applyConfig(msg.cfg)
		m.protect.syncPaths(msg.cfg.ProtectedPaths)
		m.protect.syncScopes(msg.cfg.CommandExcludes)
		if msg.removed {
			m.protect.setMessage("Protected path removed: "+msg.path, routeMessageTicks)
		} else {
			m.protect.setMessage("Protected path not found: "+msg.path, routeMessageLongTicks)
		}
		m.refreshProtectExplanation(m.protect.selectedPath())
		return m, nil
	case analyzeActionFinishedMsg:
		m.loadingLabel = ""
		if msg.err != nil {
			m.clearNotice()
			m.errorMsg = msg.err.Error()
			return m, nil
		}
		completed, deleted, failed, skipped, protected := countResultStatuses(msg.result)
		m.pendingAnalyzeSelectionPath = m.analyze.fallbackSelectionPathAfterRemoval(analyzeResultDeletedPaths(msg.result))
		for _, item := range msg.result.Items {
			if item.Status == domain.StatusDeleted || item.Status == domain.StatusCompleted {
				m.analyze.removeStage(item.Path)
			}
		}
		if m.analyze.filter == analyzeFilterQueued && len(m.analyze.stageOrder) == 0 {
			m.analyze.filter = analyzeFilterAll
		}
		m.analyze.syncPreviewWindow()
		if deleted > 0 || completed > 0 {
			m.errorMsg = ""
			m.clearNotice()
			return m.startAnalyzeReload("analysis refresh", false)
		}
		if protected > 0 || failed > 0 || skipped > 0 {
			m.errorMsg = ""
			m.setNotice(analyzeActionSummary(msg.result))
			return m, nil
		}
		m.errorMsg = ""
		m.clearNotice()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m appModel) View() string {
	var body string
	if m.planLoadActive() {
		body = m.transitionView()
	} else if m.route == RouteUninstall && m.uninstall.loading {
		// Show loading transition while inventory is loading
		m.loadingLabel = m.uninstall.loadingLabel
		body = m.transitionView()
	} else {
		switch m.route {
		case RouteHome:
			m.home.width, m.home.height = m.width, m.height
			body = m.home.View()
		case RouteClean:
			m.clean.width, m.clean.height = m.width, m.height
			body = m.clean.View()
		case RouteTools:
			m.tools.width, m.tools.height = m.width, m.height
			body = m.tools.View()
		case RouteProtect:
			m.protect.width, m.protect.height = m.width, m.height
			body = m.protect.View()
		case RouteUninstall:
			m.uninstall.width, m.uninstall.height = m.width, m.height
			body = m.uninstall.View()
		case RouteStatus:
			m.status.width, m.status.height = m.width, m.height
			body = m.status.View()
		case RouteDoctor:
			m.doctor.width, m.doctor.height = m.width, m.height
			body = m.doctor.View()
		case RouteAnalyze:
			m.analyze.width, m.analyze.height = m.width, m.height
			body = m.analyze.View()
		case RouteReview:
			m.review.width, m.review.height = m.width, m.height
			body = m.review.View()
		case RoutePreflight:
			m.preflight.width, m.preflight.height = m.width, m.height
			body = m.preflight.View()
		case RouteProgress:
			m.progress.width, m.progress.height = m.width, m.height
			m.progress.spinnerFrame = m.spinnerFrame
			body = m.progress.View()
		case RouteResult:
			m.result.width, m.result.height = m.width, m.height
			m.result.spinnerFrame = m.spinnerFrame
			body = m.result.View()
		default:
			body = "Unknown route"
		}
	}
	if m.helpVisible {
		body = m.helpOverlayView()
	}
	extras := []string{}
	if label := m.currentLoadingLabel(); label != "" && !m.planLoadActive() {
		extras = append(extras, renderInfoBar(m.width, loadingPulseLine(label, appMotionState(m))))
	}
	if m.errorMsg != "" {
		extras = append(extras, renderErrorBar(m.width, "Error: "+m.errorMsg))
	} else if m.noticeMsg != "" && !m.planLoadActive() && m.currentLoadingLabel() == "" {
		extras = append(extras, renderInfoBar(m.width, m.noticeMsg))
	}
	extras = append(extras, renderFooterBar(m.width, m.footerContent()))
	_, height := effectiveSize(m.width, m.height)
	availableBodyLines := height - len(extras)
	if availableBodyLines < 1 {
		availableBodyLines = 1
	}
	parts := []string{clipRendered(body, availableBodyLines)}
	parts = append(parts, extras...)
	return strings.Join(parts, "\n")
}

func (m appModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.shouldToggleHelp(msg) {
		m.helpVisible = !m.helpVisible
		return m, nil
	}
	if m.helpVisible {
		switch {
		case m.matchesBack(msg), m.matchesQuit(msg), m.matchesActivate(msg):
			m.helpVisible = false
		}
		return m, nil
	}
	if m.planLoadPending() {
		return m.handlePlanLoadKey(msg)
	}
	switch m.route {
	case RouteHome:
		return m.updateHome(msg)
	case RouteClean:
		return m.updateClean(msg)
	case RouteTools:
		return m.updateTools(msg)
	case RouteProtect:
		return m.updateProtect(msg)
	case RouteUninstall:
		return m.updateUninstall(msg)
	case RouteStatus:
		return m.updateStatus(msg)
	case RouteDoctor:
		return m.updateDoctor(msg)
	case RouteAnalyze:
		return m.updateAnalyze(msg)
	case RouteReview:
		return m.updateReview(msg)
	case RoutePreflight:
		return m.updatePreflight(msg)
	case RouteProgress:
		return m.updateProgress(msg)
	case RouteResult:
		return m.updateResult(msg)
	default:
		return m, tea.Quit
	}
}

func (m appModel) inputModeActive() bool {
	return m.protect.inputActive || m.uninstall.searchActive || m.analyze.searchActive
}

func (m appModel) shouldToggleHelp(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyF1 {
		if m.planLoadPending() && !m.helpVisible {
			return false
		}
		return true
	}
	if m.inputModeActive() {
		return false
	}
	if m.planLoadPending() && !m.helpVisible {
		return false
	}
	return m.matchesHelp(msg)
}

func (m appModel) updateProtect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.protect.inputActive {
		switch msg.String() {
		case "ctrl+c":
			return m.navigate(RouteHomeOrExit(m), m.hasHome)
		case "esc":
			m.protect.cancelInput()
			return m, nil
		case "enter":
			value := strings.TrimSpace(m.protect.input.Value())
			if value == "" {
				m.protect.setMessage("Enter a path to protect.", routeMessageTicks)
				return m, nil
			}
			// Validate path before adding
			validatedPath, isValid := validateProtectedPath(value)
			if !isValid {
				m.protect.setMessage(validatedPath, routeMessageTicks)
				return m, nil
			}
			m.loadingLabel = "protect path"
			return m, batchWithUITick(addProtectedPathCmd(m.callbacks.AddProtectedPath, validatedPath))
		}
		var cmd tea.Cmd
		m.protect.input, cmd = m.protect.input.Update(msg)
		return m, cmd
	}
	switch {
	case m.matchesQuit(msg), m.matchesBack(msg):
		return m.navigate(RouteTools, false)
	case m.matchesUp(msg):
		if m.protect.cursor > 0 {
			m.protect.cursor--
			m.refreshProtectExplanation(m.protect.selectedPath())
		}
	case m.matchesDown(msg):
		if m.protect.cursor < len(m.protect.paths)-1 {
			m.protect.cursor++
			m.refreshProtectExplanation(m.protect.selectedPath())
		}
	case m.matchesAdd(msg):
		m.protect.startInput()
	case m.matchesDelete(msg):
		path := m.protect.selectedPath()
		if path == "" {
			m.protect.setMessage("No protected path selected.", routeMessageTicks)
			return m, nil
		}
		m.loadingLabel = "remove protected path"
		return m, batchWithUITick(removeProtectedPathCmd(m.callbacks.RemoveProtectedPath, path))
	case m.matchesExplain(msg), m.matchesActivate(msg), m.matchesRefresh(msg):
		m.refreshProtectExplanation(m.protect.selectedPath())
	}
	return m, nil
}

func loadDashboardCmd(loader func() (DashboardData, error)) tea.Cmd {
	if loader == nil {
		return nil
	}
	return func() tea.Msg {
		data, err := loader()
		return dashboardLoadedMsg{data: data, err: err}
	}
}

func dashboardTickCmd() tea.Cmd {
	return tea.Tick(dashboardRefreshInterval, func(time.Time) tea.Msg {
		return dashboardTickMsg{}
	})
}

func uiTickCmd() tea.Cmd {
	return tea.Tick(uiTickInterval, func(time.Time) tea.Msg {
		return uiTickMsg{}
	})
}

func loadAppsCmd(loader func() ([]domain.AppEntry, error)) tea.Cmd {
	return loadAppsCmdWithRequest(loader, 0)
}

func loadAppsCmdWithRequest(loader func() ([]domain.AppEntry, error), requestID int) tea.Cmd {
	if loader == nil {
		return nil
	}
	return func() tea.Msg {
		apps, err := loader()
		return appsLoadedMsg{apps: apps, err: err, loadedAt: time.Now(), requestID: requestID}
	}
}

func loadCachedAppsCmd(loader func() ([]domain.AppEntry, time.Time, error)) tea.Cmd {
	return loadCachedAppsCmdWithRequest(loader, 0)
}

func loadCachedAppsCmdWithRequest(loader func() ([]domain.AppEntry, time.Time, error), requestID int) tea.Cmd {
	if loader == nil {
		return nil
	}
	return func() tea.Msg {
		apps, loadedAt, err := loader()
		return appsLoadedMsg{apps: apps, err: err, cached: true, loadedAt: loadedAt, requestID: requestID}
	}
}

func loadUninstallInventoryCmd(fresh func() ([]domain.AppEntry, error), cached func() ([]domain.AppEntry, time.Time, error), requestID int) tea.Cmd {
	if cached != nil {
		return tea.Batch(loadCachedAppsCmdWithRequest(cached, requestID), loadAppsCmdWithRequest(fresh, requestID))
	}
	return loadAppsCmdWithRequest(fresh, requestID)
}

func addProtectedPathCmd(mutate func(path string) (config.Config, string, error), path string) tea.Cmd {
	if mutate == nil {
		return nil
	}
	return func() tea.Msg {
		cfg, normalized, err := mutate(path)
		return protectAddedMsg{cfg: cfg, path: normalized, err: err}
	}
}

func removeProtectedPathCmd(mutate func(path string) (config.Config, string, bool, error), path string) tea.Cmd {
	if mutate == nil {
		return nil
	}
	return func() tea.Msg {
		cfg, normalized, removed, err := mutate(path)
		return protectRemovedMsg{cfg: cfg, path: normalized, removed: removed, err: err}
	}
}

func loadPlanCmd(loader func() (domain.ExecutionPlan, error)) tea.Cmd {
	if loader == nil {
		return nil
	}
	return func() tea.Msg {
		plan, err := loader()
		return planLoadedMsg{plan: plan, err: err}
	}
}

func tagPlanLoadCmd(cmd tea.Cmd, requestID int) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		loaded, ok := msg.(planLoadedMsg)
		if !ok {
			return msg
		}
		loaded.requestID = requestID
		return loaded
	}
}

func loadMenuPreviewCmd(route Route, key string, requestID int, loader func() (domain.ExecutionPlan, error)) tea.Cmd {
	if loader == nil {
		return nil
	}
	return func() tea.Msg {
		plan, err := loader()
		return menuPreviewLoadedMsg{route: route, key: key, plan: plan, err: err, requestID: requestID}
	}
}

func executePlanCmd(executor func(plan domain.ExecutionPlan) (domain.ExecutionResult, error), plan domain.ExecutionPlan) tea.Cmd {
	if executor == nil {
		return nil
	}
	return func() tea.Msg {
		result, err := executor(plan)
		return resultLoadedMsg{plan: plan, result: result, err: err}
	}
}

func trashAnalyzePathsCmd(executor func(paths []string) (domain.ExecutionResult, error), paths []string) tea.Cmd {
	if executor == nil {
		return nil
	}
	return func() tea.Msg {
		result, err := executor(paths)
		return analyzeActionFinishedMsg{result: result, err: err}
	}
}

func waitForExecutionStreamCmd(stream <-chan tea.Msg) tea.Cmd {
	if stream == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-stream
		if !ok {
			return nil
		}
		return msg
	}
}

func (m appModel) wantsUITick() bool {
	return m.currentLoadingLabel() != "" || m.route == RouteProgress || m.route == RouteResult ||
		m.route == RouteHome || m.route == RouteStatus || m.route == RouteDoctor
}

func formatDashboardSync(live *engine.SystemSnapshot) string {
	if live == nil || strings.TrimSpace(live.CollectedAt) == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, live.CollectedAt)
	if err != nil {
		return ""
	}
	return parsed.Local().Format("15:04:05")
}

func errorsIsCanceled(err error) bool {
	return errors.Is(err, context.Canceled)
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func loadingPulseSuffix(frame int, pulse bool) string {
	dots := []string{" .", " ..", " ..."}
	index := frame % len(dots)
	if pulse {
		index = (index + 1) % len(dots)
	}
	return dots[index]
}
