package tui

import (
	"testing"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func TestAccessibilityHomeAction(t *testing.T) {
	action := homeAction{
		ID:          "clean",
		Title:       "🧹 Clean",
		Description: "Clean junk files",
		Safety:      "Safe",
		Enabled:     true,
	}

	metadata := GetHomeActionAccessibility(action)

	if metadata.Role != "menuitem" {
		t.Errorf("expected role 'menuitem', got '%s'", metadata.Role)
	}
	if metadata.Label != action.Title {
		t.Errorf("expected label '%s', got '%s'", action.Title, metadata.Label)
	}
	if metadata.Shortcut != "enter" {
		t.Errorf("expected shortcut 'enter', got '%s'", metadata.Shortcut)
	}
}

func TestAccessibilityDisabledAction(t *testing.T) {
	action := homeAction{
		ID:      "uninstall",
		Title:   "Uninstall",
		Enabled: false,
	}

	metadata := GetHomeActionAccessibility(action)

	if metadata.Role != "menuitem disabled" {
		t.Errorf("expected role 'menuitem disabled', got '%s'", metadata.Role)
	}
}

func TestAccessibilityMenuItem(t *testing.T) {
	finding := domain.Finding{
		Path:   "/tmp/test",
		Status: domain.StatusCompleted,
	}

	metadata := GetMenuItemAccessibility(finding, 0, 10)

	if metadata.Role != "listitem" {
		t.Errorf("expected role 'listitem', got '%s'", metadata.Role)
	}
	if metadata.Description != finding.Path {
		t.Errorf("expected description '%s', got '%s'", finding.Path, metadata.Description)
	}
}

func TestStatusToAccessibility(t *testing.T) {
	tests := []struct {
		status   domain.FindingStatus
		expected string
	}{
		{domain.StatusPlanned, "planned"},
		{domain.StatusCompleted, "completed"},
		{domain.StatusFailed, "failed"},
		{domain.StatusProtected, "protected"},
		{domain.StatusDeleted, "deleted"},
	}

	for _, tt := range tests {
		result := StatusToAccessibility(tt.status)
		if result != tt.expected {
			t.Errorf("StatusToAccessibility(%s) = %s; want %s", tt.status, result, tt.expected)
		}
	}
}

func TestCategoryToAccessibility(t *testing.T) {
	tests := []struct {
		category domain.Category
		expected string
	}{
		{domain.Category("TEMP_FILES"), "Temp Files"},
		{domain.Category("BROWSER_DATA"), "Browser Data"},
		{domain.Category("LOGS"), "Logs"},
		{domain.Category("APP_LEFTOVERS"), "App Leftovers"},
	}

	for _, tt := range tests {
		result := CategoryToAccessibility(tt.category)
		if result != tt.expected {
			t.Errorf("CategoryToAccessibility(%s) = %s; want %s", tt.category, result, tt.expected)
		}
	}
}

func TestScreenReaderFormat(t *testing.T) {
	metadata := AccessibilityMetadata{
		Role:        "button",
		Label:       "Clean",
		Description: "Clean junk files",
		State:       "safe",
		Shortcut:    "enter",
	}

	expected := "Clean. Clean junk files. State: safe. Shortcut: enter."
	result := metadata.FormatForScreenReader()

	if result != expected {
		t.Errorf("FormatForScreenReader() = %s; want %s", result, expected)
	}
}