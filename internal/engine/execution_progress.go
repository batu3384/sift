package engine

import (
	"fmt"
	"strings"

	"github.com/batu3384/sift/internal/domain"
)

type executionSection struct {
	Key       string
	Label     string
	Category  domain.Category
	Index     int
	Total     int
	Items     int
	Bytes     int64
	LastItem  int
	Completed int
}

func buildExecutionSections(plan domain.ExecutionPlan) []*executionSection {
	switch plan.Command {
	case "clean", "uninstall", "optimize", "autofix":
	default:
		return nil
	}
	if len(plan.Items) == 0 {
		return nil
	}
	order := make([]string, 0)
	sections := map[string]*executionSection{}
	byIndex := make([]*executionSection, len(plan.Items))
	for idx, item := range plan.Items {
		key, label := executionSectionKeyLabel(plan.Command, item)
		if key == "" {
			continue
		}
		section := sections[key]
		if section == nil {
			order = append(order, key)
			section = &executionSection{
				Key:      key,
				Label:    label,
				Category: item.Category,
			}
			sections[key] = section
		}
		section.Items++
		section.Bytes += item.Bytes
		section.LastItem = idx + 1
		byIndex[idx] = section
	}
	for idx, key := range order {
		section := sections[key]
		section.Index = idx + 1
		section.Total = len(order)
	}
	return byIndex
}

func executionSectionKeyLabel(command string, item domain.Finding) (string, string) {
	switch command {
	case "uninstall":
		switch {
		case item.Action == domain.ActionNative:
			return "uninstall::handoff", "Native handoff"
		case item.Action == domain.ActionCommand && strings.TrimSpace(item.TaskPhase) != "":
			phase := strings.TrimSpace(item.TaskPhase)
			return "uninstall::phase::" + strings.ToLower(phase), strings.ToUpper(phase[:1]) + phase[1:]
		default:
			return "uninstall::remnants", "Remnants"
		}
	case "optimize", "autofix":
		if phase := strings.TrimSpace(item.TaskPhase); phase != "" {
			return command + "::phase::" + strings.ToLower(phase), strings.ToUpper(phase[:1]) + phase[1:]
		}
		if command == "autofix" {
			return "autofix::fix", "Fix"
		}
		return command + "::task", "Task"
	default:
		return domain.ExecutionGroupKey(item), domain.ExecutionGroupLabel(item)
	}
}

func progressSectionStartDetail(command string, section *executionSection) string {
	if section == nil {
		return "starting section"
	}
	label := strings.ToLower(strings.TrimSpace(section.Label))
	if label == "" {
		label = "cleanup section"
	}
	switch command {
	case "clean":
		return fmt.Sprintf("starting %s reclaim", label)
	case "uninstall":
		switch label {
		case "native handoff":
			return "starting native handoff"
		case "remnants":
			return "starting remnant cleanup"
		default:
			return "starting " + label
		}
	case "optimize", "autofix":
		return fmt.Sprintf("starting %s phase", label)
	}
	return "starting " + label
}

func progressSectionFinishDetail(command string, section *executionSection) string {
	if section == nil {
		return "section settled"
	}
	label := strings.ToLower(strings.TrimSpace(section.Label))
	if label == "" {
		label = "cleanup section"
	}
	switch command {
	case "clean":
		return fmt.Sprintf("%s settled", label)
	case "uninstall":
		switch label {
		case "native handoff":
			return "native handoff settled"
		case "remnants":
			return "remnant cleanup settled"
		default:
			return label + " settled"
		}
	case "optimize", "autofix":
		return label + " phase settled"
	}
	return label + " settled"
}

func progressQueueDetail(command string, item domain.Finding) string {
	switch {
	case item.Action == domain.ActionNative:
		return "queued native handoff  •  " + progressLabel(item)
	case item.Action == domain.ActionCommand:
		if phase := strings.TrimSpace(item.TaskPhase); phase != "" {
			return "queued " + phase + " task  •  " + progressLabel(item)
		}
		if command == "autofix" {
			return "queued fix  •  " + progressLabel(item)
		}
		return "queued task  •  " + progressLabel(item)
	case command == "uninstall":
		return "queued remnant  •  " + progressLabel(item)
	case command == "clean":
		return "queued reclaim  •  " + progressLabel(item)
	default:
		return "queued " + progressLabel(item)
	}
}

func progressCheckDetail(command string, item domain.Finding) string {
	switch item.Action {
	case domain.ActionCommand:
		if phase := strings.TrimSpace(item.TaskPhase); phase != "" {
			return "checking " + phase + " access"
		}
		if command == "autofix" {
			return "checking fix access"
		}
		return "checking task access"
	case domain.ActionNative:
		return "checking native uninstall access"
	default:
		if command == "uninstall" {
			return "checking remnant access"
		}
		if command == "clean" {
			return "checking reclaim target"
		}
		return "checking selected item"
	}
}

func progressApplyStep(command string, item domain.Finding, permanent bool) string {
	if item.Action == domain.ActionCommand {
		if phase := strings.TrimSpace(item.TaskPhase); phase != "" {
			return phase
		}
		if command == "autofix" {
			return "fix"
		}
		return "run"
	}
	if item.Action == domain.ActionNative {
		return "handoff"
	}
	if permanent {
		return "delete"
	}
	if command == "clean" {
		return "reclaim"
	}
	if command == "uninstall" {
		return "remove"
	}
	return "trash"
}

func progressApplyDetail(command string, item domain.Finding, permanent bool) string {
	if item.Action == domain.ActionCommand {
		return progressManagedTaskDetail(command, item)
	}
	if item.Action == domain.ActionNative {
		return "opening native uninstall  •  " + progressLabel(item)
	}
	if permanent {
		return "deleting " + progressLabel(item)
	}
	if command == "clean" {
		return "reclaiming " + progressLabel(item)
	}
	if command == "uninstall" {
		return "removing remnant  •  " + progressLabel(item)
	}
	return "moving " + progressLabel(item) + " to trash"
}

func progressVerifyDetail(command string, item domain.Finding) string {
	switch item.Action {
	case domain.ActionCommand:
		return progressManagedVerifyDetail(command, item)
	case domain.ActionNative:
		return "waiting for native handoff"
	default:
		if command == "uninstall" {
			return "checking remnant result  •  " + progressLabel(item)
		}
		if command == "clean" {
			return "checking reclaim result  •  " + progressLabel(item)
		}
		return "checking result for " + progressLabel(item)
	}
}

func progressManagedTaskDetail(command string, item domain.Finding) string {
	phase := strings.TrimSpace(item.TaskPhase)
	if phase != "" {
		return phase + " task  •  " + progressLabel(item)
	}
	if command == "autofix" {
		return "running fix  •  " + progressLabel(item)
	}
	return "running task  •  " + progressLabel(item)
}

func progressManagedVerifyDetail(command string, item domain.Finding) string {
	if len(item.TaskVerify) > 0 && strings.TrimSpace(item.TaskVerify[0]) != "" {
		return item.TaskVerify[0]
	}
	if command == "autofix" {
		return "checking fix result"
	}
	return "checking task result"
}

func progressCommandStep(item domain.Finding) string {
	phase := strings.TrimSpace(item.TaskPhase)
	if phase != "" {
		return phase
	}
	return "run"
}

func progressResultDetail(item domain.Finding, op domain.OperationResult) string {
	if strings.TrimSpace(op.Message) != "" {
		return op.Message
	}
	switch op.Status {
	case domain.StatusDeleted:
		return "done"
	case domain.StatusCompleted:
		return "completed"
	case domain.StatusSkipped:
		return "skipped"
	case domain.StatusProtected:
		return "blocked"
	case domain.StatusFailed:
		return "failed"
	default:
		return progressLabel(item)
	}
}

func progressLabel(item domain.Finding) string {
	label := strings.TrimSpace(item.DisplayPath)
	if label == "" {
		label = strings.TrimSpace(item.Path)
	}
	if label == "" {
		label = strings.TrimSpace(item.Name)
	}
	if label == "" {
		label = "item"
	}
	return label
}
