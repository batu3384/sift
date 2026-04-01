package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/batu3384/sift/internal/domain"
)

type versionInfo struct {
	Version         string `json:"version"`
	Platform        string `json:"platform"`
	Arch            string `json:"arch"`
	Executable      string `json:"executable,omitempty"`
	InstallMethod   string `json:"install_method"`
	Channel         string `json:"channel"`
	Shell           string `json:"shell,omitempty"`
	InteractionMode string `json:"interaction_mode"`
	DiskFree        string `json:"disk_free,omitempty"`
	SIP             string `json:"sip,omitempty"`
	UpdateMessage   string `json:"update_message,omitempty"`
}

func (r *runtimeState) collectVersionInfo(ctx context.Context) versionInfo {
	info := versionInfo{
		Version:         version,
		Platform:        runtime.GOOS,
		Arch:            runtime.GOARCH,
		Shell:           detectShellName(os.Getenv("SHELL")),
		InteractionMode: r.cfg.InteractionMode,
	}
	if exe, err := os.Executable(); err == nil {
		info.Executable = exe
	}
	if r.service != nil {
		guidance := r.service.UpdateGuidance()
		info.Version = guidance.CurrentVersion
		info.InstallMethod = guidance.InstallMethod
		info.Channel = guidance.Channel
		if strings.TrimSpace(guidance.Message) != "" {
			info.UpdateMessage = guidance.Message
		}
		report, err := r.service.StatusReport(ctx, 1)
		if err == nil && report.Live != nil {
			info.Platform = report.Live.Platform
			if strings.TrimSpace(report.Live.PlatformVersion) != "" {
				info.Platform += " " + report.Live.PlatformVersion
			}
			if report.Live.DiskFreeBytes > 0 {
				info.DiskFree = humanBytes(int64(report.Live.DiskFreeBytes))
			}
		}
		for _, diagnostic := range r.service.Diagnostics(ctx) {
			if diagnostic.Name == "sip" {
				info.SIP = diagnostic.Message
				break
			}
		}
	}
	if info.Version == "" {
		info.Version = "dev"
	}
	if info.Platform == "" {
		info.Platform = runtime.GOOS
	}
	if info.InteractionMode == "" {
		info.InteractionMode = "auto"
	}
	return info
}

func printVersionInfo(writer io.Writer, info versionInfo) error {
	_, _ = fmt.Fprintf(writer, "SIFT %s\n", info.Version)
	_, _ = fmt.Fprintf(writer, "Platform: %s  %s\n", info.Platform, info.Arch)
	if info.Executable != "" {
		_, _ = fmt.Fprintf(writer, "Executable: %s\n", info.Executable)
	}
	if info.InstallMethod != "" || info.Channel != "" {
		_, _ = fmt.Fprintf(writer, "Install method: %s  Channel: %s\n", emptyDash(info.InstallMethod), emptyDash(info.Channel))
	}
	if info.Shell != "" || info.InteractionMode != "" {
		_, _ = fmt.Fprintf(writer, "Shell: %s  Interaction: %s\n", emptyDash(info.Shell), emptyDash(info.InteractionMode))
	}
	if info.DiskFree != "" || info.SIP != "" {
		_, _ = fmt.Fprintf(writer, "Disk free: %s  SIP: %s\n", emptyDash(info.DiskFree), emptyDash(info.SIP))
	}
	if info.UpdateMessage != "" {
		_, _ = fmt.Fprintf(writer, "Update: %s\n", info.UpdateMessage)
	}
	return nil
}

func runVersionCommand(ctx context.Context, cmd *cobra.Command, state *runtimeState) error {
	info := state.collectVersionInfo(ctx)
	if state.flags.JSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(info)
	}
	return printVersionInfo(cmd.OutOrStdout(), info)
}

func newVersionCommand(state *runtimeState) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show build, install-method and shell details",
		Long:  "Version shows the current SIFT version, install method, update channel, shell context, executable path, and a short platform posture summary.",
		Example: "  sift version\n" +
			"  sift version --json\n" +
			"  sift --version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersionCommand(cmd.Context(), cmd, state)
		},
	}
}

func emptyDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func humanBytes(bytes int64) string {
	return domain.HumanBytes(bytes)
}
