package engine

import (
	"strings"
	"testing"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func TestUninstallLoginItemAliasesDedupesDisplayBundleAndName(t *testing.T) {
	app := domain.AppEntry{
		Name:        "Example",
		DisplayName: "Example",
		BundlePath:  "/Applications/Example.app",
	}

	aliases := uninstallLoginItemAliases(app)
	if len(aliases) != 1 || aliases[0] != "Example" {
		t.Fatalf("expected deduped alias list, got %v", aliases)
	}
}

func TestDarwinLoginItemScriptArgsEscapesQuotedNames(t *testing.T) {
	args := darwinLoginItemScriptArgs([]string{`Acme "Helper"`})
	joined := strings.Join(args, "\n")
	if !strings.Contains(joined, `Acme \"Helper\"`) {
		t.Fatalf("expected quoted alias to be escaped, got %v", args)
	}
	if !strings.Contains(joined, `tell application "System Events"`) {
		t.Fatalf("expected System Events login item script, got %v", args)
	}
}

func TestDarwinLaunchctlUnloadFindingsCarryUserAndSystemMetadata(t *testing.T) {
	app := domain.AppEntry{
		DisplayName: "Example",
		BundlePath:  "/Applications/Example.app",
	}

	findings := darwinLaunchctlUnloadFindings("Example", app)
	if len(findings) != 2 {
		t.Fatalf("expected user and system launchctl findings, got %v", findings)
	}
	if findings[0].TaskPhase != "aftercare" || findings[0].CommandPath != "/bin/sh" {
		t.Fatalf("unexpected user launchctl finding: %+v", findings[0])
	}
	if findings[1].TaskPhase != "secure" || !findings[1].RequiresAdmin || findings[1].CommandPath != "/usr/bin/sudo" {
		t.Fatalf("unexpected system launchctl finding: %+v", findings[1])
	}
}

func TestNativeContinuationWarningIncludesTarget(t *testing.T) {
	warning := nativeContinuationWarning(domain.ExecutionPlan{
		Command: "uninstall",
		Targets: []string{`Arc "Beta"`},
	})
	for _, needle := range []string{`Arc "Beta"`, "continued with remnant cleanup", "aftercare"} {
		if !strings.Contains(warning, needle) {
			t.Fatalf("expected %q in continuation warning, got %q", needle, warning)
		}
	}
}

func TestUninstallAftermathCommandsReturnsAdvisoryDisplayPathsOnly(t *testing.T) {
	commands := uninstallAftermathCommands(domain.ExecutionPlan{
		Command: "uninstall",
		Items: []domain.Finding{
			{Action: domain.ActionAdvisory, DisplayPath: "brew autoremove"},
			{Action: domain.ActionCommand, DisplayPath: "/usr/bin/osascript ..."},
			{Action: domain.ActionAdvisory, DisplayPath: "brew autoremove"},
			{Action: domain.ActionAdvisory, DisplayPath: "Open System Settings"},
		},
	})
	if len(commands) != 2 {
		t.Fatalf("expected two advisory commands, got %v", commands)
	}
	if commands[0] != "brew autoremove" || commands[1] != "Open System Settings" {
		t.Fatalf("unexpected advisory commands ordering/content: %v", commands)
	}
}
