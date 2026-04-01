package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/batu3384/sift/internal/domain"
)

type uninstallItem struct {
	Name          string
	HasNative     bool
	Origin        string
	Location      string
	LastModified  string
	LastSeenAt    time.Time
	ApproxBytes   int64
	SizeLabel     string
	Sensitive     bool
	FamilyMatches []string
	RequiresAdmin bool
}

type uninstallModel struct {
	items         []uninstallItem
	filtered      []int
	cursor        int
	width         int
	height        int
	loading       bool
	loadingLabel  string
	staged        map[string]uninstallItem
	stageOrder    []string
	search        textinput.Model
	searchActive  bool
	message       string
	messageTicks  int
	lastRefreshed string
	preview       menuPreviewState
}

func newUninstallModel() uninstallModel {
	search := textinput.New()
	search.Prompt = "search> "
	search.Placeholder = "type app name"
	search.CharLimit = 128
	search.Width = 28
	return uninstallModel{
		search:   search,
		filtered: nil,
		staged:   map[string]uninstallItem{},
	}
}

func (m *uninstallModel) setItems(items []uninstallItem) {
	existingOrder := append([]string{}, m.stageOrder...)
	selectedKey := ""
	if selected, ok := m.selected(); ok {
		selectedKey = uninstallStageKey(selected.Name)
	}
	m.items = append([]uninstallItem{}, items...)
	sort.SliceStable(m.items, func(i, j int) bool {
		return compareUninstallItems(m.items[i], m.items[j])
	})
	index := map[string]uninstallItem{}
	for _, item := range m.items {
		index[uninstallStageKey(item.Name)] = item
	}
	nextStaged := map[string]uninstallItem{}
	nextOrder := make([]string, 0, len(existingOrder))
	for _, key := range existingOrder {
		if item, ok := index[key]; ok {
			nextStaged[key] = item
			nextOrder = append(nextOrder, key)
		}
	}
	m.staged = nextStaged
	m.stageOrder = nextOrder
	m.preview = menuPreviewState{}
	m.applyFilter()
	if cursor, ok := m.cursorForKey(selectedKey); ok {
		m.cursor = cursor
	} else {
		m.applyFilter()
	}
}

func (m *uninstallModel) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(m.search.Value()))
	m.filtered = m.filtered[:0]
	for idx, item := range m.items {
		if query == "" || uninstallMatchesQuery(item, query) {
			m.filtered = append(m.filtered, idx)
		}
	}
	if m.cursor >= len(m.filtered) {
		if len(m.filtered) == 0 {
			m.cursor = 0
		} else {
			m.cursor = len(m.filtered) - 1
		}
	}
}

func (m uninstallModel) selected() (uninstallItem, bool) {
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return uninstallItem{}, false
	}
	idx := m.filtered[m.cursor]
	if idx < 0 || idx >= len(m.items) {
		return uninstallItem{}, false
	}
	return m.items[idx], true
}

func (m uninstallModel) cursorForKey(key string) (int, bool) {
	key = uninstallStageKey(key)
	if key == "" {
		return 0, false
	}
	for filteredIdx, idx := range m.filtered {
		if idx < 0 || idx >= len(m.items) {
			continue
		}
		if uninstallStageKey(m.items[idx].Name) == key {
			return filteredIdx, true
		}
	}
	return 0, false
}

func (m *uninstallModel) startSearch() {
	m.searchActive = true
	m.search.Focus()
	m.setMessage("Type to filter installed apps.", routeMessageTicks)
}

func (m *uninstallModel) stopSearch() {
	m.searchActive = false
	m.search.Blur()
}

func (m *uninstallModel) setMessage(message string, ticks int) {
	m.message = message
	m.messageTicks = ticks
}

func (m *uninstallModel) setPreviewLoading(key string) {
	m.preview = menuPreviewState{key: key, loading: strings.TrimSpace(key) != ""}
}

func (m *uninstallModel) applyPreview(key string, plan domain.ExecutionPlan, err error) {
	preview := menuPreviewState{key: key}
	if err != nil {
		preview.err = err.Error()
		m.preview = preview
		return
	}
	preview.plan = plan
	preview.loaded = true
	m.preview = preview
}

func (m uninstallModel) previewPlanForSelected() (domain.ExecutionPlan, bool) {
	item, ok := m.selected()
	if !ok {
		return domain.ExecutionPlan{}, false
	}
	key := uninstallStageKey(item.Name)
	if !m.preview.loaded || strings.TrimSpace(m.preview.key) != key {
		return domain.ExecutionPlan{}, false
	}
	return m.preview.plan, true
}

func (m uninstallModel) stageNames() []string {
	names := make([]string, 0, len(m.stageOrder))
	for _, key := range m.stageOrder {
		if item, ok := m.staged[key]; ok {
			names = append(names, item.Name)
		}
	}
	return names
}

func (m uninstallModel) stageCount() int {
	return len(m.stageOrder)
}

func (m uninstallModel) isStaged(item uninstallItem) bool {
	_, ok := m.staged[uninstallStageKey(item.Name)]
	return ok
}

func (m *uninstallModel) toggleSelectedStage() (uninstallItem, bool, bool) {
	item, ok := m.selected()
	if !ok {
		return uninstallItem{}, false, false
	}
	return item, true, m.toggleStage(item)
}

func (m *uninstallModel) toggleStage(item uninstallItem) bool {
	key := uninstallStageKey(item.Name)
	if key == "" {
		return false
	}
	if _, ok := m.staged[key]; ok {
		delete(m.staged, key)
		for idx, staged := range m.stageOrder {
			if staged == key {
				m.stageOrder = append(m.stageOrder[:idx], m.stageOrder[idx+1:]...)
				break
			}
		}
		return false
	}
	m.staged[key] = item
	m.stageOrder = append(m.stageOrder, key)
	return true
}

func (m uninstallModel) View() string {
	width, height := effectiveSize(m.width, m.height)
	compact := width < 118 || height < 28
	leftWidth := 54
	if width < 124 {
		leftWidth = 46
	}
	rightWidth := width - leftWidth - 10
	if rightWidth < 30 {
		rightWidth = 30
	}
	panelLines := bodyLineBudget(height, 15, 7)
	selected, ok := m.selected()
	lines := []string{}
	if m.message != "" {
		lines = append(lines, mutedStyle.Render(m.message), "")
	}
	lines = append(lines, railStyle.Render("FILTER"), m.search.View(), "")
	focusLine := len(lines)
	if len(m.filtered) == 0 {
		lines = append(lines, mutedStyle.Render("No apps match the current filter."))
	} else {
		for pos, idx := range m.filtered {
			item := m.items[idx]
			badge := reviewTokenStyle.Render("NATIVE")
			if !item.HasNative {
				badge = highTokenStyle.Render("REMNANTS")
			}
			queue := ""
			if m.isStaged(item) {
				queue = " " + safeTokenStyle.Render("QUEUED")
			}
			extras := []string{}
			if item.Sensitive {
				extras = append(extras, highTokenStyle.Render("SENSITIVE"))
			} else if item.RequiresAdmin {
				extras = append(extras, reviewTokenStyle.Render("ADMIN"))
			}
			if item.SizeLabel != "" && leftWidth >= 52 {
				extras = append(extras, mutedStyle.Render(item.SizeLabel))
			}
			line := fmt.Sprintf("%s%s  %s%s", selectionPrefix(pos == m.cursor), truncateText(item.Name, 20), badge, queue)
			if len(extras) > 0 {
				line += " " + strings.Join(extras, " ")
			}
			line = singleLine(line, leftWidth-4)
			if pos == m.cursor {
				line = selectedLine.Render(line)
				focusLine = len(lines)
			}
			lines = append(lines, line)
		}
	}
	lines = viewportLines(lines, focusLine, panelLines)
	leftBody := strings.Join(lines, "\n")
	rightLines := []string{}
	if !ok {
		rightLines = append(rightLines, mutedStyle.Render("Pick an app to review."))
	} else {
		rightLines = append(rightLines, mutedStyle.Render(uninstallSelectionLine(m, selected, ok)))
		rightLines = append(rightLines, mutedStyle.Render(uninstallNextLine(m, selected, ok)))
		rightLines = append(rightLines, "")
		rightLines = append(rightLines,
			renderToneBadge(toneForUninstall(selected.HasNative))+" "+headerStyle.Render(selected.Name),
		)
		if rightWidth-4 < 40 || panelLines < 9 {
			planLines := uninstallPreviewLines(m, selected, 34)
			rightLines = append(rightLines,
				mutedStyle.Render("Plan    ")+headerStyle.Render("review uninstall"),
				mutedStyle.Render(uninstallModeText(selected.HasNative)),
			)
			rightLines = append(rightLines, planLines...)
			rightLines = append(rightLines,
				mutedStyle.Render("Origin  ")+selected.Origin,
				mutedStyle.Render("Size    ")+coalesceText(selected.SizeLabel, "unknown"),
			)
		} else {
			overview := []string{
				uninstallModeText(selected.HasNative),
				uninstallOriginText(selected),
			}
			if risk := uninstallRiskText(selected); risk != "" {
				overview = append(overview, risk)
			}
			footprint := []string{
				coalesceText(selected.SizeLabel, "unknown"),
				coalesceText(selected.LastModified, "Unknown"),
			}
			if scope := uninstallScopeText(selected); scope != "" {
				footprint = append(footprint, scope)
			}
			rightLines = append(rightLines,
				mutedStyle.Render("Plan      ")+headerStyle.Render("review uninstall"),
				mutedStyle.Render("State     ")+wrapText(strings.Join(overview, "  •  "), width),
			)
			rightLines = append(rightLines, uninstallPreviewLines(m, selected, rightWidth-8)...)
			rightLines = append(rightLines,
				mutedStyle.Render("Footprint ")+wrapText(strings.Join(footprint, "  •  "), width),
				mutedStyle.Render("Location  ")+truncateText(selected.Location, max(rightWidth-18, 18)),
				mutedStyle.Render("Families  ")+uninstallFamilyText(selected),
				mutedStyle.Render("Note      ")+uninstallNote(selected),
			)
		}
		if count := m.stageCount(); count > 0 {
			rightLines = append(rightLines,
				"",
				fmt.Sprintf("%s%d %s queued", mutedStyle.Render("Batch     "), count, pl(count, "app", "apps")),
				mutedStyle.Render("Queue     ")+m.batchSummary(3),
				mutedStyle.Render("Review    ")+"x batch uninstall",
			)
		}
	}
	rightLines = viewportLines(rightLines, 0, panelLines)
	var body string
	if compact {
		body = strings.Join([]string{
			renderPanel("INSTALLED APPS", fmt.Sprintf("%d loaded • %d matching", len(m.items), len(m.filtered)), leftBody, width-4, true),
			renderPanel("UNINSTALL PLAN", uninstallSubtitle(selected, ok), strings.Join(rightLines, "\n"), width-4, false),
		}, "\n")
	} else {
		body = joinPanels(
			renderPanel("INSTALLED APPS", fmt.Sprintf("%d loaded • %d matching", len(m.items), len(m.filtered)), leftBody, leftWidth, true),
			renderPanel("UNINSTALL PLAN", uninstallSubtitle(selected, ok), strings.Join(rightLines, "\n"), rightWidth, false),
			width-4,
		)
	}
	return renderChrome("SIFT / Uninstall", "search and review installed apps", uninstallStats(m, width), body, nil, width, false, m.height)
}

func toneForUninstall(hasNative bool) string {
	if hasNative {
		return "review"
	}
	return "high"
}

func uninstallModeText(hasNative bool) string {
	if hasNative {
		return "native uninstall + remnants"
	}
	return "remnants only"
}

func uninstallNote(item uninstallItem) string {
	if item.Sensitive {
		return "Sensitive data detected. Review carefully."
	}
	hasNative := item.HasNative
	if hasNative {
		return "Native uninstall stays explicit."
	}
	return "No native uninstall found."
}

func uninstallOriginText(item uninstallItem) string {
	if strings.TrimSpace(item.Origin) == "" {
		return "discovered install"
	}
	return item.Origin
}

func uninstallScopeText(item uninstallItem) string {
	if item.RequiresAdmin {
		return "admin boundary"
	}
	return "user space"
}

func uninstallRiskText(item uninstallItem) string {
	if item.Sensitive {
		return "sensitive review"
	}
	if item.RequiresAdmin {
		return "admin review"
	}
	if item.HasNative {
		return "review"
	}
	return "remnant only"
}

func uninstallSubtitle(item uninstallItem, ok bool) string {
	if !ok {
		return "select an app"
	}
	subtitle := uninstallModeText(item.HasNative)
	if item.Origin != "" {
		subtitle += " • " + item.Origin
	}
	if item.Sensitive {
		subtitle += " • sensitive"
	}
	return subtitle
}

func uninstallSelectionLine(m uninstallModel, item uninstallItem, ok bool) string {
	if !ok || len(m.filtered) == 0 {
		return "State   none  •  choose an app"
	}
	index := m.cursor + 1
	parts := []string{fmt.Sprintf("State   %d/%d", index, len(m.filtered)), item.Name}
	if item.HasNative {
		parts = append(parts, "native")
	} else {
		parts = append(parts, "remnants")
	}
	if m.isStaged(item) {
		parts = append(parts, "queued")
	}
	return strings.Join(parts, "  •  ")
}

func uninstallNextLine(m uninstallModel, item uninstallItem, ok bool) string {
	parts := []string{"Next"}
	switch {
	case m.loading:
		parts = append(parts, "wait for refresh")
	case m.searchActive:
		parts = append(parts, "type to filter", "enter apply", "esc clear")
	case !ok:
		parts = append(parts, "pick an app", "/ filter")
	default:
		if _, loaded := m.previewPlanForSelected(); loaded {
			parts = append(parts, "enter review")
		} else {
			parts = append(parts, "enter open")
		}
		if m.isStaged(item) {
			parts = append(parts, "u remove")
		} else {
			parts = append(parts, "space queue")
		}
		if m.stageCount() > 0 {
			parts = append(parts, "x batch")
		}
		if !item.HasNative {
			parts = append(parts, "check files")
		}
	}
	return strings.Join(parts, "  •  ")
}

func uninstallPreviewLines(m uninstallModel, item uninstallItem, width int) []string {
	if strings.TrimSpace(m.preview.key) != uninstallStageKey(item.Name) {
		return nil
	}
	switch {
	case m.preview.loading:
		return []string{mutedStyle.Render("Preview  loading review")}
	case m.preview.err != "":
		return []string{highStyle.Render("Preview  unavailable")}
	case m.preview.loaded:
		mods := planModuleCount(m.preview.plan)
		lines := []string{
			mutedStyle.Render("Preview   ") + fmt.Sprintf("%d ready  •  %d %s  •  %s", actionableCount(m.preview.plan), mods, pl(mods, "module", "modules"), domain.HumanBytes(planDisplayBytes(m.preview.plan))),
		}
		if len(m.preview.plan.Warnings) > 0 {
			lines = append(lines, mutedStyle.Render("Note      ")+truncateText(m.preview.plan.Warnings[0], width))
		}
		return lines
	default:
		return nil
	}
}

func uninstallStats(m uninstallModel, width int) []string {
	cardWidth := 24
	if width < 110 {
		cardWidth = width - 8
	}
	mode := "idle"
	tone := "review"
	if m.loading {
		mode = "loading"
		tone = "review"
	} else if m.searchActive {
		mode = "filtering"
		tone = "safe"
	}
	native := 0
	admin := 0
	sensitive := 0
	for _, item := range m.items {
		if item.HasNative {
			native++
		}
		if item.RequiresAdmin {
			admin++
		}
		if item.Sensitive {
			sensitive++
		}
	}
	remnants := len(m.items) - native
	return []string{
		renderStatCard("installed", fmt.Sprintf("%d %s", len(m.items), pl(len(m.items), "app", "apps")), "safe", cardWidth),
		renderStatCard("native", fmt.Sprintf("%d native", native), "review", cardWidth),
		renderStatCard("remnants", fmt.Sprintf("%d review", remnants), "high", cardWidth),
		renderStatCard("sensitive", fmt.Sprintf("%d guarded", sensitive), "high", cardWidth),
		renderStatCard("queued", fmt.Sprintf("%d staged", m.stageCount()), "safe", cardWidth),
		renderStatCard("matches", fmt.Sprintf("%d %s", len(m.filtered), pl(len(m.filtered), "result", "results")), "review", cardWidth),
		renderStatCard("admin", fmt.Sprintf("%d gated", admin), "high", cardWidth),
		renderStatCard("cache", mode, tone, cardWidth),
	}
}

func uninstallOriginToken(item uninstallItem) string {
	switch strings.ToLower(strings.TrimSpace(item.Origin)) {
	case "homebrew cask":
		return safeTokenStyle.Render("BREW")
	case "setapp":
		return reviewTokenStyle.Render("SETAPP")
	case "registry uninstall":
		return reviewTokenStyle.Render("REG")
	case "user program", "user application":
		return safeTokenStyle.Render("USER")
	case "system application":
		return highTokenStyle.Render("SYSTEM")
	default:
		if item.RequiresAdmin {
			return highTokenStyle.Render("ADMIN")
		}
		return mutedStyle.Render("APP")
	}
}

func uninstallRiskToken(item uninstallItem) string {
	if item.Sensitive {
		return highTokenStyle.Render("SENSITIVE")
	}
	if item.RequiresAdmin {
		return reviewTokenStyle.Render("ADMIN")
	}
	return mutedStyle.Render("SAFE")
}

func coalesceText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func uninstallInventoryMessage(at time.Time, cached bool) string {
	if at.IsZero() {
		if cached {
			return "Cached app inventory loaded. Refreshing live install data..."
		}
		return "Installed app inventory refreshed."
	}
	label := formatAppModified(at)
	if label == "" {
		label = "just now"
	}
	if cached {
		return "Cached app inventory loaded (" + label + "). Refreshing live install data..."
	}
	return "Installed app inventory refreshed (" + label + ")."
}

func uninstallMatchesQuery(item uninstallItem, query string) bool {
	fields := []string{
		item.Name,
		item.Origin,
		item.Location,
		item.SizeLabel,
		strings.Join(item.FamilyMatches, " "),
		uninstallScopeText(item),
		uninstallModeText(item.HasNative),
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func compareUninstallItems(left, right uninstallItem) bool {
	leftScore := uninstallPriority(left)
	rightScore := uninstallPriority(right)
	if leftScore != rightScore {
		return leftScore > rightScore
	}
	if !left.LastSeenAt.Equal(right.LastSeenAt) {
		if left.LastSeenAt.IsZero() {
			return false
		}
		if right.LastSeenAt.IsZero() {
			return true
		}
		return left.LastSeenAt.After(right.LastSeenAt)
	}
	return strings.ToLower(left.Name) < strings.ToLower(right.Name)
}

func uninstallPriority(item uninstallItem) int {
	score := 0
	if item.HasNative {
		score += 4
	}
	if item.Sensitive {
		score += 2
	}
	if !item.RequiresAdmin {
		score += 2
	}
	switch strings.ToLower(strings.TrimSpace(item.Origin)) {
	case "homebrew cask", "setapp", "user program", "user application":
		score += 2
	case "registry uninstall":
		score += 1
	}
	return score
}

func uninstallSizeLabel(bytes int64) string {
	if bytes <= 0 {
		return ""
	}
	return domain.HumanBytes(bytes)
}

func uninstallFamilyText(item uninstallItem) string {
	if len(item.FamilyMatches) == 0 {
		return "none"
	}
	labels := make([]string, 0, len(item.FamilyMatches))
	for _, family := range item.FamilyMatches {
		labels = append(labels, strings.ReplaceAll(family, "_", " "))
	}
	return strings.Join(labels, ", ")
}

func uninstallStageKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (m uninstallModel) batchSummary(limit int) string {
	names := m.stageNames()
	if len(names) == 0 {
		return "none"
	}
	if len(names) <= limit {
		return strings.Join(names, ", ")
	}
	return strings.Join(names[:limit], ", ") + fmt.Sprintf(" +%d", len(names)-limit)
}
