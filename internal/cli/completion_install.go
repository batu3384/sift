package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/batu3384/sift/internal/domain"
	"github.com/spf13/cobra"
)

type completionInstallResult struct {
	Shell          string `json:"shell"`
	CompletionPath string `json:"completion_path"`
	ConfigPath     string `json:"config_path,omitempty"`
	Message        string `json:"message"`
}

type completionInstallTarget struct {
	shell          string
	completionPath string
	configPath     string
	sourceBlock    string
}

func installCompletion(root *cobra.Command, shell, completionDir, shellConfig string) (completionInstallResult, error) {
	target, err := completionTargetForShell(shell, completionDir, shellConfig)
	if err != nil {
		return completionInstallResult{}, err
	}
	content, err := completionContent(root, target.shell)
	if err != nil {
		return completionInstallResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(target.completionPath), 0o755); err != nil {
		return completionInstallResult{}, err
	}
	if err := os.WriteFile(target.completionPath, content, 0o644); err != nil {
		return completionInstallResult{}, err
	}
	if strings.TrimSpace(target.sourceBlock) != "" {
		if err := ensureConfigBlock(target.configPath, target.sourceBlock); err != nil {
			return completionInstallResult{}, err
		}
	}
	message := fmt.Sprintf("Installed %s completion to %s", target.shell, target.completionPath)
	if target.configPath != "" && target.sourceBlock != "" {
		message += fmt.Sprintf(" and wired %s", target.configPath)
	}
	return completionInstallResult{
		Shell:          target.shell,
		CompletionPath: target.completionPath,
		ConfigPath:     target.configPath,
		Message:        message,
	}, nil
}

func detectShellName(shellEnv string) string {
	shell := strings.TrimSpace(shellEnv)
	if shell == "" {
		return ""
	}
	base := strings.ToLower(filepath.Base(shell))
	switch base {
	case "bash", "zsh", "fish", "powershell", "pwsh":
		if base == "pwsh" {
			return "powershell"
		}
		return base
	default:
		return ""
	}
}

func completionTargetForShell(shell, completionDir, shellConfig string) (completionInstallTarget, error) {
	shell = detectShellName(shell)
	if shell == "" {
		return completionInstallTarget{}, fmt.Errorf("unsupported shell; pass one of bash, zsh, fish, powershell")
	}
	home, err := domain.CurrentHomeDir()
	if err != nil {
		return completionInstallTarget{}, err
	}
	switch shell {
	case "bash":
		dir := completionDir
		if dir == "" {
			dir = filepath.Join(home, ".local", "share", "bash-completion", "completions")
		}
		configPath := shellConfig
		if configPath == "" {
			configPath = filepath.Join(home, ".bashrc")
		}
		completionPath := filepath.Join(dir, "sift")
		return completionInstallTarget{
			shell:          shell,
			completionPath: completionPath,
			configPath:     configPath,
			sourceBlock:    blockForShell(shell, completionPath),
		}, nil
	case "zsh":
		dir := completionDir
		if dir == "" {
			dir = filepath.Join(home, ".zfunc")
		}
		configPath := shellConfig
		if configPath == "" {
			configPath = filepath.Join(home, ".zshrc")
		}
		completionPath := filepath.Join(dir, "_sift")
		return completionInstallTarget{
			shell:          shell,
			completionPath: completionPath,
			configPath:     configPath,
			sourceBlock:    blockForShell(shell, dir),
		}, nil
	case "fish":
		dir := completionDir
		if dir == "" {
			dir = filepath.Join(home, ".config", "fish", "completions")
		}
		return completionInstallTarget{
			shell:          shell,
			completionPath: filepath.Join(dir, "sift.fish"),
		}, nil
	case "powershell":
		dir := completionDir
		if dir == "" {
			dir = filepath.Join(home, "Documents", "PowerShell", "Completions")
		}
		configPath := shellConfig
		if configPath == "" {
			configPath = filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
		}
		completionPath := filepath.Join(dir, "sift.ps1")
		return completionInstallTarget{
			shell:          shell,
			completionPath: completionPath,
			configPath:     configPath,
			sourceBlock:    blockForShell(shell, completionPath),
		}, nil
	default:
		return completionInstallTarget{}, fmt.Errorf("unsupported shell %q", shell)
	}
}

func completionContent(root *cobra.Command, shell string) ([]byte, error) {
	var buf bytes.Buffer
	switch shell {
	case "bash":
		if err := root.GenBashCompletion(&buf); err != nil {
			return nil, err
		}
	case "zsh":
		if err := root.GenZshCompletion(&buf); err != nil {
			return nil, err
		}
	case "fish":
		if err := root.GenFishCompletion(&buf, true); err != nil {
			return nil, err
		}
	case "powershell":
		if err := root.GenPowerShellCompletionWithDesc(&buf); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported shell %q", shell)
	}
	return buf.Bytes(), nil
}

func blockForShell(shell, value string) string {
	switch shell {
	case "bash":
		return fmt.Sprintf("source %q", value)
	case "zsh":
		return strings.Join([]string{
			fmt.Sprintf("fpath=(%q $fpath)", value),
			"autoload -Uz compinit",
			"compinit",
		}, "\n")
	case "powershell":
		return fmt.Sprintf(". %q", value)
	default:
		return ""
	}
}

func ensureConfigBlock(path, block string) error {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(block) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	const start = "# >>> sift completion >>>"
	const end = "# <<< sift completion <<<"
	body := start + "\n" + block + "\n" + end + "\n"
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	text := string(raw)
	if strings.Contains(text, start) && strings.Contains(text, end) {
		prefix, _, found := strings.Cut(text, start)
		if !found {
			return nil
		}
		_, suffix, found := strings.Cut(text, end)
		if !found {
			return nil
		}
		text = strings.TrimRight(prefix, "\n") + "\n" + body + strings.TrimLeft(suffix, "\n")
	} else {
		if strings.TrimSpace(text) != "" && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text += body
	}
	return os.WriteFile(path, []byte(text), 0o644)
}
