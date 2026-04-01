package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func (s *Service) BuildUninstallPlan(ctx context.Context, appName string, dryRun, allowAdmin bool) (domain.ExecutionPlan, error) {
	apps, err := s.Adapter.ListApps(ctx, allowAdmin)
	if err != nil {
		return domain.ExecutionPlan{}, err
	}
	requestedName := strings.TrimSpace(appName)
	if requestedName == "" {
		return domain.ExecutionPlan{}, fmt.Errorf("app name is required")
	}
	requestedKey := normalizedAppKey(requestedName)
	app, found := findInstalledApp(apps, requestedName)
	targetLabel := requestedName
	warnings := []string{}
	if !found {
		fallback := domain.AppEntry{
			Name:        requestedName,
			DisplayName: requestedName,
		}
		app = &fallback
		targetLabel = uninstallTargetLabel(*app)
		warnings = append(warnings, fmt.Sprintf("%s is not currently listed as installed. Scanning for leftover files by name only.", requestedName))
		if requestedKey == "sift" {
			warnings = append(warnings, "Use `sift remove` to delete SIFT-owned state. Binary uninstall remains a separate manual or package-manager step.")
		}
	} else {
		targetLabel = uninstallTargetLabel(*app)
	}
	remnants, remnantWarnings, err := s.Adapter.DiscoverRemnants(ctx, *app)
	if err != nil {
		return domain.ExecutionPlan{}, err
	}
	candidatePaths := dedupePaths(append(append([]string{app.BundlePath}, app.SupportPaths...), remnants...))
	var items []domain.Finding
	warnings = append(warnings, remnantWarnings...)
	if advisory, ok := uninstallAdvisoryFinding(*app); ok {
		items = append(items, advisory)
		if advisory.Action == domain.ActionNative {
			warnings = append(warnings, "Native uninstall is available. If you launch it, SIFT will continue with remnant cleanup and aftercare in the same run.")
		}
	}
	if found && requestedKey != "sift" {
		items = append(items, s.uninstallAftermathAdvisories(*app)...)
	}
	if requestedKey == "sift" {
		items = append(items, domain.Finding{
			ID:          uuid.NewString(),
			RuleID:      "uninstall.sift",
			Name:        "Use remove for SIFT-owned state",
			Category:    domain.CategoryMaintenance,
			Path:        "sift remove",
			DisplayPath: "sift remove",
			Risk:        domain.RiskReview,
			Action:      domain.ActionAdvisory,
			Status:      domain.StatusAdvisory,
			Recovery: domain.RecoveryHint{
				Message:  "Run `sift remove --dry-run=false --yes` to remove SIFT config, reports, and audit data. Then uninstall the binary with your package manager or manually.",
				Location: "manual follow-up",
			},
			Source: "SIFT self-removal guidance",
		})
	}
	for _, path := range candidatePaths {
		if path == "" {
			continue
		}
		normalized := domain.NormalizePath(path)
		if requestedKey == "sift" && isSIFTOwnedStatePath(normalized) {
			continue
		}
		info, err := os.Lstat(normalized)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			continue
		}
		size, newest, err := rulesMeasurePath(ctx, normalized)
		if err != nil {
			continue
		}
		item := domain.Finding{
			ID:            uuid.NewString(),
			RuleID:        "uninstall",
			Name:          filepath.Base(normalized),
			Category:      domain.CategoryAppLeftovers,
			Path:          normalized,
			DisplayPath:   normalized,
			Risk:          domain.RiskReview,
			Bytes:         size,
			RequiresAdmin: app.RequiresAdmin || s.Adapter.IsAdminPath(normalized),
			Action:        domain.ActionTrash,
			Recovery: domain.RecoveryHint{
				Message:  app.UninstallHint,
				Location: "system trash",
			},
			Status:       domain.StatusPlanned,
			LastModified: newest,
			Fingerprint: domain.Fingerprint{
				Mode:    uint32(info.Mode()),
				Size:    size,
				ModTime: newest,
			},
			Source: targetLabel + " app bundle",
		}
		if normalized != domain.NormalizePath(app.BundlePath) {
			item.Source = targetLabel + " remnants"
		}
		items = append(items, item)
	}
	// Build list of allowed roots for uninstall - include the app bundle path
	// This allows deleting apps even though /Applications is system-protected
	allowedRoots := appPathsForPolicy(items)
	if app.BundlePath != "" {
		allowedRoots = append(allowedRoots, app.BundlePath)
	}

	policy := s.buildPolicy(ScanOptions{
		Command:    "uninstall",
		DryRun:     dryRun,
		AllowAdmin: allowAdmin,
	}, nil, allowedRoots)
	for idx := range items {
		items[idx] = applyPolicy(items[idx], evaluatePolicy(policy, items[idx], false))
	}
	if runningProcesses, err := listRunningProcesses(ctx); err == nil {
		if appIsRunning(*app, runningProcesses) {
			items = protectForRunningApp(items, *app)
			warnings = append(warnings, fmt.Sprintf("%s appears to be running. Close it and rerun uninstall.", app.DisplayName))
		}
	} else {
		warnings = append(warnings, "running process detection unavailable: "+err.Error())
	}
	if !found && len(items) == 0 {
		warnings = append(warnings, fmt.Sprintf("No installed app or leftover files were found for %s.", requestedName))
	}
	plan := domain.ExecutionPlan{
		ScanID:               uuid.NewString(),
		Command:              "uninstall",
		Platform:             s.Adapter.Name(),
		CreatedAt:            time.Now().UTC(),
		PlanState:            "preview",
		DryRun:               dryRun,
		RequiresConfirmation: s.requiresConfirmation("uninstall", dryRun, items),
		Warnings:             dedupe(warnings),
		Items:                items,
		Totals:               calculateTotals(items),
		Targets:              []string{coalesce(app.DisplayName, requestedName)},
		Policy:               policy,
	}
	if len(items) == 0 {
		plan.PlanState = "empty"
	}
	if s.Store != nil {
		_ = s.Store.SavePlan(ctx, plan)
	}
	return plan, nil
}

func findInstalledApp(apps []domain.AppEntry, requestedName string) (*domain.AppEntry, bool) {
	requestedKey := normalizedAppKey(requestedName)
	for _, candidate := range apps {
		for _, value := range []string{
			candidate.DisplayName,
			candidate.Name,
			strings.TrimSuffix(filepath.Base(candidate.BundlePath), filepath.Ext(candidate.BundlePath)),
		} {
			if strings.EqualFold(strings.TrimSpace(value), requestedName) {
				candidateCopy := candidate
				return &candidateCopy, true
			}
			if requestedKey != "" && normalizedAppKey(value) == requestedKey {
				candidateCopy := candidate
				return &candidateCopy, true
			}
		}
	}
	return nil, false
}

func normalizedAppKey(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isSIFTOwnedStatePath(path string) bool {
	normalized := strings.ToLower(filepath.ToSlash(domain.NormalizePath(path)))
	switch {
	case strings.Contains(normalized, "/library/application support/sift"):
		return true
	case strings.Contains(normalized, "/library/caches/sift"):
		return true
	case strings.Contains(normalized, "/library/logs/sift"):
		return true
	case strings.Contains(normalized, "/appdata/roaming/sift"):
		return true
	case strings.Contains(normalized, "/appdata/local/sift"):
		return true
	case strings.Contains(normalized, "/.cache/sift"):
		return true
	default:
		return false
	}
}
