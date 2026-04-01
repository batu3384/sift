package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/store"
)

func TestPrintStatusReportIncludesOperatorAlerts(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := printStatusReport(&out, engine.StatusReport{
		Live: &engine.SystemSnapshot{
			Platform:          "darwin",
			PlatformVersion:   "15.0",
			Hostname:          "batuhan",
			HealthScore:       82,
			HealthLabel:       "watch",
			UptimeSeconds:     3600,
			ProcessCount:      128,
			LoggedInUsers:     1,
			CPUPercent:        21.4,
			MemoryUsedBytes:   8 << 30,
			MemoryTotalBytes:  16 << 30,
			MemoryUsedPercent: 50,
			DiskFreeBytes:     32 << 30,
			DiskTotalBytes:    128 << 30,
			OperatorAlerts:    []string{"thermal warm 61.5°C", "gpu load 78%"},
		},
		StatusSummary: store.StatusSummary{
			RecentScans: []store.RecentScan{{
				Command: "analyze",
				Profile: "safe",
			}},
			AuditLogPath: "/tmp/audit.log",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	view := out.String()
	for _, needle := range []string{
		"System: darwin 15.0 on batuhan",
		"Operator alerts: thermal warm 61.5°C | gpu load 78%",
		"analyze    safe",
		"Audit log: /tmp/audit.log",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in plain status report, got %s", needle, view)
		}
	}
}

func TestDoctorHelpMentionsUpstreamBaseline(t *testing.T) {
	t.Parallel()

	root := NewRootCommand()
	doctorCmd, _, err := root.Find([]string{"doctor"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doctorCmd.Long, "upstream Mole baseline compare range") {
		t.Fatalf("expected doctor long help to mention upstream baseline compare range, got %q", doctorCmd.Long)
	}
	if !strings.Contains(doctorCmd.Example, "upstream_baseline") {
		t.Fatalf("expected doctor examples to mention upstream_baseline, got %q", doctorCmd.Example)
	}
}

func TestDoctorCommandJSONIncludesParityAndUpstreamBaseline(t *testing.T) {
	prepareCLIEnv(t)

	out, err := executeRootCommand(t, "doctor", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var diagnostics []map[string]any
	if err := json.Unmarshal([]byte(out), &diagnostics); err != nil {
		t.Fatalf("expected JSON diagnostics, got %q: %v", out, err)
	}
	seen := map[string]bool{}
	for _, diagnostic := range diagnostics {
		if name, ok := diagnostic["name"].(string); ok {
			seen[name] = true
		}
	}
	for _, name := range []string{"parity_matrix", "upstream_baseline"} {
		if !seen[name] {
			t.Fatalf("expected %q in doctor JSON diagnostics, got %v", name, seen)
		}
	}
}
