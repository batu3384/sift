package tui

import "testing"

func TestPanelThemeUsesDistinctAccentsByPanelType(t *testing.T) {
	t.Parallel()

	overviewBorder, overviewBG, _, _ := panelTheme("OVERVIEW", false)
	queueBorder, queueBG, _, _ := panelTheme("QUEUE", false)
	activeOverviewBorder, activeOverviewBG, _, _ := panelTheme("OVERVIEW", true)

	if string(overviewBorder) == string(queueBorder) {
		t.Fatalf("expected overview and queue borders to differ, got %q", overviewBorder)
	}
	if string(overviewBG) == string(queueBG) {
		t.Fatalf("expected overview and queue backgrounds to differ, got %q", overviewBG)
	}
	if string(activeOverviewBG) == string(overviewBG) {
		t.Fatalf("expected active overview background to brighten, got %q", activeOverviewBG)
	}
	if string(activeOverviewBorder) == string(queueBorder) {
		t.Fatalf("expected active overview border to stay distinct from queue border, got %q", activeOverviewBorder)
	}
}

func TestCardTonePaletteHonorsToneAndLabelOverrides(t *testing.T) {
	t.Parallel()

	safeBorder, safeLabel, _, safeBG := cardTonePalette("health", "safe")
	reviewBorder, reviewLabel, reviewValue, reviewBG := cardTonePalette("update", "review")
	highBorder, highLabel, highValue, highBG := cardTonePalette("alerts", "high")

	if string(safeBorder) == string(reviewBorder) || string(reviewBorder) == string(highBorder) {
		t.Fatalf("expected card borders to vary across tones/labels, got safe=%q review=%q high=%q", safeBorder, reviewBorder, highBorder)
	}
	if string(reviewLabel) == string(safeLabel) {
		t.Fatalf("expected update label palette to differ from safe health label, got %q", reviewLabel)
	}
	if string(highValue) == string(reviewValue) {
		t.Fatalf("expected high and review value colors to differ, got %q", highValue)
	}
	if string(safeBG) == string(reviewBG) || string(reviewBG) == string(highBG) {
		t.Fatalf("expected card backgrounds to vary, got safe=%q review=%q high=%q", safeBG, reviewBG, highBG)
	}
	if string(highLabel) == "" {
		t.Fatal("expected non-empty high label color")
	}
}
