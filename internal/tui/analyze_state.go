package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/batuhanyuksel/sift/internal/analyze"
	"github.com/batuhanyuksel/sift/internal/domain"
)

func canDescendInto(item domain.Finding) bool {
	if item.Category != domain.CategoryDiskUsage {
		return false
	}
	return os.FileMode(item.Fingerprint.Mode).IsDir()
}

func (m analyzeBrowserModel) selectedItem() (domain.Finding, bool) {
	visible := m.visibleIndices()
	if m.cursor < 0 || m.cursor >= len(visible) {
		return domain.Finding{}, false
	}
	return m.plan.Items[visible[m.cursor]], true
}

func (m analyzeBrowserModel) selectedQueuedItem() (domain.Finding, bool) {
	order := m.sortedStageOrder()
	if m.queueCursor < 0 || m.queueCursor >= len(order) {
		return domain.Finding{}, false
	}
	item, ok := m.staged[order[m.queueCursor]]
	return item, ok
}

func (m analyzeBrowserModel) selectedActiveItem() (domain.Finding, bool) {
	if m.activePane() == analyzePaneQueue {
		return m.selectedQueuedItem()
	}
	return m.selectedItem()
}

func (m analyzeBrowserModel) currentSelectionPath() string {
	if item, ok := m.selectedActiveItem(); ok {
		if path := strings.TrimSpace(item.Path); path != "" {
			return path
		}
	}
	if item, ok := m.selectedItem(); ok {
		return strings.TrimSpace(item.Path)
	}
	return ""
}

func (m analyzeBrowserModel) selectedPreview() analyzeDirectoryPreview {
	if item, ok := m.selectedActiveItem(); ok {
		if preview, ok := m.previewCache[strings.TrimSpace(item.Path)]; ok {
			return preview
		}
	}
	if item, ok := m.selectedItem(); ok {
		if preview, ok := m.previewCache[strings.TrimSpace(item.Path)]; ok {
			return preview
		}
	}
	return analyzeDirectoryPreview{}
}

func loadAnalyzeTarget(loader analyzeLoader, target string) tea.Cmd {
	return func() tea.Msg {
		plan, err := loader(target)
		return analyzeLoadedMsg{plan: plan, err: err}
	}
}

func loadAnalyzeReview(loader analyzeReviewLoader, paths []string) tea.Cmd {
	return func() tea.Msg {
		plan, err := loader(paths)
		return analyzeReviewLoadedMsg{plan: plan, err: err}
	}
}

func canStage(item domain.Finding) bool {
	return item.Path != "" && item.Status == domain.StatusAdvisory
}

func (m *analyzeBrowserModel) toggleStage(item domain.Finding) {
	if m.staged == nil {
		m.staged = map[string]domain.Finding{}
	}
	if _, ok := m.staged[item.Path]; ok {
		m.removeStage(item.Path)
		return
	}
	m.staged[item.Path] = item
	m.stageOrder = append(m.stageOrder, item.Path)
	if len(m.stageOrder) == 1 {
		m.queueCursor = 0
	}
	m.reviewPreview = menuPreviewState{}
}

func (m *analyzeBrowserModel) syncPreviewWindow() {
	if m.previewCache == nil {
		m.previewCache = map[string]analyzeDirectoryPreview{}
	}
	next := make(map[string]analyzeDirectoryPreview, 4)
	missing := make([]string, 0, 4)
	for _, item := range m.previewCandidates() {
		path := strings.TrimSpace(item.Path)
		if path == "" {
			continue
		}
		if preview, ok := m.previewCache[path]; ok {
			next[path] = preview
			continue
		}
		missing = append(missing, path)
	}
	if len(missing) > 0 && m.previewLoader != nil {
		for path, preview := range m.previewLoader(missing) {
			next[strings.TrimSpace(path)] = preview
		}
	}
	for _, item := range m.previewCandidates() {
		path := strings.TrimSpace(item.Path)
		if path == "" {
			continue
		}
		if _, ok := next[path]; ok {
			continue
		}
		if preview, ok := analyze.Preview(path); ok {
			next[path] = preview
		}
	}
	m.previewCache = next
}

func (m analyzeBrowserModel) previewCandidates() []domain.Finding {
	paths := map[string]struct{}{}
	candidates := make([]domain.Finding, 0, 4)
	appendCandidate := func(item domain.Finding, ok bool) {
		if !ok || item.Category != domain.CategoryDiskUsage || strings.TrimSpace(item.Path) == "" {
			return
		}
		path := strings.TrimSpace(item.Path)
		if _, seen := paths[path]; seen {
			return
		}
		paths[path] = struct{}{}
		candidates = append(candidates, item)
	}
	appendCandidate(m.selectedActiveItem())
	appendCandidate(m.selectedItem())
	visible := m.visibleIndices()
	if len(visible) == 0 {
		return candidates
	}
	for _, offset := range []int{-1, 1} {
		index := m.cursor + offset
		if index < 0 || index >= len(visible) {
			continue
		}
		appendCandidate(m.plan.Items[visible[index]], true)
	}
	return candidates
}

func (m *analyzeBrowserModel) removeStage(path string) {
	if m.staged == nil {
		return
	}
	delete(m.staged, path)
	filtered := make([]string, 0, len(m.stageOrder))
	for _, itemPath := range m.stageOrder {
		if itemPath == path {
			continue
		}
		filtered = append(filtered, itemPath)
	}
	m.stageOrder = filtered
	m.reviewPreview = menuPreviewState{}
	m.clampQueueCursor()
	if len(m.stageOrder) == 0 {
		m.pane = analyzePaneBrowse
	}
}

func (m *analyzeBrowserModel) setReviewPreviewLoading(key string) {
	m.reviewPreview = menuPreviewState{key: key, loading: strings.TrimSpace(key) != ""}
}

func (m *analyzeBrowserModel) applyReviewPreview(key string, plan domain.ExecutionPlan, err error) {
	preview := menuPreviewState{key: key}
	if err != nil {
		preview.err = err.Error()
		m.reviewPreview = preview
		return
	}
	preview.plan = plan
	preview.loaded = true
	m.reviewPreview = preview
}

func (m analyzeBrowserModel) reviewPreviewPaths() []string {
	if paths := m.stagedPaths(); len(paths) > 0 {
		return paths
	}
	item, ok := m.selectedItem()
	if !ok || !canStage(item) {
		return nil
	}
	return []string{item.Path}
}

func (m analyzeBrowserModel) stagedReviewKey() string {
	paths := m.reviewPreviewPaths()
	if len(paths) == 0 {
		return ""
	}
	return strings.Join(paths, "\x1f")
}

func (m analyzeBrowserModel) reviewPreviewPlan() (domain.ExecutionPlan, bool) {
	key := m.stagedReviewKey()
	if key == "" || !m.reviewPreview.loaded || strings.TrimSpace(m.reviewPreview.key) != key {
		return domain.ExecutionPlan{}, false
	}
	return m.reviewPreview.plan, true
}

func (m analyzeBrowserModel) hasQueuedBatch() bool {
	return len(m.stageOrder) > 1
}

func analyzeQueueFocusLabel(item domain.Finding) string {
	label := strings.TrimSpace(item.Name)
	if label == "" {
		label = filepath.Base(strings.TrimSpace(item.Path))
	}
	if label == "" {
		label = item.DisplayPath
	}
	if label == "" {
		label = item.Path
	}
	return label
}

func (m analyzeBrowserModel) stagedPaths() []string {
	out := make([]string, 0, len(m.stageOrder))
	for _, path := range m.stageOrder {
		if _, ok := m.staged[path]; ok {
			out = append(out, path)
		}
	}
	return out
}

func (m *analyzeBrowserModel) startSearch() {
	if m.search.CharLimit == 0 {
		m.search = newAnalyzeSearchInput()
	}
	m.searchActive = true
	m.search.Focus()
}

func (m *analyzeBrowserModel) stopSearch(clear bool) {
	if clear {
		m.search.SetValue("")
	}
	m.searchActive = false
	m.search.Blur()
	m.clampCursor()
}

func (m *analyzeBrowserModel) cycleFilter() {
	switch m.filter {
	case analyzeFilterQueued:
		m.filter = analyzeFilterHigh
	case analyzeFilterHigh:
		m.filter = analyzeFilterAll
	default:
		m.filter = analyzeFilterQueued
	}
	m.clampCursor()
}

func (m *analyzeBrowserModel) cycleQueueSort() {
	switch coalesceAnalyzeQueueSort(m.queueSort) {
	case analyzeQueueSortSize:
		m.queueSort = analyzeQueueSortAge
	case analyzeQueueSortAge:
		m.queueSort = analyzeQueueSortOrder
	default:
		m.queueSort = analyzeQueueSortSize
	}
	m.clampQueueCursor()
}

func (m *analyzeBrowserModel) cyclePane() {
	if !m.hasQueuedBatch() {
		m.pane = analyzePaneBrowse
		return
	}
	switch m.activePane() {
	case analyzePaneQueue:
		m.pane = analyzePaneBrowse
	default:
		m.pane = analyzePaneQueue
	}
}

func (m *analyzeBrowserModel) clampCursor() {
	visible := m.visibleIndices()
	if len(visible) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(visible) {
		m.cursor = len(visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m analyzeBrowserModel) cursorForPath(path string) (int, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return 0, false
	}
	visible := m.visibleIndices()
	for cursor, index := range visible {
		if strings.TrimSpace(m.plan.Items[index].Path) == path {
			return cursor, true
		}
	}
	return 0, false
}

func (m analyzeBrowserModel) queueCursorForPath(path string) (int, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return 0, false
	}
	order := m.sortedStageOrder()
	for idx, itemPath := range order {
		if strings.TrimSpace(itemPath) == path {
			return idx, true
		}
	}
	return 0, false
}

func (m analyzeBrowserModel) fallbackSelectionPathAfterRemoval(paths []string) string {
	if len(paths) == 0 {
		return m.currentSelectionPath()
	}
	removed := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path != "" {
			removed[path] = struct{}{}
		}
	}
	visible := m.visibleIndices()
	remaining := make([]int, 0, len(visible))
	for _, index := range visible {
		if _, ok := removed[strings.TrimSpace(m.plan.Items[index].Path)]; ok {
			continue
		}
		remaining = append(remaining, index)
	}
	if len(remaining) == 0 {
		return ""
	}
	cursor := m.cursor
	if cursor >= len(remaining) {
		cursor = len(remaining) - 1
	}
	if cursor < 0 {
		cursor = 0
	}
	return strings.TrimSpace(m.plan.Items[remaining[cursor]].Path)
}

func (m *analyzeBrowserModel) clampQueueCursor() {
	order := m.sortedStageOrder()
	if len(order) == 0 {
		m.queueCursor = 0
		return
	}
	if m.queueCursor >= len(order) {
		m.queueCursor = len(order) - 1
	}
	if m.queueCursor < 0 {
		m.queueCursor = 0
	}
}

func (m analyzeBrowserModel) activePane() analyzePane {
	if m.pane == analyzePaneQueue && m.hasQueuedBatch() {
		return analyzePaneQueue
	}
	return analyzePaneBrowse
}

func (m analyzeBrowserModel) visibleIndices() []int {
	indices := make([]int, 0, len(m.plan.Items))
	query := strings.ToLower(strings.TrimSpace(m.search.Value()))
	for idx, item := range m.plan.Items {
		if analyzeFilterMatch(m.filter, item, m.staged) && analyzeSearchMatch(item, query) {
			indices = append(indices, idx)
		}
	}
	return indices
}

func (m analyzeBrowserModel) sortedStageOrder() []string {
	order := append([]string{}, m.stageOrder...)
	switch coalesceAnalyzeQueueSort(m.queueSort) {
	case analyzeQueueSortSize:
		sort.SliceStable(order, func(i, j int) bool {
			left, lok := m.staged[order[i]]
			right, rok := m.staged[order[j]]
			if !lok || !rok {
				return order[i] < order[j]
			}
			if left.Bytes == right.Bytes {
				return left.DisplayPath < right.DisplayPath
			}
			return left.Bytes > right.Bytes
		})
	case analyzeQueueSortAge:
		sort.SliceStable(order, func(i, j int) bool {
			left, lok := m.staged[order[i]]
			right, rok := m.staged[order[j]]
			if !lok || !rok {
				return order[i] < order[j]
			}
			if left.LastModified.Equal(right.LastModified) {
				return left.DisplayPath < right.DisplayPath
			}
			if left.LastModified.IsZero() {
				return false
			}
			if right.LastModified.IsZero() {
				return true
			}
			return left.LastModified.Before(right.LastModified)
		})
	}
	return order
}

func queueIndexForPath(order []string, path string) int {
	for idx, candidate := range order {
		if candidate == path {
			return idx
		}
	}
	return -1
}
