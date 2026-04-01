package engine

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/platform"
)

const (
	checkIDFileVault      = "check.filevault"
	checkIDFirewall       = "check.firewall"
	checkIDGatekeeper     = "check.gatekeeper"
	checkIDSIP            = "check.sip"
	checkIDMacOSUpdates   = "check.macos_updates"
	checkIDBrewUpdates    = "check.brew_updates"
	checkIDBrewHealth     = "check.brew_health"
	checkIDTouchID        = "check.touchid"
	checkIDRosetta        = "check.rosetta2"
	checkIDGitIdentity    = "check.git_identity"
	checkIDLoginItems     = "check.login_items"
	checkIDSiftUpdate     = "check.sift_update"
	checkIDMemoryPressure = "check.memory_pressure"
	checkIDSwapPressure   = "check.swap_pressure"
	checkIDDiskPressure   = "check.disk_pressure"
	checkIDHealthScore    = "check.health_score"
)

func (s *Service) CheckReport(ctx context.Context) (domain.CheckReport, error) {
	report := domain.CheckReport{
		CreatedAt: time.Now().UTC(),
		Platform:  s.Adapter.Name(),
	}
	diagnostics := s.Diagnostics(ctx)
	for _, diagnostic := range diagnostics {
		if item, ok := s.checkItemFromDiagnostic(diagnostic); ok {
			report.Items = append(report.Items, item)
		}
	}
	if live, err := Snapshot(ctx); err == nil && live != nil {
		report.Items = append(report.Items, healthCheckItems(live)...)
	}
	notice := s.UpdateNotice(ctx)
	if item, ok := checkItemFromUpdateNotice(notice); ok {
		report.Items = append(report.Items, item)
	}
	sort.SliceStable(report.Items, func(i, j int) bool {
		left, right := report.Items[i], report.Items[j]
		if groupOrder(left.Group) != groupOrder(right.Group) {
			return groupOrder(left.Group) < groupOrder(right.Group)
		}
		if left.Status != right.Status {
			return left.Status > right.Status
		}
		return left.Name < right.Name
	})
	for _, item := range report.Items {
		report.Summary.Total++
		switch item.Status {
		case "ok":
			report.Summary.OK++
		case "warn":
			report.Summary.Warn++
		}
		if item.AutofixAvailable {
			report.Summary.Autofixable++
		}
	}
	return report, nil
}

func (s *Service) BuildAutofixPlan(ctx context.Context, dryRun, allowAdmin bool) (domain.ExecutionPlan, error) {
	report, err := s.CheckReport(ctx)
	if err != nil {
		return domain.ExecutionPlan{}, err
	}
	items := make([]domain.Finding, 0, report.Summary.Autofixable)
	var warnings []string
	tasksByID := map[string]domain.MaintenanceTask{}
	for _, task := range s.MaintenanceTasks(ctx) {
		tasksByID[task.ID] = task
	}
	for _, check := range report.Items {
		if check.Status != "warn" {
			continue
		}
		checkItems, checkWarnings := s.autofixItemsForCheck(check, tasksByID)
		items = append(items, checkItems...)
		warnings = append(warnings, checkWarnings...)
	}
	policy := s.buildPolicy(ScanOptions{
		Command:    "autofix",
		DryRun:     dryRun,
		AllowAdmin: allowAdmin,
	}, nil, nil)
	for idx := range items {
		items[idx] = applyPolicy(items[idx], evaluatePolicy(policy, items[idx], false))
	}
	plan := domain.ExecutionPlan{
		ScanID:               uuid.NewString(),
		Command:              "autofix",
		Platform:             s.Adapter.Name(),
		CreatedAt:            time.Now().UTC(),
		PlanState:            "preview",
		DryRun:               dryRun,
		RequiresConfirmation: s.requiresConfirmation("autofix", dryRun, items),
		Warnings:             dedupe(warnings),
		Items:                items,
		Totals:               calculateTotals(items),
		Policy:               policy,
	}
	if len(items) == 0 {
		plan.PlanState = "empty"
		plan.Warnings = append(plan.Warnings, "No autofixable findings are active right now.")
	}
	if s.Store != nil {
		_ = s.Store.SavePlan(ctx, plan)
	}
	return plan, nil
}

func (s *Service) checkItemFromDiagnostic(diagnostic platform.Diagnostic) (domain.CheckItem, bool) {
	item := domain.CheckItem{
		Name:    diagnostic.Name,
		Status:  diagnostic.Status,
		Message: diagnostic.Message,
	}
	switch diagnostic.Name {
	case "filevault":
		item.ID = checkIDFileVault
		item.Group = domain.CheckGroupSecurity
		item.Name = "FileVault"
		item.Commands = []string{`sudo fdesetup enable -user "$USER"`}
	case "firewall":
		item.ID = checkIDFirewall
		item.Group = domain.CheckGroupSecurity
		item.Name = "Firewall"
		item.AutofixAvailable = diagnostic.Status == "warn"
		item.Commands = []string{`sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate on`}
	case "gatekeeper":
		item.ID = checkIDGatekeeper
		item.Group = domain.CheckGroupSecurity
		item.Name = "Gatekeeper"
		item.AutofixAvailable = diagnostic.Status == "warn"
		item.Commands = []string{"sudo spctl --master-enable"}
	case "sip":
		item.ID = checkIDSIP
		item.Group = domain.CheckGroupSecurity
		item.Name = "System Integrity Protection"
	case "macos_updates":
		item.ID = checkIDMacOSUpdates
		item.Group = domain.CheckGroupUpdates
		item.Name = "macOS updates"
		item.Commands = []string{"sudo softwareupdate -i -a"}
	case "brew_updates":
		item.ID = checkIDBrewUpdates
		item.Group = domain.CheckGroupUpdates
		item.Name = "Homebrew updates"
		item.Commands = []string{"brew upgrade"}
	case "brew_health":
		item.ID = checkIDBrewHealth
		item.Group = domain.CheckGroupHealth
		item.Name = "Homebrew health"
		item.Commands = []string{"brew doctor"}
	case "touchid":
		item.ID = checkIDTouchID
		item.Group = domain.CheckGroupConfig
		item.Name = "Touch ID for sudo"
		item.AutofixAvailable = diagnostic.Status == "warn"
		item.Commands = []string{"sift touchid enable --dry-run=false --yes"}
	case "rosetta2":
		item.ID = checkIDRosetta
		item.Group = domain.CheckGroupConfig
		item.Name = "Rosetta 2"
		item.AutofixAvailable = diagnostic.Status == "warn"
		item.Commands = []string{"sudo softwareupdate --install-rosetta --agree-to-license"}
	case "git_identity":
		item.ID = checkIDGitIdentity
		item.Group = domain.CheckGroupConfig
		item.Name = "Git identity"
		item.Commands = []string{`git config --global user.name "Your Name"`, `git config --global user.email "you@example.com"`}
	case "login_items":
		item.ID = checkIDLoginItems
		item.Group = domain.CheckGroupConfig
		item.Name = "Login items"
		item.Commands = []string{"Open System Settings > General > Login Items"}
	default:
		return domain.CheckItem{}, false
	}
	return item, true
}

func (s *Service) autofixItemsForCheck(item domain.CheckItem, tasksByID map[string]domain.MaintenanceTask) ([]domain.Finding, []string) {
	switch item.ID {
	case checkIDFirewall:
		return []domain.Finding{autofixCommandFinding(
			"autofix.firewall",
			"Enable macOS firewall",
			"/usr/libexec/ApplicationFirewall/socketfilterfw",
			[]string{"--setglobalstate", "on"},
			domain.RiskReview,
			"Turns on the built-in macOS application firewall.",
			"ApplicationFirewall available",
			15,
			true,
		)}, nil
	case checkIDGatekeeper:
		return []domain.Finding{autofixCommandFinding(
			"autofix.gatekeeper",
			"Enable Gatekeeper",
			"/usr/sbin/spctl",
			[]string{"--master-enable"},
			domain.RiskReview,
			"Re-enables Gatekeeper app assessment checks.",
			"spctl available",
			10,
			true,
		)}, nil
	case checkIDRosetta:
		return []domain.Finding{autofixCommandFinding(
			"autofix.rosetta2",
			"Install Rosetta 2",
			"/usr/sbin/softwareupdate",
			[]string{"--install-rosetta", "--agree-to-license"},
			domain.RiskReview,
			"Installs Rosetta 2 for Intel app translation.",
			"softwareupdate available",
			600,
			true,
		)}, nil
	case checkIDTouchID:
		executable, err := s.currentExecutable()
		if err != nil || strings.TrimSpace(executable) == "" {
			return nil, []string{"Touch ID autofix is unavailable because the current SIFT executable path could not be resolved."}
		}
		return []domain.Finding{autofixCommandFinding(
			"autofix.touchid",
			"Enable Touch ID for sudo",
			executable,
			[]string{"touchid", "enable", "--dry-run=false", "--yes"},
			domain.RiskReview,
			"Configures pam_tid through the supported sudo_local path.",
			"sift executable available",
			30,
			true,
		)}, nil
	case checkIDSiftUpdate:
		executable, err := s.currentExecutable()
		if err != nil || strings.TrimSpace(executable) == "" {
			return nil, []string{"SIFT self-update autofix is unavailable because the current executable path could not be resolved."}
		}
		return []domain.Finding{autofixCommandFinding(
			"autofix.sift-update",
			"Apply SIFT update",
			executable,
			[]string{"update", "--dry-run=false", "--yes"},
			domain.RiskReview,
			"Runs the standard SIFT update flow for the detected install method.",
			"sift executable available",
			900,
			false,
		)}, nil
	case checkIDMemoryPressure, checkIDSwapPressure:
		task, ok := tasksByID["macos.optimize.memory-relief"]
		if !ok {
			return nil, []string{"Memory relief optimize task is not available on this platform."}
		}
		return []domain.Finding{commandMaintenanceFinding(task, nil)}, nil
	default:
		return nil, nil
	}
}

func autofixCommandFinding(ruleID, title, path string, args []string, risk domain.Risk, message, capability string, timeout int, requiresAdmin bool) domain.Finding {
	display := strings.TrimSpace(strings.Join(append([]string{path}, args...), " "))
	return domain.Finding{
		ID:             uuid.NewString(),
		RuleID:         ruleID,
		Name:           title,
		Category:       domain.CategoryMaintenance,
		Path:           ruleID,
		DisplayPath:    display,
		Risk:           risk,
		Action:         domain.ActionCommand,
		Status:         domain.StatusPlanned,
		CommandPath:    path,
		CommandArgs:    append([]string{}, args...),
		TimeoutSeconds: timeout,
		RequiresAdmin:  requiresAdmin,
		Capability:     capability,
		TaskPhase:      "autofix",
		Recovery: domain.RecoveryHint{
			Message:  message,
			Location: "autofix command",
		},
		Source: "Autofix",
	}
}

func groupOrder(group domain.CheckGroup) int {
	switch group {
	case domain.CheckGroupSecurity:
		return 0
	case domain.CheckGroupUpdates:
		return 1
	case domain.CheckGroupConfig:
		return 2
	case domain.CheckGroupHealth:
		return 3
	default:
		return 4
	}
}

func checkItemFromUpdateNotice(notice UpdateNotice) (domain.CheckItem, bool) {
	if !notice.Available {
		return domain.CheckItem{}, false
	}
	return domain.CheckItem{
		ID:               checkIDSiftUpdate,
		Group:            domain.CheckGroupUpdates,
		Name:             "SIFT update",
		Status:           "warn",
		Message:          notice.Message,
		AutofixAvailable: true,
		Commands:         append([]string{}, notice.Commands...),
	}, true
}

func healthCheckItems(live *SystemSnapshot) []domain.CheckItem {
	items := []domain.CheckItem{}
	if live == nil {
		return items
	}
	if live.MemoryUsedPercent >= 80 {
		items = append(items, domain.CheckItem{
			ID:               checkIDMemoryPressure,
			Group:            domain.CheckGroupHealth,
			Name:             "Memory pressure",
			Status:           "warn",
			Message:          fmt.Sprintf("Memory usage is high at %.1f%%.", live.MemoryUsedPercent),
			AutofixAvailable: true,
			Commands:         []string{"sift autofix", "sift optimize"},
		})
	} else {
		items = append(items, domain.CheckItem{
			ID:      checkIDMemoryPressure,
			Group:   domain.CheckGroupHealth,
			Name:    "Memory pressure",
			Status:  "ok",
			Message: fmt.Sprintf("Memory usage is steady at %.1f%%.", live.MemoryUsedPercent),
		})
	}
	if live.SwapUsedPercent >= 20 {
		items = append(items, domain.CheckItem{
			ID:               checkIDSwapPressure,
			Group:            domain.CheckGroupHealth,
			Name:             "Swap pressure",
			Status:           "warn",
			Message:          fmt.Sprintf("Swap usage is elevated at %.1f%%.", live.SwapUsedPercent),
			AutofixAvailable: true,
			Commands:         []string{"sift autofix", "sift optimize"},
		})
	}
	diskFreeGB := float64(live.DiskFreeBytes) / (1024 * 1024 * 1024)
	if diskFreeGB > 0 && diskFreeGB < 20 {
		items = append(items, domain.CheckItem{
			ID:      checkIDDiskPressure,
			Group:   domain.CheckGroupHealth,
			Name:    "Disk headroom",
			Status:  "warn",
			Message: fmt.Sprintf("Only %.1f GB of free disk space remain.", diskFreeGB),
			Commands: []string{
				"sift clean",
				"sift purge scan",
			},
		})
	}
	if live.HealthScore < 70 {
		items = append(items, domain.CheckItem{
			ID:      checkIDHealthScore,
			Group:   domain.CheckGroupHealth,
			Name:    "System health score",
			Status:  "warn",
			Message: fmt.Sprintf("Health score is %d (%s).", live.HealthScore, live.HealthLabel),
			Commands: []string{
				"sift status",
				"sift optimize",
			},
		})
	}
	return items
}
