package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type routeDeckChrome struct {
	Callsign string
	Doctrine string
	HotHint  string
	NextHint string
}

func routeDeckChromeForRoute(route string) routeDeckChrome {
	switch normalizeCardRoute(route) {
	case "clean":
		return routeDeckChrome{
			Callsign: "FORGE",
			Doctrine: "item-first sweep",
			HotHint:  "residue stays live",
			NextHint: "queued reclaim in order",
		}
	case "uninstall":
		return routeDeckChrome{
			Callsign: "COURIER",
			Doctrine: "handoff continuity across native and remnants",
			HotHint:  "target stays warm through handoff",
			NextHint: "remnants hold behind the live pass",
		}
	case "analyze":
		return routeDeckChrome{
			Callsign: "ORACLE",
			Doctrine: "trace focus before reclaim",
			HotHint:  "trace context stays live",
			NextHint: "staged findings keep reclaim order",
		}
	default:
		return routeDeckChrome{
			Callsign: "SIGNAL",
			Doctrine: "steady operator focus",
			HotHint:  "active work stays visible",
			NextHint: "next pass keeps order",
		}
	}
}

func renderRouteDeckTag(route, label, meta string) string {
	chrome := routeDeckChromeForRoute(route)
	label = strings.ToUpper(strings.TrimSpace(label))
	meta = strings.ToUpper(strings.TrimSpace(meta))
	prefix := strings.TrimSpace(chrome.Callsign + " " + label)
	tone := routeDeckTone(meta)
	_, labelColor, valueColor, backgroundColor := routeCardTonePalette(route, prefix, tone)
	tagStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(labelColor).
		Background(backgroundColor).
		Padding(0, 1)
	if meta == "" {
		return tagStyle.Render(prefix)
	}
	metaStyle := lipgloss.NewStyle().Bold(true).Foreground(valueColor)
	return tagStyle.Render(prefix) + "  " + metaStyle.Render(meta)
}

func routeDeckHint(route, label string) string {
	chrome := routeDeckChromeForRoute(route)
	switch routeDeckSlot(label) {
	case "hot":
		return chrome.HotHint
	case "next":
		return chrome.NextHint
	default:
		return chrome.Doctrine
	}
}

func renderRouteStickyRail(route, phase string, motion motionState, width int) string {
	chrome := routeDeckChromeForRoute(route)
	lead := strings.ToUpper(strings.TrimSpace(chrome.Callsign + " RAIL"))
	phase = strings.ToUpper(strings.TrimSpace(phase))
	if phase == "" {
		phase = "STEADY RAIL"
	}
	tone := routeStickyRailTone(motion)
	borderColor, labelColor, valueColor, backgroundColor := routeCardTonePalette(route, lead, tone)
	markerStyle := lipgloss.NewStyle().Bold(true).Foreground(valueColor)
	leadStyle := lipgloss.NewStyle().Bold(true).Foreground(labelColor)
	phaseStyle := lipgloss.NewStyle().Bold(true).Foreground(valueColor)
	parts := []string{
		markerStyle.Render(spinnerGlyph(motion)),
		leadStyle.Render(lead),
		phaseStyle.Render(phase),
	}
	line := strings.Join(parts, "  ")
	if chrome.Doctrine != "" {
		line += "  •  " + panelMetaStyle.Render(chrome.Doctrine)
	}
	return renderRouteStickyBand(route, line, motion, width, borderColor, backgroundColor)
}

func renderRouteStickyDivider(route, phase string, motion motionState, width int) string {
	chrome := routeDeckChromeForRoute(route)
	label := strings.ToUpper(strings.TrimSpace(chrome.Callsign))
	tone := routeStickyRailTone(motion)
	borderColor, _, _, backgroundColor := routeCardTonePalette(route, label, tone)
	divider := "╺━━━ " + label + " ━━━╸"
	if meter := routeStickyPhaseMeter(route, phase); meter != "" {
		divider += "  " + meter
	}
	return renderRouteStickyBand(route, lipgloss.NewStyle().Bold(true).Foreground(borderColor).Render(divider), motion, width, borderColor, backgroundColor)
}

func renderRoutePinnedBlock(route, phase string, motion motionState, width int, banners ...string) []string {
	lines := []string{}
	for _, banner := range banners {
		if strings.TrimSpace(banner) == "" {
			continue
		}
		lines = append(lines, renderRouteStickyBanner(route, banner, motion, width))
	}
	lines = append(lines,
		renderRouteStickyRail(route, phase, motion, width),
		renderRouteStickyDivider(route, phase, motion, width),
	)
	return lines
}

func renderRouteStickyBanner(route, banner string, motion motionState, width int) string {
	label := strings.ToUpper(strings.TrimSpace(routeDeckChromeForRoute(route).Callsign))
	tone := routeStickyRailTone(motion)
	borderColor, _, _, backgroundColor := routeCardTonePalette(route, label, tone)
	return renderRouteStickyBand(route, banner, motion, width, borderColor, backgroundColor)
}

func renderRouteStickyBand(route, line string, motion motionState, width int, borderColor lipgloss.Color, backgroundColor lipgloss.Color) string {
	line = singleLine(line, max(width-6, 12))
	stripWidth := max(width-2, 10)
	capStyle := lipgloss.NewStyle().Bold(true).Foreground(borderColor).Background(backgroundColor)
	stripStyle := lipgloss.NewStyle().Background(backgroundColor).Padding(0, 1).Width(stripWidth)
	return capStyle.Render("▌") + stripStyle.Render(line) + capStyle.Render("▐")
}

func routeStickyRailTone(motion motionState) string {
	switch motion.Mode {
	case motionModeAlert:
		return "high"
	case motionModeProgress:
		if motion.Pulse {
			return "safe"
		}
		return "review"
	case motionModeReview:
		return "review"
	default:
		return "review"
	}
}

func routeStickyPhaseMeter(route, phase string) string {
	phase = strings.ToUpper(strings.TrimSpace(phase))
	meters := []string{}
	switch normalizeCardRoute(route) {
	case "clean":
		meters = []string{
			"SCAN RAIL|SCAN 01/05",
			"REVIEW RAIL|REVIEW 02/05",
			"ACCESS RAIL|ACCESS 03/05",
			"RECLAIM RAIL|RECLAIM 04/05",
			"SETTLED RAIL|SETTLED 05/05",
		}
	case "uninstall":
		meters = []string{
			"TARGET RAIL|TARGET 01/05",
			"REVIEW RAIL|REVIEW 02/05",
			"ACCESS RAIL|ACCESS 03/05",
			"HANDOFF RAIL|HANDOFF 04/05",
			"AFTERCARE RAIL|AFTERCARE 05/05",
		}
	case "analyze":
		meters = []string{
			"TRACE RAIL|TRACE 01/05",
			"REVIEW RAIL|REVIEW 02/05",
			"ACCESS RAIL|ACCESS 03/05",
			"RECLAIM RAIL|RECLAIM 04/05",
			"SETTLED RAIL|SETTLED 05/05",
		}
	}
	for _, entry := range meters {
		parts := strings.SplitN(entry, "|", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[0] == phase {
			return parts[1]
		}
	}
	return ""
}

func routeStickyPhaseCount(route, phase string) string {
	meter := routeStickyPhaseMeter(route, phase)
	if meter == "" {
		return ""
	}
	parts := strings.Fields(meter)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func routeDeckSlot(label string) string {
	switch strings.ToUpper(strings.TrimSpace(label)) {
	case "HOT PATH":
		return "hot"
	case "NEXT PASS":
		return "next"
	default:
		return ""
	}
}

func routeDeckTone(meta string) string {
	upper := strings.ToUpper(strings.TrimSpace(meta))
	switch {
	case upper == "":
		return "review"
	case strings.Contains(upper, "FAILED"), strings.Contains(upper, "GUARDED"), strings.Contains(upper, "PROTECTED"), strings.Contains(upper, "WATCH"):
		return "high"
	case strings.Contains(upper, "SETTLED"), strings.Contains(upper, "CLEARED"), strings.Contains(upper, "REMOVED"):
		return "safe"
	default:
		return "review"
	}
}
