//go:build darwin

package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func (d darwinAdapter) Diagnostics(ctx context.Context) []Diagnostic {
	cfgRoot := filepath.Join(d.home, "Library", "Application Support", "sift")
	_, err := os.Stat(cfgRoot)
	status := "ok"
	msg := "app support root reachable"
	if err != nil && !os.IsNotExist(err) {
		status = "warn"
		msg = err.Error()
	}
	diagnostics := []Diagnostic{
		{Name: "platform", Status: "ok", Message: "darwin adapter active"},
		{Name: "user_home", Status: "ok", Message: d.home},
		{Name: "app_support", Status: status, Message: msg},
	}
	if _, err := os.Stat("/usr/bin/defaults"); err == nil {
		diagnostics = append(diagnostics, Diagnostic{Name: "defaults_binary", Status: "ok", Message: "/usr/bin/defaults"})
	} else {
		diagnostics = append(diagnostics, Diagnostic{Name: "defaults_binary", Status: "warn", Message: err.Error()})
	}
	brewCache := filepath.Join(d.home, "Library", "Caches", "Homebrew")
	if _, err := os.Stat(brewCache); err == nil {
		diagnostics = append(diagnostics, Diagnostic{Name: "homebrew_cache", Status: "ok", Message: brewCache})
	}
	diagnostics = append(diagnostics, d.rosettaDiagnostic())
	diagnostics = append(diagnostics, darwinFileVaultDiagnostic(runDarwinDiagnosticCommand(ctx, 4*time.Second, "/usr/bin/fdesetup", "status")))
	if thirdPartyFirewall := darwinThirdPartyFirewall(d.home); thirdPartyFirewall != "" {
		diagnostics = append(diagnostics, Diagnostic{Name: "firewall", Status: "ok", Message: thirdPartyFirewall + " active"})
	} else {
		diagnostics = append(diagnostics, darwinFirewallDiagnostic(runDarwinDiagnosticCommand(ctx, 4*time.Second, "/usr/libexec/ApplicationFirewall/socketfilterfw", "--getglobalstate")))
	}
	diagnostics = append(diagnostics, darwinGatekeeperDiagnostic(runDarwinDiagnosticCommand(ctx, 3*time.Second, "/usr/sbin/spctl", "--status")))
	diagnostics = append(diagnostics, darwinSIPDiagnostic(runDarwinDiagnosticCommand(ctx, 3*time.Second, "/usr/bin/csrutil", "status")))
	if gitPath, lookErr := execLookPathDarwin("git"); lookErr == nil {
		nameOutput, nameErr := runDarwinDiagnosticCommand(ctx, 2*time.Second, gitPath, "config", "--global", "user.name")
		emailOutput, emailErr := runDarwinDiagnosticCommand(ctx, 2*time.Second, gitPath, "config", "--global", "user.email")
		diagnostics = append(diagnostics, darwinGitIdentityDiagnostic(nameOutput, nameErr, emailOutput, emailErr))
	}
	diagnostics = append(diagnostics, darwinLoginItemsDiagnosticForMode(ctx))
	diagnostics = append(diagnostics, darwinSoftwareUpdateDiagnostic(runDarwinDiagnosticCommand(ctx, 10*time.Second, "/usr/sbin/softwareupdate", "-l", "--no-scan")))
	if brewPath, lookErr := execLookPathDarwin("brew"); lookErr == nil {
		diagnostics = append(diagnostics, darwinHomebrewUpdatesDiagnostic(runDarwinDiagnosticCommandEnv(ctx, 5*time.Second, []string{"HOMEBREW_NO_AUTO_UPDATE=1"}, brewPath, "outdated", "--quiet")))
		diagnostics = append(diagnostics, darwinHomebrewHealthDiagnostic(runDarwinDiagnosticCommandEnv(ctx, 8*time.Second, []string{"HOMEBREW_NO_AUTO_UPDATE=1"}, brewPath, "doctor")))
	}
	return diagnostics
}

func darwinThirdPartyFirewall(home string) string {
	for _, candidate := range []struct {
		name  string
		paths []string
	}{
		{name: "Little Snitch", paths: []string{"/Applications/Little Snitch.app", "/Library/Little Snitch", filepath.Join(home, "Applications", "Little Snitch.app")}},
		{name: "LuLu", paths: []string{"/Applications/LuLu.app", filepath.Join(home, "Applications", "LuLu.app")}},
		{name: "Radio Silence", paths: []string{"/Applications/Radio Silence.app", filepath.Join(home, "Applications", "Radio Silence.app")}},
		{name: "Hands Off!", paths: []string{"/Applications/Hands Off!.app", filepath.Join(home, "Applications", "Hands Off!.app")}},
		{name: "Murus", paths: []string{"/Applications/Murus.app", filepath.Join(home, "Applications", "Murus.app")}},
		{name: "Vallum", paths: []string{"/Applications/Vallum.app", filepath.Join(home, "Applications", "Vallum.app")}},
	} {
		for _, candidatePath := range candidate.paths {
			if _, err := os.Stat(candidatePath); err == nil {
				return candidate.name
			}
		}
	}
	return ""
}

func (d darwinAdapter) rosettaDiagnostic() Diagnostic {
	if runtime.GOARCH != "arm64" {
		return Diagnostic{Name: "rosetta2", Status: "ok", Message: "not required on this architecture"}
	}
	for _, candidate := range []string{
		"/Library/Apple/usr/share/rosetta/rosetta",
		"/Library/Apple/usr/share/rosetta",
	} {
		if _, err := os.Stat(candidate); err == nil {
			return Diagnostic{Name: "rosetta2", Status: "ok", Message: "Intel app translation ready"}
		}
	}
	return Diagnostic{Name: "rosetta2", Status: "warn", Message: "Rosetta 2 not installed"}
}

func runDarwinDiagnosticCommand(ctx context.Context, timeout time.Duration, path string, args ...string) (string, error) {
	return runDarwinDiagnosticCommandEnv(ctx, timeout, nil, path, args...)
}

func runDarwinDiagnosticCommandEnv(ctx context.Context, timeout time.Duration, env []string, path string, args ...string) (string, error) {
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(commandCtx, path, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func darwinFileVaultDiagnostic(output string, err error) Diagnostic {
	if err != nil && strings.TrimSpace(output) == "" {
		return Diagnostic{Name: "filevault", Status: "warn", Message: "status unavailable"}
	}
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "filevault is on"):
		return Diagnostic{Name: "filevault", Status: "ok", Message: "disk encryption active"}
	case strings.Contains(lower, "filevault is off"):
		return Diagnostic{Name: "filevault", Status: "warn", Message: "disk encryption disabled"}
	default:
		return Diagnostic{Name: "filevault", Status: "warn", Message: firstDiagnosticLine(output, "status unavailable")}
	}
}

func darwinGatekeeperDiagnostic(output string, err error) Diagnostic {
	if err != nil && strings.TrimSpace(output) == "" {
		return Diagnostic{Name: "gatekeeper", Status: "warn", Message: "status unavailable"}
	}
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "assessments enabled"):
		return Diagnostic{Name: "gatekeeper", Status: "ok", Message: "app download protection active"}
	case strings.Contains(lower, "assessments disabled"):
		return Diagnostic{Name: "gatekeeper", Status: "warn", Message: "app security disabled"}
	default:
		return Diagnostic{Name: "gatekeeper", Status: "warn", Message: firstDiagnosticLine(output, "status unavailable")}
	}
}

func darwinFirewallDiagnostic(output string, err error) Diagnostic {
	if err != nil && strings.TrimSpace(output) == "" {
		return Diagnostic{Name: "firewall", Status: "warn", Message: "status unavailable"}
	}
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(output, "State = 1"), strings.Contains(output, "State = 2"):
		return Diagnostic{Name: "firewall", Status: "ok", Message: "network protection enabled"}
	case strings.Contains(output, "State = 0"), strings.Contains(lower, "disabled"):
		return Diagnostic{Name: "firewall", Status: "warn", Message: "network protection disabled"}
	default:
		return Diagnostic{Name: "firewall", Status: "warn", Message: firstDiagnosticLine(output, "status unavailable")}
	}
}

func darwinSIPDiagnostic(output string, err error) Diagnostic {
	if err != nil && strings.TrimSpace(output) == "" {
		return Diagnostic{Name: "sip", Status: "warn", Message: "status unavailable"}
	}
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "enabled"):
		return Diagnostic{Name: "sip", Status: "ok", Message: "system integrity protected"}
	case strings.Contains(lower, "disabled"):
		return Diagnostic{Name: "sip", Status: "warn", Message: "system protection disabled"}
	default:
		return Diagnostic{Name: "sip", Status: "warn", Message: firstDiagnosticLine(output, "status unavailable")}
	}
}

func darwinGitIdentityDiagnostic(nameOutput string, nameErr error, emailOutput string, emailErr error) Diagnostic {
	nameOutput = strings.TrimSpace(nameOutput)
	emailOutput = strings.TrimSpace(emailOutput)
	if nameOutput != "" && emailOutput != "" {
		return Diagnostic{Name: "git_identity", Status: "ok", Message: fmt.Sprintf("%s <%s>", nameOutput, emailOutput)}
	}
	if nameErr != nil && emailErr != nil && nameOutput == "" && emailOutput == "" {
		return Diagnostic{Name: "git_identity", Status: "warn", Message: "global git identity not configured"}
	}
	if nameOutput == "" && emailOutput == "" {
		return Diagnostic{Name: "git_identity", Status: "warn", Message: "global git identity not configured"}
	}
	missing := []string{}
	if nameOutput == "" {
		missing = append(missing, "user.name")
	}
	if emailOutput == "" {
		missing = append(missing, "user.email")
	}
	return Diagnostic{Name: "git_identity", Status: "warn", Message: "missing " + strings.Join(missing, " and ")}
}

func darwinLoginItemsDiagnostic(output string, err error) Diagnostic {
	if err != nil && strings.TrimSpace(output) == "" {
		return Diagnostic{Name: "login_items", Status: "warn", Message: "login item scan unavailable"}
	}
	trimmed := strings.TrimSpace(output)
	if trimmed == "" || strings.EqualFold(trimmed, "missing value") {
		return Diagnostic{Name: "login_items", Status: "ok", Message: "none configured"}
	}
	items := []string{}
	for _, part := range strings.Split(trimmed, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	if len(items) == 0 {
		return Diagnostic{Name: "login_items", Status: "ok", Message: "none configured"}
	}
	preview := strings.Join(items[:min(len(items), 3)], ", ")
	if len(items) > 3 {
		preview += fmt.Sprintf(" +%d", len(items)-3)
	}
	status := "ok"
	if len(items) > 15 {
		status = "warn"
	}
	appWord := map[bool]string{true: "app", false: "apps"}[len(items) == 1]
	return Diagnostic{Name: "login_items", Status: status, Message: fmt.Sprintf("%d %s  •  %s", len(items), appWord, preview)}
}

func darwinLoginItemsDiagnosticForMode(ctx context.Context) Diagnostic {
	if !AllowDialogSensitiveActions() {
		return Diagnostic{Name: "login_items", Status: "ok", Message: "skipped in ci-safe test mode"}
	}
	loginItemsOutput, loginItemsErr := runDarwinDiagnosticCommand(ctx, 4*time.Second, "/usr/bin/osascript", "-e", `tell application "System Events" to get the name of every login item`)
	return darwinLoginItemsDiagnostic(loginItemsOutput, loginItemsErr)
}

func darwinSoftwareUpdateDiagnostic(output string, err error) Diagnostic {
	if err != nil && strings.TrimSpace(output) == "" {
		return Diagnostic{Name: "macos_updates", Status: "warn", Message: "update scan unavailable"}
	}
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "no new software available"), strings.Contains(lower, "no updates are available"):
		return Diagnostic{Name: "macos_updates", Status: "ok", Message: "macOS updates up to date"}
	case strings.Contains(output, "*"), strings.Contains(lower, "label:"):
		return Diagnostic{Name: "macos_updates", Status: "warn", Message: "macOS updates available"}
	default:
		return Diagnostic{Name: "macos_updates", Status: "warn", Message: firstDiagnosticLine(output, "update scan unavailable")}
	}
}

func darwinHomebrewUpdatesDiagnostic(output string, err error) Diagnostic {
	if err != nil && strings.TrimSpace(output) == "" {
		return Diagnostic{Name: "brew_updates", Status: "warn", Message: "Homebrew update check unavailable"}
	}
	lines := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) != "" {
			lines++
		}
	}
	if lines == 0 {
		return Diagnostic{Name: "brew_updates", Status: "ok", Message: "Homebrew formulas up to date"}
	}
	updateWord := map[bool]string{true: "update", false: "updates"}[lines == 1]
	return Diagnostic{Name: "brew_updates", Status: "warn", Message: fmt.Sprintf("%d Homebrew %s available", lines, updateWord)}
}

func darwinHomebrewHealthDiagnostic(output string, err error) Diagnostic {
	trimmed := strings.TrimSpace(output)
	if err != nil && trimmed == "" {
		return Diagnostic{Name: "brew_health", Status: "warn", Message: "Homebrew health check unavailable"}
	}
	lower := strings.ToLower(trimmed)
	switch {
	case trimmed == "", strings.Contains(lower, "your system is ready to brew"):
		return Diagnostic{Name: "brew_health", Status: "ok", Message: "Homebrew doctor clean"}
	default:
		return Diagnostic{Name: "brew_health", Status: "warn", Message: firstDiagnosticLine(trimmed, "Homebrew doctor reported issues")}
	}
}

func firstDiagnosticLine(output string, fallback string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return fallback
}
