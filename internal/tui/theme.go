package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/batu3384/sift/internal/tui/design"
)

const (
	defaultViewWidth  = 118
	defaultViewHeight = 34
)

// Theme styles - now using design package colors
var (
	// Text styles using design tokens
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(design.ColorTextPrimary)
	headerStyle     = lipgloss.NewStyle().Bold(true).Foreground(design.ColorAccentSecondary)
	panelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(design.ColorTextSecondary)
	panelMetaStyle  = lipgloss.NewStyle().Foreground(design.ColorTextMuted)

	// Semantic tone styles using design tokens
	safeStyle   = lipgloss.NewStyle().Foreground(design.ColorSuccess)
	reviewStyle = lipgloss.NewStyle().Foreground(design.ColorWarning)
	highStyle   = lipgloss.NewStyle().Foreground(design.ColorDanger)
	mutedStyle  = lipgloss.NewStyle().Foreground(design.ColorTextMuted)
	footerStyle = lipgloss.NewStyle().Foreground(design.ColorTextSecondary)

	// Bar styles using design tokens
	footerBarStyle = lipgloss.NewStyle().
			Foreground(design.ColorTextSecondary).
			Background(design.ColorBackground).
			BorderTop(true).
			BorderForeground(design.ColorBorderMuted).
			Padding(0, 1)
	infoBarStyle = lipgloss.NewStyle().
			Foreground(design.ColorTextPrimary).
			Background(design.ColorSurface).
			Padding(0, 1)
	errorBarStyle = lipgloss.NewStyle().
			Foreground(design.ColorDanger).
			Background(lipgloss.Color("#241614")).
			Padding(0, 1)

	// Selection style
	selectedLine = lipgloss.NewStyle().
			Foreground(design.ColorBackground).
			Background(design.ColorSelectionBg).
			Bold(true)

	// Layout styles
	appStyle    = lipgloss.NewStyle().Padding(0, 1)
	topBarStyle = lipgloss.NewStyle().
			BorderBottom(true).
			BorderForeground(design.ColorBorderMuted).
			PaddingBottom(0)

	// Panel styles using design tokens
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(design.ColorBorderDefault).
			Background(design.ColorSurface).
			Padding(0, 1)
	activePanelStyle = panelStyle.
				BorderForeground(design.ColorBorderAccent).
				Background(design.ColorOverlay)
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(design.ColorBorderMuted).
			Background(design.ColorSurface).
			Padding(0, 1)
	compactCardStyle = lipgloss.NewStyle().Padding(0, 1)

	// Key binding styles using design tokens
	keyStyle = lipgloss.NewStyle().
			Foreground(design.ColorBackground).
			Background(design.ColorAccentSecondary).
			Bold(true).
			Padding(0, 1)
	keyTextStyle = lipgloss.NewStyle().Foreground(design.ColorTextMuted)

	// Brand styles
	wordmarkStyle = lipgloss.NewStyle().Bold(true).Foreground(design.ColorTextPrimary)
	railStyle     = lipgloss.NewStyle().Foreground(design.ColorTextSecondary).Bold(true)
	brandBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(design.ColorBorderDefault).
			Padding(0, 1)

	// Badge styles using design tokens
	safeBadgeStyle = lipgloss.NewStyle().
			Foreground(design.ColorSurface).
			Background(design.ColorSuccess).
			Bold(true).
			Padding(0, 1)
	reviewBadgeStyle = lipgloss.NewStyle().
				Foreground(design.ColorSurface).
				Background(design.ColorWarning).
				Bold(true).
				Padding(0, 1)
	highBadgeStyle = lipgloss.NewStyle().
			Foreground(design.ColorSurface).
			Background(design.ColorDanger).
			Bold(true).
			Padding(0, 1)

	// Token styles (bold tone indicators)
	safeTokenStyle   = lipgloss.NewStyle().Foreground(design.ColorSuccess).Bold(true)
	reviewTokenStyle = lipgloss.NewStyle().Foreground(design.ColorWarning).Bold(true)
	highTokenStyle   = lipgloss.NewStyle().Foreground(design.ColorDanger).Bold(true)

	// Label and accent styles
	cardLabelStyle   = lipgloss.NewStyle().Foreground(design.ColorTextMuted)
	accentFrameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#54717B"))
)

func newProgram(model tea.Model) *tea.Program {
	return tea.NewProgram(model, tea.WithAltScreen())
}

func effectiveSize(width, height int) (int, int) {
	if width <= 0 {
		width = defaultViewWidth
	}
	if height <= 0 {
		height = defaultViewHeight
	}
	if width < 84 {
		width = 84
	}
	if height < 20 {
		height = 20
	}
	return width, height
}

func renderChrome(title, subtitle string, stats []string, body string, keys []string, width int, hero bool, height int) string {
	width, height = effectiveSize(width, height)
	stats = trimStatsForHeight(stats, height, hero)
	header := renderRouteHeader(title, subtitle, width, height)
	if hero {
		header = renderHomeHeader(width, height, subtitle)
	}
	sections := []string{header}
	reservedLines := renderedLineCount(header)
	if len(stats) > 0 {
		statsBlock := joinStats(stats, width-4)
		if statsBlock != "" {
			sections = append(sections, statsBlock)
			reservedLines += renderedLineCount(statsBlock)
		}
	}
	keyBlock := ""
	if len(keys) > 0 {
		keyBlock = renderKeyBar(keys...)
		reservedLines += renderedLineCount(keyBlock)
	}
	bodyBudget := height - reservedLines
	if bodyBudget < 4 {
		bodyBudget = 4
	}
	sections = append(sections, clipRendered(body, bodyBudget))
	if keyBlock != "" {
		sections = append(sections, keyBlock)
	}
	return appStyle.Width(width).Render(strings.Join(sections, "\n"))
}

func renderHomeHeader(width, height int, subtitle string) string {
	if compactWidth(width) || compactHeight(height) {
		brand := titleStyle.Render("SIFT") + mutedStyle.Render("  /  ") + railStyle.Render("HOME")
		lines := []string{lipgloss.JoinHorizontal(lipgloss.Top, renderCompactMonogram(), "  ", brand)}
		if subtitle != "" {
			lines = append(lines, mutedStyle.Render(compactHomeSubtitle(subtitle, width)))
		}
		return topBarStyle.Width(width - 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	}
	mono := renderMonogram(width >= 96 && height >= 18)
	wordmark := renderWordmark(width, height)
	brand := lipgloss.JoinHorizontal(lipgloss.Top, brandBoxStyle.Render(mono), "  ", wordmarkStyle.Render(wordmark))
	caption := railStyle.Render("HOME")
	if subtitle != "" {
		caption = mutedStyle.Render(subtitle)
	}
	return topBarStyle.Width(width - 2).Render(lipgloss.JoinVertical(lipgloss.Left, brand, caption))
}

func renderRouteHeader(title, subtitle string, width int, height int) string {
	route := strings.ToUpper(strings.TrimPrefix(title, "SIFT / "))
	if compactWidth(width) || compactHeight(height) {
		line := titleStyle.Render("SIFT") + mutedStyle.Render(" / ") + railStyle.Render(route)
		if subtitle != "" {
			line += mutedStyle.Render("  •  " + truncateText(subtitle, max(width-28, 12)))
		}
		return topBarStyle.Width(width - 2).Render(lipgloss.JoinHorizontal(lipgloss.Top, renderCompactMonogram(), "  ", line))
	}
	mono := brandBoxStyle.Render(renderMonogram(false))
	label := titleStyle.Render("SIFT") + mutedStyle.Render(" / ") + railStyle.Render(route)
	if subtitle != "" {
		label += "\n" + mutedStyle.Render(subtitle)
	}
	return topBarStyle.Width(width - 2).Render(lipgloss.JoinHorizontal(lipgloss.Top, mono, "  ", label))
}

func renderWordmark(width, height int) string {
	if width < 72 || height < 18 {
		return "SIFT"
	}
	if width < 124 || height < 28 {
		return strings.Join([]string{
			"######  ## ####### #######",
			"##      ## ##         ##   ",
			"######  ## #####     ##   ",
			"     ## ## ##         ##   ",
			"######  ## ##         ##   ",
		}, "\n")
	}
	return strings.Join([]string{
		"####### ## ####### #######",
		"##      ## ##         ##   ",
		"####### ## #####     ##   ",
		"     ## ## ##         ##   ",
		"####### ## ##         ##   ",
	}, "\n")
}

func renderMonogram(large bool) string {
	if large {
		return strings.Join([]string{
			"+----+",
			"|  S |",
			"+----+",
		}, "\n")
	}
	return "S"
}

func renderCompactMonogram() string {
	return railStyle.Render("S")
}

func renderPanel(title, subtitle, body string, width int, active bool) string {
	style := panelStyle
	if active {
		style = activePanelStyle
	}
	borderColor, bodyBackground, titleStyle, markerStyle := panelTheme(title, active)
	style = style.BorderForeground(borderColor).Background(bodyBackground)
	marker := markerStyle.Render("·")
	if active {
		marker = markerStyle.Bold(true).Render("▶")
	}
	header := marker + " " + titleStyle.Render(title)
	if subtitle != "" {
		header += "  " + panelMetaStyle.Render(singleLine(subtitle, max(width-18, 10)))
	}
	rule := renderPanelRule(borderColor, width)
	return style.Width(width).Render(header + "\n" + rule + "\n" + body)
}

func renderPanelRule(borderColor lipgloss.Color, width int) string {
	ruleWidth := max(width-10, 8)
	core := strings.Repeat("─", ruleWidth)
	return accentFrameStyle.Foreground(borderColor).Render("╺" + core + "╸")
}

func renderStatCard(label, value, tone string, width int) string {
	return renderRouteStatCard("", label, value, tone, width)
}

func renderRouteStatCard(route, label, value, tone string, width int) string {
	style := cardStyle.Width(width)
	valueStyle := headerStyle
	labelStyle := cardLabelStyle
	borderColor, labelColor, valueColor, backgroundColor := routeCardTonePalette(route, label, tone)
	style = style.BorderForeground(borderColor).Background(backgroundColor)
	labelStyle = labelStyle.Foreground(labelColor)
	valueStyle = valueStyle.Foreground(valueColor).Bold(true)
	if compactWidth(width) {
		compact := compactCardStyle.Width(width)
		compact = compact.Foreground(valueColor).Background(backgroundColor)
		return compact.Render(labelStyle.Render(strings.ToUpper(label)) + "  " + valueStyle.Render(value))
	}
	labelLine := renderRouteStatLabelLine(route, borderColor, labelStyle, label)
	valueLine := renderRouteStatValueLine(borderColor, valueStyle, value)
	return style.Render(labelLine + "\n" + valueLine)
}

func renderRouteStatLabelLine(route string, borderColor lipgloss.Color, labelStyle lipgloss.Style, label string) string {
	accent := accentFrameStyle.Foreground(borderColor).Render("╺━")
	token := routeCardLabelToken(route)
	if token != "" {
		tokenStyle := labelStyle.Bold(true)
		return accent + " " + tokenStyle.Render(token) + " " + labelStyle.Render(strings.ToUpper(label))
	}
	return accent + " " + labelStyle.Render(strings.ToUpper(label))
}

func renderRouteStatValueLine(borderColor lipgloss.Color, valueStyle lipgloss.Style, value string) string {
	accent := accentFrameStyle.Foreground(borderColor).Render("╰─")
	return accent + " " + valueStyle.Render(value)
}

func routeCardLabelToken(route string) string {
	switch normalizeCardRoute(route) {
	case "home":
		return "SCOUT"
	case "status":
		return "OBS"
	case "clean":
		return "FORGE"
	case "uninstall":
		return "COURIER"
	case "analyze":
		return "ORACLE"
	case "progress":
		return "ACTION"
	case "result":
		return "SETTLED"
	case "review":
		return "GATE"
	case "preflight":
		return "ACCESS"
	case "tools":
		return "UTILITY"
	case "doctor":
		return "DIAG"
	case "protect":
		return "GUARD"
	default:
		return ""
	}
}

func renderKeyBar(items ...string) string {
	parts := make([]string, 0, len(items)*2-1)
	sep := footerStyle.Render("  ·  ")
	for i, item := range items {
		if i > 0 {
			parts = append(parts, sep)
		}
		chunks := strings.SplitN(item, " ", 2)
		if len(chunks) == 1 {
			parts = append(parts, keyStyle.Render(chunks[0]))
			continue
		}
		parts = append(parts, keyStyle.Render(chunks[0])+" "+keyTextStyle.Render(chunks[1]))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

func joinStats(cards []string, width int) string {
	if len(cards) == 0 {
		return ""
	}
	if compactWidth(width) {
		return lipgloss.JoinVertical(lipgloss.Left, cards...)
	}
	if len(cards) == 1 {
		return lipgloss.JoinVertical(lipgloss.Left, cards...)
	}
	if width < 108 || len(cards) == 2 {
		return lipgloss.JoinHorizontal(lipgloss.Top, appendInterleaved(cards, "  ")...)
	}
	rows := make([]string, 0, (len(cards)+1)/2)
	for i := 0; i < len(cards); i += 2 {
		row := cards[i:min(i+2, len(cards))]
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, appendInterleaved(row, "  ")...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func joinPanels(left, right string, width int) string {
	if width < 112 {
		return lipgloss.JoinVertical(lipgloss.Left, left, "", right)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
}

func renderSectionRule(width int) string {
	ruleWidth := max(min(width-2, 48), 8)
	return accentFrameStyle.Foreground(lipgloss.Color("#253F3A")).Render(strings.Repeat("─", ruleWidth))
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	return lipgloss.NewStyle().Width(width).Render(text)
}

func renderToneBadge(tone string) string {
	switch tone {
	case "safe":
		return safeBadgeStyle.Render("SAFE")
	case "high":
		return highBadgeStyle.Render("HIGH")
	default:
		return reviewBadgeStyle.Render("REVIEW")
	}
}

func renderToneToken(tone string) string {
	switch tone {
	case "safe":
		return safeTokenStyle.Render("SAFE")
	case "high":
		return highTokenStyle.Render("HIGH")
	default:
		return reviewTokenStyle.Render("REVIEW")
	}
}

func truncateText(text string, max int) string {
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

func rowLabelWidth(width int, floor int) int {
	if width <= 0 {
		return floor
	}
	return max(floor, min(width-32, 128))
}

func appendInterleaved(values []string, spacer string) []string {
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

func splitColumns(totalWidth int, leftRatio float64, minLeft int, minRight int) (int, int) {
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

func compactWidth(width int) bool {
	return width < 100
}

func compactHeight(height int) bool {
	return height < 26
}

func renderFooterBar(width int, body string) string {
	width, _ = effectiveSize(width, defaultViewHeight)
	return footerBarStyle.Width(width - 2).Render(singleLine(body, width-6))
}

func renderInfoBar(width int, body string) string {
	width, _ = effectiveSize(width, defaultViewHeight)
	return infoBarStyle.Width(width - 2).Render(singleLine(body, width-6))
}

func renderErrorBar(width int, body string) string {
	width, _ = effectiveSize(width, defaultViewHeight)
	return errorBarStyle.Width(width - 2).Render(singleLine(body, width-6))
}

func trimStatsForHeight(cards []string, height int, hero bool) []string {
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

func clipRendered(content string, maxLines int) string {
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
			last = mutedStyle.Render("…")
		} else {
			last = truncateText(last, max(defaultViewWidth/2, 16))
			last += " " + mutedStyle.Render("…")
		}
		clipped[maxLines-1] = last
	}
	return strings.Join(clipped, "\n")
}

func viewportLines(lines []string, cursor int, maxLines int) []string {
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
		window[0] = mutedStyle.Render("…") + " " + strings.TrimLeft(window[0], " ")
	}
	if end < len(lines) {
		window[len(window)-1] = strings.TrimRight(window[len(window)-1], " ") + " " + mutedStyle.Render("…")
	}
	return window
}

func bodyLineBudget(height int, reserve int, minLines int) int {
	_, effectiveHeight := effectiveSize(defaultViewWidth, height)
	lines := effectiveHeight - reserve
	if lines < minLines {
		return minLines
	}
	return lines
}

func singleLine(text string, max int) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")
	return truncateText(text, max)
}

func renderedLineCount(content string) int {
	if content == "" {
		return 0
	}
	return len(strings.Split(content, "\n"))
}

func compactHomeSubtitle(subtitle string, width int) string {
	parts := strings.Split(subtitle, "  •  ")
	if len(parts) == 0 {
		return singleLine(subtitle, max(width-6, 18))
	}
	compact := []string{parts[0]}
	if len(parts) > 1 {
		compact = append(compact, parts[1])
	}
	if len(parts) > 2 {
		compact = append(compact, parts[2])
	}
	return singleLine(strings.Join(compact, "  •  "), max(width-6, 18))
}

func panelTheme(panelName string, active bool) (borderColor lipgloss.Color, backgroundColor lipgloss.Color, titleStyle lipgloss.Style, marker lipgloss.Style) {
	upper := strings.ToUpper(strings.TrimSpace(panelName))
	borderColor = lipgloss.Color("#39505A")
	backgroundColor = lipgloss.Color("#11181C")
	titleStyle = panelTitleStyle
	marker = mutedStyle.Foreground(lipgloss.Color("#91A4AD"))
	if active {
		borderColor = lipgloss.Color("#6C909D")
		backgroundColor = lipgloss.Color("#152027")
		marker = railStyle.Foreground(design.ColorWarning)
	}
	switch upper {
	case "SPOTLIGHT", "OVERVIEW":
		borderColor = lipgloss.Color("#44616C")
		backgroundColor = lipgloss.Color("#111A1E")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#D9E7EB"))
		marker = railStyle.Foreground(design.ColorAccentPrimary)
	case "COMMAND DECK":
		borderColor = lipgloss.Color("#6A7553")
		backgroundColor = lipgloss.Color("#141812")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#E5E0C8"))
		marker = reviewStyle.Foreground(design.ColorWarning)
	case "ROUTE RAIL":
		borderColor = lipgloss.Color("#4A6A74")
		backgroundColor = lipgloss.Color("#10191D")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#DCE8EC"))
		marker = railStyle.Foreground(design.ColorAccentPrimary)
	case "ROUTE DECK":
		borderColor = lipgloss.Color("#596D61")
		backgroundColor = lipgloss.Color("#121816")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#DFE8E1"))
		marker = safeStyle.Foreground(design.ColorSuccess)
	case "OBSERVATORY":
		borderColor = lipgloss.Color("#4A6972")
		backgroundColor = lipgloss.Color("#10191D")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#DAE7EB"))
		marker = railStyle.Foreground(design.ColorAccentPrimary)
	case "LIVE RAIL":
		borderColor = lipgloss.Color("#6B745C")
		backgroundColor = lipgloss.Color("#131813")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#E0E5D2"))
		marker = reviewStyle.Foreground(design.ColorWarning)
	case "STORAGE RAIL":
		borderColor = lipgloss.Color("#566E60")
		backgroundColor = lipgloss.Color("#121814")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#DDE8DE"))
		marker = safeStyle.Foreground(design.ColorSuccess)
	case "POWER RAIL":
		borderColor = lipgloss.Color("#7A6646")
		backgroundColor = lipgloss.Color("#18140F")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#F0DFC2"))
		marker = reviewStyle.Foreground(design.ColorWarning)
	case "SESSION RAIL":
		borderColor = lipgloss.Color("#765F54")
		backgroundColor = lipgloss.Color("#171312")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#E5D4CF"))
		marker = highStyle.Foreground(design.ColorDanger)
	case "UTILITY RAIL":
		borderColor = lipgloss.Color("#566C61")
		backgroundColor = lipgloss.Color("#111814")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#D9E4DC"))
		marker = safeStyle.Foreground(design.ColorSuccess)
	case "PROGRESS RAIL":
		borderColor = lipgloss.Color("#7B6742")
		backgroundColor = lipgloss.Color("#17130E")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#F0DEC0"))
		marker = reviewStyle.Foreground(design.ColorWarning)
	case "REVIEW RAIL":
		borderColor = lipgloss.Color("#7A6541")
		backgroundColor = lipgloss.Color("#17130E")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#EFDDBD"))
		marker = reviewStyle.Foreground(design.ColorWarning)
	case "SETTLED RAIL":
		borderColor = lipgloss.Color("#58705F")
		backgroundColor = lipgloss.Color("#111712")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#DDE7DF"))
		marker = safeStyle.Foreground(design.ColorSuccess)
	case "TOOL DECK", "DIAGNOSIS DECK", "POLICY DECK":
		borderColor = lipgloss.Color("#45606A")
		backgroundColor = lipgloss.Color("#121B20")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#DDE9ED"))
		marker = railStyle.Foreground(design.ColorAccentPrimary)
	case "FOCUS DECK", "ACTION DECK", "OUTCOME DECK":
		borderColor = lipgloss.Color("#45606A")
		backgroundColor = lipgloss.Color("#121B20")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#DDE9ED"))
		marker = railStyle.Foreground(design.ColorAccentPrimary)
	case "RUN GATE":
		borderColor = lipgloss.Color("#745B3C")
		backgroundColor = lipgloss.Color("#18120D")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#F1DFC5"))
		marker = reviewStyle.Foreground(design.ColorWarning)
	case "SETTLED GATE":
		borderColor = lipgloss.Color("#58705F")
		backgroundColor = lipgloss.Color("#111712")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#DDE7DF"))
		marker = safeStyle.Foreground(design.ColorSuccess)
	case "CHECK RAIL":
		borderColor = lipgloss.Color("#7B6742")
		backgroundColor = lipgloss.Color("#17130E")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#F0DEC0"))
		marker = reviewStyle.Foreground(design.ColorWarning)
	case "GUARD RAIL":
		borderColor = lipgloss.Color("#7A6541")
		backgroundColor = lipgloss.Color("#17130E")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#EFDDBD"))
		marker = reviewStyle.Foreground(design.ColorWarning)
	case "CONTROL DECK":
		borderColor = lipgloss.Color("#4F6470")
		backgroundColor = lipgloss.Color("#12191E")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#E0EAEC"))
		marker = railStyle.Foreground(design.ColorAccentPrimary)
	case "SWEEP LANES":
		borderColor = lipgloss.Color("#8A6C44")
		backgroundColor = lipgloss.Color("#18130D")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#F1D3AE"))
		marker = reviewStyle.Foreground(design.ColorWarning)
	case "SWEEP DECK":
		borderColor = lipgloss.Color("#466069")
		backgroundColor = lipgloss.Color("#121B20")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#DDE9ED"))
		marker = railStyle.Foreground(design.ColorAccentPrimary)
	case "DETAIL", "INSPECT", "FOLLOW-UP":
		borderColor = lipgloss.Color("#45606A")
		backgroundColor = lipgloss.Color("#121B20")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#DDE9ED"))
	case "QUEUE", "OPERATIONS":
		borderColor = lipgloss.Color("#7B6742")
		backgroundColor = lipgloss.Color("#17130E")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#F0DEC0"))
		marker = reviewStyle.Foreground(design.ColorWarning)
	case "RESCAN", "COMPLETE":
		borderColor = lipgloss.Color("#6C6454")
		backgroundColor = lipgloss.Color("#151411")
		titleStyle = panelTitleStyle.Foreground(lipgloss.Color("#E6DAC5"))
	}
	if active {
		backgroundColor = brightenTone(backgroundColor)
	}
	return borderColor, backgroundColor, titleStyle, marker
}

func cardTonePalette(label string, tone string) (borderColor lipgloss.Color, labelColor lipgloss.Color, valueColor lipgloss.Color, backgroundColor lipgloss.Color) {
	borderColor = lipgloss.Color("#485A63")
	labelColor = lipgloss.Color("#8EA2AC")
	valueColor = lipgloss.Color("#A7C5CF")
	backgroundColor = lipgloss.Color("#141C22")
	switch tone {
	case "safe":
		borderColor = lipgloss.Color("#50685A")
		labelColor = lipgloss.Color("#A7C0B1")
		valueColor = design.ColorSuccess
		backgroundColor = lipgloss.Color("#121A16")
	case "review":
		borderColor = lipgloss.Color("#7A6847")
		labelColor = lipgloss.Color("#C9B48A")
		valueColor = design.ColorWarning
		backgroundColor = lipgloss.Color("#1C1711")
	case "high":
		borderColor = lipgloss.Color("#865649")
		labelColor = lipgloss.Color("#D5A28F")
		valueColor = design.ColorDanger
		backgroundColor = lipgloss.Color("#211613")
	}
	switch strings.ToUpper(strings.TrimSpace(label)) {
	case "UPDATE", "ALERTS", "OPERATOR":
		borderColor = lipgloss.Color("#55707B")
		if tone == "review" {
			borderColor = lipgloss.Color("#8A734E")
		}
		if tone == "high" {
			borderColor = lipgloss.Color("#8D5F53")
		}
	case "HEALTH", "STATE", "SESSION":
		labelColor = lipgloss.Color("#B7C9CF")
	}
	return borderColor, labelColor, valueColor, backgroundColor
}

func routeCardTonePalette(route, label, tone string) (borderColor lipgloss.Color, labelColor lipgloss.Color, valueColor lipgloss.Color, backgroundColor lipgloss.Color) {
	borderColor, labelColor, valueColor, backgroundColor = cardTonePalette(label, tone)
	switch normalizeCardRoute(route) {
	case "clean":
		return routePaletteForTone(tone,
			lipgloss.Color("#4E6B5D"), lipgloss.Color("#B8D0C1"), lipgloss.Color("#9FD7AF"), lipgloss.Color("#111915"),
			lipgloss.Color("#8A7048"), lipgloss.Color("#E1C796"), lipgloss.Color("#E6B36F"), lipgloss.Color("#1D1710"),
			lipgloss.Color("#8A5A4C"), lipgloss.Color("#E0AE9B"), lipgloss.Color("#E08A74"), lipgloss.Color("#251713"),
		)
	case "uninstall":
		return routePaletteForTone(tone,
			lipgloss.Color("#54695E"), lipgloss.Color("#C0D1C9"), lipgloss.Color("#9CC6AD"), lipgloss.Color("#121817"),
			lipgloss.Color("#876445"), lipgloss.Color("#E2C099"), lipgloss.Color("#E2A66B"), lipgloss.Color("#1E1510"),
			lipgloss.Color("#93584D"), lipgloss.Color("#E2AB98"), lipgloss.Color("#E07667"), lipgloss.Color("#251612"),
		)
	case "analyze":
		return routePaletteForTone(tone,
			lipgloss.Color("#4B6E74"), lipgloss.Color("#B9D5DA"), lipgloss.Color("#90D0DB"), lipgloss.Color("#10191C"),
			lipgloss.Color("#6F7864"), lipgloss.Color("#D7D1B8"), lipgloss.Color("#D7B57C"), lipgloss.Color("#141612"),
			lipgloss.Color("#7C5B56"), lipgloss.Color("#D8B1A5"), lipgloss.Color("#DB8B7D"), lipgloss.Color("#211513"),
		)
	case "home", "status":
		return routePaletteForTone(tone,
			lipgloss.Color("#4C7075"), lipgloss.Color("#B7D5DA"), lipgloss.Color("#A7C5CF"), lipgloss.Color("#10191C"),
			lipgloss.Color("#5E6F78"), lipgloss.Color("#CBD6DD"), lipgloss.Color("#D4B27C"), lipgloss.Color("#12181D"),
			lipgloss.Color("#7A5D55"), lipgloss.Color("#D7B6AC"), lipgloss.Color("#D79283"), lipgloss.Color("#1B1513"),
		)
	case "progress":
		return routePaletteForTone(tone,
			lipgloss.Color("#546C5E"), lipgloss.Color("#C0D1C2"), lipgloss.Color("#9FD1AD"), lipgloss.Color("#121816"),
			lipgloss.Color("#816C48"), lipgloss.Color("#DBC69B"), lipgloss.Color("#E1AF72"), lipgloss.Color("#19150F"),
			lipgloss.Color("#895E4E"), lipgloss.Color("#DAB09E"), lipgloss.Color("#DE846D"), lipgloss.Color("#221713"),
		)
	case "result":
		return routePaletteForTone(tone,
			lipgloss.Color("#5C725F"), lipgloss.Color("#C7D5C7"), lipgloss.Color("#A7D1A2"), lipgloss.Color("#121712"),
			lipgloss.Color("#6E7159"), lipgloss.Color("#D5D4B7"), lipgloss.Color("#CFCB9B"), lipgloss.Color("#151610"),
			lipgloss.Color("#855E51"), lipgloss.Color("#DBB3A7"), lipgloss.Color("#DD8C79"), lipgloss.Color("#1F1513"),
		)
	case "review":
		return routePaletteForTone(tone,
			lipgloss.Color("#57706A"), lipgloss.Color("#C0D7CF"), lipgloss.Color("#97CEBF"), lipgloss.Color("#101816"),
			lipgloss.Color("#806A43"), lipgloss.Color("#E0CB9B"), lipgloss.Color("#E3B779"), lipgloss.Color("#19150E"),
			lipgloss.Color("#8C5C50"), lipgloss.Color("#DEB0A3"), lipgloss.Color("#DE8776"), lipgloss.Color("#221612"),
		)
	case "preflight":
		return routePaletteForTone(tone,
			lipgloss.Color("#567066"), lipgloss.Color("#C0D6CC"), lipgloss.Color("#95CDBA"), lipgloss.Color("#101715"),
			lipgloss.Color("#826A46"), lipgloss.Color("#E0C8A0"), lipgloss.Color("#E2B57C"), lipgloss.Color("#1A150E"),
			lipgloss.Color("#8E604E"), lipgloss.Color("#DFB39E"), lipgloss.Color("#E28A73"), lipgloss.Color("#211611"),
		)
	case "tools", "doctor", "protect":
		return routePaletteForTone(tone,
			lipgloss.Color("#557168"), lipgloss.Color("#BCD7CE"), lipgloss.Color("#97CFBF"), lipgloss.Color("#101817"),
			lipgloss.Color("#766C52"), lipgloss.Color("#D6CEB0"), lipgloss.Color("#D9BF88"), lipgloss.Color("#161610"),
			lipgloss.Color("#7A6159"), lipgloss.Color("#D8BAB0"), lipgloss.Color("#D58E84"), lipgloss.Color("#1B1514"),
		)
	default:
		return borderColor, labelColor, valueColor, backgroundColor
	}
}

func normalizeCardRoute(route string) string {
	switch strings.ToLower(strings.TrimSpace(route)) {
	case "home", "command deck":
		return "home"
	case "status", "observatory":
		return "status"
	case "clean", "sweep":
		return "clean"
	case "uninstall", "removal":
		return "uninstall"
	case "analyze", "inspect":
		return "analyze"
	case "progress":
		return "progress"
	case "result", "settled":
		return "result"
	case "review":
		return "review"
	case "preflight", "access":
		return "preflight"
	case "tools", "menu":
		return "tools"
	case "doctor", "diagnosis":
		return "doctor"
	case "protect", "guard":
		return "protect"
	default:
		return strings.ToLower(strings.TrimSpace(route))
	}
}

func routePaletteForTone(
	tone string,
	safeBorder, safeLabel, safeValue, safeBackground lipgloss.Color,
	reviewBorder, reviewLabel, reviewValue, reviewBackground lipgloss.Color,
	highBorder, highLabel, highValue, highBackground lipgloss.Color,
) (borderColor lipgloss.Color, labelColor lipgloss.Color, valueColor lipgloss.Color, backgroundColor lipgloss.Color) {
	switch tone {
	case "safe":
		return safeBorder, safeLabel, safeValue, safeBackground
	case "high":
		return highBorder, highLabel, highValue, highBackground
	default:
		return reviewBorder, reviewLabel, reviewValue, reviewBackground
	}
}

func brightenTone(color lipgloss.Color) lipgloss.Color {
	switch string(color) {
	case "#11181C":
		return lipgloss.Color("#152027")
	case "#111A1E":
		return lipgloss.Color("#162229")
	case "#111814":
		return lipgloss.Color("#151E19")
	case "#17130E":
		return lipgloss.Color("#1C1711")
	case "#111712":
		return lipgloss.Color("#161C17")
	case "#121B20":
		return lipgloss.Color("#172229")
	case "#18120D":
		return lipgloss.Color("#1D1711")
	case "#12191E":
		return lipgloss.Color("#172128")
	case "#121816":
		return lipgloss.Color("#17201B")
	case "#141712":
		return lipgloss.Color("#191C16")
	case "#10191C":
		return lipgloss.Color("#152126")
	case "#12181D":
		return lipgloss.Color("#172026")
	case "#101816":
		return lipgloss.Color("#151F1B")
	case "#1A150E":
		return lipgloss.Color("#201912")
	case "#101817":
		return lipgloss.Color("#15201C")
	case "#151411":
		return lipgloss.Color("#1A1815")
	default:
		return color
	}
}
