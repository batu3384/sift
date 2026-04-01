package tui

import (
	"strings"
)

func menuSpecializedDetailView(actions []homeAction, cursor int, width int, maxLines int) (string, bool) {
	if cursor < 0 || cursor >= len(actions) {
		return "", false
	}
	action := actions[cursor]
	switch action.ID {
	case "check", "autofix", "optimize", "installer", "purge_scan", "protect":
	default:
		return "", false
	}
	lines := []string{
		renderToneBadge(action.Tone) + " " + headerStyle.Render(action.Title),
	}
	if state := menuStateText(action, cursor, len(actions)); state != "" {
		lines = append(lines, mutedStyle.Render("State   ")+wrapText(state, width))
	}
	if next := menuNextActionLine(action); next != "" {
		lines = append(lines, mutedStyle.Render("Next    ")+wrapText(next, width))
	}
	if action.Description != "" {
		lines = append(lines, mutedStyle.Render("What    ")+wrapText(truncateText(action.Description, width), width))
	}
	if action.Command != "" {
		lines = append(lines, mutedStyle.Render("Action  ")+headerStyle.Render(action.Command))
	}
	if lane := toolActionLaneLine(action); lane != "" {
		lines = append(lines, mutedStyle.Render("Focus   ")+wrapText(lane, width))
	}
	if flow := toolActionFlowLine(action); flow != "" {
		lines = append(lines, mutedStyle.Render("Steps   ")+wrapText(flow, width))
	}
	if len(action.Modules) > 0 {
		lines = append(lines, mutedStyle.Render("Scope   ")+wrapText(strings.Join(action.Modules, "  •  "), width))
	}
	if action.When != "" {
		lines = append(lines, mutedStyle.Render("Use     ")+wrapText(truncateText(action.When, width), width))
	}
	if action.Safety != "" {
		lines = append(lines, mutedStyle.Render("Risk    ")+wrapText(truncateText(action.Safety, width), width))
	}
	if !action.Enabled {
		lines = append(lines, highStyle.Render("Setup   ")+"not ready in this session")
	}
	lines = viewportLines(lines, 0, maxLines)
	return strings.Join(lines, "\n"), true
}

func toolActionLaneLine(action homeAction) string {
	switch action.ID {
	case "check":
		return "security posture  •  updates  •  config drift  •  health pressure"
	case "autofix":
		return "safe fixes  •  managed commands  •  verify after apply"
	case "optimize":
		return "preflight  •  cache resets  •  network repair  •  verify"
	case "installer":
		return "download roots  •  shared roots  •  package types  •  stale payloads"
	case "purge_scan":
		return "workspace scan  •  artifact ranking  •  vendor guard  •  reclaim review"
	case "protect":
		return "paths  •  families  •  command scopes  •  safe exceptions"
	default:
		return ""
	}
}

func toolActionFlowLine(action homeAction) string {
	switch action.ID {
	case "check":
		return "inspect findings → choose autofix or doctor → rerun check"
	case "autofix":
		return "load autofixable warnings → review fixes → execute → verify posture"
	case "optimize":
		return "load suggested maintenance → review tasks → execute → verify result"
	case "installer":
		return "find stale installers → review remnants → execute cleanup"
	case "purge_scan":
		return "map repositories → stage artifact clusters → review purge batch"
	case "protect":
		return "edit guardrails → inspect decision path → preserve cleanup safety"
	default:
		return ""
	}
}
