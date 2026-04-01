package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type UpdateGuidance struct {
	CurrentVersion string   `json:"current_version"`
	Platform       string   `json:"platform"`
	InstallMethod  string   `json:"install_method"`
	Channel        string   `json:"channel"`
	Force          bool     `json:"force"`
	Message        string   `json:"message"`
	Commands       []string `json:"commands,omitempty"`
}

type UpdateResult struct {
	CurrentVersion string   `json:"current_version"`
	Platform       string   `json:"platform"`
	InstallMethod  string   `json:"install_method"`
	Channel        string   `json:"channel"`
	Force          bool     `json:"force"`
	DryRun         bool     `json:"dry_run"`
	Changed        bool     `json:"changed"`
	Executable     string   `json:"executable,omitempty"`
	Message        string   `json:"message"`
	Commands       []string `json:"commands,omitempty"`
}

type UpdateNotice struct {
	CurrentVersion string    `json:"current_version"`
	LatestVersion  string    `json:"latest_version,omitempty"`
	InstallMethod  string    `json:"install_method"`
	Available      bool      `json:"available"`
	CheckedAt      time.Time `json:"checked_at,omitempty"`
	Message        string    `json:"message"`
	Commands       []string  `json:"commands,omitempty"`
}

type UpdateChannel string

const (
	UpdateChannelStable  UpdateChannel = "stable"
	UpdateChannelNightly UpdateChannel = "nightly"
)

type UpdateOptions struct {
	Channel UpdateChannel
	Force   bool
}

const updateNoticeTTL = 6 * time.Hour

var updateNoticeCacheDir = os.UserCacheDir

var fetchLatestReleaseTag = func(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/batu3384/sift/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "sift-update-check")
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("update check returned %s", resp.Status)
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if strings.TrimSpace(payload.TagName) == "" {
		return "", errors.New("latest release tag is empty")
	}
	return payload.TagName, nil
}

func NormalizeUpdateChannel(value string) UpdateChannel {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(UpdateChannelStable):
		return UpdateChannelStable
	case string(UpdateChannelNightly):
		return UpdateChannelNightly
	default:
		return UpdateChannelStable
	}
}

func (s *Service) UpdateGuidance() UpdateGuidance {
	return s.UpdateGuidanceForOptions(UpdateOptions{Channel: UpdateChannelStable})
}

func (s *Service) UpdateNotice(ctx context.Context) UpdateNotice {
	guidance := s.UpdateGuidance()
	notice := UpdateNotice{
		CurrentVersion: guidance.CurrentVersion,
		InstallMethod:  guidance.InstallMethod,
		Message:        "No update notice available.",
		Commands:       guidance.Commands,
	}
	if guidance.CurrentVersion == "dev" {
		notice.Message = "Local development build detected. Background release checks are skipped."
		return notice
	}
	if cached, ok := loadCachedUpdateNotice(guidance.CurrentVersion, guidance.InstallMethod); ok {
		return cached
	}
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	latest, err := fetchLatestReleaseTag(checkCtx)
	if err != nil {
		notice.Message = "Update check unavailable right now."
		return notice
	}
	notice.CheckedAt = time.Now().UTC()
	notice.LatestVersion = normalizeReleaseTag(latest)
	switch compareReleaseVersions(notice.LatestVersion, guidance.CurrentVersion) {
	case 1:
		notice.Available = true
		notice.Message = fmt.Sprintf("Update %s available. Run `sift update` to review the upgrade path.", notice.LatestVersion)
	default:
		notice.Message = fmt.Sprintf("Running latest release (%s).", normalizeReleaseTag(guidance.CurrentVersion))
	}
	_ = saveCachedUpdateNotice(notice)
	return notice
}

func (s *Service) UpdateGuidanceForOptions(opts UpdateOptions) UpdateGuidance {
	channel := NormalizeUpdateChannel(string(opts.Channel))
	method, commands := s.installMethodAndCommands(channel, opts.Force)
	version := currentVersion()
	message := "Review the suggested command for your install method."
	if version == "dev" {
		message = "This build looks like a local development binary. Rebuild from source or download a tagged release."
	}
	if channel == UpdateChannelNightly {
		message = "Review the suggested nightly update command for your install method."
	}
	if opts.Force {
		message = "Review the forced reinstall command for your install method."
	}
	return UpdateGuidance{
		CurrentVersion: version,
		Platform:       s.Adapter.Name(),
		InstallMethod:  method,
		Channel:        string(channel),
		Force:          opts.Force,
		Message:        message,
		Commands:       commands,
	}
}

func loadCachedUpdateNotice(currentVersion, installMethod string) (UpdateNotice, bool) {
	cacheDir, err := updateNoticeCacheDir()
	if err != nil {
		return UpdateNotice{}, false
	}
	path := filepath.Join(cacheDir, "sift", "update_notice.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return UpdateNotice{}, false
	}
	var notice UpdateNotice
	if err := json.Unmarshal(raw, &notice); err != nil {
		return UpdateNotice{}, false
	}
	if notice.CurrentVersion != currentVersion || notice.InstallMethod != installMethod {
		return UpdateNotice{}, false
	}
	if notice.CheckedAt.IsZero() || time.Since(notice.CheckedAt) > updateNoticeTTL {
		return UpdateNotice{}, false
	}
	return notice, true
}

func saveCachedUpdateNotice(notice UpdateNotice) error {
	cacheDir, err := updateNoticeCacheDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(cacheDir, "sift")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(notice, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "update_notice.json"), raw, 0o644)
}

func normalizeReleaseTag(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if strings.HasPrefix(strings.ToLower(value), "v") {
		return "v" + strings.TrimPrefix(strings.TrimPrefix(value, "v"), "V")
	}
	return "v" + value
}

func compareReleaseVersions(left, right string) int {
	parse := func(value string) []int {
		trimmed := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(value), "v"), "V")
		parts := strings.Split(trimmed, ".")
		out := make([]int, 0, len(parts))
		for _, part := range parts {
			digits := part
			for i, r := range part {
				if r < '0' || r > '9' {
					digits = part[:i]
					break
				}
			}
			if digits == "" {
				out = append(out, 0)
				continue
			}
			n, err := strconv.Atoi(digits)
			if err != nil {
				out = append(out, 0)
				continue
			}
			out = append(out, n)
		}
		return out
	}
	a := parse(left)
	b := parse(right)
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	for len(a) < maxLen {
		a = append(a, 0)
	}
	for len(b) < maxLen {
		b = append(b, 0)
	}
	for i := 0; i < maxLen; i++ {
		switch {
		case a[i] > b[i]:
			return 1
		case a[i] < b[i]:
			return -1
		}
	}
	return 0
}

func (s *Service) RunUpdate(ctx context.Context, dryRun bool) (UpdateResult, error) {
	return s.RunUpdateWithOptions(ctx, dryRun, UpdateOptions{Channel: UpdateChannelStable})
}

func (s *Service) RunUpdateWithOptions(ctx context.Context, dryRun bool, opts UpdateOptions) (UpdateResult, error) {
	channel := NormalizeUpdateChannel(string(opts.Channel))
	guidance := s.UpdateGuidanceForOptions(UpdateOptions{Channel: channel, Force: opts.Force})
	result := UpdateResult{
		CurrentVersion: guidance.CurrentVersion,
		Platform:       guidance.Platform,
		InstallMethod:  guidance.InstallMethod,
		Channel:        guidance.Channel,
		Force:          guidance.Force,
		DryRun:         dryRun,
		Message:        guidance.Message,
		Commands:       guidance.Commands,
	}
	command, ok := updateCommandForMethod(guidance.InstallMethod, channel, opts.Force)
	if !ok {
		return result, nil
	}
	executable, err := s.resolveExecutable(command.Name)
	if err != nil {
		result.Message = err.Error()
		if dryRun {
			return result, nil
		}
		return result, err
	}
	result.Executable = executable
	if dryRun {
		prefix := "Update"
		if channel == UpdateChannelNightly {
			prefix = "Nightly update"
		}
		if opts.Force {
			prefix = "Forced reinstall"
		}
		result.Message = fmt.Sprintf("%s preview is ready. Re-run with --dry-run=false --yes to apply.", prefix)
		return result, nil
	}
	if err := s.execCommand(ctx, executable, command.Args...); err != nil {
		result.Message = err.Error()
		return result, err
	}
	result.Changed = true
	result.Message = "Update command completed."
	if channel == UpdateChannelNightly {
		result.Message = "Nightly update command completed."
	}
	if opts.Force {
		result.Message = "Forced reinstall command completed."
	}
	return result, nil
}

func (s *Service) installMethodAndCommands(channel UpdateChannel, force bool) (string, []string) {
	executable, _ := s.currentExecutable()
	lower := strings.ToLower(filepath.Clean(executable))
	switch {
	case strings.Contains(lower, "cellar") || strings.Contains(lower, "homebrew"):
		commands := []string{"brew upgrade sift", "brew uninstall sift"}
		if force {
			commands[0] = "brew reinstall sift"
		}
		if channel == UpdateChannelNightly {
			commands = []string{"Nightly builds are not available for Homebrew installs", "Switch to a manual install to use nightly builds"}
		}
		return "homebrew", commands
	case strings.Contains(lower, string(filepath.Separator)+"scoop"+string(filepath.Separator)):
		commands := []string{"scoop update sift", "scoop uninstall sift"}
		if force {
			commands[0] = "scoop uninstall sift && scoop install sift"
		}
		if channel == UpdateChannelNightly {
			commands = []string{"Nightly builds are not available for Scoop installs", "Switch to a manual install to use nightly builds"}
		}
		return "scoop", commands
	case runtime.GOOS == "windows":
		commands := []string{"winget upgrade SIFT", "winget uninstall SIFT"}
		if force {
			commands[0] = "winget uninstall SIFT && winget install SIFT"
		}
		if channel == UpdateChannelNightly {
			commands = []string{"Nightly builds are not available for Winget installs", "Switch to a manual install to use nightly builds"}
		}
		return "winget", commands
	default:
		ref := "@latest"
		if channel == UpdateChannelNightly {
			ref = "@main"
		}
		return "manual", []string{
			"go install github.com/batu3384/sift/cmd/sift" + ref,
			"Delete the installed sift binary after state cleanup",
		}
	}
}

func updateCommandForMethod(method string, channel UpdateChannel, force bool) (managedCommand, bool) {
	switch method {
	case "homebrew":
		if channel == UpdateChannelNightly {
			return managedCommand{}, false
		}
		if force {
			return managedCommand{Name: "brew", Args: []string{"reinstall", "sift"}}, true
		}
		return managedCommand{Name: "brew", Args: []string{"upgrade", "sift"}}, true
	case "scoop":
		if channel == UpdateChannelNightly {
			return managedCommand{}, false
		}
		if force {
			return managedCommand{Name: "powershell", Args: []string{"-NoProfile", "-Command", "scoop uninstall sift; scoop install sift"}}, true
		}
		return managedCommand{Name: "scoop", Args: []string{"update", "sift"}}, true
	case "winget":
		if channel == UpdateChannelNightly {
			return managedCommand{}, false
		}
		if force {
			return managedCommand{Name: "winget", Args: []string{"install", "--id", "SIFT", "--force"}}, true
		}
		return managedCommand{Name: "winget", Args: []string{"upgrade", "SIFT"}}, true
	case "manual":
		ref := "github.com/batu3384/sift/cmd/sift@latest"
		if channel == UpdateChannelNightly {
			ref = "github.com/batu3384/sift/cmd/sift@main"
		}
		return managedCommand{Name: "go", Args: []string{"install", ref}}, true
	default:
		return managedCommand{}, false
	}
}
