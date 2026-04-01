package engine

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/batuhanyuksel/sift/internal/config"
	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/report"
)

type RemoveGuidance struct {
	ExecutablePath string   `json:"executable_path,omitempty"`
	InstallMethod  string   `json:"install_method"`
	Message        string   `json:"message"`
	Commands       []string `json:"commands,omitempty"`
}

func (s *Service) BuildRemovePlan(ctx context.Context) (domain.ExecutionPlan, error) {
	var items []domain.Finding
	var allowed []string
	var warnings []string
	configPath, _ := config.Path()
	if configPath != "" {
		items = append(items, newOwnedDataFinding("remove.config", "SIFT config", configPath, domain.RiskReview))
		allowed = append(allowed, configPath)
	}
	if reportDir, err := report.Dir(); err == nil && reportDir != "" {
		items = append(items, newOwnedDataFinding("remove.reports", "Report cache", reportDir, domain.RiskSafe))
		allowed = append(allowed, reportDir)
	}
	if s.Store != nil && s.Store.AuditLogPath() != "" {
		auditDir := filepath.Dir(s.Store.AuditLogPath())
		items = append(items, newOwnedDataFinding("remove.audit", "Audit logs", auditDir, domain.RiskSafe))
		allowed = append(allowed, auditDir)
	}
	remove := s.RemoveGuidance()
	switch command, ok := removeCommandForMethod(remove.InstallMethod); {
	case ok:
		executable, err := s.resolveExecutable(command.Name)
		if err != nil {
			warnings = append(warnings, "Install-method removal command is unavailable on this machine: "+err.Error())
			items = append(items, removeAdvisoryFinding(remove))
		} else {
			items = append(items, removeManagedCommandFinding(remove, executable, command))
			warnings = append(warnings, "Review the full removal sequence before executing package-manager uninstall.")
		}
	case remove.InstallMethod == "manual":
		if finding, ok := newRemoveExecutableFinding(remove); ok {
			items = append(items, finding)
			allowed = append(allowed, finding.Path)
			warnings = append(warnings, "Manual installs now include the executable in the review plan.")
		} else {
			items = append(items, removeAdvisoryFinding(remove))
			warnings = append(warnings, "Executable cleanup could not be staged automatically; review the follow-up guidance.")
		}
	default:
		items = append(items, removeAdvisoryFinding(remove))
	}
	policy := s.buildPolicy(ScanOptions{Command: "remove", DryRun: false, AllowAdmin: true}, nil, allowed)
	for idx := range items {
		items[idx] = applyPolicy(items[idx], evaluatePolicy(policy, items[idx], false))
	}
	plan := domain.ExecutionPlan{
		ScanID:               uuid.NewString(),
		Command:              "remove",
		Platform:             s.Adapter.Name(),
		CreatedAt:            time.Now().UTC(),
		PlanState:            "preview",
		DryRun:               false,
		RequiresConfirmation: true,
		Items:                items,
		Totals:               calculateTotals(items),
		Policy:               policy,
		Warnings:             dedupe(warnings),
	}
	if s.Store != nil {
		_ = s.Store.SavePlan(ctx, plan)
	}
	return plan, nil
}

func (s *Service) RemoveGuidance() RemoveGuidance {
	method, commands := s.installMethodAndCommands(UpdateChannelStable, false)
	executable, _ := s.currentExecutable()
	message := "Remove SIFT state with `sift remove --dry-run=false --yes`, then uninstall the binary with your install method."
	if method == "manual" {
		message = "Remove SIFT state with `sift remove --dry-run=false --yes`, then delete the installed binary manually."
	}
	return RemoveGuidance{
		ExecutablePath: executable,
		InstallMethod:  method,
		Message:        message,
		Commands:       commands,
	}
}

func removeCommandForMethod(method string) (managedCommand, bool) {
	switch method {
	case "homebrew":
		return managedCommand{Name: "brew", Args: []string{"uninstall", "sift"}}, true
	case "scoop":
		return managedCommand{Name: "scoop", Args: []string{"uninstall", "sift"}}, true
	case "winget":
		return managedCommand{Name: "winget", Args: []string{"uninstall", "SIFT"}}, true
	default:
		return managedCommand{}, false
	}
}

func newOwnedDataFinding(ruleID, name, path string, risk domain.Risk) domain.Finding {
	normalized := domain.NormalizePath(path)
	finding := domain.Finding{
		ID:          uuid.NewString(),
		RuleID:      ruleID,
		Name:        name,
		Category:    domain.CategoryMaintenance,
		Path:        normalized,
		DisplayPath: normalized,
		Risk:        risk,
		Action:      domain.ActionTrash,
		Status:      domain.StatusPlanned,
		Recovery: domain.RecoveryHint{
			Message:  "SIFT-owned state can be recreated on next run.",
			Location: "system trash",
		},
		Source: "SIFT-owned data",
	}
	if info, err := os.Stat(normalized); err == nil {
		size := info.Size()
		if info.IsDir() {
			if measured, newest, measureErr := rulesMeasurePath(context.Background(), normalized); measureErr == nil {
				size = measured
				finding.LastModified = newest
				finding.Fingerprint = domain.Fingerprint{
					Mode:    uint32(info.Mode()),
					Size:    measured,
					ModTime: newest,
				}
				finding.Bytes = measured
				return finding
			}
		}
		finding.Bytes = size
		finding.LastModified = info.ModTime()
		finding.Fingerprint = domain.Fingerprint{
			Mode:    uint32(info.Mode()),
			Size:    size,
			ModTime: info.ModTime(),
		}
	}
	return finding
}

func removeAdvisoryFinding(remove RemoveGuidance) domain.Finding {
	return domain.Finding{
		ID:          uuid.NewString(),
		RuleID:      "remove.binary",
		Name:        "Binary and package manager removal",
		Category:    domain.CategoryMaintenance,
		Path:        remove.ExecutablePath,
		DisplayPath: remove.InstallMethod,
		Risk:        domain.RiskReview,
		Action:      domain.ActionAdvisory,
		Status:      domain.StatusAdvisory,
		Recovery: domain.RecoveryHint{
			Message:  strings.Join(remove.Commands, " | "),
			Location: "manual uninstall",
		},
		Source: remove.Message,
	}
}

func removeManagedCommandFinding(remove RemoveGuidance, executable string, command managedCommand) domain.Finding {
	return domain.Finding{
		ID:             uuid.NewString(),
		RuleID:         "remove.uninstall",
		Name:           "Uninstall SIFT package",
		Category:       domain.CategoryMaintenance,
		Path:           remove.ExecutablePath,
		DisplayPath:    strings.Join(append([]string{executable}, command.Args...), " "),
		Risk:           domain.RiskReview,
		Action:         domain.ActionCommand,
		Status:         domain.StatusPlanned,
		CommandPath:    executable,
		CommandArgs:    append([]string{}, command.Args...),
		TimeoutSeconds: 120,
		Recovery: domain.RecoveryHint{
			Message:  strings.Join(remove.Commands, " | "),
			Location: remove.InstallMethod + " uninstall",
		},
		Source: "Install-method aware removal",
	}
}

func newRemoveExecutableFinding(remove RemoveGuidance) (domain.Finding, bool) {
	normalized := domain.NormalizePath(remove.ExecutablePath)
	if normalized == "" || runtime.GOOS == "windows" {
		return domain.Finding{}, false
	}
	info, err := os.Stat(normalized)
	if err != nil {
		return domain.Finding{}, false
	}
	return domain.Finding{
		ID:          uuid.NewString(),
		RuleID:      "remove.executable",
		Name:        "Installed SIFT binary",
		Category:    domain.CategoryMaintenance,
		Path:        normalized,
		DisplayPath: normalized,
		Risk:        domain.RiskReview,
		Action:      domain.ActionTrash,
		Status:      domain.StatusPlanned,
		Recovery: domain.RecoveryHint{
			Message:  strings.Join(remove.Commands, " | "),
			Location: "system trash",
		},
		Source:       "Manual install binary cleanup",
		Bytes:        info.Size(),
		LastModified: info.ModTime(),
		Fingerprint: domain.Fingerprint{
			Mode:    uint32(info.Mode()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		},
	}, true
}
