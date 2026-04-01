package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/batuhanyuksel/sift/internal/domain"
)

const envSIFTReducedMotion = "SIFT_REDUCED_MOTION"

// AccessibilityMetadata holds accessibility information for UI elements
type AccessibilityMetadata struct {
	Role        string // "button", "list", "listitem", "dialog", etc.
	Label       string // Human-readable label
	Description string // Detailed description
	State       string // "checked", "disabled", "expanded", etc.
	Shortcut    string // Keyboard shortcut if any
}

func ReducedMotionEnabled() bool {
	return truthyEnv(os.Getenv(envSIFTReducedMotion))
}

func truthyEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// GetHomeActionAccessibility returns accessibility metadata for home actions
func GetHomeActionAccessibility(action homeAction) AccessibilityMetadata {
	role := "menuitem"
	if !action.Enabled {
		role = "menuitem disabled"
	}

	return AccessibilityMetadata{
		Role:        role,
		Label:       action.Title,
		Description: action.Description,
		State:       getSafetyState(action.Safety),
		Shortcut:    getShortcutForAction(action.ID),
	}
}

// GetMenuItemAccessibility returns accessibility metadata for menu items
func GetMenuItemAccessibility(item domain.Finding, index, total int) AccessibilityMetadata {
	role := "listitem"
	state := string(item.Status)

	return AccessibilityMetadata{
		Role:        role,
		Label:       fmt.Sprintf("Item %d of %d", index+1, total),
		Description: item.Path,
		State:       state,
	}
}

// GetProgressAccessibility returns accessibility metadata for progress display
func GetProgressAccessibility(progress progressModel) AccessibilityMetadata {
	completed, _, failed, skipped, _ := countResultStatuses(domain.ExecutionResult{Items: progress.items})

	state := "in progress"
	if progress.cancelRequested {
		state = "cancelling"
	}
	if len(progress.items) == len(progress.plan.Items) && len(progress.plan.Items) > 0 {
		state = "wrapping up"
	}

	description := fmt.Sprintf("%d completed, %d failed, %d skipped out of %d total",
		completed, failed, skipped, len(progress.plan.Items))

	return AccessibilityMetadata{
		Role:        "progressbar",
		Label:       "Cleaning progress",
		Description: description,
		State:       state,
	}
}

// GetResultAccessibility returns accessibility metadata for result display
func GetResultAccessibility(result resultModel) AccessibilityMetadata {
	completed, deleted, failed, skipped, protected := countResultStatuses(domain.ExecutionResult{Items: result.result.Items})

	state := "complete"
	if failed > 0 {
		state = "has errors"
	}

	description := fmt.Sprintf("%d completed, %d deleted, %d failed, %d skipped, %d protected",
		completed, deleted, failed, skipped, protected)

	return AccessibilityMetadata{
		Role:        "status",
		Label:       "Operation results",
		Description: description,
		State:       state,
	}
}

// FormatForScreenReader formats accessibility metadata for screen readers
func (a AccessibilityMetadata) FormatForScreenReader() string {
	var parts []string

	if a.Label != "" {
		parts = append(parts, a.Label)
	}
	if a.Description != "" {
		parts = append(parts, a.Description)
	}
	if a.State != "" {
		parts = append(parts, "State: "+a.State)
	}
	if a.Shortcut != "" {
		parts = append(parts, "Shortcut: "+a.Shortcut)
	}

	result := strings.Join(parts, ". ")
	if result != "" {
		result += "."
	}
	return result
}

// AnnounceForScreenReader creates a screen reader announcement message
func AnnounceForScreenReader(message string) string {
	return "[ANNOUNCE] " + message
}

// getSafetyState converts safety level to accessibility state
func getSafetyState(safety string) string {
	switch safety {
	case "Güvenli", "Safe":
		return "safe"
	case "Onay alır", "Review":
		return "requires confirmation"
	case "Tehlikeli", "High":
		return "danger"
	default:
		return safety
	}
}

// getShortcutForAction returns keyboard shortcut for an action
func getShortcutForAction(actionID string) string {
	shortcuts := map[string]string{
		"clean":      "enter",
		"uninstall":  "enter",
		"analyze":    "enter",
		"status":     "enter",
		"tools":      "t",
		"protect":    "p",
		"doctor":     "d",
		"optimize":   "o",
		"preflight":  "f",
		"duplicates": "l",
		"largefiles": "L",
	}
	return shortcuts[actionID]
}

// StatusToAccessibility converts domain status to accessibility state
func StatusToAccessibility(status domain.FindingStatus) string {
	switch status {
	case domain.StatusPlanned:
		return "planned"
	case domain.StatusSkipped:
		return "skipped"
	case domain.StatusDeleted:
		return "deleted"
	case domain.StatusCompleted:
		return "completed"
	case domain.StatusAdvisory:
		return "advisory"
	case domain.StatusFailed:
		return "failed"
	case domain.StatusProtected:
		return "protected"
	default:
		return string(status)
	}
}

// CategoryToAccessibility converts category to human-readable form
func CategoryToAccessibility(category domain.Category) string {
	// Convert SCREAMING_SNAKE_CASE to Title Case
	name := strings.ReplaceAll(string(category), "_", " ")
	return strings.Title(strings.ToLower(name))
}
