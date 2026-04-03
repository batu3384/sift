package platform

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

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
