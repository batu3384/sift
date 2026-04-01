package platform

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCurrentAdminSessionProfileFallsBackToGUIWithoutTTY(t *testing.T) {
	originalHasTTY := adminHasTTY
	defer func() { adminHasTTY = originalHasTTY }()

	adminHasTTY = func() bool { return false }

	profile := CurrentAdminSessionProfile()
	if profile.Mode != AdminAccessGUI {
		t.Fatalf("expected gui mode, got %+v", profile)
	}
	if !profile.PasswordHint {
		t.Fatalf("expected password hint, got %+v", profile)
	}
}

func TestWarmAdminSessionUsesGUIFallbackWhenTTYUnavailable(t *testing.T) {
	originalHasTTY := adminHasTTY
	originalActiveCheck := adminSessionActiveCheck
	originalInvalidate := adminSessionInvalidate
	originalPrompt := adminPromptPasswordDialog
	originalValidateGUI := adminSessionValidateGUI
	defer func() {
		adminHasTTY = originalHasTTY
		adminSessionActiveCheck = originalActiveCheck
		adminSessionInvalidate = originalInvalidate
		adminPromptPasswordDialog = originalPrompt
		adminSessionValidateGUI = originalValidateGUI
	}()

	adminHasTTY = func() bool { return false }
	adminSessionActiveCheck = func() bool { return false }
	invalidateCalls := 0
	adminSessionInvalidate = func(context.Context) { invalidateCalls++ }
	adminPromptPasswordDialog = func(ctx context.Context, prompt string) (string, error) {
		if prompt == "" {
			t.Fatal("expected non-empty prompt")
		}
		return "secret", nil
	}
	adminSessionValidateGUI = func(ctx context.Context, password string) error {
		if password != "secret" {
			t.Fatalf("expected dialog password to be reused, got %q", password)
		}
		return nil
	}

	if err := WarmAdminSession(context.Background(), "Need admin"); err != nil {
		t.Fatalf("expected GUI warmup to succeed, got %v", err)
	}
	if invalidateCalls == 0 {
		t.Fatal("expected sudo cache to be invalidated before GUI auth")
	}
}

func TestWarmAdminSessionRetriesRejectedGUIPassword(t *testing.T) {
	originalHasTTY := adminHasTTY
	originalActiveCheck := adminSessionActiveCheck
	originalInvalidate := adminSessionInvalidate
	originalPrompt := adminPromptPasswordDialog
	originalValidateGUI := adminSessionValidateGUI
	defer func() {
		adminHasTTY = originalHasTTY
		adminSessionActiveCheck = originalActiveCheck
		adminSessionInvalidate = originalInvalidate
		adminPromptPasswordDialog = originalPrompt
		adminSessionValidateGUI = originalValidateGUI
	}()

	adminHasTTY = func() bool { return false }
	adminSessionActiveCheck = func() bool { return false }
	adminSessionInvalidate = func(context.Context) {}
	prompts := []string{}
	adminPromptPasswordDialog = func(ctx context.Context, prompt string) (string, error) {
		prompts = append(prompts, prompt)
		return "secret", nil
	}
	attempt := 0
	adminSessionValidateGUI = func(ctx context.Context, password string) error {
		attempt++
		if attempt == 1 {
			return errors.New("bad password")
		}
		return nil
	}

	if err := WarmAdminSession(context.Background(), "Need admin"); err != nil {
		t.Fatalf("expected retry flow to succeed, got %v", err)
	}
	if len(prompts) != 2 {
		t.Fatalf("expected two password prompts, got %d", len(prompts))
	}
	if prompts[1] != "Password was not accepted. Try again to continue." {
		t.Fatalf("expected retry hint prompt, got %q", prompts[1])
	}
}

func TestWarmAdminSessionUsesTTYValidationWhenTTYAvailable(t *testing.T) {
	originalHasTTY := adminHasTTY
	originalActiveCheck := adminSessionActiveCheck
	originalValidateTTY := adminSessionValidateTTY
	defer func() {
		adminHasTTY = originalHasTTY
		adminSessionActiveCheck = originalActiveCheck
		adminSessionValidateTTY = originalValidateTTY
	}()

	adminHasTTY = func() bool { return true }
	adminSessionActiveCheck = func() bool { return false }
	called := false
	adminSessionValidateTTY = func(ctx context.Context) error {
		called = true
		return nil
	}

	if err := WarmAdminSession(context.Background(), "Need admin"); err != nil {
		t.Fatalf("expected tty warmup to succeed, got %v", err)
	}
	if !called {
		t.Fatal("expected tty warmup to be used")
	}
}

func TestStartAdminKeepaliveInvalidatesOnCancel(t *testing.T) {
	originalSleep := adminKeepaliveSleep
	originalInterval := adminKeepaliveInterval
	originalDelay := adminKeepaliveStabilizeDelay
	originalRefresh := adminSessionRefresh
	originalInvalidate := adminSessionInvalidate
	defer func() {
		adminKeepaliveSleep = originalSleep
		adminKeepaliveInterval = originalInterval
		adminKeepaliveStabilizeDelay = originalDelay
		adminSessionRefresh = originalRefresh
		adminSessionInvalidate = originalInvalidate
	}()

	adminKeepaliveSleep = func(time.Duration) {}
	adminKeepaliveInterval = 5 * time.Millisecond
	adminKeepaliveStabilizeDelay = 0
	adminSessionRefresh = func(ctx context.Context) error { return nil }
	invalidated := make(chan struct{}, 1)
	adminSessionInvalidate = func(context.Context) {
		select {
		case invalidated <- struct{}{}:
		default:
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	StartAdminKeepalive(ctx)
	cancel()

	select {
	case <-invalidated:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected keepalive to invalidate sudo session on cancel")
	}
}
