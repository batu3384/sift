package domain

import (
	"testing"
)

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expected string
	}{
		{"zero", 0, "0 B"},
		{"bytes", 512, "512 B"},
		{"bytes_exact", 1023, "1023 B"},
		{"kilobytes", 1024, "1.0 KB"},
		{"kilobytes_decimal", 1536, "1.5 KB"},
		{"megabytes", 1048576, "1.0 MB"},
		{"gigabytes", 1073741824, "1.0 GB"},
		{"terabytes", 1099511627776, "1.0 TB"},
		{"petabytes", 1125899906842624, "1.0 PB"},
		{"exabytes", 1152921504606846976, "1.0 EB"},
		{"zettabytes", int64(1<<60), "1.0 EB"},
		{"large_value", int64(1<<61), "2.0 EB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HumanBytes(tt.size)
			if result != tt.expected {
				t.Errorf("HumanBytes(%d) = %q, want %q", tt.size, result, tt.expected)
			}
		})
	}
}

func TestHumanDuration(t *testing.T) {
	tests := []struct {
		name     string
		seconds  uint64
		expected string
	}{
		{"zero", 0, "0m"},
		{"minutes", 60, "1m"},
		{"minutes_decimal", 90, "1m"},
		{"hours", 3600, "1h"},
		{"hours_minutes", 3660, "1h 1m"},
		{"days", 86400, "1d"},
		{"days_hours", 90000, "1d 1h"},
		{"days_hours_minutes", 90060, "1d 1h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HumanDuration(tt.seconds)
			if result != tt.expected {
				t.Errorf("HumanDuration(%d) = %q, want %q", tt.seconds, result, tt.expected)
			}
		})
	}
}

func TestCategoryTitle(t *testing.T) {
	tests := []struct {
		category Category
		expected string
	}{
		{CategoryTempFiles, "TEMP FILES"},
		{CategoryDeveloperCaches, "DEVELOPER CACHES"},
		{CategoryBrowserData, "BROWSER DATA"},
		{CategoryLargeFiles, "LARGE FILES"},
		{CategoryDiskUsage, "DISK USAGE"},
		{CategoryNode, "NODE MODULES"},
		{CategoryPython, "PYTHON CACHE"},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			result := CategoryTitle(tt.category)
			if result != tt.expected {
				t.Errorf("CategoryTitle(%q) = %q, want %q", tt.category, result, tt.expected)
			}
		})
	}
}