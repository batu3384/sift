package analyze

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewBatchBuildsDirectorySummary(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.db"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "blob.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	previews := PreviewBatch([]string{root})
	preview, ok := previews[root]
	if !ok {
		t.Fatalf("expected preview for %s, got %+v", root, previews)
	}
	if preview.Dirs != 1 || preview.Files != 2 || preview.Total != 3 {
		t.Fatalf("unexpected preview counts: %+v", preview)
	}
	if len(preview.FileSamples) == 0 || len(preview.Names) == 0 {
		t.Fatalf("expected preview names and file samples, got %+v", preview)
	}
}

func TestPreviewReturnsClonedCachedPreview(t *testing.T) {
	ResetCachesForTests()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "alpha.log"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "beta.log"), []byte("beta-beta"), 0o644); err != nil {
		t.Fatal(err)
	}

	first, ok := Preview(root)
	if !ok {
		t.Fatalf("expected preview for %s", root)
	}
	if len(first.FileSamples) == 0 {
		t.Fatalf("expected file samples in first preview, got %+v", first)
	}

	first.FileSamples[0].Name = "mutated"
	first.Names[0] = "changed"

	second, ok := Preview(root)
	if !ok {
		t.Fatalf("expected cached preview for %s", root)
	}
	if second.FileSamples[0].Name == "mutated" || second.Names[0] == "changed" {
		t.Fatalf("expected cached preview clone isolation, got %+v", second)
	}
}

func TestPreviewReturnsFalseForFilesAndMissingPaths(t *testing.T) {
	ResetCachesForTests()

	root := t.TempDir()
	file := filepath.Join(root, "single.txt")
	if err := os.WriteFile(file, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	if preview, ok := Preview(file); ok {
		t.Fatalf("expected file path to skip preview, got %+v", preview)
	}
	if preview, ok := Preview(filepath.Join(root, "missing")); ok {
		t.Fatalf("expected missing path to skip preview, got %+v", preview)
	}
}
