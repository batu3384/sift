package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
)

func parseNativeCommand(raw string) (nativeCommand, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nativeCommand{}, fmt.Errorf("empty native uninstall command")
	}
	if strings.ContainsAny(value, "|&;<>`\n\r") {
		return nativeCommand{}, fmt.Errorf("shell metacharacters are not allowed")
	}
	if command, ok, err := parseAbsoluteNativeCommand(value); ok || err != nil {
		return command, err
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

func parseAbsoluteNativeCommand(raw string) (nativeCommand, bool, error) {
	if raw == "" || strings.HasPrefix(raw, `"`) || strings.HasPrefix(raw, `'`) {
		return nativeCommand{}, false, nil
	}
	if !filepath.IsAbs(raw) && !hasWindowsDrivePrefix(raw) {
		return nativeCommand{}, false, nil
	}
	splitPoints := []int{len(raw)}
	for idx, r := range raw {
		if unicode.IsSpace(r) {
			splitPoints = append(splitPoints, idx)
		}
	}
	for _, idx := range splitPoints {
		executable := strings.TrimSpace(raw[:idx])
		if executable == "" {
			continue
		}
		if !filepath.IsAbs(executable) && !hasWindowsDrivePrefix(executable) {
			continue
		}
		info, err := os.Stat(executable)
		if err != nil || info.IsDir() {
			continue
		}
		if err := validateNativeExecutable(executable); err != nil {
			return nativeCommand{}, true, err
		}
		argText := strings.TrimSpace(raw[idx:])
		if argText == "" {
			return nativeCommand{Path: executable}, true, nil
		}
		args, err := splitCommandLine(argText)
		if err != nil {
			return nativeCommand{}, true, err
		}
		return nativeCommand{Path: executable, Args: args}, true, nil
	}
	return nativeCommand{}, false, nil
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

func hasWindowsDrivePrefix(path string) bool {
	if len(path) < 2 || path[1] != ':' {
		return false
	}
	return unicode.IsLetter(rune(path[0]))
}
