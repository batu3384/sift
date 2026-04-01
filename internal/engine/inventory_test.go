package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/batu3384/sift/internal/domain"
)

func TestCachedAppsWithoutStoreReturnsEmptySnapshot(t *testing.T) {
	t.Parallel()

	service := &Service{}
	apps, cachedAt, err := service.CachedApps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if apps != nil {
		t.Fatalf("expected nil apps without store, got %+v", apps)
	}
	if !cachedAt.IsZero() {
		t.Fatalf("expected zero cached timestamp, got %v", cachedAt)
	}
}

func TestListAppsEnrichesFamilyMatchesAndApproxBytes(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	raycastRoot := filepath.Join(home, "Library", "Application Support", "Raycast")
	if err := os.MkdirAll(raycastRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	cacheFile := filepath.Join(raycastRoot, "state.json")
	if err := os.WriteFile(cacheFile, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := &Service{
		Adapter: inventoryStubAdapter{
			stubAdapter: stubAdapter{name: "darwin"},
			home:        home,
			apps: []domain.AppEntry{
				{
					Name:         "Raycast",
					DisplayName:  "Raycast",
					SupportPaths: []string{raycastRoot},
				},
			},
		},
	}

	apps, err := service.ListApps(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 {
		t.Fatalf("expected one app, got %+v", apps)
	}
	if apps[0].ApproxBytes == 0 {
		t.Fatalf("expected approx bytes to be measured, got %+v", apps[0])
	}
	if !apps[0].Sensitive {
		t.Fatalf("expected app to be marked sensitive, got %+v", apps[0])
	}
	if len(apps[0].FamilyMatches) == 0 || apps[0].FamilyMatches[0] != "launcher_state" {
		t.Fatalf("expected launcher_state family match, got %+v", apps[0].FamilyMatches)
	}
}
