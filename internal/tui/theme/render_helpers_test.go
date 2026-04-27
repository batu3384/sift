package theme

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestEffectiveSizeAppliesDefaultsAndMinimums(t *testing.T) {
	width, height := EffectiveSize(0, 0)
	if width != DefaultViewWidth || height != DefaultViewHeight {
		t.Fatalf("EffectiveSize(0, 0) = (%d, %d), want defaults (%d, %d)", width, height, DefaultViewWidth, DefaultViewHeight)
	}

	width, height = EffectiveSize(40, 10)
	if width != MinWidth || height != MinHeight {
		t.Fatalf("EffectiveSize below minimum = (%d, %d), want (%d, %d)", width, height, MinWidth, MinHeight)
	}
}

func TestTextHelpersKeepRenderedWidthWithinBudget(t *testing.T) {
	truncated := TruncateText("abcdef", 4)
	if ansi.StringWidth(truncated) > 4 {
		t.Fatalf("TruncateText width = %d, want <= 4 for %q", ansi.StringWidth(truncated), truncated)
	}

	single := SingleLine("  alpha\n\tbeta   gamma  ", 12)
	if strings.Contains(single, "\n") || strings.Contains(single, "\t") {
		t.Fatalf("SingleLine kept whitespace controls: %q", single)
	}
	if ansi.StringWidth(single) > 12 {
		t.Fatalf("SingleLine width = %d, want <= 12 for %q", ansi.StringWidth(single), single)
	}
}

func TestSplitColumnsHonorsMinimums(t *testing.T) {
	left, right := SplitColumns(120, 0.6, 30, 30)
	if left != 72 || right != 48 {
		t.Fatalf("SplitColumns balanced = (%d, %d), want (72, 48)", left, right)
	}

	left, right = SplitColumns(70, 0.8, 30, 28)
	if left < 30 || right < 28 {
		t.Fatalf("SplitColumns minimums = (%d, %d), want at least (30, 28)", left, right)
	}
	if left+right != 70 {
		t.Fatalf("SplitColumns total = %d, want 70", left+right)
	}
}

func TestViewportLinesCentersCursorAndMarksClippedEdges(t *testing.T) {
	lines := []string{"zero", "one", "two", "three", "four", "five"}
	window := ViewportLines(lines, 3, 3)

	if len(window) != 3 {
		t.Fatalf("window length = %d, want 3", len(window))
	}
	if !strings.Contains(window[0], "two") || !strings.Contains(window[0], "…") {
		t.Fatalf("first visible line = %q, want clipped prefix marker with original content", window[0])
	}
	if !strings.Contains(window[2], "four") || !strings.Contains(window[2], "…") {
		t.Fatalf("last visible line = %q, want clipped suffix marker with original content", window[2])
	}
}

func TestClipRenderedAndTrimStatsForHeight(t *testing.T) {
	clipped := ClipRendered("one\ntwo\nthree", 2)
	if RenderedLineCount(clipped) != 2 {
		t.Fatalf("clipped line count = %d, want 2", RenderedLineCount(clipped))
	}
	if !strings.Contains(clipped, "…") {
		t.Fatalf("clipped content = %q, want ellipsis marker", clipped)
	}

	cards := []string{"a", "b", "c", "d"}
	if got := TrimStatsForHeight(cards, 23, true); len(got) != 2 {
		t.Fatalf("hero cards at height 23 = %d, want 2", len(got))
	}
	if got := TrimStatsForHeight(cards, 25, false); len(got) != 2 {
		t.Fatalf("non-hero cards at height 25 = %d, want 2", len(got))
	}
}
