//go:build windows

package platform

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

type windowsAdapter struct {
	home      string
	localApp  string
	roaming   string
	programDT string
}

func Current() Adapter {
	home, _ := os.UserHomeDir()
	return windowsAdapter{
		home:      home,
		localApp:  os.Getenv("LOCALAPPDATA"),
		roaming:   os.Getenv("APPDATA"),
		programDT: os.Getenv("ProgramData"),
	}
}

func (w windowsAdapter) Name() string { return "windows" }

func (w windowsAdapter) IsProcessRunning(...string) bool { return false }

func (w windowsAdapter) ResolveTargets(in []string) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		if strings.HasPrefix(item, "%LOCALAPPDATA%") {
			out = append(out, filepath.Join(w.localApp, strings.TrimPrefix(item, "%LOCALAPPDATA%\\")))
			continue
		}
		out = append(out, item)
	}
	return out
}

func (w windowsAdapter) IsFileInUse(context.Context, string) bool {
	return false
}

func (w windowsAdapter) IsAdminPath(path string) bool {
	for _, prefix := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramData")} {
		if prefix != "" && strings.HasPrefix(strings.ToLower(filepath.Clean(path)), strings.ToLower(prefix)) {
			return true
		}
	}
	return false
}
