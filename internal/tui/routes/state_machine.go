// Package routes provides the AppPhase state machine for the SIFT TUI.
// This formalizes the informal pending* route fields into a proper state machine.
package routes

import (
	"github.com/batu3384/sift/internal/tui"
)

// AppPhase represents the current phase of the application.
type AppPhase string

const (
	PhaseHome      AppPhase = "home"
	PhaseLoading   AppPhase = "loading"
	PhaseScanning  AppPhase = "scanning"
	PhaseReview    AppPhase = "review"
	PhasePreflight AppPhase = "preflight"
	PhaseExecuting AppPhase = "executing"
	PhaseResult    AppPhase = "result"
	PhaseExit      AppPhase = "exit"
)

// AppState represents the complete application state machine.
type AppState struct {
	Phase       AppPhase
	Route       tui.Route
	SubRoute    string
	ReturnRoute tui.Route
	CanGoBack   bool
	History     []AppPhase
}

// NewAppState creates a new application state.
func NewAppState(initialRoute tui.Route) AppState {
	return AppState{
		Phase:       phaseFromRoute(initialRoute),
		Route:       initialRoute,
		CanGoBack:   initialRoute != tui.RouteHome,
		History:     []AppPhase{},
	}
}

// Transition moves the state machine to a new phase/route.
func (s *AppState) Transition(phase AppPhase, route tui.Route, returnRoute tui.Route) {
	if s.Phase != phase {
		s.History = append(s.History, s.Phase)
	}
	s.Phase = phase
	s.Route = route
	s.ReturnRoute = returnRoute
	s.CanGoBack = route != tui.RouteHome
}

// GoBack pops the history and returns to the previous state.
func (s *AppState) GoBack() bool {
	if len(s.History) == 0 {
		return false
	}
	last := s.History[len(s.History)-1]
	s.History = s.History[:len(s.History)-1]
	s.Phase = last
	return true
}

// CurrentPhase returns the current phase.
func (s AppState) CurrentPhase() AppPhase {
	return s.Phase
}

// IsLoading returns true if the state is in a loading phase.
func (s AppState) IsLoading() bool {
	return s.Phase == PhaseLoading || s.Phase == PhaseScanning
}

// CanNavigate returns true if navigation is allowed in the current phase.
func (s AppState) CanNavigate() bool {
	return s.Phase != PhaseExecuting && s.Phase != PhasePreflight
}

// phaseFromRoute determines the initial phase from a route.
func phaseFromRoute(route tui.Route) AppPhase {
	switch route {
	case tui.RouteHome:
		return PhaseHome
	case tui.RouteClean, tui.RouteAnalyze, tui.RouteUninstall:
		return PhaseScanning
	case tui.RouteReview:
		return PhaseReview
	case tui.RoutePreflight:
		return PhasePreflight
	case tui.RouteProgress:
		return PhaseExecuting
	case tui.RouteResult:
		return PhaseResult
	default:
		return PhaseHome
	}
}

// RouteFromPhase returns the appropriate route for a phase.
func RouteFromPhase(phase AppPhase) tui.Route {
	switch phase {
	case PhaseHome:
		return tui.RouteHome
	case PhaseScanning:
		return tui.RouteClean
	case PhaseReview:
		return tui.RouteReview
	case PhasePreflight:
		return tui.RoutePreflight
	case PhaseExecuting:
		return tui.RouteProgress
	case PhaseResult:
		return tui.RouteResult
	default:
		return tui.RouteHome
	}
}

// FlowPhase represents the phase within a flow (clean, analyze, uninstall).
type FlowPhase string

const (
	FlowPhaseIdle        FlowPhase = "idle"
	FlowPhaseScanning     FlowPhase = "scanning"
	FlowPhaseReviewReady  FlowPhase = "review_ready"
	FlowPhasePermissions  FlowPhase = "permissions"
	FlowPhaseReclaiming   FlowPhase = "reclaiming"
	FlowPhaseResult       FlowPhase = "result"
)

// String returns the string representation of a flow phase.
func (p FlowPhase) String() string {
	return string(p)
}

// IsTerminal returns true if this is a terminal phase.
func (p FlowPhase) IsTerminal() bool {
	return p == FlowPhaseResult
}

// CanTransitionTo returns true if this phase can transition to the target phase.
func (p FlowPhase) CanTransitionTo(target FlowPhase) bool {
	switch p {
	case FlowPhaseIdle:
		return target == FlowPhaseScanning
	case FlowPhaseScanning:
		return target == FlowPhaseReviewReady || target == FlowPhaseResult
	case FlowPhaseReviewReady:
		return target == FlowPhasePermissions || target == FlowPhaseScanning || target == FlowPhaseResult
	case FlowPhasePermissions:
		return target == FlowPhaseReclaiming || target == FlowPhaseReviewReady
	case FlowPhaseReclaiming:
		return target == FlowPhaseResult
	case FlowPhaseResult:
		return false // Terminal state
	default:
		return false
	}
}