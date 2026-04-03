//go:build darwin

package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type darwinAdapter struct {
	home string
}

var execLookPathDarwin = exec.LookPath

func Current() Adapter {
	home, _ := os.UserHomeDir()
	return darwinAdapter{home: home}
}

func (d darwinAdapter) Name() string { return "darwin" }

func (d darwinAdapter) IsFileInUse(ctx context.Context, path string) bool {
	lsofPath, err := execLookPathDarwin("lsof")
	if err != nil {
		// Log warning but assume file is not in use (safe default)
		fmt.Fprintf(os.Stderr, "warning: lsof not found, cannot check if file is in use: %v\n", err)
		return false
	}
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return exec.CommandContext(checkCtx, lsofPath, "-F", "n", "--", path).Run() == nil
}

func darwinAnyProcessRunning(names ...string) bool {
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if err := exec.Command("/usr/bin/pgrep", "-x", name).Run(); err == nil {
			return true
		}
	}
	return false
}

func (d darwinAdapter) IsProcessRunning(names ...string) bool {
	return darwinAnyProcessRunning(names...)
}

func (d darwinAdapter) IsAdminPath(path string) bool {
	for _, prefix := range []string{"/Applications", "/Library", "/System"} {
		if strings.HasPrefix(filepath.Clean(path), prefix) {
			return true
		}
	}
	return false
}
