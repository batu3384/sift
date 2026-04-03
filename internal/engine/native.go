package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/batu3384/sift/internal/domain"
)

type nativeCommand struct {
	Path string
	Args []string
}

var startNativeProcess = defaultStartNativeProcess

func nativeUninstallCommand(app domain.AppEntry) string {
	if strings.TrimSpace(app.QuietUninstallCommand) != "" {
		return strings.TrimSpace(app.QuietUninstallCommand)
	}
	return strings.TrimSpace(app.UninstallCommand)
}

func launchNativeUninstall(ctx context.Context, item domain.Finding) error {
	command, err := parseNativeCommand(item.NativeCommand)
	if err != nil {
		return err
	}
	return startNativeProcess(ctx, command)
}

func defaultStartNativeProcess(ctx context.Context, command nativeCommand) error {
	cmd := exec.CommandContext(ctx, command.Path, command.Args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}
	return nil
}

func runManagedProcess(ctx context.Context, path string, args ...string) error {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func validateManagedCommand(path string, args []string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("managed command executable is empty")
	}
	if !filepath.IsAbs(path) && !hasWindowsDrivePrefix(path) {
		return fmt.Errorf("managed command executable %q must be absolute", path)
	}
	if domain.HasControlChars(path) {
		return fmt.Errorf("managed command executable contains control characters")
	}
	for _, arg := range args {
		if domain.HasControlChars(arg) {
			return fmt.Errorf("managed command contains control characters")
		}
	}
	return nil
}
