//go:build darwin

package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/batu3384/sift/internal/platform"
)

func TestLiveIntegrationOpenAndRevealPath(t *testing.T) {
	t.Parallel()
	if !platform.LiveIntegrationEnabled() {
		t.Skip("set SIFT_LIVE_INTEGRATION=1 to run live macOS integration tests")
	}

	root := t.TempDir()
	target := filepath.Join(root, "sample.txt")
	if err := os.WriteFile(target, []byte("live integration"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := OpenPath(target); err != nil {
		t.Fatalf("expected live open to succeed, got %v", err)
	}
	if err := RevealPath(target); err != nil {
		t.Fatalf("expected live reveal to succeed, got %v", err)
	}
}
