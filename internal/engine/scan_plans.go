package engine

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/rules"
)

func (s *Service) Scan(ctx context.Context, opts ScanOptions) (domain.ExecutionPlan, error) {
	hadTargets := len(opts.Targets) > 0
	sanitizedTargets, targetWarnings := s.sanitizeTargets(opts.Targets)
	opts.Targets = sanitizedTargets
	selected := s.selectDefinitions(opts, hadTargets)
	policy := s.buildPolicy(opts, selected, nil)
	findingsByPath := map[string]domain.Finding{}
	warnings := append([]string{}, targetWarnings...)
	// Resolve targets for the scan (handles ~ expansion, etc.)
	resolvedTargets := s.resolveTargetsForPlan(opts)

	for _, definition := range selected {
		items, ruleWarnings, err := definition.Scanner(ctx, s.Adapter, resolvedTargets)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", definition.ID, err))
			continue
		}
		warnings = append(warnings, ruleWarnings...)

		// Calculate bytes for this category
		var categoryBytes int64
		for _, item := range items {
			categoryBytes += item.Bytes
		}

		// Call progress callback if provided (for CLI progress feedback like Mole)
		if opts.CategoryCallback != nil && len(items) > 0 {
			opts.CategoryCallback(definition.ID, definition.Name, len(items), categoryBytes)
		}

		for _, item := range items {
			item.Path = domain.NormalizePath(item.Path)
			item.DisplayPath = coalesce(item.DisplayPath, item.Path)
			item.RuleID = coalesce(item.RuleID, definition.ID)
			item.Name = coalesce(item.Name, definition.Name)
			if item.Category == "" {
				item.Category = definition.Category
			}
			if item.Risk == "" {
				item.Risk = definition.Risk
			}
			if item.Action == "" {
				item.Action = definition.Action
			}
			item = applyPolicy(item, evaluatePolicy(policy, item, false))
			if prev, ok := findingsByPath[item.Path]; ok {
				if item.Bytes > prev.Bytes {
					findingsByPath[item.Path] = item
				}
				continue
			}
			findingsByPath[item.Path] = item
		}
	}
	items := make([]domain.Finding, 0, len(findingsByPath))
	for _, item := range findingsByPath {
		items = append(items, item)
	}
	sortPlanItems(opts.Command, items)
	plan := domain.ExecutionPlan{
		ScanID:               uuid.NewString(),
		Command:              opts.Command,
		Profile:              opts.Profile,
		Platform:             s.Adapter.Name(),
		CreatedAt:            time.Now().UTC(),
		PlanState:            "preview",
		DryRun:               opts.DryRun,
		RequiresConfirmation: s.requiresConfirmation(opts.Command, opts.DryRun, items),
		Warnings:             dedupe(warnings),
		Items:                items,
		Totals:               calculateTotals(items),
		Targets:              s.resolveTargetsForPlan(opts),
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

func sortPlanItems(command string, items []domain.Finding) {
	if len(items) == 0 {
		return
	}
	if command == "analyze" {
		slices.SortFunc(items, compareFindingBySize)
		return
	}
	slices.SortFunc(items, func(a, b domain.Finding) int {
		aRank := categoryExecutionRank(a.Category)
		bRank := categoryExecutionRank(b.Category)
		if aRank != bRank {
			if aRank < bRank {
				return -1
			}
			return 1
		}
		if a.Action != b.Action {
			if a.Action == domain.ActionAdvisory {
				return 1
			}
			if b.Action == domain.ActionAdvisory {
				return -1
			}
		}
		aGroup := strings.ToLower(domain.ExecutionGroupLabel(a))
		bGroup := strings.ToLower(domain.ExecutionGroupLabel(b))
		if aGroup != bGroup {
			return strings.Compare(aGroup, bGroup)
		}
		return compareFindingBySize(a, b)
	})
}

func compareFindingBySize(a, b domain.Finding) int {
	if a.Bytes == b.Bytes {
		return strings.Compare(a.Path, b.Path)
	}
	if a.Bytes > b.Bytes {
		return -1
	}
	return 1
}

func categoryExecutionRank(category domain.Category) int {
	switch category {
	case domain.CategoryTempFiles:
		return 10
	case domain.CategorySystemClutter:
		return 20
	case domain.CategoryBrowserData:
		return 30
	case domain.CategoryPackageCaches:
		return 40
	case domain.CategoryDeveloperCaches:
		return 50
	case domain.CategoryLogs:
		return 60
	case domain.CategoryInstallerLeft:
		return 70
	case domain.CategoryProjectArtifacts:
		return 80
	case domain.CategoryAppLeftovers:
		return 90
	case domain.CategoryMaintenance:
		return 100
	case domain.CategoryDiskUsage:
		return 110
	case domain.CategoryLargeFiles:
		return 120
	default:
		return 999
	}
}

func (s *Service) requiresConfirmation(command string, dryRun bool, items []domain.Finding) bool {
	if dryRun {
		return false
	}
	actionable := 0
	for _, item := range items {
		if item.Status == domain.StatusProtected || item.Action == domain.ActionAdvisory {
			continue
		}
		actionable++
	}
	if actionable == 0 {
		return false
	}
	if s.Config.ConfirmLevel != "balanced" {
		return true
	}
	if command == "purge" || command == "uninstall" {
		return true
	}
	for _, item := range items {
		if item.Status == domain.StatusProtected || item.Action == domain.ActionAdvisory {
			continue
		}
		if item.Risk != domain.RiskSafe || item.Action != domain.ActionTrash || item.RequiresAdmin {
			return true
		}
	}
	return false
}

func (s *Service) selectDefinitions(opts ScanOptions, explicitTargets bool) []rules.Definition {
	if explicitTargets && len(opts.Targets) == 0 {
		return nil
	}
	if opts.Command == "analyze" {
		return rules.AnalysisDefinitions(s.resolveAnalysisTargets(opts.Targets))
	}
	if opts.Command == "purge_scan" {
		return rules.PurgeDiscoveryDefinitions(opts.Targets)
	}
	if opts.Command == "purge" && len(opts.Targets) > 0 {
		return rules.PurgeTargetDefinitions(opts.Targets)
	}
	if len(opts.Targets) > 0 {
		return rules.TargetDefinitions(opts.Targets)
	}
	if explicitTargets {
		return nil
	}
	if len(opts.RuleIDs) > 0 {
		return rules.ByIDs(opts.RuleIDs)
	}
	ruleIDs := s.Config.Profiles[opts.Profile]
	if len(ruleIDs) == 0 {
		ruleIDs = s.Config.Profiles["safe"]
	}
	filtered := make([]string, 0, len(ruleIDs))
	for _, ruleID := range ruleIDs {
		if slices.Contains(s.Config.DisabledRules, ruleID) {
			continue
		}
		filtered = append(filtered, ruleID)
	}
	return rules.ByIDs(filtered)
}

func (s *Service) resolveTargetsForPlan(opts ScanOptions) []string {
	if opts.Command == "analyze" {
		return s.resolveAnalysisTargets(opts.Targets)
	}
	if opts.Command == "purge_scan" {
		return s.Adapter.ResolveTargets(opts.Targets)
	}
	return s.Adapter.ResolveTargets(opts.Targets)
}

func (s *Service) resolveAnalysisTargets(targets []string) []string {
	if len(targets) > 0 {
		return s.Adapter.ResolveTargets(targets)
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return []string{home}
	}
	return []string{"."}
}

func (s *Service) sanitizeTargets(targets []string) ([]string, []string) {
	out := make([]string, 0, len(targets))
	warnings := make([]string, 0)
	for _, target := range targets {
		if domain.HasControlChars(target) {
			warnings = append(warnings, "target rejected: control characters are not allowed")
			continue
		}
		if domain.ContainsTraversal(target) {
			warnings = append(warnings, fmt.Sprintf("target rejected: traversal is not allowed: %s", target))
			continue
		}
		out = append(out, target)
	}
	return out, warnings
}

func dedupePaths(paths []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		normalized := domain.NormalizePath(path)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}
