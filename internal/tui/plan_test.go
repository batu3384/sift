package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/platform"
	"github.com/batu3384/sift/internal/store"
)

func TestPlanViewContainsKeyDetails(t *testing.T) {
	t.Parallel()
	model := planModel{
		plan: domain.ExecutionPlan{
			Command:  "clean",
			Platform: "darwin",
			Totals:   domain.Totals{ItemCount: 2, Bytes: 3072, SafeBytes: 2048, ReviewBytes: 1024},
			Items: []domain.Finding{
				{
					DisplayPath: "/tmp/cache",
					Bytes:       2048,
					Risk:        domain.RiskSafe,
					Status:      domain.StatusPlanned,
					Category:    domain.CategorySystemClutter,
					Source:      "Chrome code cache",
				},
				{
					DisplayPath: "/tmp/log",
					Bytes:       1024,
					Risk:        domain.RiskReview,
					Status:      domain.StatusProtected,
					Category:    domain.CategoryLogs,
					Policy:      domain.PolicyDecision{Reason: domain.ProtectionProtectedPath},
					Source:      "Application logs",
				},
			},
		},
		requiresDecision: true,
	}
	view := model.View()
	if !strings.Contains(view, "CLEAN") || !strings.Contains(view, "darwin • review") || !strings.Contains(view, "/tmp/cache") || !strings.Contains(view, "SYSTEM CLUTTER") || !strings.Contains(view, "LOGS") || !strings.Contains(view, "Ready 1") || !strings.Contains(view, "Status") || !strings.Contains(view, "Scope") || !strings.Contains(view, "Next") || !strings.Contains(view, "Gate") || !strings.Contains(view, "Modules") || !strings.Contains(view, "Chrome code cache") || !strings.Contains(view, "REVIEW RAIL") || !strings.Contains(view, "FOCUS DECK") || !strings.Contains(view, "RUN GATE") {
		t.Fatalf("unexpected view output: %s", view)
	}
}

func TestPlanListLineUsesReadableNaturalOrder(t *testing.T) {
	t.Parallel()

	line := planListLine(domain.ExecutionPlan{Command: "clean"}, domain.Finding{
		DisplayPath: "/tmp/cache",
		Bytes:       2 * 1024 * 1024,
		Risk:        domain.RiskReview,
		Status:      domain.StatusProtected,
		Action:      domain.ActionSkip,
		Policy:      domain.PolicyDecision{Reason: domain.ProtectionProtectedPath},
	})

	for _, needle := range []string{"/tmp/cache", "excluded", "2.0 MB", "⊘", "protected_path"} {
		if !strings.Contains(line, needle) {
			t.Fatalf("expected %q in readable plan line, got %q", needle, line)
		}
	}
}

func TestPlanSummaryUsesReadableTopLine(t *testing.T) {
	t.Parallel()

	summary := planSummary(domain.ExecutionPlan{
		Totals: domain.Totals{
			SafeBytes:   2 * 1024,
			ReviewBytes: 1024,
		},
		Items: []domain.Finding{
			{Action: domain.ActionTrash, Status: domain.StatusPlanned, Risk: domain.RiskSafe, Bytes: 2 * 1024},
			{Action: domain.ActionAdvisory, Status: domain.StatusAdvisory, Risk: domain.RiskReview, Bytes: 4 * 1024},
			{Action: domain.ActionTrash, Status: domain.StatusPlanned, Risk: domain.RiskReview, Bytes: 1024},
			{Action: domain.ActionTrash, Status: domain.StatusProtected},
		},
	})

	for _, needle := range []string{"Ready 2", "Protected 1", "Safe 2.0 KB", "Review 1.0 KB"} {
		if !strings.Contains(summary, needle) {
			t.Fatalf("expected %q in readable plan summary, got %q", needle, summary)
		}
	}
}

func TestDecisionViewExplainsTrustScope(t *testing.T) {
	t.Parallel()

	plan := domain.ExecutionPlan{
		Command: "clean",
		Items: []domain.Finding{
			{ID: "ready", Path: "/tmp/ready", DisplayPath: "/tmp/ready", Action: domain.ActionTrash, Status: domain.StatusPlanned, Bytes: 1024, Risk: domain.RiskSafe},
			{ID: "held", Path: "/tmp/held", DisplayPath: "/tmp/held", Action: domain.ActionSkip, Status: domain.StatusSkipped, Bytes: 2048, Risk: domain.RiskReview},
			{ID: "protected", Path: "/tmp/protected", DisplayPath: "/tmp/protected", Action: domain.ActionTrash, Status: domain.StatusProtected, Bytes: 4096, Risk: domain.RiskReview, Policy: domain.PolicyDecision{Reason: domain.ProtectionProtectedPath}},
		},
	}

	view := decisionView(planModel{plan: plan, requiresDecision: true}, 120)
	for _, needle := range []string{
		"Will touch 1 approved item",
		"Safe    protected and excluded items stay out of this run",
		"Not touched  1 protected  •  1 excluded",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in trust-focused decision view, got:\n%s", needle, view)
		}
	}
}

func TestPlanDetailViewUsesReadableSelectedAndModuleLines(t *testing.T) {
	t.Parallel()

	model := planModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{
				{
					ID:          "a",
					Path:        "/tmp/cache-a",
					DisplayPath: "/tmp/cache-a",
					Bytes:       2 * 1024 * 1024,
					Risk:        domain.RiskReview,
					Status:      domain.StatusPlanned,
					Category:    domain.CategoryBrowserData,
					Action:      domain.ActionTrash,
					Source:      "Chrome code cache",
				},
				{
					ID:          "b",
					Path:        "/tmp/cache-b",
					DisplayPath: "/tmp/cache-b",
					Bytes:       1024,
					Risk:        domain.RiskReview,
					Status:      domain.StatusPlanned,
					Category:    domain.CategoryBrowserData,
					Action:      domain.ActionTrash,
					Source:      "Chrome code cache",
				},
			},
		},
		width:  132,
		height: 28,
	}

	view := planDetailView(model, 72, 20)
	for _, needle := range []string{"/tmp/cache-a", "2.0 MB", "Planned", "REVIEW", "trash", "Status   review item ready", "Scope    Chrome code cache", "2/2 included", "Next     space toggles this item", "m toggles current module", "From     Chrome code cache"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in review detail view, got %s", needle, view)
		}
	}
}

func TestPlanModelCanToggleCurrentModule(t *testing.T) {
	t.Parallel()

	model := planModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", DisplayPath: "/tmp/a", Action: domain.ActionTrash, Status: domain.StatusPlanned, Category: domain.CategoryBrowserData, Source: "Chrome code cache", Bytes: 1024},
				{ID: "b", Path: "/tmp/b", DisplayPath: "/tmp/b", Action: domain.ActionTrash, Status: domain.StatusPlanned, Category: domain.CategoryBrowserData, Source: "Chrome code cache", Bytes: 2048},
				{ID: "c", Path: "/tmp/c", DisplayPath: "/tmp/c", Action: domain.ActionTrash, Status: domain.StatusPlanned, Category: domain.CategoryLogs, Source: "Application logs", Bytes: 512},
			},
		},
	}

	model.toggleCurrentGroup()
	effective := model.effectivePlan()
	if effective.Items[0].Action != domain.ActionSkip || effective.Items[1].Action != domain.ActionSkip {
		t.Fatalf("expected current module items to be excluded, got %+v", effective.Items)
	}
	if effective.Items[2].Action == domain.ActionSkip {
		t.Fatalf("expected other modules to remain included, got %+v", effective.Items)
	}

	model.toggleCurrentGroup()
	effective = model.effectivePlan()
	if effective.Items[0].Action == domain.ActionSkip || effective.Items[1].Action == domain.ActionSkip {
		t.Fatalf("expected current module items to be restored, got %+v", effective.Items)
	}
}

func TestStatusViewContainsRecentScan(t *testing.T) {
	t.Parallel()
	model := statusModel{
		live: &engine.SystemSnapshot{
			HealthScore:           87,
			HealthLabel:           "healthy",
			PlatformFamily:        "macOS",
			Architecture:          "arm64",
			KernelVersion:         "24.3.0",
			BootTimeSeconds:       1700000000,
			HardwareModel:         "MacBookPro18,3",
			CPUModel:              "Apple M1 Pro",
			GPUModel:              "Apple GPU 18-core",
			GPUUsagePercent:       57,
			GPURendererPercent:    56,
			GPUTilerPercent:       54,
			DisplayResolution:     "2560 x 1440",
			DisplayRefreshRate:    "120Hz",
			DisplayCount:          2,
			BluetoothPowered:      true,
			BluetoothConnected:    1,
			BluetoothDevices:      []engine.BluetoothDeviceSnapshot{{Name: "AirPods Pro", Connected: true, Battery: "82%"}},
			ThermalState:          "warm",
			CPUTempCelsius:        30.6,
			FanSpeedRPM:           2384,
			SystemPowerWatts:      42,
			AdapterPowerWatts:     96,
			BatteryPowerWatts:     18,
			UptimeSeconds:         3600,
			CPUCores:              10,
			CPUPhysicalCores:      8,
			ProcessCount:          128,
			LoggedInUsers:         1,
			CPUPercent:            22.5,
			MemoryUsedPercent:     61.4,
			DiskFreeBytes:         5 * 1024 * 1024 * 1024,
			CPUPerCore:            []float64{12, 34},
			DiskIO:                &engine.DiskIOSnapshot{ReadBytes: 1024, WriteBytes: 2048},
			NetworkInterfaceCount: 3,
			ActiveNetworkIfaces:   []string{"en0", "utun4"},
			Battery:               &engine.BatterySnapshot{Percent: 84, State: "charging", RemainingMinutes: 45, Condition: "Normal", CycleCount: 142, CapacityPercent: 96},
			PowerSource:           "ac",
			OperatorAlerts:        []string{"thermal warm 30.6°C", "gpu load 57%"},
			Proxy:                 &engine.ProxySnapshot{Enabled: true, HTTP: "proxy.local:8080"},
			Highlights:            []string{"Top process: Code using 512.0 MB RSS"},
			TopProcesses: []engine.ProcessSnapshot{{
				Name:           "Code",
				CPUPercent:     12.4,
				MemoryPercent:  8.3,
				MemoryRSSBytes: 512 * 1024 * 1024,
			}},
		},
		scans: []store.RecentScan{{Command: "analyze", Profile: "safe"}},
		diagnostics: []platform.Diagnostic{
			{Name: "filevault", Status: "warn", Message: "disk encryption disabled"},
		},
		updateNotice: &engine.UpdateNotice{Available: true, LatestVersion: "v9.9.9", Message: "Update available"},
		lastExecution: &store.ExecutionSummary{
			Completed:        1,
			Deleted:          1,
			Warnings:         []string{"Rerun uninstall after the vendor uninstaller finishes."},
			FollowUpCommands: []string{`sift uninstall "Example"`},
		},
		networkRxRate: 2048,
		networkTxRate: 1024,
		diskReadRate:  4096,
		diskWriteRate: 2048,
		cpuTrend:      []float64{18, 22, 27, 22},
		memoryTrend:   []float64{58, 60, 61, 61.4},
		networkTrend:  []float64{1024, 2048, 3072},
		diskTrend:     []float64{1024, 2048, 4096},
		width:         160,
		height:        40,
	}
	view := model.View()
	for _, needle := range []string{
		"STATUS",
		"OBSERVATORY",
		"Observatory",
		"Status",
		"Watch",
		"alert load",
		"thermal warm 30.6°C",
		"Session",
		"87 / HEALTHY",
		"Battery 84% charging",
		"ARM64",
		"macOS",
		"8p/10l cores",
		"MacBookPro18,3",
		"Apple M1 Pro",
		"GPU 57% • render 56% • tiler 54%",
		"Bluetooth on",
		"Devices AirPods Pro (82%)",
		"Thermal warm • 30.6°C",
		"42W system",
		"30.6°C",
		"Pressure steady",
		"Top",
		"Processes 128",
		"STORAGE RAIL",
		"Disk 0.0% used",
		"Net rate",
		"Interfaces en0, utun4",
		"Battery 84% charging",
		"Proxy enabled=true",
		"ANALYZE / SAFE",
		"EXECUTION",
		"V9.9.9 ready",
		"Host",
		"Next",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in status view, got %s", needle, view)
		}
	}
}

func TestStatusHardwareSummaryIncludesGPUAndDisplay(t *testing.T) {
	t.Parallel()

	summary := statusHardwareSummary(&engine.SystemSnapshot{
		HardwareModel:      "MacBookPro18,3",
		CPUModel:           "Apple M1 Pro",
		GPUModel:           "Apple GPU 18-core",
		GPUUsagePercent:    57,
		DisplayResolution:  "2560 x 1440",
		DisplayRefreshRate: "120Hz",
		DisplayCount:       2,
		KernelVersion:      "24.3.0",
		BootTimeSeconds:    1700000000,
	})

	for _, needle := range []string{
		"MacBookPro18,3",
		"Apple M1 Pro",
		"gpu Apple GPU 18-core 57%",
		"display 2x 2560 x 1440 @120Hz",
		"kernel 24.3.0",
		"boot ",
	} {
		if !strings.Contains(summary, needle) {
			t.Fatalf("expected %q in hardware summary, got %q", needle, summary)
		}
	}
}

func TestStatusViewCompactsWithinSmallTerminal(t *testing.T) {
	t.Parallel()
	model := statusModel{
		live: &engine.SystemSnapshot{
			HealthScore:        87,
			HealthLabel:        "healthy",
			Architecture:       "arm64",
			CPUPercent:         14.2,
			MemoryUsedPercent:  61.4,
			DiskFreeBytes:      1024 * 1024 * 1024,
			DiskUsedPercent:    52.0,
			NetworkRxBytes:     4096,
			NetworkTxBytes:     2048,
			Highlights:         []string{"Top process: Code using 512 MB RSS"},
			Battery:            &engine.BatterySnapshot{Percent: 84, State: "charging", Condition: "Normal", CycleCount: 142},
			BluetoothPowered:   true,
			BluetoothConnected: 1,
			BluetoothDevices:   []engine.BluetoothDeviceSnapshot{{Name: "Keyboard", Connected: true}},
			CPUTempCelsius:     30.6,
			SystemPowerWatts:   42,
			AdapterPowerWatts:  96,
			BatteryPowerWatts:  18,
			FanSpeedRPM:        2384,
			GPUUsagePercent:    57,
			OperatorAlerts:     []string{"gpu load 57%"},
		},
		diagnostics: []platform.Diagnostic{
			{Name: "gatekeeper", Status: "warn", Message: "disabled"},
		},
		updateNotice: &engine.UpdateNotice{Available: true, LatestVersion: "v9.9.9", Message: "Update available"},
		width:        100,
		height:       24,
	}
	view := model.View()
	for _, needle := range []string{"STATUS", "OBSERVATORY", "LIVE RAIL", "SESSION RAIL"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in compact status view, got %s", needle, view)
		}
	}
	if !strings.Contains(view, "Pressure steady") {
		t.Fatalf("expected compact pressure line, got %s", view)
	}
	if !strings.Contains(view, "142 cycles") {
		t.Fatalf("expected compact battery health line, got %s", view)
	}
	if !strings.Contains(view, "42W system") {
		t.Fatalf("expected compact power summary, got %s", view)
	}
	if !strings.Contains(view, "Devices Keyboard") {
		t.Fatalf("expected compact bluetooth device summary, got %s", view)
	}
	if !strings.Contains(view, "18W battery") {
		t.Fatalf("expected compact battery power summary, got %s", view)
	}
	if !strings.Contains(view, "GPU 57%") {
		t.Fatalf("expected compact gpu summary, got %s", view)
	}
	if !strings.Contains(view, "Watch") {
		t.Fatalf("expected compact watch line, got %s", view)
	}
	if got := len(strings.Split(view, "\n")); got > model.height {
		t.Fatalf("expected compact status view to stay within %d lines, got %d", model.height, got)
	}
}

func TestStatusViewRenderMatrix(t *testing.T) {
	t.Parallel()

	base := statusModel{
		live: &engine.SystemSnapshot{
			HealthScore:           87,
			HealthLabel:           "healthy",
			PlatformFamily:        "macOS",
			Architecture:          "arm64",
			KernelVersion:         "24.3.0",
			BootTimeSeconds:       1700000000,
			HardwareModel:         "MacBookPro18,3",
			CPUModel:              "Apple M1 Pro",
			GPUModel:              "Apple GPU 18-core",
			GPUUsagePercent:       57,
			GPURendererPercent:    56,
			GPUTilerPercent:       54,
			DisplayResolution:     "2560 x 1440",
			DisplayRefreshRate:    "120Hz",
			DisplayCount:          2,
			BluetoothPowered:      true,
			BluetoothConnected:    1,
			BluetoothDevices:      []engine.BluetoothDeviceSnapshot{{Name: "AirPods Pro", Connected: true, Battery: "82%"}},
			ThermalState:          "warm",
			CPUTempCelsius:        30.6,
			FanSpeedRPM:           2384,
			SystemPowerWatts:      42,
			AdapterPowerWatts:     96,
			BatteryPowerWatts:     18,
			UptimeSeconds:         3600,
			CPUCores:              10,
			CPUPhysicalCores:      8,
			ProcessCount:          128,
			LoggedInUsers:         1,
			CPUPercent:            22.5,
			MemoryUsedPercent:     61.4,
			DiskFreeBytes:         5 * 1024 * 1024 * 1024,
			CPUPerCore:            []float64{12, 34},
			DiskIO:                &engine.DiskIOSnapshot{ReadBytes: 1024, WriteBytes: 2048},
			NetworkInterfaceCount: 3,
			ActiveNetworkIfaces:   []string{"en0", "utun4"},
			Battery:               &engine.BatterySnapshot{Percent: 84, State: "charging", RemainingMinutes: 45, Condition: "Normal", CycleCount: 142, CapacityPercent: 96},
			PowerSource:           "ac",
			OperatorAlerts:        []string{"thermal warm 30.6°C", "gpu load 57%"},
			Proxy:                 &engine.ProxySnapshot{Enabled: true, HTTP: "proxy.local:8080"},
			Highlights:            []string{"Top process: Code using 512.0 MB RSS"},
			TopProcesses: []engine.ProcessSnapshot{{
				Name:           "Code",
				CPUPercent:     12.4,
				MemoryPercent:  8.3,
				MemoryRSSBytes: 512 * 1024 * 1024,
			}},
		},
		scans: []store.RecentScan{{Command: "analyze", Profile: "safe"}},
		diagnostics: []platform.Diagnostic{
			{Name: "filevault", Status: "warn", Message: "disk encryption disabled"},
		},
		updateNotice: &engine.UpdateNotice{Available: true, LatestVersion: "v9.9.9", Message: "Update available"},
		lastExecution: &store.ExecutionSummary{
			Completed:        1,
			Deleted:          1,
			Warnings:         []string{"Rerun uninstall after the vendor uninstaller finishes."},
			FollowUpCommands: []string{`sift uninstall "Example"`},
		},
		networkRxRate: 2048,
		networkTxRate: 1024,
		diskReadRate:  4096,
		diskWriteRate: 2048,
		cpuTrend:      []float64{18, 22, 27, 22},
		memoryTrend:   []float64{58, 60, 61, 61.4},
		networkTrend:  []float64{1024, 2048, 3072},
		diskTrend:     []float64{1024, 2048, 4096},
	}

	for _, tc := range []struct {
		name    string
		width   int
		height  int
		needles []string
	}{
		{name: "80x24", width: 80, height: 24, needles: []string{"STATUS", "OBSERVATORY", "LIVE RAIL", "SESSION RAIL"}},
		{name: "100x30", width: 100, height: 30, needles: []string{"STATUS", "OBSERVATORY", "Status", "Watch", "Session", "Next", "LIVE RAIL"}},
		{name: "144x40", width: 144, height: 40, needles: []string{"STATUS", "OBSERVATORY", "Status", "Watch", "Host", "Next", "SESSION RAIL", "STORAGE RAIL"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := base
			model.width = tc.width
			model.height = tc.height
			view := model.View()
			for _, needle := range tc.needles {
				if !strings.Contains(view, needle) {
					t.Fatalf("expected %q in %s status view, got %s", needle, tc.name, view)
				}
			}
			if got := len(strings.Split(view, "\n")); got > model.height {
				t.Fatalf("expected %s status view to stay within %d lines, got %d", tc.name, model.height, got)
			}
		})
	}
}

func TestStatusModelToggleCompanionMode(t *testing.T) {
	t.Parallel()

	model := statusModel{}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	next := updated.(statusModel)
	if next.companionMode != "off" {
		t.Fatalf("expected companion mode off after first toggle, got %q", next.companionMode)
	}
	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	next = updated.(statusModel)
	if next.companionMode != "full" {
		t.Fatalf("expected companion mode full after second toggle, got %q", next.companionMode)
	}
}

func TestReviewAndResultViewsRenderWithinMatrix(t *testing.T) {
	t.Parallel()

	planModelBase := planModel{
		plan: domain.ExecutionPlan{
			Command:  "optimize",
			Platform: "darwin",
			Totals:   domain.Totals{ItemCount: 2, Bytes: 4096},
			Items: []domain.Finding{
				{
					ID:            "a",
					Path:          "/tmp/cache-a",
					DisplayPath:   "/tmp/cache-a",
					Action:        domain.ActionCommand,
					Status:        domain.StatusPlanned,
					Category:      domain.CategoryMaintenance,
					Name:          "Flush DNS cache",
					TaskPhase:     "preflight",
					TaskImpact:    "network reset",
					TaskVerify:    []string{"Run status again"},
					SuggestedBy:   []string{"firewall", "network"},
					RequiresAdmin: false,
				},
				{
					ID:          "b",
					Path:        "/tmp/cache-b",
					DisplayPath: "/tmp/cache-b",
					Action:      domain.ActionCommand,
					Status:      domain.StatusPlanned,
					Category:    domain.CategoryMaintenance,
					Name:        "Refresh LaunchServices",
					TaskPhase:   "aftercare",
					TaskImpact:  "refresh app registration",
				},
			},
		},
		requiresDecision: true,
	}

	resultModelBase := resultModel{
		result: domain.ExecutionResult{
			ScanID: "scan-1",
			Items: []domain.OperationResult{
				{FindingID: "a", Path: "/tmp/cache-a", Status: domain.StatusCompleted, Message: "command completed"},
				{FindingID: "b", Path: "/tmp/cache-b", Status: domain.StatusFailed, Message: "permission denied"},
			},
		},
		plan: planModelBase.plan,
	}

	cases := []struct {
		width  int
		height int
	}{
		{80, 24},
		{100, 30},
		{144, 40},
	}

	for _, tc := range cases {
		pm := planModelBase
		pm.width = tc.width
		pm.height = tc.height
		reviewView := pm.View()
		if !strings.Contains(strings.ToLower(reviewView), "optimize") {
			t.Fatalf("expected optimize heading in review view for %dx%d, got %s", tc.width, tc.height, reviewView)
		}
		if tc.width >= 140 && !strings.Contains(strings.ToLower(reviewView), "task board") {
			t.Fatalf("expected task board in widest review view for %dx%d, got %s", tc.width, tc.height, reviewView)
		}
		if got := len(strings.Split(reviewView, "\n")); got > tc.height {
			t.Fatalf("expected review view to fit within %d lines, got %d", tc.height, got)
		}

		rm := resultModelBase
		rm.width = tc.width
		rm.height = tc.height
		resultView := rm.View()
		if !strings.Contains(strings.ToLower(resultView), "result") {
			t.Fatalf("expected result heading in result view for %dx%d, got %s", tc.width, tc.height, resultView)
		}
		if tc.width >= 140 && !strings.Contains(strings.ToLower(resultView), "task board") {
			t.Fatalf("expected task board in widest result view for %dx%d, got %s", tc.width, tc.height, resultView)
		}
		if got := len(strings.Split(resultView, "\n")); got > tc.height {
			t.Fatalf("expected result view to fit within %d lines, got %d", tc.height, got)
		}
	}
}

func TestStatusViewEmptyStateMentionsHistory(t *testing.T) {
	t.Parallel()
	model := statusModel{
		live: &engine.SystemSnapshot{
			HealthScore:       72,
			HealthLabel:       "watch",
			CPUPercent:        11,
			MemoryUsedPercent: 40,
			DiskFreeBytes:     3 * 1024 * 1024 * 1024,
			CollectedAt:       time.Now().UTC().Format(time.RFC3339),
		},
		width:  132,
		height: 32,
	}
	view := model.View()
	if !strings.Contains(view, "OBSERVATORY") || !strings.Contains(view, "No scan history yet.") {
		t.Fatalf("unexpected empty status view: %s", view)
	}
}

func TestAnalyzeDetailViewMentionsFoldedDirectoryChains(t *testing.T) {
	t.Parallel()

	model := analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command: "analyze",
			Targets: []string{"/repo"},
			Items: []domain.Finding{
				{
					Name:        filepath.Join("apps", "web", "cache"),
					Path:        filepath.Join("/repo", "apps", "web", "cache"),
					DisplayPath: filepath.Join("/repo", "apps", "web", "cache"),
					Category:    domain.CategoryDiskUsage,
					Risk:        domain.RiskReview,
					Action:      domain.ActionAdvisory,
					Status:      domain.StatusAdvisory,
					Source:      "Immediate child of /repo • folded",
					Bytes:       2048,
					Fingerprint: domain.Fingerprint{Mode: uint32(os.ModeDir)},
				},
			},
		},
		width:  140,
		height: 32,
	}
	model.syncPreviewWindow()

	view := model.View()
	if !strings.Contains(view, "Preview") || !strings.Contains(view, "Focus  rank 1/1") {
		t.Fatalf("expected folded preview hint, got %s", view)
	}
}

func TestAnalyzeDetailViewShowsDirectoryPreviewCounts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.db"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "blob.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	model := analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command: "analyze",
			Targets: []string{root},
			Items: []domain.Finding{
				{
					Name:        "workspace",
					Path:        root,
					DisplayPath: root,
					Category:    domain.CategoryDiskUsage,
					Risk:        domain.RiskReview,
					Action:      domain.ActionAdvisory,
					Status:      domain.StatusAdvisory,
					Source:      "Immediate child of /repo",
					Bytes:       4096,
					Fingerprint: domain.Fingerprint{Mode: uint32(os.ModeDir)},
				},
			},
		},
		width:  140,
		height: 40,
	}
	model.syncPreviewWindow()

	view := model.View()
	for _, needle := range []string{
		"Children  3 total  •  1 dir  •  2 files",
		"Dirs cache",
		"Files blob.bin (7 B), index.db (7 B)",
		"Next blob.bin, cache, index.db",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in analyze preview, got %s", needle, view)
		}
	}
}

func TestAnalyzeDetailViewShowsCurrentViewRankAndPeers(t *testing.T) {
	t.Parallel()

	model := analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command: "analyze",
			Targets: []string{"/repo"},
			Items: []domain.Finding{
				{
					Name:        "alpha",
					Path:        "/repo/alpha",
					DisplayPath: "/repo/alpha",
					Category:    domain.CategoryDiskUsage,
					Risk:        domain.RiskReview,
					Action:      domain.ActionAdvisory,
					Status:      domain.StatusAdvisory,
					Source:      "Immediate child of /repo",
					Bytes:       9,
					Fingerprint: domain.Fingerprint{Mode: uint32(os.ModeDir)},
				},
				{
					Name:        "beta",
					Path:        "/repo/beta",
					DisplayPath: "/repo/beta",
					Category:    domain.CategoryDiskUsage,
					Risk:        domain.RiskReview,
					Action:      domain.ActionAdvisory,
					Status:      domain.StatusAdvisory,
					Source:      "Immediate child of /repo",
					Bytes:       5,
					Fingerprint: domain.Fingerprint{Mode: uint32(os.ModeDir)},
				},
				{
					Name:        "gamma",
					Path:        "/repo/gamma",
					DisplayPath: "/repo/gamma",
					Category:    domain.CategoryDiskUsage,
					Risk:        domain.RiskReview,
					Action:      domain.ActionAdvisory,
					Status:      domain.StatusAdvisory,
					Source:      "Immediate child of /repo",
					Bytes:       1,
					Fingerprint: domain.Fingerprint{Mode: uint32(os.ModeDir)},
				},
			},
		},
		cursor: 1,
		width:  140,
		height: 32,
	}
	model.syncPreviewWindow()

	view := model.View()
	for _, needle := range []string{"Focus", "Focus  rank 2/3", "Peers alpha (9 B)", "gamma (1 B)"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in analyze current-view context, got %s", needle, view)
		}
	}
}

func TestAnalyzeViewShowsSearchStateAndFiltersVisibleItems(t *testing.T) {
	t.Parallel()

	model := analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command: "analyze",
			Targets: []string{"/repo"},
			Items: []domain.Finding{
				{
					Name:        "chrome-cache",
					Path:        "/repo/chrome-cache",
					DisplayPath: "/repo/chrome-cache",
					Category:    domain.CategoryDiskUsage,
					Risk:        domain.RiskReview,
					Action:      domain.ActionAdvisory,
					Status:      domain.StatusAdvisory,
					Source:      "Immediate child of /repo",
					Bytes:       9,
					Fingerprint: domain.Fingerprint{Mode: uint32(os.ModeDir)},
				},
				{
					Name:        "slack-cache",
					Path:        "/repo/slack-cache",
					DisplayPath: "/repo/slack-cache",
					Category:    domain.CategoryDiskUsage,
					Risk:        domain.RiskReview,
					Action:      domain.ActionAdvisory,
					Status:      domain.StatusAdvisory,
					Source:      "Immediate child of /repo",
					Bytes:       5,
					Fingerprint: domain.Fingerprint{Mode: uint32(os.ModeDir)},
				},
			},
		},
		search:       newAnalyzeSearchInput(),
		searchActive: true,
		width:        140,
		height:       36,
	}
	model.search.SetValue("slack")
	model.syncPreviewWindow()

	view := model.View()
	if len(model.visibleIndices()) != 1 {
		t.Fatalf("expected one visible analyze result, got %d", len(model.visibleIndices()))
	}
	if !strings.Contains(view, "search> slack") {
		t.Fatalf("expected analyze search prompt, got %s", view)
	}
	if !strings.Contains(view, `Filter ALL  •  Search "slack"  •  1 visible`) {
		t.Fatalf("expected analyze search filter label, got %s", view)
	}
	if !strings.Contains(view, "slack-cache") {
		t.Fatalf("expected matching item to remain visible, got %s", view)
	}
	if !strings.Contains(view, "Focus  rank 1/1") {
		t.Fatalf("expected filtered current-view rank, got %s", view)
	}
	if strings.Contains(view, "Peers chrome-cache") {
		t.Fatalf("expected hidden peers to stay out of current-view context, got %s", view)
	}
}

func TestResultViewContainsSummaryAndReason(t *testing.T) {
	t.Parallel()
	model := resultModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{
				{ID: "native", Path: "/tmp/native", DisplayPath: "/tmp/native", Category: domain.CategoryMaintenance, Source: "vendor uninstall"},
				{ID: "trash-a", Path: "/tmp/a", DisplayPath: "/tmp/a", Category: domain.CategorySystemClutter, Source: "Application logs"},
				{ID: "protect-b", Path: "/tmp/b", DisplayPath: "/tmp/b", Category: domain.CategoryBrowserData, Source: "Chrome code cache", Bytes: 2 << 20},
			},
		},
		result: domain.ExecutionResult{
			Warnings:         []string{"Native uninstaller launched for Example. Rerun uninstall after it finishes to scan and clean remnants."},
			FollowUpCommands: []string{`sift uninstall "Example"`},
			Items: []domain.OperationResult{
				{FindingID: "native", Path: "/tmp/native", Status: domain.StatusCompleted, Message: "native uninstaller launched"},
				{FindingID: "trash-a", Path: "/tmp/a", Status: domain.StatusDeleted, Message: "moved to trash"},
				{FindingID: "protect-b", Path: "/tmp/b", Status: domain.StatusProtected, Reason: domain.ProtectionProtectedPath, Message: "Protected by policy."},
			},
		},
		cursor: 2,
		width:  160,
		height: 56,
	}
	view := model.View()
	for _, needle := range []string{"SETTLED RAIL", "OUTCOME DECK", "Warning", "Run", `sift uninstall "Example"`, "RECLAIM  2", "protected_path", "/tmp/b", "Recovery", "Current", "m opens current module", "Chrome code cache", "1 issue across 1 module", "Result", "Scope   Clean review", "Status   1 issue", "Rail    2 sections", "Next    m reopens current module"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in result view, got %s", needle, view)
		}
	}
	if !strings.Contains(view, "ALL • 2 sections • current + recovery") {
		t.Fatalf("unexpected result view: %s", view)
	}
}

func TestUninstallReviewViewContainsTargetBatchSummary(t *testing.T) {
	t.Parallel()

	model := planModel{
		plan: domain.ExecutionPlan{
			Command: "uninstall",
			Targets: []string{"Example", "Builder"},
			Totals:  domain.Totals{ItemCount: 3, Bytes: 3 << 20},
			Items: []domain.Finding{
				{ID: "native", Path: "/Applications/Example.app", DisplayPath: "Example", Category: domain.CategoryAppLeftovers, Action: domain.ActionNative, Status: domain.StatusPlanned},
				{ID: "trash-a", Path: "/tmp/example", DisplayPath: "/tmp/example", Category: domain.CategoryAppLeftovers, Action: domain.ActionTrash, Status: domain.StatusPlanned},
				{ID: "protect-b", Path: "/tmp/builder", DisplayPath: "/tmp/builder", Category: domain.CategoryAppLeftovers, Action: domain.ActionTrash, Status: domain.StatusProtected},
			},
		},
		requiresDecision: true,
		width:            160,
		height:           40,
	}

	view := model.View()
	for _, needle := range []string{"Target Batch", "2 apps  •  1 native step  •  2 remnants", "1 protected item", "Example", "Builder"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in uninstall review view, got %s", needle, view)
		}
	}
}

func TestUninstallDetailUsesTargetLanguage(t *testing.T) {
	t.Parallel()

	model := planModel{
		plan: domain.ExecutionPlan{
			Command: "uninstall",
			Targets: []string{"Example", "Builder"},
			Items: []domain.Finding{
				{ID: "native", Path: "/Applications/Example.app", DisplayPath: "Example", Category: domain.CategoryAppLeftovers, Action: domain.ActionNative, Status: domain.StatusPlanned, Source: "Example native uninstall"},
				{ID: "trash-a", Path: "/tmp/example", DisplayPath: "/tmp/example", Category: domain.CategoryAppLeftovers, Action: domain.ActionTrash, Status: domain.StatusPlanned, Source: "Example remnants"},
				{ID: "protect-b", Path: "/tmp/builder", DisplayPath: "/tmp/builder", Category: domain.CategoryAppLeftovers, Action: domain.ActionTrash, Status: domain.StatusProtected, Source: "Builder remnants"},
			},
		},
		width:  140,
		height: 28,
	}

	view := planDetailView(model, 80, 24)
	for _, needle := range []string{"Target", "Example native uninstall • 1/1 included • 0 B", "m toggles current target"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in uninstall detail view, got %s", needle, view)
		}
	}
}

func TestUninstallResultViewContainsTargetBatchSummary(t *testing.T) {
	t.Parallel()

	model := resultModel{
		plan: domain.ExecutionPlan{
			Command: "uninstall",
			Targets: []string{"Example", "Builder"},
			Totals:  domain.Totals{ItemCount: 3, Bytes: 3 << 20},
			Items: []domain.Finding{
				{ID: "native", Path: "/Applications/Example.app", DisplayPath: "Example", Category: domain.CategoryAppLeftovers, Action: domain.ActionNative, Status: domain.StatusPlanned},
				{ID: "trash-a", Path: "/tmp/example", DisplayPath: "/tmp/example", Category: domain.CategoryAppLeftovers, Action: domain.ActionTrash, Status: domain.StatusPlanned},
				{ID: "protect-b", Path: "/tmp/builder", DisplayPath: "/tmp/builder", Category: domain.CategoryAppLeftovers, Action: domain.ActionTrash, Status: domain.StatusProtected},
			},
		},
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{FindingID: "native", Path: "/Applications/Example.app", Status: domain.StatusCompleted, Message: "native uninstaller launched"},
				{FindingID: "trash-a", Path: "/tmp/example", Status: domain.StatusDeleted, Message: "moved to trash"},
				{FindingID: "protect-b", Path: "/tmp/builder", Status: domain.StatusProtected, Reason: domain.ProtectionProtectedPath, Message: "Protected by policy."},
			},
			FollowUpCommands: []string{`sift uninstall "Example"`, `sift uninstall "Builder"`},
		},
		width:  160,
		height: 56,
	}

	view := model.View()
	for _, needle := range []string{"Target Batch", "2 apps  •  1 native step  •  2 remnants", `sift uninstall "Builder"`} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in uninstall result view, got %s", needle, view)
		}
	}
}

func TestAnalyzePlanViewContainsInsights(t *testing.T) {
	t.Parallel()
	model := planModel{
		plan: domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Totals:   domain.Totals{ItemCount: 2, Bytes: 3 << 20, ReviewBytes: 3 << 20},
			Items: []domain.Finding{
				{
					DisplayPath: "/tmp/cache",
					Name:        "cache",
					Bytes:       2 << 20,
					Risk:        domain.RiskReview,
					Status:      domain.StatusAdvisory,
					Category:    domain.CategoryDiskUsage,
					Source:      "Immediate child of /tmp",
				},
				{
					DisplayPath: "/tmp/cache/big.bin",
					Name:        "big.bin",
					Bytes:       1 << 20,
					Risk:        domain.RiskReview,
					Status:      domain.StatusAdvisory,
					Category:    domain.CategoryLargeFiles,
					Source:      "/tmp",
				},
			},
		},
	}
	view := model.View()
	if !strings.Contains(view, "Summary  children 1  •  files 1") || !strings.Contains(view, "Top  child cache") || !strings.Contains(view, "LARGEST CHILDREN") || !strings.Contains(view, "LARGE FILES") {
		t.Fatalf("unexpected analyze view: %s", view)
	}
}

func TestAnalyzeBrowserCompactLayoutStaysWithinBounds(t *testing.T) {
	t.Parallel()
	model := analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items: []domain.Finding{
				{
					Name:         "cache",
					DisplayPath:  "/tmp/cache",
					Path:         "/tmp/cache",
					Bytes:        2 << 20,
					Risk:         domain.RiskReview,
					Status:       domain.StatusAdvisory,
					Category:     domain.CategoryDiskUsage,
					LastModified: time.Now().Add(-time.Hour),
				},
			},
		},
		history: []analyzeHistoryEntry{
			{plan: domain.ExecutionPlan{Targets: []string{"/tmp/root"}}},
		},
		staged: map[string]domain.Finding{},
		width:  100,
		height: 24,
	}
	model.syncPreviewWindow()
	view := model.View()
	for _, needle := range []string{"ANALYZE", "FILES", "DETAIL", "Path", "Back  esc", "History 1 level"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in compact analyze view, got %s", needle, view)
		}
	}
	if got := len(strings.Split(view, "\n")); got > model.height {
		t.Fatalf("expected compact analyze view to stay within %d lines, got %d", model.height, got)
	}

	model.width = 132
	model.height = 32
	wideView := model.View()
	for _, needle := range []string{"Preview", "Focus", "Next  x review selected", "space add"} {
		if !strings.Contains(wideView, needle) {
			t.Fatalf("expected %q in wide analyze view, got %s", needle, wideView)
		}
	}
	if strings.Contains(wideView, "Sort  staged order") {
		t.Fatalf("expected single-item analyze view to avoid batch queue panel, got %s", wideView)
	}
}

func TestAnalyzeViewRenderMatrix(t *testing.T) {
	t.Parallel()

	model := analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command:  "analyze",
			Platform: "darwin",
			Targets:  []string{"/tmp"},
			Items: []domain.Finding{
				{
					Name:         "cache",
					DisplayPath:  "/tmp/cache",
					Path:         "/tmp/cache",
					Bytes:        2 << 20,
					Risk:         domain.RiskReview,
					Status:       domain.StatusAdvisory,
					Category:     domain.CategoryDiskUsage,
					LastModified: time.Now().Add(-time.Hour),
				},
			},
		},
		history: []analyzeHistoryEntry{
			{plan: domain.ExecutionPlan{Targets: []string{"/tmp/root"}}},
		},
		staged: map[string]domain.Finding{},
	}
	model.syncPreviewWindow()

	for _, tc := range []struct {
		name    string
		width   int
		height  int
		needles []string
	}{
		{name: "80x24", width: 80, height: 24, needles: []string{"ANALYZE", "FILES", "DETAIL"}},
		{name: "100x30", width: 100, height: 30, needles: []string{"ANALYZE", "FILES", "DETAIL", "Path", "Back  esc"}},
		{name: "144x40", width: 144, height: 40, needles: []string{"ANALYZE", "FILES", "DETAIL", "Preview", "Next"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sized := model
			sized.width = tc.width
			sized.height = tc.height
			view := sized.View()
			for _, needle := range tc.needles {
				if !strings.Contains(view, needle) {
					t.Fatalf("expected %q in %s analyze view, got %s", needle, tc.name, view)
				}
			}
			if got := len(strings.Split(view, "\n")); got > sized.height {
				t.Fatalf("expected %s analyze view to stay within %d lines, got %d", tc.name, sized.height, got)
			}
		})
	}
}

func TestAnalyzePreviewSyncTracksSelectionWindow(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	left := filepath.Join(root, "alpha")
	right := filepath.Join(root, "beta")
	if err := os.MkdirAll(filepath.Join(left, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(right, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}

	model := analyzeBrowserModel{
		plan: domain.ExecutionPlan{
			Command: "analyze",
			Targets: []string{root},
			Items: []domain.Finding{
				{Name: "alpha", Path: left, DisplayPath: left, Category: domain.CategoryDiskUsage, Status: domain.StatusAdvisory, Fingerprint: domain.Fingerprint{Mode: uint32(os.ModeDir)}},
				{Name: "beta", Path: right, DisplayPath: right, Category: domain.CategoryDiskUsage, Status: domain.StatusAdvisory, Fingerprint: domain.Fingerprint{Mode: uint32(os.ModeDir)}},
			},
		},
		width:  140,
		height: 36,
	}

	model.syncPreviewWindow()
	if preview := model.selectedPreview(); preview.Path != left || preview.Dirs != 1 {
		t.Fatalf("expected left preview to be cached, got %+v", preview)
	}
	if _, ok := model.previewCache[right]; !ok {
		t.Fatalf("expected neighbor preview to be prefetched, got %+v", model.previewCache)
	}

	model.cursor = 1
	model.syncPreviewWindow()
	if preview := model.selectedPreview(); preview.Path != right || preview.Dirs != 1 {
		t.Fatalf("expected right preview after cursor move, got %+v", preview)
	}
}

func TestAnalyzeBrowserNavigatesIntoDirectoryAndBack(t *testing.T) {
	t.Parallel()
	root := domain.ExecutionPlan{
		Command:  "analyze",
		Platform: "darwin",
		Targets:  []string{"/tmp"},
		Items: []domain.Finding{
			{
				Name:        "cache",
				DisplayPath: "/tmp/cache",
				Path:        "/tmp/cache",
				Category:    domain.CategoryDiskUsage,
				Status:      domain.StatusAdvisory,
				Fingerprint: domain.Fingerprint{Mode: uint32(os.ModeDir)},
			},
		},
	}
	child := domain.ExecutionPlan{
		Command:  "analyze",
		Platform: "darwin",
		Targets:  []string{"/tmp/cache"},
		Items: []domain.Finding{
			{
				Name:        "nested.bin",
				DisplayPath: "/tmp/cache/nested.bin",
				Path:        "/tmp/cache/nested.bin",
				Category:    domain.CategoryLargeFiles,
				Status:      domain.StatusAdvisory,
			},
		},
	}
	model := analyzeBrowserModel{
		plan:   root,
		staged: map[string]domain.Finding{},
		loader: func(target string) (domain.ExecutionPlan, error) {
			if target != "/tmp/cache" {
				t.Fatalf("unexpected target %s", target)
			}
			return child, nil
		},
	}
	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected navigate command")
	}
	msg := cmd()
	resolved, cmd := next.(analyzeBrowserModel).Update(msg)
	if cmd != nil {
		t.Fatal("expected no follow-up command")
	}
	browsed := resolved.(analyzeBrowserModel)
	if len(browsed.history) != 1 || browsed.plan.Targets[0] != "/tmp/cache" {
		t.Fatalf("expected child plan with history, got %+v", browsed)
	}
	back, _ := browsed.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	restored := back.(analyzeBrowserModel)
	if len(restored.history) != 0 || restored.plan.Targets[0] != "/tmp" {
		t.Fatalf("expected root plan restored, got %+v", restored)
	}
}

func TestAnalyzeBrowserStagesItemsAndReturnsReviewPlan(t *testing.T) {
	t.Parallel()
	root := domain.ExecutionPlan{
		Command:  "analyze",
		Platform: "darwin",
		Targets:  []string{"/tmp"},
		Totals:   domain.Totals{Bytes: 2048},
		Items: []domain.Finding{
			{
				Name:        "cache",
				DisplayPath: "/tmp/cache",
				Path:        "/tmp/cache",
				Bytes:       2048,
				Category:    domain.CategoryDiskUsage,
				Status:      domain.StatusAdvisory,
				Source:      "Immediate child of /tmp",
			},
		},
	}
	review := domain.ExecutionPlan{
		Command: "clean",
		Targets: []string{"/tmp/cache"},
	}
	model := analyzeBrowserModel{
		plan:   root,
		staged: map[string]domain.Finding{},
		width:  132,
		height: 32,
		reviewLoader: func(paths []string) (domain.ExecutionPlan, error) {
			if len(paths) != 1 || paths[0] != "/tmp/cache" {
				t.Fatalf("unexpected staged paths: %+v", paths)
			}
			return review, nil
		},
	}
	model.syncPreviewWindow()
	next, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	staged := next.(analyzeBrowserModel)
	if len(staged.staged) != 1 {
		t.Fatalf("expected staged item, got %+v", staged.staged)
	}
	view := staged.View()
	for _, needle := range []string{"Impact", "100% of current reclaim", "State", "children 1", "x review selected"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in staged analyze view, got %s", needle, view)
		}
	}
	next, cmd := staged.Update(tea.KeyMsg{Runes: []rune{'x'}, Type: tea.KeyRunes})
	if cmd == nil {
		t.Fatal("expected review loader command")
	}
	msg := cmd()
	resolved, followUp := next.(analyzeBrowserModel).Update(msg)
	if followUp == nil {
		t.Fatal("expected quit command after review plan is ready")
	}
	final := resolved.(analyzeBrowserModel)
	if final.nextPlan == nil || final.nextPlan.Command != "clean" || len(final.nextPlan.Targets) != 1 || final.nextPlan.Targets[0] != "/tmp/cache" {
		t.Fatalf("expected staged review plan, got %+v", final.nextPlan)
	}
}

func TestProgressViewShowsCurrentStage(t *testing.T) {
	t.Parallel()
	model := progressModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{
				{
					Path:        "/tmp/cache-a",
					DisplayPath: "/tmp/cache-a",
					Category:    domain.CategoryBrowserData,
					Risk:        domain.RiskReview,
					Action:      domain.ActionTrash,
					Source:      "Chrome code cache",
				},
				{
					Path:        "/tmp/cache-b",
					DisplayPath: "/tmp/cache-b",
					Category:    domain.CategoryBrowserData,
					Risk:        domain.RiskReview,
					Action:      domain.ActionTrash,
					Source:      "Chrome code cache",
				},
				{
					Path:        "/tmp/pnpm",
					DisplayPath: "/tmp/pnpm",
					Category:    domain.CategoryPackageCaches,
					Risk:        domain.RiskSafe,
					Action:      domain.ActionTrash,
					Source:      "pnpm store",
				},
			},
		},
		items: []domain.OperationResult{
			{Path: "/tmp/cache-a", Status: domain.StatusDeleted},
		},
		cursor:       1,
		current:      &domain.Finding{Path: "/tmp/cache-b", DisplayPath: "/tmp/cache-b", Category: domain.CategoryBrowserData, Risk: domain.RiskReview, Source: "Chrome code cache"},
		currentPhase: domain.ProgressPhaseStarting,
		width:        132,
		height:       32,
	}
	view := model.View()
	for _, needle := range []string{"PROGRESS RAIL", "ACTION DECK", "BROWSER DATA", "Chrome code cache", "Progress", "33%", "1/3 settled", "Meter", "Phase", "RECLAIM", "Current", "moving item to trash", "Next", "pnpm store", "Status", "1 reclaimed", "Flow", "Now"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in progress view, got %s", needle, view)
		}
	}
}

func TestUninstallProgressViewShowsTargetFlow(t *testing.T) {
	t.Parallel()
	model := progressModel{
		plan: domain.ExecutionPlan{
			Command: "uninstall",
			Targets: []string{"Example", "Builder"},
			Items: []domain.Finding{
				{
					Path:        "/Applications/Example.app",
					DisplayPath: "Example",
					Category:    domain.CategoryAppLeftovers,
					Risk:        domain.RiskReview,
					Action:      domain.ActionNative,
					Source:      "Example native uninstall",
				},
				{
					Path:        "/tmp/example",
					DisplayPath: "/tmp/example",
					Category:    domain.CategoryAppLeftovers,
					Risk:        domain.RiskReview,
					Action:      domain.ActionTrash,
					Source:      "Example remnants",
				},
				{
					Path:        "/tmp/builder",
					DisplayPath: "/tmp/builder",
					Category:    domain.CategoryAppLeftovers,
					Risk:        domain.RiskReview,
					Action:      domain.ActionTrash,
					Source:      "Builder remnants",
				},
			},
		},
		items: []domain.OperationResult{
			{Path: "/Applications/Example.app", Status: domain.StatusCompleted},
		},
		cursor:       1,
		current:      &domain.Finding{Path: "/tmp/example", DisplayPath: "/tmp/example", Category: domain.CategoryAppLeftovers, Risk: domain.RiskReview, Source: "Example remnants"},
		currentPhase: domain.ProgressPhaseStarting,
		width:        132,
		height:       32,
	}
	view := model.View()
	for _, needle := range []string{"PROGRESS RAIL", "ACTION DECK", "Progress", "33%", "Phase", "REMNANT", "Current", "moving item to trash", "Next", "Builder remnants", "Status", "1 native", "Flow", "Example remnants"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in uninstall progress view, got %s", needle, view)
		}
	}
}

func TestOptimizeDetailViewShowsTaskBoardMetadata(t *testing.T) {
	t.Parallel()

	model := planModel{
		plan: domain.ExecutionPlan{
			Command: "optimize",
			Items: []domain.Finding{
				{
					ID:          "task-1",
					Name:        "Release inactive memory",
					DisplayPath: "/usr/bin/sudo /usr/bin/purge",
					Action:      domain.ActionCommand,
					Status:      domain.StatusPlanned,
					Category:    domain.CategoryMaintenance,
					Risk:        domain.RiskReview,
					Source:      "Runs purge to drop reclaimable inactive memory pages.",
					TaskPhase:   "repair",
					TaskImpact:  "Drops reclaimable inactive pages to relieve memory pressure.",
					TaskVerify:  []string{"Check memory pressure again in `sift status`"},
					SuggestedBy: []string{"Memory pressure", "Swap pressure"},
				},
			},
		},
		width:  132,
		height: 28,
	}

	view := planDetailView(model, 64, 20)
	for _, needle := range []string{"Task    REPAIR  •  Drops reclaimable inactive pages to relieve memory pressure.", "Status   task review ready", "Scope    Runs purge to drop reclaimable inactive memory pages. • 1/1 ready • 0 B", "Next     space toggles this task • m toggles current phase", "Suggested  Memory pressure, Swap pressure", "Verify", "Check memory pressure again in `sift status`", "Task Board", "1 task", "Flow"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in optimize detail view, got %s", needle, view)
		}
	}
	if subtitle := decisionSubtitle(model.plan); !strings.Contains(subtitle, "task") || !strings.Contains(subtitle, "suggested") || !strings.Contains(subtitle, "phase") {
		t.Fatalf("expected optimize decision subtitle to use task board language, got %q", subtitle)
	}
	decision := decisionView(model, 80)
	for _, needle := range []string{"Status   task gate ready", "Scope    Optimize", "Next     y runs maintenance + verify", "Gate     space toggles this task • m toggles current phase", "Task Board", "1 task", "Flow", "Phases", "1 phase"} {
		if !strings.Contains(decision, needle) {
			t.Fatalf("expected %q in optimize decision view, got %s", needle, decision)
		}
	}
}

func TestCleanDecisionViewShowsScopeAndOutcome(t *testing.T) {
	t.Parallel()

	model := planModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Profile: "safe",
			Totals:  domain.Totals{Bytes: 3 * 1024 * 1024, ItemCount: 2},
			Items: []domain.Finding{
				{ID: "a", DisplayPath: "/tmp/cache", Category: domain.CategoryTempFiles, Action: domain.ActionTrash, Status: domain.StatusPlanned},
				{ID: "b", DisplayPath: "/tmp/log", Category: domain.CategoryLogs, Action: domain.ActionTrash, Status: domain.StatusPlanned},
			},
		},
		width:  120,
		height: 28,
	}

	view := decisionView(model, 84)
	for _, needle := range []string{"Status   review gate ready", "Scope    Quick Clean", "2 modules", "3.0 MB", "Next     y runs cleanup • esc returns", "Gate     space toggles this item • m toggles current module"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean decision view, got %s", needle, view)
		}
	}
}

func TestAnalyzeDecisionViewUsesTraceAndReclaimLanguage(t *testing.T) {
	t.Parallel()

	model := planModel{
		plan: domain.ExecutionPlan{
			Command: "analyze",
			Totals:  domain.Totals{Bytes: 2 * 1024 * 1024},
			Items: []domain.Finding{
				{ID: "a", DisplayPath: "/tmp/cache", Category: domain.CategoryBrowserData, Action: domain.ActionTrash, Status: domain.StatusPlanned},
			},
		},
	}

	view := decisionView(model, 84)
	for _, needle := range []string{"Status   trace gate ready", "Scope    Staged Cleanup", "Next     y trashes staged paths", "Gate     space toggles this trace • m toggles current trace", "2.0 MB"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in analyze decision view, got %s", needle, view)
		}
	}
}

func TestDecisionViewShowsPermissionSummaryWhenNeeded(t *testing.T) {
	t.Parallel()

	model := planModel{
		plan: domain.ExecutionPlan{
			Command: "uninstall",
			Targets: []string{"Example"},
			Items: []domain.Finding{
				{Name: "Reset LaunchServices", Action: domain.ActionCommand, RequiresAdmin: true, CommandPath: "/usr/bin/sudo", Status: domain.StatusPlanned},
				{Name: "Finder prompt", Action: domain.ActionCommand, CommandPath: "/usr/bin/osascript", Status: domain.StatusPlanned},
				{Name: "Example", Action: domain.ActionNative, Status: domain.StatusPlanned},
			},
		},
		width:  120,
		height: 28,
	}

	view := decisionView(model, 84)
	for _, needle := range []string{
		"Status   handoff gate ready",
		"Gate     space toggles this target • m toggles current target",
		"Removal access  •  1 admin  •  1 dialog  •  1 native",
		"Need    Reset LaunchServices  •  Finder prompt  •  Example",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in decision view, got %s", needle, view)
		}
	}
}

func TestDecisionViewCollapsesSingleWarningLine(t *testing.T) {
	t.Parallel()

	view := decisionView(planModel{
		plan: domain.ExecutionPlan{
			Command:  "clean",
			Profile:  "safe",
			Totals:   domain.Totals{Bytes: 1024},
			Items:    []domain.Finding{{DisplayPath: "/tmp/cache", Action: domain.ActionTrash, Status: domain.StatusPlanned}},
			Warnings: []string{"Review cache ownership before cleanup."},
		},
	}, 84)

	if !strings.Contains(view, "Warning  Review cache ownership before cleanup.") {
		t.Fatalf("expected compact warning line, got %s", view)
	}
	if strings.Contains(view, "Warnings") {
		t.Fatalf("expected single warning to avoid warnings block, got %s", view)
	}
}

func TestInstallerAndPurgeBoardsRenderSpecializedSummaries(t *testing.T) {
	t.Parallel()

	installerPlan := domain.ExecutionPlan{
		Command: "installer",
		Totals:  domain.Totals{Bytes: 6 * 1024 * 1024},
		Items: []domain.Finding{
			{Path: "/Users/test/Downloads/sample.dmg", DisplayPath: "/Users/test/Downloads/sample.dmg", Category: domain.CategoryInstallerLeft, Status: domain.StatusPlanned, Action: domain.ActionTrash},
			{Path: "/Users/Shared/sample.pkg", DisplayPath: "/Users/Shared/sample.pkg", Category: domain.CategoryInstallerLeft, Status: domain.StatusPlanned, Action: domain.ActionTrash},
		},
	}
	installerView := decisionView(planModel{plan: installerPlan}, 80)
	for _, needle := range []string{"Installer Scope", "payloads", ".dmg", ".pkg"} {
		if !strings.Contains(strings.ToLower(installerView), strings.ToLower(needle)) {
			t.Fatalf("expected %q in installer board, got %s", needle, installerView)
		}
	}

	purgePlan := domain.ExecutionPlan{
		Command: "purge",
		Totals:  domain.Totals{Bytes: 2 * 1024 * 1024},
		Items: []domain.Finding{
			{Path: "/Users/test/dev/app/.turbo", DisplayPath: "/Users/test/dev/app/.turbo", Category: domain.CategoryProjectArtifacts, Risk: domain.RiskReview, Status: domain.StatusPlanned, Action: domain.ActionTrash},
			{Path: "/Users/test/dev/api/node_modules/.cache", DisplayPath: "/Users/test/dev/api/node_modules/.cache", Category: domain.CategoryDeveloperCaches, Risk: domain.RiskHigh, Status: domain.StatusPlanned, Action: domain.ActionTrash},
		},
	}
	purgeView := decisionView(planModel{plan: purgePlan}, 84)
	for _, needle := range []string{"Workspace Sweep", "root", "high-risk", "PROJECT ARTIFACTS"} {
		if !strings.Contains(strings.ToLower(purgeView), strings.ToLower(needle)) {
			t.Fatalf("expected %q in purge board, got %s", needle, purgeView)
		}
	}
}

func TestCommandBoardsRenderWithinResponsiveMatrix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		plan    domain.ExecutionPlan
		needles map[int][]string
	}{
		{
			name: "optimize",
			plan: domain.ExecutionPlan{
				Command: "optimize",
				Totals:  domain.Totals{Bytes: 128 * 1024 * 1024},
				Items: []domain.Finding{
					{
						ID:          "task-1",
						Name:        "Release inactive memory",
						DisplayPath: "/usr/bin/sudo /usr/bin/purge",
						Action:      domain.ActionCommand,
						Status:      domain.StatusPlanned,
						Category:    domain.CategoryMaintenance,
						Risk:        domain.RiskReview,
						TaskPhase:   "repair",
						TaskImpact:  "Drops reclaimable inactive pages to relieve memory pressure.",
						SuggestedBy: []string{"Memory pressure"},
					},
					{
						ID:         "task-2",
						Name:       "Refresh LaunchServices",
						Action:     domain.ActionCommand,
						Status:     domain.StatusPlanned,
						Category:   domain.CategoryMaintenance,
						Risk:       domain.RiskReview,
						TaskPhase:  "refresh",
						TaskImpact: "Refreshes the LaunchServices database after maintenance.",
					},
				},
			},
			needles: map[int][]string{
				80:  {"Task Board", "2 tasks"},
				100: {"Task Board", "2 tasks", "Flow"},
				144: {"Task Board", "2 tasks", "Flow"},
			},
		},
		{
			name: "autofix",
			plan: domain.ExecutionPlan{
				Command: "autofix",
				Totals:  domain.Totals{Bytes: 16 * 1024},
				Items: []domain.Finding{
					{
						ID:          "fix-1",
						Name:        "Enable firewall",
						DisplayPath: "/usr/libexec/ApplicationFirewall/socketfilterfw",
						Action:      domain.ActionCommand,
						Status:      domain.StatusPlanned,
						Category:    domain.CategoryMaintenance,
						Risk:        domain.RiskReview,
						TaskPhase:   "secure",
						TaskImpact:  "Turns on the macOS application firewall.",
						SuggestedBy: []string{"Firewall"},
					},
				},
			},
			needles: map[int][]string{
				80:  {"Fix Board", "1 task"},
				100: {"Fix Board", "1 task", "Flow"},
				144: {"Fix Board", "1 task", "Flow"},
			},
		},
		{
			name: "installer",
			plan: domain.ExecutionPlan{
				Command: "installer",
				Totals:  domain.Totals{Bytes: 6 * 1024 * 1024},
				Items: []domain.Finding{
					{Path: "/Users/test/Downloads/sample.dmg", DisplayPath: "/Users/test/Downloads/sample.dmg", Category: domain.CategoryInstallerLeft, Status: domain.StatusPlanned, Action: domain.ActionTrash},
					{Path: "/Users/Shared/sample.pkg", DisplayPath: "/Users/Shared/sample.pkg", Category: domain.CategoryInstallerLeft, Status: domain.StatusPlanned, Action: domain.ActionTrash},
				},
			},
			needles: map[int][]string{
				80:  {"Installer Scope", "payloads"},
				100: {"Installer Scope", "payloads", ".dmg"},
				144: {"Installer Scope", "payloads", ".dmg", ".pkg"},
			},
		},
		{
			name: "purge",
			plan: domain.ExecutionPlan{
				Command: "purge",
				Totals:  domain.Totals{Bytes: 2 * 1024 * 1024},
				Items: []domain.Finding{
					{Path: "/Users/test/dev/app/.turbo", DisplayPath: "/Users/test/dev/app/.turbo", Category: domain.CategoryProjectArtifacts, Risk: domain.RiskReview, Status: domain.StatusPlanned, Action: domain.ActionTrash},
					{Path: "/Users/test/dev/api/node_modules/.cache", DisplayPath: "/Users/test/dev/api/node_modules/.cache", Category: domain.CategoryDeveloperCaches, Risk: domain.RiskHigh, Status: domain.StatusPlanned, Action: domain.ActionTrash},
				},
			},
			needles: map[int][]string{
				80:  {"Workspace Sweep", "root"},
				100: {"Workspace Sweep", "root", "high-risk"},
				144: {"Workspace Sweep", "root", "high-risk", "PROJECT ARTIFACTS"},
			},
		},
	}

	widths := []int{80, 100, 144}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, width := range widths {
				width := width
				t.Run(fmt.Sprintf("%d", width), func(t *testing.T) {
					t.Parallel()
					model := planModel{plan: tc.plan}
					decision := decisionView(model, width)
					for _, needle := range tc.needles[width] {
						if !strings.Contains(strings.ToLower(decision), strings.ToLower(needle)) {
							t.Fatalf("expected %q in %s decision view at width %d, got %s", needle, tc.name, width, decision)
						}
					}
					detail := planDetailView(model, max(44, width/2), 20)
					if title := planCommandBoardTitle(tc.plan.Command); title != "" && !strings.Contains(detail, title) {
						t.Fatalf("expected %q in %s detail view at width %d, got %s", title, tc.name, width, detail)
					}
				})
			}
		})
	}
}
