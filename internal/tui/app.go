package tui

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/platform"
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
	ReducedMotion bool
}

type AppCallbacks struct {
	LoadDashboard                       func() (DashboardData, error)
	LoadCachedInstalledApps             func() ([]domain.AppEntry, time.Time, error)
	LoadInstalledApps                   func() ([]domain.AppEntry, error)
	LoadAnalyzeHome                     func() (domain.ExecutionPlan, error)
	LoadAnalyzeTarget                   func(target string) (domain.ExecutionPlan, error)
	LoadAnalyzePreviews                 func(paths []string) map[string]domain.DirectoryPreview
	LoadCleanProfileWithProgress        func(profile string, emit func(ruleID string, ruleName string, itemsFound int, bytesFound int64)) (domain.ExecutionPlan, error)
	LoadCleanProfileWithFindingProgress func(profile string, emit func(ruleID string, ruleName string, item domain.Finding)) (domain.ExecutionPlan, error)
	LoadCleanProfile                    func(profile string) (domain.ExecutionPlan, error)
	LoadInstaller                       func() (domain.ExecutionPlan, error)
	LoadOptimize                        func() (domain.ExecutionPlan, error)
	LoadAutofix                         func() (domain.ExecutionPlan, error)
	LoadPurgeScan                       func() (domain.ExecutionPlan, error)
	LoadUninstallPlan                   func(app string) (domain.ExecutionPlan, error)
	LoadUninstallBatchPlan              func(apps []string) (domain.ExecutionPlan, error)
	LoadReviewForPaths                  func(paths []string) (domain.ExecutionPlan, error)
	AddProtectedPath                    func(path string) (config.Config, string, error)
	RemoveProtectedPath                 func(path string) (config.Config, string, bool, error)
	ExplainProtection                   func(path string) domain.ProtectionExplanation
	ExecutePlan                         func(plan domain.ExecutionPlan) (domain.ExecutionResult, error)
	ExecutePlanWithProgress             func(ctx context.Context, plan domain.ExecutionPlan, emit func(domain.ExecutionProgress)) (domain.ExecutionResult, error)
	OpenPath                            func(path string) error
	RevealPath                          func(path string) error
	TrashPaths                          func(paths []string) (domain.ExecutionResult, error)
	OnScanProgress                      func(ruleID string, ruleName string, itemsFound int, bytesFound int64)
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
	route      Route
	key        string
	ruleID     string
	ruleName   string
	itemsFound int
	bytesFound int64
	requestID  int
}

type scanFindingMsg struct {
	route     Route
	key       string
	ruleID    string
	ruleName  string
	item      domain.Finding
	requestID int
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
	cleanFlow                       cleanFlowModel
	tools                           menuModel
	protect                         protectModel
	uninstall                       uninstallModel
	uninstallFlow                   uninstallFlowModel
	status                          statusModel
	doctor                          doctorModel
	analyze                         analyzeBrowserModel
	analyzeFlow                     analyzeFlowModel
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
	reducedMotion                   bool
	nextPlanRequestID               int
	activePlanRequestID             int
	nextMenuPreviewRequestID        int
	activeCleanPreviewRequestID     int
	cleanPreviewStream              <-chan tea.Msg
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
	executionStream                 <-chan tea.Msg
	executionCancel                 context.CancelFunc
	activeExecutionSourceRoute      Route
	acceptedPermissionProfiles      map[string]struct{}
}

const dashboardRefreshInterval = 15 * time.Second
const uiTickInterval = 100 * time.Millisecond // 10 FPS for smoother animation

func RunApp(opts AppOptions, callbacks AppCallbacks) error {
	model := newAppModel(opts, callbacks)
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
		m.applyWindowSize(msg)
		return m, nil
	case dashboardLoadedMsg:
		return m.handleDashboardLoaded(msg)
	case dashboardTickMsg:
		return m.handleDashboardTick()
	case uiTickMsg:
		return m.handleUITick()
	case scanProgressMsg:
		return m.handleScanProgress(msg)
	case scanFindingMsg:
		return m.handleScanFinding(msg)
	case planLoadedMsg:
		return m.handlePlanLoaded(msg)
	case planLoadTransitionMsg:
		if msg.requestID == 0 || msg.requestID != m.activePlanRequestID || !m.planLoadPending() {
			return m, nil
		}
		m.planLoadTransitionVisible = true
		return m, nil
	case menuPreviewLoadedMsg:
		return m.handleMenuPreviewLoaded(msg)
	case resultLoadedMsg:
		return m.handleResultLoaded(msg)
	case executionProgressMsg:
		return m.handleExecutionProgress(msg)
	case permissionWarmupFinishedMsg:
		return m.handlePermissionWarmupFinished(msg)
	case executionFinishedMsg:
		return m.handleExecutionFinished(msg)
	case appsLoadedMsg:
		return m.handleAppsLoaded(msg)
	case protectAddedMsg:
		return m.handleProtectAdded(msg)
	case protectRemovedMsg:
		return m.handleProtectRemoved(msg)
	case analyzeActionFinishedMsg:
		return m.handleAnalyzeActionFinished(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m appModel) View() string {
	m.syncMotionSettings()
	var body string
	if m.planLoadActive() {
		body = m.transitionView()
	} else if m.route == RouteUninstall && m.uninstall.loading {
		// Show loading transition while inventory is loading
		m.loadingLabel = m.uninstall.loadingLabel
		body = m.transitionView()
	} else {
		body = m.routeView()
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

func loadAppsCmdWithRequest(loader func() ([]domain.AppEntry, error), requestID int) tea.Cmd {
	if loader == nil {
		return nil
	}
	return func() tea.Msg {
		apps, err := loader()
		return appsLoadedMsg{apps: apps, err: err, loadedAt: time.Now(), requestID: requestID}
	}
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

func waitForPreviewStreamCmd(stream <-chan tea.Msg) tea.Cmd {
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
	if m.reducedMotion {
		return m.noticeTicks > 0 || m.uninstall.messageTicks > 0 || m.protect.messageTicks > 0
	}
	return m.currentLoadingLabel() != "" || m.route == RouteProgress || m.route == RouteResult ||
		m.route == RouteHome || m.route == RouteStatus || m.route == RouteDoctor ||
		(m.route == RouteClean && m.cleanFlow.wantsAnimation()) ||
		(m.route == RouteUninstall && m.uninstallFlow.wantsAnimation()) ||
		(m.route == RouteAnalyze && m.analyzeFlow.wantsAnimation())
}

func (m *appModel) syncMotionSettings() {
	m.home.reducedMotion = m.reducedMotion
	m.home.pulse = m.livePulse
	m.home.signalFrame = m.spinnerFrame
	m.cleanFlow.reducedMotion = m.reducedMotion
	m.cleanFlow.pulse = m.livePulse
	if m.reducedMotion {
		m.cleanFlow.spinnerFrame = 0
	} else {
		m.cleanFlow.spinnerFrame = m.spinnerFrame
	}
	m.uninstallFlow.reducedMotion = m.reducedMotion
	m.uninstallFlow.pulse = m.livePulse
	if m.reducedMotion {
		m.uninstallFlow.spinnerFrame = 0
	} else {
		m.uninstallFlow.spinnerFrame = m.spinnerFrame
	}
	m.analyzeFlow.reducedMotion = m.reducedMotion
	m.analyzeFlow.pulse = m.livePulse
	if m.reducedMotion {
		m.analyzeFlow.spinnerFrame = 0
	} else {
		m.analyzeFlow.spinnerFrame = m.spinnerFrame
	}
	m.status.reducedMotion = m.reducedMotion
	m.status.pulse = m.livePulse
	m.status.signalFrame = m.spinnerFrame
	m.progress.reducedMotion = m.reducedMotion
	m.progress.pulse = m.livePulse && !m.reducedMotion
	if m.reducedMotion {
		m.progress.spinnerFrame = 0
	} else {
		m.progress.spinnerFrame = m.spinnerFrame
	}
	m.result.reducedMotion = m.reducedMotion
	if m.reducedMotion {
		m.result.spinnerFrame = 0
	} else {
		m.result.spinnerFrame = m.spinnerFrame
	}
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

// ASCII spinner frames for smooth animation
var spinnerFrames = []string{"/", "-", "\\", "|", "/", "-", "\\", "|"}

func loadingPulseSuffix(frame int, pulse bool) string {
	dots := []string{" .", " ..", " ..."}
	index := frame % len(dots)
	if pulse {
		index = (index + 1) % len(dots)
	}
	return dots[index]
}
