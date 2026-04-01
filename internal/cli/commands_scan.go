package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/batuhanyuksel/sift/internal/config"
	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/engine"
)

func newAnalyzeCommand(state *runtimeState) *cobra.Command {
	return &cobra.Command{
		Use:   "analyze [targets...]",
		Short: "Analyze disk usage for the given path or your home directory",
		Long:  "Analyze explores large directories and files. When stdout is piped, it automatically switches to JSON unless --plain is set.",
		Example: "  sift analyze\n" +
			"  sift analyze ~/Downloads\n" +
			"  sift analyze ~/Downloads | jq",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if we should output JSON (before any output)
			isJSONMode := state.wantsJSONOutput("analyze", os.Stdout)

			// Show target being scanned (only in non-JSON mode)
			targets := args
			if len(targets) == 0 {
				targets = []string{"~"}
			}
			if !isJSONMode {
				fmt.Printf("Scanning: %s\n", strings.Join(targets, ", "))
			}

			// Create progress output for CLI feedback
			progress := NewProgressOutput(os.Stdout)
			if isJSONMode {
				progress.Disable()
			}

			// Start scan with progress output
			progress.OnScanStart("analyze")
			progress.SetCategoryCount(2) // disk_usage and large_files

			plan, err := state.service.Scan(cmd.Context(), engine.ScanOptions{
				Command:    "analyze",
				Profile:    state.flags.Profile,
				Targets:    args,
				DryRun:     true,
				AllowAdmin: state.flags.Admin,
				CategoryCallback: func(ruleID string, ruleName string, itemsFound int, bytesFound int64) {
					progress.OnCategoryScan(ruleID, ruleName, itemsFound, bytesFound)
				},
			})
			if err != nil {
				return err
			}

			// Print scan complete message
			progress.OnScanComplete(plan.Totals.Bytes, len(plan.Items))

			return state.runPlanFlow(cmd.Context(), plan)
		},
	}
}

func newCheckCommand(state *runtimeState) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Inspect actionable system posture findings",
		Long:  "Check reports actionable posture findings across security, updates, config, and health. When stdout is piped, it automatically switches to JSON unless --plain is set.",
		Example: "  sift check\n" +
			"  sift check --json\n" +
			"  sift check | jq",
		RunE: func(cmd *cobra.Command, args []string) error {
			writer := cmd.OutOrStdout()
			report, err := state.service.CheckReport(cmd.Context())
			if err != nil {
				return err
			}
			if state.wantsJSONOutput("check", writer) {
				return json.NewEncoder(writer).Encode(report)
			}
			return printCheckReport(writer, report)
		},
	}
}

func newInstallerCommand(state *runtimeState) *cobra.Command {
	return &cobra.Command{
		Use:   "installer",
		Short: "Plan or execute cleanup for stale installer files",
		RunE: func(cmd *cobra.Command, args []string) error {
			plan, err := state.service.Scan(cmd.Context(), engine.ScanOptions{
				Command:    "installer",
				RuleIDs:    []string{"installer_leftovers"},
				DryRun:     state.flags.DryRun,
				AllowAdmin: state.flags.Admin,
			})
			if err != nil {
				return err
			}
			return state.runPlanFlow(cmd.Context(), plan)
		},
	}
}

func newCleanCommand(state *runtimeState) *cobra.Command {
	var whitelistMode bool
	cmd := &cobra.Command{
		Use:   "clean [profile]",
		Short: "Plan or execute cleanup for a profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			if whitelistMode {
				return state.handleCommandWhitelist(cmd, "clean", args)
			}
			profile := state.flags.Profile
			if len(args) > 0 {
				profile = args[0]
			}

			// Create progress output for CLI feedback (like Mole) - but not in JSON mode
			progress := NewProgressOutput(os.Stdout)
			isJSONMode := state.wantsJSONOutput("clean", os.Stdout)
			if isJSONMode {
				progress.Disable()
			}

			// Start scan with progress output
			progress.OnScanStart(profile)

			// Set category count for progress calculation
			cfg := state.cfg
			progress.SetCategoryCount(config.ProfileCategoryCount(profile, cfg))

			// Check for running apps that might interfere with cleanup (like Mole)
			// Only in non-JSON mode
			if !isJSONMode {
				runningApps := []string{
					"Google Chrome",
					"Firefox",
					"Safari",
					"Microsoft Edge",
					"Brave",
					"Visual Studio Code",
					"VS Code",
					"Docker Desktop",
					"Slack",
					"Microsoft Teams",
					"Zoom",
					"Postman",
					"iTerm2",
				}
				for _, app := range runningApps {
					// Use process name for checking (lowercase for matching)
					processName := strings.ToLower(strings.ReplaceAll(app, " ", ""))
					if state.service.IsProcessRunning(processName) {
						PrintRunningAppWarning(os.Stdout, app)
					}
				}
			}

			plan, err := state.service.Scan(cmd.Context(), engine.ScanOptions{
				Command:    "clean",
				Profile:    profile,
				DryRun:     state.flags.DryRun,
				AllowAdmin: state.flags.Admin,
				CategoryCallback: func(ruleID string, ruleName string, itemsFound int, bytesFound int64) {
					progress.OnCategoryScan(ruleID, ruleName, itemsFound, bytesFound)
				},
			})
			if err != nil {
				return err
			}

			// Print scan complete message
			progress.OnScanComplete(plan.Totals.Bytes, len(plan.Items))

			return state.runPlanFlow(cmd.Context(), plan)
		},
	}
	cmd.Flags().BoolVar(&whitelistMode, "whitelist", false, "manage clean-specific exclusion paths")
	return cmd
}

func newPurgeCommand(state *runtimeState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "purge <rule-or-path>",
		Short: "Purge a specific rule or path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			scan := engine.ScanOptions{
				Command:    "purge",
				DryRun:     state.flags.DryRun,
				AllowAdmin: state.flags.Admin,
			}
			if looksLikePath(target) {
				scan.Targets = []string{target}
			} else {
				scan.RuleIDs = []string{target}
			}
			plan, err := state.service.Scan(cmd.Context(), scan)
			if err != nil {
				return err
			}
			return state.runPlanFlow(cmd.Context(), plan)
		},
	}
	cmd.AddCommand(newPurgeScanCommand(state))
	return cmd
}

func newPurgeScanCommand(state *runtimeState) *cobra.Command {
	return &cobra.Command{
		Use:   "scan [roots...]",
		Short: "Discover known project artifacts under one or more roots",
		RunE: func(cmd *cobra.Command, args []string) error {
			roots := append([]string{}, args...)
			if len(roots) == 0 {
				roots = append(roots, state.cfg.PurgeSearchPaths...)
			}
			if len(roots) == 0 {
				return fmt.Errorf("no purge search roots configured; add purge_search_paths or pass roots explicitly")
			}
			plan, err := state.service.Scan(cmd.Context(), engine.ScanOptions{
				Command:    "purge_scan",
				Targets:    roots,
				DryRun:     true,
				AllowAdmin: state.flags.Admin,
			})
			if err != nil {
				return err
			}
			return state.runPlanFlow(cmd.Context(), plan)
		},
	}
}

func newProtectCommand(state *runtimeState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "protect",
		Short: "Manage protected paths that SIFT must never delete",
	}
	explainCommand := ""
	familyCmd := &cobra.Command{
		Use:   "family",
		Short: "Manage protected data families",
	}
	scopeCmd := &cobra.Command{
		Use:   "scope",
		Short: "Manage command-scoped exclusion paths",
	}
	familyCmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List available protected families and active selections",
			RunE: func(cmd *cobra.Command, args []string) error {
				families := state.service.AvailableProtectedFamilies()
				active := map[string]struct{}{}
				for _, id := range state.cfg.ProtectedFamilies {
					active[id] = struct{}{}
				}
				if state.flags.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
						"active":   state.cfg.ProtectedFamilies,
						"families": families,
					})
				}
				for _, family := range families {
					stateLabel := "inactive"
					if _, ok := active[family.ID]; ok {
						stateLabel = "active"
					}
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-18s %-8s %s\n", family.ID, stateLabel, family.Description)
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "add <family>",
			Short: "Activate a protected data family",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, normalized, err := config.AddProtectedFamily(state.cfg, args[0])
				if err != nil {
					return err
				}
				if err := state.persistConfig(cfg); err != nil {
					return err
				}
				if state.flags.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
						"family":             normalized,
						"protected_families": state.cfg.ProtectedFamilies,
					})
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Protected family added: %s\n", normalized)
				return err
			},
		},
		&cobra.Command{
			Use:   "remove <family>",
			Short: "Deactivate a protected data family",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, normalized, removed, err := config.RemoveProtectedFamily(state.cfg, args[0])
				if err != nil {
					return err
				}
				if removed {
					if err := state.persistConfig(cfg); err != nil {
						return err
					}
				}
				if state.flags.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
						"family":             normalized,
						"removed":            removed,
						"protected_families": state.cfg.ProtectedFamilies,
					})
				}
				if !removed {
					_, err := fmt.Fprintf(cmd.OutOrStdout(), "Protected family not found: %s\n", normalized)
					return err
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Protected family removed: %s\n", normalized)
				return err
			},
		},
	)
	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List configured protected paths",
			RunE: func(cmd *cobra.Command, args []string) error {
				if state.flags.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string][]string{
						"protected_paths": state.cfg.ProtectedPaths,
					})
				}
				if len(state.cfg.ProtectedPaths) == 0 {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "No protected paths configured.")
					return err
				}
				for _, path := range state.cfg.ProtectedPaths {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), path)
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "add <path>",
			Short: "Add a protected path",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, normalized, err := config.AddProtectedPath(state.cfg, args[0])
				if err != nil {
					return err
				}
				if err := state.persistConfig(cfg); err != nil {
					return err
				}
				if state.flags.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
						"path":            normalized,
						"protected_paths": state.cfg.ProtectedPaths,
					})
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Protected path added: %s\n", normalized)
				return err
			},
		},
	)
	cmd.AddCommand(familyCmd)
	scopeCmd.AddCommand(
		&cobra.Command{
			Use:   "list [command]",
			Short: "List configured command-scoped exclusions",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				command := ""
				if len(args) == 1 {
					command = config.NormalizeCommandName(args[0])
				}
				scopes := config.Normalize(state.cfg).CommandExcludes
				if state.flags.JSON {
					payload := map[string]any{
						"command_excludes": scopes,
					}
					if command != "" {
						payload["command"] = command
						payload["paths"] = scopes[command]
					}
					return json.NewEncoder(cmd.OutOrStdout()).Encode(payload)
				}
				if len(scopes) == 0 {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "No command-scoped exclusions configured.")
					return err
				}
				if command != "" {
					paths := scopes[command]
					if len(paths) == 0 {
						_, err := fmt.Fprintf(cmd.OutOrStdout(), "No exclusions configured for %s.\n", command)
						return err
					}
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s:\n", command)
					for _, path := range paths {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", path)
					}
					return nil
				}
				for command, paths := range scopes {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s:\n", command)
					for _, path := range paths {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", path)
					}
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "add <command> <path>",
			Short: "Add a command-scoped exclusion path",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, normalizedCommand, normalizedPath, err := config.AddCommandExclude(state.cfg, args[0], args[1])
				if err != nil {
					return err
				}
				if err := state.persistConfig(cfg); err != nil {
					return err
				}
				if state.flags.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
						"command":          normalizedCommand,
						"path":             normalizedPath,
						"command_excludes": state.cfg.CommandExcludes,
					})
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Command-scoped exclusion added: %s -> %s\n", normalizedCommand, normalizedPath)
				return err
			},
		},
		&cobra.Command{
			Use:   "remove <command> <path>",
			Short: "Remove a command-scoped exclusion path",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, normalizedCommand, normalizedPath, removed, err := config.RemoveCommandExclude(state.cfg, args[0], args[1])
				if err != nil {
					return err
				}
				if removed {
					if err := state.persistConfig(cfg); err != nil {
						return err
					}
				}
				if state.flags.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
						"command":          normalizedCommand,
						"path":             normalizedPath,
						"removed":          removed,
						"command_excludes": state.cfg.CommandExcludes,
					})
				}
				if !removed {
					_, err := fmt.Fprintf(cmd.OutOrStdout(), "Command-scoped exclusion not found: %s -> %s\n", normalizedCommand, normalizedPath)
					return err
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Command-scoped exclusion removed: %s -> %s\n", normalizedCommand, normalizedPath)
				return err
			},
		},
	)
	cmd.AddCommand(scopeCmd)
	explainCmd := &cobra.Command{
		Use:   "explain <path>",
		Short: "Explain whether a path is protected by user, command or built-in policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			explanation := state.service.ExplainProtectionForCommand(args[0], explainCommand)
			if state.flags.JSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(explanation)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Path: %s\n", explanation.Path)
			if explanation.Command != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Command: %s\n", explanation.Command)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "State: %s\n", explanation.State)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Message: %s\n", explanation.Message)
			if len(explanation.CommandMatches) > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Command matches:")
				for _, match := range explanation.CommandMatches {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", match)
				}
			}
			if len(explanation.UserMatches) > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "User matches:")
				for _, match := range explanation.UserMatches {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", match)
				}
			}
			if len(explanation.SystemMatches) > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Built-in matches:")
				for _, match := range explanation.SystemMatches {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", match)
				}
			}
			if len(explanation.FamilyMatches) > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Family matches:")
				for _, match := range explanation.FamilyMatches {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", match)
				}
			}
			if len(explanation.ExceptionMatches) > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Safe exceptions:")
				for _, match := range explanation.ExceptionMatches {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", match)
				}
			}
			return nil
		},
	}
	explainCmd.Flags().StringVar(&explainCommand, "command", "", "include command-scoped exclusion evaluation")
	cmd.AddCommand(
		&cobra.Command{
			Use:   "remove <path>",
			Short: "Remove a protected path",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, normalized, removed, err := config.RemoveProtectedPath(state.cfg, args[0])
				if err != nil {
					return err
				}
				if removed {
					if err := state.persistConfig(cfg); err != nil {
						return err
					}
				}
				if state.flags.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
						"path":            normalized,
						"removed":         removed,
						"protected_paths": state.cfg.ProtectedPaths,
					})
				}
				if !removed {
					_, err := fmt.Fprintf(cmd.OutOrStdout(), "Protected path not found: %s\n", normalized)
					return err
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Protected path removed: %s\n", normalized)
				return err
			},
		},
		explainCmd,
	)
	return cmd
}

func newDuplicatesCommand(state *runtimeState) *cobra.Command {
	return &cobra.Command{
		Use:   "duplicates [path]",
		Short: "Find duplicate files in the given path",
		Long:  "Duplicates scans for files with identical content using hash comparison.",
		Example: "  sift duplicates ~/Downloads\n" +
			"  sift duplicates ~/Documents --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			targets := args
			if len(targets) == 0 {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				targets = []string{home}
			}
			plan, err := state.service.Scan(cmd.Context(), engine.ScanOptions{
				Command:    "analyze",
				Profile:    "safe",
				Targets:    targets,
				DryRun:     true,
				AllowAdmin: state.flags.Admin,
			})
			if err != nil {
				return err
			}
			// Filter to only duplicates
			var dupItems []domain.Finding
			for _, item := range plan.Items {
				if item.RuleID == "analyze.duplicates" {
					dupItems = append(dupItems, item)
				}
			}
			plan.Items = dupItems
			plan.Totals.Bytes = 0
			plan.Totals.ItemCount = len(dupItems)
			for _, item := range dupItems {
				plan.Totals.Bytes += item.Bytes
			}
			return state.runPlanFlow(cmd.Context(), plan)
		},
	}
}

func newLargeFilesCommand(state *runtimeState) *cobra.Command {
	var minSize string
	cmd := &cobra.Command{
		Use:   "largefiles [path]",
		Short: "Find large files in the given path",
		Long:  "largefiles scans for the largest files under the given path.",
		Example: "  sift largefiles ~/Downloads\n" +
			"  sift largefiles ~/ --min-size 100MB\n" +
			"  sift largefiles ~/Documents --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			targets := args
			if len(targets) == 0 {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				targets = []string{home}
			}
			plan, err := state.service.Scan(cmd.Context(), engine.ScanOptions{
				Command:    "analyze",
				Profile:    "safe",
				Targets:    targets,
				DryRun:     true,
				AllowAdmin: state.flags.Admin,
			})
			if err != nil {
				return err
			}
			// Filter to only large files
			var largeItems []domain.Finding
			for _, item := range plan.Items {
				if item.RuleID == "analyze.large_files" {
					largeItems = append(largeItems, item)
				}
			}
			plan.Items = largeItems
			plan.Totals.Bytes = 0
			plan.Totals.ItemCount = len(largeItems)
			for _, item := range largeItems {
				plan.Totals.Bytes += item.Bytes
			}
			return state.runPlanFlow(cmd.Context(), plan)
		},
	}
	cmd.Flags().StringVar(&minSize, "min-size", "10MB", "minimum file size (e.g., 100MB, 1GB)")
	return cmd
}
