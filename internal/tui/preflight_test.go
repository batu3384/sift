package tui

import (
	"strings"
	"testing"

	"github.com/batu3384/sift/internal/domain"
)

func TestBuildPermissionPreflightCollectsAccessKinds(t *testing.T) {
	t.Parallel()

	model := buildPermissionPreflight(domain.ExecutionPlan{
		Command: "uninstall",
		Items: []domain.Finding{
			{Action: domain.ActionCommand, RequiresAdmin: true, CommandPath: "/usr/bin/sudo"},
			{Action: domain.ActionCommand, CommandPath: "/usr/bin/osascript"},
			{Action: domain.ActionNative},
		},
	}, "/tmp/example")

	if !model.required() {
		t.Fatal("expected preflight to be required")
	}
	if !model.needsAdmin || model.adminItems != 1 {
		t.Fatalf("expected one admin step, got %+v", model)
	}
	if !model.needsDialogs || model.dialogItems != 1 {
		t.Fatalf("expected one dialog step, got %+v", model)
	}
	if !model.needsNative || model.nativeItems != 1 {
		t.Fatalf("expected one native handoff, got %+v", model)
	}
	if len(model.adminLabels) != 1 || len(model.dialogLabels) != 1 || len(model.nativeLabels) != 1 {
		t.Fatalf("expected manifest labels for each access kind, got %+v", model)
	}
}

func TestPermissionPreflightViewShowsAccessAndFlowBlocks(t *testing.T) {
	t.Parallel()

	model := buildPermissionPreflight(domain.ExecutionPlan{
		Command: "uninstall",
		Targets: []string{"Example"},
		Items: []domain.Finding{
			{Action: domain.ActionCommand, RequiresAdmin: true, CommandPath: "/usr/bin/sudo"},
			{Action: domain.ActionCommand, CommandPath: "/usr/bin/osascript"},
			{Action: domain.ActionNative},
		},
	}, "/Applications/Example.app")
	model.width = 140
	model.height = 34

	view := model.View()
	for _, needle := range []string{
		"SIFT / ACCESS CHECK",
		"ACCESS RAIL",
		"MANIFEST DECK",
		"LAUNCH FLOW",
		"Removal Uninstall",
		"Removal access  •  1 admin  •  1 dialog  •  1 native",
		"Touch ID, your sudo password, or a macOS password dialog",
		"macOS may show one or more system prompts",
		"An uninstaller or external app may open outside SIFT.",
		"ADMIN",
		"DIALOGS",
		"NATIVE",
		"Lift 1   warm admin access",
		"keep access alive while execution runs",
		"native app opens outside SIFT and returns to",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in preflight view, got:\n%s", needle, view)
		}
	}
}

func TestPermissionPreflightManifestShowsItemLabels(t *testing.T) {
	t.Parallel()

	model := buildPermissionPreflight(domain.ExecutionPlan{
		Command: "uninstall",
		Items: []domain.Finding{
			{Name: "Reset LaunchServices", DisplayPath: "/usr/bin/sudo /usr/bin/true", Action: domain.ActionCommand, RequiresAdmin: true, CommandPath: "/usr/bin/sudo"},
			{Name: "Finder prompt", DisplayPath: "/usr/bin/osascript -e ...", Action: domain.ActionCommand, CommandPath: "/usr/bin/osascript"},
			{Name: "Example", DisplayPath: "Example", Action: domain.ActionNative},
		},
	}, "")

	manifest := model.manifestLines(80)
	for _, needle := range []string{"Reset LaunchServices", "Finder prompt", "Example"} {
		if !strings.Contains(manifest, needle) {
			t.Fatalf("expected %q in manifest, got:\n%s", needle, manifest)
		}
	}
}

func TestPermissionPreflightSummaryLinesStayCompact(t *testing.T) {
	t.Parallel()

	model := buildPermissionPreflight(domain.ExecutionPlan{
		Command: "clean",
		Items: []domain.Finding{
			{Name: "Reset LaunchServices", Action: domain.ActionCommand, RequiresAdmin: true, CommandPath: "/usr/bin/sudo"},
			{Name: "Finder prompt", Action: domain.ActionCommand, CommandPath: "/usr/bin/osascript"},
			{Name: "Example", Action: domain.ActionNative},
			{Name: "More", Action: domain.ActionNative},
		},
	}, "")

	if got := model.accessSummaryLine(); got != "Sweep access  •  1 admin  •  1 dialog  •  2 native" {
		t.Fatalf("unexpected access summary: %q", got)
	}
	need := model.manifestSummaryLine(120)
	for _, needle := range []string{"Need", "Reset LaunchServices", "Finder prompt", "Example", "1 more"} {
		if !strings.Contains(need, needle) {
			t.Fatalf("expected %q in manifest summary, got %q", needle, need)
		}
	}
}

func TestPermissionPreflightProfileSignatureNormalizesLabels(t *testing.T) {
	t.Parallel()

	left := buildPermissionPreflight(domain.ExecutionPlan{
		Command: "uninstall",
		Items: []domain.Finding{
			{Name: "B", Action: domain.ActionNative},
			{Name: "A", Action: domain.ActionCommand, RequiresAdmin: true, CommandPath: "/usr/bin/sudo"},
		},
	}, "")
	right := buildPermissionPreflight(domain.ExecutionPlan{
		Command: "uninstall",
		Items: []domain.Finding{
			{Name: "A", Action: domain.ActionCommand, RequiresAdmin: true, CommandPath: "/usr/bin/sudo"},
			{Name: "B", Action: domain.ActionNative},
		},
	}, "")

	if left.profileSignature() == "" {
		t.Fatal("expected non-empty signature")
	}
	if left.profileSignature() != right.profileSignature() {
		t.Fatalf("expected stable signatures, got %q vs %q", left.profileSignature(), right.profileSignature())
	}
}
