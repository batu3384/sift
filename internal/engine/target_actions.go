package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/rules"
)

func (s *Service) TrashPaths(ctx context.Context, command string, targets []string, allowAdmin bool) (domain.ExecutionResult, error) {
	plan, err := s.buildTargetActionPlan(ctx, command, targets, false, allowAdmin)
	if err != nil {
		return domain.ExecutionResult{}, err
	}
	if len(plan.Items) == 0 {
		return domain.ExecutionResult{}, fmt.Errorf("no actionable paths selected")
	}
	return s.ExecuteWithOptions(ctx, plan, ExecuteOptions{})
}

func (s *Service) buildTargetActionPlan(ctx context.Context, command string, targets []string, dryRun, allowAdmin bool) (domain.ExecutionPlan, error) {
	if len(targets) == 0 {
		return domain.ExecutionPlan{}, fmt.Errorf("at least one target path is required")
	}
	sanitizedTargets, targetWarnings := s.sanitizeTargets(targets)
	if len(sanitizedTargets) == 0 {
		return domain.ExecutionPlan{}, fmt.Errorf("no valid target paths remain after policy sanitization")
	}
	defs := rules.TargetDefinitions(sanitizedTargets)
	opts := ScanOptions{
		Command:    command,
		Targets:    sanitizedTargets,
		DryRun:     dryRun,
		AllowAdmin: allowAdmin,
	}
	policy := s.buildPolicy(opts, defs, nil)
	findingsByPath := map[string]domain.Finding{}
	warnings := append([]string{}, targetWarnings...)
	for _, definition := range defs {
		items, ruleWarnings, err := definition.Scanner(ctx, s.Adapter, sanitizedTargets)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", definition.ID, err))
			continue
		}
		warnings = append(warnings, ruleWarnings...)
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
	sortPlanItems(command, items)
	plan := domain.ExecutionPlan{
		ScanID:               uuid.NewString(),
		Command:              command,
		Platform:             s.Adapter.Name(),
		CreatedAt:            time.Now().UTC(),
		PlanState:            "preview",
		DryRun:               dryRun,
		RequiresConfirmation: false,
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

func mergeProtectionPolicies(policies []domain.ProtectionPolicy) domain.ProtectionPolicy {
	if len(policies) == 0 {
		return domain.ProtectionPolicy{}
	}
	merged := policies[0]
	for _, policy := range policies[1:] {
		merged.ProtectedPaths = dedupe(append(merged.ProtectedPaths, policy.ProtectedPaths...))
		merged.UserProtectedPaths = dedupe(append(merged.UserProtectedPaths, policy.UserProtectedPaths...))
		merged.SystemProtectedPaths = dedupe(append(merged.SystemProtectedPaths, policy.SystemProtectedPaths...))
		merged.FamilyProtectedPaths = dedupe(append(merged.FamilyProtectedPaths, policy.FamilyProtectedPaths...))
		merged.ProtectedFamilies = dedupeLower(append(merged.ProtectedFamilies, policy.ProtectedFamilies...))
		merged.ProtectedPathExceptions = dedupe(append(merged.ProtectedPathExceptions, policy.ProtectedPathExceptions...))
		merged.AllowedRoots = dedupe(append(merged.AllowedRoots, policy.AllowedRoots...))
		merged.TrashOnly = merged.TrashOnly || policy.TrashOnly
		merged.AllowAdmin = merged.AllowAdmin || policy.AllowAdmin
		merged.BlockSymlinks = merged.BlockSymlinks || policy.BlockSymlinks
	}
	return merged
}

func appPathsForPolicy(items []domain.Finding) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item.Action == domain.ActionAdvisory || !filepath.IsAbs(item.Path) {
			continue
		}
		out = append(out, item.Path)
	}
	return out
}
