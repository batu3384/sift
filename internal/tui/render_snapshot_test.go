package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/platform"
	"github.com/batu3384/sift/internal/store"
)

func snapshotNormalizeLines(rendered string) []string {
	raw := ansi.Strip(rendered)
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, strings.Join(strings.Fields(line), " "))
	}
	return out
}

func requireSnapshotPrefix(t *testing.T, rendered string, expected []string) {
	t.Helper()

	lines := snapshotNormalizeLines(rendered)
	if len(lines) < len(expected) {
		t.Fatalf("expected at least %d snapshot lines, got %d:\n%s", len(expected), len(lines), strings.Join(lines, "\n"))
	}
	for idx, want := range expected {
		if lines[idx] != want {
			t.Fatalf("snapshot mismatch at line %d\nwant: %q\ngot:  %q\n\nfull snapshot:\n%s", idx+1, want, lines[idx], strings.Join(lines, "\n"))
		}
	}
}

func TestHomeSpotlightSnapshot(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.ProtectedPaths = []string{"/tmp/protected"}
	cfg.ProtectedFamilies = []string{"raycast"}
	cfg.CommandExcludes = map[string][]string{"clean": {"/tmp/cache"}}
	cfg.PurgeSearchPaths = []string{"/tmp/work"}

	rendered := homeSpotlightView(
		[]homeAction{{ID: "analyze", Title: "Analyze", Command: "sift analyze", Tone: "review", Enabled: true}},
		0,
		&engine.SystemSnapshot{
			HealthScore:       87,
			HealthLabel:       "healthy",
			CPUPercent:        14.2,
			MemoryUsedPercent: 61.4,
			DiskFreeBytes:     1024 * 1024 * 1024,
			OperatorAlerts:    []string{"thermal warm 61.5°C"},
		},
		&store.ExecutionSummary{Completed: 2, Deleted: 1, Protected: 1, Skipped: 3},
		[]platform.Diagnostic{{Name: "gatekeeper", Status: "warn"}},
		&engine.UpdateNotice{Available: true, LatestVersion: "v9.9.9"},
		cfg,
		newMotionState(0, false, motionModeAlert, "alert", "control"),
		180,
		9,
	)

	requireSnapshotPrefix(t, rendered, []string{
		"Signal A1 alert • needs attention ╭─────╮",
		"Focus Analyze • sift analyze REVIEW │ ◈ ◈ │",
		"Alerts 1 doctor issue • V9.9.9 ready • thermal warm 61.5°C │ ∧ │",
		"Activity 2 completed • 1 deleted • 1 protected • 1 scope • 1 purge root ╰──┬──╯",
		"Live 87 (HEALTHY) • CPU 14.2% • Mem 61.4% • Free 1.0 GB │",
		"Next run sift update • t opens check ▃▁▄▁▃",
	})
}

func TestStatusOverviewSnapshot(t *testing.T) {
	t.Parallel()

	rendered := statusOverviewView(statusModel{
		live: &engine.SystemSnapshot{
			HealthScore:           87,
			HealthLabel:           "healthy",
			PlatformFamily:        "macOS",
			Architecture:          "arm64",
			CPUPhysicalCores:      8,
			CPUCores:              10,
			CPUPercent:            22.5,
			MemoryUsedPercent:     61.4,
			DiskFreeBytes:         5 * 1024 * 1024 * 1024,
			GPUUsagePercent:       57,
			GPURendererPercent:    56,
			GPUTilerPercent:       54,
			ThermalState:          "warm",
			CPUTempCelsius:        30.6,
			SystemPowerWatts:      42,
			AdapterPowerWatts:     96,
			BatteryPowerWatts:     18,
			Battery:               &engine.BatterySnapshot{Percent: 84, State: "charging", Condition: "Normal", CycleCount: 142, CapacityPercent: 96},
			PowerSource:           "ac",
			ActiveNetworkIfaces:   []string{"en0", "utun4"},
			NetworkInterfaceCount: 3,
			OperatorAlerts:        []string{"thermal warm 30.6°C"},
		},
		diagnostics:   []platform.Diagnostic{{Name: "filevault", Status: "warn"}},
		updateNotice:  &engine.UpdateNotice{Available: true, LatestVersion: "v9.9.9"},
		lastExecution: &store.ExecutionSummary{Deleted: 1},
		scans:         []store.RecentScan{{Command: "analyze", Profile: "safe"}},
		signalFrame:   0,
	}, 180, 12)

	requireSnapshotPrefix(t, rendered, []string{
		"Signal A1 alert • companion ✦ upgrade watch (g mute) ╭─────╮",
		"Now 87 / HEALTHY • Pressure steady • Battery 84% charging • 42W system • Thermal warm │ ◈ ◈ │",
		"Risk 1 doctor issue • V9.9.9 ready • thermal warm 30.6°C • alert load • 30.6°C • 42W system │ ∧ │",
		"Recent just now • 1 deleted • ANALYZE / SAFE • 0 B • just now ╰──┬──╯",
		"Host macOS • ARM64 • 8p/10l cores • Interfaces en0, utun4 • 3 total │",
		"Next apply update window • rerun doctor after upgrade ▄▂▅▂▄",
	})
}

func TestAnalyzeDetailSnapshot(t *testing.T) {
	t.Parallel()

	rendered := analyzeDetailView(analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command: "analyze",
			Targets: []string{"/tmp"},
			Items: []domain.Finding{
				{
					Name:        "cache",
					Path:        "/tmp/cache",
					DisplayPath: "/tmp/cache",
					Bytes:       2 << 20,
					Category:    domain.CategoryDiskUsage,
					Status:      domain.StatusAdvisory,
					Source:      "Immediate child of /tmp",
				},
			},
		},
		history:    []analyzeHistoryEntry{{plan: domain.ExecutionPlan{Targets: []string{"/tmp/root"}}}},
		staged:     map[string]domain.Finding{"/tmp/cache": {Name: "cache", Path: "/tmp/cache", DisplayPath: "/tmp/cache", Bytes: 2 << 20, Category: domain.CategoryDiskUsage}},
		stageOrder: []string{"/tmp/cache"},
		width:      160,
		height:     32,
	}, 160, 10)

	requireSnapshotPrefix(t, rendered, []string{
		"Path root → tmp",
		"Review 1 item • 2.0 MB • 1 module • staged order",
		"Focus BROWSE 1/1 • cache • 2.0 MB • ready",
		"State ALL • 1 visible • 1 staged • preview",
		"Next x review selected • u unstage",
	})
}

func TestProgressDetailSnapshot(t *testing.T) {
	t.Parallel()

	rendered := progressDetailView(progressModel{
		plan: domain.ExecutionPlan{
			Command: "optimize",
			Items: []domain.Finding{{
				Path:        "/tmp/optimize",
				DisplayPath: "/tmp/optimize",
				Bytes:       1024,
				Category:    domain.CategoryMaintenance,
				Action:      domain.ActionCommand,
			}},
		},
		current:      &domain.Finding{Path: "/tmp/optimize", DisplayPath: "/tmp/optimize", Category: domain.CategoryMaintenance, Action: domain.ActionCommand},
		currentPhase: domain.ProgressPhaseStarting,
		spinnerFrame: 0,
		pulse:        true,
		width:        160,
		height:       28,
	}, 160, 10)

	requireSnapshotPrefix(t, rendered, []string{
		"◌ APPLYING MAINTENANCE [░░░░░░░░░░░░] 0/1 • 0 B / 1.0 KB • stage 1/1 ╭─────╮",
		"Summary task 1/1 • 0/1 settled • 1 task • 1 phase │ ◉ ● │",
		"Status running task • /tmp/optimize • no completed operations yet │ ○ │",
		"Step maintenance • running task • /tmp/optimize ╰──┬──╯",
		"Freed 0 B / 1.0 KB │",
		"Done [····················] 0% ▁▂▃▂▁",
	})
}

func TestResultDetailSnapshot(t *testing.T) {
	t.Parallel()

	rendered := resultDetailView(resultModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Profile: "safe",
			Items: []domain.Finding{{
				ID:          "a",
				Path:        "/tmp/a",
				DisplayPath: "/tmp/a",
				Category:    domain.CategoryBrowserData,
			}},
		},
		result: domain.ExecutionResult{
			Warnings:         []string{"review"},
			FollowUpCommands: []string{"sift clean"},
			Items: []domain.OperationResult{{
				FindingID: "a",
				Path:      "/tmp/a",
				Status:    domain.StatusFailed,
				Message:   "permission denied",
			}},
		},
		width:  160,
		height: 28,
	}, 160, 10)

	requireSnapshotPrefix(t, rendered, []string{
		"Summary 1 item • 1 module • 1 issue • 1 warning • 1 follow-up command • 1 failed",
		"Scope Quick Clean • 1 module • 0 B",
		"Focus 1 failed • r retries failed first",
		"Outcome 0 sections settled • 0 reclaimed • 1 open",
		"Track 0 sections • 0 reclaimed • 1 open",
		"Next r retries failed • x reopens recovery batch",
		"────────────────────────────────────────────────",
		"Selected",
	})
}
