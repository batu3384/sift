package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/batu3384/sift/internal/domain"
)

// SunburstRenderer renders a sunburst chart as ASCII art
type SunburstRenderer struct {
	Width      int
	Height     int
	MaxDepth   int
	ColorScheme []lipgloss.Color
}

// NewSunburstRenderer creates a new sunburst renderer
func NewSunburstRenderer(width, height int) *SunburstRenderer {
	return &SunburstRenderer{
		Width:    width,
		Height:   height,
		MaxDepth: 3, // Show up to 3 levels deep
		ColorScheme: []lipgloss.Color{
			lipgloss.Color("#50FA7B"), // Green
			lipgloss.Color("#8BE9FD"), // Cyan
			lipgloss.Color("#BD93F9"), // Purple
			lipgloss.Color("#FFB86C"), // Orange
			lipgloss.Color("#FF79C6"), // Pink
			lipgloss.Color("#F1FA8C"), // Yellow
			lipgloss.Color("#FF5555"), // Red
			lipgloss.Color("#6272A4"), // Comment
		},
	}
}

// Render renders the sunburst chart
func (sr *SunburstRenderer) Render(chart *SunburstChart) string {
	if chart.Root == nil || chart.TotalBytes == 0 {
		return mutedStyle.Render("No data to visualize")
	}

	// Use a square aspect ratio
	size := sr.Width
	if sr.Height < size {
		size = sr.Height
	}
	if size < 20 {
		size = 20
	}

	// Create the canvas
	canvas := make([][]rune, size)
	colors := make([][]lipgloss.Color, size)
	for i := range canvas {
		canvas[i] = make([]rune, size)
		colors[i] = make([]lipgloss.Color, size)
		for j := range canvas[i] {
			canvas[i][j] = ' '
		}
	}

	centerX := size / 2
	centerY := size / 2
	maxRadius := size / 2 - 1

	// Draw each level
	for depth := 0; depth <= sr.MaxDepth && depth <= chart.MaxDepth; depth++ {
		innerRadius := (maxRadius * depth) / (sr.MaxDepth + 1)
		outerRadius := (maxRadius * (depth + 1)) / (sr.MaxDepth + 1)

		if depth == 0 {
			// Draw center circle
			sr.drawCircle(canvas, colors, centerX, centerY, outerRadius, chart.Root, 0)
		} else {
			// Draw ring
			sr.drawRing(canvas, colors, centerX, centerY, innerRadius, outerRadius, chart.Root, depth)
		}
	}

	// Convert canvas to string
	return sr.canvasToString(canvas, colors, size)
}

// drawCircle draws a filled circle
func (sr *SunburstRenderer) drawCircle(canvas [][]rune, colors [][]lipgloss.Color, centerX, centerY, radius int, node *SunburstSegment, depth int) {
	color := sr.ColorScheme[depth%len(sr.ColorScheme)]

	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			if x*x+y*y <= radius*radius {
				px, py := centerX+x, centerY+y
				if px >= 0 && px < len(canvas[0]) && py >= 0 && py < len(canvas) {
					canvas[py][px] = '█'
					colors[py][px] = color
				}
			}
		}
	}
}

// drawRing draws a ring segment
func (sr *SunburstRenderer) drawRing(canvas [][]rune, colors [][]lipgloss.Color, centerX, centerY, innerRadius, outerRadius int, node *SunburstSegment, targetDepth int) {
	segments := sr.getSegmentsAtDepth(node, targetDepth)

	for _, segment := range segments {
		color := sr.ColorScheme[segment.Depth%len(sr.ColorScheme)]
		sr.drawSegment(canvas, colors, centerX, centerY, innerRadius, outerRadius, segment, color)
	}
}

// getSegmentsAtDepth returns all segments at a specific depth
func (sr *SunburstRenderer) getSegmentsAtDepth(node *SunburstSegment, targetDepth int) []*SunburstSegment {
	var result []*SunburstSegment
	if node.Depth == targetDepth {
		return []*SunburstSegment{node}
	}

	for _, child := range node.Children {
		result = append(result, sr.getSegmentsAtDepth(child, targetDepth)...)
	}
	return result
}

// drawSegment draws a single segment of the ring
func (sr *SunburstRenderer) drawSegment(canvas [][]rune, colors [][]lipgloss.Color, centerX, centerY, innerRadius, outerRadius int, segment *SunburstSegment, color lipgloss.Color) {
	// Convert angles to degrees for iteration
	startDeg := segment.StartAngle * 180 / math.Pi
	endDeg := segment.EndAngle * 180 / math.Pi

	// Ensure we draw at least a small segment
	if endDeg-startDeg < 1 {
		endDeg = startDeg + 1
	}

	for angle := startDeg; angle <= endDeg; angle += 0.5 {
		rad := angle * math.Pi / 180

		// Draw from inner to outer radius
		for r := innerRadius; r <= outerRadius; r++ {
			x := centerX + int(float64(r)*math.Cos(rad))
			y := centerY + int(float64(r)*math.Sin(rad))

			if x >= 0 && x < len(canvas[0]) && y >= 0 && y < len(canvas) {
				canvas[y][x] = '█'
				colors[y][x] = color
			}
		}
	}
}

// canvasToString converts the canvas to a colored string
func (sr *SunburstRenderer) canvasToString(canvas [][]rune, colors [][]lipgloss.Color, size int) string {
	var lines []string

	for y := 0; y < size; y++ {
		var line strings.Builder
		currentColor := lipgloss.Color("")
		var segment strings.Builder

		for x := 0; x < size; x++ {
			char := canvas[y][x]
			color := colors[y][x]

			if color != currentColor {
				// Flush current segment
				if segment.Len() > 0 {
					if currentColor != "" {
						line.WriteString(lipgloss.NewStyle().Foreground(currentColor).Render(segment.String()))
					} else {
						line.WriteString(segment.String())
					}
					segment.Reset()
				}
				currentColor = color
			}

			if char == ' ' {
				segment.WriteRune(' ')
			} else {
				segment.WriteRune(char)
			}
		}

		// Flush final segment
		if segment.Len() > 0 {
			if currentColor != "" {
				line.WriteString(lipgloss.NewStyle().Foreground(currentColor).Render(segment.String()))
			} else {
				line.WriteString(segment.String())
			}
		}

		lines = append(lines, line.String())
	}

	return strings.Join(lines, "\n")
}

// RenderSunburstWithLegend renders the sunburst with a legend
func (sr *SunburstRenderer) RenderSunburstWithLegend(chart *SunburstChart, width int) string {
	// Calculate layout
	chartWidth := width * 2 / 3
	legendWidth := width - chartWidth - 2

	if legendWidth < 20 {
		chartWidth = width / 2
		legendWidth = width - chartWidth - 2
	}

	// Render chart
	chartStr := sr.Render(chart)

	// Build legend
	var legendLines []string
	legendLines = append(legendLines, panelTitleStyle.Render("Disk Usage Map"))
	legendLines = append(legendLines, "")
	legendLines = append(legendLines, fmt.Sprintf("Total: %s", domain.HumanBytes(chart.TotalBytes)))
	legendLines = append(legendLines, "")

	// Top segments by depth
	for depth := 1; depth <= sr.MaxDepth && depth <= chart.MaxDepth; depth++ {
		segments := chart.GetTopSegments(depth, 5)
		if len(segments) == 0 {
			continue
		}

		legendLines = append(legendLines, subtitleStyle.Render(fmt.Sprintf("Level %d", depth)))

		for i, seg := range segments {
			if i >= 3 { // Show top 3 per level
				break
			}
			color := sr.ColorScheme[seg.Depth%len(sr.ColorScheme)]
			percentage := seg.GetPercentage(chart.TotalBytes)
			line := fmt.Sprintf("  %s %s (%s, %.1f%%)",
				lipgloss.NewStyle().Foreground(color).Render("■"),
				truncateText(seg.Name, legendWidth-15),
				domain.HumanBytes(seg.Bytes),
				percentage,
			)
			legendLines = append(legendLines, line)
		}
		legendLines = append(legendLines, "")
	}

	legendStr := strings.Join(legendLines, "\n")

	// Combine chart and legend side by side
	chartLines := strings.Split(chartStr, "\n")
	legendLines = strings.Split(legendStr, "\n")

	var combined []string
	maxLines := len(chartLines)
	if len(legendLines) > maxLines {
		maxLines = len(legendLines)
	}

	for i := 0; i < maxLines; i++ {
		var chartLine, legendLine string
		if i < len(chartLines) {
			chartLine = chartLines[i]
		}
		if i < len(legendLines) {
			legendLine = legendLines[i]
		}

		// Pad chart line to fixed width
		chartLine = padRight(chartLine, chartWidth)

		combined = append(combined, chartLine+"  "+legendLine)
	}

	return strings.Join(combined, "\n")
}

// padRight pads a string to a minimum width
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
