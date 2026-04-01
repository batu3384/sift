package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/batu3384/sift/internal/domain"
)

func TestNormalizedAppKeyStripsNoise(t *testing.T) {
	t.Parallel()

	if got := normalizedAppKey(" Raycast.app "); got != "raycastapp" {
		t.Fatalf("unexpected normalized key: %q", got)
	}
	if got := normalizedAppKey("ChatGPT (Beta)!"); got != "chatgptbeta" {
		t.Fatalf("unexpected normalized key: %q", got)
	}
}

func TestFindInstalledAppMatchesBundleStem(t *testing.T) {
	t.Parallel()

	app, ok := findInstalledApp([]domain.AppEntry{
		{
			Name:       "Claude Desktop",
			BundlePath: "/Applications/Claude.app",
		},
	}, "Claude")
	if !ok || app == nil {
		t.Fatal("expected Claude bundle stem to match")
	}
}

func TestBuildUninstallPlanForSIFTSkipsOwnedStatePaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ownedDir := filepath.Join(root, "Library", "Application Support", "SIFT")
	otherDir := filepath.Join(root, "Library", "Application Support", "Example")
	if err := os.MkdirAll(ownedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ownedFile := filepath.Join(ownedDir, "state.json")
	otherFile := filepath.Join(otherDir, "leftover.bin")
	if err := os.WriteFile(ownedFile, []byte("owned"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(otherFile, []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := &Service{
		Adapter: uninstallStubAdapter{
			stubAdapter: stubAdapter{
				name:     "darwin",
				remnants: []string{ownedFile, otherFile},
			},
			apps: nil,
		},
	}

	plan, err := service.BuildUninstallPlan(context.Background(), "sift", true, false)
	if err != nil {
		t.Fatal(err)
	}
	var foundSIFTOwned, foundOther, foundGuidance bool
	for _, item := range plan.Items {
		if item.RuleID == "uninstall.sift" {
			foundGuidance = true
		}
		if item.Path == ownedFile {
			foundSIFTOwned = true
		}
		if item.Path == otherFile {
			foundOther = true
		}
	}
	if !foundGuidance {
		t.Fatalf("expected sift self-removal guidance, got %+v", plan.Items)
	}
	if foundSIFTOwned {
		t.Fatalf("expected sift-owned state path to be skipped, got %+v", plan.Items)
	}
	if !foundOther {
		t.Fatalf("expected non-owned leftover to remain actionable, got %+v", plan.Items)
	}
}
