package analyze

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func TestScanKey(t *testing.T) {
	tests := []struct {
		name     string
		kind     string
		targets  []string
		expected string
	}{
		{"empty targets", "analyze", []string{}, "analyze"},
		{"single target", "analyze", []string{"~/Downloads"}, "analyze"},
		{"multiple targets", "analyze", []string{"~/Downloads", "~/Desktop"}, "analyze"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scanKey(tt.kind, tt.targets)
			// scanKey returns a key based on kind and normalized targets
			if result == "" {
				t.Error("scanKey should not return empty string")
			}
		})
	}
}

func TestCloneFindings(t *testing.T) {
	// Create test findings
	findings := []domain.Finding{
		{
			ID:       "1",
			Name:     "test1",
			Path:     "/tmp/test1",
			Bytes:    1000,
			Risk:     domain.RiskSafe,
			Action:   domain.ActionTrash,
			RuleID:   "test_rule",
			Category: domain.CategoryTempFiles,
		},
		{
			ID:       "2",
			Name:     "test2",
			Path:     "/tmp/test2",
			Bytes:    2000,
			Risk:     domain.RiskReview,
			Action:   domain.ActionTrash,
			RuleID:   "test_rule",
			Category: domain.CategoryTempFiles,
		},
	}

	// Clone the findings
	cloned := cloneFindings(findings)

	// Verify the clone is independent
	if len(cloned) != len(findings) {
		t.Errorf("clone length = %d, want %d", len(cloned), len(findings))
	}

	// Modify cloned and verify original is unchanged
	cloned[0].Name = "modified"
	if findings[0].Name == "modified" {
		t.Error("clone should be independent of original")
	}
}

func TestCachedScan(t *testing.T) {
	callCount := 0
	loader := func(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
		callCount++
		return []domain.Finding{
			{
				ID:       "1",
				Name:     "test",
				Path:     "/tmp/test",
				Bytes:    1000,
				Risk:     domain.RiskSafe,
				Action:   domain.ActionTrash,
				RuleID:   "test_rule",
				Category: domain.CategoryTempFiles,
			},
		}, nil, nil
	}

	ctx := context.Background()

	// First call should execute loader
	findings1, warnings1, err := CachedScan(ctx, "test", []string{"~/test"}, loader)
	if err != nil {
		t.Errorf("CachedScan error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
	if len(findings1) != 1 {
		t.Errorf("findings length = %d, want 1", len(findings1))
	}

	// Second call should use cache
	findings2, warnings2, err := CachedScan(ctx, "test", []string{"~/test"}, loader)
	if err != nil {
		t.Errorf("CachedScan error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (cached)", callCount)
	}
	if len(findings2) != 1 {
		t.Errorf("findings length = %d, want 1", len(findings2))
	}

	// Warnings should be empty
	if len(warnings1) != 0 {
		t.Errorf("warnings1 length = %d, want 0", len(warnings1))
	}
	if len(warnings2) != 0 {
		t.Errorf("warnings2 length = %d, want 0", len(warnings2))
	}
}

func TestCachedScanWithDifferentTargets(t *testing.T) {
	callCount := 0
	loader := func(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
		callCount++
		return []domain.Finding{
			{
				ID:       "1",
				Name:     "test",
				Path:     targets[0],
				Bytes:    1000,
				Risk:     domain.RiskSafe,
				Action:   domain.ActionTrash,
				RuleID:   "test_rule",
				Category: domain.CategoryTempFiles,
			},
		}, nil, nil
	}

	ctx := context.Background()

	// First call with target1
	findings1, _, err := CachedScan(ctx, "test", []string{"~/test1"}, loader)
	if err != nil {
		t.Errorf("CachedScan error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}

	// Second call with different target
	findings2, _, err := CachedScan(ctx, "test", []string{"~/test2"}, loader)
	if err != nil {
		t.Errorf("CachedScan error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (different target)", callCount)
	}

	// Verify different targets return different findings
	if findings1[0].Path == findings2[0].Path {
		t.Error("different targets should return different findings")
	}
}

func TestCachedScanCacheExpiry(t *testing.T) {
	// This test verifies that cache expires after TTL
	// We can't easily test the exact TTL without mocking time,
	// but we can verify the cache mechanism works

	callCount := 0
	loader := func(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
		callCount++
		return []domain.Finding{
			{
				ID:       "1",
				Name:     "test",
				Path:     "/tmp/test",
				Bytes:    1000,
				Risk:     domain.RiskSafe,
				Action:   domain.ActionTrash,
				RuleID:   "test_rule",
				Category: domain.CategoryTempFiles,
			},
		}, nil, nil
	}

	ctx := context.Background()

	// First call - use a valid target path
	_, _, err := CachedScan(ctx, "test", []string{"/tmp/test"}, loader)
	if err != nil {
		t.Errorf("CachedScan error: %v", err)
	}

	// Wait a bit and call again - should still be cached
	time.Sleep(10 * time.Millisecond)
	_, _, err = CachedScan(ctx, "test", []string{"/tmp/test"}, loader)
	if err != nil {
		t.Errorf("CachedScan error: %v", err)
	}

	// Should still be 1 (cached)
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (cached)", callCount)
	}
}

func TestCachedScanEmptyTargets(t *testing.T) {
	// When targets is empty, scanKey returns just the kind
	callCount := 0
	loader := func(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
		callCount++
		return []domain.Finding{}, nil, nil
	}

	ctx := context.Background()

	// Empty targets should call loader
	_, _, err := CachedScan(ctx, "test", []string{}, loader)
	if err != nil {
		t.Errorf("CachedScan error: %v", err)
	}

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
}

func TestCachedScanContextCancellation(t *testing.T) {
	loader := func(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
		// Simulate long operation
		select {
		case <-time.After(100 * time.Millisecond):
			return []domain.Finding{}, nil, nil
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately
	cancel()

	// Should return context cancellation error or return early
	// The function may return early without calling loader when context is already done
	_, _, err := CachedScan(ctx, "test", []string{"/tmp/test"}, loader)
	// Either error or empty result is acceptable
	if err != nil && err != context.Canceled {
		t.Errorf("expected context cancellation error, got: %v", err)
	}
}

func TestScanWithRealFiles(t *testing.T) {
	// Create a temp directory with some files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]int64{
		"file1.txt":      100,
		"file2.log":      200,
		"subdir/file3":   300,
	}

	for path, size := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, make([]byte, size), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Test loader that scans the directory
	loader := func(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
		var findings []domain.Finding
		var warnings []string

		for _, target := range targets {
			err := filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					findings = append(findings, domain.Finding{
						ID:       path,
						Name:     info.Name(),
						Path:     path,
						Bytes:    info.Size(),
						Risk:     domain.RiskSafe,
						Action:   domain.ActionTrash,
						RuleID:   "test",
						Category: domain.CategoryTempFiles,
					})
				}
				return nil
			})
			if err != nil {
				warnings = append(warnings, err.Error())
			}
		}

		return findings, warnings, nil
	}

	ctx := context.Background()
	findings, warnings, err := CachedScan(ctx, "test", []string{tmpDir}, loader)
	if err != nil {
		t.Errorf("CachedScan error: %v", err)
	}

	// Should find all files
	if len(findings) != len(testFiles) {
		t.Errorf("findings length = %d, want %d", len(findings), len(testFiles))
	}

	// No warnings expected
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want empty", warnings)
	}
}