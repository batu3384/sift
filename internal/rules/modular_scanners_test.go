package rules

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/batu3384/sift/internal/domain"
)

type adminStubAdapter struct {
	stubAdapter
	admin map[string]bool
}

func (a adminStubAdapter) IsAdminPath(path string) bool {
	return a.admin[domain.NormalizePath(path)]
}

func TestScanTargetsSkipsMissingAndSymlinkAndSetsAdmin(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "target-cache")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "blob.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "target-link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(root, "missing")

	adapter := adminStubAdapter{
		stubAdapter: stubAdapter{name: "test"},
		admin:       map[string]bool{domain.NormalizePath(target): true},
	}
	findings, warnings, err := scanTargets(context.Background(), []string{missing, link, target}, adapter)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %+v", findings)
	}
	if findings[0].Path != domain.NormalizePath(target) {
		t.Fatalf("expected target finding, got %+v", findings[0])
	}
	if !findings[0].RequiresAdmin {
		t.Fatalf("expected admin target to be marked")
	}
	warningText := strings.Join(warnings, " | ")
	if !strings.Contains(warningText, "target not found") || !strings.Contains(warningText, "symlink skipped") {
		t.Fatalf("expected missing and symlink warnings, got %v", warnings)
	}
}

func TestScanCuratedRootLeafUsesCleanupSourceLabel(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "Library", "Caches", "com.reincubate.camo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Add(-time.Hour)
	payload := filepath.Join(root, "thumb.bin")
	if err := os.WriteFile(payload, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(root, now, now); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(payload, now, now); err != nil {
		t.Fatal(err)
	}

	findings, warnings, err := scanCuratedRoot(context.Background(), root, domain.CategoryBrowserData, domain.RiskReview, domain.ActionTrash, "fallback", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 leaf finding, got %+v", findings)
	}
	if findings[0].Name != "Camo cache" || findings[0].Source != "Camo cache" {
		t.Fatalf("expected cleanup source label to be applied, got %+v", findings[0])
	}
	if findings[0].Path != domain.NormalizePath(root) {
		t.Fatalf("expected leaf path to equal root, got %+v", findings[0])
	}
}

func TestDedupeLabeledRootsDropsEmptyAndDuplicates(t *testing.T) {
	t.Parallel()

	in := []labeledRoot{
		{},
		{path: "/tmp/a", label: "Cache"},
		{path: "/tmp/a", label: "Cache"},
		{path: "/tmp/a", label: "Logs"},
		{path: "/tmp/b", label: "Cache"},
	}
	out := dedupeLabeledRoots(in)
	if len(out) != 3 {
		t.Fatalf("expected 3 unique labeled roots, got %+v", out)
	}
	if out[0] != (labeledRoot{path: "/tmp/a", label: "Cache"}) ||
		out[1] != (labeledRoot{path: "/tmp/a", label: "Logs"}) ||
		out[2] != (labeledRoot{path: "/tmp/b", label: "Cache"}) {
		t.Fatalf("unexpected dedupe order/content: %+v", out)
	}
}
