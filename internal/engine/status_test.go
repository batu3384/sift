package engine

import (
	"context"
	"testing"
)

func TestRecentScansWithoutStoreReturnsNil(t *testing.T) {
	t.Parallel()

	service := &Service{}
	scans, err := service.RecentScans(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if scans != nil {
		t.Fatalf("expected nil scans without store, got %+v", scans)
	}
}

func TestStatusSummaryWithoutStoreUsesAdapterName(t *testing.T) {
	t.Parallel()

	service := &Service{Adapter: stubAdapter{name: "darwin"}}
	summary, err := service.StatusSummary(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Platform != "darwin" {
		t.Fatalf("expected platform to come from adapter, got %+v", summary)
	}
}
