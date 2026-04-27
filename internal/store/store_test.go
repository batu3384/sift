package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/batu3384/sift/internal/domain"
)

func TestSaveAndGetPlan(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "state.db")
	st, err := OpenAt(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	plan := domain.ExecutionPlan{
		ScanID:    "scan-1",
		Command:   "analyze",
		Platform:  "darwin",
		CreatedAt: time.Now().UTC(),
		Totals:    domain.Totals{ItemCount: 1, Bytes: 1024},
		Items: []domain.Finding{{
			ID:          "finding-1",
			Path:        "/tmp/cache",
			DisplayPath: "/tmp/cache",
		}},
	}
	if err := st.SavePlan(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetPlan(context.Background(), "scan-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ScanID != plan.ScanID || got.Totals.Bytes != plan.Totals.Bytes {
		t.Fatalf("unexpected plan roundtrip: %+v", got)
	}
	summary, err := st.BuildStatusSummary(context.Background(), "darwin", 10)
	if err != nil {
		t.Fatal(err)
	}
	if summary.LastScan == nil || summary.LastScan.ScanID != plan.ScanID {
		t.Fatalf("unexpected status summary: %+v", summary)
	}
	if _, err := os.Stat(summary.AuditLogPath); err != nil {
		t.Fatalf("expected audit log file, stat err: %v", err)
	}
	records, err := st.RecentAuditRecords(time.Now().UTC(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) == 0 {
		t.Fatal("expected audit records after saving plan")
	}
	if records[len(records)-1].Kind != "plan" {
		t.Fatalf("expected latest audit kind to be plan, got %s", records[len(records)-1].Kind)
	}
}

func TestLatestExecutionSummarizesCompletedAndProtected(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "state.db")
	st, err := OpenAt(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	result := domain.ExecutionResult{
		ID:               "exec-1",
		ScanID:           "scan-1",
		StartedAt:        time.Now().UTC(),
		FinishedAt:       time.Now().UTC(),
		Warnings:         []string{"Rerun uninstall after the vendor uninstaller finishes for /Users/example/App."},
		FollowUpCommands: []string{`sift uninstall "/Users/example/App"`},
		Items: []domain.OperationResult{
			{FindingID: "a", Status: domain.StatusCompleted},
			{FindingID: "b", Status: domain.StatusDeleted},
			{FindingID: "c", Status: domain.StatusProtected},
			{FindingID: "d", Status: domain.StatusSkipped},
		},
	}
	if err := st.SaveExecution(context.Background(), result); err != nil {
		t.Fatal(err)
	}
	summary, err := st.LatestExecution(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary == nil {
		t.Fatal("expected execution summary")
	}
	if summary.Completed != 1 || summary.Deleted != 1 || summary.Protected != 1 || summary.Skipped != 1 {
		t.Fatalf("unexpected execution summary: %+v", summary)
	}
	if len(summary.Warnings) != 1 {
		t.Fatalf("expected warnings to be preserved, got %+v", summary)
	}
	if len(summary.FollowUpCommands) != 1 {
		t.Fatalf("expected follow-up commands to be preserved, got %+v", summary)
	}
}

func TestGetExecutionForScanReturnsRequestedScanExecution(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "state.db")
	st, err := OpenAt(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	base := time.Now().UTC().Add(-time.Hour)
	first := domain.ExecutionResult{
		ID:         "exec-scan-1",
		ScanID:     "scan-1",
		StartedAt:  base,
		FinishedAt: base.Add(time.Minute),
		Items: []domain.OperationResult{{
			FindingID: "a",
			Status:    domain.StatusCompleted,
			Bytes:     128,
		}},
	}
	second := domain.ExecutionResult{
		ID:         "exec-scan-2",
		ScanID:     "scan-2",
		StartedAt:  base.Add(2 * time.Minute),
		FinishedAt: base.Add(3 * time.Minute),
		Items: []domain.OperationResult{{
			FindingID: "b",
			Status:    domain.StatusDeleted,
			Bytes:     256,
		}},
	}
	if err := st.SaveExecution(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveExecution(context.Background(), second); err != nil {
		t.Fatal(err)
	}

	latest, err := st.LatestExecution(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if latest == nil || latest.ScanID != "scan-2" {
		t.Fatalf("expected latest execution to be scan-2, got %+v", latest)
	}

	got, err := st.GetExecutionForScan(context.Background(), "scan-1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected execution summary for scan-1")
	}
	if got.ID != "exec-scan-1" || got.ScanID != "scan-1" || got.Completed != 1 || got.FreedBytes != 128 {
		t.Fatalf("expected scan-1 execution summary, got %+v", got)
	}

	missing, err := st.GetExecutionForScan(context.Background(), "missing")
	if err != nil {
		t.Fatal(err)
	}
	if missing != nil {
		t.Fatalf("expected nil execution for missing scan, got %+v", missing)
	}
}

func TestSaveAndLoadAppInventory(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "state.db")
	st, err := OpenAt(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	apps := []domain.AppEntry{
		{
			Name:                  "example",
			DisplayName:           "Example",
			BundlePath:            "/Applications/Example.app",
			Origin:                "homebrew cask",
			RequiresAdmin:         true,
			UninstallCommand:      "/Applications/Example Uninstaller.app",
			LastModified:          time.Now().UTC().Round(time.Second),
			SupportPaths:          []string{"/Users/test/Library/Application Support/Example"},
			UninstallHint:         "Review before uninstall.",
			QuietUninstallCommand: "",
		},
	}

	if err := st.SaveAppInventory(context.Background(), "darwin", apps); err != nil {
		t.Fatal(err)
	}
	got, updatedAt, err := st.LoadAppInventory(context.Background(), "darwin")
	if err != nil {
		t.Fatal(err)
	}
	if updatedAt.IsZero() {
		t.Fatal("expected cache updated time")
	}
	if len(got) != 1 || got[0].DisplayName != "Example" || got[0].Origin != "homebrew cask" {
		t.Fatalf("unexpected app inventory roundtrip: %+v", got)
	}
}
