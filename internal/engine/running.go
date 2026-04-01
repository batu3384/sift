package engine

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"github.com/shirou/gopsutil/v4/process"

	"github.com/batuhanyuksel/sift/internal/domain"
)

type runningProcess struct {
	Name string
	Exe  string
}

var listRunningProcesses = defaultListRunningProcesses
var runningProcessProbeMu sync.Mutex

func defaultListRunningProcesses(ctx context.Context) ([]runningProcess, error) {
	runningProcessProbeMu.Lock()
	defer runningProcessProbeMu.Unlock()

	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]runningProcess, 0, len(procs))
	for _, proc := range procs {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		default:
		}
		entry := runningProcess{}
		entry.Name, _ = proc.NameWithContext(ctx)
		entry.Exe, _ = proc.ExeWithContext(ctx)
		if strings.TrimSpace(entry.Name) == "" && strings.TrimSpace(entry.Exe) == "" {
			continue
		}
		out = append(out, entry)
	}
	return out, nil
}

func appRunningKeys(app domain.AppEntry) []string {
	values := []string{
		app.DisplayName,
		app.Name,
		strings.TrimSuffix(filepath.Base(app.BundlePath), filepath.Ext(app.BundlePath)),
	}
	for _, support := range app.SupportPaths {
		values = append(values, strings.TrimSuffix(filepath.Base(support), filepath.Ext(support)))
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		key := normalizedRuntimeKey(value)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func appIsRunning(app domain.AppEntry, processes []runningProcess) bool {
	keys := appRunningKeys(app)
	if len(keys) == 0 {
		return false
	}
	for _, proc := range processes {
		candidates := []string{
			normalizedRuntimeKey(proc.Name),
			normalizedRuntimeKey(strings.TrimSuffix(filepath.Base(proc.Exe), filepath.Ext(proc.Exe))),
		}
		for _, candidate := range candidates {
			if candidate == "" {
				continue
			}
			for _, key := range keys {
				if candidate == key || strings.Contains(candidate, key) || strings.Contains(key, candidate) {
					return true
				}
			}
		}
	}
	return false
}

func protectForRunningApp(items []domain.Finding, app domain.AppEntry) []domain.Finding {
	message := "Close the app before launching its uninstaller or deleting remnants."
	if app.DisplayName != "" {
		message = "Close " + app.DisplayName + " before launching its uninstaller or deleting remnants."
	}
	for idx := range items {
		if items[idx].Action == domain.ActionAdvisory {
			continue
		}
		items[idx].Status = domain.StatusProtected
		items[idx].Policy = domain.PolicyDecision{
			Allowed: false,
			Reason:  domain.ProtectionRunningApp,
			Message: message,
		}
		items[idx].Recovery = domain.RecoveryHint{
			Message:  message,
			Location: "running process guard",
		}
	}
	return items
}

func normalizedRuntimeKey(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
