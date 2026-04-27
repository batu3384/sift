package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/batu3384/sift/internal/domain"
)

type cleanFlowLane struct {
	Label string
	Bytes int64
	Rows  []cleanFlowRow
}

type cleanFlowRow struct {
	Label string
	Bytes int64
	Path  string
	State string
}

type cleanFlowFocusEntry struct {
	Label string
	Meta  string
}

func cleanFlowStats(m cleanFlowModel, width int) []string {
	cardWidth := 24
	if width < 110 {
		cardWidth = width - 8
	}
	safeBytes, reviewBytes, totalBytes := cleanFlowByteMix(m)
	return []string{
		renderRouteStatCard("clean", "sweep", cleanFlowPhaseCardValue(m), "review", cardWidth),
		renderRouteStatCard("clean", "clear", cleanFlowHumanBytes(safeBytes), "safe", cardWidth),
		renderRouteStatCard("clean", "watch", cleanFlowHumanBytes(reviewBytes), "review", cardWidth),
		renderRouteStatCard("clean", "yield", cleanFlowHumanBytes(totalBytes), "high", cardWidth),
	}
}

func cleanFlowPhaseCardValue(m cleanFlowModel) string {
	phase := m.phase
	value := "READY"
	switch phase {
	case cleanFlowScanning:
		value = "SCANNING"
	case cleanFlowReviewReady:
		value = "FROZEN"
	case cleanFlowPermissions:
		value = "ACCESS"
	case cleanFlowReclaiming:
		value = "RECLAIM"
	case cleanFlowResult:
		value = "SETTLED"
	}
	if meter := routeStickyPhaseCount("clean", cleanFlowSignalLabel(m)); meter != "" {
		return value + " " + meter
	}
	return value
}

func cleanFlowByteMix(m cleanFlowModel) (safeBytes, reviewBytes, totalBytes int64) {
	if !m.preview.loaded {
		for _, row := range m.scanRows {
			totalBytes += row.Bytes
		}
		return 0, 0, totalBytes
	}
	safeBytes = m.preview.plan.Totals.SafeBytes
	reviewBytes = m.preview.plan.Totals.ReviewBytes + m.preview.plan.Totals.HighBytes
	totalBytes = planDisplayBytes(m.preview.plan)
	return
}

func cleanFlowLanes(m cleanFlowModel) []cleanFlowLane {
	if len(m.scanRows) > 0 {
		switch m.phase {
		case cleanFlowScanning, cleanFlowReviewReady, cleanFlowPermissions, cleanFlowReclaiming, cleanFlowResult:
			return cleanFlowLanesFromScanRows(m.scanRows)
		}
	}
	if m.preview.loading && len(m.scanRows) > 0 {
		return cleanFlowLanesFromScanRows(m.scanRows)
	}
	if m.preview.loaded && len(m.preview.plan.Items) > 0 {
		return cleanFlowLanesFromPlan(m.preview.plan)
	}
	return cleanFlowPlaceholderLanes(m)
}

func cleanFlowPlaceholderLanes(m cleanFlowModel) []cleanFlowLane {
	action, ok := m.selectedAction()
	if !ok {
		return nil
	}
	rows := make([]cleanFlowRow, 0, len(action.Modules))
	for idx, module := range action.Modules {
		state := "queued"
		if idx == 0 && m.phase == cleanFlowScanning {
			state = "scanning"
		}
		rows = append(rows, cleanFlowRow{Label: module, State: state})
	}
	return []cleanFlowLane{{
		Label: strings.TrimSpace(action.Title) + " lane",
		Rows:  rows,
	}}
}

func cleanFlowLanesFromPlan(plan domain.ExecutionPlan) []cleanFlowLane {
	byCategory := map[domain.Category]*cleanFlowLane{}
	order := []domain.Category{}
	for idx, item := range plan.Items {
		category := item.Category
		lane := byCategory[category]
		if lane == nil {
			order = append(order, category)
			lane = &cleanFlowLane{Label: cleanFlowLaneLabel(item)}
			byCategory[category] = lane
		}
		state := "queued"
		switch {
		case item.Status == domain.StatusProtected:
			state = "protected"
		case item.Risk == domain.RiskReview || item.Risk == domain.RiskHigh:
			state = "review"
		case idx == 0:
			state = "focus"
		}
		lane.Rows = append(lane.Rows, cleanFlowRow{
			Label: cleanFlowRowLabel(item),
			Bytes: item.Bytes,
			Path:  strings.TrimSpace(item.DisplayPath),
			State: state,
		})
		lane.Bytes += item.Bytes
	}
	lanes := make([]cleanFlowLane, 0, len(order))
	for _, category := range order {
		lane := byCategory[category]
		if lane == nil {
			continue
		}
		if len(lane.Rows) > 4 {
			lane.Rows = lane.Rows[:4]
		}
		lanes = append(lanes, *lane)
	}
	return lanes
}

func cleanFlowLanesFromScanRows(rows []cleanFlowScanRow) []cleanFlowLane {
	byLane := map[string]*cleanFlowLane{}
	lastSeen := map[string]int{}
	for idx, row := range rows {
		laneLabel := strings.TrimSpace(row.Lane)
		if laneLabel == "" {
			laneLabel = "Cleanup lane"
		}
		lane := byLane[laneLabel]
		if lane == nil {
			lane = &cleanFlowLane{Label: laneLabel}
			byLane[laneLabel] = lane
		}
		state := row.State
		if row.Items > 1 {
			state = fmt.Sprintf("%d %s", row.Items, pl(row.Items, "item", "items"))
		} else if strings.TrimSpace(state) == "" {
			state = "queued"
		}
		lane.Rows = append(lane.Rows, cleanFlowRow{
			Label: row.Label,
			Bytes: row.Bytes,
			Path:  row.Path,
			State: state,
		})
		lane.Bytes += row.Bytes
		lastSeen[laneLabel] = idx
	}
	order := make([]string, 0, len(byLane))
	for laneLabel := range byLane {
		order = append(order, laneLabel)
	}
	sort.SliceStable(order, func(i, j int) bool {
		return lastSeen[order[i]] > lastSeen[order[j]]
	})
	lanes := make([]cleanFlowLane, 0, len(order))
	for _, label := range order {
		if lane := byLane[label]; lane != nil {
			lanes = append(lanes, *lane)
		}
	}
	return lanes
}

func cleanFlowLaneLabel(item domain.Finding) string {
	switch item.Category {
	case domain.CategoryBrowserData:
		return "Browser lane"
	case domain.CategoryDeveloperCaches:
		return "Dev lane"
	case domain.CategoryPackageCaches:
		return "Package lane"
	case domain.CategoryAppLeftovers:
		return "Residue lane"
	default:
		if group := strings.TrimSpace(domain.ExecutionGroupLabel(item)); group != "" {
			return group + " lane"
		}
		return "Cleanup lane"
	}
}

func cleanFlowRowLabel(item domain.Finding) string {
	if label := strings.TrimSpace(item.Name); label != "" {
		return label
	}
	if label := strings.TrimSpace(item.DisplayPath); label != "" {
		return label
	}
	return strings.TrimSpace(item.Path)
}

func cleanFlowFocusTitle(m cleanFlowModel) string {
	if m.preview.loading && len(m.scanRows) > 0 {
		row := m.scanRows[len(m.scanRows)-1]
		if row.Bytes > 0 {
			return row.Label + "  •  " + cleanFlowHumanBytes(row.Bytes)
		}
		return row.Label
	}
	if m.preview.loaded && len(m.preview.plan.Items) > 0 {
		item := m.preview.plan.Items[0]
		label := cleanFlowRowLabel(item)
		if item.Bytes > 0 {
			return label + "  •  " + cleanFlowHumanBytes(item.Bytes)
		}
		return label
	}
	action, ok := m.selectedAction()
	if !ok {
		return "Scan focus pending"
	}
	return action.Title + "  •  preview warming"
}

func cleanFlowFocusCopy(m cleanFlowModel) string {
	switch {
	case m.phase == cleanFlowPermissions:
		return "Access hold stays inside the sweep."
	case m.phase == cleanFlowReclaiming:
		return "Live reclaim is rolling through the same sweep rows."
	case m.phase == cleanFlowResult && m.hasResult:
		return "The sweep settled. Check yield, watch items, and the next pass."
	case m.preview.loaded && len(m.scanRows) > 0:
		return "Review froze the live sweep in place."
	case m.preview.loading && len(m.scanRows) > 0:
		row := m.scanRows[len(m.scanRows)-1]
		return fmt.Sprintf("%d %s found in %s.", row.Items, pl(row.Items, "item", "items"), row.Label)
	case m.preview.loading:
		return "Sweep rail is warming live."
	case m.preview.loaded:
		return "Preview is ready on this same sweep."
	case m.preview.err != "":
		return "Preview failed, but the sweep rail stayed up."
	default:
		return "Sweep, gate, reclaim, settled."
	}
}

func cleanFlowGateLine(m cleanFlowModel) string {
	switch m.phase {
	case cleanFlowPermissions:
		return "access manifest"
	case cleanFlowReclaiming:
		return "same ledger"
	case cleanFlowResult:
		if _, _, failed, _, protected := countResultStatuses(m.result); failed > 0 || protected > 0 {
			return "recovery ready"
		}
		return "run settled"
	default:
		return "review gate on"
	}
}

func cleanFlowStatusLine(m cleanFlowModel) string {
	switch m.phase {
	case cleanFlowScanning:
		return "live sweep running"
	case cleanFlowReviewReady:
		return "waiting at review gate"
	case cleanFlowPermissions:
		return "access checks running"
	case cleanFlowReclaiming:
		return "reclaim pass running"
	case cleanFlowResult:
		return "run settled"
	default:
		return "controller ready"
	}
}

func cleanFlowNextLine(m cleanFlowModel) string {
	switch m.phase {
	case cleanFlowScanning:
		return "review gate"
	case cleanFlowReviewReady:
		return "access check if needed"
	case cleanFlowPermissions:
		return "reclaim rail"
	case cleanFlowReclaiming:
		return "settled rail"
	case cleanFlowResult:
		return "inspect watch or rerun"
	default:
		return "start sweep"
	}
}

func cleanFlowCurrentSweepEntry(m cleanFlowModel) (cleanFlowFocusEntry, bool) {
	if len(m.scanRows) > 0 {
		activeIdx := -1
		for idx := len(m.scanRows) - 1; idx >= 0; idx-- {
			switch m.scanRows[idx].State {
			case "scanning", "reclaiming", "verifying", "ready", "focus":
				activeIdx = idx
			}
		}
		if activeIdx == -1 {
			activeIdx = len(m.scanRows) - 1
		}
		row := m.scanRows[activeIdx]
		return cleanFlowFocusEntry{
			Label: cleanFlowFocusEntryLabel(row.Label, row.Bytes),
			Meta:  cleanFlowFocusEntryMeta(row.Lane, row.State),
		}, true
	}
	if m.preview.loaded && len(m.preview.plan.Items) > 0 {
		item := m.preview.plan.Items[0]
		return cleanFlowFocusEntry{
			Label: cleanFlowFocusEntryLabel(cleanFlowRowLabel(item), item.Bytes),
			Meta:  cleanFlowFocusEntryMeta(cleanFlowLaneLabel(item), "ready"),
		}, true
	}
	return cleanFlowFocusEntry{}, false
}

func cleanFlowNextReclaimEntry(m cleanFlowModel) (cleanFlowFocusEntry, bool) {
	if len(m.scanRows) > 1 {
		activeIdx := -1
		for idx := len(m.scanRows) - 1; idx >= 0; idx-- {
			switch m.scanRows[idx].State {
			case "scanning", "reclaiming", "verifying", "ready", "focus":
				activeIdx = idx
			}
		}
		for idx := len(m.scanRows) - 1; idx >= 0; idx-- {
			if idx == activeIdx {
				continue
			}
			row := m.scanRows[idx]
			return cleanFlowFocusEntry{
				Label: cleanFlowFocusEntryLabel(row.Label, row.Bytes),
				Meta:  cleanFlowFocusEntryMeta(row.Lane, row.State),
			}, true
		}
	}
	if m.preview.loaded && len(m.preview.plan.Items) > 1 {
		item := m.preview.plan.Items[1]
		return cleanFlowFocusEntry{
			Label: cleanFlowFocusEntryLabel(cleanFlowRowLabel(item), item.Bytes),
			Meta:  cleanFlowFocusEntryMeta(cleanFlowLaneLabel(item), cleanFlowPlanRowState(item, 1)),
		}, true
	}
	return cleanFlowFocusEntry{}, false
}

func cleanFlowFocusEntryLabel(label string, bytes int64) string {
	label = strings.TrimSpace(label)
	if bytes > 0 {
		return label + "  •  " + cleanFlowHumanBytes(bytes)
	}
	return label
}

func cleanFlowFocusEntryMeta(lane string, state string) string {
	parts := []string{}
	if lane = strings.TrimSpace(lane); lane != "" {
		parts = append(parts, lane)
	}
	if state = strings.TrimSpace(state); state != "" {
		parts = append(parts, state)
	}
	return strings.Join(parts, "  •  ")
}

func cleanFlowPlanRowState(item domain.Finding, idx int) string {
	switch {
	case item.Status == domain.StatusProtected:
		return "protected"
	case item.Risk == domain.RiskReview || item.Risk == domain.RiskHigh:
		return "review"
	case idx == 0:
		return "focus"
	default:
		return "queued"
	}
}

func cleanFlowHumanBytes(size int64) string {
	label := domain.HumanBytes(size)
	return strings.ReplaceAll(label, ".0 ", " ")
}

func cleanFlowScanLaneLabel(action homeAction, ruleID string, ruleName string) string {
	text := strings.ToLower(strings.TrimSpace(ruleID + " " + ruleName))
	matchModule := func(keyword string) string {
		for _, module := range action.Modules {
			if strings.Contains(strings.ToLower(module), keyword) {
				return module
			}
		}
		return ""
	}
	switch {
	case containsAny(text, "browser", "chrome", "firefox", "safari", "edge", "webkit"):
		if module := matchModule("browser"); module != "" {
			return module
		}
	case containsAny(text, "developer", "xcode", "deriveddata", "go ", "build", "gradle", "maven", "pod", "simulator"):
		if module := matchModule("developer"); module != "" {
			return module
		}
	case containsAny(text, "package", "homebrew", "npm", "pnpm", "yarn", "pip", "cargo", "gem", "composer", "cache"):
		if module := matchModule("package"); module != "" {
			return module
		}
	case containsAny(text, "installer", "dmg", "pkg", "xip", "archive"):
		if module := matchModule("installer"); module != "" {
			return module
		}
	case containsAny(text, "leftover", "remnant", "application support", "logs", "temporary", "temp", "clutter"):
		if module := matchModule("leftover"); module != "" {
			return module
		}
		if module := matchModule("logs"); module != "" {
			return module
		}
		if module := matchModule("temporary"); module != "" {
			return module
		}
		if module := matchModule("clutter"); module != "" {
			return module
		}
	}
	if len(action.Modules) > 0 {
		return action.Modules[0]
	}
	return "Cleanup lane"
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func cleanFlowAccessLine(m cleanFlowModel) string {
	if !m.preflight.required() {
		return ""
	}
	parts := []string{}
	if m.preflight.needsAdmin {
		parts = append(parts, formatPreflightCount(m.preflight.adminItems, "admin"))
	}
	if m.preflight.needsDialogs {
		parts = append(parts, formatPreflightCount(m.preflight.dialogItems, "dialog"))
	}
	if m.preflight.needsNative {
		parts = append(parts, formatPreflightCount(m.preflight.nativeItems, "native"))
	}
	return strings.Join(parts, "  •  ")
}

func cleanFlowOutcomeLine(m cleanFlowModel) string {
	if !m.hasResult {
		return ""
	}
	completed, deleted, failed, skipped, protected := countResultStatuses(m.result)
	parts := []string{}
	if freed := resultFreedBytes(m.preview.plan, m.result); freed > 0 {
		parts = append(parts, cleanFlowHumanBytes(freed)+" reclaimed")
	}
	if settled := completed + deleted; settled > 0 {
		parts = append(parts, fmt.Sprintf("%d settled", settled))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	if protected > 0 {
		parts = append(parts, fmt.Sprintf("%d protected", protected))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}
	return strings.Join(parts, "  •  ")
}
