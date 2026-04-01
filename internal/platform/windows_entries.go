//go:build windows

package platform

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/batu3384/sift/internal/domain"
)

type windowsAppEnvironment struct {
	LocalApp string
	Roaming  string
}

type windowsUninstallEntry struct {
	DisplayName          string
	InstallLocation      string
	UninstallString      string
	QuietUninstallString string
	ReleaseType          string
	SystemComponent      bool
	MachineScope         bool
}

func appEntryFromWindowsEntry(entry windowsUninstallEntry, env windowsAppEnvironment) (domain.AppEntry, bool) {
	display := strings.TrimSpace(entry.DisplayName)
	if display == "" || entry.SystemComponent {
		return domain.AppEntry{}, false
	}
	if isWindowsUpdateReleaseType(entry.ReleaseType) {
		return domain.AppEntry{}, false
	}
	return domain.AppEntry{
		Name:          strings.ToLower(display),
		DisplayName:   display,
		BundlePath:    strings.TrimSpace(entry.InstallLocation),
		Origin:        "registry uninstall",
		RequiresAdmin: entry.MachineScope,
		SupportPaths: []string{
			filepath.Join(env.LocalApp, display),
			filepath.Join(env.Roaming, display),
		},
		UninstallHint:         "Review native uninstaller before remnant cleanup.",
		UninstallCommand:      strings.TrimSpace(entry.UninstallString),
		QuietUninstallCommand: strings.TrimSpace(entry.QuietUninstallString),
		LastModified:          modTime(strings.TrimSpace(entry.InstallLocation)),
	}, true
}

func isWindowsUpdateReleaseType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "hotfix", "security update", "update rollup":
		return true
	default:
		return false
	}
}

func discoverWindowsProgramApps(localApp, roaming string) ([]domain.AppEntry, []string) {
	root := filepath.Join(localApp, "Programs")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []string{domain.NormalizePath(root) + ": " + err.Error()}
	}
	apps := make([]domain.AppEntry, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		display := strings.TrimSpace(entry.Name())
		if display == "" {
			continue
		}
		bundlePath := filepath.Join(root, entry.Name())
		uninstall := firstExistingPath(
			filepath.Join(bundlePath, "uninstall.exe"),
			filepath.Join(bundlePath, "Uninstall.exe"),
			filepath.Join(bundlePath, "uninstaller.exe"),
			filepath.Join(bundlePath, "Uninstaller.exe"),
			filepath.Join(bundlePath, "unins000.exe"),
		)
		if uninstall == "" {
			continue
		}
		apps = append(apps, domain.AppEntry{
			Name:        strings.ToLower(display),
			DisplayName: display,
			BundlePath:  bundlePath,
			SupportPaths: []string{
				filepath.Join(localApp, display),
				filepath.Join(roaming, display),
			},
			Origin:           "user program",
			UninstallHint:    "Review native uninstaller before remnant cleanup.",
			UninstallCommand: uninstall,
			LastModified:     modTime(bundlePath),
		})
	}
	return apps, nil
}

func modTime(path string) (modified time.Time) {
	if strings.TrimSpace(path) == "" {
		return time.Time{}
	}
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func firstExistingPath(paths ...string) string {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return domain.NormalizePath(path)
		}
	}
	return ""
}
