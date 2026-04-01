package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/batuhanyuksel/sift/internal/config"
	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/engine"
	"github.com/batuhanyuksel/sift/internal/platform"
	"github.com/batuhanyuksel/sift/internal/report"
	"github.com/batuhanyuksel/sift/internal/tui"
)

func newStatusCommand(state *runtimeState) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show live system status and recent scan history",
		Long:  "Status shows live system telemetry and recent scan history. When stdout is piped, it automatically switches to JSON unless --plain is set.",
		Example: "  sift status\n" +
			"  sift status --json\n" +
			"  sift status | jq",
		RunE: func(cmd *cobra.Command, args []string) error {
			writer := cmd.OutOrStdout()
			report, err := state.service.StatusReport(cmd.Context(), 10)
			if err != nil {
				return err
			}
			if state.wantsJSONOutput("status", writer) {
				return json.NewEncoder(writer).Encode(report)
			}
			if state.shouldUseTUI() {
				return state.runInteractive(cmd.Context(), tui.RouteStatus, nil, nil)
			}
			return printStatusReport(writer, report)
		},
	}
}

func newDoctorCommand(state *runtimeState) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Inspect platform, config and store health",
		Long:  "Doctor validates platform diagnostics, config integrity, store health, and parity metadata, including the upstream Mole baseline compare range.",
		Example: "  sift doctor\n" +
			"  sift doctor --json\n" +
			"  sift doctor --plain | rg 'parity_matrix|upstream_baseline'\n" +
			"  sift doctor --plain",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, err := config.WriteDefaultIfMissing()
			if err != nil {
				return err
			}
			diagnostics := state.service.Diagnostics(cmd.Context())
			configWarnings := config.Validate(state.cfg)
			diagnostics = append(diagnostics,
				platform.Diagnostic{Name: "config", Status: "ok", Message: cfgPath},
				platform.Diagnostic{Name: "store", Status: "ok", Message: state.store.Path()},
			)
			for _, warning := range configWarnings {
				diagnostics = append(diagnostics, platform.Diagnostic{Name: "config_validation", Status: "warn", Message: warning})
			}
			if state.flags.JSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(diagnostics)
			}
			if state.shouldUseTUI() {
				return state.runInteractive(cmd.Context(), tui.RouteDoctor, nil, nil)
			}
			for _, diagnostic := range diagnostics {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[%s] %-14s %s\n", strings.ToUpper(diagnostic.Status), diagnostic.Name, diagnostic.Message)
			}
			return nil
		},
	}
}

func printStatusReport(writer io.Writer, report engine.StatusReport) error {
	if report.Live != nil {
		_, _ = fmt.Fprintf(writer, "System: %s %s on %s\n", report.Live.Platform, report.Live.PlatformVersion, report.Live.Hostname)
		_, _ = fmt.Fprintf(writer, "Health %d (%s)\n", report.Live.HealthScore, report.Live.HealthLabel)
		_, _ = fmt.Fprintf(writer, "Uptime %s  Processes %d  Users %d\n",
			domain.HumanDuration(report.Live.UptimeSeconds),
			report.Live.ProcessCount,
			report.Live.LoggedInUsers,
		)
		_, _ = fmt.Fprintf(writer, "CPU %.1f%%  Memory %s / %s (%.1f%%)  Disk free %s of %s\n",
			report.Live.CPUPercent,
			domain.HumanBytes(int64(report.Live.MemoryUsedBytes)),
			domain.HumanBytes(int64(report.Live.MemoryTotalBytes)),
			report.Live.MemoryUsedPercent,
			domain.HumanBytes(int64(report.Live.DiskFreeBytes)),
			domain.HumanBytes(int64(report.Live.DiskTotalBytes)),
		)
		if report.Live.PerformanceCores > 0 || report.Live.EfficiencyCores > 0 {
			_, _ = fmt.Fprintf(writer, "CPU topology: %dP + %dE cores\n", report.Live.PerformanceCores, report.Live.EfficiencyCores)
		}
		if len(report.Live.CPUPerCore) > 0 {
			_, _ = fmt.Fprintf(writer, "Per-core CPU: %s\n", formatFloatSeries(report.Live.CPUPerCore, 8))
		}
		if report.Live.SwapTotalBytes > 0 {
			_, _ = fmt.Fprintf(writer, "Swap %s / %s (%.1f%%)\n",
				domain.HumanBytes(int64(report.Live.SwapUsedBytes)),
				domain.HumanBytes(int64(report.Live.SwapTotalBytes)),
				report.Live.SwapUsedPercent,
			)
		}
		if report.Live.DiskIO != nil {
			_, _ = fmt.Fprintf(writer, "Disk I/O read %s  write %s\n",
				domain.HumanBytes(int64(report.Live.DiskIO.ReadBytes)),
				domain.HumanBytes(int64(report.Live.DiskIO.WriteBytes)),
			)
		}
		if report.Live.Load1 > 0 {
			_, _ = fmt.Fprintf(writer, "Load %.2f (%.2fx/core)  Network RX %s  TX %s\n",
				report.Live.Load1,
				report.Live.LoadPerCPU,
				domain.HumanBytes(int64(report.Live.NetworkRxBytes)),
				domain.HumanBytes(int64(report.Live.NetworkTxBytes)),
			)
		}
		if report.Live.Battery != nil {
			_, _ = fmt.Fprintf(writer, "Battery %.0f%%  %s", report.Live.Battery.Percent, report.Live.Battery.State)
			if report.Live.Battery.RemainingMinutes > 0 {
				_, _ = fmt.Fprintf(writer, "  %d min remaining", report.Live.Battery.RemainingMinutes)
			}
			if report.Live.PowerSource != "" {
				_, _ = fmt.Fprintf(writer, "  source %s", report.Live.PowerSource)
			}
			_, _ = fmt.Fprintln(writer)
		}
		if len(report.Live.OperatorAlerts) > 0 {
			_, _ = fmt.Fprintf(writer, "Operator alerts: %s\n", strings.Join(report.Live.OperatorAlerts, " | "))
		}
		if report.Live.Proxy != nil {
			_, _ = fmt.Fprintf(writer, "Proxy enabled=%t", report.Live.Proxy.Enabled)
			if report.Live.Proxy.HTTP != "" {
				_, _ = fmt.Fprintf(writer, "  http %s", report.Live.Proxy.HTTP)
			}
			if report.Live.Proxy.HTTPS != "" {
				_, _ = fmt.Fprintf(writer, "  https %s", report.Live.Proxy.HTTPS)
			}
			if report.Live.Proxy.Bypass != "" {
				_, _ = fmt.Fprintf(writer, "  bypass %s", report.Live.Proxy.Bypass)
			}
			_, _ = fmt.Fprintln(writer)
		}
		if report.Live.VirtualizationSystem != "" {
			_, _ = fmt.Fprintf(writer, "Virtualization: %s %s\n",
				report.Live.VirtualizationSystem,
				strings.TrimSpace(report.Live.VirtualizationRole),
			)
		}
		if len(report.Live.Highlights) > 0 {
			_, _ = fmt.Fprintln(writer, "Highlights:")
			for _, highlight := range report.Live.Highlights {
				_, _ = fmt.Fprintf(writer, "  - %s\n", highlight)
			}
		}
		if len(report.Live.Warnings) > 0 {
			_, _ = fmt.Fprintf(writer, "Live warnings: %s\n", strings.Join(report.Live.Warnings, " | "))
		}
		if len(report.Live.TopProcesses) > 0 {
			_, _ = fmt.Fprintln(writer, "Top processes:")
			for _, proc := range report.Live.TopProcesses {
				_, _ = fmt.Fprintf(writer, "  %-18s cpu %5.1f%%  mem %5.1f%%  rss %s\n",
					proc.Name,
					proc.CPUPercent,
					proc.MemoryPercent,
					domain.HumanBytes(int64(proc.MemoryRSSBytes)),
				)
			}
		}
		_, _ = fmt.Fprintln(writer, "")
	}
	if len(report.RecentScans) == 0 {
		_, err := fmt.Fprintln(writer, "No scan history yet.")
		return err
	}
	for _, scan := range report.RecentScans {
		_, _ = fmt.Fprintf(writer, "%-10s %-10s %-8s %s\n", scan.Command, scan.Profile, domain.HumanBytes(scan.Totals.Bytes), scan.CreatedAt.Local().Format("2006-01-02 15:04"))
	}
	if report.PreviousScan != nil {
		_, _ = fmt.Fprintf(writer, "Delta: %s and %+d items versus previous scan\n", domain.HumanBytes(report.DeltaBytes), report.DeltaItems)
	}
	if report.LastExecution != nil {
		_, _ = fmt.Fprintf(writer, "Last execution: %d completed, %d deleted, %d failed, %d protected, %d skipped\n",
			report.LastExecution.Completed,
			report.LastExecution.Deleted,
			report.LastExecution.Failed,
			report.LastExecution.Protected,
			report.LastExecution.Skipped,
		)
		if len(report.LastExecution.Warnings) > 0 {
			_, _ = fmt.Fprintln(writer, "Follow-up:")
			for _, warning := range report.LastExecution.Warnings {
				_, _ = fmt.Fprintf(writer, "  - %s\n", warning)
			}
		}
		if len(report.LastExecution.FollowUpCommands) > 0 {
			_, _ = fmt.Fprintln(writer, "Suggested commands:")
			for _, followUp := range report.LastExecution.FollowUpCommands {
				_, _ = fmt.Fprintf(writer, "  - %s\n", followUp)
			}
		}
	}
	_, _ = fmt.Fprintf(writer, "Audit log: %s\n", report.AuditLogPath)
	return nil
}

func newReportCommand(state *runtimeState) *cobra.Command {
	return &cobra.Command{
		Use:   "report [scan-id]",
		Short: "Export a local debug bundle for a scan",
		RunE: func(cmd *cobra.Command, args []string) error {
			var scanID string
			if len(args) > 0 {
				scanID = args[0]
			} else {
				scans, err := state.service.RecentScans(cmd.Context(), 1)
				if err != nil {
					return err
				}
				if len(scans) == 0 {
					return fmt.Errorf("no stored scans available")
				}
				scanID = scans[0].ScanID
			}
			plan, err := state.store.GetPlan(cmd.Context(), scanID)
			if err != nil {
				return err
			}
			reportID, path, err := report.Bundle(cmd.Context(), state.store, plan, state.cfg)
			if err != nil {
				return err
			}
			if state.flags.JSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{
					"report_id": reportID,
					"path":      path,
				})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Report %s written to %s\n", reportID, path)
			return err
		},
	}
}

func newCompletionCommand() *cobra.Command {
	var install bool
	var completionDir string
	var shellConfig string
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate or install shell completion",
		Long:  "Completion prints shell completion to stdout by default. With --install it writes the completion file to a standard shell location and updates the matching shell profile when needed.",
		Args:  cobra.MaximumNArgs(1),
		Example: "  sift completion zsh > ~/.zfunc/_sift\n" +
			"  sift completion zsh --install\n" +
			"  sift completion --install",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			shell := ""
			if len(args) > 0 {
				shell = args[0]
			}
			if install {
				if shell == "" {
					shell = detectShellName(os.Getenv("SHELL"))
				}
				result, err := installCompletion(root, shell, completionDir, shellConfig)
				if err != nil {
					return err
				}
				if cmd.Flag("json") != nil && stateForCommandJSON(cmd) {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), result.Message)
				return err
			}
			if shell == "" {
				return fmt.Errorf("shell is required unless --install is used with an interactive shell")
			}
			switch shell {
			case "bash":
				return root.GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return root.GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return root.GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell %q", shell)
			}
		},
	}
	cmd.Flags().BoolVar(&install, "install", false, "install completion into the current shell's standard location")
	cmd.Flags().StringVar(&completionDir, "dir", "", "override the completion output directory")
	cmd.Flags().StringVar(&shellConfig, "shell-config", "", "override the shell profile path to update")
	return cmd
}

func stateForCommandJSON(cmd *cobra.Command) bool {
	flag := cmd.Flags().Lookup("json")
	return flag != nil && flag.Value.String() == "true"
}

func newHistoryCommand(state *runtimeState) *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "Show scan history and statistics",
		Long:  "History displays past scans, totals cleaned, and trends over time.",
		Example: "  sift history\n" +
			"  sift history --json\n" +
			"  sift history --weeks 4",
		RunE: func(cmd *cobra.Command, args []string) error {
			writer := cmd.OutOrStdout()
			ctx := cmd.Context()

			// Get stats
			scanStats, err := state.store.GetScanStats(ctx)
			if err != nil {
				return err
			}
			execStats, err := state.store.GetExecutionStats(ctx)
			if err != nil {
				return err
			}

			// Get recent scans
			scans, err := state.store.RecentScans(ctx, 20)
			if err != nil {
				return err
			}

			// Get weekly trend
			trend, err := state.store.GetWeeklyTrend(ctx, 7)
			if err != nil {
				return err
			}

			historyReport := map[string]interface{}{
				"scan_stats":    scanStats,
				"exec_stats":    execStats,
				"recent_scans":  scans,
				"weekly_trend":  trend,
			}

			if state.wantsJSONOutput("history", writer) {
				return json.NewEncoder(writer).Encode(historyReport)
			}

			// Print human-readable output
			_, _ = fmt.Fprintf(writer, "\n")
			_, _ = infoStyle.Fprintf(writer, "╭─────────────────────────────────────────────╮\n")
			_, _ = fmt.Fprintf(writer, "│  Sift History & Statistics                 │\n")
			_, _ = infoStyle.Fprintf(writer, "╰─────────────────────────────────────────────╯\n\n")

			// Scan stats
			_, _ = safeStyle.Fprintf(writer, "Scan Statistics\n")
			_, _ = fmt.Fprintf(writer, "  Total scans:       %d\n", scanStats.TotalScans)
			_, _ = fmt.Fprintf(writer, "  Total found:       %s\n", domain.HumanBytes(scanStats.TotalBytesFound))
			_, _ = fmt.Fprintf(writer, "  Total items:       %d\n", scanStats.TotalItemsFound)
			_, _ = fmt.Fprintf(writer, "  Average per scan:  %s\n", domain.HumanBytes(scanStats.AverageBytes))
			_, _ = fmt.Fprintf(writer, "  Largest scan:     %s\n", domain.HumanBytes(scanStats.LargestScan))
			_, _ = fmt.Fprintf(writer, "\n")

			// Execution stats
			_, _ = reviewStyle.Fprintf(writer, "Cleanup Statistics\n")
			_, _ = fmt.Fprintf(writer, "  Total cleanups:   %d\n", execStats.TotalExecutions)
			_, _ = fmt.Fprintf(writer, "  Total deleted:    %d\n", execStats.TotalDeleted)
			_, _ = fmt.Fprintf(writer, "  Total freed:      %s\n", domain.HumanBytes(execStats.TotalFreedBytes))
			_, _ = fmt.Fprintf(writer, "  Average freed:    %s\n", domain.HumanBytes(execStats.AverageFreed))
			_, _ = fmt.Fprintf(writer, "  Success rate:     %.1f%%\n", execStats.SuccessRate)
			_, _ = fmt.Fprintf(writer, "\n")

			// Recent scans
			_, _ = infoStyle.Fprintf(writer, "Recent Scans\n")
			for i, scan := range scans {
				if i >= 10 {
					break
				}
				_, _ = fmt.Fprintf(writer, "  %s  %s  %s  %s\n",
					scan.CreatedAt.Format("2006-01-02 15:04"),
					scan.Profile,
					domain.HumanBytes(scan.Totals.Bytes),
					fmt.Sprintf("(%d items)", scan.Totals.ItemCount),
				)
			}

			return nil
		},
	}
}

func newStatsCommand(state *runtimeState) *cobra.Command {
	var weeks int
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show detailed cleanup statistics and trends",
		Long:  "Stats displays detailed statistics about cleanup operations and trends over time.",
		Example: "  sift stats\n" +
			"  sift stats --json\n" +
			"  sift stats --weeks 12",
		RunE: func(cmd *cobra.Command, args []string) error {
			writer := cmd.OutOrStdout()
			ctx := cmd.Context()

			// Get stats
			scanStats, err := state.store.GetScanStats(ctx)
			if err != nil {
				return err
			}
			execStats, err := state.store.GetExecutionStats(ctx)
			if err != nil {
				return err
			}

			// Get trends
			weekly, err := state.store.GetWeeklyTrend(ctx, weeks)
			if err != nil {
				return err
			}
			monthly, err := state.store.GetMonthlyTrend(ctx, 4)
			if err != nil {
				return err
			}

			statsReport := map[string]interface{}{
				"scan_stats":   scanStats,
				"exec_stats":   execStats,
				"weekly":       weekly,
				"monthly":      monthly,
			}

			if state.wantsJSONOutput("stats", writer) {
				return json.NewEncoder(writer).Encode(statsReport)
			}

			// Print human-readable output
			_, _ = fmt.Fprintf(writer, "\n")
			_, _ = infoStyle.Fprintf(writer, "╭─────────────────────────────────────────────╮\n")
			_, _ = fmt.Fprintf(writer, "│  Sift Statistics & Trends                   │\n")
			_, _ = infoStyle.Fprintf(writer, "╰─────────────────────────────────────────────╯\n\n")

			// Summary
			_, _ = safeStyle.Fprintf(writer, "Lifetime Summary\n")
			_, _ = fmt.Fprintf(writer, "  Total freed:      %s\n", domain.HumanBytes(execStats.TotalFreedBytes))
			_, _ = fmt.Fprintf(writer, "  Items cleaned:    %d\n", execStats.TotalDeleted)
			_, _ = fmt.Fprintf(writer, "  Success rate:     %.1f%%\n", execStats.SuccessRate)
			_, _ = fmt.Fprintf(writer, "\n")

			// Profile breakdown
			if len(scanStats.ProfileBreakdown) > 0 {
				_, _ = reviewStyle.Fprintf(writer, "By Profile\n")
				for profile, bytes := range scanStats.ProfileBreakdown {
					_, _ = fmt.Fprintf(writer, "  %-12s  %s\n", profile, domain.HumanBytes(bytes))
				}
				_, _ = fmt.Fprintf(writer, "\n")
			}

			// Weekly trend
			if len(weekly) > 0 {
				_, _ = infoStyle.Fprintf(writer, "Weekly Trend (last 7 days)\n")
				for _, day := range weekly {
					d := day["date"].(string)
					b := day["bytes"].(int64)
					i := day["items"].(int)
					_, _ = fmt.Fprintf(writer, "  %s  %s  (%d items)\n", d, domain.HumanBytes(b), i)
				}
			}

			return nil
		},
	}
	cmd.Flags().IntVar(&weeks, "weeks", 4, "number of weeks to show in trend")
	return cmd
}
