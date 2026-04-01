package engine

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type TouchIDStatus struct {
	Supported          bool     `json:"supported"`
	Enabled            bool     `json:"enabled"`
	PAMPath            string   `json:"pam_path,omitempty"`
	LocalPAMPath       string   `json:"local_pam_path,omitempty"`
	ActivePAMPath      string   `json:"active_pam_path,omitempty"`
	BackupPath         string   `json:"backup_path,omitempty"`
	SudoLocalSupported bool     `json:"sudo_local_supported,omitempty"`
	LegacyConfigured   bool     `json:"legacy_configured,omitempty"`
	MigrationNeeded    bool     `json:"migration_needed,omitempty"`
	Message            string   `json:"message"`
	Commands           []string `json:"commands,omitempty"`
}

type TouchIDResult struct {
	Action             string   `json:"action"`
	Supported          bool     `json:"supported"`
	Enabled            bool     `json:"enabled"`
	DesiredEnabled     bool     `json:"desired_enabled"`
	DryRun             bool     `json:"dry_run"`
	Changed            bool     `json:"changed"`
	PAMPath            string   `json:"pam_path,omitempty"`
	LocalPAMPath       string   `json:"local_pam_path,omitempty"`
	ActivePAMPath      string   `json:"active_pam_path,omitempty"`
	BackupPath         string   `json:"backup_path,omitempty"`
	SudoLocalSupported bool     `json:"sudo_local_supported,omitempty"`
	LegacyConfigured   bool     `json:"legacy_configured,omitempty"`
	MigrationNeeded    bool     `json:"migration_needed,omitempty"`
	Migrated           bool     `json:"migrated,omitempty"`
	Message            string   `json:"message"`
	Commands           []string `json:"commands,omitempty"`
}

func (s *Service) TouchIDStatus() TouchIDStatus {
	if runtime.GOOS != "darwin" {
		return TouchIDStatus{
			Supported: false,
			Message:   "Touch ID sudo integration is only available on macOS.",
		}
	}
	state, err := s.readTouchIDState()
	if err != nil {
		return TouchIDStatus{
			Supported:          true,
			PAMPath:            state.pamPath,
			LocalPAMPath:       state.localPath,
			ActivePAMPath:      state.activePath(),
			BackupPath:         touchIDBackupPath(state.targetPath()),
			SudoLocalSupported: state.sudoLocalSupported,
			Message:            err.Error(),
		}
	}
	return TouchIDStatus{
		Supported:          true,
		Enabled:            state.enabled(),
		PAMPath:            state.pamPath,
		LocalPAMPath:       state.localPath,
		ActivePAMPath:      state.activePath(),
		BackupPath:         touchIDBackupPath(state.targetPath()),
		SudoLocalSupported: state.sudoLocalSupported,
		LegacyConfigured:   state.legacyEnabled,
		MigrationNeeded:    state.migrationNeeded(),
		Message:            touchIDStatusMessage(state),
		Commands:           touchIDCommands(state, !state.enabled() || state.migrationNeeded()),
	}
}

func (s *Service) ConfigureTouchID(enable, dryRun bool) (TouchIDResult, error) {
	status := s.TouchIDStatus()
	result := TouchIDResult{
		Action:             "disable",
		Supported:          status.Supported,
		Enabled:            status.Enabled,
		DesiredEnabled:     enable,
		DryRun:             dryRun,
		PAMPath:            status.PAMPath,
		LocalPAMPath:       status.LocalPAMPath,
		ActivePAMPath:      status.ActivePAMPath,
		BackupPath:         status.BackupPath,
		SudoLocalSupported: status.SudoLocalSupported,
		LegacyConfigured:   status.LegacyConfigured,
		MigrationNeeded:    status.MigrationNeeded,
	}
	if enable {
		result.Action = "enable"
	}
	if !status.Supported {
		result.Message = status.Message
		return result, nil
	}
	if status.PAMPath == "" {
		result.Message = "Touch ID PAM file could not be resolved."
		return result, errors.New(result.Message)
	}
	if enable == status.Enabled {
		if !status.MigrationNeeded || !enable {
			result.Message = touchIDNoopMessage(enable)
			result.Commands = status.Commands
			return result, nil
		}
	}
	state, err := s.readTouchIDState()
	if err != nil {
		result.Message = err.Error()
		return result, err
	}
	result.ActivePAMPath = state.activePath()
	result.BackupPath = touchIDBackupPath(state.targetPath())
	result.Commands = touchIDCommands(state, enable)

	targetPath := state.targetPath()
	targetRaw := state.targetRaw()
	nextTarget, targetChanged := renderTouchIDTarget(targetPath, targetRaw, enable)
	mainNext, mainChanged := touchIDRenderLegacyCleanup(state, enable)
	if !targetChanged && !mainChanged {
		result.Message = touchIDNoopMessage(enable)
		result.Enabled = enable
		return result, nil
	}
	if dryRun {
		result.Message = touchIDPreviewMessage(enable, state)
		return result, nil
	}
	if targetChanged {
		if err := s.writeTouchIDPath(targetPath, targetRaw, nextTarget); err != nil {
			result.Message = err.Error()
			return result, err
		}
	}
	if mainChanged {
		if err := s.writeTouchIDPath(state.pamPath, state.mainRaw, mainNext); err != nil {
			result.Message = err.Error()
			return result, err
		}
	}
	result.Migrated = enable && state.sudoLocalSupported && state.legacyEnabled
	result.Enabled = enable
	result.Changed = targetChanged || mainChanged
	result.Message = touchIDAppliedMessage(enable, state, result.Migrated)
	return result, nil
}

func (s *Service) touchIDPAMPath() string {
	if strings.TrimSpace(s.TouchIDPAMPath) != "" {
		return s.TouchIDPAMPath
	}
	return "/etc/pam.d/sudo"
}

func (s *Service) touchIDLocalPAMPath() string {
	if strings.TrimSpace(s.TouchIDLocalPAMPath) != "" {
		return s.TouchIDLocalPAMPath
	}
	return filepath.Join(filepath.Dir(s.touchIDPAMPath()), "sudo_local")
}

type touchIDState struct {
	pamPath            string
	localPath          string
	mainRaw            []byte
	localRaw           []byte
	localExists        bool
	sudoLocalSupported bool
	legacyEnabled      bool
	localEnabled       bool
}

func (s *Service) readTouchIDState() (touchIDState, error) {
	state := touchIDState{
		pamPath:   s.touchIDPAMPath(),
		localPath: s.touchIDLocalPAMPath(),
	}
	raw, err := s.readTextFile(state.pamPath)
	if err != nil {
		return state, err
	}
	state.mainRaw = raw
	state.sudoLocalSupported = touchIDSudoLocalSupported(raw)
	state.legacyEnabled = touchIDConfiguredIn(raw)
	localRaw, err := s.readTextFile(state.localPath)
	switch {
	case err == nil:
		state.localExists = true
		state.localRaw = localRaw
		state.localEnabled = touchIDConfiguredIn(localRaw)
	case errors.Is(err, os.ErrNotExist):
	default:
		return state, err
	}
	return state, nil
}

func (s *Service) readTextFile(path string) ([]byte, error) {
	if s.ReadFile != nil {
		return s.ReadFile(path)
	}
	return os.ReadFile(path)
}

func (s *Service) writeTextFile(path string, content []byte, perm os.FileMode) error {
	if s.WriteFile != nil {
		return s.WriteFile(path, content, perm)
	}
	return os.WriteFile(path, content, perm)
}

func touchIDBackupPath(pamPath string) string {
	return pamPath + ".backup.sift"
}

func touchIDConfiguredIn(raw []byte) bool {
	return strings.Contains(string(raw), "pam_tid.so")
}

func touchIDSudoLocalSupported(raw []byte) bool {
	return strings.Contains(string(raw), "sudo_local")
}

func (state touchIDState) enabled() bool {
	return state.localEnabled || state.legacyEnabled
}

func (state touchIDState) migrationNeeded() bool {
	return state.sudoLocalSupported && state.legacyEnabled
}

func (state touchIDState) activePath() string {
	switch {
	case state.localEnabled:
		return state.localPath
	case state.legacyEnabled:
		return state.pamPath
	case state.sudoLocalSupported:
		return state.localPath
	default:
		return state.pamPath
	}
}

func (state touchIDState) targetPath() string {
	if state.sudoLocalSupported {
		return state.localPath
	}
	return state.pamPath
}

func (state touchIDState) targetRaw() []byte {
	if state.sudoLocalSupported {
		if state.localExists {
			return state.localRaw
		}
		return touchIDDefaultLocalPAM()
	}
	return state.mainRaw
}

func touchIDCommands(state touchIDState, enable bool) []string {
	target := state.targetPath()
	backup := touchIDBackupPath(target)
	if enable {
		commands := []string{}
		if state.sudoLocalSupported {
			if state.localExists {
				commands = append(commands, fmt.Sprintf("sudo cp %s %s", target, backup))
			}
			commands = append(commands,
				fmt.Sprintf("sudo sh -c 'printf \"# sudo_local: local customizations for sudo\\nauth       sufficient     pam_tid.so\\n\" > %s'", state.localPath),
			)
			if state.legacyEnabled {
				commands = append(commands,
					fmt.Sprintf("sudo cp %s %s", state.pamPath, touchIDBackupPath(state.pamPath)),
					fmt.Sprintf("sudo sed -i '' '/pam_tid\\.so/d' %s", state.pamPath),
				)
			}
			return commands
		}
		commands = append(commands, fmt.Sprintf("sudo cp %s %s", target, backup))
		commands = append(commands, fmt.Sprintf("sudo sh -c 'printf \"auth       sufficient     pam_tid.so\\n\" > /tmp/sift-sudo && cat %s >> /tmp/sift-sudo && mv /tmp/sift-sudo %s'", state.pamPath, state.pamPath))
		return commands
	}
	commands := []string{}
	if state.sudoLocalSupported {
		if state.localExists {
			commands = append(commands, fmt.Sprintf("sudo cp %s %s", target, backup))
			commands = append(commands, fmt.Sprintf("sudo sed -i '' '/pam_tid\\.so/d' %s", state.localPath))
		}
		if state.legacyEnabled {
			commands = append(commands,
				fmt.Sprintf("sudo cp %s %s", state.pamPath, touchIDBackupPath(state.pamPath)),
				fmt.Sprintf("sudo sed -i '' '/pam_tid\\.so/d' %s", state.pamPath),
			)
		}
		return commands
	}
	return []string{
		fmt.Sprintf("sudo cp %s %s", target, backup),
		fmt.Sprintf("sudo sed -i '' '/pam_tid\\.so/d' %s", state.pamPath),
	}
}

func renderTouchIDPAM(raw []byte, enable bool) ([]byte, bool) {
	normalized := strings.ReplaceAll(string(raw), "\r\n", "\n")
	lines := strings.Split(strings.TrimSuffix(normalized, "\n"), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, "pam_tid.so") {
			continue
		}
		if line == "" && len(lines) == 1 {
			continue
		}
		filtered = append(filtered, line)
	}
	if enable {
		filtered = append([]string{"auth       sufficient     pam_tid.so"}, filtered...)
	}
	rendered := strings.Join(filtered, "\n")
	if rendered != "" || strings.HasSuffix(normalized, "\n") {
		rendered += "\n"
	}
	return []byte(rendered), rendered != normalized
}

func renderTouchIDTarget(path string, raw []byte, enable bool) ([]byte, bool) {
	if strings.HasSuffix(path, "sudo_local") && len(bytes.TrimSpace(raw)) == 0 {
		raw = touchIDDefaultLocalPAM()
	}
	return renderTouchIDPAM(raw, enable)
}

func touchIDRenderLegacyCleanup(state touchIDState, enable bool) ([]byte, bool) {
	if !state.sudoLocalSupported || !state.legacyEnabled {
		return nil, false
	}
	if !enable && !state.localEnabled {
		return renderTouchIDPAM(state.mainRaw, false)
	}
	return renderTouchIDPAM(state.mainRaw, false)
}

func touchIDDefaultLocalPAM() []byte {
	return []byte("# sudo_local: local customizations for sudo\n")
}

func touchIDStatusMessage(state touchIDState) string {
	switch {
	case state.localEnabled && state.legacyEnabled:
		return "Touch ID is enabled via sudo_local. Legacy sudo PAM cleanup is still pending."
	case state.localEnabled:
		return "Touch ID is enabled for sudo via sudo_local."
	case state.legacyEnabled && state.sudoLocalSupported:
		return "Touch ID is enabled via the legacy sudo PAM file. Migration to sudo_local is recommended."
	case state.legacyEnabled:
		return "Touch ID is enabled for sudo."
	case state.sudoLocalSupported:
		return "Touch ID is not enabled for sudo. sudo_local is available for a safer setup."
	default:
		return "Touch ID is not enabled for sudo."
	}
}

func touchIDPreviewMessage(enable bool, state touchIDState) string {
	if enable {
		if state.sudoLocalSupported && state.legacyEnabled {
			return "Touch ID enable preview is ready. Re-run with --dry-run=false --yes to migrate sudo PAM into sudo_local."
		}
		return "Touch ID enable preview is ready. Re-run with --dry-run=false --yes to apply."
	}
	return "Touch ID disable preview is ready. Re-run with --dry-run=false --yes to apply."
}

func touchIDAppliedMessage(enable bool, state touchIDState, migrated bool) string {
	if enable {
		if migrated {
			return "Touch ID is enabled for sudo via sudo_local. Legacy sudo PAM entries were cleaned up."
		}
		if state.sudoLocalSupported {
			return "Touch ID is enabled for sudo via sudo_local."
		}
		return "Touch ID is enabled for sudo."
	}
	return "Touch ID is disabled for sudo."
}

func touchIDNoopMessage(enable bool) string {
	if enable {
		return "Touch ID is already enabled for sudo."
	}
	return "Touch ID is already disabled for sudo."
}

func (s *Service) writeTouchIDPath(path string, previous []byte, next []byte) error {
	mode := os.FileMode(0o444)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode()
		if err := s.writeTextFile(touchIDBackupPath(path), previous, mode); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if s.WriteFile != nil {
		return s.writeTextFile(path, next, mode)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".sift-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempPath)
		return err
	}
	if _, err := temp.Write(next); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempPath)
		return err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}
