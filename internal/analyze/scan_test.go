package analyze

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func TestCachedScanCoalescesConcurrentLoads(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	target := t.TempDir()
	loader := func(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond)
		return []domain.Finding{{Path: "/tmp/cache", DisplayPath: "/tmp/cache"}}, nil, nil
	}

	errCh := make(chan error, 2)
	for range 2 {
		go func() {
			_, _, err := CachedScan(context.Background(), "disk_usage", []string{target}, loader)
			errCh <- err
		}()
	}
	for range 2 {
		if err := <-errCh; err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("expected loader to run once, got %d", calls.Load())
	}
}

func TestCachedScanReturnsClonedFindingsFromCache(t *testing.T) {
	ResetCachesForTests()

	var calls atomic.Int32
	target := t.TempDir()
	loader := func(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
		calls.Add(1)
		return []domain.Finding{{
			Path:        "/tmp/cache",
			DisplayPath: "/tmp/cache",
			CommandArgs: []string{"one"},
			TaskVerify:  []string{"verify"},
			SuggestedBy: []string{"Disk pressure"},
		}}, []string{"warning"}, nil
	}

	first, warnings, err := CachedScan(context.Background(), "disk_usage", []string{target}, loader)
	if err != nil {
		t.Fatal(err)
	}
	first[0].CommandArgs[0] = "mutated"
	first[0].TaskVerify[0] = "changed"
	first[0].SuggestedBy[0] = "mutated"
	warnings[0] = "changed"

	second, secondWarnings, err := CachedScan(context.Background(), "disk_usage", []string{target}, loader)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected loader to run once with cache hit, got %d", calls.Load())
	}
	if second[0].CommandArgs[0] != "one" || second[0].TaskVerify[0] != "verify" || second[0].SuggestedBy[0] != "Disk pressure" {
		t.Fatalf("expected cloned cached finding, got %+v", second[0])
	}
	if secondWarnings[0] != "warning" {
		t.Fatalf("expected cloned cached warnings, got %v", secondWarnings)
	}
}

func TestCachedScanBypassesCacheForEmptyTargets(t *testing.T) {
	ResetCachesForTests()

	var calls atomic.Int32
	loader := func(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
		calls.Add(1)
		return nil, nil, nil
	}

	for range 2 {
		if _, _, err := CachedScan(context.Background(), "disk_usage", nil, loader); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("expected empty-target scan to bypass cache, got %d calls", calls.Load())
	}
}
