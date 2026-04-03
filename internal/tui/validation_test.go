package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateProtectedPath(t *testing.T) {
	// Test empty path
	msg, ok := validateProtectedPath("")
	if ok {
		t.Error("expected empty path to be invalid")
	}
	if msg == "" {
		t.Error("expected error message for empty path")
	}

	// Test whitespace only
	msg, ok = validateProtectedPath("   ")
	if ok {
		t.Error("expected whitespace path to be invalid")
	}
	if msg == "" {
		t.Error("expected error message for whitespace path")
	}

	// Test valid path with tilde expansion
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	msg, ok = validateProtectedPath("~/test")
	if !ok {
		t.Errorf("expected valid path, got error: %s", msg)
	}
	if msg != filepath.Join(home, "test") {
		t.Errorf("expected path to be expanded, got %s", msg)
	}

	// Test absolute path
	msg, ok = validateProtectedPath("/tmp")
	if !ok {
		t.Errorf("expected valid absolute path, got error: %s", msg)
	}

	// Test non-existent path (should be valid but warn)
	msg, ok = validateProtectedPath("/nonexistent/path/12345")
	if !ok {
		t.Errorf("expected non-existent path to be valid, got error: %s", msg)
	}
}

func TestMaxAnalyzeHistoryLimit(t *testing.T) {
	// Test that the constant is defined
	if maxAnalyzeHistory != 50 {
		t.Errorf("expected maxAnalyzeHistory to be 50, got %d", maxAnalyzeHistory)
	}
}
