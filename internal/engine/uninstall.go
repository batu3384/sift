package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func uninstallAdvisoryFinding(app domain.AppEntry) (domain.Finding, bool) {
	command := nativeUninstallCommand(app)
	display := strings.TrimSpace(command)
	if display == "" {
		display = strings.TrimSpace(app.UninstallHint)
	}
	if display == "" {
		return domain.Finding{}, false
	}
	message := "Run the native uninstaller before deleting leftover files."
	if app.UninstallHint != "" {
		message = app.UninstallHint
	}
	action := domain.ActionAdvisory
	status := domain.StatusAdvisory
	if command != "" {
		action = domain.ActionNative
		status = domain.StatusPlanned
	}
	return domain.Finding{
		ID:            uuid.NewString(),
		RuleID:        "uninstall.native_step",
		Name:          "Native uninstall step",
		Category:      domain.CategoryAppLeftovers,
		Path:          coalesce(domain.NormalizePath(app.BundlePath), app.DisplayName),
		DisplayPath:   display,
		Risk:          domain.RiskReview,
		Action:        action,
		Status:        status,
		RequiresAdmin: app.RequiresAdmin,
		NativeCommand: command,
		Recovery: domain.RecoveryHint{
			Message:  message,
			Location: "native uninstaller",
		},
		Source: uninstallTargetLabel(app) + " native uninstall",
	}, true
}

func uninstallTargetLabel(app domain.AppEntry) string {
	return coalesce(strings.TrimSpace(app.DisplayName), strings.TrimSpace(app.Name), "App")
}

func (s *Service) uninstallAftermathAdvisories(app domain.AppEntry) []domain.Finding {
	label := uninstallTargetLabel(app)
	items := make([]domain.Finding, 0, 8)
	addAdvisory := func(ruleID, name, display, message, phase, impact string, verify, suggestedBy []string) {
		items = append(items, domain.Finding{
			ID:          uuid.NewString(),
			RuleID:      ruleID,
			Name:        name,
			Category:    domain.CategoryMaintenance,
			Path:        display,
			DisplayPath: display,
			Risk:        domain.RiskReview,
			Action:      domain.ActionAdvisory,
			Status:      domain.StatusAdvisory,
			Recovery: domain.RecoveryHint{
				Message:  message,
				Location: "manual follow-up",
			},
			Source:      label + " aftermath",
			TaskPhase:   phase,
			TaskImpact:  impact,
			TaskVerify:  append([]string{}, verify...),
			SuggestedBy: append([]string{}, suggestedBy...),
		})
	}
	addCommand := func(ruleID, name, path string, args []string, message, capability string, timeout int, requiresAdmin bool, phase, impact string, verify, suggestedBy []string) {
		display := strings.TrimSpace(strings.Join(append([]string{path}, args...), " "))
		items = append(items, domain.Finding{
			ID:             uuid.NewString(),
			RuleID:         ruleID,
			Name:           name,
			Category:       domain.CategoryMaintenance,
			Path:           ruleID,
			DisplayPath:    display,
			Risk:           domain.RiskReview,
			Action:         domain.ActionCommand,
			Status:         domain.StatusPlanned,
			CommandPath:    path,
			CommandArgs:    append([]string{}, args...),
			TimeoutSeconds: timeout,
			RequiresAdmin:  requiresAdmin,
			Capability:     capability,
			Recovery: domain.RecoveryHint{
				Message:  message,
				Location: "managed follow-up",
			},
			Source:      label + " aftermath",
			TaskPhase:   phase,
			TaskImpact:  impact,
			TaskVerify:  append([]string{}, verify...),
			SuggestedBy: append([]string{}, suggestedBy...),
		})
	}
	switch s.Adapter.Name() {
	case "darwin":
		addCommand(
			"uninstall.aftermath.launchservices",
			"Refresh LaunchServices registration",
			"/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister",
			[]string{"-r", "-f", "-domain", "local", "-domain", "system", "-domain", "user"},
			"Use if the app still appears in Open With, Launchpad, or Finder after uninstall.",
			"lsregister available",
			20,
			false,
			"refresh",
			"Refreshes LaunchServices registration so removed apps disappear from Finder and Open With menus.",
			[]string{"Confirm the app no longer appears in Open With or Launchpad"},
			[]string{"LaunchServices"},
		)
		addCommand(
			"uninstall.aftermath.dock",
			"Refresh Dock and stale app tiles",
			"/usr/bin/killall",
			[]string{"Dock"},
			"Use if Dock tiles or recents still reference the removed app after uninstall.",
			"Dock process available",
			10,
			false,
			"refresh",
			"Reloads Dock tiles and recents so stale app references disappear.",
			[]string{"Confirm Dock recents and app tiles no longer show the removed app"},
			[]string{"Dock"},
		)
		items = append(items, darwinLaunchctlUnloadFindings(label, app)...)
		if loginItemFinding, ok := darwinLoginItemCleanupFinding(label, app); ok {
			items = append(items, loginItemFinding)
		} else {
			addAdvisory(
				"uninstall.aftermath.login-items",
				"Review login items",
				"System Settings > General > Login Items",
				"Remove lingering login or background items if the app still appears after uninstall.",
				"aftercare",
				"Surfaces lingering login or background items tied to the removed app.",
				[]string{"Confirm unused login or background items are disabled in System Settings"},
				[]string{"Login items"},
			)
		}
		if strings.Contains(strings.ToLower(app.Origin), "homebrew") {
			if brewPath, err := s.resolveExecutable("brew"); err == nil {
				addCommand(
					"uninstall.aftermath.homebrew",
					"Run Homebrew autoremove",
					brewPath,
					[]string{"autoremove"},
					"Use if package-manager dependencies remain after uninstall.",
					"brew available",
					120,
					false,
					"cleanup",
					"Prunes now-unused Homebrew dependencies left behind by package removal.",
					[]string{"Run `brew cleanup` if cached downloads still dominate disk usage"},
					[]string{"Homebrew"},
				)
			} else {
				addAdvisory(
					"uninstall.aftermath.homebrew",
					"Review Homebrew autoremove",
					"brew autoremove",
					"Use if package-manager dependencies remain after uninstall.",
					"cleanup",
					"Removes package-manager dependencies that are no longer required.",
					[]string{"Run `brew autoremove` manually if package-manager dependencies remain"},
					[]string{"Homebrew"},
				)
			}
		}
	case "windows":
		addAdvisory(
			"uninstall.aftermath.shortcuts",
			"Review shortcuts and startup entries",
			"Settings > Apps > Startup",
			"Remove lingering shortcuts or startup entries if the app still appears after uninstall.",
			"aftercare",
			"Surfaces stale startup and shortcut entries left behind after app removal.",
			[]string{"Confirm startup entries and shortcuts no longer reference the removed app"},
			[]string{"Startup entries"},
		)
	}
	return items
}

func darwinLaunchctlUnloadFindings(label string, app domain.AppEntry) []domain.Finding {
	aliases := uninstallLoginItemAliases(app)
	if len(aliases) == 0 {
		return nil
	}
	makeFinding := func(ruleID, name, commandPath string, args []string, timeout int, requiresAdmin bool, message, phase, impact string, verify, suggestedBy []string) domain.Finding {
		display := strings.TrimSpace(strings.Join(append([]string{commandPath}, args...), " "))
		return domain.Finding{
			ID:             uuid.NewString(),
			RuleID:         ruleID,
			Name:           name,
			Category:       domain.CategoryMaintenance,
			Path:           ruleID,
			DisplayPath:    display,
			Risk:           domain.RiskReview,
			Action:         domain.ActionCommand,
			Status:         domain.StatusPlanned,
			CommandPath:    commandPath,
			CommandArgs:    append([]string{}, args...),
			TimeoutSeconds: timeout,
			RequiresAdmin:  requiresAdmin,
			Capability:     "launchctl available",
			Recovery: domain.RecoveryHint{
				Message:  message,
				Location: "managed follow-up",
			},
			Source:      label + " aftermath",
			TaskPhase:   phase,
			TaskImpact:  impact,
			TaskVerify:  append([]string{}, verify...),
			SuggestedBy: append([]string{}, suggestedBy...),
		}
	}
	return []domain.Finding{
		makeFinding(
			"uninstall.aftermath.launchctl-user",
			"Unload matching user launch agents",
			"/bin/sh",
			append([]string{"-c", darwinLaunchctlUnloadScript(false), "sift-launchctl-user"}, aliases...),
			20,
			false,
			"Use if user launch agents still reference the removed app after uninstall.",
			"aftercare",
			"Unloads lingering per-user launch agents that still point at the removed app.",
			[]string{"Confirm no matching user launch agents are still loaded"},
			[]string{"Launch agents"},
		),
		makeFinding(
			"uninstall.aftermath.launchctl-system",
			"Unload matching system launch agents",
			"/usr/bin/sudo",
			append([]string{"/bin/sh", "-c", darwinLaunchctlUnloadScript(true), "sift-launchctl-system"}, aliases...),
			25,
			true,
			"Use if system launch agents or daemons still reference the removed app after uninstall.",
			"secure",
			"Unloads lingering system launch agents or daemons tied to the removed app.",
			[]string{"Confirm no matching system launch agents or daemons are still loaded"},
			[]string{"Launch daemons"},
		),
	}
}

func darwinLaunchctlUnloadScript(includeSystem bool) string {
	roots := []string{`"$HOME/Library/LaunchAgents"`}
	if includeSystem {
		roots = append(roots, `"/Library/LaunchAgents"`, `"/Library/LaunchDaemons"`)
	}
	return `for alias in "$@"; do lower_alias=$(printf '%s' "$alias" | tr '[:upper:]' '[:lower:]'); for root in ` +
		strings.Join(roots, " ") +
		`; do [ -d "$root" ] || continue; find "$root" -maxdepth 1 -type f -name '*.plist' -print | while IFS= read -r plist; do base=$(basename "$plist" | tr '[:upper:]' '[:lower:]'); case "$base" in *"$lower_alias"*) launchctl unload "$plist" >/dev/null 2>&1 || true ;; esac; done; done; done`
}

func darwinLoginItemCleanupFinding(label string, app domain.AppEntry) (domain.Finding, bool) {
	aliases := uninstallLoginItemAliases(app)
	if len(aliases) == 0 {
		return domain.Finding{}, false
	}
	return domain.Finding{
		ID:             uuid.NewString(),
		RuleID:         "uninstall.aftermath.login-items",
		Name:           "Remove login items",
		Category:       domain.CategoryMaintenance,
		Path:           "uninstall.aftermath.login-items",
		DisplayPath:    "Remove login items for " + label,
		Risk:           domain.RiskReview,
		Action:         domain.ActionCommand,
		Status:         domain.StatusPlanned,
		CommandPath:    "/usr/bin/osascript",
		CommandArgs:    darwinLoginItemScriptArgs(aliases),
		TimeoutSeconds: 15,
		Capability:     "osascript available",
		Recovery: domain.RecoveryHint{
			Message:  "Use if the app still appears in Login Items or background items after uninstall.",
			Location: "managed follow-up",
		},
		Source:      label + " aftermath",
		TaskPhase:   "aftercare",
		TaskImpact:  "Removes lingering login or background items still registered for the app.",
		TaskVerify:  []string{"Confirm the app no longer appears in Login Items or background items"},
		SuggestedBy: []string{"Login items"},
	}, true
}

func uninstallLoginItemAliases(app domain.AppEntry) []string {
	values := []string{
		app.DisplayName,
		app.Name,
		strings.TrimSuffix(filepath.Base(strings.TrimSpace(app.BundlePath)), filepath.Ext(strings.TrimSpace(app.BundlePath))),
	}
	seen := map[string]struct{}{}
	var aliases []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		value = strings.TrimSuffix(value, ".app")
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		aliases = append(aliases, value)
	}
	return aliases
}

func darwinLoginItemScriptArgs(aliases []string) []string {
	if len(aliases) == 0 {
		return nil
	}
	quoted := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		escaped := strings.ReplaceAll(alias, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		quoted = append(quoted, `"`+escaped+`"`)
	}
	lines := []string{
		"set targetNames to {" + strings.Join(quoted, ", ") + "}",
		`tell application "System Events"`,
		"try",
		"set itemCount to count of login items",
		"repeat with i from itemCount to 1 by -1",
		"try",
		"set itemName to name of login item i",
		"repeat with targetName in targetNames",
		"if itemName is (contents of targetName) then",
		"delete login item i",
		"exit repeat",
		"end if",
		"end repeat",
		"end try",
		"end repeat",
		"end try",
		"end tell",
	}
	args := make([]string, 0, len(lines)*2)
	for _, line := range lines {
		args = append(args, "-e", line)
	}
	return args
}

func nativeContinuationWarning(plan domain.ExecutionPlan) string {
	if len(plan.Targets) == 0 {
		return "Native uninstaller launched. SIFT continued with remnant cleanup and aftercare in this run."
	}
	return fmt.Sprintf("Native uninstaller launched for %s. SIFT continued with remnant cleanup and aftercare in this run.", strings.Join(plan.Targets, ", "))
}

func uninstallAftermathCommands(plan domain.ExecutionPlan) []string {
	if plan.Command != "uninstall" {
		return nil
	}
	var commands []string
	for _, item := range plan.Items {
		if item.Action != domain.ActionAdvisory {
			continue
		}
		commands = append(commands, item.DisplayPath)
	}
	return dedupe(trimmedValues(commands))
}

func hasCompletedUninstallWork(items []domain.OperationResult) bool {
	for _, item := range items {
		if item.Status == domain.StatusDeleted || item.Status == domain.StatusCompleted {
			return true
		}
	}
	return false
}

func uninstallPlanItemKey(item domain.Finding) string {
	return strings.Join([]string{
		item.RuleID,
		item.Path,
		item.DisplayPath,
		string(item.Action),
	}, "|")
}

func (s *Service) BuildBatchUninstallPlan(ctx context.Context, appNames []string, dryRun, allowAdmin bool) (domain.ExecutionPlan, error) {
	names := dedupe(trimmedValues(appNames))
	if len(names) == 0 {
		return domain.ExecutionPlan{}, fmt.Errorf("at least one app is required")
	}
	if len(names) == 1 {
		return s.BuildUninstallPlan(ctx, names[0], dryRun, allowAdmin)
	}
	merged := domain.ExecutionPlan{
		ScanID:    uuid.NewString(),
		Command:   "uninstall",
		Platform:  s.Adapter.Name(),
		CreatedAt: time.Now().UTC(),
		PlanState: "preview",
		DryRun:    dryRun,
	}
	policies := make([]domain.ProtectionPolicy, 0, len(names))
	seenItems := map[string]struct{}{}
	for _, name := range names {
		plan, err := s.BuildUninstallPlan(ctx, name, dryRun, allowAdmin)
		if err != nil {
			return domain.ExecutionPlan{}, err
		}
		merged.Warnings = append(merged.Warnings, plan.Warnings...)
		merged.Targets = append(merged.Targets, plan.Targets...)
		policies = append(policies, plan.Policy)
		for _, item := range plan.Items {
			key := uninstallPlanItemKey(item)
			if _, ok := seenItems[key]; ok {
				continue
			}
			seenItems[key] = struct{}{}
			merged.Items = append(merged.Items, item)
		}
	}
	merged.Warnings = dedupe(merged.Warnings)
	merged.Targets = dedupe(merged.Targets)
	merged.Policy = mergeProtectionPolicies(policies)
	merged.Totals = calculateTotals(merged.Items)
	merged.RequiresConfirmation = s.requiresConfirmation("uninstall", dryRun, merged.Items)
	if len(merged.Items) == 0 {
		merged.PlanState = "empty"
	}
	if s.Store != nil {
		_ = s.Store.SavePlan(ctx, merged)
	}
	return merged, nil
}
