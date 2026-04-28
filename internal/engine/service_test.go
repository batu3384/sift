package engine

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/platform"
	"github.com/batu3384/sift/internal/store"
)

type stubAdapter struct {
	name        string
	roots       platform.CuratedRoots
	protected   []string
	admin       []string
	remnants    []string
	tasks       []domain.MaintenanceTask
	diagnostics []platform.Diagnostic
}

func (s stubAdapter) Name() string {
	if s.name != "" {
		return s.name
	}
	return "stub"
}
func (s stubAdapter) CuratedRoots() platform.CuratedRoots                       { return s.roots }
func (s stubAdapter) ProtectedPaths() []string                                  { return s.protected }
func (s stubAdapter) ResolveTargets(in []string) []string                       { return in }
func (s stubAdapter) ListApps(context.Context, bool) ([]domain.AppEntry, error) { return nil, nil }
func (s stubAdapter) DiscoverRemnants(context.Context, domain.AppEntry) ([]string, []string, error) {
	return s.remnants, nil, nil
}
func (s stubAdapter) MaintenanceTasks(context.Context) []domain.MaintenanceTask {
	return append([]domain.MaintenanceTask{}, s.tasks...)
}
func (s stubAdapter) Diagnostics(context.Context) []platform.Diagnostic {
	return append([]platform.Diagnostic{}, s.diagnostics...)
}
func (s stubAdapter) IsAdminPath(path string) bool {
	for _, prefix := range s.admin {
		if prefix != "" && len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
func (s stubAdapter) IsFileInUse(context.Context, string) bool { return false }
func (s stubAdapter) IsProcessRunning(...string) bool          { return false }

func testManagedCommandPath() string {
	if runtime.GOOS == "windows" {
		return `C:\Windows\System32\cmd.exe`
	}
	return "/usr/bin/true"
}

func testManagedCommandArgs() []string {
	if runtime.GOOS == "windows" {
		return []string{"/c", "ver"}
	}
	return []string{"--version"}
}

func testDialogSensitiveCommandPath() string {
	if runtime.GOOS == "windows" {
		return `C:\Tools\osascript.exe`
	}
	return "/usr/bin/osascript"
}

func testAdminCommandPath() string {
	if runtime.GOOS == "windows" {
		return `C:\Tools\sudo.exe`
	}
	return "/usr/bin/sudo"
}

func testNativeUninstallCommand() string {
	switch runtime.GOOS {
	case "darwin":
		return "/Applications/Example Uninstaller.app"
	case "windows":
		return "MsiExec.exe /x Example"
	default:
		return "/usr/bin/true"
	}
}

func testManualExecutablePath() string {
	if runtime.GOOS == "windows" {
		return `C:\Users\runneradmin\go\bin\sift.exe`
	}
	return "/usr/local/bin/sift"
}

func testGoExecutablePath() string {
	if runtime.GOOS == "windows" {
		return `C:\Go\bin\go.exe`
	}
	return "/usr/local/go/bin/go"
}

func TestCheckReportIncludesDiagnosticsAndAutofixableFindings(t *testing.T) {
	t.Parallel()

	service := &Service{
		Adapter: stubAdapter{
			diagnostics: []platform.Diagnostic{
				{Name: "firewall", Status: "warn", Message: "Firewall is disabled."},
				{Name: "touchid", Status: "warn", Message: "Touch ID is disabled for sudo."},
				{Name: "brew_health", Status: "ok", Message: "Homebrew doctor clean."},
				{Name: "git_identity", Status: "ok", Message: "Git identity configured."},
			},
		},
		Config: config.Default(),
	}

	report, err := service.CheckReport(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Platform != "stub" {
		t.Fatalf("expected stub platform, got %q", report.Platform)
	}
	if report.Summary.Warn < 2 {
		t.Fatalf("expected at least two warning findings, got %+v", report.Summary)
	}
	if report.Summary.Autofixable < 2 {
		t.Fatalf("expected at least two autofixable findings, got %+v", report.Summary)
	}
	var firewall, touchID, gitIdentity, brewHealth *domain.CheckItem
	for idx := range report.Items {
		switch report.Items[idx].ID {
		case checkIDFirewall:
			if firewall == nil || report.Items[idx].Status == "warn" {
				firewall = &report.Items[idx]
			}
		case checkIDTouchID:
			if touchID == nil || report.Items[idx].Status == "warn" {
				touchID = &report.Items[idx]
			}
		case checkIDGitIdentity:
			gitIdentity = &report.Items[idx]
		case checkIDBrewHealth:
			brewHealth = &report.Items[idx]
		}
	}
	if firewall == nil || firewall.Group != domain.CheckGroupSecurity || !firewall.AutofixAvailable {
		t.Fatalf("expected firewall security autofix item, got %+v", firewall)
	}
	if touchID == nil || !touchID.AutofixAvailable || len(touchID.Commands) == 0 {
		t.Fatalf("expected touch id autofix commands, got %+v", touchID)
	}
	if gitIdentity == nil || gitIdentity.Group != domain.CheckGroupConfig || gitIdentity.Status != "ok" {
		t.Fatalf("expected git identity config check, got %+v", gitIdentity)
	}
	if brewHealth == nil || brewHealth.Group != domain.CheckGroupHealth || brewHealth.Status != "ok" {
		t.Fatalf("expected Homebrew health check item, got %+v", brewHealth)
	}
}

func TestBuildAutofixPlanBuildsManagedCommandItems(t *testing.T) {
	t.Parallel()

	service := &Service{
		Adapter: stubAdapter{
			diagnostics: []platform.Diagnostic{
				{Name: "firewall", Status: "warn", Message: "Firewall is disabled."},
				{Name: "touchid", Status: "warn", Message: "Touch ID is disabled for sudo."},
			},
		},
		Config: config.Default(),
		Executable: func() (string, error) {
			return "/usr/local/bin/sift", nil
		},
	}

	plan, err := service.BuildAutofixPlan(context.Background(), true, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Command != "autofix" {
		t.Fatalf("expected autofix command, got %+v", plan)
	}
	if len(plan.Items) < 2 {
		t.Fatalf("expected autofix plan items, got %+v", plan.Items)
	}
	var firewall, touchID *domain.Finding
	for idx := range plan.Items {
		item := &plan.Items[idx]
		if item.RuleID == "autofix.firewall" {
			firewall = item
		}
		if item.RuleID == "autofix.touchid" {
			touchID = item
		}
	}
	if firewall == nil || firewall.Action != domain.ActionCommand || firewall.CommandPath != "/usr/libexec/ApplicationFirewall/socketfilterfw" {
		t.Fatalf("expected firewall managed command finding, got %+v", firewall)
	}
	if touchID == nil || touchID.CommandPath != "/usr/local/bin/sift" {
		t.Fatalf("expected touch id managed command finding, got %+v", touchID)
	}
	if touchID.TimeoutSeconds == 0 || !strings.Contains(touchID.DisplayPath, "touchid enable") {
		t.Fatalf("expected touch id display command details, got %+v", touchID)
	}
}

func TestScanMarksProtectedPaths(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	protected := filepath.Join(root, "protected")
	if err := os.MkdirAll(protected, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(protected, "cache.bin")
	if err := os.WriteFile(file, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Profiles["safe"] = []string{"temp_files"}
	cfg.ProtectedPaths = []string{protected}
	service := &Service{
		Adapter: stubAdapter{
			roots: platform.CuratedRoots{
				Temp: []string{protected},
			},
		},
		Config: cfg,
	}
	plan, err := service.Scan(context.Background(), ScanOptions{
		Command: "clean",
		Profile: "safe",
		DryRun:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("expected one finding, got %d", len(plan.Items))
	}
	if plan.Items[0].Status != domain.StatusProtected {
		t.Fatalf("expected protected status, got %s", plan.Items[0].Status)
	}
	if plan.Items[0].Policy.Reason != domain.ProtectionProtectedPath {
		t.Fatalf("expected protected path reason, got %s", plan.Items[0].Policy.Reason)
	}
}

func TestScanEmitsFindingProgressForEachNormalizedItem(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cacheRoot := filepath.Join(root, "scratch-items")
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(cacheRoot, "first.bin")
	second := filepath.Join(cacheRoot, "second.bin")
	if err := os.WriteFile(first, []byte("payload-1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("payload-2"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Profiles["safe"] = []string{"temp_files"}
	service := &Service{
		Adapter: stubAdapter{
			roots: platform.CuratedRoots{
				Temp: []string{cacheRoot},
			},
		},
		Config: cfg,
	}

	var seen []domain.Finding
	plan, err := service.Scan(context.Background(), ScanOptions{
		Command: "clean",
		Profile: "safe",
		DryRun:  true,
		FindingCallback: func(ruleID string, ruleName string, item domain.Finding) {
			seen = append(seen, item)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) != 2 {
		t.Fatalf("expected two findings in plan, got %d", len(plan.Items))
	}
	if len(seen) != 2 {
		t.Fatalf("expected two streamed findings, got %d", len(seen))
	}
	for _, item := range seen {
		if item.Path == "" || item.DisplayPath == "" {
			t.Fatalf("expected normalized streamed finding, got %+v", item)
		}
		if item.RuleID == "" || item.Name == "" || item.Category == "" {
			t.Fatalf("expected streamed finding metadata, got %+v", item)
		}
	}
}

func TestTrashPathsMovesSelectedTargetsToTrash(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "cache.bin")
	if err := os.WriteFile(target, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	var trashed []string
	service := &Service{
		Adapter: stubAdapter{},
		Config:  config.Default(),
		MoveToTrash: func(path string) error {
			trashed = append(trashed, path)
			return nil
		},
	}
	result, err := service.TrashPaths(context.Background(), "analyze", []string{target}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(trashed) != 1 || trashed[0] != target {
		t.Fatalf("expected target to be moved to trash, got %+v", trashed)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected one operation result, got %+v", result.Items)
	}
	if result.Items[0].Status != domain.StatusDeleted {
		t.Fatalf("expected deleted status, got %+v", result.Items[0])
	}
}

func TestTrashPathsRespectsProtectionPolicy(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "cache.bin")
	if err := os.WriteFile(target, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.ProtectedPaths = []string{root}
	called := false
	service := &Service{
		Adapter: stubAdapter{},
		Config:  cfg,
		MoveToTrash: func(path string) error {
			called = true
			return nil
		},
	}
	result, err := service.TrashPaths(context.Background(), "analyze", []string{target}, false)
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("expected protected target to skip trash executor")
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected one operation result, got %+v", result.Items)
	}
	if result.Items[0].Status != domain.StatusProtected {
		t.Fatalf("expected protected status, got %+v", result.Items[0])
	}
}

func TestScanAllowsSafeBrowserCacheUnderBuiltInProtectedRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	protectedRoot := filepath.Join(root, "Chrome", "User Data")
	safeRoot := filepath.Join(protectedRoot, "Default", "Code Cache")
	child := filepath.Join(safeRoot, "js")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Profiles["safe"] = []string{"browser_data"}
	service := &Service{
		Adapter: stubAdapter{
			roots: platform.CuratedRoots{
				Browser: []string{safeRoot},
			},
			protected: []string{protectedRoot},
		},
		Config: cfg,
	}
	plan, err := service.Scan(context.Background(), ScanOptions{
		Command: "clean",
		Profile: "safe",
		DryRun:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("expected one browser finding, got %+v", plan.Items)
	}
	if plan.Items[0].Status != domain.StatusPlanned {
		t.Fatalf("expected safe browser cache to remain planned, got %+v", plan.Items[0])
	}
}

func TestExplainProtectionReportsSafeException(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	protectedRoot := filepath.Join(root, "Chrome", "User Data")
	safeRoot := filepath.Join(protectedRoot, "Default", "Code Cache")
	explanation := (&Service{
		Adapter: stubAdapter{
			roots: platform.CuratedRoots{
				Browser: []string{safeRoot},
			},
			protected: []string{protectedRoot},
		},
		Config: config.Default(),
	}).ExplainProtection(filepath.Join(safeRoot, "js"))
	if explanation.State != domain.ProtectionStateSafeException {
		t.Fatalf("expected safe exception, got %+v", explanation)
	}
	if len(explanation.SystemMatches) == 0 || len(explanation.ExceptionMatches) == 0 {
		t.Fatalf("expected matching protected root and exception, got %+v", explanation)
	}
}

func TestUserProtectedPathsOverrideBuiltInSafeExceptions(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	protectedRoot := filepath.Join(root, "Chrome", "User Data")
	safeRoot := filepath.Join(protectedRoot, "Default", "Code Cache")
	child := filepath.Join(safeRoot, "js")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Profiles["safe"] = []string{"browser_data"}
	cfg.ProtectedPaths = []string{safeRoot}
	service := &Service{
		Adapter: stubAdapter{
			roots: platform.CuratedRoots{
				Browser: []string{safeRoot},
			},
			protected: []string{protectedRoot},
		},
		Config: cfg,
	}
	plan, err := service.Scan(context.Background(), ScanOptions{
		Command: "clean",
		Profile: "safe",
		DryRun:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("expected one browser finding, got %+v", plan.Items)
	}
	if plan.Items[0].Status != domain.StatusProtected {
		t.Fatalf("expected user protected path to win, got %+v", plan.Items[0])
	}
	if plan.Items[0].Policy.Reason != domain.ProtectionProtectedPath {
		t.Fatalf("expected protected path reason, got %+v", plan.Items[0].Policy)
	}
}

func TestExplainProtectionReportsUserProtectedOverride(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	protectedRoot := filepath.Join(root, "Chrome", "User Data")
	safeRoot := filepath.Join(protectedRoot, "Default", "Code Cache")
	explanation := (&Service{
		Adapter: stubAdapter{
			roots: platform.CuratedRoots{
				Browser: []string{safeRoot},
			},
			protected: []string{protectedRoot},
		},
		Config: config.Config{
			ProtectedPaths: []string{safeRoot},
		},
	}).ExplainProtection(filepath.Join(safeRoot, "js"))
	if explanation.State != domain.ProtectionStateUserProtected {
		t.Fatalf("expected user protected override, got %+v", explanation)
	}
	if len(explanation.UserMatches) == 0 {
		t.Fatalf("expected matching user protected path, got %+v", explanation)
	}
}

func TestScanAllowsPackageManagerCacheUnderBuiltInProtectedRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	protectedRoot := filepath.Join(root, "ProgramData")
	safeRoot := filepath.Join(protectedRoot, "chocolatey", "cache")
	child := filepath.Join(safeRoot, "pkg")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "archive.nupkg"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Profiles["safe"] = []string{"package_manager_caches"}
	service := &Service{
		Adapter: stubAdapter{
			roots: platform.CuratedRoots{
				PackageManager: []string{safeRoot},
			},
			protected: []string{protectedRoot},
		},
		Config: cfg,
	}
	plan, err := service.Scan(context.Background(), ScanOptions{
		Command: "clean",
		Profile: "safe",
		DryRun:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("expected one package cache finding, got %+v", plan.Items)
	}
	if plan.Items[0].Status != domain.StatusPlanned {
		t.Fatalf("expected safe package cache to remain planned, got %+v", plan.Items[0])
	}
}

func TestScanAllowsSafeAICacheUnderProtectedFamilyRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	protectedRoot := filepath.Join(root, "Library", "Application Support", "Claude")
	safeRoot := filepath.Join(protectedRoot, "Code Cache")
	child := filepath.Join(safeRoot, "js")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Profiles["safe"] = []string{"temp_files"}
	cfg.ProtectedFamilies = []string{"ai_workspaces"}
	service := &Service{
		Adapter: stubAdapter{
			name: "darwin",
			roots: platform.CuratedRoots{
				Temp: []string{safeRoot},
			},
		},
		Config: cfg,
	}
	plan, err := service.Scan(context.Background(), ScanOptions{
		Command: "clean",
		Profile: "safe",
		DryRun:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("expected one AI cache finding, got %+v", plan.Items)
	}
	if plan.Items[0].Status != domain.StatusPlanned {
		t.Fatalf("expected safe AI cache to remain planned, got %+v", plan.Items[0])
	}
}

func TestExplainProtectionReportsSafeExceptionForAICache(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	safeRoot := filepath.Join(root, "Library", "Application Support", "ChatGPT", "GPUCache")
	explanation := (&Service{
		Adapter: stubAdapter{
			name: "darwin",
			roots: platform.CuratedRoots{
				Temp: []string{safeRoot},
			},
		},
		Config: config.Config{
			ProtectedFamilies: []string{"ai_workspaces"},
		},
	}).ExplainProtection(filepath.Join(safeRoot, "index"))
	if explanation.State != domain.ProtectionStateSafeException {
		t.Fatalf("expected safe exception, got %+v", explanation)
	}
	if len(explanation.FamilyMatches) == 0 || len(explanation.ExceptionMatches) == 0 {
		t.Fatalf("expected matching family and safe exception, got %+v", explanation)
	}
}

func TestDiagnosticsIncludePolicyAndRuntimeSummary(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "state.db")
	st, err := store.OpenAt(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cfg := config.Default()
	cfg.ProtectedPaths = []string{"/tmp/keep-me"}
	cfg.CommandExcludes = map[string][]string{"clean": {"/tmp/cache-only"}}
	cfg.PurgeSearchPaths = []string{"/tmp/projects", "/tmp/work"}
	service := &Service{
		Adapter: stubAdapter{
			protected: []string{"/System", "/Library"},
		},
		Config: cfg,
		Store:  st,
		LookPath: func(name string) (string, error) {
			switch name {
			case "pwsh":
				return "/opt/homebrew/bin/pwsh", nil
			case "goreleaser":
				return "/opt/homebrew/bin/goreleaser", nil
			default:
				return "", errors.New("unexpected tool lookup")
			}
		},
	}
	diagnostics := service.Diagnostics(context.Background())
	index := map[string]platform.Diagnostic{}
	for _, diagnostic := range diagnostics {
		index[diagnostic.Name] = diagnostic
	}
	for _, name := range []string{"interaction_mode", "trash_mode", "confirm_level", "test_policy", "live_integration", "protected_paths", "command_excludes", "built_in_protected_paths", "protected_families", "safe_exceptions", "purge_search_paths", "report_cache", "audit_log", "pwsh", "goreleaser", "parity_matrix", "upstream_baseline", "touchid"} {
		if _, ok := index[name]; !ok {
			t.Fatalf("expected diagnostic %q, got %+v", name, diagnostics)
		}
	}
	if index["purge_search_paths"].Status != "ok" {
		t.Fatalf("expected configured purge search paths to be ok, got %+v", index["purge_search_paths"])
	}
	if !strings.Contains(index["protected_paths"].Message, "1 user-defined root") {
		t.Fatalf("unexpected protected_paths diagnostic: %+v", index["protected_paths"])
	}
	if !strings.Contains(index["command_excludes"].Message, "1 command scope") {
		t.Fatalf("unexpected command_excludes diagnostic: %+v", index["command_excludes"])
	}
	if index["test_policy"].Status != "ok" || !strings.Contains(index["test_policy"].Message, "host mode") {
		t.Fatalf("expected host-mode test policy diagnostic, got %+v", index["test_policy"])
	}
	if index["live_integration"].Status != "ok" || !strings.Contains(index["live_integration"].Message, "integration-live-macos") {
		t.Fatalf("expected live integration diagnostic, got %+v", index["live_integration"])
	}
	if index["pwsh"].Status != "ok" || index["goreleaser"].Status != "ok" {
		t.Fatalf("expected tool diagnostics to be ok, got pwsh=%+v goreleaser=%+v", index["pwsh"], index["goreleaser"])
	}
	if index["protected_families"].Status != "ok" || index["safe_exceptions"].Status != "ok" {
		t.Fatalf("expected protection diagnostics to be ok, got protected_families=%+v safe_exceptions=%+v", index["protected_families"], index["safe_exceptions"])
	}
	if index["parity_matrix"].Status != "ok" {
		t.Fatalf("expected parity matrix to be green once missing gaps are closed, got %+v", index["parity_matrix"])
	}
	if !strings.Contains(index["parity_matrix"].Message, "0 missing") || !strings.Contains(index["parity_matrix"].Message, "0 regression-risk") {
		t.Fatalf("expected parity matrix summary to report zero missing gaps, got %+v", index["parity_matrix"])
	}
	if index["upstream_baseline"].Status != "ok" || !strings.Contains(index["upstream_baseline"].Message, "changed file") {
		t.Fatalf("expected upstream baseline diagnostic to carry compare metadata, got %+v", index["upstream_baseline"])
	}
}

func TestDiagnosticsWarnWhenPurgeSearchPathsUnset(t *testing.T) {
	t.Parallel()
	service := &Service{
		Adapter: stubAdapter{},
		Config:  config.Default(),
		LookPath: func(name string) (string, error) {
			return "", errors.New(name + " missing")
		},
	}
	diagnostics := service.Diagnostics(context.Background())
	seenPurge := false
	seenPwsh := false
	seenGoreleaser := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Name == "purge_search_paths" {
			seenPurge = true
			if diagnostic.Status != "warn" {
				t.Fatalf("expected warn status for unset purge search paths, got %+v", diagnostic)
			}
		}
		if diagnostic.Name == "pwsh" {
			seenPwsh = true
			if diagnostic.Status != "warn" {
				t.Fatalf("expected warn status for missing pwsh, got %+v", diagnostic)
			}
		}
		if diagnostic.Name == "goreleaser" {
			seenGoreleaser = true
			if diagnostic.Status != "warn" {
				t.Fatalf("expected warn status for missing goreleaser, got %+v", diagnostic)
			}
		}
	}
	if !seenPurge || !seenPwsh || !seenGoreleaser {
		t.Fatalf("expected purge_search_paths, pwsh, and goreleaser diagnostics, got %+v", diagnostics)
	}
}

func TestDiagnosticsReflectCiSafeAndLiveIntegrationModes(t *testing.T) {
	t.Setenv("SIFT_TEST_MODE", "ci-safe")
	t.Setenv("SIFT_LIVE_INTEGRATION", "1")
	service := &Service{
		Adapter: stubAdapter{},
		Config:  config.Default(),
		LookPath: func(name string) (string, error) {
			return "", errors.New(name + " missing")
		},
	}
	diagnostics := service.Diagnostics(context.Background())
	index := map[string]platform.Diagnostic{}
	for _, diagnostic := range diagnostics {
		index[diagnostic.Name] = diagnostic
	}
	if !strings.Contains(index["test_policy"].Message, "live integration") {
		t.Fatalf("expected ci-safe override message, got %+v", index["test_policy"])
	}
	if !strings.Contains(index["live_integration"].Message, "enabled") {
		t.Fatalf("expected live integration enabled message, got %+v", index["live_integration"])
	}
}

func TestExplainProtectionForCommandReportsScopedMatches(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "cache", "artifact.bin")
	service := &Service{
		Adapter: stubAdapter{},
		Config: config.Config{
			CommandExcludes: map[string][]string{
				"clean": {filepath.Join(root, "cache")},
			},
		},
	}
	explanation := service.ExplainProtectionForCommand(path, "clean")
	if explanation.State != domain.ProtectionStateCommandProtected {
		t.Fatalf("expected command protected state, got %+v", explanation)
	}
	if explanation.Command != "clean" || len(explanation.CommandMatches) != 1 {
		t.Fatalf("expected clean command match, got %+v", explanation)
	}
}

func TestExplainProtectionReportsProtectedFamilyMatches(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	service := &Service{
		Adapter: stubAdapter{
			roots: platform.CuratedRoots{
				Temp: []string{filepath.Join(home, "Library", "Caches")},
			},
		},
		Config: config.Config{
			ProtectedFamilies: []string{"browser_profiles"},
		},
	}
	path := filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Default", "History")
	explanation := service.ExplainProtection(path)
	if explanation.State != domain.ProtectionStateUserProtected {
		t.Fatalf("expected user protected state from family, got %+v", explanation)
	}
	if len(explanation.FamilyMatches) == 0 || explanation.FamilyMatches[0] != "browser_profiles" {
		t.Fatalf("expected browser_profiles family match, got %+v", explanation)
	}
}

func TestBuildOptimizePlanCreatesActionableAndAdvisoryMaintenanceItems(t *testing.T) {
	t.Parallel()
	cacheRoot := filepath.Join(t.TempDir(), "optimize-cache")
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheRoot, "thumbs.db"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	service := &Service{
		Adapter: stubAdapter{
			tasks: []domain.MaintenanceTask{
				{
					ID:                "maintenance.reset-icons",
					Title:             "Reset icon cache",
					Description:       "Clear stale icon cache files.",
					Risk:              domain.RiskSafe,
					Phase:             "cleanup",
					EstimatedImpact:   "Refreshes stale icon caches.",
					Verification:      []string{"Reopen Finder"},
					SuggestedByChecks: []string{"check.disk_pressure"},
					Action:            domain.ActionTrash,
					Paths:             []string{cacheRoot},
					Steps:             []string{"Explorer or Finder will rebuild icons automatically"},
				},
				{
					ID:                "maintenance.review",
					Title:             "Review startup items",
					Description:       "Check startup entries before cleanup.",
					Risk:              domain.RiskReview,
					Phase:             "preflight",
					SuggestedByChecks: []string{"check.login_items"},
					Steps:             []string{"Open startup settings", "Disable unused entries"},
				},
				{
					ID:          "maintenance.flush-dns",
					Title:       "Flush DNS cache",
					Description: "Refresh resolver state.",
					Risk:        domain.RiskReview,
					Action:      domain.ActionCommand,
					CommandPath: testManagedCommandPath(),
					Steps:       []string{"DNS cache refreshes immediately"},
				},
			},
			diagnostics: []platform.Diagnostic{
				{Name: "login_items", Status: "warn", Message: "5 login items active."},
			},
		},
		Config: config.Default(),
	}
	plan, err := service.BuildOptimizePlan(context.Background(), false, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Command != "optimize" || len(plan.Items) != 3 {
		t.Fatalf("unexpected optimize plan: %+v", plan)
	}
	if len(plan.Warnings) == 0 || !strings.Contains(plan.Warnings[0], "Preflight found") || !strings.Contains(plan.Warnings[0], "Login items") {
		t.Fatalf("expected preflight warning summary, got %+v", plan.Warnings)
	}
	itemsByRule := map[string]domain.Finding{}
	for _, item := range plan.Items {
		itemsByRule[item.RuleID] = item
	}
	resetIcons := itemsByRule["maintenance.reset-icons"]
	if resetIcons.Action != domain.ActionTrash || resetIcons.Category != domain.CategoryMaintenance || resetIcons.Status != domain.StatusPlanned {
		t.Fatalf("expected actionable maintenance finding, got %+v", resetIcons)
	}
	if resetIcons.TaskPhase != "cleanup" || resetIcons.TaskImpact != "Refreshes stale icon caches." || len(resetIcons.TaskVerify) != 1 {
		t.Fatalf("expected maintenance metadata on actionable item, got %+v", resetIcons)
	}
	reviewStartup := itemsByRule["maintenance.review"]
	if reviewStartup.Action != domain.ActionAdvisory || reviewStartup.Category != domain.CategoryMaintenance || reviewStartup.Status != domain.StatusAdvisory {
		t.Fatalf("expected advisory maintenance finding, got %+v", reviewStartup)
	}
	if len(reviewStartup.SuggestedBy) != 1 || reviewStartup.SuggestedBy[0] != "Login items" {
		t.Fatalf("expected advisory item to reflect optimize suggestion source, got %+v", reviewStartup)
	}
	flushDNS := itemsByRule["maintenance.flush-dns"]
	if flushDNS.Action != domain.ActionCommand || flushDNS.CommandPath != testManagedCommandPath() || flushDNS.Status != domain.StatusPlanned {
		t.Fatalf("expected managed command maintenance finding, got %+v", flushDNS)
	}
}

func TestExecuteWithOptionsRunsManagedCommandItems(t *testing.T) {
	t.Parallel()

	var (
		runPath string
		runArgs []string
	)
	service := &Service{
		RunCommand: func(ctx context.Context, path string, args ...string) error {
			runPath = path
			runArgs = append([]string{}, args...)
			return nil
		},
	}
	result, err := service.ExecuteWithOptions(context.Background(), domain.ExecutionPlan{
		ScanID: "scan",
		Items: []domain.Finding{{
			ID:          uuid.NewString(),
			Action:      domain.ActionCommand,
			Status:      domain.StatusPlanned,
			DisplayPath: testManagedCommandPath(),
			CommandPath: testManagedCommandPath(),
			CommandArgs: testManagedCommandArgs(),
		}},
	}, ExecuteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].Status != domain.StatusCompleted {
		t.Fatalf("expected completed managed command item, got %+v", result.Items)
	}
	if runPath != testManagedCommandPath() {
		t.Fatalf("expected managed command path to run, got %q", runPath)
	}
	if strings.Join(runArgs, " ") != strings.Join(testManagedCommandArgs(), " ") {
		t.Fatalf("unexpected managed command args: %+v", runArgs)
	}
}

func TestExecuteWithOptionsSkipsManagedCommandInDryRun(t *testing.T) {
	t.Parallel()

	service := &Service{}
	result, err := service.ExecuteWithOptions(context.Background(), domain.ExecutionPlan{
		ScanID:  "scan",
		DryRun:  true,
		Command: "optimize",
		Items: []domain.Finding{{
			ID:          uuid.NewString(),
			Action:      domain.ActionCommand,
			Status:      domain.StatusPlanned,
			DisplayPath: testManagedCommandPath(),
			CommandPath: testManagedCommandPath(),
		}},
	}, ExecuteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].Status != domain.StatusSkipped {
		t.Fatalf("expected dry-run managed command skip, got %+v", result.Items)
	}
}

func TestConfigureTouchIDEnableDryRunLeavesPAMUntouched(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "darwin" {
		t.Skip("Touch ID PAM management is macOS-only")
	}
	root := t.TempDir()
	pamPath := filepath.Join(root, "sudo")
	original := []byte("# sudo: auth account password session\naccount required pam_permit.so\n")
	if err := os.WriteFile(pamPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	service := &Service{TouchIDPAMPath: pamPath}
	result, err := service.ConfigureTouchID(true, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "enable" || result.Changed || !result.DryRun || result.Enabled {
		t.Fatalf("unexpected dry-run enable result: %+v", result)
	}
	after, err := os.ReadFile(pamPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(original) {
		t.Fatalf("expected dry-run PAM to remain unchanged, got %q", string(after))
	}
}

func TestConfigureTouchIDEnableAndDisableWritesBackupAndPAM(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "darwin" {
		t.Skip("Touch ID PAM management is macOS-only")
	}
	root := t.TempDir()
	pamPath := filepath.Join(root, "sudo")
	original := "# sudo: auth account password session\naccount required pam_permit.so\n"
	if err := os.WriteFile(pamPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	service := &Service{TouchIDPAMPath: pamPath}
	enableResult, err := service.ConfigureTouchID(true, false)
	if err != nil {
		t.Fatal(err)
	}
	if !enableResult.Changed || !enableResult.Enabled {
		t.Fatalf("expected enable to apply, got %+v", enableResult)
	}
	enabledContent, err := os.ReadFile(pamPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(enabledContent), "pam_tid.so") {
		t.Fatalf("expected Touch ID line in PAM file, got %q", string(enabledContent))
	}
	backupContent, err := os.ReadFile(pamPath + ".backup.sift")
	if err != nil {
		t.Fatal(err)
	}
	if string(backupContent) != original {
		t.Fatalf("expected original backup, got %q", string(backupContent))
	}
	disableResult, err := service.ConfigureTouchID(false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !disableResult.Changed || disableResult.Enabled {
		t.Fatalf("expected disable to apply, got %+v", disableResult)
	}
	disabledContent, err := os.ReadFile(pamPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(disabledContent), "pam_tid.so") {
		t.Fatalf("expected Touch ID line to be removed, got %q", string(disabledContent))
	}
}

func TestTouchIDStatusReportsSudoLocalMigrationNeeded(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "darwin" {
		t.Skip("Touch ID PAM management is macOS-only")
	}
	root := t.TempDir()
	pamPath := filepath.Join(root, "sudo")
	localPath := filepath.Join(root, "sudo_local")
	mainContent := "# sudo\n" +
		"auth       sufficient     pam_tid.so\n" +
		"auth       include        sudo_local\n" +
		"account required pam_permit.so\n"
	if err := os.WriteFile(pamPath, []byte(mainContent), 0o644); err != nil {
		t.Fatal(err)
	}
	service := &Service{TouchIDPAMPath: pamPath, TouchIDLocalPAMPath: localPath}
	status := service.TouchIDStatus()
	if !status.Enabled || !status.SudoLocalSupported || !status.LegacyConfigured || !status.MigrationNeeded {
		t.Fatalf("expected sudo_local migration status, got %+v", status)
	}
	if status.ActivePAMPath != pamPath {
		t.Fatalf("expected active legacy pam path, got %+v", status)
	}
	if !strings.Contains(status.Message, "Migration") && !strings.Contains(status.Message, "migration") {
		t.Fatalf("expected migration guidance message, got %+v", status)
	}
}

func TestConfigureTouchIDEnableMigratesLegacyConfigToSudoLocal(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "darwin" {
		t.Skip("Touch ID PAM management is macOS-only")
	}
	root := t.TempDir()
	pamPath := filepath.Join(root, "sudo")
	localPath := filepath.Join(root, "sudo_local")
	mainContent := "# sudo\n" +
		"auth       sufficient     pam_tid.so\n" +
		"auth       include        sudo_local\n" +
		"account required pam_permit.so\n"
	if err := os.WriteFile(pamPath, []byte(mainContent), 0o644); err != nil {
		t.Fatal(err)
	}
	service := &Service{TouchIDPAMPath: pamPath, TouchIDLocalPAMPath: localPath}
	result, err := service.ConfigureTouchID(true, false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || !result.Enabled || !result.Migrated {
		t.Fatalf("expected migrated touchid result, got %+v", result)
	}
	mainAfter, err := os.ReadFile(pamPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mainAfter), "pam_tid.so") {
		t.Fatalf("expected legacy pam_tid entry to be cleaned, got %q", string(mainAfter))
	}
	if !strings.Contains(string(mainAfter), "sudo_local") {
		t.Fatalf("expected sudo_local include to remain, got %q", string(mainAfter))
	}
	localAfter, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(localAfter), "pam_tid.so") {
		t.Fatalf("expected sudo_local to contain pam_tid, got %q", string(localAfter))
	}
	if _, err := os.Stat(pamPath + ".backup.sift"); err != nil {
		t.Fatalf("expected legacy backup to exist: %v", err)
	}
}

func TestConfigureTouchIDDisableCleansSudoLocalAndLegacyConfig(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "darwin" {
		t.Skip("Touch ID PAM management is macOS-only")
	}
	root := t.TempDir()
	pamPath := filepath.Join(root, "sudo")
	localPath := filepath.Join(root, "sudo_local")
	mainContent := "# sudo\n" +
		"auth       sufficient     pam_tid.so\n" +
		"auth       include        sudo_local\n" +
		"account required pam_permit.so\n"
	localContent := "# sudo_local: local customizations for sudo\n" +
		"auth       sufficient     pam_tid.so\n"
	if err := os.WriteFile(pamPath, []byte(mainContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localPath, []byte(localContent), 0o444); err != nil {
		t.Fatal(err)
	}
	service := &Service{TouchIDPAMPath: pamPath, TouchIDLocalPAMPath: localPath}
	result, err := service.ConfigureTouchID(false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Enabled {
		t.Fatalf("expected disable to apply, got %+v", result)
	}
	mainAfter, err := os.ReadFile(pamPath)
	if err != nil {
		t.Fatal(err)
	}
	localAfter, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mainAfter), "pam_tid.so") || strings.Contains(string(localAfter), "pam_tid.so") {
		t.Fatalf("expected touchid entries removed from both pam files, got sudo=%q local=%q", string(mainAfter), string(localAfter))
	}
	if _, err := os.Stat(pamPath + ".backup.sift"); err != nil {
		t.Fatalf("expected main backup to exist: %v", err)
	}
	if _, err := os.Stat(localPath + ".backup.sift"); err != nil {
		t.Fatalf("expected sudo_local backup to exist: %v", err)
	}
}

func TestRunUpdateHomebrewPreviewAndApply(t *testing.T) {
	t.Parallel()
	var (
		runCount int
		runPath  string
		runArgs  []string
	)
	service := &Service{
		Adapter: platform.Current(),
		Executable: func() (string, error) {
			return "/opt/homebrew/Cellar/sift/1.0.0/bin/sift", nil
		},
		LookPath: func(name string) (string, error) {
			if name != "brew" {
				t.Fatalf("unexpected lookpath name %q", name)
			}
			return "/opt/homebrew/bin/brew", nil
		},
		RunCommand: func(_ context.Context, path string, args ...string) error {
			runCount++
			runPath = path
			runArgs = append([]string{}, args...)
			return nil
		},
	}
	preview, err := service.RunUpdateWithOptions(context.Background(), true, UpdateOptions{Channel: UpdateChannelStable})
	if err != nil {
		t.Fatal(err)
	}
	if preview.InstallMethod != "homebrew" || preview.Changed || preview.Executable != "/opt/homebrew/bin/brew" {
		t.Fatalf("unexpected update preview: %+v", preview)
	}
	if runCount != 0 {
		t.Fatalf("expected preview to avoid running commands, got %d", runCount)
	}
	applied, err := service.RunUpdateWithOptions(context.Background(), false, UpdateOptions{Channel: UpdateChannelStable})
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Changed {
		t.Fatalf("expected applied update to report change, got %+v", applied)
	}
	if runCount != 1 || runPath != "/opt/homebrew/bin/brew" {
		t.Fatalf("expected brew to run once, got count=%d path=%q", runCount, runPath)
	}
	if strings.Join(runArgs, " ") != "upgrade sift" {
		t.Fatalf("unexpected update args %+v", runArgs)
	}
}

func TestRunUpdateHomebrewForcePreviewAndApply(t *testing.T) {
	t.Parallel()
	var (
		runCount int
		runPath  string
		runArgs  []string
	)
	service := &Service{
		Adapter: platform.Current(),
		Executable: func() (string, error) {
			return "/opt/homebrew/Cellar/sift/1.0.0/bin/sift", nil
		},
		LookPath: func(name string) (string, error) {
			if name != "brew" {
				t.Fatalf("unexpected lookpath name %q", name)
			}
			return "/opt/homebrew/bin/brew", nil
		},
		RunCommand: func(_ context.Context, path string, args ...string) error {
			runCount++
			runPath = path
			runArgs = append([]string{}, args...)
			return nil
		},
	}
	preview, err := service.RunUpdateWithOptions(context.Background(), true, UpdateOptions{
		Channel: UpdateChannelStable,
		Force:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Force || preview.Channel != string(UpdateChannelStable) || preview.Executable != "/opt/homebrew/bin/brew" {
		t.Fatalf("unexpected forced update preview: %+v", preview)
	}
	if runCount != 0 {
		t.Fatalf("expected preview to avoid running commands, got %d", runCount)
	}
	applied, err := service.RunUpdateWithOptions(context.Background(), false, UpdateOptions{
		Channel: UpdateChannelStable,
		Force:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Changed {
		t.Fatalf("expected applied force update to report change, got %+v", applied)
	}
	if runCount != 1 || runPath != "/opt/homebrew/bin/brew" {
		t.Fatalf("expected brew to run once, got count=%d path=%q", runCount, runPath)
	}
	if strings.Join(runArgs, " ") != "reinstall sift" {
		t.Fatalf("unexpected forced update args %+v", runArgs)
	}
}

func TestRunUpdateManualNightlyPreviewAndApply(t *testing.T) {
	t.Parallel()
	var (
		runCount int
		runPath  string
		runArgs  []string
	)
	service := &Service{
		Adapter: platform.Current(),
		Executable: func() (string, error) {
			return testManualExecutablePath(), nil
		},
		LookPath: func(name string) (string, error) {
			if name != "go" {
				t.Fatalf("unexpected lookpath name %q", name)
			}
			return testGoExecutablePath(), nil
		},
		RunCommand: func(_ context.Context, path string, args ...string) error {
			runCount++
			runPath = path
			runArgs = append([]string{}, args...)
			return nil
		},
	}
	preview, err := service.RunUpdateWithOptions(context.Background(), true, UpdateOptions{
		Channel: UpdateChannelNightly,
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Channel != string(UpdateChannelNightly) || preview.Executable != testGoExecutablePath() {
		t.Fatalf("unexpected nightly preview: %+v", preview)
	}
	if runCount != 0 {
		t.Fatalf("expected preview to avoid running commands, got %d", runCount)
	}
	applied, err := service.RunUpdateWithOptions(context.Background(), false, UpdateOptions{
		Channel: UpdateChannelNightly,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Changed {
		t.Fatalf("expected applied nightly update to report change, got %+v", applied)
	}
	if runCount != 1 || runPath != testGoExecutablePath() {
		t.Fatalf("expected go to run once, got count=%d path=%q", runCount, runPath)
	}
	if strings.Join(runArgs, " ") != "install github.com/batu3384/sift/cmd/sift@main" {
		t.Fatalf("unexpected nightly update args %+v", runArgs)
	}
}

func TestBuildRemovePlanTargetsSiftOwnedState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	executable := filepath.Join(home, "bin", "sift")
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(t.TempDir(), "state.db")
	st, err := store.OpenAt(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	service := &Service{
		Adapter: stubAdapter{},
		Config:  config.Default(),
		Store:   st,
		Executable: func() (string, error) {
			return executable, nil
		},
	}
	plan, err := service.BuildRemovePlan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if plan.Command != "remove" || len(plan.Items) == 0 {
		t.Fatalf("expected remove plan items, got %+v", plan)
	}
	foundExecutable := false
	foundRemoveGuidance := false
	foundActionable := false
	for _, item := range plan.Items {
		if item.RuleID == "remove.executable" && item.Action == domain.ActionTrash && item.Path == executable {
			foundExecutable = true
		}
		if item.RuleID == "remove.binary" && item.Action == domain.ActionAdvisory {
			foundRemoveGuidance = true
		}
		if item.Action == domain.ActionTrash {
			foundActionable = true
		}
	}
	if !foundActionable {
		t.Fatalf("expected actionable state cleanup, got %+v", plan.Items)
	}
	if runtime.GOOS == "windows" {
		if !foundRemoveGuidance {
			t.Fatalf("expected manual remove guidance on windows, got %+v", plan.Items)
		}
	} else if !foundExecutable {
		t.Fatalf("expected executable cleanup item, got %+v", plan.Items)
	}
	if runtime.GOOS == "windows" {
		if !strings.Contains(strings.Join(plan.Warnings, " | "), "Executable cleanup could not be staged automatically") {
			t.Fatalf("expected windows executable guidance warning, got %+v", plan.Warnings)
		}
	} else if !strings.Contains(strings.Join(plan.Warnings, " | "), "executable in the review plan") {
		t.Fatalf("expected manual executable warning, got %+v", plan.Warnings)
	}
}

func TestBuildRemovePlanUsesInstallMethodCommand(t *testing.T) {
	executable := filepath.Join("/opt/homebrew/Cellar/sift/1.0.0/bin", "sift")
	service := &Service{
		Adapter: stubAdapter{},
		Config:  config.Default(),
		Executable: func() (string, error) {
			return executable, nil
		},
		LookPath: func(name string) (string, error) {
			if name == "brew" {
				return "/opt/homebrew/bin/brew", nil
			}
			return "", os.ErrNotExist
		},
	}
	plan, err := service.BuildRemovePlan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	foundCommand := false
	for _, item := range plan.Items {
		if item.RuleID == "remove.uninstall" {
			foundCommand = true
			if item.Action != domain.ActionCommand || item.CommandPath != "/opt/homebrew/bin/brew" {
				t.Fatalf("expected managed brew uninstall item, got %+v", item)
			}
			if strings.Join(item.CommandArgs, " ") != "uninstall sift" {
				t.Fatalf("unexpected remove command args %+v", item.CommandArgs)
			}
		}
	}
	if !foundCommand {
		t.Fatalf("expected install-method aware uninstall command, got %+v", plan.Items)
	}
}

func TestExecuteDryRunSkipsDeletion(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	file := filepath.Join(root, "temp.txt")
	if err := os.WriteFile(file, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{}
	result, err := service.ExecuteWithOptions(context.Background(), domain.ExecutionPlan{
		ScanID: "scan",
		DryRun: true,
		Items: []domain.Finding{{
			ID:          uuid.NewString(),
			Path:        file,
			Action:      domain.ActionTrash,
			Status:      domain.StatusPlanned,
			Fingerprint: domain.Fingerprint{Mode: uint32(info.Mode()), Size: info.Size(), ModTime: info.ModTime()},
		}},
	}, ExecuteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].Status != domain.StatusSkipped {
		t.Fatalf("expected dry-run skipped result, got %+v", result.Items)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("expected file to remain, stat err: %v", err)
	}
}

func TestExecuteWithProgressEmitsPerItemUpdates(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	first := filepath.Join(root, "first.txt")
	second := filepath.Join(root, "second.txt")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("payload"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	firstInfo, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	secondInfo, err := os.Stat(second)
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{}
	progress := make([]domain.ExecutionProgress, 0, 4)
	result, err := service.ExecuteWithProgress(context.Background(), domain.ExecutionPlan{
		ScanID: "scan",
		DryRun: true,
		Items: []domain.Finding{
			{
				ID:          uuid.NewString(),
				Path:        first,
				DisplayPath: first,
				Action:      domain.ActionTrash,
				Status:      domain.StatusPlanned,
				Fingerprint: domain.Fingerprint{Mode: uint32(firstInfo.Mode()), Size: firstInfo.Size(), ModTime: firstInfo.ModTime()},
			},
			{
				ID:          uuid.NewString(),
				Path:        second,
				DisplayPath: second,
				Action:      domain.ActionTrash,
				Status:      domain.StatusPlanned,
				Fingerprint: domain.Fingerprint{Mode: uint32(secondInfo.Mode()), Size: secondInfo.Size(), ModTime: secondInfo.ModTime()},
			},
		},
	}, ExecuteOptions{}, func(update domain.ExecutionProgress) {
		progress = append(progress, update)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(progress) != 6 {
		t.Fatalf("expected substep progress updates, got %+v", progress)
	}
	if progress[0].Phase != domain.ProgressPhaseStarting || progress[1].Phase != domain.ProgressPhasePreparing || progress[2].Phase != domain.ProgressPhaseFinished {
		t.Fatalf("expected starting/preparing/finished phases for first item, got %+v", progress[:3])
	}
	if progress[3].Phase != domain.ProgressPhaseStarting || progress[4].Phase != domain.ProgressPhasePreparing || progress[5].Phase != domain.ProgressPhaseFinished {
		t.Fatalf("expected starting/preparing/finished phases for second item, got %+v", progress[3:])
	}
	if progress[2].Current != 1 || progress[5].Current != 2 {
		t.Fatalf("expected current counts 1 and 2, got %+v", progress)
	}
	if progress[2].Result.Path != first || progress[5].Result.Path != second {
		t.Fatalf("unexpected progress paths: %+v", progress)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 final result items, got %+v", result.Items)
	}
}

func TestExecuteWithProgressEmitsCleanSectionEvents(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	first := filepath.Join(root, "cache.bin")
	second := filepath.Join(root, "logs.txt")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("payload"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	firstInfo, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	secondInfo, err := os.Stat(second)
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{}
	progress := make([]domain.ExecutionProgress, 0, 10)
	_, err = service.ExecuteWithProgress(context.Background(), domain.ExecutionPlan{
		ScanID:   "scan",
		Command:  "clean",
		DryRun:   true,
		Platform: "darwin",
		Items: []domain.Finding{
			{
				ID:          uuid.NewString(),
				Name:        "Cache",
				Path:        first,
				DisplayPath: first,
				Category:    domain.CategoryTempFiles,
				Source:      "User cache",
				Action:      domain.ActionTrash,
				Status:      domain.StatusPlanned,
				Fingerprint: domain.Fingerprint{Mode: uint32(firstInfo.Mode()), Size: firstInfo.Size(), ModTime: firstInfo.ModTime()},
			},
			{
				ID:          uuid.NewString(),
				Name:        "Log",
				Path:        second,
				DisplayPath: second,
				Category:    domain.CategoryLogs,
				Source:      "App logs",
				Action:      domain.ActionTrash,
				Status:      domain.StatusPlanned,
				Fingerprint: domain.Fingerprint{Mode: uint32(secondInfo.Mode()), Size: secondInfo.Size(), ModTime: secondInfo.ModTime()},
			},
		},
	}, ExecuteOptions{}, func(update domain.ExecutionProgress) {
		progress = append(progress, update)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(progress) != 10 {
		t.Fatalf("expected section and item progress updates, got %+v", progress)
	}
	if progress[0].Event != domain.ProgressEventSection || progress[0].SectionIndex != 1 || progress[0].SectionTotal != 2 {
		t.Fatalf("expected first section start event, got %+v", progress[0])
	}
	if progress[4].Event != domain.ProgressEventSection || progress[4].Phase != domain.ProgressPhaseFinished || progress[4].SectionDone != 1 {
		t.Fatalf("expected first section finish event, got %+v", progress[4])
	}
	if progress[5].Event != domain.ProgressEventSection || progress[5].SectionIndex != 2 {
		t.Fatalf("expected second section start event, got %+v", progress[5])
	}
	if progress[9].Event != domain.ProgressEventSection || progress[9].Phase != domain.ProgressPhaseFinished || progress[9].SectionDone != 1 {
		t.Fatalf("expected second section finish event, got %+v", progress[9])
	}
}

func TestVerifyFingerprintMismatchFails(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "temp.txt")
	if err := os.WriteFile(file, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyFingerprint(domain.Finding{
		Path: file,
		Fingerprint: domain.Fingerprint{
			Size:    1,
			ModTime: time.Now().UTC(),
		},
	})
	if err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestVerifyFingerprintRejectsModeSwap(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "temp.txt")
	if err := os.WriteFile(file, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	fingerprint, err := currentFingerprint(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(file, fingerprint.ModTime, fingerprint.ModTime); err != nil {
		t.Fatal(err)
	}
	changed, err := currentFingerprint(file)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Mode == fingerprint.Mode {
		t.Skip("platform does not expose chmod mode changes for this file")
	}
	if err := verifyFingerprint(domain.Finding{Path: file, Fingerprint: fingerprint}); err == nil {
		t.Fatal("expected mode mismatch error")
	}
}

func TestExecuteResultCarriesPlanDigest(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	file := filepath.Join(root, "temp.txt")
	if err := os.WriteFile(file, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	plan := domain.ExecutionPlan{
		ScanID: "scan",
		DryRun: true,
		Items: []domain.Finding{{
			ID:          uuid.NewString(),
			Path:        file,
			Action:      domain.ActionTrash,
			Status:      domain.StatusPlanned,
			Fingerprint: domain.Fingerprint{Mode: uint32(info.Mode()), Size: info.Size(), ModTime: info.ModTime()},
		}},
	}
	result, err := (&Service{}).ExecuteWithOptions(context.Background(), plan, ExecuteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	digest, ok := payload["plan_digest"].(string)
	if !ok || digest == "" {
		t.Fatalf("expected execution result to carry plan digest, got %s", raw)
	}
}

func TestExecuteWithProgressEmitsUninstallAndOptimizeSections(t *testing.T) {
	prevStartNativeProcess := startNativeProcess
	startNativeProcess = func(ctx context.Context, command nativeCommand) error { return nil }
	defer func() { startNativeProcess = prevStartNativeProcess }()

	service := &Service{
		RunCommand: func(ctx context.Context, name string, args ...string) error { return nil },
	}
	root := t.TempDir()

	uninstallPlan := domain.ExecutionPlan{
		Command: "uninstall",
		Items: []domain.Finding{
			{ID: "native", Name: "Example", DisplayPath: "Example", Action: domain.ActionNative, Status: domain.StatusPlanned, NativeCommand: testNativeUninstallCommand()},
			{ID: "remnant", Path: filepath.Join(root, "example"), DisplayPath: filepath.Join(root, "example"), Action: domain.ActionTrash, Status: domain.StatusPlanned, Fingerprint: domain.Fingerprint{Size: 0, ModTime: time.Time{}}},
			{ID: "aftercare", DisplayPath: testManagedCommandPath(), Action: domain.ActionCommand, Status: domain.StatusPlanned, CommandPath: testManagedCommandPath(), TaskPhase: "aftercare"},
		},
	}
	// Fix fingerprint for remnant path.
	remnantPath := uninstallPlan.Items[1].Path
	if err := os.WriteFile(remnantPath, []byte("remnant"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(remnantPath)
	if err != nil {
		t.Fatal(err)
	}
	uninstallPlan.Items[1].Fingerprint = domain.Fingerprint{Mode: uint32(info.Mode()), Size: info.Size(), ModTime: info.ModTime()}

	var uninstallProgress []domain.ExecutionProgress
	if _, err := service.ExecuteWithProgress(context.Background(), uninstallPlan, ExecuteOptions{NativeUninstall: true}, func(update domain.ExecutionProgress) {
		uninstallProgress = append(uninstallProgress, update)
	}); err != nil {
		t.Fatal(err)
	}

	found := []string{}
	for _, update := range uninstallProgress {
		if update.Event == domain.ProgressEventSection && update.Phase == domain.ProgressPhaseStarting {
			found = append(found, update.SectionLabel)
		}
	}
	for _, needle := range []string{"Native handoff", "Remnants", "Aftercare"} {
		if !containsString(found, needle) {
			t.Fatalf("expected uninstall section %q in %+v", needle, found)
		}
	}

	optimizePlan := domain.ExecutionPlan{
		Command: "optimize",
		Items: []domain.Finding{
			{ID: "repair", DisplayPath: testManagedCommandPath(), Action: domain.ActionCommand, Status: domain.StatusPlanned, CommandPath: testManagedCommandPath(), TaskPhase: "repair"},
			{ID: "refresh", DisplayPath: testManagedCommandPath(), Action: domain.ActionCommand, Status: domain.StatusPlanned, CommandPath: testManagedCommandPath(), TaskPhase: "refresh"},
		},
	}

	var optimizeProgress []domain.ExecutionProgress
	if _, err := service.ExecuteWithProgress(context.Background(), optimizePlan, ExecuteOptions{}, func(update domain.ExecutionProgress) {
		optimizeProgress = append(optimizeProgress, update)
	}); err != nil {
		t.Fatal(err)
	}

	found = found[:0]
	for _, update := range optimizeProgress {
		if update.Event == domain.ProgressEventSection && update.Phase == domain.ProgressPhaseStarting {
			found = append(found, update.SectionLabel)
		}
	}
	for _, needle := range []string{"Repair", "Refresh"} {
		if !containsString(found, needle) {
			t.Fatalf("expected optimize section %q in %+v", needle, found)
		}
	}
}

func containsString(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}

func containsSubstring(items []string, needle string) bool {
	for _, item := range items {
		if strings.Contains(item, needle) {
			return true
		}
	}
	return false
}

func TestPolicyAllowedRootsUseBoundaryMatch(t *testing.T) {
	t.Parallel()
	if isUnderAnyRoot("/tmp/protected-cache", []string{"/tmp/protected"}) {
		t.Fatal("expected sibling path not to be treated as protected")
	}
	if !isUnderAnyRoot("/tmp/protected/cache", []string{"/tmp/protected"}) {
		t.Fatal("expected nested path to be treated as protected")
	}
}

func TestExecuteBlocksSymlinkTargets(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{}
	result, err := service.ExecuteWithOptions(context.Background(), domain.ExecutionPlan{
		ScanID: "scan",
		Items: []domain.Finding{{
			ID:          uuid.NewString(),
			Path:        link,
			Action:      domain.ActionTrash,
			Status:      domain.StatusPlanned,
			Fingerprint: domain.Fingerprint{Mode: uint32(info.Mode()), Size: info.Size(), ModTime: info.ModTime()},
		}},
		Policy: domain.ProtectionPolicy{
			AllowedRoots:  []string{root},
			BlockSymlinks: true,
		},
	}, ExecuteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected one result item, got %d", len(result.Items))
	}
	if result.Items[0].Status != domain.StatusProtected {
		t.Fatalf("expected protected status, got %+v", result.Items[0])
	}
	if result.Items[0].Reason != domain.ProtectionSymlink {
		t.Fatalf("expected symlink reason, got %+v", result.Items[0])
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("expected link to remain, stat err: %v", err)
	}
}

func TestScanRejectsTraversalTargets(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	service := &Service{
		Adapter: stubAdapter{},
		Config:  cfg,
	}
	plan, err := service.Scan(context.Background(), ScanOptions{
		Command: "purge",
		Targets: []string{"../etc"},
		DryRun:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlanState != "empty" {
		t.Fatalf("expected empty plan, got %s", plan.PlanState)
	}
	if len(plan.Warnings) == 0 {
		t.Fatal("expected warning for rejected traversal target")
	}
}

func TestScanPreservesScannerRisk(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	project := filepath.Join(root, "app")
	artifact := filepath.Join(project, "dist")
	if err := os.MkdirAll(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "bundle.js"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	service := &Service{
		Adapter: stubAdapter{},
		Config:  cfg,
	}
	plan, err := service.Scan(context.Background(), ScanOptions{
		Command: "purge",
		Targets: []string{artifact},
		DryRun:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("expected one plan item, got %d", len(plan.Items))
	}
	if plan.Items[0].Risk != domain.RiskHigh {
		t.Fatalf("expected recent purge target to stay high risk, got %s", plan.Items[0].Risk)
	}
}

func TestScanOrdersCleanPlanByExecutionCategory(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tempRoot := filepath.Join(root, "temp")
	browserRoot := filepath.Join(root, "browser")
	pkgRoot := filepath.Join(root, "pkg")
	for _, entry := range []struct {
		root string
		dir  string
		size int
	}{
		{root: browserRoot, dir: "Chrome", size: 32},
		{root: pkgRoot, dir: "pnpm", size: 24},
		{root: tempRoot, dir: "tmp-cache", size: 8},
	} {
		target := filepath.Join(entry.root, entry.dir)
		if err := os.MkdirAll(target, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, "data.bin"), []byte(strings.Repeat("x", entry.size)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Default()
	cfg.Profiles["safe"] = []string{"temp_files", "browser_data", "package_manager_caches"}
	service := &Service{
		Adapter: stubAdapter{
			roots: platform.CuratedRoots{
				Temp:           []string{tempRoot},
				Browser:        []string{browserRoot},
				PackageManager: []string{pkgRoot},
			},
		},
		Config: cfg,
	}
	plan, err := service.Scan(context.Background(), ScanOptions{
		Command: "clean",
		Profile: "safe",
		DryRun:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(plan.Items))
	}
	got := []domain.Category{plan.Items[0].Category, plan.Items[1].Category, plan.Items[2].Category}
	want := []domain.Category{domain.CategoryTempFiles, domain.CategoryBrowserData, domain.CategoryPackageCaches}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("expected ordered categories %v, got %v", want, got)
		}
	}
}

func TestBuildUninstallPlanIncludesDiscoveredRemnants(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	bundle := filepath.Join(root, "Example.app")
	remnant := filepath.Join(root, "Library", "Caches", "Example")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(remnant, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "binary"), []byte("app"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(remnant, "cache.bin"), []byte("cache"), 0o644); err != nil {
		t.Fatal(err)
	}
	service := &Service{
		Adapter: uninstallStubAdapter{
			stubAdapter: stubAdapter{name: "darwin", remnants: []string{remnant}},
			apps: []domain.AppEntry{{
				Name:             "example",
				DisplayName:      "Example",
				BundlePath:       bundle,
				UninstallCommand: testNativeUninstallCommand(),
			}},
		},
		Config: config.Default(),
	}
	plan, err := service.BuildUninstallPlan(context.Background(), "Example", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) != 8 {
		t.Fatalf("expected native step, managed aftermath actions, bundle and remnant items, got %d", len(plan.Items))
	}
	foundAdvisory := false
	foundAftermath := false
	foundLoginItems := false
	foundLaunchctlUser := false
	foundLaunchctlSystem := false
	foundRemnant := false
	for _, item := range plan.Items {
		if item.RuleID == "uninstall.native_step" {
			foundAdvisory = true
			if item.Action != domain.ActionNative {
				t.Fatalf("expected native uninstall step to be native action, got %s", item.Action)
			}
			if item.Status != domain.StatusPlanned {
				t.Fatalf("expected native uninstall step to stay planned, got %s", item.Status)
			}
		}
		if item.RuleID == "uninstall.aftermath.launchservices" {
			foundAftermath = true
			if item.Action != domain.ActionCommand {
				t.Fatalf("expected launchservices aftermath step to be managed, got %s", item.Action)
			}
			if item.TaskPhase != "refresh" || item.TaskImpact == "" {
				t.Fatalf("expected launchservices aftermath metadata, got phase=%q impact=%q", item.TaskPhase, item.TaskImpact)
			}
			if len(item.TaskVerify) != 1 || len(item.SuggestedBy) != 1 || item.SuggestedBy[0] != "LaunchServices" {
				t.Fatalf("expected launchservices verify/suggested metadata, got verify=%v suggested=%v", item.TaskVerify, item.SuggestedBy)
			}
		}
		if item.RuleID == "uninstall.aftermath.login-items" {
			foundLoginItems = true
			if item.Action != domain.ActionCommand {
				t.Fatalf("expected login items aftermath step to be managed, got %s", item.Action)
			}
			if item.TaskPhase != "aftercare" || item.TaskImpact == "" {
				t.Fatalf("expected login items aftermath metadata, got phase=%q impact=%q", item.TaskPhase, item.TaskImpact)
			}
			if len(item.TaskVerify) != 1 || len(item.SuggestedBy) != 1 || item.SuggestedBy[0] != "Login items" {
				t.Fatalf("expected login items verify/suggested metadata, got verify=%v suggested=%v", item.TaskVerify, item.SuggestedBy)
			}
		}
		if item.RuleID == "uninstall.aftermath.launchctl-user" {
			foundLaunchctlUser = true
			if item.TaskPhase != "aftercare" || len(item.TaskVerify) != 1 {
				t.Fatalf("expected user launchctl metadata, got phase=%q verify=%v", item.TaskPhase, item.TaskVerify)
			}
		}
		if item.RuleID == "uninstall.aftermath.launchctl-system" {
			foundLaunchctlSystem = true
			if item.TaskPhase != "secure" || !item.RequiresAdmin {
				t.Fatalf("expected system launchctl secure metadata, got phase=%q admin=%v", item.TaskPhase, item.RequiresAdmin)
			}
		}
		if item.Path == remnant {
			foundRemnant = true
			if item.Source != "Example remnants" {
				t.Fatalf("expected discovered remnant source, got %s", item.Source)
			}
		}
	}
	if !foundAdvisory {
		t.Fatal("expected native uninstall advisory to be included")
	}
	if !foundAftermath {
		t.Fatal("expected uninstall aftermath advisory to be included")
	}
	if !foundLoginItems {
		t.Fatal("expected login items cleanup to be included")
	}
	if !foundLaunchctlUser || !foundLaunchctlSystem {
		t.Fatalf("expected launchctl cleanup findings, user=%v system=%v", foundLaunchctlUser, foundLaunchctlSystem)
	}
	if !foundRemnant {
		t.Fatal("expected discovered remnant to be included")
	}
}

func TestBuildUninstallPlanProtectsUnsafeNativeCommand(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	bundle := filepath.Join(root, "Example.app")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	service := &Service{
		Adapter: uninstallStubAdapter{
			stubAdapter: stubAdapter{},
			apps: []domain.AppEntry{{
				Name:             "example",
				DisplayName:      "Example",
				BundlePath:       bundle,
				UninstallCommand: "cleanup-helper --silent",
			}},
		},
		Config: config.Default(),
	}
	plan, err := service.BuildUninstallPlan(context.Background(), "Example", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) == 0 {
		t.Fatal("expected uninstall plan items")
	}
	if plan.Items[0].Status != domain.StatusProtected {
		t.Fatalf("expected unsafe native command to be protected, got %s", plan.Items[0].Status)
	}
	if plan.Items[0].Policy.Reason != domain.ProtectionUnsafeCommand {
		t.Fatalf("expected unsafe command reason, got %s", plan.Items[0].Policy.Reason)
	}
}

func TestBuildUninstallPlanFallsBackToNamedRemnantsWhenAppMissing(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	remnant := filepath.Join(root, "Library", "Application Support", "Example App")
	if err := os.MkdirAll(remnant, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(remnant, "state.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	service := &Service{
		Adapter: uninstallStubAdapter{
			stubAdapter: stubAdapter{remnants: []string{remnant}},
		},
		Config: config.Default(),
	}
	plan, err := service.BuildUninstallPlan(context.Background(), "Example App", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("expected remnant-only uninstall plan, got %+v", plan.Items)
	}
	if plan.Items[0].Path != remnant {
		t.Fatalf("expected remnant path %s, got %+v", remnant, plan.Items[0])
	}
	if plan.Items[0].Source != "Example App remnants" {
		t.Fatalf("expected discovered remnant source, got %s", plan.Items[0].Source)
	}
	if len(plan.Warnings) == 0 || !strings.Contains(plan.Warnings[0], "name only") {
		t.Fatalf("expected fallback warning, got %+v", plan.Warnings)
	}
	if len(plan.Targets) != 1 || plan.Targets[0] != "Example App" {
		t.Fatalf("expected original uninstall target to be preserved, got %+v", plan.Targets)
	}
}

func TestBuildUninstallPlanReturnsEmptyPlanWhenAppAndRemnantsAreGone(t *testing.T) {
	t.Parallel()
	service := &Service{
		Adapter: uninstallStubAdapter{},
		Config:  config.Default(),
	}
	plan, err := service.BuildUninstallPlan(context.Background(), "Example App", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlanState != "empty" {
		t.Fatalf("expected empty uninstall plan, got %s", plan.PlanState)
	}
	if len(plan.Items) != 0 {
		t.Fatalf("expected no uninstall items, got %+v", plan.Items)
	}
	if len(plan.Warnings) < 2 {
		t.Fatalf("expected fallback warnings, got %+v", plan.Warnings)
	}
	if !strings.Contains(strings.Join(plan.Warnings, " | "), "not currently listed as installed") {
		t.Fatalf("expected fallback warning, got %+v", plan.Warnings)
	}
	if !strings.Contains(strings.Join(plan.Warnings, " | "), "No installed app or leftover files were found") {
		t.Fatalf("expected empty follow-up warning, got %+v", plan.Warnings)
	}
}

func TestBuildUninstallPlanGuidesToRemoveForSIFT(t *testing.T) {
	t.Parallel()
	service := &Service{
		Adapter: uninstallStubAdapter{},
		Config:  config.Default(),
	}
	plan, err := service.BuildUninstallPlan(context.Background(), "SIFT", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(plan.Warnings, " | "), "sift remove") {
		t.Fatalf("expected guidance toward sift remove, got %+v", plan.Warnings)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("expected advisory-only uninstall plan for SIFT, got %+v", plan.Items)
	}
	if plan.Items[0].RuleID != "uninstall.sift" || plan.Items[0].Action != domain.ActionAdvisory {
		t.Fatalf("expected SIFT advisory item, got %+v", plan.Items[0])
	}
}

func TestBuildUninstallPlanMatchesNormalizedAppNames(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	bundle := filepath.Join(root, "Example App.app")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	service := &Service{
		Adapter: uninstallStubAdapter{
			apps: []domain.AppEntry{{
				Name:        "exampleapp",
				DisplayName: "Example App",
				BundlePath:  bundle,
			}},
		},
		Config: config.Default(),
	}
	plan, err := service.BuildUninstallPlan(context.Background(), "Example-App", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("expected bundle-only uninstall plan, got %+v", plan.Items)
	}
	if plan.Items[0].Path != bundle {
		t.Fatalf("expected matched bundle path %s, got %+v", bundle, plan.Items[0])
	}
}

func TestBuildBatchUninstallPlanMergesTargetsAndItems(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	first := filepath.Join(root, "Example.app")
	second := filepath.Join(root, "Builder.app")
	for _, dir := range []string{first, second} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	service := &Service{
		Adapter: uninstallStubAdapter{
			stubAdapter: stubAdapter{},
			apps: []domain.AppEntry{
				{
					Name:             "example",
					DisplayName:      "Example",
					BundlePath:       first,
					UninstallCommand: testNativeUninstallCommand(),
				},
				{
					Name:        "builder",
					DisplayName: "Builder",
					BundlePath:  second,
				},
			},
		},
		Config: config.Default(),
	}

	plan, err := service.BuildBatchUninstallPlan(context.Background(), []string{"Example", "Builder"}, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Command != "uninstall" {
		t.Fatalf("expected uninstall command, got %+v", plan)
	}
	if len(plan.Targets) != 2 {
		t.Fatalf("expected two uninstall targets, got %+v", plan.Targets)
	}
	if len(plan.Items) < 3 {
		t.Fatalf("expected combined uninstall items, got %+v", plan.Items)
	}
}

func TestExecuteWithOptionsSkipsNativeUninstallWhenFlagDisabled(t *testing.T) {
	t.Parallel()
	service := &Service{}
	result, err := service.ExecuteWithOptions(context.Background(), domain.ExecutionPlan{
		ScanID:  "scan",
		Command: "uninstall",
		Targets: []string{"Example"},
		Items: []domain.Finding{{
			ID:            uuid.NewString(),
			Action:        domain.ActionNative,
			Status:        domain.StatusPlanned,
			DisplayPath:   "MsiExec.exe /X{ABC-123}",
			NativeCommand: "MsiExec.exe /X{ABC-123}",
		}},
	}, ExecuteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].Status != domain.StatusSkipped {
		t.Fatalf("expected skipped native item, got %+v", result.Items)
	}
}

func TestExecuteWithOptionsLaunchesNativeUninstallWhenEnabled(t *testing.T) {
	root := t.TempDir()
	remnant := filepath.Join(root, "example-support")
	if err := os.WriteFile(remnant, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	fingerprint, err := currentFingerprint(remnant)
	if err != nil {
		t.Fatal(err)
	}
	original := startNativeProcess
	defer func() {
		startNativeProcess = original
	}()
	var launched nativeCommand
	startNativeProcess = func(ctx context.Context, command nativeCommand) error {
		launched = command
		return nil
	}
	service := &Service{}
	result, err := service.ExecuteWithOptions(context.Background(), domain.ExecutionPlan{
		ScanID:  "scan",
		Command: "uninstall",
		Targets: []string{"Example"},
		Items: []domain.Finding{
			{
				ID:            uuid.NewString(),
				Action:        domain.ActionNative,
				Status:        domain.StatusPlanned,
				DisplayPath:   "MsiExec.exe /X{ABC-123}",
				NativeCommand: "MsiExec.exe /X{ABC-123}",
			},
			{
				ID:          uuid.NewString(),
				Path:        remnant,
				DisplayPath: remnant,
				Action:      domain.ActionTrash,
				Status:      domain.StatusPlanned,
				Fingerprint: fingerprint,
			},
		},
	}, ExecuteOptions{NativeUninstall: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 2 || result.Items[0].Status != domain.StatusCompleted {
		t.Fatalf("expected completed native item, got %+v", result.Items)
	}
	if result.Items[1].Status != domain.StatusFailed && result.Items[1].Status != domain.StatusDeleted {
		t.Fatalf("expected remnant cleanup to continue after native handoff, got %+v", result.Items)
	}
	if launched.Path != "MsiExec.exe" {
		t.Fatalf("expected native launcher to run msiexec, got %+v", launched)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected one execution warning, got %+v", result.Warnings)
	}
	if len(result.FollowUpCommands) != 0 {
		t.Fatalf("did not expect rerun uninstall follow-up command, got %+v", result.FollowUpCommands)
	}
}

func TestExecuteWithOptionsSkipsOsaScriptCommandInCiSafeMode(t *testing.T) {
	t.Setenv("SIFT_TEST_MODE", "ci-safe")

	called := false
	service := &Service{
		RunCommand: func(context.Context, string, ...string) error {
			called = true
			return nil
		},
	}
	result, err := service.ExecuteWithOptions(context.Background(), domain.ExecutionPlan{
		ScanID: "scan",
		Items: []domain.Finding{{
			ID:          uuid.NewString(),
			Action:      domain.ActionCommand,
			Status:      domain.StatusPlanned,
			DisplayPath: "Remove login items",
			CommandPath: testDialogSensitiveCommandPath(),
			CommandArgs: []string{"-e", "return 1"},
		}},
	}, ExecuteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("expected osascript command to be skipped in ci-safe mode")
	}
	if len(result.Items) != 1 || result.Items[0].Status != domain.StatusSkipped || !strings.Contains(result.Items[0].Message, "ci-safe") {
		t.Fatalf("expected ci-safe skipped command result, got %+v", result.Items)
	}
}

func TestExecuteWithOptionsSkipsAdminCommandInCiSafeMode(t *testing.T) {
	t.Setenv("SIFT_TEST_MODE", "ci-safe")

	called := false
	service := &Service{
		RunCommand: func(context.Context, string, ...string) error {
			called = true
			return nil
		},
	}
	result, err := service.ExecuteWithOptions(context.Background(), domain.ExecutionPlan{
		ScanID: "scan",
		Policy: domain.ProtectionPolicy{AllowAdmin: true},
		Items: []domain.Finding{{
			ID:            uuid.NewString(),
			Action:        domain.ActionCommand,
			Status:        domain.StatusPlanned,
			DisplayPath:   testAdminCommandPath() + " " + testManagedCommandPath(),
			CommandPath:   testAdminCommandPath(),
			CommandArgs:   []string{testManagedCommandPath()},
			RequiresAdmin: true,
		}},
	}, ExecuteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("expected admin command to be skipped in ci-safe mode")
	}
	if len(result.Items) != 1 || result.Items[0].Status != domain.StatusSkipped || !strings.Contains(result.Items[0].Message, "ci-safe") {
		t.Fatalf("expected ci-safe skipped admin command result, got %+v", result.Items)
	}
}

func TestExecuteWithOptionsBatchNativeLaunchAddsContinuationWarning(t *testing.T) {
	original := startNativeProcess
	defer func() {
		startNativeProcess = original
	}()
	startNativeProcess = func(ctx context.Context, command nativeCommand) error {
		return nil
	}
	service := &Service{}
	result, err := service.ExecuteWithOptions(context.Background(), domain.ExecutionPlan{
		ScanID:  "scan",
		Command: "uninstall",
		Targets: []string{"Example", "Builder"},
		Items: []domain.Finding{{
			ID:            uuid.NewString(),
			Action:        domain.ActionNative,
			Status:        domain.StatusPlanned,
			DisplayPath:   "MsiExec.exe /X{ABC-123}",
			NativeCommand: "MsiExec.exe /X{ABC-123}",
		}},
	}, ExecuteOptions{NativeUninstall: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.FollowUpCommands) != 0 {
		t.Fatalf("did not expect rerun uninstall follow-up commands, got %+v", result.FollowUpCommands)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "continued with remnant cleanup and aftercare") {
		t.Fatalf("expected continuation warning, got %+v", result.Warnings)
	}
}

func TestExecuteWithOptionsFailsNativeUninstallOnLauncherError(t *testing.T) {
	original := startNativeProcess
	defer func() {
		startNativeProcess = original
	}()
	startNativeProcess = func(ctx context.Context, command nativeCommand) error {
		return errors.New("launcher failed")
	}
	service := &Service{}
	result, err := service.ExecuteWithOptions(context.Background(), domain.ExecutionPlan{
		ScanID: "scan",
		Items: []domain.Finding{{
			ID:            uuid.NewString(),
			Action:        domain.ActionNative,
			Status:        domain.StatusPlanned,
			DisplayPath:   "MsiExec.exe /X{ABC-123}",
			NativeCommand: "MsiExec.exe /X{ABC-123}",
		}},
	}, ExecuteOptions{NativeUninstall: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].Status != domain.StatusFailed {
		t.Fatalf("expected failed native item, got %+v", result.Items)
	}
	if result.Items[0].Reason != domain.ProtectionUnsafeCommand {
		t.Fatalf("expected unsafe command reason on launcher error, got %+v", result.Items[0])
	}
}

func TestExecuteWithOptionsContinuesRemnantsAfterNativeLaunch(t *testing.T) {
	root := t.TempDir()
	remnant := filepath.Join(root, "Example")
	if err := os.MkdirAll(remnant, 0o755); err != nil {
		t.Fatal(err)
	}
	fingerprint, err := currentFingerprint(remnant)
	if err != nil {
		t.Fatal(err)
	}
	original := startNativeProcess
	defer func() {
		startNativeProcess = original
	}()
	startNativeProcess = func(ctx context.Context, command nativeCommand) error {
		return nil
	}
	service := &Service{}
	result, err := service.ExecuteWithOptions(context.Background(), domain.ExecutionPlan{
		ScanID:  "scan",
		Command: "uninstall",
		Targets: []string{"Example"},
		Items: []domain.Finding{
			{
				ID:            uuid.NewString(),
				Action:        domain.ActionNative,
				Status:        domain.StatusPlanned,
				DisplayPath:   "MsiExec.exe /X{ABC-123}",
				NativeCommand: "MsiExec.exe /X{ABC-123}",
			},
			{
				ID:          uuid.NewString(),
				Path:        remnant,
				DisplayPath: remnant,
				Action:      domain.ActionTrash,
				Status:      domain.StatusPlanned,
				Fingerprint: fingerprint,
			},
		},
		Policy: domain.ProtectionPolicy{AllowedRoots: []string{root}, BlockSymlinks: true},
	}, ExecuteOptions{NativeUninstall: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected two result items, got %+v", result.Items)
	}
	if result.Items[0].Status != domain.StatusCompleted {
		t.Fatalf("expected native step to complete, got %+v", result.Items[0])
	}
	if result.Items[1].Status != domain.StatusDeleted {
		t.Fatalf("expected remnant cleanup to continue after native launch, got %+v", result.Items[1])
	}
	if len(result.FollowUpCommands) != 0 {
		t.Fatalf("did not expect rerun uninstall follow-up command, got %+v", result.FollowUpCommands)
	}
	if _, statErr := os.Stat(remnant); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected remnant to be cleaned in the same run, stat err: %v", statErr)
	}
}

func TestUninstallAftermathCommandsIncludeOnlyAdvisoryFollowUps(t *testing.T) {
	t.Parallel()
	got := uninstallAftermathCommands(domain.ExecutionPlan{
		Command: "uninstall",
		Items: []domain.Finding{
			{
				RuleID:      "uninstall.aftermath.launchservices",
				Action:      domain.ActionCommand,
				DisplayPath: "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -r -f -domain local -domain system -domain user",
			},
			{
				RuleID:      "uninstall.aftermath.homebrew",
				Action:      domain.ActionCommand,
				DisplayPath: "/opt/homebrew/bin/brew autoremove",
			},
			{
				RuleID:      "uninstall.aftermath.login-items",
				Action:      domain.ActionCommand,
				DisplayPath: "Remove login items for Example",
			},
			{
				RuleID:      "uninstall.aftermath.launchctl-user",
				Action:      domain.ActionCommand,
				DisplayPath: "/bin/sh -c unload-user",
			},
			{
				RuleID:      "uninstall.aftermath.launchctl-system",
				Action:      domain.ActionCommand,
				DisplayPath: "/usr/bin/sudo /bin/sh -c unload-system",
			},
		},
	})
	if len(got) != 0 {
		t.Fatalf("expected advisory-only aftermath follow-ups, got %+v", got)
	}
}

func TestServiceUpdateNoticeUsesLatestReleaseAndCache(t *testing.T) {
	originalFetch := fetchLatestReleaseTag
	originalCacheDir := updateNoticeCacheDir
	defer func() {
		fetchLatestReleaseTag = originalFetch
		updateNoticeCacheDir = originalCacheDir
	}()
	tmp := t.TempDir()
	updateNoticeCacheDir = func() (string, error) { return tmp, nil }
	fetchCalls := 0
	fetchLatestReleaseTag = func(ctx context.Context) (string, error) {
		fetchCalls++
		return "v9.9.9", nil
	}
	SetVersion("v1.2.3")
	defer SetVersion("dev")
	service := &Service{Adapter: stubAdapter{name: "darwin"}}
	first := service.UpdateNotice(context.Background())
	if !first.Available || first.LatestVersion != "v9.9.9" {
		t.Fatalf("expected update notice, got %+v", first)
	}
	second := service.UpdateNotice(context.Background())
	if fetchCalls != 1 {
		t.Fatalf("expected cached update notice on second call, got %d fetches", fetchCalls)
	}
	if second.LatestVersion != first.LatestVersion || !second.Available {
		t.Fatalf("expected cached notice to match first result, got %+v", second)
	}
}

func TestDarwinLoginItemCleanupFindingUsesManagedOsaScript(t *testing.T) {
	t.Parallel()
	finding, ok := darwinLoginItemCleanupFinding("Example", domain.AppEntry{
		Name:        "example",
		DisplayName: "Example",
		BundlePath:  "/Applications/Example.app",
	})
	if !ok {
		t.Fatal("expected login item cleanup finding")
	}
	if finding.Action != domain.ActionCommand || finding.CommandPath != "/usr/bin/osascript" {
		t.Fatalf("expected osascript managed action, got %+v", finding)
	}
	if len(finding.CommandArgs) == 0 || finding.CommandArgs[0] != "-e" {
		t.Fatalf("expected AppleScript arguments, got %+v", finding.CommandArgs)
	}
	if !strings.Contains(strings.Join(finding.CommandArgs, " "), "targetNames") {
		t.Fatalf("expected targetNames AppleScript list, got %+v", finding.CommandArgs)
	}
}

func TestBuildUninstallPlanProtectsRunningApps(t *testing.T) {
	root := t.TempDir()
	bundle := filepath.Join(root, "Example.app")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	original := listRunningProcesses
	defer func() {
		listRunningProcesses = original
	}()
	listRunningProcesses = func(ctx context.Context) ([]runningProcess, error) {
		return []runningProcess{{Name: "Example"}}, nil
	}
	service := &Service{
		Adapter: uninstallStubAdapter{
			stubAdapter: stubAdapter{},
			apps: []domain.AppEntry{{
				Name:             "example",
				DisplayName:      "Example",
				BundlePath:       bundle,
				UninstallCommand: testNativeUninstallCommand(),
			}},
		},
		Config: config.Default(),
	}
	plan, err := service.BuildUninstallPlan(context.Background(), "Example", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) == 0 {
		t.Fatal("expected uninstall plan items")
	}
	for _, item := range plan.Items {
		if item.Status != domain.StatusProtected {
			t.Fatalf("expected running app protection, got %+v", item)
		}
		if item.Policy.Reason != domain.ProtectionRunningApp {
			t.Fatalf("expected running app reason, got %+v", item.Policy)
		}
	}
	if len(plan.Warnings) == 0 {
		t.Fatal("expected running app warning")
	}
}

func TestScanWarnsWhenPlanPersistenceFails(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "cache.bin"), []byte("cache"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := store.OpenAt(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	service := &Service{
		Adapter: stubAdapter{roots: platform.CuratedRoots{Temp: []string{root}}},
		Config:  config.Default(),
		Store:   st,
	}
	plan, err := service.Scan(context.Background(), ScanOptions{
		Command: "clean",
		RuleIDs: []string{"temp_files"},
		DryRun:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsSubstring(plan.Warnings, "audit persistence unavailable") {
		t.Fatalf("expected plan persistence warning, got %v", plan.Warnings)
	}
}

func TestExecuteWarnsWhenExecutionPersistenceFails(t *testing.T) {
	st, err := store.OpenAt(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	service := &Service{
		Adapter: stubAdapter{},
		Config:  config.Default(),
		Store:   st,
	}
	result, err := service.ExecuteWithOptions(context.Background(), domain.ExecutionPlan{
		ScanID:   "scan",
		Command:  "clean",
		Platform: "stub",
		DryRun:   true,
		Items: []domain.Finding{{
			ID:          "advisory",
			DisplayPath: "manual review",
			Action:      domain.ActionAdvisory,
			Status:      domain.StatusAdvisory,
		}},
	}, ExecuteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !containsSubstring(result.Warnings, "audit persistence unavailable") {
		t.Fatalf("expected execution persistence warning, got %v", result.Warnings)
	}
}

func TestVerifyFingerprintRejectsReplacedFileWithSameSizeAndModTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.bin")
	fixedTime := time.Now().Add(-time.Hour).Round(time.Second)
	if err := os.WriteFile(path, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, fixedTime, fixedTime); err != nil {
		t.Fatal(err)
	}
	fingerprint, err := currentFingerprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprint.Identity == "" {
		t.Skip("platform does not expose stable file identity")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, fixedTime, fixedTime); err != nil {
		t.Fatal(err)
	}
	err = verifyFingerprint(domain.Finding{Path: path, Fingerprint: fingerprint})
	if err == nil {
		t.Fatal("expected replaced file with same mode/size/modtime to fail identity verification")
	}
	if !strings.Contains(err.Error(), "preview identity mismatch") {
		t.Fatalf("expected identity mismatch error, got %v", err)
	}
}

func TestListAppsEnrichesInventoryMetadata(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	bundle := filepath.Join(home, "Applications", "Example.app")
	support := filepath.Join(home, "Library", "Application Support", "1Password")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(support, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "binary"), []byte(strings.Repeat("x", 32)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(support, "vault.db"), []byte(strings.Repeat("y", 16)), 0o644); err != nil {
		t.Fatal(err)
	}
	service := &Service{
		Adapter: inventoryStubAdapter{
			home: home,
			apps: []domain.AppEntry{{
				Name:         "example",
				DisplayName:  "Example",
				BundlePath:   bundle,
				SupportPaths: []string{support},
				Origin:       "user application",
			}},
		},
		Config: config.Default(),
	}

	apps, err := service.ListApps(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 {
		t.Fatalf("expected one app, got %+v", apps)
	}
	if apps[0].ApproxBytes < 48 {
		t.Fatalf("expected approximate bytes to include bundle and support paths, got %+v", apps[0])
	}
	if !apps[0].Sensitive {
		t.Fatalf("expected app to be marked sensitive, got %+v", apps[0])
	}
	if len(apps[0].FamilyMatches) == 0 || apps[0].FamilyMatches[0] != "password_managers" {
		t.Fatalf("expected password manager family match, got %+v", apps[0])
	}
}

func TestBalancedConfirmLevelSkipsSafeCleanConfirmation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	cache := filepath.Join(root, "cache")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "junk.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.ConfirmLevel = "balanced"
	cfg.Profiles["safe"] = []string{"temp_files"}
	service := &Service{
		Adapter: stubAdapter{roots: platform.CuratedRoots{Temp: []string{root}}},
		Config:  cfg,
	}
	plan, err := service.Scan(context.Background(), ScanOptions{
		Command: "clean",
		Profile: "safe",
		DryRun:  false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.RequiresConfirmation {
		t.Fatal("expected balanced safe cleanup to skip confirmation")
	}
}

func TestBalancedConfirmLevelKeepsPurgeConfirmation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	project := filepath.Join(root, "app")
	artifact := filepath.Join(project, "dist")
	if err := os.MkdirAll(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "bundle.js"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.ConfirmLevel = "balanced"
	service := &Service{
		Adapter: stubAdapter{},
		Config:  cfg,
	}
	plan, err := service.Scan(context.Background(), ScanOptions{
		Command: "purge",
		Targets: []string{artifact},
		DryRun:  false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.RequiresConfirmation {
		t.Fatal("expected purge to keep confirmation even in balanced mode")
	}
}

type uninstallStubAdapter struct {
	stubAdapter
	apps []domain.AppEntry
}

func (s uninstallStubAdapter) ListApps(context.Context, bool) ([]domain.AppEntry, error) {
	return s.apps, nil
}

type inventoryStubAdapter struct {
	stubAdapter
	home string
	apps []domain.AppEntry
}

func (s inventoryStubAdapter) ResolveTargets(in []string) []string {
	if len(in) == 1 && in[0] == "~" {
		return []string{s.home}
	}
	return in
}

func (s inventoryStubAdapter) ListApps(context.Context, bool) ([]domain.AppEntry, error) {
	return s.apps, nil
}
