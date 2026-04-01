package domain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	home, _ := CurrentHomeDir()
	cwd, _ := os.Getwd()
	absolutePath := filepath.Join(t.TempDir(), "usr", "local")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"tilde_only", "~", home},
		{"tilde_slash", "~/Downloads", filepath.Join(home, "Downloads")},
		{"tilde_slash_subpath", "~/Documents/Project", filepath.Join(home, "Documents/Project")},
		{"tilde_not_expanded", "~config", filepath.Join(cwd, "~config")},
		{"tilde_username", "~root", filepath.Join(cwd, "~root")},
		{"absolute_path", absolutePath, absolutePath},
		{"relative_path", "Documents", filepath.Join(cwd, "Documents")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHasPathPrefix(t *testing.T) {
	home, _ := CurrentHomeDir()
	root := t.TempDir()
	userRoot := filepath.Join(root, "user")
	otherRoot := filepath.Join(root, "other")
	rootPrefix := filepath.VolumeName(root) + string(filepath.Separator)
	if rootPrefix == string(filepath.Separator) && filepath.IsAbs(root) {
		rootPrefix = string(filepath.Separator)
	}

	tests := []struct {
		name     string
		path     string
		prefix   string
		expected bool
	}{
		{"empty_path", "", "/home", false},
		{"empty_prefix", "/home", "", false},
		{"exact_match", userRoot, userRoot, true},
		{"direct_child", filepath.Join(userRoot, "file"), userRoot, true},
		{"not_child", otherRoot, userRoot, false},
		{"root_prefix", userRoot, rootPrefix, true},
		{"home_relative", "Documents", filepath.Join(userRoot, "Documents"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPathPrefix(tt.path, tt.prefix)
			if result != tt.expected {
				t.Errorf("HasPathPrefix(%q, %q) = %v, want %v", tt.path, tt.prefix, result, tt.expected)
			}
		})
	}

	// Test with home directory
	t.Run("home_prefix", func(t *testing.T) {
		result := HasPathPrefix(filepath.Join(home, "Documents"), home)
		if !result {
			t.Errorf("HasPathPrefix(%q, %q) = false, want true", filepath.Join(home, "Documents"), home)
		}
	})
}

func TestIsRootPath(t *testing.T) {
	root := filepath.VolumeName(os.TempDir()) + string(filepath.Separator)
	if root == string(filepath.Separator) && filepath.IsAbs(os.TempDir()) {
		root = string(filepath.Separator)
	}
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"empty", "", false},
		{"root_path", root, true},
		{"non_root", filepath.Join(os.TempDir(), "home"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRootPath(tt.path)
			if result != tt.expected {
				t.Errorf("IsRootPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestContainsTraversal(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"empty", "", false},
		{"no_traversal", "/home/user/file", false},
		{"traversal_single", "/home/../etc", true},
		{"traversal_double", "/home/user/../../etc", true},
		{"traversal_mixed", "/home/./user/../file", true},
		{"windows_traversal", "C:\\Users\\..\\Windows", true},
		{"traversal_at_start", "../etc", true},
		{"traversal_in_middle", "/home/./user", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsTraversal(tt.path)
			if result != tt.expected {
				t.Errorf("ContainsTraversal(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestHasControlChars(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"empty", "", false},
		{"normal_path", "/home/user/file", false},
		{"with_tab", "/home/user\tfile", true},
		{"with_newline", "/home/user\nfile", true},
		{"with_null", "/home\x00user", true},
		{"with_carriage_return", "/home\ruser", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasControlChars(tt.path)
			if result != tt.expected {
				t.Errorf("HasControlChars(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestRedactPath(t *testing.T) {
	home, _ := CurrentHomeDir()
	outsideHome := filepath.Join(t.TempDir(), "usr", "local")

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"empty", "", ""},
		{"home_exact", home, "~"},
		{"home_subpath", filepath.Join(home, "Documents"), "~/Documents"},
		{"home_subpath_deep", filepath.Join(home, "Documents/Project/File"), "~/Documents/Project/File"},
		{"outside_home", outsideHome, outsideHome},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filepath.ToSlash(RedactPath(tt.path))
			expected := filepath.ToSlash(tt.expected)
			if result != expected {
				t.Errorf("RedactPath(%q) = %q, want %q", tt.path, result, expected)
			}
		})
	}
}
