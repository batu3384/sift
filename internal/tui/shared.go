package tui

import "strings"

// pl returns singular when n == 1, otherwise plural.
func pl(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func selectionPrefix(selected bool) string {
	if selected {
		return "▸ "
	}
	return "· "
}

// titleCase uppercases the first letter of s. Used instead of the deprecated
// strings.Title for single-word command and phase labels.
func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
