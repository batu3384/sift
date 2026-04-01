package config

import (
	"path/filepath"
	"testing"
)

func TestNormalizeCommandNameCanonicalizesSpacing(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":             "",
		"   ":          "",
		"Clean":        "clean",
		" purge scan ": "purge_scan",
		"Touch ID":     "touch_id",
	}

	for input, want := range cases {
		if got := NormalizeCommandName(input); got != want {
			t.Fatalf("NormalizeCommandName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMergeProfileRulesPreservesExistingOrderAndAppendsDefaults(t *testing.T) {
	t.Parallel()

	got := mergeProfileRules(
		[]string{"custom_rule", "logs", "custom_rule"},
		[]string{"logs", "safe_system_clutter", "developer_caches"},
	)

	want := []string{"custom_rule", "logs", "safe_system_clutter", "developer_caches"}
	if len(got) != len(want) {
		t.Fatalf("unexpected merged rule count: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mergeProfileRules() = %v, want %v", got, want)
		}
	}
}

func TestNormalizePathsDropsInvalidAndDedupes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cachePath := filepath.Join(t.TempDir(), "cache")

	got := normalizePaths([]string{"~/Projects/app", "   ", "~/Projects/app", cachePath})
	want := []string{filepath.Join(home, "Projects", "app"), cachePath}
	if len(got) != len(want) {
		t.Fatalf("normalizePaths() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("normalizePaths() = %v, want %v", got, want)
		}
	}
}

func TestValidateReturnsNoWarningsForDefaultConfig(t *testing.T) {
	t.Parallel()

	if warnings := Validate(Default()); len(warnings) != 0 {
		t.Fatalf("expected default config to validate cleanly, got %v", warnings)
	}
}
