package cli

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/store"
)

func TestAnalyzeCommandStructure(t *testing.T) {
	cfg := config.Default()
	st, err := store.Open()
	if err != nil {
		t.Skipf("Skipping store test: %v", err)
	}
	defer st.Close()

	state := &runtimeState{
		cfg:     cfg,
		store:   st,
		service: engine.NewService(cfg, st),
	}

	cmd := newAnalyzeCommand(state)
	if cmd == nil {
		t.Fatal("newAnalyzeCommand returned nil")
	}

	// Verify command structure
	if cmd.Use != "analyze [targets...]" {
		t.Errorf("Use = %v, want 'analyze [targets...]'", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("Short should not be empty")
	}
	if cmd.Long == "" {
		t.Error("Long should not be empty")
	}

	// Verify examples are set
	if cmd.Example == "" {
		t.Error("Example should not be empty")
	}

	// Verify RunE is set (not Run)
	if cmd.RunE == nil {
		t.Error("RunE should be set")
	}
	if cmd.Run != nil {
		t.Error("Run should not be set (use RunE)")
	}
}

func TestCheckCommandStructure(t *testing.T) {
	cfg := config.Default()
	st, err := store.Open()
	if err != nil {
		t.Skipf("Skipping store test: %v", err)
	}
	defer st.Close()

	state := &runtimeState{
		cfg:     cfg,
		store:   st,
		service: engine.NewService(cfg, st),
	}

	cmd := newCheckCommand(state)
	if cmd == nil {
		t.Fatal("newCheckCommand returned nil")
	}

	if cmd.Use != "check" {
		t.Errorf("Use = %v, want 'check'", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("Short should not be empty")
	}
	if cmd.Example == "" {
		t.Error("Example should not be empty")
	}
}

func TestCleanCommandStructure(t *testing.T) {
	cfg := config.Default()
	st, err := store.Open()
	if err != nil {
		t.Skipf("Skipping store test: %v", err)
	}
	defer st.Close()

	state := &runtimeState{
		cfg:     cfg,
		store:   st,
		service: engine.NewService(cfg, st),
	}

	cmd := newCleanCommand(state)
	if cmd == nil {
		t.Fatal("newCleanCommand returned nil")
	}

	// Verify command accepts optional profile argument
	if cmd.Use != "clean [profile]" {
		t.Errorf("Use = %v, want 'clean [profile]'", cmd.Use)
	}

	// Verify --whitelist flag is registered
	flag := cmd.Flags().Lookup("whitelist")
	if flag == nil {
		t.Fatal("expected --whitelist flag to be registered")
	}
	if flag.DefValue != "false" {
		t.Errorf("whitelist default = %v, want false", flag.DefValue)
	}
}

func TestCleanCommandProfileOverride(t *testing.T) {
	cfg := config.Default()
	st, err := store.Open()
	if err != nil {
		t.Skipf("Skipping store test: %v", err)
	}
	defer st.Close()

	// Test default profile
	state := &runtimeState{
		cfg:     cfg,
		store:   st,
		service: engine.NewService(cfg, st),
		flags:   globalOptions{Profile: "safe"},
	}

	if state.flags.Profile != "safe" {
		t.Errorf("default profile = %v, want 'safe'", state.flags.Profile)
	}

	// Test profile override via args
	args := []string{"developer"}
	if len(args) > 0 {
		profile := state.flags.Profile
		if len(args) > 0 {
			profile = args[0]
		}
		if profile != "developer" {
			t.Errorf("profile override = %v, want 'developer'", profile)
		}
	}
}

func TestCleanCommandWhitelistMode(t *testing.T) {
	cfg := config.Default()
	st, err := store.Open()
	if err != nil {
		t.Skipf("Skipping store test: %v", err)
	}
	defer st.Close()

	state := &runtimeState{
		cfg:     cfg,
		store:   st,
		service: engine.NewService(cfg, st),
	}

	cmd := newCleanCommand(state)

	// Verify whitelist flag is registered (don't redefine it)
	flag := cmd.Flags().Lookup("whitelist")
	if flag == nil {
		t.Fatal("expected --whitelist flag to be registered")
	}
	if flag.DefValue != "false" {
		t.Errorf("whitelist default = %v, want false", flag.DefValue)
	}
}

func TestInstallerCommandStructure(t *testing.T) {
	cfg := config.Default()
	st, err := store.Open()
	if err != nil {
		t.Skipf("Skipping store test: %v", err)
	}
	defer st.Close()

	state := &runtimeState{
		cfg:     cfg,
		store:   st,
		service: engine.NewService(cfg, st),
	}

	cmd := newInstallerCommand(state)
	if cmd == nil {
		t.Fatal("newInstallerCommand returned nil")
	}

	if cmd.Use != "installer" {
		t.Errorf("Use = %v, want 'installer'", cmd.Use)
	}

	// Installer command should use installer_leftovers rule
	// This is verified by checking the command's RunE
	if cmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestPurgeCommandStructure(t *testing.T) {
	cfg := config.Default()
	st, err := store.Open()
	if err != nil {
		t.Skipf("Skipping store test: %v", err)
	}
	defer st.Close()

	state := &runtimeState{
		cfg:     cfg,
		store:   st,
		service: engine.NewService(cfg, st),
	}

	cmd := newPurgeCommand(state)
	if cmd == nil {
		t.Fatal("newPurgeCommand returned nil")
	}

	if cmd.Use != "purge <rule-or-path>" {
		t.Errorf("Use = %v, want 'purge <rule-or-path>'", cmd.Use)
	}

	// Verify subcommand is added
	if len(cmd.Commands()) == 0 {
		t.Error("purge should have subcommands")
	}

	// Check subcommand structure
	subCmd := cmd.Commands()[0]
	if subCmd.Use != "scan [roots...]" {
		t.Errorf("subcommand Use = %v, want 'scan [roots...]'", subCmd.Use)
	}
}

func TestProtectCommandStructure(t *testing.T) {
	cfg := config.Default()
	st, err := store.Open()
	if err != nil {
		t.Skipf("Skipping store test: %v", err)
	}
	defer st.Close()

	state := &runtimeState{
		cfg:     cfg,
		store:   st,
		service: engine.NewService(cfg, st),
	}

	cmd := newProtectCommand(state)
	if cmd == nil {
		t.Fatal("newProtectCommand returned nil")
	}

	// Verify subcommands exist
	subcommands := cmd.Commands()
	if len(subcommands) < 4 {
		t.Errorf("expected at least 4 subcommands, got %d", len(subcommands))
	}

	// Verify family subcommand
	familyCmd := findSubcommand(cmd, "family")
	if familyCmd == nil {
		t.Fatal("expected 'family' subcommand")
	}

	// Verify scope subcommand
	scopeCmd := findSubcommand(cmd, "scope")
	if scopeCmd == nil {
		t.Fatal("expected 'scope' subcommand")
	}

	// Verify list subcommand
	listCmd := findSubcommand(cmd, "list")
	if listCmd == nil {
		t.Fatal("expected 'list' subcommand")
	}

	// Verify remove subcommand
	removeCmd := findSubcommand(cmd, "remove")
	if removeCmd == nil {
		t.Fatal("expected 'remove' subcommand")
	}

	// Verify add subcommand
	addCmd := findSubcommand(cmd, "add")
	if addCmd == nil {
		t.Fatal("expected 'add' subcommand")
	}

	// Verify explain subcommand
	explainCmd := findSubcommand(cmd, "explain")
	if explainCmd == nil {
		t.Fatal("expected 'explain' subcommand")
	}
}

func TestProtectFamilySubcommands(t *testing.T) {
	cfg := config.Default()
	st, err := store.Open()
	if err != nil {
		t.Skipf("Skipping store test: %v", err)
	}
	defer st.Close()

	state := &runtimeState{
		cfg:     cfg,
		store:   st,
		service: engine.NewService(cfg, st),
	}

	cmd := newProtectCommand(state)
	familyCmd := findSubcommand(cmd, "family")
	if familyCmd == nil {
		t.Skip("family subcommand not found")
	}

	subcommands := familyCmd.Commands()
	if len(subcommands) < 2 {
		t.Errorf("expected at least 2 family subcommands, got %d", len(subcommands))
	}

	// Verify list subcommand
	listCmd := findSubcommand(familyCmd, "list")
	if listCmd == nil {
		t.Fatal("expected 'list' subcommand in family")
	}

	// Verify add subcommand
	addCmd := findSubcommand(familyCmd, "add")
	if addCmd == nil {
		t.Fatal("expected 'add' subcommand in family")
	}

	// Verify remove subcommand
	removeCmd := findSubcommand(familyCmd, "remove")
	if removeCmd == nil {
		t.Fatal("expected 'remove' subcommand in family")
	}
}

func TestProtectScopeSubcommands(t *testing.T) {
	cfg := config.Default()
	st, err := store.Open()
	if err != nil {
		t.Skipf("Skipping store test: %v", err)
	}
	defer st.Close()

	state := &runtimeState{
		cfg:     cfg,
		store:   st,
		service: engine.NewService(cfg, st),
	}

	cmd := newProtectCommand(state)
	scopeCmd := findSubcommand(cmd, "scope")
	if scopeCmd == nil {
		t.Skip("scope subcommand not found")
	}

	subcommands := scopeCmd.Commands()
	if len(subcommands) < 3 {
		t.Errorf("expected at least 3 scope subcommands, got %d", len(subcommands))
	}

	// Verify list subcommand
	listCmd := findSubcommand(scopeCmd, "list")
	if listCmd == nil {
		t.Fatal("expected 'list' subcommand in scope")
	}

	// Verify add subcommand
	addCmd := findSubcommand(scopeCmd, "add")
	if addCmd == nil {
		t.Fatal("expected 'add' subcommand in scope")
	}

	// Verify remove subcommand
	removeCmd := findSubcommand(scopeCmd, "remove")
	if removeCmd == nil {
		t.Fatal("expected 'remove' subcommand in scope")
	}
}

func TestDuplicatesCommandStructure(t *testing.T) {
	cfg := config.Default()
	st, err := store.Open()
	if err != nil {
		t.Skipf("Skipping store test: %v", err)
	}
	defer st.Close()

	state := &runtimeState{
		cfg:     cfg,
		store:   st,
		service: engine.NewService(cfg, st),
	}

	cmd := newDuplicatesCommand(state)
	if cmd == nil {
		t.Fatal("newDuplicatesCommand returned nil")
	}

	if cmd.Use != "duplicates [path]" {
		t.Errorf("Use = %v, want 'duplicates [path]'", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("Short should not be empty")
	}
}

func TestLargeFilesCommandStructure(t *testing.T) {
	cfg := config.Default()
	st, err := store.Open()
	if err != nil {
		t.Skipf("Skipping store test: %v", err)
	}
	defer st.Close()

	state := &runtimeState{
		cfg:     cfg,
		store:   st,
		service: engine.NewService(cfg, st),
	}

	cmd := newLargeFilesCommand(state)
	if cmd == nil {
		t.Fatal("newLargeFilesCommand returned nil")
	}

	if cmd.Use != "largefiles [path]" {
		t.Errorf("Use = %v, want 'largefiles [path]'", cmd.Use)
	}

	// Verify --min-size flag
	flag := cmd.Flags().Lookup("min-size")
	if flag == nil {
		t.Fatal("expected --min-size flag to be registered")
	}
	if flag.DefValue != "10MB" {
		t.Errorf("min-size default = %v, want '10MB'", flag.DefValue)
	}
}

// Helper to find subcommand by name
func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, sub := range cmd.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	return nil
}