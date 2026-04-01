package tui

import (
	"testing"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func TestBuildSunburstChart(t *testing.T) {
	findings := []domain.Finding{
		{Path: "/home/user/docs/file1.txt", Bytes: 1000, Name: "file1.txt"},
		{Path: "/home/user/docs/file2.txt", Bytes: 2000, Name: "file2.txt"},
		{Path: "/home/user/pics/photo.jpg", Bytes: 5000, Name: "photo.jpg"},
		{Path: "/home/user/pics/icon.png", Bytes: 1000, Name: "icon.png"},
		{Path: "/home/user/music/song.mp3", Bytes: 10000, Name: "song.mp3"},
	}

	chart := BuildSunburstChart(findings)

	if chart.Root == nil {
		t.Fatal("Expected root node")
	}

	if chart.TotalBytes != 19000 {
		t.Errorf("Expected total bytes 19000, got %d", chart.TotalBytes)
	}

	// Check root has children
	if len(chart.Root.Children) == 0 {
		t.Error("Expected root to have children")
	}
}

func TestBuildSunburstChartEmpty(t *testing.T) {
	chart := BuildSunburstChart([]domain.Finding{})

	if chart.Root == nil {
		t.Fatal("Expected root node even for empty chart")
	}

	if chart.TotalBytes != 0 {
		t.Error("Expected 0 bytes for empty chart")
	}
}

func TestSunburstSegmentGetPercentage(t *testing.T) {
	seg := &SunburstSegment{
		Name:  "test",
		Bytes: 2500,
	}

	percentage := seg.GetPercentage(10000)
	if percentage != 25.0 {
		t.Errorf("Expected 25%%, got %.1f%%", percentage)
	}

	// Test with 0 total
	percentage = seg.GetPercentage(0)
	if percentage != 0 {
		t.Errorf("Expected 0%% with 0 total, got %.1f%%", percentage)
	}
}

func TestGetTopSegments(t *testing.T) {
	findings := []domain.Finding{
		{Path: "/a/file1.txt", Bytes: 10000, Name: "file1.txt"},
		{Path: "/b/file2.txt", Bytes: 5000, Name: "file2.txt"},
		{Path: "/c/file3.txt", Bytes: 3000, Name: "file3.txt"},
		{Path: "/d/file4.txt", Bytes: 2000, Name: "file4.txt"},
	}

	chart := BuildSunburstChart(findings)

	// Get top 2 segments at depth 1
	topSegments := chart.GetTopSegments(1, 2)

	if len(topSegments) != 2 {
		t.Errorf("Expected 2 top segments, got %d", len(topSegments))
	}

	// First should be largest
	if len(topSegments) > 0 && topSegments[0].Bytes < topSegments[1].Bytes {
		t.Error("Top segments should be sorted by size (descending)")
	}
}

func TestGetPathToRoot(t *testing.T) {
	root := &SunburstSegment{Name: "root", Depth: 0}
	child := &SunburstSegment{Name: "child", Depth: 1, Parent: root}
	grandchild := &SunburstSegment{Name: "grandchild", Depth: 2, Parent: child}

	path := grandchild.GetPathToRoot()

	if len(path) != 3 {
		t.Errorf("Expected path length 3, got %d", len(path))
	}

	if path[0].Name != "root" {
		t.Error("First element should be root")
	}

	if path[2].Name != "grandchild" {
		t.Error("Last element should be grandchild")
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"/a/b/c", []string{"a", "b", "c"}},
		{"/a", []string{"a"}},
		{"/", []string{}},
		{"", []string{}},
	}

	for _, test := range tests {
		result := splitPath(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("splitPath(%s): expected %v, got %v", test.input, test.expected, result)
			continue
		}
		for i := range result {
			if result[i] != test.expected[i] {
				t.Errorf("splitPath(%s)[%d]: expected %s, got %s", test.input, i, test.expected[i], result[i])
			}
		}
	}
}

func TestSunburstRenderer(t *testing.T) {
	findings := []domain.Finding{
		{Path: "/home/user/docs/file1.txt", Bytes: 1000, Name: "file1.txt"},
		{Path: "/home/user/docs/file2.txt", Bytes: 2000, Name: "file2.txt"},
		{Path: "/home/user/pics/photo.jpg", Bytes: 5000, Name: "photo.jpg"},
	}

	chart := BuildSunburstChart(findings)
	renderer := NewSunburstRenderer(40, 20)

	output := renderer.Render(chart)

	if output == "" {
		t.Error("Expected non-empty render output")
	}

	if output == "No data to visualize" {
		t.Error("Should render chart, not empty message")
	}
}

func TestSunburstRendererEmpty(t *testing.T) {
	chart := BuildSunburstChart([]domain.Finding{})
	renderer := NewSunburstRenderer(40, 20)

	output := renderer.Render(chart)

	if output != "No data to visualize" {
		t.Errorf("Expected 'No data to visualize', got: %s", output)
	}
}

func TestRenderSunburstWithLegend(t *testing.T) {
	findings := []domain.Finding{
		{Path: "/home/user/docs/file1.txt", Bytes: 1000, Name: "file1.txt"},
		{Path: "/home/user/docs/file2.txt", Bytes: 2000, Name: "file2.txt"},
		{Path: "/home/user/pics/photo.jpg", Bytes: 5000, Name: "photo.jpg"},
	}

	chart := BuildSunburstChart(findings)
	renderer := NewSunburstRenderer(40, 20)

	output := renderer.RenderSunburstWithLegend(chart, 80)

	if output == "" {
		t.Error("Expected non-empty render output with legend")
	}

	// Should contain legend elements
	if !contains(output, "Disk Usage Map") {
		t.Error("Output should contain 'Disk Usage Map' title")
	}

	if !contains(output, "Total:") {
		t.Error("Output should contain 'Total:' label")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
