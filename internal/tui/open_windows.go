//go:build windows

package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func OpenPath(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	if os.Getenv("CI") != "" {
		return fmt.Errorf("open/reveal is disabled in CI")
	}
	cmd := exec.CommandContext(context.Background(), "explorer.exe", path)
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
	if os.Getenv("CI") != "" {
		return fmt.Errorf("open/reveal is disabled in CI")
	}
	cmd := exec.CommandContext(context.Background(), "explorer.exe", "/select,"+filepath.Clean(path))
	if err := cmd.Start(); err != nil {
		return err
	}
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}
	return nil
}
