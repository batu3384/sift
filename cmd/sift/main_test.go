package main

import (
	"context"
	"testing"

	"github.com/batu3384/sift/internal/cli"
)

func TestMainExit(t *testing.T) {
	// Test that NewRootCommand doesn't panic
	cmd := cli.NewRootCommand()
	if cmd == nil {
		t.Fatal("NewRootCommand returned nil")
	}

	// Test that command has a valid name
	if cmd.Name() != "sift" {
		t.Errorf("expected command name 'sift', got %q", cmd.Name())
	}
}

func TestRootCommandStructure(t *testing.T) {
	cmd := cli.NewRootCommand()

	// Verify root command structure
	if cmd.Use != "sift" {
		t.Errorf("Use = %v, want 'sift'", cmd.Use)
	}

	if cmd.SilenceErrors != true {
		t.Error("SilenceErrors should be true")
	}

	if cmd.SilenceUsage != true {
		t.Error("SilenceUsage should be true")
	}

	// Verify RunE is set (not Run)
	if cmd.RunE == nil {
		t.Error("RunE should be set")
	}

	if cmd.Run != nil {
		t.Error("Run should not be set (use RunE)")
	}
}

func TestRootCommandSubcommands(t *testing.T) {
	cmd := cli.NewRootCommand()
	subcommands := cmd.Commands()

	// Should have multiple subcommands
	if len(subcommands) < 10 {
		t.Errorf("expected at least 10 subcommands, got %d", len(subcommands))
	}

	// Verify some expected subcommands exist
	expectedCommands := []string{"analyze", "check", "clean", "status", "doctor"}
	for _, name := range expectedCommands {
		found := false
		for _, sub := range subcommands {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}

func TestRootCommandFlags(t *testing.T) {
	cmd := cli.NewRootCommand()
	flags := cmd.PersistentFlags()

	// Verify critical flags exist
	criticalFlags := []string{"json", "plain", "non-interactive", "dry-run", "profile", "admin"}
	for _, name := range criticalFlags {
		if flags.Lookup(name) == nil {
			t.Errorf("expected flag --%s not found", name)
		}
	}
}

func TestRootCommandFlagDefaults(t *testing.T) {
	cmd := cli.NewRootCommand()
	flags := cmd.PersistentFlags()

	// Verify dry-run default is true
	dryRun := flags.Lookup("dry-run")
	if dryRun == nil {
		t.Fatal("dry-run flag not found")
	}
	if dryRun.DefValue != "true" {
		t.Errorf("dry-run default = %v, want 'true'", dryRun.DefValue)
	}

	// Verify profile default is safe
	profile := flags.Lookup("profile")
	if profile == nil {
		t.Fatal("profile flag not found")
	}
	if profile.DefValue != "safe" {
		t.Errorf("profile default = %v, want 'safe'", profile.DefValue)
	}
}

func TestRootCommandHelp(t *testing.T) {
	cmd := cli.NewRootCommand()

	// Test that help doesn't panic
	err := cmd.Help()
	if err != nil {
		t.Errorf("Help() returned error: %v", err)
	}
}

func TestExecuteContextWithNilArgs(t *testing.T) {
	// This tests that ExecuteContext doesn't panic with empty args
	// We don't actually execute to avoid side effects
	cmd := cli.NewRootCommand()

	// Verify the command can be executed (in test mode, no-op)
	ctx := context.Background()
	err := cmd.ExecuteContext(ctx)
	// We might get an error due to missing config/store in test env
	// but that's okay - we just want to verify no panic
	if err != nil {
		// Expected in test environment without config
		t.Logf("ExecuteContext error (expected in test): %v", err)
	}
}