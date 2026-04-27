package tui

import "strings"

type routeSignalSignature struct {
	Mascot   string
	Doctrine string
}

func routeSignalSignatureForRoute(route string) routeSignalSignature {
	switch strings.ToLower(strings.TrimSpace(route)) {
	case "clean":
		return routeSignalSignature{
			Mascot:   "FORGE RAIL",
			Doctrine: "item-first reclaim discipline",
		}
	case "uninstall":
		return routeSignalSignature{
			Mascot:   "COURIER RAIL",
			Doctrine: "handoff continuity",
		}
	case "analyze":
		return routeSignalSignature{
			Mascot:   "ORACLE RAIL",
			Doctrine: "trace focus",
		}
	case "home":
		return routeSignalSignature{
			Mascot:   "SCOUT RAIL",
			Doctrine: "route launch discipline",
		}
	case "status":
		return routeSignalSignature{
			Mascot:   "PULSE RAIL",
			Doctrine: "live observatory focus",
		}
	default:
		return routeSignalSignature{
			Mascot:   "SIGNAL RAIL",
			Doctrine: "steady operator focus",
		}
	}
}

func routeHasNamedSignal(route string) bool {
	switch normalizeCardRoute(route) {
	case "clean", "uninstall", "analyze", "home", "status":
		return true
	default:
		return false
	}
}

func renderRouteSignalBlock(route string, motion motionState, load float64, summary string, meta ...string) string {
	signature := routeSignalSignatureForRoute(route)
	extras := make([]string, 0, len(meta))
	for _, part := range meta {
		part = strings.TrimSpace(part)
		if part != "" {
			extras = append(extras, part)
		}
	}
	lines := []string{}
	head := railStyle.Render(signature.Mascot)
	if summary = strings.TrimSpace(summary); summary != "" {
		head += "  " + panelMetaStyle.Render(summary)
	}
	lines = append(lines, head)
	footParts := make([]string, 0, 1+len(extras))
	if signature.Doctrine != "" {
		footParts = append(footParts, signature.Doctrine)
	}
	footParts = append(footParts, extras...)
	if len(footParts) > 0 {
		lines = append(lines, mutedStyle.Render(strings.Join(footParts, "  •  ")))
	}
	return joinPanels(
		compactMascotFrame(motion, load),
		strings.Join(lines, "\n"),
		40,
	)
}
