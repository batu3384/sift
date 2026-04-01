package rules

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/platform"
)

func userHomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

var staleLoginItemsRoot = func() string {
	return filepath.Join(userHomeDir(), "Library", "LaunchAgents")
}

var systemServiceRoots = func() []string {
	return []string{"/Library/LaunchAgents", "/Library/LaunchDaemons"}
}

var privilegedHelperToolsRoot = func() string {
	return "/Library/PrivilegedHelperTools"
}

var currentAdapterIsFileInUse = func(ctx context.Context, path string) bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	return platform.Current().IsFileInUse(ctx, path)
}

func staleLaunchAgentTarget(raw []byte) (string, string) {
	content := string(raw)
	for _, key := range []string{"Program", "ProgramArguments"} {
		values := plistStringValues(content, key)
		for _, candidate := range values {
			candidate = strings.TrimSpace(candidate)
			if candidate == "" || launchAgentIsSystemPath(candidate) {
				continue
			}
			if looksLikeLaunchAgentAppPath(candidate) {
				return candidate, "missing app/helper target"
			}
		}
	}
	return "", ""
}

func plistStringValues(content, key string) []string {
	keyToken := "<key>" + key + "</key>"
	index := strings.Index(content, keyToken)
	if index < 0 {
		return nil
	}
	rest := content[index+len(keyToken):]
	nextKey := strings.Index(rest, "<key>")
	segment := rest
	if nextKey >= 0 {
		segment = rest[:nextKey]
	}
	var values []string
	for {
		start := strings.Index(segment, "<string>")
		if start < 0 {
			break
		}
		segment = segment[start+len("<string>"):]
		end := strings.Index(segment, "</string>")
		if end < 0 {
			break
		}
		value := strings.TrimSpace(segment[:end])
		if value != "" {
			values = append(values, value)
		}
		segment = segment[end+len("</string>"):]
	}
	return values
}

func launchAgentIsSystemPath(path string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(path), `\`, `/`))
	for _, prefix := range []string{"/system/", "/usr/bin/", "/usr/sbin/", "/bin/", "/sbin/", "/opt/homebrew/bin/", "/opt/homebrew/sbin/"} {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	return false
}

func looksLikeLaunchAgentAppPath(path string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(path), `\`, `/`))
	return strings.Contains(normalized, ".app/") ||
		strings.HasPrefix(normalized, "/applications/") ||
		strings.Contains(normalized, "/applications/") ||
		strings.Contains(normalized, "/library/privilegedhelpertools/") ||
		strings.Contains(normalized, "/library/application support/")
}

func launchAgentTargetExists(path string) bool {
	expanded := strings.Replace(path, "~", userHomeDir(), 1)
	if _, err := os.Stat(expanded); err == nil {
		return true
	}
	if strings.Contains(expanded, ".app/") {
		bundle := expanded[:strings.Index(expanded, ".app/")+len(".app")]
		if _, err := os.Stat(bundle); err == nil {
			return true
		}
	}
	return false
}

func scanSystemUpdateArtifacts(ctx context.Context, adapter platform.Adapter) ([]domain.Finding, []string, error) {
	findings, warnings, err := scanRootEntries(ctx, adapter, adapter.CuratedRoots().SystemUpdate, domain.CategorySystemClutter, domain.RiskReview, domain.ActionTrash, "System update artifact")
	if err != nil {
		return findings, warnings, err
	}
	installerApps, installerWarnings, installerErr := scanMacOSInstallerApps(ctx, adapter)
	if installerErr != nil {
		return findings, append(warnings, installerWarnings...), installerErr
	}
	findings = append(findings, installerApps...)
	sortFindings(findings)
	return findings, dedupeStrings(append(warnings, installerWarnings...)), nil
}

func scanMacOSInstallerApps(ctx context.Context, adapter platform.Adapter) ([]domain.Finding, []string, error) {
	if runtime.GOOS != "darwin" {
		return nil, nil, nil
	}
	matches, err := filepath.Glob("/Applications/Install macOS*.app")
	if err != nil {
		return nil, []string{err.Error()}, nil
	}
	var findings []domain.Finding
	var warnings []string
	for _, match := range matches {
		select {
		case <-ctx.Done():
			return findings, warnings, ctx.Err()
		default:
		}
		info, err := os.Stat(match)
		if err != nil {
			warnings = append(warnings, match+": "+err.Error())
			continue
		}
		if !info.IsDir() {
			continue
		}
		if time.Since(info.ModTime()) < 14*24*time.Hour {
			continue
		}
		size, newest, err := MeasurePath(ctx, match)
		if err != nil || size == 0 {
			continue
		}
		findings = append(findings, newFinding(filepath.Base(match), domain.NormalizePath(match), domain.CategorySystemClutter, domain.RiskReview, domain.ActionTrash, size, newest, info.Mode(), "macOS installer payload"))
	}
	return findings, warnings, nil
}

func scanStaleLoginItems(ctx context.Context, adapter platform.Adapter) ([]domain.Finding, []string, error) {
	if runtime.GOOS != "darwin" {
		return nil, nil, nil
	}
	launchAgentsDir := staleLoginItemsRoot()
	entries, err := os.ReadDir(launchAgentsDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, []string{launchAgentsDir + ": " + err.Error()}, nil
	}
	var findings []domain.Finding
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return findings, nil, ctx.Err()
		default:
		}
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".plist") || strings.HasPrefix(entry.Name(), "com.apple.") {
			continue
		}
		path := domain.NormalizePath(filepath.Join(launchAgentsDir, entry.Name()))
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		target, reason := staleLaunchAgentTarget(raw)
		if target == "" || reason == "" {
			continue
		}
		if launchAgentTargetExists(target) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		findings = append(findings, domain.Finding{
			ID:          uuid.NewString(),
			RuleID:      "stale_login_items",
			Name:        entry.Name(),
			Category:    domain.CategoryMaintenance,
			Path:        path,
			DisplayPath: path,
			Risk:        domain.RiskReview,
			Action:      domain.ActionAdvisory,
			Recovery: domain.RecoveryHint{
				Message:  "Open ~/Library/LaunchAgents and remove only items you recognize.",
				Location: "manual review",
			},
			Status:       domain.StatusAdvisory,
			LastModified: info.ModTime(),
			Fingerprint: domain.Fingerprint{
				Mode:    uint32(info.Mode()),
				Size:    info.Size(),
				ModTime: info.ModTime(),
			},
			Source: fmt.Sprintf("Potential stale login item (%s: %s)", reason, target),
		})
	}
	sortFindings(findings)
	return findings, nil, nil
}

func scanOrphanedServices(ctx context.Context, adapter platform.Adapter) ([]domain.Finding, []string, error) {
	if runtime.GOOS != "darwin" {
		return nil, nil, nil
	}
	var findings []domain.Finding
	var warnings []string
	helperRoot := privilegedHelperToolsRoot()
	for _, root := range systemServiceRoots() {
		entries, err := os.ReadDir(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			warnings = append(warnings, root+": "+err.Error())
			continue
		}
		for _, entry := range entries {
			select {
			case <-ctx.Done():
				return findings, warnings, ctx.Err()
			default:
			}
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".plist") || strings.HasPrefix(entry.Name(), "com.apple.") {
				continue
			}
			path := domain.NormalizePath(filepath.Join(root, entry.Name()))
			raw, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			target, reason := staleLaunchAgentTarget(raw)
			if target == "" || reason == "" || launchAgentTargetExists(target) {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			findings = append(findings, domain.Finding{
				ID:            uuid.NewString(),
				RuleID:        "orphaned_services",
				Name:          entry.Name(),
				Category:      domain.CategoryAppLeftovers,
				Path:          path,
				DisplayPath:   path,
				Risk:          domain.RiskReview,
				Action:        domain.ActionTrash,
				RequiresAdmin: adapter.IsAdminPath(path),
				Recovery: domain.RecoveryHint{
					Message:  "Restore from Trash if the service still belongs to an installed app.",
					Location: "system trash",
				},
				Status:       domain.StatusPlanned,
				LastModified: info.ModTime(),
				Fingerprint: domain.Fingerprint{
					Mode:    uint32(info.Mode()),
					Size:    info.Size(),
					ModTime: info.ModTime(),
				},
				Source: fmt.Sprintf("Orphaned launch service (%s: %s)", reason, target),
			})
			helper := filepath.Join(helperRoot, strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
			helperInfo, err := os.Stat(helper)
			if err != nil || helperInfo.IsDir() {
				continue
			}
			findings = append(findings, newFinding(filepath.Base(helper), domain.NormalizePath(helper), domain.CategoryAppLeftovers, domain.RiskReview, domain.ActionTrash, helperInfo.Size(), helperInfo.ModTime(), helperInfo.Mode(), "Orphaned privileged helper"))
		}
	}
	sortFindings(findings)
	return dedupeFindings(findings), warnings, nil
}

func scanInstallerFiles(ctx context.Context, adapter platform.Adapter) ([]domain.Finding, []string, error) {
	var findings []domain.Finding
	var warnings []string
	exts := []string{".dmg", ".pkg", ".mpkg", ".zip", ".exe", ".msi", ".iso", ".xip", ".appinstaller", ".crdownload", ".part", ".download"}
	// Incomplete downloads should be found regardless of age
	incompleteExts := []string{".crdownload", ".part", ".download"}

	// Use filepath.WalkDir for recursive directory scanning
	scanDir := func(root string) error {
		return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if !slices.Contains(exts, ext) {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				warnings = append(warnings, d.Name()+": "+err.Error())
				return nil
			}
			// Skip 14-day check for incomplete downloads
			isIncomplete := slices.Contains(incompleteExts, ext)
			if !isIncomplete && time.Since(info.ModTime()) < 14*24*time.Hour {
				return nil
			}
			normalizedPath := domain.NormalizePath(path)
			rootSource := installerRootLabel(root)
			source := rootSource + " installer payload"
			if ext == ".zip" {
				ok, zipWarnings := zipLooksLikeInstaller(normalizedPath)
				warnings = append(warnings, zipWarnings...)
				if !ok {
					return nil
				}
				source = rootSource + " installer archive"
			}
			if isIncomplete {
				// Check if the file is currently being used/downloaded
				if currentAdapterIsFileInUse(ctx, normalizedPath) {
					warnings = append(warnings, fmt.Sprintf("skipped active download: %s", d.Name()))
					return nil
				}
				source = rootSource + " incomplete download"
			}
			findings = append(findings, domain.Finding{
				ID:          uuid.NewString(),
				RuleID:      "installer_leftovers",
				Name:        d.Name(),
				Category:    domain.CategoryInstallerLeft,
				Path:        normalizedPath,
				DisplayPath: normalizedPath,
				Risk:        domain.RiskReview,
				Bytes:       info.Size(),
				Action:      domain.ActionTrash,
				Recovery: domain.RecoveryHint{
					Message:  "Installer packages can be restored from Trash/Recycle Bin.",
					Location: "system trash",
				},
				Status:       domain.StatusPlanned,
				LastModified: info.ModTime(),
				Fingerprint: domain.Fingerprint{
					Mode:    uint32(info.Mode()),
					Size:    info.Size(),
					ModTime: info.ModTime(),
				},
				Source: source,
			})
			return nil
		})
	}

	// Scan all installer roots recursively
	for _, root := range adapter.CuratedRoots().Installer {
		if err := scanDir(root); err != nil {
			warnings = append(warnings, err.Error())
		}
		incomplete, incompleteWarnings, err := scanIncompleteDownloads(ctx, root)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		findings = append(findings, incomplete...)
		warnings = append(warnings, incompleteWarnings...)
	}
	return findings, warnings, nil
}

func scanIncompleteDownloads(ctx context.Context, root string) ([]domain.Finding, []string, error) {
	if !strings.Contains(strings.ToLower(root), "downloads") {
		return nil, nil, nil
	}
	patterns := []struct {
		suffix string
		source string
	}{
		{suffix: ".download", source: "Safari incomplete download"},
		{suffix: ".crdownload", source: "Chrome incomplete download"},
		{suffix: ".part", source: "Partial incomplete download"},
	}
	var findings []domain.Finding
	var warnings []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(root, "*"+pattern.suffix))
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		slices.Sort(matches)
		for _, match := range matches {
			select {
			case <-ctx.Done():
				return findings, warnings, ctx.Err()
			default:
			}
			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}
			if currentAdapterIsFileInUse(ctx, match) {
				warnings = append(warnings, fmt.Sprintf("skipped active download: %s", filepath.Base(match)))
				continue
			}
			findings = append(findings, domain.Finding{
				ID:          uuid.NewString(),
				RuleID:      "installer_leftovers",
				Name:        filepath.Base(match),
				Category:    domain.CategoryInstallerLeft,
				Path:        domain.NormalizePath(match),
				DisplayPath: domain.NormalizePath(match),
				Risk:        domain.RiskReview,
				Bytes:       info.Size(),
				Action:      domain.ActionTrash,
				Recovery: domain.RecoveryHint{
					Message:  "Incomplete downloads can be restored from Trash/Recycle Bin.",
					Location: "system trash",
				},
				Status:       domain.StatusPlanned,
				LastModified: info.ModTime(),
				Fingerprint: domain.Fingerprint{
					Mode:    uint32(info.Mode()),
					Size:    info.Size(),
					ModTime: info.ModTime(),
				},
				Source: pattern.source,
			})
		}
	}
	return findings, warnings, nil
}

var chromeAppPaths = func() []string {
	home := userHomeDir()
	return []string{
		"/Applications/Google Chrome.app",
		filepath.Join(home, "Applications", "Google Chrome.app"),
	}
}

var edgeAppPaths = func() []string {
	home := userHomeDir()
	return []string{
		"/Applications/Microsoft Edge.app",
		filepath.Join(home, "Applications", "Microsoft Edge.app"),
	}
}

func scanChromeOldVersions(ctx context.Context, _ platform.Adapter) ([]domain.Finding, []string, error) {
	if runtime.GOOS != "darwin" {
		return nil, nil, nil
	}
	if currentAdapterIsProcessRunning("Google Chrome") {
		return nil, nil, nil
	}
	appPaths := chromeAppPaths()
	var findings []domain.Finding
	var warnings []string
	for _, appPath := range appPaths {
		versionsDir := filepath.Join(appPath, "Contents", "Frameworks", "Google Chrome Framework.framework", "Versions")
		currentLink := filepath.Join(versionsDir, "Current")
		target, err := os.Readlink(currentLink)
		if err != nil {
			continue
		}
		currentVersion := filepath.Base(target)
		if currentVersion == "" || currentVersion == "." {
			continue
		}
		entries, err := os.ReadDir(versionsDir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			warnings = append(warnings, versionsDir+": "+err.Error())
			continue
		}
		for _, entry := range entries {
			select {
			case <-ctx.Done():
				return findings, warnings, ctx.Err()
			default:
			}
			name := entry.Name()
			if name == "Current" || name == currentVersion || !entry.IsDir() {
				continue
			}
			path := domain.NormalizePath(filepath.Join(versionsDir, name))
			size, newest, err := MeasurePath(ctx, path)
			if err != nil || size == 0 {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			findings = append(findings, newFinding(name, path, domain.CategoryBrowserData, domain.RiskReview, domain.ActionTrash, size, newest, info.Mode(), "Chrome old framework version"))
		}
	}
	sortFindings(findings)
	return findings, warnings, nil
}

func scanEdgeOldVersions(ctx context.Context, _ platform.Adapter) ([]domain.Finding, []string, error) {
	if runtime.GOOS != "darwin" {
		return nil, nil, nil
	}
	if currentAdapterIsProcessRunning("Microsoft Edge") {
		return nil, nil, nil
	}
	appPaths := edgeAppPaths()
	var findings []domain.Finding
	var warnings []string
	for _, appPath := range appPaths {
		versionsDir := filepath.Join(appPath, "Contents", "Frameworks", "Microsoft Edge Framework.framework", "Versions")
		currentLink := filepath.Join(versionsDir, "Current")
		target, err := os.Readlink(currentLink)
		if err != nil {
			continue
		}
		currentVersion := filepath.Base(target)
		if currentVersion == "" || currentVersion == "." {
			continue
		}
		entries, err := os.ReadDir(versionsDir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			warnings = append(warnings, versionsDir+": "+err.Error())
			continue
		}
		for _, entry := range entries {
			select {
			case <-ctx.Done():
				return findings, warnings, ctx.Err()
			default:
			}
			name := entry.Name()
			if name == "Current" || name == currentVersion || !entry.IsDir() {
				continue
			}
			path := domain.NormalizePath(filepath.Join(versionsDir, name))
			size, newest, err := MeasurePath(ctx, path)
			if err != nil || size == 0 {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			findings = append(findings, newFinding(name, path, domain.CategoryBrowserData, domain.RiskReview, domain.ActionTrash, size, newest, info.Mode(), "Edge old framework version"))
		}
	}
	sortFindings(findings)
	return findings, warnings, nil
}

func scanTimeMachineFailedBackups(ctx context.Context, _ platform.Adapter) ([]domain.Finding, []string, error) {
	if runtime.GOOS != "darwin" {
		return nil, nil, nil
	}
	volumesDir := "/Volumes"
	volumeEntries, err := os.ReadDir(volumesDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, []string{volumesDir + ": " + err.Error()}, nil
	}
	var findings []domain.Finding
	var warnings []string
	for _, vol := range volumeEntries {
		select {
		case <-ctx.Done():
			return findings, warnings, ctx.Err()
		default:
		}
		if !vol.IsDir() {
			continue
		}
		volPath := filepath.Join(volumesDir, vol.Name())
		backupDB := filepath.Join(volPath, "Backups.backupdb")
		info, err := os.Lstat(backupDB)
		if err != nil || !info.IsDir() {
			continue
		}
		// Walk backupdb looking for *.inProgress directories
		hostEntries, err := os.ReadDir(backupDB)
		if err != nil {
			warnings = append(warnings, backupDB+": "+err.Error())
			continue
		}
		for _, host := range hostEntries {
			if !host.IsDir() {
				continue
			}
			hostDir := filepath.Join(backupDB, host.Name())
			snapshots, err := os.ReadDir(hostDir)
			if err != nil {
				continue
			}
			for _, snap := range snapshots {
				select {
				case <-ctx.Done():
					return findings, warnings, ctx.Err()
				default:
				}
				if !snap.IsDir() || !strings.HasSuffix(snap.Name(), ".inProgress") {
					continue
				}
				snapInfo, err := snap.Info()
				if err != nil {
					continue
				}
				if time.Since(snapInfo.ModTime()) < 24*time.Hour {
					continue
				}
				path := domain.NormalizePath(filepath.Join(hostDir, snap.Name()))
				size, newest, err := MeasurePath(ctx, path)
				if err != nil {
					continue
				}
				findings = append(findings, newFinding(snap.Name(), path, domain.CategorySystemClutter, domain.RiskReview, domain.ActionTrash, size, newest, snapInfo.Mode(), "Time Machine incomplete backup"))
			}
		}
	}
	sortFindings(findings)
	return findings, warnings, nil
}

func scanDSStoreFiles(ctx context.Context, _ platform.Adapter) ([]domain.Finding, []string, error) {
	if runtime.GOOS != "darwin" {
		return nil, nil, nil
	}
	home := userHomeDir()
	if home == "" {
		return nil, nil, nil
	}
	const maxDepth = 5
	const maxFindings = 200
	skipDirs := map[string]struct{}{
		"node_modules": {},
		".git":         {},
		".Trash":       {},
	}
	var findings []domain.Finding
	var totalBytes int64
	var count int
	type stackEntry struct {
		path  string
		depth int
	}
	stack := []stackEntry{{path: home, depth: 0}}
	for len(stack) > 0 && len(findings) < maxFindings {
		select {
		case <-ctx.Done():
			return findings, nil, ctx.Err()
		default:
		}
		entry := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if entry.depth > maxDepth {
			continue
		}
		items, err := os.ReadDir(entry.path)
		if err != nil {
			continue
		}
		for _, item := range items {
			name := item.Name()
			itemPath := filepath.Join(entry.path, name)
			if item.IsDir() {
				if _, skip := skipDirs[name]; skip {
					continue
				}
				// skip Library subdirs that aren't user-level
				if entry.depth == 0 && name == "Library" {
					continue
				}
				if entry.depth < maxDepth {
					stack = append(stack, stackEntry{path: itemPath, depth: entry.depth + 1})
				}
				continue
			}
			if name != ".DS_Store" {
				continue
			}
			info, err := item.Info()
			if err != nil {
				continue
			}
			totalBytes += info.Size()
			count++
			if len(findings) < maxFindings {
				findings = append(findings, domain.Finding{
					ID:          uuid.NewString(),
					RuleID:      "finder_metadata",
					Name:        ".DS_Store",
					Category:    domain.CategorySystemClutter,
					Path:        domain.NormalizePath(itemPath),
					DisplayPath: domain.NormalizePath(itemPath),
					Risk:        domain.RiskSafe,
					Bytes:       info.Size(),
					Action:      domain.ActionPermanent,
					Recovery: domain.RecoveryHint{
						Message:  "Finder re-creates .DS_Store files automatically.",
						Location: "permanent",
					},
					Status:       domain.StatusPlanned,
					LastModified: info.ModTime(),
					Fingerprint: domain.Fingerprint{
						Mode:    uint32(info.Mode()),
						Size:    info.Size(),
						ModTime: info.ModTime(),
					},
					Source: "Finder metadata",
				})
			}
		}
	}
	_ = totalBytes
	_ = count
	return findings, nil, nil
}

func scanIOSDeviceBackups(ctx context.Context, _ platform.Adapter) ([]domain.Finding, []string, error) {
	if runtime.GOOS != "darwin" {
		return nil, nil, nil
	}
	home := userHomeDir()
	backupDir := filepath.Join(home, "Library", "Application Support", "MobileSync", "Backup")
	info, err := os.Stat(backupDir)
	if os.IsNotExist(err) || (err == nil && !info.IsDir()) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, []string{backupDir + ": " + err.Error()}, nil
	}
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, []string{backupDir + ": " + err.Error()}, nil
	}
	var findings []domain.Finding
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return findings, nil, ctx.Err()
		default:
		}
		if !entry.IsDir() {
			continue
		}
		backupPath := filepath.Join(backupDir, entry.Name())
		size, newest, err := MeasurePath(ctx, backupPath)
		if err != nil || size < 100*1024*1024 {
			continue
		}
		entryInfo, err := entry.Info()
		if err != nil {
			continue
		}
		findings = append(findings, domain.Finding{
			ID:          uuid.NewString(),
			RuleID:      "ios_device_backups",
			Name:        entry.Name(),
			Category:    domain.CategorySystemClutter,
			Path:        domain.NormalizePath(backupPath),
			DisplayPath: domain.NormalizePath(backupPath),
			Risk:        domain.RiskReview,
			Bytes:       size,
			Action:      domain.ActionAdvisory,
			Recovery: domain.RecoveryHint{
				Message:  "Delete old device backups in Finder > Preferences > iCloud or via iTunes/Finder.",
				Location: "manual review",
			},
			Status:       domain.StatusAdvisory,
			LastModified: newest,
			Fingerprint: domain.Fingerprint{
				Mode:    uint32(entryInfo.Mode()),
				Size:    size,
				ModTime: newest,
			},
			Source: "iOS device backup",
		})
	}
	return findings, nil, nil
}

func installerRootLabel(root string) string {
	normalized := strings.ToLower(filepath.ToSlash(root))
	switch {
	case strings.Contains(normalized, "homebrew/downloads"):
		return "Homebrew"
	case strings.Contains(normalized, "mail downloads"):
		return "Mail"
	case strings.Contains(normalized, "telegram desktop"):
		return "Telegram"
	case strings.Contains(normalized, "mobile documents") && strings.Contains(normalized, "/downloads"):
		return "iCloud"
	case strings.HasSuffix(normalized, "/downloads"):
		return "Downloads"
	case strings.HasSuffix(normalized, "/desktop"):
		return "Desktop"
	case strings.HasSuffix(normalized, "/documents"):
		return "Documents"
	case strings.Contains(normalized, "/public"):
		return "Public"
	case strings.Contains(normalized, "/users/shared"):
		return "Shared"
	default:
		return "Installer"
	}
}

func zipLooksLikeInstaller(path string) (bool, []string) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return false, []string{domain.NormalizePath(path) + ": " + err.Error()}
	}
	defer reader.Close()
	for _, file := range reader.File {
		name := strings.ToLower(file.Name)
		ext := strings.ToLower(filepath.Ext(name))
		if slices.Contains([]string{".pkg", ".dmg", ".exe", ".msi", ".appinstaller"}, ext) {
			return true, nil
		}
		base := strings.ToLower(strings.TrimSuffix(filepath.Base(name), ext))
		if strings.Contains(base, "install") || strings.Contains(base, "setup") {
			return true, nil
		}
	}
	return false, nil
}
