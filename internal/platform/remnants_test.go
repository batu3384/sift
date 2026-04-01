package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func TestNormalizedNameKeyStripsNoise(t *testing.T) {
	t.Parallel()
	got := normalizedNameKey("Visual Studio Code (Insiders)")
	if got != "visualstudiocodeinsiders" {
		t.Fatalf("unexpected normalized key: %s", got)
	}
}

func TestAliasCandidatesIncludeBundleBase(t *testing.T) {
	t.Parallel()
	app := domain.AppEntry{
		Name:        "code",
		DisplayName: "Visual Studio Code",
		BundlePath:  "/Applications/Code.app",
	}
	values := aliasCandidates(app)
	if len(values) == 0 {
		t.Fatal("expected aliases")
	}
	found := false
	for _, value := range values {
		if value == "Code" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected bundle basename alias, got %v", values)
	}
}

func TestExactNameMatchesUsesNormalizedComparison(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Visual Studio Code"), 0o755); err != nil {
		t.Fatal(err)
	}
	matches, warnings := exactNameMatches(root, []string{"visual-studio-code"})
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one match, got %d", len(matches))
	}
}
