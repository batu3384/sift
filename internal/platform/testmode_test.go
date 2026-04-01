package platform

import "testing"

func TestAllowDialogSensitiveActionsRespectsTestModeAndLiveIntegration(t *testing.T) {
	t.Setenv(envSiftLiveIntegration, "")
	t.Setenv(envSiftTestMode, "ci-safe")
	if AllowDialogSensitiveActions() {
		t.Fatal("expected ci-safe test mode to disable dialog-sensitive actions")
	}

	t.Setenv(envSiftLiveIntegration, "1")
	if !AllowDialogSensitiveActions() {
		t.Fatal("expected live integration opt-in to re-enable dialog-sensitive actions")
	}
}

func TestAllowDesktopIntegrationBlocksInCIWithoutLiveIntegration(t *testing.T) {
	t.Setenv(envSiftLiveIntegration, "")
	t.Setenv("CI", "1")
	if AllowDesktopIntegration() {
		t.Fatal("expected CI to disable desktop integration by default")
	}

	t.Setenv(envSiftLiveIntegration, "1")
	if !AllowDesktopIntegration() {
		t.Fatal("expected live integration opt-in to allow desktop integration in CI")
	}
}
