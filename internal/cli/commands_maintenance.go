package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/batu3384/sift/internal/engine"
)

func newUninstallCommand(state *runtimeState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall <app>",
		Short: "Plan or execute app bundle removal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use admin to properly find all app files including system apps
			plan, err := state.service.BuildUninstallPlan(cmd.Context(), args[0], state.flags.DryRun, true)
			if err != nil {
				return err
			}
			return state.runPlanFlow(cmd.Context(), plan)
		},
	}
	cmd.Flags().BoolVar(&state.flags.NativeUninstall, "native-uninstall", false, "launch the vendor/native uninstaller before deleting remnants")
	return cmd
}

func newOptimizeCommand(state *runtimeState) *cobra.Command {
	var whitelistMode bool
	cmd := &cobra.Command{
		Use:   "optimize",
		Short: "Plan or execute curated maintenance tasks",
		Long:  "Optimize builds a reviewed maintenance plan from curated system tasks. Keep --dry-run=true for preview, then rerun with --dry-run=false --yes to apply.",
		Example: "  sift optimize\n" +
			"  sift optimize --dry-run=false --yes\n" +
			"  sift optimize --whitelist list",
		RunE: func(cmd *cobra.Command, args []string) error {
			if whitelistMode {
				return state.handleCommandWhitelist(cmd, "optimize", args)
			}
			plan, err := state.service.BuildOptimizePlan(cmd.Context(), state.flags.DryRun, state.flags.Admin)
			if err != nil {
				return err
			}
			return state.runPlanFlow(cmd.Context(), plan)
		},
	}
	cmd.Flags().BoolVar(&whitelistMode, "whitelist", false, "manage optimize-specific exclusion paths")
	return cmd
}

func newAutofixCommand(state *runtimeState) *cobra.Command {
	return &cobra.Command{
		Use:   "autofix",
		Short: "Plan or apply safe fixes for actionable check findings",
		Long:  "Autofix turns actionable check findings into a reviewed maintenance plan. It shares the same execution safeguards as optimize.",
		Example: "  sift autofix\n" +
			"  sift autofix --dry-run=false --yes",
		RunE: func(cmd *cobra.Command, args []string) error {
			plan, err := state.service.BuildAutofixPlan(cmd.Context(), state.flags.DryRun, state.flags.Admin)
			if err != nil {
				return err
			}
			return state.runPlanFlow(cmd.Context(), plan)
		},
	}
}

func newUpdateCommand(state *runtimeState) *cobra.Command {
	var (
		updateChannel string
		nightly       bool
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Preview or apply an update for the current install method",
		Long:  "Update inspects the active install method and builds an update action for the selected channel. Applying requires --dry-run=false --yes.",
		Example: "  sift update\n" +
			"  sift update --channel nightly\n" +
			"  sift update --dry-run=false --yes\n" +
			"  sift update --force --dry-run=false --yes",
		RunE: func(cmd *cobra.Command, args []string) error {
			if nightly {
				updateChannel = string(engine.UpdateChannelNightly)
			}
			channel, err := normalizeUpdateChannelFlag(updateChannel)
			if err != nil {
				return err
			}
			if !state.flags.DryRun && !state.flags.Yes {
				return fmt.Errorf("--yes is required when applying updates")
			}
			result, err := state.service.RunUpdateWithOptions(cmd.Context(), state.flags.DryRun, engine.UpdateOptions{
				Channel: channel,
				Force:   state.flags.Force,
			})
			if err != nil {
				return err
			}
			if state.flags.JSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Version: %s\n", result.CurrentVersion)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Platform: %s\n", result.Platform)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Install method: %s\n", result.InstallMethod)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Channel: %s\n", result.Channel)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Force: %t\n", result.Force)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Dry run: %t\n", result.DryRun)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Changed: %t\n", result.Changed)
			if result.Executable != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Executable: %s\n", result.Executable)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Message: %s\n", result.Message)
			if len(result.Commands) > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Commands:")
				for _, command := range result.Commands {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", command)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&updateChannel, "channel", string(engine.UpdateChannelStable), "update channel: stable or nightly")
	cmd.Flags().BoolVar(&nightly, "nightly", false, "shortcut for --channel nightly")
	return cmd
}

func newRemoveCommand(state *runtimeState) *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Plan removal of SIFT-owned local state",
		Long:  "Remove targets SIFT-owned local state such as config, reports, and audit data. It does not uninstall the binary itself.",
		Example: "  sift remove\n" +
			"  sift remove --dry-run=false --yes",
		RunE: func(cmd *cobra.Command, args []string) error {
			plan, err := state.service.BuildRemovePlan(cmd.Context())
			if err != nil {
				return err
			}
			plan.DryRun = state.flags.DryRun
			plan.RequiresConfirmation = !state.flags.DryRun
			return state.runPlanFlow(cmd.Context(), plan)
		},
	}
}

func newTouchIDCommand(state *runtimeState) *cobra.Command {
	renderStatus := func(cmd *cobra.Command, status engine.TouchIDStatus) error {
		if state.flags.JSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Supported: %t\n", status.Supported)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Enabled: %t\n", status.Enabled)
		if status.PAMPath != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "PAM path: %s\n", status.PAMPath)
		}
		if status.LocalPAMPath != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "sudo_local path: %s\n", status.LocalPAMPath)
		}
		if status.ActivePAMPath != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Active path: %s\n", status.ActivePAMPath)
		}
		if status.BackupPath != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Backup path: %s\n", status.BackupPath)
		}
		if status.SudoLocalSupported {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "sudo_local supported: true")
		}
		if status.LegacyConfigured {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Legacy sudo entry: true")
		}
		if status.MigrationNeeded {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Migration needed: true")
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Message: %s\n", status.Message)
		if len(status.Commands) > 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Suggested commands:")
			for _, line := range status.Commands {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", line)
			}
		}
		return nil
	}
	runAction := func(enable bool) func(cmd *cobra.Command, args []string) error {
		return func(cmd *cobra.Command, args []string) error {
			if !state.flags.DryRun && !state.flags.Yes {
				return fmt.Errorf("--yes is required when applying Touch ID changes")
			}
			result, err := state.service.ConfigureTouchID(enable, state.flags.DryRun)
			if err != nil {
				return err
			}
			if state.flags.JSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Action: %s\n", result.Action)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Supported: %t\n", result.Supported)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Enabled: %t\n", result.Enabled)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Desired enabled: %t\n", result.DesiredEnabled)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Dry run: %t\n", result.DryRun)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Changed: %t\n", result.Changed)
			if result.PAMPath != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "PAM path: %s\n", result.PAMPath)
			}
			if result.LocalPAMPath != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "sudo_local path: %s\n", result.LocalPAMPath)
			}
			if result.ActivePAMPath != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Active path: %s\n", result.ActivePAMPath)
			}
			if result.BackupPath != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Backup path: %s\n", result.BackupPath)
			}
			if result.SudoLocalSupported {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "sudo_local supported: true")
			}
			if result.LegacyConfigured {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Legacy sudo entry: true")
			}
			if result.MigrationNeeded {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Migration needed: true")
			}
			if result.Migrated {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Migrated: true")
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Message: %s\n", result.Message)
			if len(result.Commands) > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Commands:")
				for _, line := range result.Commands {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", line)
				}
			}
			return nil
		}
	}
	cmd := &cobra.Command{
		Use:   "touchid",
		Short: "Inspect or manage macOS Touch ID sudo integration",
		Long:  "Touch ID inspects and manages sudo Touch ID integration, including sudo_local migration on supported macOS setups.",
		Example: "  sift touchid\n" +
			"  sift touchid enable\n" +
			"  sift touchid enable --dry-run=false --yes\n" +
			"  sift touchid disable --dry-run=false --yes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return renderStatus(cmd, state.service.TouchIDStatus())
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Preview or enable Touch ID for sudo",
		RunE:  runAction(true),
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Preview or disable Touch ID for sudo",
		RunE:  runAction(false),
	})
	return cmd
}
