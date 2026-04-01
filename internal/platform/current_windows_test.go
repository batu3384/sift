//go:build windows

package platform

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/batu3384/sift/internal/domain"
)

func TestWindowsResolveTargetsExpandsLocalAppData(t *testing.T) {
	t.Parallel()
	adapter := windowsAdapter{
		localApp: `C:\Users\batuhan\AppData\Local`,
	}
	targets := adapter.ResolveTargets([]string{`%LOCALAPPDATA%\Temp`, `C:\Scratch`})
	if len(targets) != 2 {
		t.Fatalf("unexpected resolved target count: %v", targets)
	}
	if targets[0] != filepath.Join(adapter.localApp, "Temp") {
		t.Fatalf("expected LOCALAPPDATA expansion, got %q", targets[0])
	}
}

func TestWindowsIsAdminPathUsesCaseInsensitivePrefix(t *testing.T) {
	t.Parallel()
	t.Setenv("ProgramFiles", `C:\Program Files`)
	t.Setenv("ProgramFiles(x86)", `C:\Program Files (x86)`)
	t.Setenv("ProgramData", `C:\ProgramData`)
	adapter := windowsAdapter{}
	if !adapter.IsAdminPath(`c:\program files\Example\app.exe`) {
		t.Fatal("expected program files path to require admin")
	}
	if adapter.IsAdminPath(`C:\Users\batuhan\AppData\Local\Example`) {
		t.Fatal("expected user-space path not to require admin")
	}
}

func TestWindowsDiscoverRemnantsFindsExactNameMatches(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	localApp := filepath.Join(root, "Local")
	roaming := filepath.Join(root, "Roaming")
	programData := filepath.Join(root, "ProgramData")
	installLocation := filepath.Join(root, "Program Files", "Example App")
	for _, path := range []string{
		filepath.Join(localApp, "Example App"),
		filepath.Join(roaming, "Example App"),
		filepath.Join(programData, "Example App"),
		installLocation,
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	adapter := windowsAdapter{
		localApp:  localApp,
		roaming:   roaming,
		programDT: programData,
	}
	paths, warnings, err := adapter.DiscoverRemnants(context.Background(), domain.AppEntry{
		Name:        "exampleapp",
		DisplayName: "Example App",
		BundlePath:  installLocation,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	expected := map[string]bool{
		filepath.Join(localApp, "Example App"):    false,
		filepath.Join(roaming, "Example App"):     false,
		filepath.Join(programData, "Example App"): false,
	}
	for _, path := range paths {
		if _, ok := expected[path]; ok {
			expected[path] = true
		}
	}
	for path, seen := range expected {
		if !seen {
			t.Fatalf("expected remnant path %s in %v", path, paths)
		}
	}
}

func TestWindowsCuratedRootsIncludeUserAndPackageAreas(t *testing.T) {
	t.Parallel()
	adapter := windowsAdapter{
		home:      `C:\Users\batuhan`,
		localApp:  `C:\Users\batuhan\AppData\Local`,
		roaming:   `C:\Users\batuhan\AppData\Roaming`,
		programDT: `C:\ProgramData`,
	}
	roots := adapter.CuratedRoots()
	if len(roots.Temp) == 0 || len(roots.PackageManager) == 0 || len(roots.Browser) == 0 {
		t.Fatalf("expected populated curated roots, got %+v", roots)
	}
	expectedDeveloper := map[string]bool{
		filepath.Join(adapter.localApp, "ms-playwright"): false,
		filepath.Join(adapter.localApp, "puppeteer"):     false,
	}
	for _, root := range roots.Developer {
		if _, ok := expectedDeveloper[root]; ok {
			expectedDeveloper[root] = true
		}
	}
	for root, seen := range expectedDeveloper {
		if !seen {
			t.Fatalf("expected developer root %s in %+v", root, roots.Developer)
		}
	}
	dockerLog := filepath.Join(adapter.localApp, "Docker", "log")
	foundLog := false
	for _, root := range roots.Logs {
		if root == dockerLog {
			foundLog = true
			break
		}
	}
	if !foundLog {
		t.Fatalf("expected docker log root %s in %+v", dockerLog, roots.Logs)
	}

	expectedTemp := map[string]bool{
		filepath.Join(adapter.roaming, "Figma", "GPUCache"):                          false,
		filepath.Join(adapter.roaming, "Figma", "Service Worker", "CacheStorage"):    false,
		filepath.Join(adapter.roaming, "Postman", "GPUCache"):                        false,
		filepath.Join(adapter.roaming, "Postman", "Service Worker", "CacheStorage"):  false,
		filepath.Join(adapter.roaming, "Zed", "GPUCache"):                            false,
		filepath.Join(adapter.roaming, "Zed", "Service Worker", "CacheStorage"):      false,
		filepath.Join(adapter.roaming, "Claude", "GPUCache"):                         false,
		filepath.Join(adapter.roaming, "Claude", "Service Worker", "CacheStorage"):   false,
		filepath.Join(adapter.roaming, "ChatGPT", "GPUCache"):                        false,
		filepath.Join(adapter.roaming, "ChatGPT", "Service Worker", "CacheStorage"):  false,
		filepath.Join(adapter.roaming, "Cursor", "GPUCache"):                         false,
		filepath.Join(adapter.roaming, "Cursor", "Service Worker", "CacheStorage"):   false,
		filepath.Join(adapter.roaming, "VSCodium", "GPUCache"):                       false,
		filepath.Join(adapter.roaming, "VSCodium", "Service Worker", "CacheStorage"): false,
		filepath.Join(adapter.roaming, "Slack", "GPUCache"):                          false,
		filepath.Join(adapter.roaming, "Slack", "Service Worker", "CacheStorage"):    false,
		filepath.Join(adapter.roaming, "discord", "GPUCache"):                        false,
		filepath.Join(adapter.roaming, "discord", "Service Worker", "CacheStorage"):  false,
	}
	for _, root := range roots.Temp {
		if _, ok := expectedTemp[root]; ok {
			expectedTemp[root] = true
		}
	}
	for root, seen := range expectedTemp {
		if !seen {
			t.Fatalf("expected temp root %s in %+v", root, roots.Temp)
		}
	}

	expectedLogs := map[string]bool{
		filepath.Join(adapter.roaming, "Figma", "logs"):    false,
		filepath.Join(adapter.roaming, "Postman", "logs"):  false,
		filepath.Join(adapter.roaming, "Zed", "logs"):      false,
		filepath.Join(adapter.roaming, "Claude", "logs"):   false,
		filepath.Join(adapter.roaming, "ChatGPT", "logs"):  false,
		filepath.Join(adapter.roaming, "Cursor", "logs"):   false,
		filepath.Join(adapter.roaming, "VSCodium", "logs"): false,
	}
	for _, root := range roots.Logs {
		if _, ok := expectedLogs[root]; ok {
			expectedLogs[root] = true
		}
	}
	for root, seen := range expectedLogs {
		if !seen {
			t.Fatalf("expected log root %s in %+v", root, roots.Logs)
		}
	}
}

func TestWindowsListAppsIncludesUserSpaceProgramsInstall(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	localApp := filepath.Join(root, "LocalAppData")
	roaming := filepath.Join(root, "Roaming")
	installRoot := filepath.Join(localApp, "Programs", "Example App")
	if err := os.MkdirAll(installRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installRoot, "uninstall.exe"), []byte("binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	adapter := windowsAdapter{
		localApp: localApp,
		roaming:  roaming,
	}
	apps, err := adapter.ListApps(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, app := range apps {
		if app.DisplayName == "Example App" {
			found = true
			if app.UninstallCommand == "" {
				t.Fatalf("expected uninstall command for %+v", app)
			}
			if app.Origin != "user program" {
				t.Fatalf("expected user program origin for %+v", app)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected Example App in app list, got %v", apps)
	}
}
