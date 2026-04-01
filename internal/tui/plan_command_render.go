package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func planCommandBoardTitle(command string) string {
	switch command {
	case "optimize":
		return "Task Board"
	case "autofix":
		return "Fix Board"
	case "installer":
		return "Installer Scope"
	case "purge":
		return "Workspace Sweep"
	default:
		return ""
	}
}

func planReviewScopeLine(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "clean":
		label := cleanProfileLabel(plan.Profile)
		if label == "" {
			label = "Clean review"
		}
		mods := planModuleCount(plan)
		return fmt.Sprintf("%s  •  %d %s  •  %s", label, mods, pl(mods, "module", "modules"), domain.HumanBytes(planDisplayBytes(plan)))
	case "optimize":
		phases := maintenancePhaseCount(plan)
		if phases == 0 && actionableCount(plan) > 0 {
			phases = 1
		}
		tasks := actionableCount(plan)
		return fmt.Sprintf("Optimize  •  %d %s  •  %d %s  •  %s", tasks, pl(tasks, "task", "tasks"), phases, pl(phases, "phase", "phases"), domain.HumanBytes(planDisplayBytes(plan)))
	case "autofix":
		fixes := actionableCount(plan)
		suggested := suggestedTaskCount(plan)
		return fmt.Sprintf("Autofix  •  %d %s  •  %d suggested  •  %s", fixes, pl(fixes, "fix", "fixes"), suggested, domain.HumanBytes(planDisplayBytes(plan)))
	case "installer":
		payloads := len(plan.Items)
		return fmt.Sprintf("Installer Cleanup  •  %d %s  •  %s", payloads, pl(payloads, "payload", "payloads"), domain.HumanBytes(planDisplayBytes(plan)))
	case "purge":
		findings := len(plan.Items)
		roots := purgeRootCount(plan)
		return fmt.Sprintf("Purge Scan  •  %d %s  •  %d workspace %s  •  %s", findings, pl(findings, "finding", "findings"), roots, pl(roots, "root", "roots"), domain.HumanBytes(planDisplayBytes(plan)))
	case "uninstall":
		targets := len(plan.Targets)
		if targets == 0 {
			targets = uninstallTargetCount(plan)
		}
		return fmt.Sprintf("Uninstall  •  %d %s  •  %s", targets, pl(targets, "target", "targets"), domain.HumanBytes(planDisplayBytes(plan)))
	case "analyze":
		queued := actionableCount(plan)
		return fmt.Sprintf("Staged Cleanup  •  %d queued %s  •  %s", queued, pl(queued, "item", "items"), domain.HumanBytes(planDisplayBytes(plan)))
	default:
		return ""
	}
}

func planReviewOutcomeLine(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "clean":
		return "y runs cleanup • esc returns"
	case "optimize":
		return "y runs maintenance + verify"
	case "autofix":
		return "y applies fixes + rechecks"
	case "installer":
		return "y removes installer leftovers"
	case "purge":
		return "y removes workspace artifacts"
	case "uninstall":
		return "y runs uninstall + leftovers"
	case "analyze":
		return "y trashes staged paths"
	default:
		return ""
	}
}

func planCommandBoardLines(plan domain.ExecutionPlan, width int) []string {
	switch plan.Command {
	case "optimize", "autofix":
		return maintenanceBoardLines(plan, width)
	case "installer":
		return installerBoardLines(plan, width)
	case "purge":
		return purgeBoardLines(plan, width)
	default:
		return nil
	}
}

func cleanProfileLabel(profile string) string {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case "safe":
		return "Quick Clean"
	case "developer":
		return "Workstation Clean"
	case "deep":
		return "Deep Reclaim"
	default:
		return ""
	}
}

func purgeRootCount(plan domain.ExecutionPlan) int {
	rootSet := map[string]struct{}{}
	for _, item := range plan.Items {
		root := purgeRootLabel(item.Path)
		if root == "" {
			continue
		}
		rootSet[root] = struct{}{}
	}
	return len(rootSet)
}

func uninstallTargetCount(plan domain.ExecutionPlan) int {
	targets := map[string]struct{}{}
	for _, item := range plan.Items {
		label := strings.TrimSpace(item.Name)
		if label == "" {
			label = strings.TrimSpace(item.Source)
		}
		if label == "" {
			label = strings.TrimSpace(item.DisplayPath)
		}
		if label == "" {
			continue
		}
		targets[label] = struct{}{}
	}
	return len(targets)
}

func planDisplayBytes(plan domain.ExecutionPlan) int64 {
	if plan.Totals.Bytes > 0 {
		return plan.Totals.Bytes
	}
	var total int64
	for _, item := range plan.Items {
		total += item.Bytes
	}
	return total
}

func maintenanceBoardLines(plan domain.ExecutionPlan, width int) []string {
	phaseOrder := []string{}
	phaseCounts := map[string]int{}
	commandCount := 0
	advisoryCount := 0
	adminCount := 0
	suggested := 0
	for _, item := range plan.Items {
		if item.Action == domain.ActionCommand {
			commandCount++
		}
		if item.Action == domain.ActionAdvisory {
			advisoryCount++
		}
		if item.RequiresAdmin {
			adminCount++
		}
		if len(item.SuggestedBy) > 0 {
			suggested++
		}
		phase := strings.TrimSpace(item.TaskPhase)
		if phase == "" {
			continue
		}
		if _, ok := phaseCounts[phase]; !ok {
			phaseOrder = append(phaseOrder, phase)
		}
		phaseCounts[phase]++
	}
	lines := []string{
		wrapText(mutedStyle.Render(fmt.Sprintf("%d %s  •  %d %s  •  %d suggested  •  %d admin", commandCount, pl(commandCount, "task", "tasks"), advisoryCount, pl(advisoryCount, "check", "checks"), suggested, adminCount)), width),
	}
	if len(phaseOrder) > 0 {
		parts := make([]string, 0, len(phaseOrder))
		for _, phase := range phaseOrder {
			parts = append(parts, fmt.Sprintf("%s %d", strings.ToUpper(phase), phaseCounts[phase]))
		}
		lines = append(lines, wrapText(mutedStyle.Render("Flow    "+strings.Join(parts, "  •  ")), width))
	}
	return lines
}

func installerBoardLines(plan domain.ExecutionPlan, width int) []string {
	rootSet := map[string]struct{}{}
	extCounts := map[string]int{}
	for _, item := range plan.Items {
		root := installerRootLabel(item.Path)
		if root != "" {
			rootSet[root] = struct{}{}
		}
		ext := strings.ToLower(filepath.Ext(item.Path))
		if ext == "" {
			ext = "folder"
		}
		extCounts[ext]++
	}
	payloads := len(plan.Items)
	roots := len(rootSet)
	lines := []string{
		wrapText(mutedStyle.Render(fmt.Sprintf("%d %s  •  %d %s  •  %s", payloads, pl(payloads, "payload", "payloads"), roots, pl(roots, "root", "roots"), domain.HumanBytes(plan.Totals.Bytes))), width),
	}
	if len(extCounts) > 0 {
		keys := make([]string, 0, len(extCounts))
		for key := range extCounts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys[:min(len(keys), 5)] {
			parts = append(parts, fmt.Sprintf("%s %d", key, extCounts[key]))
		}
		lines = append(lines, wrapText(mutedStyle.Render("Types   "+strings.Join(parts, "  •  ")), width))
	}
	return lines
}

func installerRootLabel(path string) string {
	path = filepath.Clean(path)
	switch {
	case strings.Contains(path, "/Downloads/"):
		return "Downloads"
	case strings.Contains(path, "/Documents/"):
		return "Documents"
	case strings.Contains(path, "/Public/"):
		return "Public"
	case strings.Contains(path, "/Users/Shared/"):
		return "Shared"
	default:
		return filepath.Dir(path)
	}
}

func purgeBoardLines(plan domain.ExecutionPlan, width int) []string {
	rootSet := map[string]struct{}{}
	highRisk := 0
	categoryCounts := map[domain.Category]int{}
	for _, item := range plan.Items {
		if item.Risk == domain.RiskHigh {
			highRisk++
		}
		categoryCounts[item.Category]++
		root := purgeRootLabel(item.Path)
		if root != "" {
			rootSet[root] = struct{}{}
		}
	}
	findings := len(plan.Items)
	roots := len(rootSet)
	lines := []string{
		wrapText(mutedStyle.Render(fmt.Sprintf("%d %s  •  %d %s  •  %d high-risk", findings, pl(findings, "finding", "findings"), roots, pl(roots, "root", "roots"), highRisk)), width),
	}
	if len(categoryCounts) > 0 {
		parts := make([]string, 0, len(categoryCounts))
		for _, key := range []domain.Category{domain.CategoryProjectArtifacts, domain.CategoryDeveloperCaches, domain.CategoryLogs, domain.CategoryLargeFiles} {
			if categoryCounts[key] == 0 {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s %d", sectionTitle(plan, key), categoryCounts[key]))
		}
		if len(parts) > 0 {
			lines = append(lines, wrapText(mutedStyle.Render("Kinds   "+strings.Join(parts, "  •  ")), width))
		}
	}
	return lines
}

func purgeRootLabel(path string) string {
	clean := filepath.Clean(path)
	parts := strings.Split(clean, string(filepath.Separator))
	if len(parts) >= 4 {
		return filepath.Join(parts[:4]...)
	}
	return filepath.Dir(clean)
}
