package routes

import (
	"testing"

	"github.com/batu3384/sift/internal/tui"
)

func TestNewAppStateDerivesInitialPhaseAndBackState(t *testing.T) {
	tests := []struct {
		name      string
		route     tui.Route
		wantPhase AppPhase
		wantBack  bool
	}{
		{name: "home", route: tui.RouteHome, wantPhase: PhaseHome, wantBack: false},
		{name: "clean", route: tui.RouteClean, wantPhase: PhaseScanning, wantBack: true},
		{name: "analyze", route: tui.RouteAnalyze, wantPhase: PhaseScanning, wantBack: true},
		{name: "uninstall", route: tui.RouteUninstall, wantPhase: PhaseScanning, wantBack: true},
		{name: "review", route: tui.RouteReview, wantPhase: PhaseReview, wantBack: true},
		{name: "preflight", route: tui.RoutePreflight, wantPhase: PhasePreflight, wantBack: true},
		{name: "progress", route: tui.RouteProgress, wantPhase: PhaseExecuting, wantBack: true},
		{name: "result", route: tui.RouteResult, wantPhase: PhaseResult, wantBack: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := NewAppState(tt.route)
			if state.Phase != tt.wantPhase {
				t.Fatalf("phase = %q, want %q", state.Phase, tt.wantPhase)
			}
			if state.Route != tt.route {
				t.Fatalf("route = %q, want %q", state.Route, tt.route)
			}
			if state.CanGoBack != tt.wantBack {
				t.Fatalf("CanGoBack = %v, want %v", state.CanGoBack, tt.wantBack)
			}
		})
	}
}

func TestAppStateTransitionHistoryAndNavigationGuards(t *testing.T) {
	state := NewAppState(tui.RouteHome)

	state.Transition(PhaseScanning, tui.RouteClean, tui.RouteHome)
	if got := state.History; len(got) != 1 || got[0] != PhaseHome {
		t.Fatalf("history after first transition = %#v, want [%q]", got, PhaseHome)
	}
	if !state.IsLoading() {
		t.Fatal("scanning phase should be treated as loading")
	}
	if !state.CanNavigate() {
		t.Fatal("scanning phase should allow navigation")
	}

	state.Transition(PhaseExecuting, tui.RouteProgress, tui.RouteClean)
	if state.CanNavigate() {
		t.Fatal("executing phase should block navigation")
	}
	if !state.GoBack() {
		t.Fatal("GoBack returned false with non-empty history")
	}
	if state.Phase != PhaseScanning {
		t.Fatalf("phase after GoBack = %q, want %q", state.Phase, PhaseScanning)
	}
}

func TestFlowPhaseTransitionRules(t *testing.T) {
	allowed := []struct {
		from FlowPhase
		to   FlowPhase
	}{
		{FlowPhaseIdle, FlowPhaseScanning},
		{FlowPhaseScanning, FlowPhaseReviewReady},
		{FlowPhaseScanning, FlowPhaseResult},
		{FlowPhaseReviewReady, FlowPhasePermissions},
		{FlowPhaseReviewReady, FlowPhaseScanning},
		{FlowPhaseReviewReady, FlowPhaseResult},
		{FlowPhasePermissions, FlowPhaseReclaiming},
		{FlowPhasePermissions, FlowPhaseReviewReady},
		{FlowPhaseReclaiming, FlowPhaseResult},
	}
	for _, tt := range allowed {
		if !tt.from.CanTransitionTo(tt.to) {
			t.Fatalf("%q should transition to %q", tt.from, tt.to)
		}
	}

	disallowed := []struct {
		from FlowPhase
		to   FlowPhase
	}{
		{FlowPhaseIdle, FlowPhaseResult},
		{FlowPhasePermissions, FlowPhaseResult},
		{FlowPhaseReclaiming, FlowPhaseScanning},
		{FlowPhaseResult, FlowPhaseIdle},
	}
	for _, tt := range disallowed {
		if tt.from.CanTransitionTo(tt.to) {
			t.Fatalf("%q should not transition to %q", tt.from, tt.to)
		}
	}
	if !FlowPhaseResult.IsTerminal() {
		t.Fatal("result phase should be terminal")
	}
}

func TestRouteFromPhaseMapsStableShellPhases(t *testing.T) {
	tests := []struct {
		phase AppPhase
		want  tui.Route
	}{
		{PhaseHome, tui.RouteHome},
		{PhaseScanning, tui.RouteClean},
		{PhaseReview, tui.RouteReview},
		{PhasePreflight, tui.RoutePreflight},
		{PhaseExecuting, tui.RouteProgress},
		{PhaseResult, tui.RouteResult},
		{PhaseExit, tui.RouteHome},
	}

	for _, tt := range tests {
		if got := RouteFromPhase(tt.phase); got != tt.want {
			t.Fatalf("RouteFromPhase(%q) = %q, want %q", tt.phase, got, tt.want)
		}
	}
}
