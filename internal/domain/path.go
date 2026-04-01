package domain

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
)

func NormalizePath(path string) string {
	if path == "" {
		return ""
	}
	// Expand ~ to user home directory (only for ~ or ~/ prefix)
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
		return path
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	// Don't expand ~something (like ~config, ~root) - treat as literal
	// Only expand if it's exactly ~ or starts with ~/
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(abs)
}

func HasPathPrefix(path, prefix string) bool {
	if path == "" || prefix == "" {
		return false
	}
	cleanPath := filepath.Clean(path)
	cleanPrefix := filepath.Clean(prefix)
	if runtime.GOOS == "windows" {
		cleanPath = strings.ToLower(cleanPath)
		cleanPrefix = strings.ToLower(cleanPrefix)
	}
	if cleanPath == cleanPrefix {
		return true
	}
	if cleanPrefix == string(filepath.Separator) {
		return strings.HasPrefix(cleanPath, cleanPrefix)
	}
	return strings.HasPrefix(cleanPath, cleanPrefix+string(filepath.Separator))
}

func IsRootPath(path string) bool {
	if path == "" {
		return false
	}
	clean := filepath.Clean(path)
	return clean == filepath.Dir(clean)
}

func ContainsTraversal(path string) bool {
	if path == "" {
		return false
	}
	normalized := strings.ReplaceAll(path, `\`, `/`)
	for _, part := range strings.Split(normalized, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func HasControlChars(path string) bool {
	for _, r := range path {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func RedactPath(path string) string {
	home, err := os.UserHomeDir()
	if err == nil && HasPathPrefix(path, home) {
		base := filepath.Clean(home)
		if path == base {
			return "~"
		}
		trimmed := strings.TrimPrefix(path, base)
		if strings.HasPrefix(trimmed, string(filepath.Separator)) {
			return "~" + trimmed
		}
		return filepath.Join("~", trimmed)
	}
	return path
}
