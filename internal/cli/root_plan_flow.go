package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/tui"
)

func (r *runtimeState) runPlanFlow(ctx context.Context, plan domain.ExecutionPlan) error {
	executable := shouldExecutePlan(plan)
	switch r.outputModeForCommand(plan.Command, os.Stdout) {
	case outputModeJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if !executable {
			return encoder.Encode(plan)
		}
		if !r.flags.Yes && r.requiresYesFlagForExecution(plan, os.Stdout) {
			return fmt.Errorf("--yes is required when dry-run is disabled in non-interactive or JSON mode")
		}
		result, err := r.executePlan(ctx, plan)
		if err != nil {
			return err
		}
		return encoder.Encode(executionEnvelope{
			Plan:   plan,
			Result: result,
		})
	case outputModeTUI:
		route := tui.RouteReview
		if plan.Command == "analyze" {
			route = tui.RouteAnalyze
		}
		return r.runInteractive(ctx, route, &plan, nil)
	default:
		if err := printPlan(os.Stdout, plan, r.flags.PlatformDebug, r.service.Diagnostics(ctx)); err != nil {
			return err
		}
		return r.maybeExecute(ctx, plan)
	}
}

func (r *runtimeState) maybeExecute(ctx context.Context, plan domain.ExecutionPlan) error {
	if !shouldExecutePlan(plan) {
		return nil
	}
	if !r.flags.Yes && r.requiresYesFlagForExecution(plan, os.Stdout) {
		return fmt.Errorf("--yes is required when dry-run is disabled in non-interactive or JSON mode")
	}
	if !r.flags.Yes && !r.flags.NonInteractive && plan.RequiresConfirmation {
		ok, err := confirm(plan)
		if err != nil {
			return err
		}
		if !ok {
			_, err := fmt.Fprintln(os.Stdout, "Execution cancelled.")
			return err
		}
	}
	if !plan.RequiresConfirmation && !r.flags.Yes && !r.flags.NonInteractive {
		_, _ = fmt.Fprintln(os.Stdout, "Auto-approved by balanced confirmation policy.")
	}

	progressOut := NewProgressOutput(os.Stdout)
	progressOut.SetCategoryCount(len(plan.Items))

	result, err := r.executePlanWithProgress(ctx, plan, func(p domain.ExecutionProgress) {
		switch p.Phase {
		case domain.ProgressPhaseRunning:
			path := p.Item.Path
			if len(path) > 50 {
				path = path[:20] + "..." + path[len(path)-27:]
			}
			bytesStr := domain.HumanBytes(p.Item.Bytes)
			_, _ = fmt.Fprintf(os.Stdout, "  Deleting: %s  (%s)\n", path, bytesStr)
		case domain.ProgressPhaseFinished:
			if p.Result.Status == domain.StatusDeleted {
				_, _ = fmt.Fprintf(os.Stdout, "  ✓ Deleted: %s\n", p.Item.Path)
			} else if p.Result.Status == domain.StatusFailed {
				_, _ = fmt.Fprintf(os.Stdout, "  ✗ Failed: %s - %s\n", p.Item.Path, p.Result.Message)
			}
		}
	})
	if err != nil {
		return err
	}
	var freedBytes int64
	for _, item := range result.Items {
		icon := "·"
		switch item.Status {
		case domain.StatusDeleted, domain.StatusCompleted:
			icon = "✓"
			for _, pi := range plan.Items {
				if (item.FindingID != "" && pi.ID == item.FindingID) || (strings.TrimSpace(pi.Path) != "" && strings.TrimSpace(pi.Path) == strings.TrimSpace(item.Path)) {
					freedBytes += pi.Bytes
					break
				}
			}
		case domain.StatusFailed:
			icon = "✗"
		case domain.StatusProtected, domain.StatusSkipped:
			icon = "⊘"
		}
		label := strings.TrimSpace(item.Path)
		msg := strings.TrimSpace(item.Message)
		if msg != "" {
			_, _ = fmt.Fprintf(os.Stdout, "  %s  %s, %s\n", icon, label, msg)
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "  %s  %s\n", icon, label)
		}
	}
	_, _ = fmt.Fprintf(os.Stdout, "\nSpace freed: %s\n", domain.HumanBytes(freedBytes))
	if len(result.Warnings) > 0 {
		for _, warning := range result.Warnings {
			_, _ = fmt.Fprintf(os.Stdout, "  ⚠  %s\n", warning)
		}
	}
	if len(result.FollowUpCommands) > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "\nSuggested:")
		for _, followUp := range result.FollowUpCommands {
			_, _ = fmt.Fprintf(os.Stdout, "  → %s\n", followUp)
		}
	}
	return nil
}

func (r *runtimeState) executePlan(ctx context.Context, plan domain.ExecutionPlan) (domain.ExecutionResult, error) {
	releaseAdminSession, err := preparePlanExecution(ctx, plan)
	if err != nil {
		return domain.ExecutionResult{}, err
	}
	defer releaseAdminSession()
	return r.service.ExecuteWithOptions(ctx, plan, engine.ExecuteOptions{
		Permanent:       r.flags.Force,
		NativeUninstall: r.flags.NativeUninstall,
	})
}

func (r *runtimeState) executePlanWithProgress(ctx context.Context, plan domain.ExecutionPlan, emit func(domain.ExecutionProgress)) (domain.ExecutionResult, error) {
	releaseAdminSession, err := preparePlanExecution(ctx, plan)
	if err != nil {
		return domain.ExecutionResult{}, err
	}
	defer releaseAdminSession()
	return r.service.ExecuteWithProgress(ctx, plan, engine.ExecuteOptions{
		Permanent:       r.flags.Force,
		NativeUninstall: r.flags.NativeUninstall,
	}, emit)
}
