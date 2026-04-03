package platform

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"
)

type AdminAccessMode string

const (
	AdminAccessTTY AdminAccessMode = "tty"
	AdminAccessGUI AdminAccessMode = "gui"
)

type AdminSessionProfile struct {
	Mode         AdminAccessMode
	TouchIDHint  bool
	PasswordHint bool
}

var ErrAdminAccessCancelled = errors.New("admin access was cancelled")

func CurrentAdminSessionProfile() AdminSessionProfile {
	profile := AdminSessionProfile{
		Mode:         AdminAccessTTY,
		PasswordHint: true,
	}
	if !adminHasTTY() {
		profile.Mode = AdminAccessGUI
	}
	if runtime.GOOS == "darwin" && profile.Mode == AdminAccessTTY {
		profile.TouchIDHint = true
	}
	return profile
}

func HasAdminSession() bool {
	return adminSessionActiveCheck()
}

func WarmAdminSession(ctx context.Context, prompt string) error {
	if HasAdminSession() {
		return nil
	}
	if TestModeEnabled() && !LiveIntegrationEnabled() {
		return fmt.Errorf("admin access is disabled in ci-safe test mode")
	}
	profile := CurrentAdminSessionProfile()
	if profile.Mode == AdminAccessGUI {
		return warmAdminSessionGUI(ctx, prompt)
	}
	return adminSessionValidateTTY(ctx)
}

func StartAdminKeepalive(ctx context.Context) {
	if TestModeEnabled() && !LiveIntegrationEnabled() {
		return
	}
	go func() {
		adminKeepaliveSleep(adminKeepaliveStabilizeDelay)
		ticker := time.NewTicker(adminKeepaliveInterval)
		defer ticker.Stop()
		retries := 0
		for {
			select {
			case <-ctx.Done():
				adminSessionInvalidate(context.Background())
				return
			case <-ticker.C:
				if err := adminSessionRefresh(ctx); err != nil {
					retries++
					if retries >= 3 {
						adminSessionInvalidate(context.Background())
						return
					}
					continue
				}
				retries = 0
			}
		}
	}()
}

func warmAdminSessionGUI(ctx context.Context, prompt string) error {
	if strings.TrimSpace(prompt) == "" {
		prompt = "SIFT needs admin access to continue."
	}
	adminSessionInvalidate(context.Background())
	dialogPrompt := prompt
	for attempt := 0; attempt < 2; attempt++ {
		password, err := adminPromptPasswordDialog(ctx, dialogPrompt)
		if err != nil {
			return err
		}
		if err := adminSessionValidateGUI(ctx, password); err == nil {
			return nil
		}
		adminSessionInvalidate(context.Background())
		dialogPrompt = "Password was not accepted. Try again to continue."
	}
	return fmt.Errorf("admin password was not accepted")
}
