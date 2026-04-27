package report

import (
	"archive/zip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/store"
)

func TestBundleAtWritesZip(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "state.db")
	st, err := store.OpenAt(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	plan := domain.ExecutionPlan{
		ScanID:    "scan-1",
		Command:   "report",
		Platform:  "darwin",
		CreatedAt: time.Now().UTC(),
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	plan.Targets = []string{filepath.Join(home, "sensitive")}
	plan.Warnings = []string{filepath.Join(home, "warnings", "target rejected")}
	plan.Policy = domain.ProtectionPolicy{
		ProtectedPaths:          []string{filepath.Join(home, "Library", "Secrets")},
		UserProtectedPaths:      []string{filepath.Join(home, "Projects", "keep-me")},
		SystemProtectedPaths:    []string{filepath.Join(home, "Library", "Application Support", "Google", "Chrome")},
		ProtectedPathExceptions: []string{filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Default", "Code Cache")},
		AllowedRoots:            []string{filepath.Join(home, "Library", "Caches")},
	}
	plan.Items = []domain.Finding{{
		ID:            "item-1",
		Path:          filepath.Join(home, "Library", "Caches", "Example"),
		DisplayPath:   filepath.Join(home, "Library", "Caches", "Example"),
		NativeCommand: `"` + filepath.Join(home, "Applications", "Example.app", "Contents", "MacOS", "uninstall") + `" --quiet`,
		Recovery: domain.RecoveryHint{
			Location: filepath.Join(home, ".Trash"),
		},
	}}
	if err := st.SavePlan(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveExecution(context.Background(), domain.ExecutionResult{
		ID:               "exec-1",
		ScanID:           plan.ScanID,
		StartedAt:        time.Now().UTC(),
		FinishedAt:       time.Now().UTC(),
		Warnings:         []string{filepath.Join(home, "follow-up", "rerun uninstall")},
		FollowUpCommands: []string{`sift uninstall "` + filepath.Join(home, "follow-up", "rerun uninstall") + `"`},
	}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "reports")
	cfg := config.Default()
	cfg.ProtectedPaths = []string{filepath.Join(home, "Projects", "keep-me")}
	cfg.PurgeSearchPaths = []string{filepath.Join(home, "Projects")}
	_, bundlePath, err := BundleAt(context.Background(), dir, st, plan, cfg)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != 8 {
		t.Fatalf("expected 8 files in bundle, got %d", len(reader.File))
	}
	seenStatus := false
	seenAudit := false
	for _, file := range reader.File {
		if file.Name == "status_summary.json" {
			seenStatus = true
		}
		if file.Name == "audit.json" {
			seenAudit = true
		}
		if file.Name == "latest_execution.json" {
			rc, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			raw, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatal(err)
			}
			var got store.ExecutionSummary
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if len(got.Warnings) != 1 || got.Warnings[0] == filepath.Join(home, "follow-up", "rerun uninstall") {
				t.Fatalf("expected redacted execution warning, got %+v", got.Warnings)
			}
			if len(got.FollowUpCommands) != 1 || strings.Contains(got.FollowUpCommands[0], filepath.Join(home, "follow-up", "rerun uninstall")) {
				t.Fatalf("expected redacted follow-up command, got %+v", got.FollowUpCommands)
			}
		}
		if file.Name == "recent_scans.json" {
			rc, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			raw, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatal(err)
			}
			var got []store.RecentScan
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || len(got[0].Warnings) != 1 || got[0].Warnings[0] == filepath.Join(home, "warnings", "target rejected") {
				t.Fatalf("expected redacted recent scan warnings, got %+v", got)
			}
		}
		if file.Name == "config.json" {
			rc, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			raw, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatal(err)
			}
			var got config.Config
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if len(got.ProtectedPaths) != 1 || got.ProtectedPaths[0] == filepath.Join(home, "Projects", "keep-me") {
				t.Fatalf("expected redacted protected paths, got %+v", got.ProtectedPaths)
			}
			if len(got.PurgeSearchPaths) != 1 || got.PurgeSearchPaths[0] == filepath.Join(home, "Projects") {
				t.Fatalf("expected redacted purge search paths, got %+v", got.PurgeSearchPaths)
			}
		}
		if file.Name == "status_summary.json" {
			rc, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			raw, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatal(err)
			}
			var got store.StatusSummary
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if got.LastScan == nil || len(got.LastScan.Warnings) != 1 || got.LastScan.Warnings[0] == filepath.Join(home, "warnings", "target rejected") {
				t.Fatalf("expected redacted status summary warnings, got %+v", got.LastScan)
			}
		}
		if file.Name != "plan.json" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		var got domain.ExecutionPlan
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		if len(got.Targets) != 1 || got.Targets[0] == plan.Targets[0] {
			t.Fatalf("expected redacted target path, got %v", got.Targets)
		}
		if len(got.Warnings) != 1 || got.Warnings[0] == filepath.Join(home, "warnings", "target rejected") {
			t.Fatalf("expected redacted plan warnings, got %+v", got.Warnings)
		}
		if len(got.Policy.ProtectedPaths) != 1 || got.Policy.ProtectedPaths[0] == filepath.Join(home, "Library", "Secrets") {
			t.Fatalf("expected redacted policy protected paths, got %+v", got.Policy.ProtectedPaths)
		}
		if len(got.Policy.UserProtectedPaths) != 1 || got.Policy.UserProtectedPaths[0] == filepath.Join(home, "Projects", "keep-me") {
			t.Fatalf("expected redacted policy user protected paths, got %+v", got.Policy.UserProtectedPaths)
		}
		if len(got.Policy.SystemProtectedPaths) != 1 || got.Policy.SystemProtectedPaths[0] == filepath.Join(home, "Library", "Application Support", "Google", "Chrome") {
			t.Fatalf("expected redacted policy system protected paths, got %+v", got.Policy.SystemProtectedPaths)
		}
		if len(got.Policy.ProtectedPathExceptions) != 1 || got.Policy.ProtectedPathExceptions[0] == filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Default", "Code Cache") {
			t.Fatalf("expected redacted policy exceptions, got %+v", got.Policy.ProtectedPathExceptions)
		}
		if len(got.Policy.AllowedRoots) != 1 || got.Policy.AllowedRoots[0] == filepath.Join(home, "Library", "Caches") {
			t.Fatalf("expected redacted allowed roots, got %+v", got.Policy.AllowedRoots)
		}
		if len(got.Items) != 1 || strings.Contains(got.Items[0].NativeCommand, filepath.Join(home, "Applications", "Example.app")) {
			t.Fatalf("expected redacted native command, got %+v", got.Items)
		}
	}
	if !seenStatus || !seenAudit {
		t.Fatalf("expected status_summary.json and audit.json, got %v files", len(reader.File))
	}
}

func TestBundleAtUsesExecutionAndAuditRecordsForPlanScan(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "state.db")
	st, err := store.OpenAt(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	base := time.Now().UTC().Add(-time.Hour)
	requestedPlan := domain.ExecutionPlan{
		ScanID:    "scan-1",
		Command:   "clean",
		Platform:  "darwin",
		CreatedAt: base,
	}
	latestPlan := domain.ExecutionPlan{
		ScanID:    "scan-2",
		Command:   "clean",
		Platform:  "darwin",
		CreatedAt: base.Add(10 * time.Minute),
	}
	if err := st.SavePlan(context.Background(), requestedPlan); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveExecution(context.Background(), domain.ExecutionResult{
		ID:         "exec-scan-1",
		ScanID:     "scan-1",
		StartedAt:  base.Add(time.Minute),
		FinishedAt: base.Add(2 * time.Minute),
		Items: []domain.OperationResult{{
			FindingID: "old",
			Status:    domain.StatusCompleted,
			Bytes:     100,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.SavePlan(context.Background(), latestPlan); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveExecution(context.Background(), domain.ExecutionResult{
		ID:         "exec-scan-2",
		ScanID:     "scan-2",
		StartedAt:  base.Add(11 * time.Minute),
		FinishedAt: base.Add(12 * time.Minute),
		Items: []domain.OperationResult{{
			FindingID: "new",
			Status:    domain.StatusDeleted,
			Bytes:     200,
		}},
	}); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Diagnostics.Redaction = false
	_, bundlePath, err := BundleAt(context.Background(), filepath.Join(t.TempDir(), "reports"), st, requestedPlan, cfg)
	if err != nil {
		t.Fatal(err)
	}

	var execution store.ExecutionSummary
	readBundleJSON(t, bundlePath, "latest_execution.json", &execution)
	if execution.ScanID != "scan-1" || execution.ID != "exec-scan-1" || execution.FreedBytes != 100 {
		t.Fatalf("expected requested scan execution in bundle, got %+v", execution)
	}

	var records []store.AuditRecord
	readBundleJSON(t, bundlePath, "audit.json", &records)
	if len(records) == 0 {
		t.Fatal("expected scan-scoped audit records")
	}
	for _, record := range records {
		if record.ScanID != "scan-1" {
			t.Fatalf("expected only scan-1 audit records, got %+v", records)
		}
	}

	var status store.StatusSummary
	readBundleJSON(t, bundlePath, "status_summary.json", &status)
	if status.LastExecution == nil || status.LastExecution.ScanID != "scan-1" {
		t.Fatalf("expected status summary execution to match requested scan, got %+v", status.LastExecution)
	}
}

func TestBundleAtReturnsSaveReportError(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "state.db")
	st, err := store.OpenAt(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	_, _, err = BundleAt(context.Background(), filepath.Join(t.TempDir(), "reports"), st, domain.ExecutionPlan{
		ScanID:    "scan-1",
		Command:   "clean",
		Platform:  "darwin",
		CreatedAt: time.Now().UTC(),
	}, config.Default())
	if err == nil {
		t.Fatal("expected SaveReport error to be returned")
	}
}

func readBundleJSON(t *testing.T, bundlePath, name string, dest any) {
	t.Helper()
	reader, err := zip.OpenReader(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, dest); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Fatalf("expected %s in %s", name, bundlePath)
}
