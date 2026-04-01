package platform

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
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

var (
	adminHasTTY = defaultAdminHasTTY

	adminPromptPasswordDialog = defaultAdminPromptPasswordDialog

	adminSessionActiveCheck = defaultAdminSessionActiveCheck
	adminSessionInvalidate  = defaultAdminSessionInvalidate
	adminSessionValidateTTY = defaultAdminSessionValidateTTY
	adminSessionValidateGUI = defaultAdminSessionValidateGUI
	adminSessionRefresh     = defaultAdminSessionRefresh

	adminKeepaliveStabilizeDelay = 2 * time.Second
	adminKeepaliveInterval       = 30 * time.Second
	adminKeepaliveSleep          = time.Sleep
)

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

func defaultAdminHasTTY() bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	_ = tty.Close()
	return true
}

func defaultAdminPromptPasswordDialog(ctx context.Context, prompt string) (string, error) {
	if !AllowDialogSensitiveActions() {
		return "", fmt.Errorf("admin dialog is disabled in ci-safe test mode")
	}
	script := fmt.Sprintf(`display dialog %q default answer "" with title "SIFT" with icon caution with hidden answer`, prompt)
	cmd := exec.CommandContext(ctx, "/usr/bin/osascript", "-e", script, "-e", "text returned of result")
	out, err := cmd.Output()
	if err != nil {
		return "", ErrAdminAccessCancelled
	}
	password := strings.TrimSpace(string(out))
	if password == "" {
		return "", ErrAdminAccessCancelled
	}
	return password, nil
}

func defaultAdminSessionActiveCheck() bool {
	return exec.Command("/usr/bin/sudo", "-n", "true").Run() == nil
}

func defaultAdminSessionInvalidate(ctx context.Context) {
	_ = exec.CommandContext(ctx, "/usr/bin/sudo", "-k").Run()
}

func defaultAdminSessionValidateTTY(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "/usr/bin/sudo", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func defaultAdminSessionValidateGUI(ctx context.Context, password string) error {
	cmd := exec.CommandContext(ctx, "/usr/bin/sudo", "-S", "-p", "", "-v")
	cmd.Stdin = strings.NewReader(password + "\n")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func defaultAdminSessionRefresh(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "/usr/bin/sudo", "-n", "-v")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}
