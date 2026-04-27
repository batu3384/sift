// Package theme provides styling constants and utilities for the SIFT TUI.
package theme

import "github.com/charmbracelet/lipgloss"

// PanelTheme holds colors for a specific panel state.
type PanelTheme struct {
	BorderColor     lipgloss.Color
	BackgroundColor lipgloss.Color
	TitleStyle      lipgloss.Style
	Marker          lipgloss.Style
}

// CardTheme holds colors for stat cards.
type CardTheme struct {
	BorderColor     lipgloss.Color
	LabelColor      lipgloss.Color
	ValueColor      lipgloss.Color
	BackgroundColor lipgloss.Color
}

// PanelThemes returns theme colors for a panel by name and active state.
func PanelThemes(panelName string, active bool) PanelTheme {
	borderColor := lipgloss.Color("#39505A")
	backgroundColor := lipgloss.Color("#11181C")
	titleStyle := PanelTitleStyle
	marker := MutedStyle.Foreground(lipgloss.Color("#91A4AD"))
	if active {
		borderColor = lipgloss.Color("#6C909D")
		backgroundColor = lipgloss.Color("#152027")
		marker = RailStyle.Foreground(lipgloss.Color("#D8A66A"))
	}
	upper := normalizePanelName(panelName)
	switch upper {
	case "SPOTLIGHT", "OVERVIEW":
		borderColor = lipgloss.Color("#44616C")
		backgroundColor = lipgloss.Color("#111A1E")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#D9E7EB"))
		marker = RailStyle.Foreground(lipgloss.Color("#A7C5CF"))
	case "COMMAND DECK":
		borderColor = lipgloss.Color("#6A7553")
		backgroundColor = lipgloss.Color("#141812")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#E5E0C8"))
		marker = ReviewStyle.Foreground(lipgloss.Color("#D8A66A"))
	case "ROUTE RAIL":
		borderColor = lipgloss.Color("#4A6A74")
		backgroundColor = lipgloss.Color("#10191D")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#DCE8EC"))
		marker = RailStyle.Foreground(lipgloss.Color("#A7C5CF"))
	case "ROUTE DECK":
		borderColor = lipgloss.Color("#596D61")
		backgroundColor = lipgloss.Color("#121816")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#DFE8E1"))
		marker = SafeStyle.Foreground(lipgloss.Color("#8FC5A0"))
	case "OBSERVATORY":
		borderColor = lipgloss.Color("#4A6972")
		backgroundColor = lipgloss.Color("#10191D")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#DAE7EB"))
		marker = RailStyle.Foreground(lipgloss.Color("#A7C5CF"))
	case "LIVE RAIL":
		borderColor = lipgloss.Color("#6B745C")
		backgroundColor = lipgloss.Color("#131813")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#E0E5D2"))
		marker = ReviewStyle.Foreground(lipgloss.Color("#D8A66A"))
	case "STORAGE RAIL":
		borderColor = lipgloss.Color("#566E60")
		backgroundColor = lipgloss.Color("#121814")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#DDE8DE"))
		marker = SafeStyle.Foreground(lipgloss.Color("#8FC5A0"))
	case "POWER RAIL":
		borderColor = lipgloss.Color("#7A6646")
		backgroundColor = lipgloss.Color("#18140F")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#F0DFC2"))
		marker = ReviewStyle.Foreground(lipgloss.Color("#D8A66A"))
	case "SESSION RAIL":
		borderColor = lipgloss.Color("#765F54")
		backgroundColor = lipgloss.Color("#171312")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#E5D4CF"))
		marker = HighStyle.Foreground(lipgloss.Color("#D97A68"))
	case "UTILITY RAIL":
		borderColor = lipgloss.Color("#566C61")
		backgroundColor = lipgloss.Color("#111814")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#D9E4DC"))
		marker = SafeStyle.Foreground(lipgloss.Color("#8FC5A0"))
	case "PROGRESS RAIL":
		borderColor = lipgloss.Color("#7B6742")
		backgroundColor = lipgloss.Color("#17130E")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#F0DEC0"))
		marker = ReviewStyle.Foreground(lipgloss.Color("#D8A66A"))
	case "REVIEW RAIL":
		borderColor = lipgloss.Color("#7A6541")
		backgroundColor = lipgloss.Color("#17130E")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#EFDDBD"))
		marker = ReviewStyle.Foreground(lipgloss.Color("#D8A66A"))
	case "SETTLED RAIL":
		borderColor = lipgloss.Color("#58705F")
		backgroundColor = lipgloss.Color("#111712")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#DDE7DF"))
		marker = SafeStyle.Foreground(lipgloss.Color("#8FC5A0"))
	case "TOOL DECK", "DIAGNOSIS DECK", "POLICY DECK":
		borderColor = lipgloss.Color("#45606A")
		backgroundColor = lipgloss.Color("#121B20")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#DDE9ED"))
		marker = RailStyle.Foreground(lipgloss.Color("#A7C5CF"))
	case "FOCUS DECK", "ACTION DECK", "OUTCOME DECK":
		borderColor = lipgloss.Color("#45606A")
		backgroundColor = lipgloss.Color("#121B20")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#DDE9ED"))
		marker = RailStyle.Foreground(lipgloss.Color("#A7C5CF"))
	case "RUN GATE":
		borderColor = lipgloss.Color("#745B3C")
		backgroundColor = lipgloss.Color("#18120D")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#F1DFC5"))
		marker = ReviewStyle.Foreground(lipgloss.Color("#D8A66A"))
	case "SETTLED GATE":
		borderColor = lipgloss.Color("#58705F")
		backgroundColor = lipgloss.Color("#111712")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#DDE7DF"))
		marker = SafeStyle.Foreground(lipgloss.Color("#8FC5A0"))
	case "CHECK RAIL":
		borderColor = lipgloss.Color("#7B6742")
		backgroundColor = lipgloss.Color("#17130E")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#F0DEC0"))
		marker = ReviewStyle.Foreground(lipgloss.Color("#D8A66A"))
	case "GUARD RAIL":
		borderColor = lipgloss.Color("#7A6541")
		backgroundColor = lipgloss.Color("#17130E")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#EFDDBD"))
		marker = ReviewStyle.Foreground(lipgloss.Color("#D8A66A"))
	case "CONTROL DECK":
		borderColor = lipgloss.Color("#4F6470")
		backgroundColor = lipgloss.Color("#12191E")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#E0EAEC"))
		marker = RailStyle.Foreground(lipgloss.Color("#A7C5CF"))
	case "SWEEP LANES":
		borderColor = lipgloss.Color("#8A6C44")
		backgroundColor = lipgloss.Color("#18130D")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#F1D3AE"))
		marker = ReviewStyle.Foreground(lipgloss.Color("#D8A66A"))
	case "SWEEP DECK":
		borderColor = lipgloss.Color("#466069")
		backgroundColor = lipgloss.Color("#121B20")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#DDE9ED"))
		marker = RailStyle.Foreground(lipgloss.Color("#A7C5CF"))
	case "DETAIL", "INSPECT", "FOLLOW-UP":
		borderColor = lipgloss.Color("#45606A")
		backgroundColor = lipgloss.Color("#121B20")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#DDE9ED"))
	case "QUEUE", "OPERATIONS":
		borderColor = lipgloss.Color("#7B6742")
		backgroundColor = lipgloss.Color("#17130E")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#F0DEC0"))
		marker = ReviewStyle.Foreground(lipgloss.Color("#D8A66A"))
	case "RESCAN", "COMPLETE":
		borderColor = lipgloss.Color("#6C6454")
		backgroundColor = lipgloss.Color("#151411")
		titleStyle = PanelTitleStyle.Foreground(lipgloss.Color("#E6DAC5"))
	}
	if active {
		backgroundColor = brightenTone(backgroundColor)
	}
	return PanelTheme{
		BorderColor:     borderColor,
		BackgroundColor: backgroundColor,
		TitleStyle:      titleStyle,
		Marker:          marker,
	}
}

// CardThemeForTone returns theme colors for a card based on label and tone.
func CardThemeForTone(label, tone string) CardTheme {
	borderColor := lipgloss.Color("#485A63")
	labelColor := lipgloss.Color("#8EA2AC")
	valueColor := lipgloss.Color("#A7C5CF")
	backgroundColor := lipgloss.Color("#141C22")
	switch tone {
	case "safe":
		borderColor = lipgloss.Color("#50685A")
		labelColor = lipgloss.Color("#A7C0B1")
		valueColor = lipgloss.Color("#8FC5A0")
		backgroundColor = lipgloss.Color("#121A16")
	case "review":
		borderColor = lipgloss.Color("#7A6847")
		labelColor = lipgloss.Color("#C9B48A")
		valueColor = lipgloss.Color("#D8A66A")
		backgroundColor = lipgloss.Color("#1C1711")
	case "high":
		borderColor = lipgloss.Color("#865649")
		labelColor = lipgloss.Color("#D5A28F")
		valueColor = lipgloss.Color("#D97A68")
		backgroundColor = lipgloss.Color("#211613")
	}
	labelUpper := normalizePanelName(label)
	switch labelUpper {
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
	return CardTheme{
		BorderColor:     borderColor,
		LabelColor:      labelColor,
		ValueColor:      valueColor,
		BackgroundColor: backgroundColor,
	}
}

// RouteCardTheme returns theme colors for a route card based on route, label, and tone.
func RouteCardTheme(route, label, tone string) CardTheme {
	cardTheme := CardThemeForTone(label, tone)
	normalized := normalizeRoute(route)
	switch normalized {
	case "clean":
		return routeCardToneToTheme(tone,
			lipgloss.Color("#4E6B5D"), lipgloss.Color("#B8D0C1"), lipgloss.Color("#9FD7AF"), lipgloss.Color("#111915"),
			lipgloss.Color("#8A7048"), lipgloss.Color("#E1C796"), lipgloss.Color("#E6B36F"), lipgloss.Color("#1D1710"),
			lipgloss.Color("#8A5A4C"), lipgloss.Color("#E0AE9B"), lipgloss.Color("#E08A74"), lipgloss.Color("#251713"),
		)
	case "uninstall":
		return routeCardToneToTheme(tone,
			lipgloss.Color("#54695E"), lipgloss.Color("#C0D1C9"), lipgloss.Color("#9CC6AD"), lipgloss.Color("#121817"),
			lipgloss.Color("#876445"), lipgloss.Color("#E2C099"), lipgloss.Color("#E2A66B"), lipgloss.Color("#1E1510"),
			lipgloss.Color("#93584D"), lipgloss.Color("#E2AB98"), lipgloss.Color("#E07667"), lipgloss.Color("#251612"),
		)
	case "analyze":
		return routeCardToneToTheme(tone,
			lipgloss.Color("#4B6E74"), lipgloss.Color("#B9D5DA"), lipgloss.Color("#90D0DB"), lipgloss.Color("#10191C"),
			lipgloss.Color("#6F7864"), lipgloss.Color("#D7D1B8"), lipgloss.Color("#D7B57C"), lipgloss.Color("#141612"),
			lipgloss.Color("#7C5B56"), lipgloss.Color("#D8B1A5"), lipgloss.Color("#DB8B7D"), lipgloss.Color("#211513"),
		)
	case "home", "status":
		return routeCardToneToTheme(tone,
			lipgloss.Color("#4C7075"), lipgloss.Color("#B7D5DA"), lipgloss.Color("#A7C5CF"), lipgloss.Color("#10191C"),
			lipgloss.Color("#5E6F78"), lipgloss.Color("#CBD6DD"), lipgloss.Color("#D4B27C"), lipgloss.Color("#12181D"),
			lipgloss.Color("#7A5D55"), lipgloss.Color("#D7B6AC"), lipgloss.Color("#D79283"), lipgloss.Color("#1B1513"),
		)
	case "progress":
		return routeCardToneToTheme(tone,
			lipgloss.Color("#546C5E"), lipgloss.Color("#C0D1C2"), lipgloss.Color("#9FD1AD"), lipgloss.Color("#121816"),
			lipgloss.Color("#816C48"), lipgloss.Color("#DBC69B"), lipgloss.Color("#E1AF72"), lipgloss.Color("#19150F"),
			lipgloss.Color("#895E4E"), lipgloss.Color("#DAB09E"), lipgloss.Color("#DE846D"), lipgloss.Color("#221713"),
		)
	case "result":
		return routeCardToneToTheme(tone,
			lipgloss.Color("#5C725F"), lipgloss.Color("#C7D5C7"), lipgloss.Color("#A7D1A2"), lipgloss.Color("#121712"),
			lipgloss.Color("#6E7159"), lipgloss.Color("#D5D4B7"), lipgloss.Color("#CFCB9B"), lipgloss.Color("#151610"),
			lipgloss.Color("#855E51"), lipgloss.Color("#DBB3A7"), lipgloss.Color("#DD8C79"), lipgloss.Color("#1F1513"),
		)
	case "review":
		return routeCardToneToTheme(tone,
			lipgloss.Color("#57706A"), lipgloss.Color("#C0D7CF"), lipgloss.Color("#97CEBF"), lipgloss.Color("#101816"),
			lipgloss.Color("#806A43"), lipgloss.Color("#E0CB9B"), lipgloss.Color("#E3B779"), lipgloss.Color("#19150E"),
			lipgloss.Color("#8C5C50"), lipgloss.Color("#DEB0A3"), lipgloss.Color("#DE8776"), lipgloss.Color("#221612"),
		)
	case "preflight":
		return routeCardToneToTheme(tone,
			lipgloss.Color("#567066"), lipgloss.Color("#C0D6CC"), lipgloss.Color("#95CDBA"), lipgloss.Color("#101715"),
			lipgloss.Color("#826A46"), lipgloss.Color("#E0C8A0"), lipgloss.Color("#E2B57C"), lipgloss.Color("#1A150E"),
			lipgloss.Color("#8E604E"), lipgloss.Color("#DFB39E"), lipgloss.Color("#E28A73"), lipgloss.Color("#211611"),
		)
	case "tools", "doctor", "protect":
		return routeCardToneToTheme(tone,
			lipgloss.Color("#557168"), lipgloss.Color("#BCD7CE"), lipgloss.Color("#97CFBF"), lipgloss.Color("#101817"),
			lipgloss.Color("#766C52"), lipgloss.Color("#D6CEB0"), lipgloss.Color("#D9BF88"), lipgloss.Color("#161610"),
			lipgloss.Color("#7A6159"), lipgloss.Color("#D8BAB0"), lipgloss.Color("#D58E84"), lipgloss.Color("#1B1514"),
		)
	default:
		return cardTheme
	}
}

func routeCardToneToTheme(
	tone string,
	safeBorder, safeLabel, safeValue, safeBackground lipgloss.Color,
	reviewBorder, reviewLabel, reviewValue, reviewBackground lipgloss.Color,
	highBorder, highLabel, highValue, highBackground lipgloss.Color,
) CardTheme {
	switch tone {
	case "safe":
		return CardTheme{BorderColor: safeBorder, LabelColor: safeLabel, ValueColor: safeValue, BackgroundColor: safeBackground}
	case "high":
		return CardTheme{BorderColor: highBorder, LabelColor: highLabel, ValueColor: highValue, BackgroundColor: highBackground}
	default:
		return CardTheme{BorderColor: reviewBorder, LabelColor: reviewLabel, ValueColor: reviewValue, BackgroundColor: reviewBackground}
	}
}

func normalizePanelName(name string) string {
	return uppercase(name)
}

func normalizeRoute(route string) string {
	switch lowercase(route) {
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
		return lowercase(route)
	}
}

// RouteLabelToken returns the 5-letter token for a route (e.g., "SCOUT", "FORGE").
func RouteLabelToken(route string) string {
	switch normalizeRoute(route) {
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

// brightenTone lightens a background color for active state.
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

func lowercase(s string) string {
	// Using simple implementation to avoid strings import
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func uppercase(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}