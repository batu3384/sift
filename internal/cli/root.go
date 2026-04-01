package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/batuhanyuksel/sift/internal/config"
	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/engine"
	"github.com/batuhanyuksel/sift/internal/store"
	"github.com/batuhanyuksel/sift/internal/tui"
)

type globalOptions struct {
	JSON            bool
	Plain           bool
	NonInteractive  bool
	Yes             bool
	ShowVersion     bool
	DryRun          bool
	Profile         string
	PlatformDebug   bool
	Admin           bool
	Force           bool
	NativeUninstall bool
}

type runtimeState struct {
	cfg     config.Config
	store   *store.Store
	service *engine.Service
	flags   globalOptions
}

type executionEnvelope struct {
	Plan   domain.ExecutionPlan   `json:"plan"`
	Result domain.ExecutionResult `json:"result"`
}

func NewRootCommand() *cobra.Command {
	state := &runtimeState{}
	root := &cobra.Command{
		Use:           "sift",
		Short:         "Safety-first terminal cleaner for macOS and Windows",
		Long:          "SIFT is a safety-first system maintenance tool. Interactive commands open the TUI by default; status, analyze, and check automatically emit JSON when stdout is piped unless --plain is set.",
		Example:       "  sift\n  sift --version\n  sift check\n  sift autofix --dry-run=false --yes\n  sift status | jq\n  sift analyze ~/Downloads | jq\n  sift touchid enable --dry-run=false --yes",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if state.flags.ShowVersion {
				return runVersionCommand(cmd.Context(), cmd, state)
			}
			if len(args) > 0 {
				return cmd.Help()
			}
			if !state.shouldUseTUI() {
				return cmd.Help()
			}
			return state.launchHome(cmd.Context(), cmd)
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg = config.Normalize(cfg)
			st, err := store.Open()
			if err != nil {
				return err
			}
			state.cfg = cfg
			state.store = st
			state.service = engine.NewService(cfg, st)
			engine.SetVersion(version)
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, _ []string) error {
			if state.store != nil {
				return state.store.Close()
			}
			return nil
		},
	}
	flags := root.PersistentFlags()
	flags.BoolVar(&state.flags.JSON, "json", false, "output JSON")
	flags.BoolVar(&state.flags.Plain, "plain", false, "plain text output")
	flags.BoolVar(&state.flags.ShowVersion, "version", false, "show version details")
	flags.BoolVar(&state.flags.NonInteractive, "non-interactive", false, "disable TUI and prompts")
	flags.BoolVar(&state.flags.Yes, "yes", false, "accept confirmation prompts")
	flags.BoolVar(&state.flags.DryRun, "dry-run", true, "preview only; do not delete")
	flags.StringVar(&state.flags.Profile, "profile", "safe", "scan profile: safe, developer, deep")
	flags.BoolVar(&state.flags.PlatformDebug, "platform-debug", false, "show platform diagnostics with results")
	flags.BoolVar(&state.flags.Admin, "admin", false, "include admin-only paths in the plan")
	flags.BoolVar(&state.flags.Force, "force", false, "permanently delete instead of using Trash/Recycle Bin")
	root.AddCommand(
		newAnalyzeCommand(state),
		newDuplicatesCommand(state),
		newLargeFilesCommand(state),
		newCheckCommand(state),
		newCleanCommand(state),
		newInstallerCommand(state),
		newPurgeCommand(state),
		newProtectCommand(state),
		newUninstallCommand(state),
		newOptimizeCommand(state),
		newAutofixCommand(state),
		newUpdateCommand(state),
		newRemoveCommand(state),
		newTouchIDCommand(state),
		newStatusCommand(state),
		newHistoryCommand(state),
		newStatsCommand(state),
		newDoctorCommand(state),
		newReportCommand(state),
		newVersionCommand(state),
		newCompletionCommand(),
	)
	return root
}

func (r *runtimeState) renderPlan(ctx context.Context, plan domain.ExecutionPlan) error {
	if r.wantsJSONOutput(plan.Command, os.Stdout) {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(plan)
	}
	if r.shouldUseTUI() {
		route := tui.RouteReview
		if plan.Command == "analyze" {
			route = tui.RouteAnalyze
		}
		return r.runInteractive(ctx, route, &plan, nil)
	}
	return printPlan(os.Stdout, plan, r.flags.PlatformDebug, r.service.Diagnostics(ctx))
}

func (r *runtimeState) runPlanFlow(ctx context.Context, plan domain.ExecutionPlan) error {
	executable := shouldExecutePlan(plan)
	if r.wantsJSONOutput(plan.Command, os.Stdout) {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if !executable {
			return encoder.Encode(plan)
		}
		if !r.flags.Yes {
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
	}
	if r.shouldUseTUI() {
		route := tui.RouteReview
		if plan.Command == "analyze" {
			route = tui.RouteAnalyze
		}
		return r.runInteractive(ctx, route, &plan, nil)
	}
	if err := r.renderPlan(ctx, plan); err != nil {
		return err
	}
	return r.maybeExecute(ctx, plan)
}

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
			// Always include admin paths for uninstall to show system applications
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
			// Use admin to properly find all app files including system apps
			return r.service.BuildUninstallPlan(ctx, app, r.flags.DryRun, true)
		},
		LoadUninstallBatchPlan: func(apps []string) (domain.ExecutionPlan, error) {
			// Use admin to properly find all app files including system apps
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
			// This callback is used by the TUI to show real-time scan progress
			// Currently a no-op since we handle progress in loading labels
		},
	})
}

func (r *runtimeState) maybeExecute(ctx context.Context, plan domain.ExecutionPlan) error {
	if !shouldExecutePlan(plan) {
		return nil
	}
	if !r.flags.Yes && r.flags.NonInteractive {
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

	// Use progress output to show each file being deleted
	progressOut := NewProgressOutput(os.Stdout)
	progressOut.SetCategoryCount(len(plan.Items))

	result, err := r.executePlanWithProgress(ctx, plan, func(p domain.ExecutionProgress) {
		switch p.Phase {
		case domain.ProgressPhaseRunning:
			// Show each file as it's being processed
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

func (r *runtimeState) persistConfig(cfg config.Config) error {
	cfg = config.Normalize(cfg)
	if err := config.Save(cfg); err != nil {
		return err
	}
	r.cfg = cfg
	r.service = engine.NewService(cfg, r.store)
	return nil
}

func parseCommandWhitelistArgs(args []string) (action string, path string, err error) {
	switch len(args) {
	case 0:
		return "list", "", nil
	case 1:
		if strings.EqualFold(args[0], "list") {
			return "list", "", nil
		}
		return "", "", fmt.Errorf("use --whitelist list, --whitelist add <path>, or --whitelist remove <path>")
	case 2:
		switch strings.ToLower(strings.TrimSpace(args[0])) {
		case "add":
			if strings.TrimSpace(args[1]) == "" {
				return "", "", fmt.Errorf("path cannot be empty")
			}
			return "add", args[1], nil
		case "remove":
			if strings.TrimSpace(args[1]) == "" {
				return "", "", fmt.Errorf("path cannot be empty")
			}
			return "remove", args[1], nil
		default:
			return "", "", fmt.Errorf("unsupported whitelist action %q", args[0])
		}
	default:
		return "", "", fmt.Errorf("use --whitelist list, --whitelist add <path>, or --whitelist remove <path>")
	}
}

func normalizeUpdateChannelFlag(value string) (engine.UpdateChannel, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(engine.UpdateChannelStable):
		return engine.UpdateChannelStable, nil
	case string(engine.UpdateChannelNightly):
		return engine.UpdateChannelNightly, nil
	default:
		return "", fmt.Errorf("unsupported update channel %q", value)
	}
}

func (r *runtimeState) handleCommandWhitelist(cmd *cobra.Command, command string, args []string) error {
	action, path, err := parseCommandWhitelistArgs(args)
	if err != nil {
		return err
	}
	command = config.NormalizeCommandName(command)
	scopes := config.Normalize(r.cfg).CommandExcludes
	switch action {
	case "list":
		if r.flags.JSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"command":          command,
				"paths":            scopes[command],
				"command_excludes": scopes,
			})
		}
		if len(scopes[command]) == 0 {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "No exclusions configured for %s.\n", command)
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s:\n", command)
		for _, item := range scopes[command] {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", item)
		}
		return nil
	case "add":
		cfg, normalizedCommand, normalizedPath, err := config.AddCommandExclude(r.cfg, command, path)
		if err != nil {
			return err
		}
		if err := r.persistConfig(cfg); err != nil {
			return err
		}
		if r.flags.JSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"command":          normalizedCommand,
				"path":             normalizedPath,
				"command_excludes": r.cfg.CommandExcludes,
			})
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s exclusion added: %s\n", normalizedCommand, normalizedPath)
		return err
	case "remove":
		cfg, normalizedCommand, normalizedPath, removed, err := config.RemoveCommandExclude(r.cfg, command, path)
		if err != nil {
			return err
		}
		if removed {
			if err := r.persistConfig(cfg); err != nil {
				return err
			}
		}
		if r.flags.JSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"command":          normalizedCommand,
				"path":             normalizedPath,
				"removed":          removed,
				"command_excludes": r.cfg.CommandExcludes,
			})
		}
		if !removed {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s exclusion not found: %s\n", normalizedCommand, normalizedPath)
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s exclusion removed: %s\n", normalizedCommand, normalizedPath)
		return err
	default:
		return fmt.Errorf("unsupported whitelist action %q", action)
	}
}

func (r *runtimeState) shouldUseTUI() bool {
	if r.flags.JSON || r.flags.Plain || r.flags.NonInteractive {
		return false
	}
	return resolveTUIEnabled(r.cfg.InteractionMode, isInteractiveTerminal())
}
