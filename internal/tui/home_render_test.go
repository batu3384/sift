package tui

import (
	"strings"
	"testing"

	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/platform"
	"github.com/batu3384/sift/internal/store"
)

func TestHomeOperatorLinePrioritizesUpdateWindow(t *testing.T) {
	t.Parallel()

	line := homeOperatorLine(
		[]homeAction{{ID: "status", Title: "Status", Command: "sift status", Enabled: true}},
		0,
		&engine.SystemSnapshot{HealthScore: 87, OperatorAlerts: []string{"thermal warm 61.5°C"}},
		&store.ExecutionSummary{Deleted: 2},
		[]platform.Diagnostic{{Name: "filevault", Status: "warn"}},
		&engine.UpdateNotice{Available: true, LatestVersion: "v9.9.9"},
	)

	for _, needle := range []string{"Recommended", "run sift update", "next run sift update"} {
		if !strings.Contains(line, needle) {
			t.Fatalf("expected %q in operator line, got %q", needle, line)
		}
	}
}

func TestHomeActionRailGuidesNextStep(t *testing.T) {
	t.Parallel()

	line := homeActionRail(
		[]homeAction{{ID: "status", Title: "Status", Command: "sift status", Enabled: true}},
		0,
		nil,
		nil,
		[]platform.Diagnostic{{Name: "filevault", Status: "warn"}},
		nil,
	)

	for _, needle := range []string{"More", "t opens check", "then run sift autofix"} {
		if !strings.Contains(line, needle) {
			t.Fatalf("expected %q in action rail, got %q", needle, line)
		}
	}
}

func TestHomeSessionAndStateRailsExposeOperationalContext(t *testing.T) {
	t.Parallel()

	session := homeSessionRailLine(nil, &store.ExecutionSummary{
		Completed: 5,
		Deleted:   3,
		Protected: 2,
		Skipped:   1,
	})
	for _, needle := range []string{"Last", "8 settled", "2 protected"} {
		if !strings.Contains(session, needle) {
			t.Fatalf("expected %q in session rail, got %q", needle, session)
		}
	}

	cfg := config.Default()
	cfg.ProtectedPaths = []string{"/tmp/a", "/tmp/b"}
	cfg.ProtectedFamilies = []string{"raycast"}
	cfg.CommandExcludes = map[string][]string{}
	cfg.CommandExcludes["clean"] = []string{"/tmp/cache"}
	cfg.PurgeSearchPaths = []string{"/tmp/work"}
	state := homeStateLine(cfg, []platform.Diagnostic{{Name: "gatekeeper", Status: "warn"}})
	for _, needle := range []string{"Setup", "2 protected paths", "1 family", "1 scope", "1 purge root", "1 issue queued"} {
		if !strings.Contains(state, needle) {
			t.Fatalf("expected %q in state rail, got %q", needle, state)
		}
	}
}

func TestHomeCompactLinesPrioritizeNextActionAndCarryState(t *testing.T) {
	t.Parallel()

	priority := homeCompactPriorityLine(
		[]homeAction{{ID: "status", Title: "Status", Command: "sift status", Enabled: true}},
		0,
		&engine.SystemSnapshot{OperatorAlerts: []string{"thermal warm 61.5°C"}},
		&store.ExecutionSummary{Deleted: 3},
		[]platform.Diagnostic{{Name: "gatekeeper", Status: "warn"}},
		&engine.UpdateNotice{Available: true, LatestVersion: "v9.9.9"},
	)
	for _, needle := range []string{"Recommended", "V9.9.9 ready", "next run sift update"} {
		if !strings.Contains(priority, needle) {
			t.Fatalf("expected %q in compact priority line, got %q", needle, priority)
		}
	}

	cfg := config.Default()
	cfg.CommandExcludes = map[string][]string{"clean": []string{"/tmp/cache"}}
	cfg.PurgeSearchPaths = []string{"/tmp/work"}
	carry := homeCompactCarryLine(&store.ExecutionSummary{Completed: 5, Deleted: 3, Protected: 2}, cfg, []platform.Diagnostic{{Name: "filevault", Status: "warn"}})
	for _, needle := range []string{"Setup", "8 settled", "1 scope", "1 purge root", "1 issue"} {
		if !strings.Contains(carry, needle) {
			t.Fatalf("expected %q in compact carry line, got %q", needle, carry)
		}
	}
}

func TestHomeSpotlightViewIncludesCoreSummary(t *testing.T) {
	t.Parallel()

	view := homeSpotlightView(
		[]homeAction{{ID: "status", Title: "Status", Command: "sift status", Enabled: true, Tone: "review"}},
		0,
		&engine.SystemSnapshot{HealthScore: 87, HealthLabel: "healthy"},
		nil,
		nil,
		nil,
		config.Default(),
		newMotionState(1, false, motionModeIdle, "steady", "control"),
		120,
		6,
	)

	for _, needle := range []string{"Signal", "ready", "Focus", "Alerts", "Activity", "Next"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in home spotlight view, got %q", needle, view)
		}
	}
}

func TestHomeDetailViewUsesNextGuardAndStateLines(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.ProtectedPaths = []string{"/tmp/a", "/tmp/b"}
	cfg.ProtectedFamilies = []string{"raycast"}
	cfg.CommandExcludes = map[string][]string{"clean": {"/tmp/cache"}}
	cfg.PurgeSearchPaths = []string{"/tmp/work"}

	view := homeDetailView(
		[]homeAction{{
			ID:          "optimize",
			Title:       "Optimize",
			Description: "Run safe maintenance and repair tasks.",
			Command:     "sift optimize",
			Safety:      "Maintenance tasks stay review-gated.",
			When:        "Use before deep cleanup or support work.",
			Tone:        "safe",
			Enabled:     true,
		}},
		0,
		nil,
		&store.ExecutionSummary{Completed: 4, Deleted: 2},
		[]platform.Diagnostic{{Name: "filevault", Status: "warn"}},
		cfg,
		90,
		16,
	)

	for _, needle := range []string{"Optimize", "Next", "enter opens review", "Guard", "State", "2 protected", "1 family", "1 scope", "1 purge root", "1 issue", "Last"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in home detail view, got %q", needle, view)
		}
	}
}

// TestDiagnosticIssueCountIncludesErrors verifies that error-status diagnostics
// are included in issue counts and trigger the correct alert and tactic copy.
// This guards against the silent bug where error diagnostics were invisible to
// the home/status alert logic (only "warn" was being counted).
func TestDiagnosticIssueCountIncludesErrors(t *testing.T) {
	t.Parallel()

	errorOnly := []platform.Diagnostic{{Name: "filevault", Status: "error"}}
	warnOnly := []platform.Diagnostic{{Name: "gatekeeper", Status: "warn"}}
	mixed := []platform.Diagnostic{
		{Name: "filevault", Status: "error"},
		{Name: "gatekeeper", Status: "warn"},
	}
	clean := []platform.Diagnostic{{Name: "sip", Status: "ok"}}

	cases := []struct {
		name  string
		diags []platform.Diagnostic
		want  int
	}{
		{"error counts as issue", errorOnly, 1},
		{"warn counts as issue", warnOnly, 1},
		{"mixed counts both", mixed, 2},
		{"ok is not an issue", clean, 0},
		{"empty is zero", nil, 0},
	}
	for _, tc := range cases {
		got := diagnosticIssueCount(tc.diags)
		if got != tc.want {
			t.Errorf("%s: diagnosticIssueCount = %d, want %d", tc.name, got, tc.want)
		}
	}

	// Verify that the alert line reflects error diagnostics.
	alertLine := homeAlertLine(nil, errorOnly, nil)
	if !strings.Contains(alertLine, "doctor issue") {
		t.Errorf("homeAlertLine with error diagnostic: expected 'doctor issue', got %q", alertLine)
	}
	steadyLine := homeAlertLine(nil, clean, nil)
	if !strings.Contains(steadyLine, "system steady") {
		t.Errorf("homeAlertLine with ok diagnostic: expected 'system steady', got %q", steadyLine)
	}

	// Verify recommended action fires for error diagnostics.
	rec := homeRecommendedAction(nil, -1, errorOnly, nil)
	if !strings.Contains(rec, "sift check") {
		t.Errorf("homeRecommendedAction with error diagnostic: expected 'sift check', got %q", rec)
	}
}

func TestAnalyzeStateLineReflectsQueueAndLoadingStates(t *testing.T) {
	t.Parallel()

	staged := analyzeAtmosphereLine(analyzeBrowserModel{
		stageOrder: []string{"/tmp/cache"},
	})
	loading := analyzeAtmosphereLine(analyzeBrowserModel{
		loading: true,
	})

	for _, needle := range []string{"State", "1 staged"} {
		if !strings.Contains(staged, needle) {
			t.Fatalf("expected %q in staged analyze atmosphere, got %q", needle, staged)
		}
	}
	for _, needle := range []string{"State", "scanning"} {
		if !strings.Contains(loading, needle) {
			t.Fatalf("expected %q in loading analyze atmosphere, got %q", needle, loading)
		}
	}
}

// TestHomeStatsAlertsCardCountsErrorAndWarnDiagnostics verifies that the
// "alerts" stat card in homeStats counts both error- and warn-status
// diagnostics, not just warn (the original bug counted warn only).
func TestHomeStatsAlertsCardCountsErrorAndWarnDiagnostics(t *testing.T) {
	t.Parallel()

	cards := homeStats(nil, nil, []platform.Diagnostic{
		{Name: "filevault", Status: "error"},
		{Name: "gatekeeper", Status: "warn"},
		{Name: "sip", Status: "ok"},
	}, nil, 140)

	// The last card is "alerts"; join all to search.
	combined := strings.Join(cards, " ")
	if !strings.Contains(combined, "2 active") {
		t.Errorf("homeStats alerts card: expected '2 active' (error+warn counted), got %q", combined)
	}
}

func TestAnalyzeActionRailGuidesStagingAndReview(t *testing.T) {
	t.Parallel()

	line := analyzeActionRail(analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Items: []domain.Finding{{
				Name:        "cache",
				Path:        "/tmp/cache",
				DisplayPath: "/tmp/cache",
				Category:    domain.CategoryDiskUsage,
				Status:      domain.StatusAdvisory,
				Fingerprint: domain.Fingerprint{Mode: 0o040000},
			}},
		},
		staged:     map[string]domain.Finding{"/tmp/cache": {Path: "/tmp/cache"}},
		stageOrder: []string{"/tmp/cache"},
	})

	for _, needle := range []string{"Next", "x review selected", "u unstage"} {
		if !strings.Contains(line, needle) {
			t.Fatalf("expected %q in analyze action rail, got %q", needle, line)
		}
	}
}
