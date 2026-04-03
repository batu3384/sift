//go:build windows

package platform

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/batu3384/sift/internal/domain"
	"golang.org/x/sys/windows/registry"
)

func (w windowsAdapter) ListApps(_ context.Context, _ bool) ([]domain.AppEntry, error) {
	keys := []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER}
	paths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}
	var apps []domain.AppEntry
	seen := map[string]struct{}{}
	for _, root := range keys {
		for _, path := range paths {
			k, err := registry.OpenKey(root, path, registry.READ)
			if err != nil {
				continue
			}
			names, err := k.ReadSubKeyNames(-1)
			if err != nil {
				_ = k.Close()
				continue
			}
			for _, name := range names {
				sub, err := registry.OpenKey(k, name, registry.READ)
				if err != nil {
					continue
				}
				display, _, err := sub.GetStringValue("DisplayName")
				if err != nil {
					_ = sub.Close()
					continue
				}
				systemComponent, _, _ := sub.GetIntegerValue("SystemComponent")
				releaseType, _, _ := sub.GetStringValue("ReleaseType")
				installLocation, _, _ := sub.GetStringValue("InstallLocation")
				uninstall, _, _ := sub.GetStringValue("UninstallString")
				quietUninstall, _, _ := sub.GetStringValue("QuietUninstallString")
				app, ok := appEntryFromWindowsEntry(windowsUninstallEntry{
					DisplayName:          display,
					InstallLocation:      installLocation,
					UninstallString:      uninstall,
					QuietUninstallString: quietUninstall,
					ReleaseType:          releaseType,
					SystemComponent:      systemComponent == 1,
					MachineScope:         root == registry.LOCAL_MACHINE,
				}, windowsAppEnvironment{
					LocalApp: w.localApp,
					Roaming:  w.roaming,
				})
				if !ok {
					_ = sub.Close()
					continue
				}
				keyName := strings.ToLower(app.DisplayName)
				if _, ok := seen[keyName]; ok {
					_ = sub.Close()
					continue
				}
				seen[keyName] = struct{}{}
				apps = append(apps, app)
				_ = sub.Close()
			}
			_ = k.Close()
		}
	}
	userApps, _ := discoverWindowsProgramApps(w.localApp, w.roaming)
	for _, app := range userApps {
		keyName := strings.ToLower(app.DisplayName)
		if _, ok := seen[keyName]; ok {
			continue
		}
		seen[keyName] = struct{}{}
		apps = append(apps, app)
	}
	return apps, nil
}

func (w windowsAdapter) DiscoverRemnants(_ context.Context, app domain.AppEntry) ([]string, []string, error) {
	aliases := aliasCandidates(app)
	candidates := append([]string{}, app.SupportPaths...)
	roots := []string{w.localApp, w.roaming, w.programDT}
	var warnings []string
	for _, root := range roots {
		matches, rootWarnings := exactNameMatches(root, aliases)
		candidates = append(candidates, matches...)
		warnings = append(warnings, rootWarnings...)
	}
	if app.BundlePath != "" {
		installBase := strings.TrimSuffix(filepath.Base(app.BundlePath), filepath.Ext(app.BundlePath))
		if installBase != "" {
			for _, root := range roots {
				matches, rootWarnings := exactNameMatches(root, []string{installBase})
				candidates = append(candidates, matches...)
				warnings = append(warnings, rootWarnings...)
			}
		}
	}
	return existingUniquePaths(candidates), warnings, nil
}
