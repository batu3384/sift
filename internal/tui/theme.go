package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	defaultViewWidth  = 118
	defaultViewHeight = 34
)

var (
	titleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
	subtitleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9"))
	headerStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#50FA7B"))
	panelTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD"))
	panelMetaStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
	safeStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B"))
	reviewStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C"))
	highStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	mutedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
	footerStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	footerBarStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#A8C6C0")).Background(lipgloss.Color("#282A36")).BorderTop(true).BorderForeground(lipgloss.Color("#44475A")).Padding(0, 1)
	infoBarStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Background(lipgloss.Color("#44475A")).Padding(0, 1)
	errorBarStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Background(lipgloss.Color("#2D1F1F")).Padding(0, 1)
	selectedLine     = lipgloss.NewStyle().Foreground(lipgloss.Color("#282A36")).Background(lipgloss.Color("#50FA7B")).Bold(true)
	appStyle         = lipgloss.NewStyle().Padding(0, 1)
	topBarStyle      = lipgloss.NewStyle().BorderBottom(true).BorderForeground(lipgloss.Color("#44475A")).PaddingBottom(0)
	panelStyle       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#2F6860")).Background(lipgloss.Color("#1A1A2E")).Padding(0, 1)
	activePanelStyle = panelStyle.Copy().BorderForeground(lipgloss.Color("#50FA7B")).Background(lipgloss.Color("#21222C"))
	cardStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#44475A")).Background(lipgloss.Color("#21222C")).Padding(0, 1)
	compactCardStyle = lipgloss.NewStyle().Padding(0, 1)
	keyStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#282A36")).Background(lipgloss.Color("#50FA7B")).Bold(true).Padding(0, 1)
	keyTextStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
	wordmarkStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8F8F2"))
	railStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9")).Bold(true)
	brandBoxStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#6272A4")).Padding(0, 1)
	safeBadgeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#282A36")).Background(lipgloss.Color("#50FA7B")).Bold(true).Padding(0, 1)
	reviewBadgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#282A36")).Background(lipgloss.Color("#FFB86C")).Bold(true).Padding(0, 1)
	highBadgeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#282A36")).Background(lipgloss.Color("#FF5555")).Bold(true).Padding(0, 1)
	safeTokenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true)
	reviewTokenStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")).Bold(true)
	highTokenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Bold(true)
	cardLabelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
	accentFrameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9"))

	// New product-focused styles
	statsCardStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#2F6860")).Background(lipgloss.Color("#1A1A2E")).Padding(1, 2)
	statsValueStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#50FA7B"))
	statsLabelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
	statsUnitStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9"))
	welcomeStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1)
	versionBadgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#282A36")).Background(lipgloss.Color("#BD93F9")).Bold(true).Padding(0, 1)
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
			"██████  ██ ███████ ████████",
			"██      ██ ██         ██   ",
			"██████  ██ █████      ██   ",
			"     ██ ██ ██         ██   ",
			"██████  ██ ██         ██   ",
		}, "\n")
	}
	return strings.Join([]string{
		"███████ ██ ███████ ████████",
		"██      ██ ██         ██   ",
		"███████ ██ █████      ██   ",
		"     ██ ██ ██         ██   ",
		"███████ ██ ██         ██   ",
	}, "\n")
}

func renderMonogram(large bool) string {
	if large {
		return strings.Join([]string{
			"╭────╮",
			"│  S │",
			"╰────╯",
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
	style = style.Copy().BorderForeground(borderColor).Background(bodyBackground)
	marker := markerStyle.Render("·")
	if active {
		marker = markerStyle.Bold(true).Render("▶")
	}
	header := marker + " " + titleStyle.Render(title)
	if subtitle != "" {
		header += "  " + panelMetaStyle.Render(singleLine(subtitle, max(width-18, 10)))
	}
	rule := accentFrameStyle.Copy().Foreground(borderColor).Render(strings.Repeat("─", max(width-6, 8)))
	return style.Width(width).Render(header + "\n" + rule + "\n" + body)
}

func renderStatCard(label, value, tone string, width int) string {
	style := cardStyle.Copy().Width(width)
	valueStyle := headerStyle
	labelStyle := cardLabelStyle
	borderColor, labelColor, valueColor, backgroundColor := cardTonePalette(label, tone)
	style = style.BorderForeground(borderColor).Background(backgroundColor)
	labelStyle = labelStyle.Copy().Foreground(labelColor)
	valueStyle = valueStyle.Copy().Foreground(valueColor).Bold(true)
	if compactWidth(width) {
		compact := compactCardStyle.Copy().Width(width)
		compact = compact.Foreground(valueColor).Background(backgroundColor)
		return compact.Render(labelStyle.Render(strings.ToUpper(label)) + "  " + valueStyle.Render(value))
	}
	return style.Render(labelStyle.Render(strings.ToUpper(label)) + "\n" + valueStyle.Render(value))
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
	return accentFrameStyle.Copy().Foreground(lipgloss.Color("#253F3A")).Render(strings.Repeat("─", ruleWidth))
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

func renderMeterBar(label string, value float64, width int) string {
	if width < 16 {
		return fmt.Sprintf("%s %.1f%%", label, value)
	}
	barWidth := 16
	if width > 40 {
		barWidth = 20
	}
	if value < 0 {
		value = 0
	}
	if value > 100 {
		value = 100
	}
	filled := int((value / 100) * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	return fmt.Sprintf("%-5s %s %5.1f%%", label, bar, value)
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
	borderColor = lipgloss.Color("#2F6860")
	backgroundColor = lipgloss.Color("#1A1A2E")
	titleStyle = panelTitleStyle
	marker = mutedStyle.Copy().Foreground(lipgloss.Color("#5FAEA2"))
	if active {
		borderColor = lipgloss.Color("#7BE2D2")
		backgroundColor = lipgloss.Color("#21222C")
		marker = railStyle.Copy().Foreground(lipgloss.Color("#50FA7B"))
	}
	switch upper {
	case "SPOTLIGHT", "OVERVIEW":
		borderColor = lipgloss.Color("#3C8D84")
		backgroundColor = lipgloss.Color("#0C1918")
		titleStyle = panelTitleStyle.Copy().Foreground(lipgloss.Color("#D8FFF7"))
		marker = railStyle.Copy().Foreground(lipgloss.Color("#BD93F9"))
	case "DETAIL", "INSPECT", "FOLLOW-UP":
		borderColor = lipgloss.Color("#4D7C75")
		backgroundColor = lipgloss.Color("#21222C")
		titleStyle = panelTitleStyle.Copy().Foreground(lipgloss.Color("#E8F7F3"))
	case "QUEUE", "OPERATIONS":
		borderColor = lipgloss.Color("#8A7B3E")
		backgroundColor = lipgloss.Color("#17150D")
		titleStyle = panelTitleStyle.Copy().Foreground(lipgloss.Color("#F7E6B1"))
		marker = reviewStyle.Copy().Foreground(lipgloss.Color("#FFB86C"))
	case "RESCAN", "COMPLETE":
		borderColor = lipgloss.Color("#6D6351")
		backgroundColor = lipgloss.Color("#151411")
		titleStyle = panelTitleStyle.Copy().Foreground(lipgloss.Color("#E8DCC2"))
	}
	if active {
		backgroundColor = brightenTone(backgroundColor)
	}
	return borderColor, backgroundColor, titleStyle, marker
}

func cardTonePalette(label string, tone string) (borderColor lipgloss.Color, labelColor lipgloss.Color, valueColor lipgloss.Color, backgroundColor lipgloss.Color) {
	borderColor = lipgloss.Color("#44475A")
	labelColor = lipgloss.Color("#6272A4")
	valueColor = lipgloss.Color("#50FA7B")
	backgroundColor = lipgloss.Color("#21222C")
	switch tone {
	case "safe":
		borderColor = lipgloss.Color("#2E6B4C")
		labelColor = lipgloss.Color("#98C9A9")
		valueColor = lipgloss.Color("#50FA7B")
		backgroundColor = lipgloss.Color("#102019")
	case "review":
		borderColor = lipgloss.Color("#7C6831")
		labelColor = lipgloss.Color("#CBB782")
		valueColor = lipgloss.Color("#FFB86C")
		backgroundColor = lipgloss.Color("#221B11")
	case "high":
		borderColor = lipgloss.Color("#844A3D")
		labelColor = lipgloss.Color("#D9A394")
		valueColor = lipgloss.Color("#FF5555")
		backgroundColor = lipgloss.Color("#271611")
	}
	switch strings.ToUpper(strings.TrimSpace(label)) {
	case "UPDATE", "ALERTS", "OPERATOR":
		borderColor = lipgloss.Color("#4E8D84")
		if tone == "review" {
			borderColor = lipgloss.Color("#9F8444")
		}
	case "HEALTH", "STATE", "SESSION":
		labelColor = lipgloss.Color("#B4DDD5")
	}
	return borderColor, labelColor, valueColor, backgroundColor
}

func brightenTone(color lipgloss.Color) lipgloss.Color {
	switch string(color) {
	case "#0C1918":
		return lipgloss.Color("#10201F")
	case "#21222C":
		return lipgloss.Color("#0E1B1A")
	case "#17150D":
		return lipgloss.Color("#1C1910")
	case "#151411":
		return lipgloss.Color("#1A1815")
	default:
		return color
	}
}
