package tui

import (
	"strings"
	"testing"

	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/engine"
	"github.com/batuhanyuksel/sift/internal/platform"
)

func TestSignalRailLabelForMotionUsesModeSpecificPrefixes(t *testing.T) {
	t.Parallel()

	alert := signalRailLabelForMotion(newMotionState(1, true, motionModeAlert, "alert", "monitor"))
	if !strings.Contains(alert, "A2 alert") {
		t.Fatalf("expected alert rail label, got %q", alert)
	}

	progress := signalRailLabelForMotion(newMotionState(2, false, motionModeProgress, "apply", "task"))
	if !strings.Contains(progress, "F3 apply") {
		t.Fatalf("expected progress rail label, got %q", progress)
	}
}

func TestAppMotionStatePrefersLoadingThenProgressThenAlerts(t *testing.T) {
	t.Parallel()

	model := appModel{loadingLabel: "dashboard", spinnerFrame: 2, livePulse: true}
	loading := appMotionState(model)
	if loading.Mode != motionModeLoading || loading.Phase != "monitor" || loading.Scene != "control" {
		t.Fatalf("expected loading motion state, got %+v", loading)
	}

	model.loadingLabel = ""
	model.route = RouteProgress
	model.progress = progressModel{
		plan:         domain.ExecutionPlan{Command: "clean"},
		currentPhase: domain.ProgressPhaseStarting,
		spinnerFrame: 3,
		pulse:        true,
	}
	progress := appMotionState(model)
	if progress.Mode != motionModeProgress || progress.Phase != "stage" {
		t.Fatalf("expected progress motion state, got %+v", progress)
	}

	model.route = RouteStatus
	model.progress = progressModel{}
	model.status = statusModel{
		live: &engine.SystemSnapshot{OperatorAlerts: []string{"thermal warm"}},
		diagnostics: []platform.Diagnostic{
			{Name: "filevault", Status: "warn", Message: "off"},
		},
		signalFrame: 1,
		pulse:       true,
	}
	alert := appMotionState(model)
	if alert.Mode != motionModeAlert || alert.Phase != "alert" {
		t.Fatalf("expected alert motion state, got %+v", alert)
	}
}

func TestLoadingPulseLineAndFooterMotionLabelIncludeCadence(t *testing.T) {
	t.Parallel()

	motion := newMotionState(1, true, motionModeLoading, "inspect", "sync")
	line := loadingPulseLine("refresh analysis view", motion)
	if !strings.Contains(line, "Refreshing") || !strings.Contains(line, "scanning files") || !strings.Contains(line, "reload folder -> refresh") || !strings.Contains(line, "inspect") {
		t.Fatalf("expected concise loading line, got %q", line)
	}

	footer := footerMotionLabel(newMotionState(0, false, motionModeAlert, "alert", "monitor"))
	if !strings.Contains(footer, "LIVE RAIL 15s") {
		t.Fatalf("expected footer motion label, got %q", footer)
	}
}

func TestLoadingSceneAndPhaseCoverExecutionAndProtect(t *testing.T) {
	t.Parallel()

	if scene := loadingScene("execution"); scene != "apply" {
		t.Fatalf("expected apply scene, got %q", scene)
	}
	if phase := loadingPhase("execution"); phase != "apply" {
		t.Fatalf("expected apply phase, got %q", phase)
	}
	if flow := loadingStageFlow("protect path"); flow != "save -> verify" {
		t.Fatalf("expected protect flow, got %q", flow)
	}
}

func TestLoadingStageScriptReflectsSceneAndPhase(t *testing.T) {
	t.Parallel()

	script := loadingStageScript("installed apps", newMotionState(0, true, motionModeLoading, "index", "inventory"))
	for _, needle := range []string{"index -> inspect", "index"} {
		if !strings.Contains(script, needle) {
			t.Fatalf("expected %q in loading stage script, got %q", needle, script)
		}
	}
}

func TestLoadingHelpersHumanizeReviewTransitions(t *testing.T) {
	t.Parallel()

	line := loadingPulseLine("failed item review", newMotionState(0, true, motionModeLoading, "review", "decision"))
	for _, needle := range []string{"Preparing failed item review", "load review -> inspect -> open", "review"} {
		if !strings.Contains(line, needle) {
			t.Fatalf("expected %q in review transition line, got %q", needle, line)
		}
	}

	if scene := loadingScene("current module recovery"); scene != "decision" {
		t.Fatalf("expected decision scene for recovery review, got %q", scene)
	}
	if phase := loadingPhase("quick clean review"); phase != "review" {
		t.Fatalf("expected review phase for clean review, got %q", phase)
	}
}

// TestDiagnosticsHaveIssuesIncludesErrorStatus verifies that error-status
// diagnostics trigger the alert motion mode, matching the broader fix that
// replaced diagnosticsHaveWarnings (warn-only) with diagnosticsHaveIssues.
func TestDiagnosticsHaveIssuesIncludesErrorStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		desc  string
		diags []platform.Diagnostic
		want  bool
	}{
		{"error triggers issues", []platform.Diagnostic{{Name: "filevault", Status: "error"}}, true},
		{"warn triggers issues", []platform.Diagnostic{{Name: "gatekeeper", Status: "warn"}}, true},
		{"ok does not trigger", []platform.Diagnostic{{Name: "sip", Status: "ok"}}, false},
		{"empty is false", nil, false},
	}
	for _, tc := range cases {
		got := diagnosticsHaveIssues(tc.diags)
		if got != tc.want {
			t.Errorf("%s: diagnosticsHaveIssues = %v, want %v", tc.desc, got, tc.want)
		}
	}

	// Error diagnostic must put statusMotionState into alert mode.
	errorModel := statusModel{
		diagnostics: []platform.Diagnostic{{Name: "filevault", Status: "error"}},
		signalFrame: 0,
	}
	motion := statusMotionState(errorModel)
	if motion.Mode != motionModeAlert {
		t.Errorf("statusMotionState with error diagnostic: expected alert mode, got %q", motion.Mode)
	}
}

func TestMotionSceneAtmosphereReflectsSceneAndMode(t *testing.T) {
	t.Parallel()

	idle := motionSceneAtmosphere(newMotionState(1, false, motionModeIdle, "steady", "control"))
	alert := motionSceneAtmosphere(newMotionState(2, true, motionModeAlert, "monitor", "monitor"))

	for _, needle := range []string{"⌂", "CONTROL FIELD", "STEADY WINDOW"} {
		if !strings.Contains(idle, needle) {
			t.Fatalf("expected %q in idle atmosphere, got %q", needle, idle)
		}
	}
	for _, needle := range []string{"◬", "MONITOR FIELD", "MONITOR WINDOW"} {
		if !strings.Contains(alert, needle) {
			t.Fatalf("expected %q in alert atmosphere, got %q", needle, alert)
		}
	}
	if idle == alert {
		t.Fatalf("expected atmosphere bands to vary across motion modes, got %q", idle)
	}
}

func TestMotionSceneGlyphVariesBySceneAndAlertState(t *testing.T) {
	t.Parallel()

	if glyph := motionSceneGlyph(newMotionState(0, false, motionModeIdle, "steady", "analyze")); glyph != "◧" {
		t.Fatalf("expected analyze glyph, got %q", glyph)
	}
	if glyph := motionSceneGlyph(newMotionState(0, true, motionModeAlert, "recover", "cleanup")); glyph != "◩" {
		t.Fatalf("expected alert cleanup glyph, got %q", glyph)
	}
	if glyph := motionSceneGlyph(newMotionState(0, false, motionModeProgress, "apply", "task")); glyph != "◆" {
		t.Fatalf("expected task glyph, got %q", glyph)
	}
}

func TestReducedMotionStateUsesStaticMotionArtifacts(t *testing.T) {
	t.Parallel()

	motion := reducedMotionState(newMotionState(3, true, motionModeLoading, "inspect", "analyze"))
	if glyph := spinnerGlyph(motion); glyph != "•" {
		t.Fatalf("expected reduced-motion spinner glyph, got %q", glyph)
	}
	if footer := footerMotionLabel(motion); footer != "live updates 15s" {
		t.Fatalf("expected reduced-motion footer label, got %q", footer)
	}
	line := loadingPulseLine("refresh analysis view", motion)
	for _, needle := range []string{"Refreshing scanning files", "reload folder -> refresh"} {
		if !strings.Contains(line, needle) {
			t.Fatalf("expected %q in reduced-motion loading line, got %q", needle, line)
		}
	}
	if strings.Contains(line, "⠋") || strings.Contains(line, "◉") {
		t.Fatalf("did not expect animated glyphs in reduced-motion loading line, got %q", line)
	}
}
