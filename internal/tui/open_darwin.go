//go:build darwin

package tui

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/batu3384/sift/internal/platform"
)

func OpenPath(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	if !platform.AllowDesktopIntegration() {
		return fmt.Errorf("open/reveal is disabled in ci-safe mode")
	}
	cmd := exec.CommandContext(context.Background(), "/usr/bin/open", path)
	if err := cmd.Start(); err != nil {
		return err
	}
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}
	return nil
}

func RevealPath(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	if !platform.AllowDesktopIntegration() {
		return fmt.Errorf("open/reveal is disabled in ci-safe mode")
	}
	cmd := exec.CommandContext(context.Background(), "/usr/bin/open", "-R", path)
	if err := cmd.Start(); err != nil {
		return err
	}
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}
	return nil
}
