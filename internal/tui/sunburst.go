package tui

import (
	"math"
	"sort"
	"strings"

	"github.com/batuhanyuksel/sift/internal/domain"
)

// SunburstSegment represents a single segment in the sunburst chart
type SunburstSegment struct {
	Name     string
	Path     string
	Bytes    int64
	Depth    int
	StartAngle float64
	EndAngle   float64
	Children   []*SunburstSegment
	Parent     *SunburstSegment
}

// SunburstChart represents the complete sunburst visualization
type SunburstChart struct {
	Root      *SunburstSegment
	MaxDepth  int
	TotalBytes int64
}

// BuildSunburstChart creates a sunburst chart from analysis findings
func BuildSunburstChart(findings []domain.Finding) *SunburstChart {
	if len(findings) == 0 {
		return &SunburstChart{
			Root: &SunburstSegment{
				Name:  "root",
				Bytes: 0,
			},
		}
	}

	// Group findings by path hierarchy
	root := &SunburstSegment{
		Name:   "root",
		Path:   "/",
		Bytes:  0,
		Depth:  0,
	}

	// Build tree structure
	for _, finding := range findings {
		addFindingToTree(root, finding)
	}

	// Calculate total bytes
	total := calculateTotalBytes(root)

	// Calculate angles
	calculateAngles(root, 0, 2*math.Pi, total)

	// Find max depth
	maxDepth := findMaxDepth(root, 0)

	return &SunburstChart{
		Root:       root,
		MaxDepth:   maxDepth,
		TotalBytes: total,
	}
}

// addFindingToTree adds a finding to the tree structure
func addFindingToTree(root *SunburstSegment, finding domain.Finding) {
	parts := splitPath(finding.Path)
	current := root

	for i, part := range parts {
		if part == "" {
			continue
		}

		// Find or create child
		var child *SunburstSegment
		for _, c := range current.Children {
			if c.Name == part {
				child = c
				break
			}
		}

		if child == nil {
			child = &SunburstSegment{
				Name:   part,
				Path:   buildPath(parts[:i+1]),
				Depth:  i + 1,
				Parent: current,
			}
			current.Children = append(current.Children, child)
		}

		// Add bytes to this level
		child.Bytes += finding.Bytes
		current = child
	}
}

// calculateTotalBytes recursively calculates total bytes
func calculateTotalBytes(node *SunburstSegment) int64 {
	if len(node.Children) == 0 {
		return node.Bytes
	}

	var total int64
	for _, child := range node.Children {
		total += calculateTotalBytes(child)
	}
	node.Bytes = total
	return total
}

// calculateAngles recursively calculates start and end angles for each segment
func calculateAngles(node *SunburstSegment, startAngle, endAngle float64, totalBytes int64) {
	if node.Bytes == 0 || len(node.Children) == 0 {
		return
	}

	node.StartAngle = startAngle
	node.EndAngle = endAngle

	// Sort children by size (largest first)
	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Bytes > node.Children[j].Bytes
	})

	// Calculate angles for children
	angleRange := endAngle - startAngle
	currentAngle := startAngle

	for _, child := range node.Children {
		childRatio := float64(child.Bytes) / float64(node.Bytes)
		childAngleRange := angleRange * childRatio
		childEndAngle := currentAngle + childAngleRange

		calculateAngles(child, currentAngle, childEndAngle, totalBytes)
		currentAngle = childEndAngle
	}
}

// findMaxDepth finds the maximum depth in the tree
func findMaxDepth(node *SunburstSegment, currentDepth int) int {
	maxDepth := currentDepth
	for _, child := range node.Children {
		childDepth := findMaxDepth(child, currentDepth+1)
		if childDepth > maxDepth {
			maxDepth = childDepth
		}
	}
	return maxDepth
}

// splitPath splits a path into components
func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}

// buildPath builds a path from components
func buildPath(parts []string) string {
	return "/" + strings.Join(parts, "/")
}

// GetTopSegments returns the top N largest segments at a given depth
func (sc *SunburstChart) GetTopSegments(depth, n int) []*SunburstSegment {
	var segments []*SunburstSegment
	collectSegmentsAtDepth(sc.Root, depth, &segments)

	// Sort by size
	sort.Slice(segments, func(i, j int) bool {
		return segments[i].Bytes > segments[j].Bytes
	})

	if len(segments) > n {
		return segments[:n]
	}
	return segments
}

// collectSegmentsAtDepth recursively collects segments at a specific depth
func collectSegmentsAtDepth(node *SunburstSegment, targetDepth int, result *[]*SunburstSegment) {
	if node.Depth == targetDepth {
		*result = append(*result, node)
		return
	}

	for _, child := range node.Children {
		collectSegmentsAtDepth(child, targetDepth, result)
	}
}

// GetSegmentAtAngle finds the segment at a specific angle and depth
func (sc *SunburstChart) GetSegmentAtAngle(angle float64, depth int) *SunburstSegment {
	return findSegmentAtAngle(sc.Root, angle, depth)
}

// findSegmentAtAngle recursively finds segment at angle
func findSegmentAtAngle(node *SunburstSegment, angle float64, targetDepth int) *SunburstSegment {
	if node.Depth == targetDepth {
		if angle >= node.StartAngle && angle <= node.EndAngle {
			return node
		}
		return nil
	}

	for _, child := range node.Children {
		if angle >= child.StartAngle && angle <= child.EndAngle {
			return findSegmentAtAngle(child, angle, targetDepth)
		}
	}

	return nil
}

// GetPathToRoot returns the path from a segment to root
func (s *SunburstSegment) GetPathToRoot() []*SunburstSegment {
	var path []*SunburstSegment
	current := s
	for current != nil {
		path = append([]*SunburstSegment{current}, path...)
		current = current.Parent
	}
	return path
}

// GetPercentage returns the percentage of total this segment represents
func (s *SunburstSegment) GetPercentage(total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(s.Bytes) / float64(total) * 100
}
