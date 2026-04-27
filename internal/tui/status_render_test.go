package tui

import (
	"strings"
	"testing"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/platform"
	"github.com/batu3384/sift/internal/store"
)

func TestStatusHeroSceneLineReflectsAlertVector(t *testing.T) {
	t.Parallel()

	line := statusHeroSceneLine(statusModel{
		live: &engine.SystemSnapshot{
			CPUPercent:         22.5,
			MemoryUsedPercent:  61.4,
			GPUUsagePercent:    57,
			GPURendererPercent: 56,
			GPUTilerPercent:    54,
			CPUTempCelsius:     30.6,
			ThermalState:       "warm",
			SystemPowerWatts:   42,
			OperatorAlerts:     []string{"thermal warm 30.6°C"},
		},
		diagnostics: []platform.Diagnostic{{Name: "filevault", Status: "warn"}},
	}, 160)

	for _, needle := range []string{"Sensors", "alert load", "C▂ M▅ G▄ T▃", "thermal warm 30.6°C"} {
		if !strings.Contains(line, needle) {
			t.Fatalf("expected %q in hero scene line, got %q", needle, line)
		}
	}
}

func TestStatusHeroVectorLabelPrefersGraphicsWhenStable(t *testing.T) {
	t.Parallel()

	label := statusHeroVectorLabel(&engine.SystemSnapshot{
		GPUUsagePercent:    63,
		GPURendererPercent: 61,
		GPUTilerPercent:    54,
		MemoryUsedPercent:  40,
		DiskUsedPercent:    30,
	}, motionState{Mode: motionModeIdle})

	if label != "graphics load" {
		t.Fatalf("expected graphics vector, got %q", label)
	}
}

func TestStatusOverviewViewIncludesSummaryLines(t *testing.T) {
	t.Parallel()

	view := statusOverviewView(statusModel{
		live: &engine.SystemSnapshot{
			HealthScore:       87,
			HealthLabel:       "healthy",
			CPUPercent:        22.5,
			MemoryUsedPercent: 61.4,
			DiskFreeBytes:     5 * 1024 * 1024 * 1024,
			OperatorAlerts:    []string{"thermal warm 30.6°C"},
		},
		signalFrame: 1,
	}, 140, 8)

	for _, needle := range []string{"PULSE RAIL", "live observatory focus", "Observatory", "Status", "Watch", "Session", "Next", "companion"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in status overview, got %q", needle, view)
		}
	}
}

func TestStatusTacticLineGuidesDoctorWhenWarningsExist(t *testing.T) {
	t.Parallel()

	line := statusTacticLine(statusModel{
		diagnostics: []platform.Diagnostic{{Name: "filevault", Status: "warn"}},
	})

	for _, needle := range []string{"Recommended", "open doctor/check", "resolve posture drift"} {
		if !strings.Contains(line, needle) {
			t.Fatalf("expected %q in tactic line, got %q", needle, line)
		}
	}
}

func TestStatusSystemAndActivityListsUseReadableRows(t *testing.T) {
	t.Parallel()

	system := statusSystemView(&engine.SystemSnapshot{
		CPUPercent:        22.5,
		MemoryUsedPercent: 61.4,
		DiskUsedPercent:   52.0,
		DiskFreeBytes:     5 * 1024 * 1024 * 1024,
		ProcessCount:      128,
		LoggedInUsers:     1,
		TopProcesses: []engine.ProcessSnapshot{{
			Name:           "Code",
			CPUPercent:     12.4,
			MemoryPercent:  8.3,
			MemoryRSSBytes: 512 * 1024 * 1024,
		}},
	}, 120, 12)
	for _, needle := range []string{"Status", "Next", "Top", "Code • cpu 12.4%", "mem 8.3%", "rss 512.0 MB"} {
		if !strings.Contains(system, needle) {
			t.Fatalf("expected %q in system view, got %q", needle, system)
		}
	}

	activity := statusActivityView(
		[]store.RecentScan{{Command: "analyze", Profile: "safe", Totals: domain.Totals{ItemCount: 3, Bytes: 2048}}},
		&store.ExecutionSummary{Completed: 2, Deleted: 1},
		120,
		12,
	)
	for _, needle := range []string{"Status", "Next", "Recent", "History", "ANALYZE / SAFE • 2.0 KB • 3 items"} {
		if !strings.Contains(activity, needle) {
			t.Fatalf("expected %q in activity view, got %q", needle, activity)
		}
	}
}

// TestStatusAlertCardAndToneReflectErrors guards against the bug where
// error-status diagnostics were invisible to the alerts card (only "warn" was
// being counted), causing the card to say "clear" when errors were present.
func TestStatusAlertCardAndToneReflectErrors(t *testing.T) {
	t.Parallel()

	errorDiag := []platform.Diagnostic{{Name: "filevault", Status: "error"}}
	warnDiag := []platform.Diagnostic{{Name: "gatekeeper", Status: "warn"}}
	okDiag := []platform.Diagnostic{{Name: "sip", Status: "ok"}}

	// Both error and warn should appear as "issue" (singular) in the card when there is one.
	errCard := statusAlertCard(nil, errorDiag, nil)
	if !strings.Contains(errCard, "issue") {
		t.Errorf("statusAlertCard with error diag: expected 'issue', got %q", errCard)
	}
	warnCard := statusAlertCard(nil, warnDiag, nil)
	if !strings.Contains(warnCard, "issue") {
		t.Errorf("statusAlertCard with warn diag: expected 'issue', got %q", warnCard)
	}
	clearCard := statusAlertCard(nil, okDiag, nil)
	if clearCard != "clear" {
		t.Errorf("statusAlertCard with ok diag: expected 'clear', got %q", clearCard)
	}

	// Both error and warn should produce "high" tone.
	if tone := statusAlertTone(nil, errorDiag, nil); tone != "high" {
		t.Errorf("statusAlertTone with error diag: expected 'high', got %q", tone)
	}
	if tone := statusAlertTone(nil, warnDiag, nil); tone != "high" {
		t.Errorf("statusAlertTone with warn diag: expected 'high', got %q", tone)
	}
	if tone := statusAlertTone(nil, okDiag, nil); tone == "high" {
		t.Errorf("statusAlertTone with ok diag: should not be 'high', got %q", tone)
	}

	// statusTacticLine should guide doctor when errors exist (not just warns).
	errTactic := statusTacticLine(statusModel{diagnostics: errorDiag})
	if !strings.Contains(errTactic, "doctor") {
		t.Errorf("statusTacticLine with error diag: expected 'doctor', got %q", errTactic)
	}
}

func TestStatusCompanionLabelReflectsWatchAndMuteModes(t *testing.T) {
	t.Parallel()

	watch := statusCompanionLabel(statusModel{
		diagnostics: []platform.Diagnostic{{Name: "filevault", Status: "warn"}},
		signalFrame: 1,
	})
	for _, needle := range []string{"companion", "guard watch", "g mute"} {
		if !strings.Contains(watch, needle) {
			t.Fatalf("expected %q in companion label, got %q", needle, watch)
		}
	}

	muted := statusCompanionLabel(statusModel{companionMode: "off"})
	for _, needle := range []string{"companion muted", "g wake"} {
		if !strings.Contains(muted, needle) {
			t.Fatalf("expected %q in muted companion label, got %q", needle, muted)
		}
	}
}
