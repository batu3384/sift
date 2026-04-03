package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/batu3384/sift/internal/domain"
)

func progressSummaryLine(progress progressModel, stage stageInfo) string {
	if stage.Total == 0 {
		return "Summary  waiting for the first approved item"
	}
	switch progress.plan.Command {
	case "clean":
		freed := progressFreedBytes(progress)
		total := progressTotalBytes(progress.plan)
		bytesLabel := fmt.Sprintf("%s / %s freed", domain.HumanBytes(freed), domain.HumanBytes(total))
		mods := planModuleCount(progress.plan)
		return fmt.Sprintf(
			"Summary  clean stage %d/%d  •  %d/%d settled  •  %d %s  •  %s",
			stage.Index,
			stage.Total,
			len(progress.items),
			len(progress.plan.Items),
			mods, pl(mods, "module", "modules"),
			bytesLabel,
		)
	case "uninstall":
		targets := uninstallTargetCount(progress.plan)
		if targets == 0 {
			targets = planModuleCount(progress.plan)
		}
		return fmt.Sprintf(
			"Summary  uninstall target %d/%d  •  %d/%d settled  •  %d %s  •  %s remnants",
			stage.Index,
			stage.Total,
			len(progress.items),
			len(progress.plan.Items),
			targets, pl(targets, "app", "apps"),
			domain.HumanBytes(progressTotalBytes(progress.plan)),
		)
	case "optimize":
		tasks := actionableCount(progress.plan)
		phases := max(maintenancePhaseCount(progress.plan), 1)
		return fmt.Sprintf(
			"Summary  task %d/%d  •  %d/%d settled  •  %d %s  •  %d %s",
			stage.Index,
			stage.Total,
			len(progress.items),
			len(progress.plan.Items),
			tasks, pl(tasks, "task", "tasks"),
			phases, pl(phases, "phase", "phases"),
		)
	case "autofix":
		fixes := actionableCount(progress.plan)
		return fmt.Sprintf(
			"Summary  fix %d/%d  •  %d/%d settled  •  %d %s  •  %d suggested",
			stage.Index,
			stage.Total,
			len(progress.items),
			len(progress.plan.Items),
			fixes, pl(fixes, "fix", "fixes"),
			suggestedTaskCount(progress.plan),
		)
	default:
		mods := planModuleCount(progress.plan)
		return fmt.Sprintf(
			"Summary  stage %d/%d  •  %d/%d settled  •  %d %s  •  %s",
			stage.Index,
			stage.Total,
			len(progress.items),
			len(progress.plan.Items),
			mods, pl(mods, "module", "modules"),
			domain.HumanBytes(progressTotalBytes(progress.plan)),
		)
	}
}

func progressStatusLine(progress progressModel) string {
	status := progressCurrentLine(progress)
	settled := strings.TrimPrefix(progressExecutionRail(progress), "Settled  •  ")
	if settled == "" {
		return "Status   " + status
	}
	return "Status   " + status + "  •  " + settled
}

func progressStepLine(progress progressModel) string {
	if progress.current == nil {
		return "Step     waiting for first item"
	}
	item := *progress.current
	label := strings.TrimSpace(progress.currentDetail)
	if label == "" {
		label = progressCurrentActionLabel(progress)
	}
	switch progress.plan.Command {
	case "clean":
		return "Step     reclaim " + progressStepLabel(progress, item) + "  •  " + label
	case "uninstall":
		if item.Action == domain.ActionNative {
			return "Step     handoff  •  " + label
		}
		if phase := strings.TrimSpace(item.TaskPhase); phase != "" {
			return "Step     " + phase + "  •  " + label
		}
		return "Step     remnants  •  " + label
	case "optimize":
		if phase := progressActiveTaskPhase(progress); phase != "" {
			return "Step     " + phase + "  •  " + label
		}
		return "Step     maintenance  •  " + label
	case "autofix":
		if phase := progressActiveTaskPhase(progress); phase != "" {
			return "Step     " + phase + "  •  " + label
		}
		return "Step     fix  •  " + label
	default:
		return "Step     " + progressStepLabel(progress, item) + "  •  " + label
	}
}

func progressCurrentLine(progress progressModel) string {
	switch {
	case progress.cancelRequested:
		return "stop requested  •  finishing the active operation before returning results"
	case progress.current != nil && progress.currentPhase == "" && len(progress.items) == 0:
		return "queued first approved item  •  " + displayFindingLabel(*progress.current)
	case progress.current != nil && progressPhaseActive(progress.currentPhase):
		if runningCursor, ok := progress.runningCursor(); ok && !progress.autoFollow && progress.cursor != runningCursor {
			return "monitoring active work  •  browsing current batch"
		}
		return progressCurrentActionLabel(progress)
	case len(progress.plan.Items) > 0 && len(progress.items) == len(progress.plan.Items):
		return "finalizing result batch  •  all approved items settled"
	default:
		return "waiting for the next approved item"
	}
}

func progressCurrentActionLabel(progress progressModel) string {
	item := *progress.current
	if progress.currentDetail != "" {
		return progress.currentDetail
	}
	label := displayFindingLabel(item)
	switch item.Action {
	case domain.ActionNative:
		return "opening native uninstall  •  " + label
	case domain.ActionCommand:
		return "running task  •  " + label
	case domain.ActionPermanent:
		return "deleting item  •  " + label
	case domain.ActionAdvisory:
		return "reviewing advisory step  •  " + label
	default:
		return "moving item to trash  •  " + label
	}
}

func progressStepLabel(progress progressModel, item domain.Finding) string {
	step := strings.TrimSpace(progress.currentStep)
	switch {
	case step == "":
		switch progress.currentPhase {
		case domain.ProgressPhaseStarting:
			return "queue"
		case domain.ProgressPhasePreparing:
			return "check"
		case domain.ProgressPhaseRunning:
			return "run"
		case domain.ProgressPhaseVerifying:
			return "verify"
		case domain.ProgressPhaseFinished:
			return "settle"
		default:
			return "run"
		}
	case step == "trash":
		return "trash"
	case step == "delete":
		return "delete"
	case step == "check":
		return "check"
	case step == "queue":
		return "queue"
	case step == "verify":
		return "verify"
	case item.Action == domain.ActionNative && step == "launch":
		return "handoff"
	default:
		return step
	}
}

func displayFindingLabel(item domain.Finding) string {
	label := strings.TrimSpace(item.DisplayPath)
	if label == "" {
		label = strings.TrimSpace(item.Path)
	}
	if label == "" {
		label = strings.TrimSpace(item.Name)
	}
	return label
}

func progressFreedBytes(progress progressModel) int64 {
	byID := make(map[string]int64, len(progress.plan.Items))
	byPath := make(map[string]int64, len(progress.plan.Items))
	for _, pi := range progress.plan.Items {
		if pi.ID != "" {
			byID[pi.ID] = pi.Bytes
		}
		if pi.Path != "" {
			byPath[strings.TrimSpace(pi.Path)] = pi.Bytes
		}
	}
	var freed int64
	for _, result := range progress.items {
		if result.Status != domain.StatusDeleted && result.Status != domain.StatusCompleted {
			continue
		}
		if result.FindingID != "" {
			if b, ok := byID[result.FindingID]; ok {
				freed += b
				continue
			}
		}
		if p := strings.TrimSpace(result.Path); p != "" {
			if b, ok := byPath[p]; ok {
				freed += b
			}
		}
	}
	return freed
}

func progressTotalBytes(plan domain.ExecutionPlan) int64 {
	if plan.Totals.Bytes > 0 {
		return plan.Totals.Bytes
	}
	var total int64
	for _, item := range plan.Items {
		total += item.Bytes
	}
	return total
}

func progressSummary(progress progressModel) string {
	switch progress.plan.Command {
	case "clean":
		mods := planModuleCount(progress.plan)
		return fmt.Sprintf("Reclaimed %d of %d  •  %d %s  •  %s", len(progress.items), len(progress.plan.Items), mods, pl(mods, "module", "modules"), domain.HumanBytes(progress.plan.Totals.Bytes))
	case "uninstall":
		apps := resultScopeCount(progress.plan)
		return fmt.Sprintf("Settled %d of %d  •  %d %s  •  %s remnants", len(progress.items), len(progress.plan.Items), apps, pl(apps, "app", "apps"), domain.HumanBytes(progress.plan.Totals.Bytes))
	case "optimize":
		tasks := actionableCount(progress.plan)
		phases := max(maintenancePhaseCount(progress.plan), 1)
		return fmt.Sprintf("Applied %d of %d  •  %d %s  •  %d %s", len(progress.items), len(progress.plan.Items), tasks, pl(tasks, "task", "tasks"), phases, pl(phases, "phase", "phases"))
	case "autofix":
		tasks := actionableCount(progress.plan)
		return fmt.Sprintf("Applied %d of %d  •  %d %s  •  %d suggested", len(progress.items), len(progress.plan.Items), tasks, pl(tasks, "task", "tasks"), suggestedTaskCount(progress.plan))
	}
	mods := planModuleCount(progress.plan)
	return fmt.Sprintf("Processed %d of %d  •  %d %s  •  %s reclaimable",
		len(progress.items),
		len(progress.plan.Items),
		mods,
		pl(mods, "module", "modules"),
		domain.HumanBytes(progress.plan.Totals.Bytes),
	)
}

type stageInfo struct {
	Category domain.Category
	Group    string
	Index    int
	Total    int
	Done     int
	Items    int
	Bytes    int64
	Freed    int64
}

type stageBucket struct {
	category domain.Category
	label    string
	total    int
	done     int
	bytes    int64
	freed    int64
}

func progressStageInfo(progress progressModel) stageInfo {
	if progress.currentSection.Total > 0 {
		return progress.currentSection
	}
	order, buckets := progressStageBuckets(progress)
	if len(order) == 0 {
		return stageInfo{}
	}
	currentKey := order[0]
	if progress.current != nil && progress.current.Category != "" {
		currentKey = progressGroupKey(*progress.current)
	} else if len(progress.items) > 0 {
		currentKey = progressGroupKey(progress.plan.Items[min(len(progress.items)-1, len(progress.plan.Items)-1)])
	} else if progress.cursor >= 0 && progress.cursor < len(progress.plan.Items) {
		currentKey = progressGroupKey(progress.plan.Items[progress.cursor])
	}
	index := 1
	for idx, key := range order {
		if key == currentKey {
			index = idx + 1
			break
		}
	}
	current := buckets[currentKey]
	if current == nil {
		return stageInfo{}
	}
	return stageInfo{
		Category: current.category,
		Group:    current.label,
		Index:    index,
		Total:    len(order),
		Done:     current.done,
		Items:    current.total,
		Bytes:    current.bytes,
		Freed:    current.freed,
	}
}

func progressStageCardValue(progress progressModel, stage stageInfo) string {
	if stage.Total == 0 {
		return "Idle"
	}
	label := stage.Group
	if label == "" {
		label = sectionTitle(domain.ExecutionPlan{}, stage.Category)
	}
	prefix := "stage"
	if progress.plan.Command == "uninstall" {
		prefix = "target"
	}
	return fmt.Sprintf("%s %d/%d %s", prefix, stage.Index, stage.Total, truncateText(label, 12))
}

func progressExecutionRail(progress progressModel) string {
	completed, deleted, failed, skipped, protected := countResultStatuses(domain.ExecutionResult{Items: progress.items})
	parts := []string{"Settled"}
	switch progress.plan.Command {
	case "clean":
		if sections := progressSettledSectionCount(progress); sections > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", sections, pl(sections, "section", "sections")))
		}
		if deleted > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", deleted, progressDeletedLabel(progress.plan)))
		}
		if completed > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", completed, progressCompletedLabel(progress.plan)))
		}
		if freed := progressFreedBytes(progress); freed > 0 {
			parts = append(parts, domain.HumanBytes(freed)+" freed")
		}
	case "uninstall":
		native, removed, aftercare := uninstallSettledBuckets(progress)
		if native > 0 {
			parts = append(parts, fmt.Sprintf("%d native", native))
		}
		if removed > 0 {
			parts = append(parts, fmt.Sprintf("%d removed", removed))
		}
		if aftercare > 0 {
			parts = append(parts, fmt.Sprintf("%d aftercare", aftercare))
		}
	case "optimize", "autofix":
		if phaseRail := progressSettledPhaseRail(progress); phaseRail != "" {
			parts = append(parts, phaseRail)
		}
		if completed > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", completed, progressCompletedLabel(progress.plan)))
		}
		if deleted > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", deleted, progressDeletedLabel(progress.plan)))
		}
	default:
		if completed > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", completed, progressCompletedLabel(progress.plan)))
		}
		if deleted > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", deleted, progressDeletedLabel(progress.plan)))
		}
	}
	if protected > 0 {
		parts = append(parts, fmt.Sprintf("%d protected", protected))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}
	if len(parts) == 1 {
		parts = append(parts, "no completed operations yet")
	}
	return strings.Join(parts, "  •  ")
}

func progressStageCardLabel(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "uninstall":
		return "target"
	case "optimize", "autofix":
		return "task"
	default:
		return "scope"
	}
}

func progressSettledCardLabel(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "optimize", "autofix":
		return "done"
	default:
		return "settled"
	}
}

func progressStageDetailLabel(progress progressModel) string {
	switch progress.plan.Command {
	case "clean":
		return "section"
	case "uninstall":
		if progress.current != nil {
			if progress.current.Action == domain.ActionNative {
				return "handoff"
			}
			if phase := strings.TrimSpace(progress.current.TaskPhase); phase != "" {
				return phase
			}
		}
		return "target"
	case "optimize", "autofix":
		if phase := progressActiveTaskPhase(progress); phase != "" {
			return phase
		}
		return "phase"
	default:
		return "stage"
	}
}

func progressActiveTaskPhase(progress progressModel) string {
	if progress.current != nil && strings.TrimSpace(progress.current.TaskPhase) != "" {
		return strings.TrimSpace(progress.current.TaskPhase)
	}
	if strings.TrimSpace(progress.currentStep) != "" && strings.TrimSpace(progress.currentStep) != "run" {
		return strings.TrimSpace(progress.currentStep)
	}
	order, buckets := progressStageBuckets(progress)
	stage := progressStageInfo(progress)
	if len(order) == 0 || stage.Index <= 0 || stage.Index > len(order) {
		return ""
	}
	if bucket := buckets[order[stage.Index-1]]; bucket != nil {
		return strings.ToLower(strings.TrimSpace(bucket.label))
	}
	return ""
}

func progressSettledSectionCount(progress progressModel) int {
	order, buckets := progressStageBuckets(progress)
	count := 0
	for _, key := range order {
		bucket := buckets[key]
		if bucket != nil && bucket.total > 0 && bucket.done >= bucket.total {
			count++
		}
	}
	return count
}

func uninstallSettledBuckets(progress progressModel) (native, removed, aftercare int) {
	for idx, result := range progress.items {
		if idx >= len(progress.plan.Items) {
			break
		}
		item := progress.plan.Items[idx]
		switch {
		case item.Action == domain.ActionNative && result.Status == domain.StatusCompleted:
			native++
		case item.Action == domain.ActionCommand && strings.TrimSpace(item.TaskPhase) != "" && result.Status == domain.StatusCompleted:
			aftercare++
		case result.Status == domain.StatusDeleted:
			removed++
		}
	}
	return native, removed, aftercare
}

func progressSettledPhaseRail(progress progressModel) string {
	counts := map[string]int{}
	order := make([]string, 0)
	for idx, result := range progress.items {
		if idx >= len(progress.plan.Items) {
			break
		}
		if result.Status != domain.StatusCompleted && result.Status != domain.StatusDeleted {
			continue
		}
		phase := strings.TrimSpace(progress.plan.Items[idx].TaskPhase)
		if phase == "" {
			continue
		}
		if _, ok := counts[phase]; !ok {
			order = append(order, phase)
		}
		counts[phase]++
	}
	if len(order) == 0 {
		return ""
	}
	parts := make([]string, 0, len(order))
	for _, phase := range order {
		parts = append(parts, fmt.Sprintf("%s %d", phase, counts[phase]))
	}
	return strings.Join(parts, "  •  ")
}

func progressCompletedLabel(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "uninstall":
		return "native"
	case "optimize", "autofix":
		return "applied"
	default:
		return "completed"
	}
}

func progressDeletedLabel(plan domain.ExecutionPlan) string {
	switch plan.Command {
	case "clean":
		return "reclaimed"
	case "uninstall":
		return "removed"
	default:
		return "deleted"
	}
}

func progressCategoryLines(progress progressModel, width int) []string {
	order, buckets := progressStageBuckets(progress)
	lines := make([]string, 0, len(order))
	currentKey := ""
	if progress.current != nil {
		currentKey = progressGroupKey(*progress.current)
	}
	motion := progressMotionState(progress)
	for _, key := range order {
		b := buckets[key]
		isActive := currentKey == key && progressPhaseActive(progress.currentPhase)
		isDone := b.total > 0 && b.done >= b.total
		var line string
		if isActive {
			barWidth := 10
			filled := 0
			if b.total > 0 {
				filled = barWidth * b.done / b.total
			}
			var bar string
			if filled <= 0 {
				bar = strings.Repeat("░", barWidth)
			} else if filled >= barWidth {
				bar = strings.Repeat("▓", barWidth)
			} else {
				edges := []string{"▒", "▓", "█", "▓"}
				edge := edges[motion.Frame%len(edges)]
				bar = strings.Repeat("▓", filled-1) + edge + strings.Repeat("░", barWidth-filled)
			}
			bytesLabel := fmt.Sprintf("%s / %s", domain.HumanBytes(b.freed), domain.HumanBytes(b.bytes))
			line = fmt.Sprintf("%s %-20s [%s] %d/%d  %s",
				spinnerGlyph(motion), b.label, bar, b.done, b.total, bytesLabel)
			lines = append(lines, wrapText(reviewStyle.Render(line), width))
		} else if isDone {
			line = fmt.Sprintf("✓ %-20s %d/%d  %s freed", b.label, b.done, b.total, domain.HumanBytes(b.freed))
			lines = append(lines, wrapText(safeStyle.Render(line), width))
		} else {
			line = fmt.Sprintf("· %-20s %d/%d  %s", b.label, b.done, b.total, domain.HumanBytes(b.bytes))
			lines = append(lines, wrapText(mutedStyle.Render(line), width))
		}
	}
	return lines
}

func progressPhaseActive(phase domain.ProgressPhase) bool {
	switch phase {
	case domain.ProgressPhaseStarting, domain.ProgressPhasePreparing, domain.ProgressPhaseRunning, domain.ProgressPhaseVerifying:
		return true
	default:
		return false
	}
}

func progressPhaseDisplay(phase domain.ProgressPhase, fallback string) string {
	switch phase {
	case domain.ProgressPhaseStarting:
		return "Queued"
	case domain.ProgressPhasePreparing:
		return "Preparing"
	case domain.ProgressPhaseRunning:
		return "Running"
	case domain.ProgressPhaseVerifying:
		return "Verifying"
	case domain.ProgressPhaseFinished:
		return "Settled"
	default:
		return titleCase(fallback)
	}
}

func progressPhaseSubtitle(phase domain.ProgressPhase) string {
	switch phase {
	case domain.ProgressPhasePreparing:
		return "preparing"
	case domain.ProgressPhaseRunning:
		return "running"
	case domain.ProgressPhaseVerifying:
		return "verifying"
	case domain.ProgressPhaseFinished:
		return "settled"
	default:
		return "queued"
	}
}

func progressStageBuckets(progress progressModel) ([]string, map[string]*stageBucket) {
	order := make([]string, 0)
	buckets := map[string]*stageBucket{}
	for _, item := range progress.plan.Items {
		key := progressGroupKey(item)
		if _, ok := buckets[key]; !ok {
			order = append(order, key)
			buckets[key] = &stageBucket{
				category: item.Category,
				label:    domain.ExecutionGroupLabel(item),
			}
		}
		buckets[key].total++
		buckets[key].bytes += item.Bytes
	}
	for idx := range progress.items {
		if idx >= len(progress.plan.Items) {
			break
		}
		planItem := progress.plan.Items[idx]
		result := progress.items[idx]
		bucket := buckets[progressGroupKey(planItem)]
		if bucket == nil {
			continue
		}
		bucket.done++
		if result.Status == domain.StatusDeleted || result.Status == domain.StatusCompleted {
			bucket.freed += planItem.Bytes
		}
	}
	return order, buckets
}

func progressModuleFlowLines(progress progressModel, width int) []string {
	order, buckets := progressStageBuckets(progress)
	stage := progressStageInfo(progress)
	if len(order) == 0 || stage.Total == 0 {
		return []string{mutedStyle.Render("Waiting for the first cleanup module.")}
	}
	lines := make([]string, 0, 3)
	appendLine := func(prefix string, key string, isActive bool, tone func(...string) string) {
		bucket := buckets[key]
		if bucket == nil {
			return
		}
		label := bucket.label
		if label == "" {
			label = sectionTitle(progress.plan, bucket.category)
		}
		var bytesLabel string
		if isActive {
			bytesLabel = fmt.Sprintf("%s / %s", domain.HumanBytes(bucket.freed), domain.HumanBytes(bucket.bytes))
		} else if bucket.done >= bucket.total && bucket.total > 0 {
			bytesLabel = domain.HumanBytes(bucket.freed) + " freed"
		} else {
			bytesLabel = domain.HumanBytes(bucket.bytes)
		}
		line := fmt.Sprintf("%-6s %-22s %d/%d  %s", prefix, label, bucket.done, bucket.total, bytesLabel)
		lines = append(lines, wrapText(tone(line), width))
	}
	currentIdx := max(stage.Index-1, 0)
	if currentIdx > 0 {
		appendLine("Done", order[currentIdx-1], false, safeStyle.Render)
	}
	appendLine("Now", order[currentIdx], true, reviewStyle.Render)
	if currentIdx+1 < len(order) {
		appendLine("Next", order[currentIdx+1], false, mutedStyle.Render)
	}
	return lines
}

func progressTone(completed, failed, skipped int) string {
	if failed > 0 {
		return "high"
	}
	if completed > 0 || skipped > 0 {
		return "safe"
	}
	return "review"
}

func progressElapsed(startedAt time.Time) string {
	if startedAt.IsZero() {
		return "0s"
	}
	elapsed := time.Since(startedAt).Round(time.Second)
	if elapsed < time.Second {
		elapsed = time.Second
	}
	if elapsed < time.Minute {
		return elapsed.String()
	}
	return domain.HumanDuration(uint64(elapsed.Seconds()))
}

func progressMeter(progress progressModel) string {
	total := len(progress.plan.Items)
	if total == 0 {
		return "0%"
	}
	percent := int(float64(len(progress.items)) / float64(total) * 100)
	if percent > 100 {
		percent = 100
	}
	return fmt.Sprintf("%d%%", percent)
}

func progressMeterLine(progress progressModel) string {
	total := len(progress.plan.Items)
	if total == 0 {
		return mutedStyle.Render("[--------------------]  0%")
	}
	width := 20
	done := int(float64(len(progress.items)) / float64(total) * float64(width))
	if done > width {
		done = width
	}
	remaining := width - done
	bar := "[" + strings.Repeat("■", done) + strings.Repeat("·", remaining) + "]"
	return mutedStyle.Render(bar + "  " + progressMeter(progress))
}
