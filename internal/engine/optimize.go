package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/batu3384/sift/internal/domain"
)

func (s *Service) BuildOptimizePlan(ctx context.Context, dryRun, allowAdmin bool) (domain.ExecutionPlan, error) {
	tasks := s.MaintenanceTasks(ctx)
	report, _ := s.CheckReport(ctx)
	activeChecks := optimizeActiveCheckMap(report)
	sort.SliceStable(tasks, func(i, j int) bool {
		return optimizeTaskPriority(tasks[i], activeChecks) < optimizeTaskPriority(tasks[j], activeChecks)
	})
	items := make([]domain.Finding, 0, len(tasks))
	var warnings []string
	var allowedRoots []string
	for _, task := range tasks {
		action := task.Action
		if action == "" {
			action = domain.ActionAdvisory
		}
		if action == domain.ActionCommand && strings.TrimSpace(task.CommandPath) != "" {
			items = append(items, commandMaintenanceFinding(task, activeChecks))
			continue
		}
		targetPaths, taskWarnings := expandMaintenanceTaskTargets(task)
		warnings = append(warnings, taskWarnings...)
		if action != domain.ActionAdvisory && len(targetPaths) > 0 {
			actionable, actionableWarnings := s.buildOptimizeActionableFindings(ctx, task, action, targetPaths, activeChecks)
			items = append(items, actionable...)
			warnings = append(warnings, actionableWarnings...)
			allowedRoots = append(allowedRoots, appPathsForPolicy(actionable)...)
			continue
		}
		items = append(items, advisoryMaintenanceFinding(task, activeChecks))
	}
	if summary := optimizePreflightSummary(report, activeChecks); summary != "" {
		warnings = append([]string{summary}, warnings...)
	}
	policy := s.buildPolicy(ScanOptions{
		Command:    "optimize",
		DryRun:     dryRun,
		AllowAdmin: allowAdmin,
	}, nil, allowedRoots)
	for idx := range items {
		items[idx] = applyPolicy(items[idx], evaluatePolicy(policy, items[idx], false))
	}
	plan := domain.ExecutionPlan{
		ScanID:               uuid.NewString(),
		Command:              "optimize",
		Platform:             s.Adapter.Name(),
		CreatedAt:            time.Now().UTC(),
		PlanState:            "preview",
		DryRun:               dryRun,
		RequiresConfirmation: s.requiresConfirmation("optimize", dryRun, items),
		Warnings:             dedupe(warnings),
		Items:                items,
		Totals:               calculateTotals(items),
		Policy:               policy,
	}
	if len(items) == 0 {
		plan.PlanState = "empty"
	}
	s.persistPlan(ctx, &plan)
	return plan, nil
}

func advisoryMaintenanceFinding(task domain.MaintenanceTask, activeChecks map[string]domain.CheckItem) domain.Finding {
	steps := strings.Join(task.Steps, " | ")
	suggestedBy := optimizeSuggestedBy(task, activeChecks)
	return domain.Finding{
		ID:          uuid.NewString(),
		RuleID:      task.ID,
		Name:        task.Title,
		Category:    domain.CategoryMaintenance,
		Path:        task.ID,
		DisplayPath: task.Title,
		Risk:        task.Risk,
		Action:      domain.ActionAdvisory,
		Status:      domain.StatusAdvisory,
		Recovery: domain.RecoveryHint{
			Message:  steps,
			Location: "manual maintenance",
		},
		Capability:  task.Capability,
		TaskPhase:   task.Phase,
		TaskImpact:  task.EstimatedImpact,
		TaskVerify:  append([]string{}, task.Verification...),
		SuggestedBy: suggestedBy,
		Source:      task.Description,
	}
}

func commandMaintenanceFinding(task domain.MaintenanceTask, activeChecks map[string]domain.CheckItem) domain.Finding {
	steps := strings.Join(task.Steps, " | ")
	display := strings.TrimSpace(strings.Join(append([]string{task.CommandPath}, task.CommandArgs...), " "))
	suggestedBy := optimizeSuggestedBy(task, activeChecks)
	return domain.Finding{
		ID:             uuid.NewString(),
		RuleID:         task.ID,
		Name:           task.Title,
		Category:       domain.CategoryMaintenance,
		Path:           task.ID,
		DisplayPath:    display,
		Risk:           task.Risk,
		Action:         domain.ActionCommand,
		Status:         domain.StatusPlanned,
		CommandPath:    task.CommandPath,
		CommandArgs:    append([]string{}, task.CommandArgs...),
		TimeoutSeconds: task.TimeoutSeconds,
		Capability:     task.Capability,
		TaskPhase:      task.Phase,
		TaskImpact:     task.EstimatedImpact,
		TaskVerify:     append([]string{}, task.Verification...),
		SuggestedBy:    suggestedBy,
		Recovery: domain.RecoveryHint{
			Message:  steps,
			Location: "managed command",
		},
		RequiresAdmin: task.RequiresAdmin,
		Source:        task.Description,
	}
}

func expandMaintenanceTaskTargets(task domain.MaintenanceTask) ([]string, []string) {
	paths := dedupePaths(task.Paths)
	var warnings []string
	for _, pattern := range task.PathGlobs {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: invalid optimize glob %q: %v", task.Title, pattern, err))
			continue
		}
		paths = append(paths, matches...)
	}
	return dedupePaths(paths), warnings
}

func optimizeActiveCheckMap(report domain.CheckReport) map[string]domain.CheckItem {
	items := map[string]domain.CheckItem{}
	for _, item := range report.Items {
		if item.Status != "warn" {
			continue
		}
		items[item.ID] = item
	}
	return items
}

func optimizeSuggestedBy(task domain.MaintenanceTask, activeChecks map[string]domain.CheckItem) []string {
	var suggested []string
	for _, checkID := range task.SuggestedByChecks {
		if item, ok := activeChecks[checkID]; ok {
			suggested = append(suggested, item.Name)
		}
	}
	sort.Strings(suggested)
	return suggested
}

func optimizeTaskPriority(task domain.MaintenanceTask, activeChecks map[string]domain.CheckItem) int {
	priority := 100
	if len(optimizeSuggestedBy(task, activeChecks)) > 0 {
		priority -= 50
	}
	switch strings.ToLower(strings.TrimSpace(task.Phase)) {
	case "preflight":
		priority -= 30
	case "repair":
		priority -= 20
	case "cleanup":
		priority -= 10
	case "verify":
		priority += 10
	}
	switch task.Risk {
	case domain.RiskSafe:
		priority -= 5
	case domain.RiskHigh:
		priority += 5
	}
	return priority
}

func optimizePreflightSummary(report domain.CheckReport, activeChecks map[string]domain.CheckItem) string {
	if len(activeChecks) == 0 {
		return ""
	}
	names := make([]string, 0, len(activeChecks))
	for _, item := range activeChecks {
		names = append(names, item.Name)
	}
	sort.Strings(names)
	if len(names) > 4 {
		names = append(names[:4], fmt.Sprintf("+%d more", len(activeChecks)-4))
	}
	n := len(activeChecks)
	checkWord := map[bool]string{true: "check", false: "checks"}[n == 1]
	return fmt.Sprintf("Preflight found %d actionable %s: %s.", n, checkWord, strings.Join(names, ", "))
}

func (s *Service) buildOptimizeActionableFindings(ctx context.Context, task domain.MaintenanceTask, action domain.Action, paths []string, activeChecks map[string]domain.CheckItem) ([]domain.Finding, []string) {
	items := make([]domain.Finding, 0, len(paths))
	var warnings []string
	recoveryMessage := coalesce(strings.Join(task.Steps, " | "), task.Description)
	suggestedBy := optimizeSuggestedBy(task, activeChecks)
	for _, candidate := range paths {
		normalized := domain.NormalizePath(candidate)
		if normalized == "" {
			continue
		}
		info, err := os.Lstat(normalized)
		if err != nil {
			if !os.IsNotExist(err) {
				warnings = append(warnings, fmt.Sprintf("%s: %s: %v", task.Title, normalized, err))
			}
			continue
		}
		size, newest, err := rulesMeasurePath(ctx, normalized)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %s: %v", task.Title, normalized, err))
			continue
		}
		status := domain.StatusPlanned
		if action == domain.ActionAdvisory {
			status = domain.StatusAdvisory
		}
		items = append(items, domain.Finding{
			ID:            uuid.NewString(),
			RuleID:        task.ID,
			Name:          coalesce(filepath.Base(normalized), task.Title),
			Category:      domain.CategoryMaintenance,
			Path:          normalized,
			DisplayPath:   normalized,
			Risk:          task.Risk,
			Bytes:         size,
			RequiresAdmin: task.RequiresAdmin || s.Adapter.IsAdminPath(normalized),
			Action:        action,
			Recovery: domain.RecoveryHint{
				Message:  recoveryMessage,
				Location: "system trash",
			},
			Status:       status,
			LastModified: newest,
			Capability:   task.Capability,
			TaskPhase:    task.Phase,
			TaskImpact:   task.EstimatedImpact,
			TaskVerify:   append([]string{}, task.Verification...),
			SuggestedBy:  suggestedBy,
			Fingerprint: domain.Fingerprint{
				Mode:    uint32(info.Mode()),
				Size:    size,
				ModTime: newest,
			},
			Source: task.Title,
		})
	}
	return items, warnings
}
