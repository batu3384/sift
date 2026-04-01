package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeFillsDefaultsAndDedupes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := Normalize(Config{
		Profiles: map[string][]string{
			"safe": {"temp_files"},
		},
		ProtectedPaths:    []string{"/tmp/a", "/tmp/a"},
		ProtectedFamilies: []string{"browser_profiles", "Browser_Profiles"},
		PurgeSearchPaths:  []string{"~/Projects/app", "~/Projects/app"},
		DisabledRules:     []string{"logs", "logs"},
		InteractionMode:   "",
	})
	if len(cfg.Profiles["developer"]) == 0 || len(cfg.Profiles["deep"]) == 0 {
		t.Fatal("expected default profiles to be preserved")
	}
	if !slicesContains(cfg.Profiles["safe"], "stale_login_items") || !slicesContains(cfg.Profiles["deep"], "system_update_artifacts") {
		t.Fatalf("expected built-in profile rules to merge forward, got %+v", cfg.Profiles)
	}
	if len(cfg.ProtectedPaths) != 1 || len(cfg.ProtectedFamilies) != 1 || len(cfg.DisabledRules) != 1 || len(cfg.PurgeSearchPaths) != 1 {
		t.Fatal("expected duplicate paths and rules to be removed")
	}
	if len(cfg.CommandExcludes) != 0 {
		t.Fatalf("expected empty command excludes by default, got %+v", cfg.CommandExcludes)
	}
	if cfg.PurgeSearchPaths[0] != filepath.Join(home, "Projects", "app") {
		t.Fatalf("expected HOME expansion, got %v", cfg.PurgeSearchPaths)
	}
}

func TestValidateReturnsWarningsForUnknownModes(t *testing.T) {
	t.Parallel()
	warnings := Validate(Config{
		Profiles:        map[string][]string{"safe": {"temp_files"}},
		InteractionMode: "weird",
		TrashMode:       "odd",
		ConfirmLevel:    "unsafe",
	})
	if len(warnings) < 3 {
		t.Fatalf("expected warnings, got %v", warnings)
	}
}

func TestAddAndRemoveProtectedPathNormalizesAndPersists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg, added, err := AddProtectedPath(Default(), "~/Projects/keep-me")
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(home, "Projects", "keep-me")
	if added != expected {
		t.Fatalf("expected normalized path %s, got %s", expected, added)
	}
	if len(cfg.ProtectedPaths) != 1 || cfg.ProtectedPaths[0] != expected {
		t.Fatalf("expected protected path to be stored, got %+v", cfg.ProtectedPaths)
	}
	cfg, removedPath, removed, err := RemoveProtectedPath(cfg, expected)
	if err != nil {
		t.Fatal(err)
	}
	if !removed || removedPath != expected {
		t.Fatalf("expected protected path to be removed, got removed=%v path=%s", removed, removedPath)
	}
	if len(cfg.ProtectedPaths) != 0 {
		t.Fatalf("expected no protected paths, got %+v", cfg.ProtectedPaths)
	}
}

func TestAddAndRemoveProtectedFamilyNormalizesAndPersists(t *testing.T) {
	cfg, added, err := AddProtectedFamily(Default(), "Browser_Profiles")
	if err != nil {
		t.Fatal(err)
	}
	if added != "browser_profiles" {
		t.Fatalf("expected normalized family id, got %s", added)
	}
	if len(cfg.ProtectedFamilies) != 1 || cfg.ProtectedFamilies[0] != "browser_profiles" {
		t.Fatalf("expected protected family to be stored, got %+v", cfg.ProtectedFamilies)
	}
	cfg, removedFamily, removed, err := RemoveProtectedFamily(cfg, "browser_profiles")
	if err != nil {
		t.Fatal(err)
	}
	if !removed || removedFamily != "browser_profiles" {
		t.Fatalf("expected protected family to be removed, got removed=%v family=%s", removed, removedFamily)
	}
	if len(cfg.ProtectedFamilies) != 0 {
		t.Fatalf("expected no protected families, got %+v", cfg.ProtectedFamilies)
	}
}

func TestSaveAtNormalizesBeforeWriting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := Default()
	cfg.ProtectedPaths = []string{"~/Projects/keep-me"}
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := SaveAt(path, cfg); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "" {
		t.Fatal("expected config file to be written")
	}
	if expected := filepath.Join(home, "Projects", "keep-me"); !strings.Contains(string(raw), expected) {
		t.Fatalf("expected normalized protected path in config, got %s", string(raw))
	}
}

func TestNormalizeCommandExcludes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := Normalize(Config{
		CommandExcludes: map[string][]string{
			" Clean ":    {"~/Projects/keep", "~/Projects/keep"},
			"OPTIMIZE":   {"~/Library/Caches/Homebrew"},
			"   ":        {"~/ignored"},
			"purge scan": {"~/repo/vendor"},
		},
	})
	if len(cfg.CommandExcludes) != 3 {
		t.Fatalf("expected three normalized command scopes, got %+v", cfg.CommandExcludes)
	}
	if got := cfg.CommandExcludes["clean"]; len(got) != 1 || got[0] != filepath.Join(home, "Projects", "keep") {
		t.Fatalf("unexpected clean command excludes: %+v", got)
	}
	if _, ok := cfg.CommandExcludes["purge_scan"]; !ok {
		t.Fatalf("expected purge_scan normalization, got %+v", cfg.CommandExcludes)
	}
}

func slicesContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestAddAndRemoveCommandExclude(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg, command, path, err := AddCommandExclude(Default(), "clean", "~/Projects/keep")
	if err != nil {
		t.Fatal(err)
	}
	if command != "clean" {
		t.Fatalf("expected clean command, got %s", command)
	}
	expected := filepath.Join(home, "Projects", "keep")
	if path != expected {
		t.Fatalf("expected normalized path %s, got %s", expected, path)
	}
	if got := cfg.CommandExcludes["clean"]; len(got) != 1 || got[0] != expected {
		t.Fatalf("unexpected command excludes: %+v", cfg.CommandExcludes)
	}
	cfg, command, path, removed, err := RemoveCommandExclude(cfg, "clean", expected)
	if err != nil {
		t.Fatal(err)
	}
	if command != "clean" || path != expected || !removed {
		t.Fatalf("unexpected remove result command=%s path=%s removed=%v", command, path, removed)
	}
	if len(cfg.CommandExcludes) != 0 {
		t.Fatalf("expected command excludes to be empty, got %+v", cfg.CommandExcludes)
	}
}
