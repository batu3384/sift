package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompletionTargetForShellUsesExpectedPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	zshTarget, err := completionTargetForShell("zsh", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if zshTarget.completionPath != filepath.Join(home, ".zfunc", "_sift") {
		t.Fatalf("unexpected zsh completion path: %s", zshTarget.completionPath)
	}
	if zshTarget.configPath != filepath.Join(home, ".zshrc") || !strings.Contains(zshTarget.sourceBlock, "compinit") {
		t.Fatalf("unexpected zsh config target: %+v", zshTarget)
	}

	bashTarget, err := completionTargetForShell("bash", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if bashTarget.completionPath != filepath.Join(home, ".local", "share", "bash-completion", "completions", "sift") {
		t.Fatalf("unexpected bash completion path: %s", bashTarget.completionPath)
	}
}

func TestEnsureConfigBlockIsIdempotent(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".zshrc")
	block := "source \"/tmp/sift\""
	if err := ensureConfigBlock(path, block); err != nil {
		t.Fatal(err)
	}
	if err := ensureConfigBlock(path, block); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Count(text, "# >>> sift completion >>>") != 1 {
		t.Fatalf("expected a single completion block, got %s", text)
	}
}

func TestPrintVersionInfoIncludesInstallMethodAndShell(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := printVersionInfo(&out, versionInfo{
		Version:         "v1.2.3",
		Platform:        "darwin 15.3",
		Arch:            "arm64",
		Executable:      "/opt/homebrew/bin/sift",
		InstallMethod:   "homebrew",
		Channel:         "stable",
		Shell:           "zsh",
		InteractionMode: "auto",
		DiskFree:        "42.0 GB",
		SIP:             "enabled",
		UpdateMessage:   "Running latest release (v1.2.3).",
	})
	if err != nil {
		t.Fatal(err)
	}
	view := out.String()
	for _, needle := range []string{"SIFT v1.2.3", "Install method: homebrew  Channel: stable", "Shell: zsh  Interaction: auto", "Disk free: 42.0 GB  SIP: enabled"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in version output, got %s", needle, view)
		}
	}
}

func TestVersionFlagAndCommandReturnStructuredOutput(t *testing.T) {
	prepareCLIEnv(t)

	out, err := executeRootCommand(t, "version", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var info versionInfo
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		t.Fatalf("expected version JSON, got %q: %v", out, err)
	}
	if strings.TrimSpace(info.Version) == "" || strings.TrimSpace(info.InstallMethod) == "" {
		t.Fatalf("expected populated version info, got %+v", info)
	}

	out, err = executeRootCommand(t, "--version", "--plain")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Install method:") {
		t.Fatalf("expected rich --version output, got %s", out)
	}
}
