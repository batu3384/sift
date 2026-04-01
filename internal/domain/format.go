package domain

import (
	"fmt"
	"strings"
	"time"
)

func HumanBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	value := float64(size)
	for _, unit := range units {
		value /= 1024
		if value < 1024 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}
	return fmt.Sprintf("%.1f YB", value/1024)
}

func HumanDuration(seconds uint64) string {
	if seconds == 0 {
		return "0m"
	}
	duration := time.Duration(seconds) * time.Second
	days := duration / (24 * time.Hour)
	duration -= days * 24 * time.Hour
	hours := duration / time.Hour
	duration -= hours * time.Hour
	minutes := duration / time.Minute
	switch {
	case days > 0:
		if hours > 0 || minutes > 0 {
			return fmt.Sprintf("%dd %dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	case hours > 0:
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	default:
		return fmt.Sprintf("%dm", minutes)
	}
}

func CategoryTitle(category Category) string {
	return strings.ToUpper(strings.ReplaceAll(string(category), "_", " "))
}

func ExecutionGroupLabel(item Finding) string {
	source := strings.TrimSpace(item.Source)
	if source != "" {
		lower := strings.ToLower(source)
		switch {
		case strings.HasPrefix(lower, "immediate child of "):
		case strings.HasPrefix(lower, "large file under "):
		case strings.HasPrefix(lower, "user supplied target"):
		case strings.HasPrefix(lower, "project purge target"):
		case strings.HasPrefix(lower, "stale support directory"):
		default:
			return source
		}
	}
	if item.Category != "" {
		return CategoryTitle(item.Category)
	}
	return "CLEANUP"
}

func ExecutionGroupKey(item Finding) string {
	label := ExecutionGroupLabel(item)
	if label == "" {
		label = CategoryTitle(item.Category)
	}
	return strings.ToLower(string(item.Category) + "::" + label)
}
