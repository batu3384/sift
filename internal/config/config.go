package config

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/batuhanyuksel/sift/internal/domain"
)

type DiagnosticsConfig struct {
	Redaction bool `toml:"redaction"`
}

type Config struct {
	Profiles          map[string][]string `toml:"profiles"`
	DisabledRules     []string            `toml:"disabled_rules"`
	ProtectedPaths    []string            `toml:"protected_paths"`
	ProtectedFamilies []string            `toml:"protected_families"`
	CommandExcludes   map[string][]string `toml:"command_excludes"`
	PurgeSearchPaths  []string            `toml:"purge_search_paths"`
	InteractionMode   string              `toml:"interaction_mode"`
	TrashMode         string              `toml:"trash_mode"`
	ConfirmLevel      string              `toml:"confirm_level"`
	Diagnostics       DiagnosticsConfig   `toml:"diagnostics"`
	// Whitelist paths to exclude from cleaning
	Whitelist []string `toml:"whitelist"`
	// Skip cache cleanup if cleaned within this many days
	SkipCacheDays int `toml:"skip_cache_days"`
	// Enable trash emptying
	EmptyTrash bool `toml:"empty_trash"`
	// Skip Xcode cleanup when Xcode is running
	SkipXcodeWhenRunning bool `toml:"skip_xcode_when_running"`
	// Protected web editor domains (Service Worker)
	ProtectedWebEditors []string `toml:"protected_web_editors"`
	// Enable sudo for system-level cleanup
	EnableSudo bool `toml:"enable_sudo"`
}

func Default() Config {
	return Config{
		Profiles: map[string][]string{
			"safe":      {"temp_files", "logs", "safe_system_clutter", "stale_login_items", "finder_metadata"},
			"developer": {"temp_files", "logs", "safe_system_clutter", "stale_login_items", "developer_caches", "package_manager_caches", "finder_metadata"},
			"deep":      {"temp_files", "logs", "safe_system_clutter", "recent_items", "stale_login_items", "orphaned_services", "system_update_artifacts", "developer_caches", "package_manager_caches", "browser_data", "installer_leftovers", "app_leftovers", "finder_metadata", "ios_device_backups", "time_machine_cleanup", "cloud_office", "virtualization", "device_backups", "time_machine", "maven_cache", "ipfs_node", "trash", "font_cache", "print_spooler", "xcode", "unity", "unreal", "android", "rust", "node_modules", "python_cache", "go_cache", "fonts", "diagnostics", "media_cache"},
		},
		InteractionMode:       "auto",
		TrashMode:             "trash_first",
		ConfirmLevel:          "strict",
		Diagnostics: DiagnosticsConfig{
			Redaction: true,
		},
		// Whitelist - paths to exclude from cleaning
		Whitelist: []string{
			"~/.config",
			"~/.ssh",
			"~/.gnupg",
		},
		// Skip cache cleanup if cleaned within 7 days
		SkipCacheDays: 7,
		// Don't empty trash by default
		EmptyTrash: false,
		// Skip Xcode cleanup when Xcode is running
		SkipXcodeWhenRunning: true,
		// Protected web editor domains (Service Worker)
		ProtectedWebEditors: []string{
			"capcut.com",
			"photopea.com",
			"pixlr.com",
			"canva.com",
			"figma.com",
		},
		// Don't enable sudo by default (requires explicit opt-in)
		EnableSudo: false,
	}
}

// ProfileCategoryCount returns the number of categories in a profile
func ProfileCategoryCount(profile string, cfg Config) int {
	if profile == "" {
		profile = "safe"
	}
	categories, ok := cfg.Profiles[profile]
	if !ok {
		return len(cfg.Profiles["safe"])
	}
	return len(categories)
}

func Dir() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "sift"), nil
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

func Load() (Config, error) {
	cfg := Default()
	path, err := Path()
	if err != nil {
		return cfg, err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, err
	}
	return Normalize(cfg), nil
}

func WriteDefaultIfMissing() (string, error) {
	cfgPath, err := Path()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(cfgPath); err == nil {
		return cfgPath, nil
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return "", err
	}
	if err := SaveAt(cfgPath, Default()); err != nil {
		return "", err
	}
	return cfgPath, nil
}

func Save(cfg Config) error {
	cfgPath, err := Path()
	if err != nil {
		return err
	}
	return SaveAt(cfgPath, cfg)
}

func SaveAt(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return toml.NewEncoder(file).Encode(Normalize(cfg))
}

func Normalize(cfg Config) Config {
	defaults := Default()
	if cfg.Profiles == nil {
		cfg.Profiles = map[string][]string{}
	}
	for name, rules := range defaults.Profiles {
		if len(cfg.Profiles[name]) == 0 {
			cfg.Profiles[name] = rules
			continue
		}
		cfg.Profiles[name] = mergeProfileRules(cfg.Profiles[name], rules)
	}
	cfg.DisabledRules = dedupe(cfg.DisabledRules)
	cfg.ProtectedPaths = normalizePaths(cfg.ProtectedPaths)
	cfg.ProtectedFamilies = dedupeLower(cfg.ProtectedFamilies)
	cfg.CommandExcludes = normalizeCommandExcludes(cfg.CommandExcludes)
	cfg.PurgeSearchPaths = normalizePaths(cfg.PurgeSearchPaths)
	if cfg.InteractionMode == "" {
		cfg.InteractionMode = defaults.InteractionMode
	}
	if cfg.TrashMode == "" {
		cfg.TrashMode = defaults.TrashMode
	}
	if cfg.ConfirmLevel == "" {
		cfg.ConfirmLevel = defaults.ConfirmLevel
	}
	return cfg
}

func mergeProfileRules(existing []string, defaults []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(existing)+len(defaults))
	for _, rule := range append(append([]string{}, existing...), defaults...) {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		if _, ok := seen[rule]; ok {
			continue
		}
		seen[rule] = struct{}{}
		out = append(out, rule)
	}
	return out
}

func AddProtectedPath(cfg Config, path string) (Config, string, error) {
	normalized, err := normalizePath(path)
	if err != nil {
		return cfg, "", err
	}
	cfg.ProtectedPaths = append(cfg.ProtectedPaths, normalized)
	cfg = Normalize(cfg)
	return cfg, normalized, nil
}

func AddProtectedFamily(cfg Config, family string) (Config, string, error) {
	normalized := strings.ToLower(strings.TrimSpace(family))
	if normalized == "" {
		return cfg, "", errors.New("family is required")
	}
	cfg.ProtectedFamilies = append(cfg.ProtectedFamilies, normalized)
	cfg = Normalize(cfg)
	return cfg, normalized, nil
}

func RemoveProtectedPath(cfg Config, path string) (Config, string, bool, error) {
	normalized, err := normalizePath(path)
	if err != nil {
		return cfg, "", false, err
	}
	out := make([]string, 0, len(cfg.ProtectedPaths))
	removed := false
	for _, candidate := range Normalize(cfg).ProtectedPaths {
		if candidate == normalized {
			removed = true
			continue
		}
		out = append(out, candidate)
	}
	cfg.ProtectedPaths = out
	cfg = Normalize(cfg)
	return cfg, normalized, removed, nil
}

func RemoveProtectedFamily(cfg Config, family string) (Config, string, bool, error) {
	normalized := strings.ToLower(strings.TrimSpace(family))
	if normalized == "" {
		return cfg, "", false, errors.New("family is required")
	}
	out := make([]string, 0, len(cfg.ProtectedFamilies))
	removed := false
	for _, candidate := range Normalize(cfg).ProtectedFamilies {
		if candidate == normalized {
			removed = true
			continue
		}
		out = append(out, candidate)
	}
	cfg.ProtectedFamilies = out
	cfg = Normalize(cfg)
	return cfg, normalized, removed, nil
}

func AddCommandExclude(cfg Config, command, path string) (Config, string, string, error) {
	normalizedCommand := NormalizeCommandName(command)
	if normalizedCommand == "" {
		return cfg, "", "", errors.New("command is required")
	}
	normalizedPath, err := normalizePath(path)
	if err != nil {
		return cfg, "", "", err
	}
	cfg = Normalize(cfg)
	if cfg.CommandExcludes == nil {
		cfg.CommandExcludes = map[string][]string{}
	}
	cfg.CommandExcludes[normalizedCommand] = append(cfg.CommandExcludes[normalizedCommand], normalizedPath)
	cfg = Normalize(cfg)
	return cfg, normalizedCommand, normalizedPath, nil
}

func RemoveCommandExclude(cfg Config, command, path string) (Config, string, string, bool, error) {
	normalizedCommand := NormalizeCommandName(command)
	if normalizedCommand == "" {
		return cfg, "", "", false, errors.New("command is required")
	}
	normalizedPath, err := normalizePath(path)
	if err != nil {
		return cfg, "", "", false, err
	}
	cfg = Normalize(cfg)
	values := cfg.CommandExcludes[normalizedCommand]
	if len(values) == 0 {
		return cfg, normalizedCommand, normalizedPath, false, nil
	}
	out := make([]string, 0, len(values))
	removed := false
	for _, candidate := range values {
		if candidate == normalizedPath {
			removed = true
			continue
		}
		out = append(out, candidate)
	}
	if len(out) == 0 {
		delete(cfg.CommandExcludes, normalizedCommand)
	} else {
		cfg.CommandExcludes[normalizedCommand] = out
	}
	cfg = Normalize(cfg)
	return cfg, normalizedCommand, normalizedPath, removed, nil
}

func Validate(cfg Config) []string {
	var warnings []string
	for _, name := range []string{"safe", "developer", "deep"} {
		if len(cfg.Profiles[name]) == 0 {
			warnings = append(warnings, "missing profile "+name)
		}
	}
	if !slices.Contains([]string{"auto", "plain", "tui"}, cfg.InteractionMode) {
		warnings = append(warnings, "unknown interaction_mode "+cfg.InteractionMode)
	}
	if !slices.Contains([]string{"trash_first", "permanent"}, cfg.TrashMode) {
		warnings = append(warnings, "unknown trash_mode "+cfg.TrashMode)
	}
	if !slices.Contains([]string{"strict", "balanced"}, cfg.ConfirmLevel) {
		warnings = append(warnings, "unknown confirm_level "+cfg.ConfirmLevel)
	}
	for command := range cfg.CommandExcludes {
		if NormalizeCommandName(command) == "" {
			warnings = append(warnings, "unknown command_excludes key "+command)
		}
	}
	return warnings
}

func normalizeCommandExcludes(values map[string][]string) map[string][]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string][]string, len(values))
	for command, paths := range values {
		normalizedCommand := NormalizeCommandName(command)
		if normalizedCommand == "" {
			continue
		}
		normalizedPaths := normalizePaths(paths)
		if len(normalizedPaths) == 0 {
			continue
		}
		out[normalizedCommand] = normalizedPaths
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func NormalizeCommandName(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return ""
	}
	normalized = strings.ReplaceAll(normalized, " ", "_")
	return normalized
}

func normalizePaths(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized, err := normalizePath(value)
		if err != nil || normalized == "" {
			continue
		}
		out = append(out, normalized)
	}
	return dedupe(out)
}

func normalizePath(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if trimmed == "~" {
			trimmed = home
		} else {
			trimmed = filepath.Join(home, trimmed[2:])
		}
	}
	return domain.NormalizePath(trimmed), nil
}

func dedupe(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func dedupeLower(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}
