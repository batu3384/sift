package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/batu3384/sift/internal/domain"
)

func TestAuditLogPathForUsesDatePartition(t *testing.T) {
	t.Parallel()

	st := &Store{path: filepath.Join(t.TempDir(), "state.db")}
	got := st.auditLogPathFor(time.Date(2026, time.March, 17, 12, 0, 0, 0, time.UTC))
	if want := filepath.Join(filepath.Dir(st.path), "audit", "2026-03-17.ndjson"); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestRecentAuditRecordsReturnsLatestLimitedRecords(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "state.db")
	st, err := OpenAt(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	for i := 0; i < 3; i++ {
		if err := st.appendAuditRecord("plan", "scan", map[string]int{"index": i}); err != nil {
			t.Fatal(err)
		}
	}
	records, err := st.RecentAuditRecords(time.Now().UTC(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 audit records, got %d", len(records))
	}
}

func TestAuditRecordsForScanFiltersAndLimitsAfterFiltering(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "state.db")
	st, err := OpenAt(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	if err := st.appendAuditRecord("plan", "scan-1", map[string]int{"index": 1}); err != nil {
		t.Fatal(err)
	}
	if err := st.appendAuditRecord("plan", "scan-2", map[string]int{"index": 2}); err != nil {
		t.Fatal(err)
	}
	if err := st.appendAuditRecord("execution", "scan-1", map[string]int{"index": 3}); err != nil {
		t.Fatal(err)
	}

	records, err := st.AuditRecordsForScan(time.Now().UTC(), "scan-1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(records))
	}
	if records[0].ScanID != "scan-1" || records[0].Kind != "execution" {
		t.Fatalf("expected latest scan-1 execution audit record, got %+v", records[0])
	}
}

func TestBuildStatusSummaryCalculatesDeltaBetweenLatestScans(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "state.db")
	st, err := OpenAt(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	first := domain.ExecutionPlan{
		ScanID:    "scan-1",
		Command:   "clean",
		Platform:  "darwin",
		CreatedAt: time.Now().UTC().Add(-time.Hour),
		Totals:    domain.Totals{ItemCount: 2, Bytes: 100},
	}
	second := domain.ExecutionPlan{
		ScanID:    "scan-2",
		Command:   "clean",
		Platform:  "darwin",
		CreatedAt: time.Now().UTC(),
		Totals:    domain.Totals{ItemCount: 5, Bytes: 240},
	}
	if err := st.SavePlan(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := st.SavePlan(context.Background(), second); err != nil {
		t.Fatal(err)
	}

	summary, err := st.BuildStatusSummary(context.Background(), "darwin", 10)
	if err != nil {
		t.Fatal(err)
	}
	if summary.LastScan == nil || summary.LastScan.ScanID != "scan-2" {
		t.Fatalf("expected latest scan to be scan-2, got %+v", summary.LastScan)
	}
	if summary.PreviousScan == nil || summary.PreviousScan.ScanID != "scan-1" {
		t.Fatalf("expected previous scan to be scan-1, got %+v", summary.PreviousScan)
	}
	if summary.DeltaBytes != 140 || summary.DeltaItems != 3 {
		t.Fatalf("expected delta bytes/items 140/3, got %d/%d", summary.DeltaBytes, summary.DeltaItems)
	}
}

func TestRecentAuditRecordsIgnoresInvalidJSONLines(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	st := &Store{path: filepath.Join(root, "state.db")}
	auditPath := st.auditLogPathFor(time.Now().UTC())
	if err := os.MkdirAll(filepath.Dir(auditPath), 0o755); err != nil {
		t.Fatal(err)
	}
	payload := []byte("{invalid json}\n{\"kind\":\"plan\",\"timestamp\":\"2026-03-17T00:00:00Z\",\"payload\":{}}\n")
	if err := os.WriteFile(auditPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	records, err := st.RecentAuditRecords(time.Now().UTC(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Kind != "plan" {
		t.Fatalf("expected one valid audit record, got %+v", records)
	}
}
