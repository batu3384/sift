package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/platform"
)

func TestIsInteractiveTerminalHonorsEnvOverrides(t *testing.T) {
	t.Setenv("SIFT_NO_TUI", "1")
	t.Setenv("SIFT_FORCE_TUI", "")
	if isInteractiveTerminal() {
		t.Fatal("expected SIFT_NO_TUI to force non-interactive mode")
	}

	t.Setenv("SIFT_NO_TUI", "")
	t.Setenv("SIFT_FORCE_TUI", "1")
	if !isInteractiveTerminal() {
		t.Fatal("expected SIFT_FORCE_TUI to force interactive mode")
	}
}

func TestWantsJSONOutputHonorsExplicitFlags(t *testing.T) {
	t.Parallel()

	state := &runtimeState{}
	var out bytes.Buffer

	state.flags.JSON = true
	if !state.wantsJSONOutput("clean", &out) {
		t.Fatal("expected --json flag to force JSON output")
	}

	state.flags.JSON = false
	state.flags.Plain = true
	if state.wantsJSONOutput("status", &out) {
		t.Fatal("expected --plain to disable JSON output")
	}
}

func TestActionableItemCountSkipsAdvisoryAndProtected(t *testing.T) {
	t.Parallel()

	plan := domain.ExecutionPlan{
		Items: []domain.Finding{
			{Status: domain.StatusProtected, Action: domain.ActionTrash},
			{Status: domain.StatusAdvisory, Action: domain.ActionAdvisory},
			{Status: domain.StatusPlanned, Action: domain.ActionTrash},
			{Status: domain.StatusPlanned, Action: domain.ActionCommand},
		},
	}
	if got := actionableItemCount(plan); got != 2 {
		t.Fatalf("expected 2 actionable items, got %d", got)
	}
}

func TestFormatFloatSeriesTruncatesAndRounds(t *testing.T) {
	t.Parallel()

	got := formatFloatSeries([]float64{12.4, 51.9, 88.1}, 2)
	if got != "12% 52% +1 more" {
		t.Fatalf("unexpected float series %q", got)
	}
	if empty := formatFloatSeries(nil, 2); empty != "" {
		t.Fatalf("expected empty series for nil values, got %q", empty)
	}
}

func TestPrintPlanMarksNativeAndProtectedItems(t *testing.T) {
	t.Parallel()

	tmp, err := os.CreateTemp(t.TempDir(), "plan-output")
	if err != nil {
		t.Fatal(err)
	}
	defer tmp.Close()

	plan := domain.ExecutionPlan{
		Command:  "uninstall",
		Platform: "darwin",
		Totals:   domain.Totals{Bytes: 4096},
		Items: []domain.Finding{
			{
				Bytes:       2048,
				Category:    domain.CategoryAppLeftovers,
				Status:      domain.StatusPlanned,
				Action:      domain.ActionNative,
				DisplayPath: "/Applications/Example.app",
			},
			{
				Bytes:       1024,
				Category:    domain.CategoryMaintenance,
				Status:      domain.StatusProtected,
				Action:      domain.ActionTrash,
				DisplayPath: "/Library/Keep",
				Policy:      domain.PolicyDecision{Reason: domain.ProtectionProtectedPath},
			},
		},
		Warnings: []string{"review native uninstall first"},
	}

	err = printPlan(tmp, plan, true, []platform.Diagnostic{{Name: "platform", Status: "ok", Message: "darwin adapter active"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if _, err := out.ReadFrom(tmp); err != nil {
		t.Fatal(err)
	}
	view := out.String()
	for _, needle := range []string{
		"UNINSTALL",
		"4.0 KB reclaimable",
		"darwin",
		"/Applications/Example.app",
		"[native]",
		"[protected_path]",
		"Warnings:",
		"Platform diagnostics:",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in plan output, got %s", needle, view)
		}
	}
}
