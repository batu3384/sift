// Package theme provides rendering utility functions for the SIFT TUI.
package theme

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const (
	DefaultViewWidth  = 118
	DefaultViewHeight = 34
	MinWidth          = 84
	MinHeight         = 20
)

// EffectiveSize returns the effective dimensions, applying minimum bounds.
func EffectiveSize(width, height int) (int, int) {
	if width <= 0 {
		width = DefaultViewWidth
	}
	if height <= 0 {
		height = DefaultViewHeight
	}
	if width < MinWidth {
		width = MinWidth
	}
	if height < MinHeight {
		height = MinHeight
	}
	return width, height
}

// CompactWidth returns true if the width indicates a compact layout.
func CompactWidth(width int) bool {
	return width < 100
}

// CompactHeight returns true if the height indicates a compact layout.
func CompactHeight(height int) bool {
	return height < 26
}

// TruncateText truncates text to fit within max width using ANSI-aware truncation.
func TruncateText(text string, max int) string {
	if max <= 0 {
		return text
	}
	if ansi.StringWidth(text) <= max {
		return text
	}
	if max <= 1 {
		return ansi.Truncate(text, max, "")
	}
	return strings.TrimSpace(ansi.Truncate(text, max, "…"))
}

// SingleLine converts text to a single line, truncating if needed.
func SingleLine(text string, max int) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")
	return TruncateText(text, max)
}

// SplitColumns calculates left and right column widths given total width and ratios.
func SplitColumns(totalWidth int, leftRatio float64, minLeft int, minRight int) (int, int) {
	if totalWidth <= 0 {
		return minLeft, minRight
	}
	left := int(float64(totalWidth) * leftRatio)
	right := totalWidth - left
	if left < minLeft {
		left = minLeft
		right = totalWidth - left
	}
	if right < minRight {
		right = minRight
		left = totalWidth - right
	}
	if left < minLeft {
		left = minLeft
	}
	if right < minRight {
		right = minRight
	}
	return left, right
}

// AppendInterleaved inserts spacer between values.
func AppendInterleaved(values []string, spacer string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values)*2-1)
	for idx, value := range values {
		if idx > 0 {
			out = append(out, spacer)
		}
		out = append(out, value)
	}
	return out
}

// RenderedLineCount returns the number of lines in a string.
func RenderedLineCount(content string) int {
	if content == "" {
		return 0
	}
	return len(strings.Split(content, "\n"))
}

// BodyLineBudget calculates available body lines given total height and reserved lines.
func BodyLineBudget(height int, reserve int, minLines int) int {
	_, effectiveHeight := EffectiveSize(DefaultViewWidth, height)
	lines := effectiveHeight - reserve
	if lines < minLines {
		return minLines
	}
	return lines
}

// ViewportLines returns a window of lines centered around the cursor.
func ViewportLines(lines []string, cursor int, maxLines int) []string {
	if maxLines <= 0 || len(lines) <= maxLines {
		return lines
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(lines) {
		cursor = len(lines) - 1
	}
	start := cursor - maxLines/2
	if start < 0 {
		start = 0
	}
	end := start + maxLines
	if end > len(lines) {
		end = len(lines)
		start = end - maxLines
		if start < 0 {
			start = 0
		}
	}
	window := append([]string{}, lines[start:end]...)
	if start > 0 {
		window[0] = MutedStyle.Render("…") + " " + strings.TrimLeft(window[0], " ")
	}
	if end < len(lines) {
		window[len(window)-1] = strings.TrimRight(window[len(window)-1], " ") + " " + MutedStyle.Render("…")
	}
	return window
}

// ClipRendered clips content to maxLines, adding ellipsis if truncated.
func ClipRendered(content string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	if len(lines) <= maxLines {
		return content
	}
	clipped := append([]string{}, lines[:maxLines]...)
	if maxLines > 0 {
		last := strings.TrimRight(clipped[maxLines-1], " ")
		if last == "" {
			last = MutedStyle.Render("…")
		} else {
			last = TruncateText(last, max(DefaultViewWidth/2, 16))
			last += " " + MutedStyle.Render("…")
		}
		clipped[maxLines-1] = last
	}
	return strings.Join(clipped, "\n")
}

// TrimStatsForHeight reduces stat cards based on available height.
func TrimStatsForHeight(cards []string, height int, hero bool) []string {
	if len(cards) == 0 {
		return cards
	}
	if hero {
		if height <= 24 {
			return cards[:min(len(cards), 2)]
		}
		if height < 28 {
			return cards[:min(len(cards), 3)]
		}
		return cards
	}
	if height < 22 {
		return cards[:min(len(cards), 1)]
	}
	if height < 26 {
		return cards[:min(len(cards), 2)]
	}
	return cards
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}