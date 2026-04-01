package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

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

func runManagedCommandItem(ctx context.Context, item domain.Finding) error {
	if err := validateManagedCommand(item.CommandPath, item.CommandArgs); err != nil {
		return err
	}
	return runManagedProcess(ctx, item.CommandPath, item.CommandArgs...)
}

func parseNativeCommand(raw string) (nativeCommand, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nativeCommand{}, fmt.Errorf("empty native uninstall command")
	}
	if strings.ContainsAny(value, "|&;<>`\n\r") {
		return nativeCommand{}, fmt.Errorf("shell metacharacters are not allowed")
	}
	fields, err := splitCommandLine(value)
	if err != nil {
		return nativeCommand{}, err
	}
	if len(fields) == 0 {
		return nativeCommand{}, fmt.Errorf("empty native uninstall command")
	}
	executable := fields[0]
	base := strings.ToLower(filepath.Base(executable))
	forbidden := map[string]struct{}{
		"cmd.exe":        {},
		"powershell.exe": {},
		"pwsh.exe":       {},
		"sh":             {},
		"bash":           {},
		"zsh":            {},
	}
	if _, ok := forbidden[base]; ok {
		return nativeCommand{}, fmt.Errorf("shell-based uninstall commands are not allowed")
	}
	if runtime.GOOS == "darwin" && strings.HasSuffix(strings.ToLower(executable), ".app") {
		args := []string{executable}
		if len(fields) > 1 {
			args = append(args, "--args")
			args = append(args, fields[1:]...)
		}
		return nativeCommand{
			Path: "/usr/bin/open",
			Args: args,
		}, nil
	}
	if err := validateNativeExecutable(executable); err != nil {
		return nativeCommand{}, err
	}
	return nativeCommand{
		Path: executable,
		Args: fields[1:],
	}, nil
}

func splitCommandLine(raw string) ([]string, error) {
	var (
		fields  []string
		current strings.Builder
		inQuote rune
	)
	flush := func() {
		if current.Len() == 0 {
			return
		}
		fields = append(fields, current.String())
		current.Reset()
	}
	for _, r := range raw {
		switch {
		case inQuote != 0:
			if r == inQuote {
				inQuote = 0
				continue
			}
			current.WriteRune(r)
		case r == '"' || r == '\'':
			inQuote = r
		case r == ' ' || r == '\t':
			flush()
		default:
			current.WriteRune(r)
		}
	}
	if inQuote != 0 {
		return nil, fmt.Errorf("unterminated quote in uninstall command")
	}
	flush()
	return fields, nil
}

func validateNativeExecutable(executable string) error {
	if executable == "" {
		return fmt.Errorf("native uninstall executable is empty")
	}
	if filepath.IsAbs(executable) || hasWindowsDrivePrefix(executable) {
		return nil
	}
	base := strings.ToLower(executable)
	allowed := map[string]struct{}{
		"msiexec.exe":  {},
		"rundll32.exe": {},
	}
	if _, ok := allowed[base]; ok {
		return nil
	}
	return fmt.Errorf("native uninstall executable %q is not trusted", executable)
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

func hasWindowsDrivePrefix(path string) bool {
	if len(path) < 2 || path[1] != ':' {
		return false
	}
	return unicode.IsLetter(rune(path[0]))
}
