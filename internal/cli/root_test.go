package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func prepareCLIEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("XDG_CACHE_HOME", root)
	t.Setenv("SIFT_NO_TUI", "1")
}

func executeRootCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func executeRootCommandPiped(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCommand()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	originalStdout := os.Stdout
	originalStderr := os.Stderr
	os.Stdout = writer
	os.Stderr = writer
	defer func() {
		os.Stdout = originalStdout
		os.Stderr = originalStderr
	}()
	root.SetOut(writer)
	root.SetErr(writer)
	root.SetArgs(args)
	runErr := root.Execute()
	_ = writer.Close()
	var out bytes.Buffer
	if _, err := out.ReadFrom(reader); err != nil {
		t.Fatal(err)
	}
	return out.String(), runErr
}

func TestShouldExecutePlan(t *testing.T) {
	t.Parallel()
	plan := domain.ExecutionPlan{
		Command: "clean",
		DryRun:  false,
		Items: []domain.Finding{{
			Status: domain.StatusPlanned,
			Action: domain.ActionTrash,
		}},
	}
	if !shouldExecutePlan(plan) {
		t.Fatal("expected destructive plan to execute")
	}
}

func TestShouldExecutePlanRejectsDryRunAnalyzeAndAdvisory(t *testing.T) {
	t.Parallel()
	cases := []domain.ExecutionPlan{
		{
			Command: "clean",
			DryRun:  true,
			Items: []domain.Finding{{
				Status: domain.StatusPlanned,
				Action: domain.ActionTrash,
			}},
		},
		{
			Command: "analyze",
			DryRun:  false,
			Items: []domain.Finding{{
				Status: domain.StatusPlanned,
				Action: domain.ActionTrash,
			}},
		},
		{
			Command: "clean",
			DryRun:  false,
			Items: []domain.Finding{{
				Status: domain.StatusAdvisory,
				Action: domain.ActionAdvisory,
			}},
		},
		{
			Command:   "clean",
			DryRun:    false,
			PlanState: "empty",
		},
	}
	for _, tc := range cases {
		if shouldExecutePlan(tc) {
			t.Fatalf("expected plan to stay in preview: %+v", tc)
		}
	}
}

func TestResolveTUIEnabled(t *testing.T) {
	t.Parallel()
	if resolveTUIEnabled("plain", true) {
		t.Fatal("expected plain mode to disable TUI")
	}
	if !resolveTUIEnabled("tui", true) {
		t.Fatal("expected forced TUI mode to honor interactive terminals")
	}
	if resolveTUIEnabled("tui", false) {
		t.Fatal("expected forced TUI mode to still require an interactive terminal")
	}
	if !resolveTUIEnabled("auto", true) {
		t.Fatal("expected auto mode to allow interactive terminals")
	}
}

func TestWantsJSONOutputAutoEnablesForPipedStatusAnalyzeAndCheck(t *testing.T) {
	t.Parallel()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()

	state := &runtimeState{}
	if !state.wantsJSONOutput("status", writer) {
		t.Fatal("expected piped status output to prefer JSON")
	}
	if !state.wantsJSONOutput("analyze", writer) {
		t.Fatal("expected piped analyze output to prefer JSON")
	}
	if !state.wantsJSONOutput("check", writer) {
		t.Fatal("expected piped check output to prefer JSON")
	}
	if state.wantsJSONOutput("clean", writer) {
		t.Fatal("expected piped clean output to stay in plain mode unless --json is set")
	}

	state.flags.Plain = true
	if state.wantsJSONOutput("status", writer) {
		t.Fatal("expected --plain to disable auto JSON")
	}
}

func TestPrintCheckReportUsesGroupedSections(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	report := domain.CheckReport{
		Platform: "darwin",
		Summary:  domain.CheckSummary{Warn: 2, Autofixable: 1},
		Items: []domain.CheckItem{
			{ID: "firewall", Group: domain.CheckGroupSecurity, Name: "Firewall", Status: "warn", Message: "Disabled", Commands: []string{"sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate on"}},
			{ID: "brew_updates", Group: domain.CheckGroupUpdates, Name: "Homebrew updates", Status: "ok", Message: "Up to date"},
		},
	}

	if err := printCheckReport(&out, report); err != nil {
		t.Fatal(err)
	}
	view := out.String()
	for _, needle := range []string{"CHECK  darwin  2 findings  1 autofixable", "SECURITY", "UPDATES", "[WARN] Firewall", "-> sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate on"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in check report, got %s", needle, view)
		}
	}
}

func TestPrintAnalyzePlanUsesSections(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	plan := domain.ExecutionPlan{
		Command:  "analyze",
		Platform: "darwin",
		Totals:   domain.Totals{ItemCount: 2, Bytes: 3072},
		Items: []domain.Finding{
			{Category: domain.CategoryDiskUsage, DisplayPath: "/tmp/cache", Bytes: 2048},
			{Category: domain.CategoryLargeFiles, DisplayPath: "/tmp/cache/big.bin", Bytes: 1024},
		},
	}
	tmp, err := os.CreateTemp(t.TempDir(), "plan-output")
	if err != nil {
		t.Fatal(err)
	}
	defer tmp.Close()
	if err := printAnalyzePlan(tmp, plan, false, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := out.ReadFrom(tmp); err != nil {
		t.Fatal(err)
	}
	view := out.String()
	if !strings.Contains(view, "Largest children: 1  Large files: 1") || !strings.Contains(view, "Largest children:") || !strings.Contains(view, "Large files:") {
		t.Fatalf("unexpected analyze plain output: %s", view)
	}
}

func TestParseCommandWhitelistArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		args       []string
		wantAction string
		wantPath   string
		wantErr    bool
	}{
		{name: "default list", args: nil, wantAction: "list"},
		{name: "explicit list", args: []string{"list"}, wantAction: "list"},
		{name: "add", args: []string{"add", "~/Projects/keep-me"}, wantAction: "add", wantPath: "~/Projects/keep-me"},
		{name: "remove", args: []string{"remove", "~/Projects/keep-me"}, wantAction: "remove", wantPath: "~/Projects/keep-me"},
		{name: "missing path", args: []string{"add"}, wantErr: true},
		{name: "unknown action", args: []string{"nope", "x"}, wantErr: true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			action, path, err := parseCommandWhitelistArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %v", tc.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %v: %v", tc.args, err)
			}
			if action != tc.wantAction || path != tc.wantPath {
				t.Fatalf("expected (%q, %q), got (%q, %q)", tc.wantAction, tc.wantPath, action, path)
			}
		})
	}
}

func TestNormalizeUpdateChannelFlag(t *testing.T) {
	t.Parallel()
	if channel, err := normalizeUpdateChannelFlag("stable"); err != nil || string(channel) != "stable" {
		t.Fatalf("expected stable, got channel=%q err=%v", channel, err)
	}
	if channel, err := normalizeUpdateChannelFlag("nightly"); err != nil || string(channel) != "nightly" {
		t.Fatalf("expected nightly, got channel=%q err=%v", channel, err)
	}
	if _, err := normalizeUpdateChannelFlag("beta"); err == nil {
		t.Fatal("expected invalid channel to fail")
	}
}

func TestRootHelpMentionsPipeJSONContract(t *testing.T) {
	t.Parallel()

	root := NewRootCommand()
	if !strings.Contains(root.Long, "automatically emit JSON when stdout is piped") {
		t.Fatalf("expected root long help to mention pipe JSON contract, got %q", root.Long)
	}
	if !strings.Contains(root.Example, "sift status | jq") {
		t.Fatalf("expected root examples to mention piped status JSON, got %q", root.Example)
	}
}

func TestCommandHelpExamplesCoverOperationalFlows(t *testing.T) {
	t.Parallel()

	root := NewRootCommand()
	checkCmd, _, err := root.Find([]string{"check"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(checkCmd.Long, "automatically switches to JSON") {
		t.Fatalf("expected check long help to mention pipe behavior, got %q", checkCmd.Long)
	}
	if !strings.Contains(checkCmd.Example, "sift check | jq") {
		t.Fatalf("expected check examples to mention jq piping, got %q", checkCmd.Example)
	}

	optimizeCmd, _, err := root.Find([]string{"optimize"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(optimizeCmd.Example, "--dry-run=false --yes") {
		t.Fatalf("expected optimize examples to mention apply flags, got %q", optimizeCmd.Example)
	}

	updateCmd, _, err := root.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(updateCmd.Example, "--force --dry-run=false --yes") {
		t.Fatalf("expected update examples to mention force apply flow, got %q", updateCmd.Example)
	}

	touchIDCmd, _, err := root.Find([]string{"touchid"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(touchIDCmd.Long, "sudo_local migration") {
		t.Fatalf("expected touchid long help to mention sudo_local migration, got %q", touchIDCmd.Long)
	}
	if !strings.Contains(touchIDCmd.Example, "touchid enable --dry-run=false --yes") {
		t.Fatalf("expected touchid examples to mention apply flow, got %q", touchIDCmd.Example)
	}
}

func TestCheckCommandAutoEmitsJSONWhenPiped(t *testing.T) {
	prepareCLIEnv(t)

	out, err := executeRootCommandPiped(t, "check")
	if err != nil {
		t.Fatal(err)
	}
	var report domain.CheckReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("expected JSON check output, got %q: %v", out, err)
	}
	if report.Platform == "" {
		t.Fatalf("expected check report platform in %q", out)
	}
}

func TestCheckCommandPlainFlagOverridesAutoJSON(t *testing.T) {
	prepareCLIEnv(t)

	out, err := executeRootCommandPiped(t, "check", "--plain")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "CHECK  ") {
		t.Fatalf("expected plain check output, got %q", out)
	}
}

func TestAnalyzeCommandAutoEmitsJSONWhenPiped(t *testing.T) {
	prepareCLIEnv(t)
	target := t.TempDir()
	if err := os.WriteFile(target+"/blob.bin", []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeRootCommandPiped(t, "analyze", target)
	if err != nil {
		t.Fatal(err)
	}
	var plan domain.ExecutionPlan
	if err := json.Unmarshal([]byte(out), &plan); err != nil {
		t.Fatalf("expected JSON analyze output, got %q: %v", out, err)
	}
	if plan.Command != "analyze" {
		t.Fatalf("expected analyze plan, got %+v", plan)
	}
}

func TestTouchIDEnableApplyRequiresYes(t *testing.T) {
	prepareCLIEnv(t)

	_, err := executeRootCommand(t, "touchid", "enable", "--dry-run=false", "--non-interactive")
	if err == nil || !strings.Contains(err.Error(), "--yes is required") {
		t.Fatalf("expected apply guard error, got %v", err)
	}
}

func TestUpdateApplyRequiresYes(t *testing.T) {
	prepareCLIEnv(t)

	_, err := executeRootCommand(t, "update", "--dry-run=false", "--non-interactive")
	if err == nil || !strings.Contains(err.Error(), "--yes is required") {
		t.Fatalf("expected update apply guard error, got %v", err)
	}
}
