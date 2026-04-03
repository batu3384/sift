//go:build darwin

package platform

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/batu3384/sift/internal/domain"
)

func (d darwinAdapter) ListApps(_ context.Context, allowAdmin bool) ([]domain.AppEntry, error) {
	homebrewCasks := d.homebrewCaskNames()
	setappRoot := filepath.Join(d.home, "Library", "Application Support", "Setapp", "Applications")
	roots := []struct {
		path          string
		origin        string
		requiresAdmin bool
	}{
		{path: "/Applications", origin: "system application", requiresAdmin: true},
		{path: filepath.Join(d.home, "Applications"), origin: "user application", requiresAdmin: false},
		{path: setappRoot, origin: "setapp", requiresAdmin: false},
	}

	if !allowAdmin {
		var filtered []struct {
			path          string
			origin        string
			requiresAdmin bool
		}
		for _, root := range roots {
			if !root.requiresAdmin {
				filtered = append(filtered, root)
			}
		}
		roots = filtered
	}

	var apps []domain.AppEntry
	seen := map[string]struct{}{}
	scanDir := func(dirPath string, origin string, requiresAdmin bool) {
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".app") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".app")
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			bundleOrigin := origin
			if dirPath != setappRoot {
				if _, ok := homebrewCasks[strings.ToLower(name)]; ok {
					bundleOrigin = "homebrew cask"
				}
			}
			bundlePath := filepath.Join(dirPath, entry.Name())
			apps = append(apps, domain.AppEntry{
				Name:        strings.ToLower(name),
				DisplayName: name,
				BundlePath:  bundlePath,
				SupportPaths: []string{
					filepath.Join(d.home, "Library", "Application Support", name),
					filepath.Join(d.home, "Library", "Caches", name),
					filepath.Join(d.home, "Library", "Logs", name),
					filepath.Join(d.home, "Library", "Preferences", "com."+strings.ToLower(strings.ReplaceAll(name, " ", ""))+".plist"),
				},
				Origin:           bundleOrigin,
				RequiresAdmin:    requiresAdmin,
				UninstallHint:    "Review bundle and support files before deletion.",
				UninstallCommand: d.findUninstallHelper(bundlePath, name),
				LastModified:     info.ModTime(),
			})
		}
	}

	for _, root := range roots {
		scanDir(root.path, root.origin, root.requiresAdmin)
		if entries, err := os.ReadDir(root.path); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				subDir := filepath.Join(root.path, entry.Name())
				if subEntries, err := os.ReadDir(subDir); err == nil {
					for _, subEntry := range subEntries {
						if subEntry.IsDir() && strings.HasSuffix(subEntry.Name(), ".app") {
							scanDir(subDir, root.origin, root.requiresAdmin)
						}
					}
				}
			}
		}
	}
	return apps, nil
}

func (d darwinAdapter) homebrewCaskNames() map[string]struct{} {
	roots := []string{"/opt/homebrew/Caskroom", "/usr/local/Caskroom"}
	names := map[string]struct{}{}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			names[strings.ToLower(strings.TrimSpace(entry.Name()))] = struct{}{}
		}
	}
	return names
}

func (d darwinAdapter) DiscoverRemnants(ctx context.Context, app domain.AppEntry) ([]string, []string, error) {
	aliases := aliasCandidates(app)
	var candidates []string
	candidates = append(candidates, app.SupportPaths...)

	bundleID := d.bundleIdentifier(ctx, app.BundlePath)
	if bundleID != "" {
		candidates = append(candidates,
			filepath.Join(d.home, "Library", "Preferences", bundleID+".plist"),
			filepath.Join(d.home, "Library", "Containers", bundleID),
			filepath.Join(d.home, "Library", "Saved Application State", bundleID+".savedState"),
			filepath.Join(d.home, "Library", "HTTPStorages", bundleID),
			filepath.Join(d.home, "Library", "WebKit", bundleID),
			filepath.Join(d.home, "Library", "Application Scripts", bundleID),
			filepath.Join(d.home, "Library", "Group Containers", "group."+bundleID),
			filepath.Join(d.home, "Library", "Caches", bundleID),
			filepath.Join(d.home, "Library", "Logs", bundleID),
			filepath.Join(d.home, "Library", "Application Support", bundleID),
		)
	}

	searchRoots := []string{
		filepath.Join(d.home, "Library", "Application Support"),
		filepath.Join(d.home, "Library", "Caches"),
		filepath.Join(d.home, "Library", "Logs"),
		filepath.Join(d.home, "Library", "Containers"),
		filepath.Join(d.home, "Library", "Saved Application State"),
	}
	var warnings []string
	for _, root := range searchRoots {
		matches, rootWarnings := exactNameMatches(root, aliases)
		candidates = append(candidates, matches...)
		warnings = append(warnings, rootWarnings...)
	}
	if bundleID != "" {
		for _, root := range []string{
			filepath.Join(d.home, "Library", "LaunchAgents"),
			"/Library/LaunchAgents",
			"/Library/LaunchDaemons",
			filepath.Join(d.home, "Library", "Preferences", "ByHost"),
		} {
			matches, rootWarnings := prefixMatches(root, bundleID)
			candidates = append(candidates, matches...)
			warnings = append(warnings, rootWarnings...)
		}
	}
	return existingUniquePaths(candidates), warnings, nil
}

func (d darwinAdapter) bundleIdentifier(ctx context.Context, bundlePath string) string {
	if bundlePath == "" {
		return ""
	}
	infoPlist := filepath.Join(bundlePath, "Contents", "Info.plist")
	if _, err := os.Stat(infoPlist); err != nil {
		return ""
	}
	out, err := exec.CommandContext(ctx, "/usr/bin/defaults", "read", infoPlist, "CFBundleIdentifier").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	raw, readErr := os.ReadFile(infoPlist)
	if readErr != nil {
		return ""
	}
	content := string(raw)
	key := "<key>CFBundleIdentifier</key>"
	keyIndex := strings.Index(content, key)
	if keyIndex < 0 {
		return ""
	}
	content = content[keyIndex+len(key):]
	start := strings.Index(content, "<string>")
	end := strings.Index(content, "</string>")
	if start < 0 || end < 0 || end <= start+len("<string>") {
		return ""
	}
	return strings.TrimSpace(content[start+len("<string>") : end])
}

func (d darwinAdapter) findUninstallHelper(bundlePath, appName string) string {
	candidates := []string{
		filepath.Join(filepath.Dir(bundlePath), "Uninstall "+appName+".app"),
		filepath.Join(filepath.Dir(bundlePath), appName+" Uninstaller.app"),
		filepath.Join(filepath.Dir(bundlePath), "Uninstaller.app"),
		filepath.Join(bundlePath, "Contents", "Resources", "Uninstaller.app"),
		filepath.Join(bundlePath, "Contents", "Resources", "uninstall.sh"),
		filepath.Join(bundlePath, "Contents", "MacOS", "uninstall"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return domain.NormalizePath(candidate)
		}
	}
	files, err := os.ReadDir(filepath.Dir(bundlePath))
	if err != nil {
		return ""
	}
	nameKey := normalizedNameKey(appName)
	for _, file := range files {
		if !file.IsDir() || !strings.HasSuffix(file.Name(), ".app") {
			continue
		}
		base := strings.TrimSuffix(file.Name(), ".app")
		key := normalizedNameKey(base)
		if key == "" || !strings.Contains(key, "uninstall") {
			continue
		}
		if strings.Contains(key, nameKey) || strings.Contains(nameKey, key) || slices.Contains([]string{"uninstaller", "uninstall"}, key) {
			return domain.NormalizePath(filepath.Join(filepath.Dir(bundlePath), file.Name()))
		}
	}
	return ""
}
