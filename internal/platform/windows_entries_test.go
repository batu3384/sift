//go:build windows

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppEntryFromWindowsEntryBuildsAppEntry(t *testing.T) {
	t.Parallel()
	app, ok := appEntryFromWindowsEntry(windowsUninstallEntry{
		DisplayName:          "Example App",
		InstallLocation:      `C:\Program Files\Example`,
		UninstallString:      `C:\Program Files\Example\uninstall.exe /S`,
		QuietUninstallString: `MsiExec.exe /X{ABC-123}`,
		MachineScope:         true,
	}, windowsAppEnvironment{
		LocalApp: `C:\Users\batuhan\AppData\Local`,
		Roaming:  `C:\Users\batuhan\AppData\Roaming`,
	})
	if !ok {
		t.Fatal("expected windows app entry to be accepted")
	}
	if app.DisplayName != "Example App" || !app.RequiresAdmin {
		t.Fatalf("unexpected app entry: %+v", app)
	}
	if app.Origin != "registry uninstall" {
		t.Fatalf("expected registry origin, got %+v", app)
	}
	if app.QuietUninstallCommand != `MsiExec.exe /X{ABC-123}` {
		t.Fatalf("unexpected quiet uninstall command: %+v", app)
	}
}

func TestAppEntryFromWindowsEntryRejectsSystemComponentsAndUpdates(t *testing.T) {
	t.Parallel()
	cases := []windowsUninstallEntry{
		{DisplayName: "Security Update", ReleaseType: "Security Update"},
		{DisplayName: "Hotfix KB", ReleaseType: "Hotfix"},
		{DisplayName: "Internal Component", SystemComponent: true},
		{DisplayName: ""},
	}
	for _, tc := range cases {
		if _, ok := appEntryFromWindowsEntry(tc, windowsAppEnvironment{}); ok {
			t.Fatalf("expected entry to be rejected: %+v", tc)
		}
	}
}

func TestDiscoverWindowsProgramAppsFindsUserSpaceInstall(t *testing.T) {
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
	apps, warnings := discoverWindowsProgramApps(localApp, roaming)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(apps) != 1 {
		t.Fatalf("expected one discovered app, got %+v", apps)
	}
	if apps[0].DisplayName != "Example App" {
		t.Fatalf("unexpected app discovery result: %+v", apps[0])
	}
	if apps[0].Origin != "user program" {
		t.Fatalf("expected user program origin, got %+v", apps[0])
	}
}
