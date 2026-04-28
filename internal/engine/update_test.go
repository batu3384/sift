package engine

import (
	"strings"
	"testing"
)

func TestNormalizeUpdateChannel(t *testing.T) {
	if got := NormalizeUpdateChannel(""); got != UpdateChannelStable {
		t.Fatalf("expected empty channel to normalize to stable, got %q", got)
	}
	if got := NormalizeUpdateChannel(" NIGHTLY "); got != UpdateChannelNightly {
		t.Fatalf("expected nightly normalization, got %q", got)
	}
	if got := NormalizeUpdateChannel("beta"); got != UpdateChannelStable {
		t.Fatalf("expected unknown channel to normalize to stable, got %q", got)
	}
}

func TestCompareReleaseVersions(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{name: "greater", left: "v1.2.3", right: "v1.2.2", want: 1},
		{name: "less", left: "1.2.2", right: "v1.2.3", want: -1},
		{name: "equal with missing patch", left: "v1.2", right: "1.2.0", want: 0},
		{name: "suffix ignored", left: "v1.2.3-beta1", right: "v1.2.3", want: 0},
	}
	for _, tt := range tests {
		if got := compareReleaseVersions(tt.left, tt.right); got != tt.want {
			t.Fatalf("%s: compareReleaseVersions(%q, %q) = %d, want %d", tt.name, tt.left, tt.right, got, tt.want)
		}
	}
}

func TestInstallMethodAndCommandsForHomebrewNightlyAndForce(t *testing.T) {
	service := &Service{
		Executable: func() (string, error) {
			return "/opt/homebrew/Cellar/sift/1.0.0/bin/sift", nil
		},
	}

	method, commands := service.installMethodAndCommands(UpdateChannelStable, true)
	if method != "homebrew" {
		t.Fatalf("expected homebrew install method, got %q", method)
	}
	if len(commands) == 0 || commands[0] != "brew reinstall sift" {
		t.Fatalf("expected forced reinstall command, got %v", commands)
	}

	method, commands = service.installMethodAndCommands(UpdateChannelNightly, false)
	if method != "homebrew" {
		t.Fatalf("expected homebrew install method for nightly, got %q", method)
	}
	if len(commands) < 2 || commands[0] != "Nightly builds are not available for Homebrew installs" {
		t.Fatalf("expected nightly guidance commands, got %v", commands)
	}
}

func TestInstallMethodAndCommandsForWindowsForceAvoidsShellChains(t *testing.T) {
	tests := []struct {
		name       string
		executable string
		wantMethod string
		wantFirst  string
	}{
		{
			name:       "scoop",
			executable: "/Users/me/scoop/apps/sift/current/sift",
			wantMethod: "scoop",
			wantFirst:  "scoop install sift --force",
		},
		{
			name:       "winget",
			executable: `C:\Users\me\AppData\Local\Microsoft\WindowsApps\sift.exe`,
			wantMethod: "winget",
			wantFirst:  "winget install --id SIFT --force",
		},
	}
	for _, tt := range tests {
		service := &Service{
			Executable: func() (string, error) {
				return tt.executable, nil
			},
		}
		method, commands := service.installMethodAndCommands(UpdateChannelStable, true)
		if method != tt.wantMethod {
			t.Fatalf("%s: expected %q install method, got %q", tt.name, tt.wantMethod, method)
		}
		if len(commands) == 0 || commands[0] != tt.wantFirst {
			t.Fatalf("%s: expected first force guidance %q, got %v", tt.name, tt.wantFirst, commands)
		}
		for _, command := range commands {
			if strings.Contains(command, "&&") || strings.Contains(command, ";") {
				t.Fatalf("%s: guidance must not chain shell commands, got %q", tt.name, command)
			}
		}
	}
}

func TestUpdateCommandForMethod(t *testing.T) {
	command, ok := updateCommandForMethod("manual", UpdateChannelNightly, false)
	if !ok || command.Name != "go" || len(command.Args) != 2 || command.Args[1] != "github.com/batu3384/sift/cmd/sift@main" {
		t.Fatalf("expected nightly manual go install command, got %+v ok=%v", command, ok)
	}

	command, ok = updateCommandForMethod("homebrew", UpdateChannelStable, true)
	if !ok || command.Name != "brew" || len(command.Args) != 2 || command.Args[0] != "reinstall" {
		t.Fatalf("expected forced homebrew reinstall command, got %+v ok=%v", command, ok)
	}

	command, ok = updateCommandForMethod("scoop", UpdateChannelStable, true)
	if !ok || command.Name != "scoop" || len(command.Args) != 3 || command.Args[0] != "install" || command.Args[2] != "--force" {
		t.Fatalf("expected forced scoop reinstall without shell command, got %+v ok=%v", command, ok)
	}

	if _, ok := updateCommandForMethod("homebrew", UpdateChannelNightly, false); ok {
		t.Fatal("expected nightly homebrew update command to be unavailable")
	}
}
