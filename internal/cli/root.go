package cli

import (
	"github.com/spf13/cobra"

	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/store"
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

func (r *runtimeState) shouldUseTUI() bool {
	if r.flags.JSON || r.flags.Plain || r.flags.NonInteractive {
		return false
	}
	return resolveTUIEnabled(r.cfg.InteractionMode, isInteractiveTerminal())
}
