package engine

import (
	"strings"
	"testing"
)

func TestTouchIDDetectionHelpers(t *testing.T) {
	mainPAM := []byte("auth       include       sudo_local\n")
	localPAM := []byte("# sudo_local\nauth       sufficient     pam_tid.so\n")

	if !touchIDSudoLocalSupported(mainPAM) {
		t.Fatal("expected sudo_local support to be detected")
	}
	if touchIDConfiguredIn(mainPAM) {
		t.Fatal("did not expect pam_tid in main PAM snippet")
	}
	if !touchIDConfiguredIn(localPAM) {
		t.Fatal("expected pam_tid in local PAM snippet")
	}
}

func TestRenderTouchIDPAMEnableAndDisable(t *testing.T) {
	raw := []byte("# sudo_local: local customizations for sudo\n")

	enabled, changed := renderTouchIDPAM(raw, true)
	if !changed {
		t.Fatal("expected enable render to report a change")
	}
	if !strings.HasPrefix(string(enabled), "auth       sufficient     pam_tid.so\n") {
		t.Fatalf("expected pam_tid to be prepended, got %q", string(enabled))
	}

	disabled, changed := renderTouchIDPAM(enabled, false)
	if !changed {
		t.Fatal("expected disable render to report a change")
	}
	if strings.Contains(string(disabled), "pam_tid.so") {
		t.Fatalf("expected pam_tid to be removed, got %q", string(disabled))
	}
}

func TestTouchIDCommandsForSudoLocalMigration(t *testing.T) {
	state := touchIDState{
		pamPath:            "/etc/pam.d/sudo",
		localPath:          "/etc/pam.d/sudo_local",
		localExists:        true,
		sudoLocalSupported: true,
		legacyEnabled:      true,
	}

	commands := touchIDCommands(state, true)
	joined := strings.Join(commands, "\n")
	if !strings.Contains(joined, "sudo_local") {
		t.Fatalf("expected sudo_local migration commands, got %v", commands)
	}
	if !strings.Contains(joined, "sed -i '' '/pam_tid\\.so/d' /etc/pam.d/sudo") {
		t.Fatalf("expected legacy cleanup command, got %v", commands)
	}

	disableCommands := touchIDCommands(state, false)
	disableJoined := strings.Join(disableCommands, "\n")
	if !strings.Contains(disableJoined, "sed -i '' '/pam_tid\\.so/d' /etc/pam.d/sudo_local") {
		t.Fatalf("expected sudo_local disable cleanup, got %v", disableCommands)
	}
}
