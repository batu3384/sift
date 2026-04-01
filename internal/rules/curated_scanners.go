package rules

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/platform"
)

// ProtectedWebEditors are web editor domains that should be protected from Service Worker cache cleanup (Mole-style)
var ProtectedWebEditors = []string{
	"capcut.com",
	"photopea.com",
	"pixlr.com",
	"canva.com",
	"figma.com",
}

func scanRootEntries(ctx context.Context, adapter platform.Adapter, roots []string, category domain.Category, risk domain.Risk, action domain.Action, source string) ([]domain.Finding, []string, error) {
	var findings []domain.Finding
	var warnings []string
	curated := allCuratedRoots(adapter)
	for _, root := range roots {
		select {
		case <-ctx.Done():
			return findings, warnings, ctx.Err()
		default:
		}
		rootFindings, rootWarnings, err := scanCuratedRoot(ctx, root, category, risk, action, source, curated)
		if err != nil {
			warnings = append(warnings, domain.NormalizePath(root)+": "+err.Error())
			continue
		}
		for idx := range rootFindings {
			rootFindings[idx].RequiresAdmin = adapter.IsAdminPath(rootFindings[idx].Path)
		}
		findings = append(findings, rootFindings...)
		warnings = append(warnings, capWarningsWithSummary(domain.NormalizePath(root), rootWarnings, 8)...)
	}
	sortFindings(findings)
	return findings, dedupeStrings(warnings), nil
}

func scanBrowserRoots(ctx context.Context, adapter platform.Adapter, roots []string) ([]domain.Finding, []string, error) {
	var findings []domain.Finding
	var warnings []string
	curated := allCuratedRoots(adapter)
	for _, root := range roots {
		select {
		case <-ctx.Done():
			return findings, warnings, ctx.Err()
		default:
		}
		normalized := domain.NormalizePath(root)
		if edgeFindings, edgeWarnings, handled, err := scanEdgeUpdaterOldVersions(ctx, normalized); handled {
			if err != nil {
				warnings = append(warnings, normalized+": "+err.Error())
				continue
			}
			findings = append(findings, edgeFindings...)
			warnings = append(warnings, edgeWarnings...)
			continue
		}
		base := strings.ToLower(filepath.Base(normalized))
		if base == "profiles" {
			profileEntries, err := os.ReadDir(normalized)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					warnings = append(warnings, normalized+": "+err.Error())
				}
				continue
			}
			for _, profileEntry := range profileEntries {
				if !profileEntry.IsDir() {
					continue
				}
				profileRoot := filepath.Join(normalized, profileEntry.Name())
				for _, cacheDir := range []string{"cache2", "startupCache", "thumbnails", "shader-cache", "jumpListCache"} {
					candidate := filepath.Join(profileRoot, cacheDir)
					rootFindings, rootWarnings, err := scanCuratedRoot(ctx, candidate, domain.CategoryBrowserData, domain.RiskReview, domain.ActionTrash, "Browser cache", curated)
					if err != nil {
						warnings = append(warnings, domain.NormalizePath(candidate)+": "+err.Error())
						continue
					}
					for idx := range rootFindings {
						rootFindings[idx].RequiresAdmin = adapter.IsAdminPath(rootFindings[idx].Path)
					}
					findings = append(findings, rootFindings...)
					warnings = append(warnings, capWarningsWithSummary(domain.NormalizePath(candidate), rootWarnings, 4)...)
				}
			}
			continue
		}
		rootFindings, rootWarnings, err := scanCuratedRoot(ctx, normalized, domain.CategoryBrowserData, domain.RiskReview, domain.ActionTrash, "Browser cache", curated)
		if err != nil {
			warnings = append(warnings, normalized+": "+err.Error())
			continue
		}
		for idx := range rootFindings {
			rootFindings[idx].RequiresAdmin = adapter.IsAdminPath(rootFindings[idx].Path)
		}
		findings = append(findings, rootFindings...)
		warnings = append(warnings, capWarningsWithSummary(normalized, rootWarnings, 8)...)
	}

	sortFindings(findings)
	return findings, dedupeStrings(warnings), nil
}

func scanEdgeUpdaterOldVersions(ctx context.Context, root string) ([]domain.Finding, []string, bool, error) {
	normalized := strings.ToLower(strings.ReplaceAll(domain.NormalizePath(root), `\`, `/`))
	if !strings.Contains(normalized, "/application support/microsoft/edgeupdater/apps/msedge-stable") {
		return nil, nil, false, nil
	}
	if runtime.GOOS == "darwin" && darwinEdgeRunning() {
		return nil, []string{"edge updater cleanup skipped because Microsoft Edge is running"}, true, nil
	}
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, true, nil
	}
	if err != nil {
		return nil, nil, true, err
	}
	var versionDirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		versionDirs = append(versionDirs, domain.NormalizePath(filepath.Join(root, entry.Name())))
	}
	if len(versionDirs) < 2 {
		return nil, nil, true, nil
	}
	slices.SortFunc(versionDirs, func(a, b string) int {
		return strings.Compare(filepath.Base(a), filepath.Base(b))
	})
	latest := versionDirs[0]
	for _, candidate := range versionDirs[1:] {
		if compareVersionish(filepath.Base(candidate), filepath.Base(latest)) > 0 {
			latest = candidate
		}
	}
	var findings []domain.Finding
	for _, candidate := range versionDirs {
		if candidate == latest {
			continue
		}
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		size, newest, err := MeasurePath(ctx, candidate)
		if err != nil || size == 0 {
			continue
		}
		findings = append(findings, newFinding(filepath.Base(candidate), candidate, domain.CategoryBrowserData, domain.RiskReview, domain.ActionTrash, size, newest, info.Mode(), "Edge updater old version"))
	}
	sortFindings(findings)
	return findings, nil, true, nil
}

func scanDeveloperRoots(ctx context.Context, adapter platform.Adapter) ([]domain.Finding, []string, error) {
	roots := adapter.CuratedRoots().Developer
	baseRoots := make([]string, 0, len(roots))

	// Check if Xcode is running (Mole-style: skip DerivedData/Archives when Xcode is running)
	xcodeRunning := adapter.IsProcessRunning("Xcode")

	for _, root := range roots {
		if isCoreSimulatorDevicesRoot(root) {
			continue
		}
		// Skip Xcode DerivedData and Archives if Xcode is running (Mole-style)
		if xcodeRunning {
			if strings.Contains(root, "DerivedData") || strings.Contains(root, "Archives") {
				continue
			}
		}
		baseRoots = append(baseRoots, root)
	}

	findings, warnings, err := scanRootEntries(ctx, adapter, baseRoots, domain.CategoryDeveloperCaches, domain.RiskReview, domain.ActionTrash, "Developer cache")
	if err != nil {
		return findings, warnings, err
	}

	// Add warning if Xcode is running
	if xcodeRunning {
		warnings = append(warnings, "Xcode is running, skipped DerivedData and Archives cleanup")
	}

	extraRoots := discoverDeveloperSpecificRoots(adapter)
	if len(extraRoots) == 0 {
		return findings, warnings, nil
	}
	curated := allCuratedRoots(adapter)
	for _, extra := range extraRoots {
		select {
		case <-ctx.Done():
			return findings, warnings, ctx.Err()
		default:
		}
		rootFindings, rootWarnings, err := scanCuratedRoot(ctx, extra.path, domain.CategoryDeveloperCaches, domain.RiskReview, domain.ActionTrash, extra.label, curated)
		if err != nil {
			warnings = append(warnings, extra.path+": "+err.Error())
			continue
		}
		for idx := range rootFindings {
			rootFindings[idx].RequiresAdmin = adapter.IsAdminPath(rootFindings[idx].Path)
		}
		findings = append(findings, rootFindings...)
		warnings = append(warnings, capWarningsWithSummary(extra.path, rootWarnings, 4)...)
	}
	sortFindings(findings)
	return findings, dedupeStrings(warnings), nil
}

type labeledRoot struct {
	path  string
	label string
}

func discoverDeveloperSpecificRoots(adapter platform.Adapter) []labeledRoot {
	var roots []labeledRoot
	for _, root := range adapter.CuratedRoots().Developer {
		if isCoreSimulatorDevicesRoot(root) {
			for _, pattern := range []struct {
				suffix string
				label  string
			}{
				{suffix: filepath.Join("*", "data", "Library", "Caches"), label: "CoreSimulator device cache"},
				{suffix: filepath.Join("*", "data", "tmp"), label: "CoreSimulator device temp"},
			} {
				matches, _ := filepath.Glob(filepath.Join(root, pattern.suffix))
				for _, match := range matches {
					normalized := domain.NormalizePath(match)
					if normalized == "" {
						continue
					}
					roots = append(roots, labeledRoot{path: normalized, label: pattern.label})
				}
			}
		}
		if strings.HasSuffix(strings.ToLower(strings.ReplaceAll(domain.NormalizePath(root), `\`, `/`)), "/.jenkins/workspace") {
			for _, pattern := range []struct {
				suffix string
				label  string
			}{
				{suffix: filepath.Join("*", "target"), label: "Jenkins workspace target"},
				{suffix: filepath.Join("*", "build"), label: "Jenkins workspace build"},
			} {
				matches, _ := filepath.Glob(filepath.Join(root, pattern.suffix))
				for _, match := range matches {
					normalized := domain.NormalizePath(match)
					if normalized == "" {
						continue
					}
					roots = append(roots, labeledRoot{path: normalized, label: pattern.label})
				}
			}
		}
	}
	return dedupeLabeledRoots(roots)
}

func dedupeLabeledRoots(in []labeledRoot) []labeledRoot {
	seen := map[string]struct{}{}
	out := make([]labeledRoot, 0, len(in))
	for _, item := range in {
		key := item.path + "|" + item.label
		if item.path == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func scanCuratedRoot(ctx context.Context, root string, category domain.Category, risk domain.Risk, action domain.Action, source string, curated []string) ([]domain.Finding, []string, error) {
	label := cleanupSourceLabel(root, source)
	if shouldTreatRootAsLeaf(root, category) {
		finding, warnings, err := scanRootAsFinding(ctx, root, category, risk, action, label)
		if err != nil {
			return nil, warnings, err
		}
		if finding.Path == "" {
			return nil, warnings, nil
		}
		return []domain.Finding{finding}, warnings, nil
	}
	return scanImmediateChildren(ctx, root, category, risk, action, label, curated)
}
