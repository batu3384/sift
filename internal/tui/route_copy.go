package tui

import (
	"fmt"
	"strings"
)

func reviewScopeKey(command string) string {
	switch strings.TrimSpace(command) {
	case "clean":
		return "Sweep"
	case "uninstall":
		return "Removal"
	case "analyze":
		return "Trace"
	case "optimize":
		return "Task"
	case "autofix":
		return "Fix"
	case "installer", "purge":
		return "Sweep"
	default:
		return "Launch"
	}
}

func preflightAccessKey(command string) string {
	switch strings.TrimSpace(command) {
	case "clean":
		return "Sweep access"
	case "uninstall":
		return "Removal access"
	case "analyze":
		return "Trace access"
	case "optimize":
		return "Task access"
	case "autofix":
		return "Fix access"
	default:
		return "Access"
	}
}

func paddedLabel(label string, width int) string {
	return fmt.Sprintf("%-*s", width, label)
}
