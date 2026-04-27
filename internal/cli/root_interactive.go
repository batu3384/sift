package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/tui"
)

func (r *runtimeState) launchHome(ctx context.Context, _ *cobra.Command) error {
	return r.runInteractive(ctx, tui.RouteHome, nil, nil)
}

func (r *runtimeState) runInteractive(ctx context.Context, route tui.Route, plan *domain.ExecutionPlan, result *domain.ExecutionResult) error {
	return tui.RunApp(tui.AppOptions{
		Config:        r.cfg,
		Executable:    !r.flags.DryRun,
		InitialRoute:  route,
		InitialPlan:   plan,
		InitialResult: result,
		ReducedMotion: tui.ReducedMotionEnabled(),
	}, tui.AppCallbacks{
		LoadDashboard: func() (tui.DashboardData, error) {
			report, err := r.service.StatusReport(ctx, 10)
			if err != nil {
				return tui.DashboardData{}, err
			}
			update := r.service.UpdateNotice(ctx)
			return tui.DashboardData{
				Report:      report,
				Diagnostics: r.service.Diagnostics(ctx),
				Update:      &update,
			}, nil
		},
		LoadCachedInstalledApps: func() ([]domain.AppEntry, time.Time, error) {
			return r.service.CachedApps(ctx)
		},
		LoadInstalledApps: func() ([]domain.AppEntry, error) {
			return r.service.ListApps(ctx, true)
		},
		LoadAnalyzeHome: func() (domain.ExecutionPlan, error) {
			return r.service.Scan(ctx, engine.ScanOptions{
				Command:    "analyze",
				Profile:    r.flags.Profile,
				DryRun:     true,
				AllowAdmin: r.flags.Admin,
			})
		},
		LoadAnalyzeTarget: func(target string) (domain.ExecutionPlan, error) {
			return r.service.Scan(ctx, engine.ScanOptions{
				Command:    "analyze",
				Profile:    r.flags.Profile,
				Targets:    []string{target},
				DryRun:     true,
				AllowAdmin: r.flags.Admin,
			})
		},
		LoadAnalyzePreviews: func(paths []string) map[string]domain.DirectoryPreview {
			return r.service.AnalyzePreviews(paths)
		},
		LoadCleanProfileWithProgress: func(profile string, emit func(ruleID string, ruleName string, itemsFound int, bytesFound int64)) (domain.ExecutionPlan, error) {
			return r.service.Scan(ctx, engine.ScanOptions{
				Command:          "clean",
				Profile:          profile,
				DryRun:           r.flags.DryRun,
				AllowAdmin:       r.flags.Admin,
				CategoryCallback: emit,
			})
		},
		LoadCleanProfileWithFindingProgress: func(profile string, emit func(ruleID string, ruleName string, item domain.Finding)) (domain.ExecutionPlan, error) {
			return r.service.Scan(ctx, engine.ScanOptions{
				Command:         "clean",
				Profile:         profile,
				DryRun:          r.flags.DryRun,
				AllowAdmin:      r.flags.Admin,
				FindingCallback: emit,
			})
		},
		LoadCleanProfile: func(profile string) (domain.ExecutionPlan, error) {
			return r.service.Scan(ctx, engine.ScanOptions{
				Command:    "clean",
				Profile:    profile,
				DryRun:     r.flags.DryRun,
				AllowAdmin: r.flags.Admin,
			})
		},
		LoadInstaller: func() (domain.ExecutionPlan, error) {
			return r.service.Scan(ctx, engine.ScanOptions{
				Command:    "installer",
				RuleIDs:    []string{"installer_leftovers"},
				DryRun:     r.flags.DryRun,
				AllowAdmin: r.flags.Admin,
			})
		},
		LoadOptimize: func() (domain.ExecutionPlan, error) {
			return r.service.BuildOptimizePlan(ctx, r.flags.DryRun, r.flags.Admin)
		},
		LoadAutofix: func() (domain.ExecutionPlan, error) {
			return r.service.BuildAutofixPlan(ctx, r.flags.DryRun, r.flags.Admin)
		},
		LoadPurgeScan: func() (domain.ExecutionPlan, error) {
			if len(r.cfg.PurgeSearchPaths) == 0 {
				return domain.ExecutionPlan{}, fmt.Errorf("no purge search roots configured; add purge_search_paths first")
			}
			return r.service.Scan(ctx, engine.ScanOptions{
				Command:    "purge_scan",
				Targets:    r.cfg.PurgeSearchPaths,
				DryRun:     true,
				AllowAdmin: r.flags.Admin,
			})
		},
		LoadUninstallPlan: func(app string) (domain.ExecutionPlan, error) {
			return r.service.BuildUninstallPlan(ctx, app, r.flags.DryRun, true)
		},
		LoadUninstallBatchPlan: func(apps []string) (domain.ExecutionPlan, error) {
			return r.service.BuildBatchUninstallPlan(ctx, apps, r.flags.DryRun, true)
		},
		LoadReviewForPaths: func(paths []string) (domain.ExecutionPlan, error) {
			return r.service.Scan(ctx, engine.ScanOptions{
				Command:    "clean",
				Targets:    paths,
				DryRun:     false,
				AllowAdmin: r.flags.Admin,
			})
		},
		AddProtectedPath: func(path string) (config.Config, string, error) {
			cfg, normalized, err := config.AddProtectedPath(r.cfg, path)
			if err != nil {
				return config.Config{}, "", err
			}
			if err := r.persistConfig(cfg); err != nil {
				return config.Config{}, "", err
			}
			return r.cfg, normalized, nil
		},
		RemoveProtectedPath: func(path string) (config.Config, string, bool, error) {
			cfg, normalized, removed, err := config.RemoveProtectedPath(r.cfg, path)
			if err != nil {
				return config.Config{}, "", false, err
			}
			if removed {
				if err := r.persistConfig(cfg); err != nil {
					return config.Config{}, "", false, err
				}
			}
			return r.cfg, normalized, removed, nil
		},
		ExplainProtection: func(path string) domain.ProtectionExplanation {
			return r.service.ExplainProtection(path)
		},
		ExecutePlan: func(plan domain.ExecutionPlan) (domain.ExecutionResult, error) {
			return r.executePlan(ctx, plan)
		},
		ExecutePlanWithProgress: func(execCtx context.Context, plan domain.ExecutionPlan, emit func(domain.ExecutionProgress)) (domain.ExecutionResult, error) {
			return r.executePlanWithProgress(execCtx, plan, emit)
		},
		OpenPath:   tui.OpenPath,
		RevealPath: tui.RevealPath,
		TrashPaths: func(paths []string) (domain.ExecutionResult, error) {
			return r.service.TrashPaths(ctx, "analyze", paths, r.flags.Admin)
		},
		OnScanProgress: func(ruleID string, ruleName string, itemsFound int, bytesFound int64) {
		},
	})
}
