package rules

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/platform"
)

func scanPurgeTargets(ctx context.Context, targets []string, adapter platform.Adapter) ([]domain.Finding, []string, error) {
	var findings []domain.Finding
	var warnings []string
	for _, target := range targets {
		select {
		case <-ctx.Done():
			return findings, warnings, ctx.Err()
		default:
		}
		normalized := domain.NormalizePath(target)
		info, err := os.Lstat(normalized)
		if errors.Is(err, os.ErrNotExist) {
			warnings = append(warnings, normalized+": target not found")
			continue
		}
		if err != nil {
			warnings = append(warnings, normalized+": "+err.Error())
			continue
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			warnings = append(warnings, normalized+": symlink targets are not valid purge artifacts")
			continue
		}
		if !info.IsDir() {
			warnings = append(warnings, normalized+": purge only supports artifact directories")
			continue
		}
		if !isKnownPurgeArtifact(normalized) {
			warnings = append(warnings, normalized+": not a known purge artifact")
			continue
		}
		if isShallowPurgeTarget(normalized) {
			warnings = append(warnings, normalized+": purge target is too shallow to be safe")
			continue
		}
		if container := protectedPurgeContainer(normalized); container != "" {
			warnings = append(warnings, normalized+": purge target is nested inside protected "+container+" dependencies")
			continue
		}
		marker, ok := nearestProjectMarker(normalized)
		if !ok {
			warnings = append(warnings, normalized+": no nearby project marker found")
			continue
		}
		size, newest, err := MeasurePath(ctx, normalized)
		if err != nil {
			warnings = append(warnings, normalized+": "+err.Error())
			continue
		}
		if size == 0 {
			continue
		}
		risk := domain.RiskReview
		recovery := domain.RecoveryHint{
			Message:  "Restore from Trash/Recycle Bin if this artifact is still needed.",
			Location: "system trash",
		}
		if time.Since(newest) < 7*24*time.Hour {
			risk = domain.RiskHigh
			recovery.Message = "Artifact was modified within the last 7 days. Review before deletion."
		}
		source := "Project purge target"
		if label := projectMarkerLabel(marker); label != "" {
			source = "Project purge target (" + label + ")"
		}
		findings = append(findings, domain.Finding{
			ID:            uuid.NewString(),
			RuleID:        "purge.project_artifact",
			Name:          filepath.Base(normalized),
			Category:      domain.CategoryProjectArtifacts,
			Path:          normalized,
			DisplayPath:   normalized,
			Risk:          risk,
			Bytes:         size,
			RequiresAdmin: adapter.IsAdminPath(normalized),
			Action:        domain.ActionTrash,
			Recovery:      recovery,
			Status:        domain.StatusPlanned,
			LastModified:  newest,
			Fingerprint: domain.Fingerprint{
				Mode:    uint32(info.Mode()),
				Size:    size,
				ModTime: newest,
			},
			Source: source,
		})
	}
	return findings, warnings, nil
}

func scanPurgeDiscovery(ctx context.Context, roots []string, adapter platform.Adapter) ([]domain.Finding, []string, error) {
	var warnings []string
	discovered := make([]string, 0, len(roots))
	seen := map[string]struct{}{}
	for _, root := range roots {
		select {
		case <-ctx.Done():
			return nil, warnings, ctx.Err()
		default:
		}
		normalized := domain.NormalizePath(root)
		info, err := os.Lstat(normalized)
		if errors.Is(err, os.ErrNotExist) {
			warnings = append(warnings, normalized+": search root not found")
			continue
		}
		if err != nil {
			warnings = append(warnings, normalized+": "+err.Error())
			continue
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			warnings = append(warnings, normalized+": search root symlink skipped")
			continue
		}
		if !info.IsDir() {
			warnings = append(warnings, normalized+": search root must be a directory")
			continue
		}
		err = filepath.WalkDir(normalized, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				warnings = append(warnings, domain.NormalizePath(path)+": "+walkErr.Error())
				if entry != nil && entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if entry.Type()&fs.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !entry.IsDir() {
				return nil
			}
			if _, ok := protectedPurgeContainers[filepath.Base(path)]; ok {
				return filepath.SkipDir
			}
			if isKnownPurgeArtifact(path) {
				normalizedPath := domain.NormalizePath(path)
				if _, ok := seen[normalizedPath]; !ok {
					seen[normalizedPath] = struct{}{}
					discovered = append(discovered, normalizedPath)
				}
				return filepath.SkipDir
			}
			base := filepath.Base(path)
			switch base {
			case ".git", ".svn", ".hg", ".Trash", "$Recycle.Bin":
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			warnings = append(warnings, normalized+": "+err.Error())
		}
	}
	findings, targetWarnings, err := scanPurgeTargets(ctx, discovered, adapter)
	warnings = append(warnings, targetWarnings...)
	if len(findings) > maxPurgeDiscoveryFindings {
		findings = findings[:maxPurgeDiscoveryFindings]
		warnings = append(warnings, "purge discovery: capped to top "+strconv.Itoa(maxPurgeDiscoveryFindings)+" artifact findings")
	}
	sortFindings(findings)
	return findings, dedupeStrings(warnings), err
}

func scanAppSupportRoots(ctx context.Context, adapter platform.Adapter) ([]domain.Finding, []string, error) {
	apps, err := adapter.ListApps(ctx, false)
	if err != nil {
		return nil, nil, err
	}
	appNames := make(map[string]struct{}, len(apps)*2)
	for _, app := range apps {
		for _, name := range []string{app.DisplayName, app.Name, strings.TrimSuffix(filepath.Base(app.BundlePath), filepath.Ext(app.BundlePath))} {
			key := normalizedCleanupKey(name)
			if key == "" {
				continue
			}
			appNames[key] = struct{}{}
		}
	}
	var findings []domain.Finding
	var warnings []string
	for _, root := range adapter.CuratedRoots().AppSupport {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			select {
			case <-ctx.Done():
				return findings, warnings, ctx.Err()
			default:
			}
			name := normalizedCleanupKey(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
			if _, ok := appNames[name]; ok {
				continue
			}
			path := domain.NormalizePath(filepath.Join(root, entry.Name()))
			if isSIFTOwnedPath(path) {
				continue
			}
			info, err := os.Stat(path)
			if err != nil {
				warnings = append(warnings, path+": "+err.Error())
				continue
			}
			size, newest, err := MeasurePath(ctx, path)
			if err != nil || size == 0 {
				continue
			}
			if time.Since(newest) < 45*24*time.Hour {
				continue
			}
			findings = append(findings, domain.Finding{
				ID:          uuid.NewString(),
				RuleID:      "app_leftovers",
				Name:        entry.Name(),
				Category:    domain.CategoryAppLeftovers,
				Path:        path,
				DisplayPath: path,
				Risk:        domain.RiskReview,
				Bytes:       size,
				Action:      domain.ActionTrash,
				Recovery: domain.RecoveryHint{
					Message:  "Verify app was removed before deleting leftovers.",
					Location: "system trash",
				},
				Status:       domain.StatusPlanned,
				LastModified: newest,
				Fingerprint: domain.Fingerprint{
					Mode:    uint32(info.Mode()),
					Size:    size,
					ModTime: newest,
				},
				RequiresAdmin: adapter.IsAdminPath(path),
				Source:        "Stale support directory",
			})
		}
	}
	return findings, warnings, nil
}

func normalizedCleanupKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isKnownPurgeArtifact(path string) bool {
	_, ok := purgeArtifactNames[filepath.Base(path)]
	return ok
}

func isShallowPurgeTarget(path string) bool {
	parent := filepath.Dir(path)
	if parent == path {
		return true
	}
	if home, err := os.UserHomeDir(); err == nil {
		cleanHome := domain.NormalizePath(home)
		if domain.HasPathPrefix(path, cleanHome) {
			rel, err := filepath.Rel(cleanHome, path)
			if err == nil {
				parts := strings.Split(rel, string(filepath.Separator))
				if len(parts) <= 1 {
					return true
				}
			}
		}
	}
	root := filepath.VolumeName(path) + string(filepath.Separator)
	return parent == root
}

func hasNearbyProjectMarker(path string) bool {
	_, ok := nearestProjectMarker(path)
	return ok
}

func nearestProjectMarker(path string) (string, bool) {
	dir := filepath.Dir(path)
	for range 6 {
		if dir == "" || dir == filepath.Dir(dir) {
			break
		}
		for _, marker := range projectMarkers {
			if strings.HasPrefix(marker, "*.") {
				matches, err := filepath.Glob(filepath.Join(dir, marker))
				if err == nil && len(matches) > 0 {
					return marker, true
				}
				continue
			}
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return marker, true
			}
		}
		dir = filepath.Dir(dir)
	}
	return "", false
}

func projectMarkerLabel(marker string) string {
	switch marker {
	case "pnpm-workspace.yaml", "turbo.json", "nx.json", "rush.json", "lerna.json":
		return "workspace"
	case "WORKSPACE", "WORKSPACE.bazel", "MODULE.bazel":
		return "bazel workspace"
	case "*.sln", "Directory.Build.props", "Directory.Build.targets":
		return ".NET solution"
	case "build.zig", "build.zig.zon":
		return "Zig project"
	case "*.xcodeproj", "*.xcworkspace":
		return "Xcode project"
	case "package.json":
		return "Node project"
	case "pubspec.yaml":
		return "Dart project"
	case "go.mod":
		return "Go module"
	case "Cargo.toml", "Cargo.lock":
		return "Rust crate"
	case "pyproject.toml", "requirements.txt", "Pipfile":
		return "Python project"
	case "composer.json":
		return "PHP project"
	case "Gemfile":
		return "Ruby project"
	case "pom.xml", "build.gradle", "build.gradle.kts":
		return "JVM project"
	case "Makefile":
		return "build workspace"
	case ".git":
		return "git repository"
	default:
		return ""
	}
}

func protectedPurgeContainer(path string) string {
	dir := filepath.Dir(path)
	for dir != "" {
		base := filepath.Base(dir)
		if _, ok := protectedPurgeContainers[base]; ok {
			return base
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
