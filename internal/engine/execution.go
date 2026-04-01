package engine

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Bios-Marcel/wastebasket/v2"
	"github.com/google/uuid"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/platform"
	"github.com/batu3384/sift/internal/rules"
)

func (s *Service) ExecuteWithOptions(ctx context.Context, plan domain.ExecutionPlan, opts ExecuteOptions) (domain.ExecutionResult, error) {
	return s.ExecuteWithProgress(ctx, plan, opts, nil)
}

type executionSection struct {
	Key       string
	Label     string
	Category  domain.Category
	Index     int
	Total     int
	Items     int
	Bytes     int64
	LastItem  int
	Completed int
}

func (s *Service) ExecuteWithProgress(ctx context.Context, plan domain.ExecutionPlan, opts ExecuteOptions, emit func(domain.ExecutionProgress)) (domain.ExecutionResult, error) {
	result := domain.ExecutionResult{
		ID:        uuid.NewString(),
		ScanID:    plan.ScanID,
		StartedAt: time.Now().UTC(),
	}
	sectionsByIndex := buildExecutionSections(plan)
	emitProgress := func(item domain.Finding, op domain.OperationResult, phase domain.ProgressPhase, step, detail string, ordinal int) {
		if emit == nil {
			return
		}
		emit(domain.ExecutionProgress{
			ScanID:    plan.ScanID,
			StartedAt: result.StartedAt,
			Current:   ordinal,
			Completed: len(result.Items),
			Total:     len(plan.Items),
			Event:     domain.ProgressEventItem,
			Phase:     phase,
			Step:      step,
			Detail:    detail,
			Item:      item,
			Result:    op,
		})
	}
	emitSectionProgress := func(item domain.Finding, section *executionSection, phase domain.ProgressPhase, step, detail string, ordinal int) {
		if emit == nil || section == nil {
			return
		}
		emit(domain.ExecutionProgress{
			ScanID:       plan.ScanID,
			StartedAt:    result.StartedAt,
			Current:      ordinal,
			Completed:    len(result.Items),
			Total:        len(plan.Items),
			Event:        domain.ProgressEventSection,
			Phase:        phase,
			Step:         step,
			Detail:       detail,
			SectionKey:   section.Key,
			SectionLabel: section.Label,
			SectionIndex: section.Index,
			SectionTotal: section.Total,
			SectionDone:  section.Completed,
			SectionItems: section.Items,
			SectionBytes: section.Bytes,
			Item:         item,
		})
	}
	appendResult := func(item domain.Finding, op domain.OperationResult, ordinal int) {
		result.Items = append(result.Items, op)
		emitProgress(item, op, domain.ProgressPhaseFinished, "settle", progressResultDetail(item, op), ordinal)
	}
	for idx, item := range plan.Items {
		ordinal := idx + 1
		var section *executionSection
		if idx < len(sectionsByIndex) {
			section = sectionsByIndex[idx]
		}
		select {
		case <-ctx.Done():
			result.FinishedAt = time.Now().UTC()
			if s.Store != nil {
				_ = s.Store.SaveExecution(ctx, result)
			}
			return result, ctx.Err()
		default:
		}
		if section != nil && section.Completed == 0 {
			emitSectionProgress(item, section, domain.ProgressPhaseStarting, "section", progressSectionStartDetail(plan.Command, section), ordinal)
		}
		emitProgress(item, domain.OperationResult{}, domain.ProgressPhaseStarting, "queue", progressQueueDetail(plan.Command, item), ordinal)
		if item.Status == domain.StatusProtected || item.Action == domain.ActionAdvisory {
			appendResult(item, domain.OperationResult{
				FindingID: item.ID,
				Path:      coalesce(item.DisplayPath, item.Path),
				Status:    item.Status,
				Reason:    item.Policy.Reason,
				Message:   item.Recovery.Message,
			}, ordinal)
			if section != nil {
				section.Completed++
				if section.Completed >= section.Items {
					emitSectionProgress(item, section, domain.ProgressPhaseFinished, "section", progressSectionFinishDetail(plan.Command, section), ordinal)
				}
			}
			continue
		}
		if item.Action == domain.ActionNative {
			nativeResult := s.executeNativeItem(ctx, plan, item, opts, func(phase domain.ProgressPhase, step, detail string) {
				emitProgress(item, domain.OperationResult{}, phase, step, detail, ordinal)
			})
			appendResult(item, nativeResult, ordinal)
			if nativeResult.Status == domain.StatusCompleted {
				result.Warnings = append(result.Warnings, nativeContinuationWarning(plan))
			}
			if section != nil {
				section.Completed++
				if section.Completed >= section.Items {
					emitSectionProgress(item, section, domain.ProgressPhaseFinished, "section", progressSectionFinishDetail(plan.Command, section), ordinal)
				}
			}
			continue
		}
		if item.Action == domain.ActionCommand {
			appendResult(item, s.executeManagedCommandItem(ctx, plan, item, func(phase domain.ProgressPhase, step, detail string) {
				emitProgress(item, domain.OperationResult{}, phase, step, detail, ordinal)
			}), ordinal)
			if section != nil {
				section.Completed++
				if section.Completed >= section.Items {
					emitSectionProgress(item, section, domain.ProgressPhaseFinished, "section", progressSectionFinishDetail(plan.Command, section), ordinal)
				}
			}
			continue
		}
		decision := evaluatePolicy(plan.Policy, item, opts.Permanent)
		emitProgress(item, domain.OperationResult{}, domain.ProgressPhasePreparing, "check", progressCheckDetail(plan.Command, item), ordinal)
		if !decision.Allowed {
			appendResult(item, domain.OperationResult{
				FindingID: item.ID,
				Path:      item.Path,
				Status:    domain.StatusProtected,
				Reason:    decision.Reason,
				Message:   decision.Message,
			}, ordinal)
			if section != nil {
				section.Completed++
				if section.Completed >= section.Items {
					emitSectionProgress(item, section, domain.ProgressPhaseFinished, "section", progressSectionFinishDetail(plan.Command, section), ordinal)
				}
			}
			continue
		}
		if err := verifyFingerprint(item); err != nil {
			appendResult(item, domain.OperationResult{
				FindingID: item.ID,
				Path:      item.Path,
				Status:    domain.StatusFailed,
				Reason:    domain.ProtectionMissingPath,
				Message:   err.Error(),
			}, ordinal)
			if section != nil {
				section.Completed++
				if section.Completed >= section.Items {
					emitSectionProgress(item, section, domain.ProgressPhaseFinished, "section", progressSectionFinishDetail(plan.Command, section), ordinal)
				}
			}
			continue
		}
		if plan.DryRun {
			appendResult(item, domain.OperationResult{
				FindingID: item.ID,
				Path:      item.Path,
				Status:    domain.StatusSkipped,
				Message:   "dry-run",
			}, ordinal)
			if section != nil {
				section.Completed++
				if section.Completed >= section.Items {
					emitSectionProgress(item, section, domain.ProgressPhaseFinished, "section", progressSectionFinishDetail(plan.Command, section), ordinal)
				}
			}
			continue
		}
		var err error
		emitProgress(item, domain.OperationResult{}, domain.ProgressPhaseRunning, progressApplyStep(plan.Command, item, opts.Permanent), progressApplyDetail(plan.Command, item, opts.Permanent), ordinal)
		if opts.Permanent {
			err = removePath(item.Path)
		} else {
			err = s.moveToTrash(item.Path)
		}
		emitProgress(item, domain.OperationResult{}, domain.ProgressPhaseVerifying, "verify", progressVerifyDetail(plan.Command, item), ordinal)
		status := domain.StatusDeleted
		message := "moved to trash"
		if opts.Permanent {
			message = "deleted permanently"
		}
		if err != nil {
			status = domain.StatusFailed
			message = err.Error()
		}
		var freedBytes int64
		if status == domain.StatusDeleted {
			freedBytes = item.Bytes
		}
		appendResult(item, domain.OperationResult{
			FindingID: item.ID,
			Path:      item.Path,
			Status:    status,
			Reason:    item.Policy.Reason,
			Message:   message,
			Bytes:     freedBytes,
		}, ordinal)
		if section != nil {
			section.Completed++
			if section.Completed >= section.Items {
				emitSectionProgress(item, section, domain.ProgressPhaseFinished, "section", progressSectionFinishDetail(plan.Command, section), ordinal)
			}
		}
	}
	result.FinishedAt = time.Now().UTC()
	if plan.Command == "uninstall" && hasCompletedUninstallWork(result.Items) {
		result.FollowUpCommands = append(result.FollowUpCommands, uninstallAftermathCommands(plan)...)
	}
	result.Warnings = dedupe(result.Warnings)
	result.FollowUpCommands = dedupe(result.FollowUpCommands)
	if s.Store != nil {
		_ = s.Store.SaveExecution(ctx, result)
	}
	return result, nil
}

func buildExecutionSections(plan domain.ExecutionPlan) []*executionSection {
	switch plan.Command {
	case "clean", "uninstall", "optimize", "autofix":
	default:
		return nil
	}
	if len(plan.Items) == 0 {
		return nil
	}
	order := make([]string, 0)
	sections := map[string]*executionSection{}
	byIndex := make([]*executionSection, len(plan.Items))
	for idx, item := range plan.Items {
		key, label := executionSectionKeyLabel(plan.Command, item)
		if key == "" {
			continue
		}
		section := sections[key]
		if section == nil {
			order = append(order, key)
			section = &executionSection{
				Key:      key,
				Label:    label,
				Category: item.Category,
			}
			sections[key] = section
		}
		section.Items++
		section.Bytes += item.Bytes
		section.LastItem = idx + 1
		byIndex[idx] = section
	}
	for idx, key := range order {
		section := sections[key]
		section.Index = idx + 1
		section.Total = len(order)
	}
	return byIndex
}

func executionSectionKeyLabel(command string, item domain.Finding) (string, string) {
	switch command {
	case "uninstall":
		switch {
		case item.Action == domain.ActionNative:
			return "uninstall::handoff", "Native handoff"
		case item.Action == domain.ActionCommand && strings.TrimSpace(item.TaskPhase) != "":
			phase := strings.TrimSpace(item.TaskPhase)
			return "uninstall::phase::" + strings.ToLower(phase), strings.ToUpper(phase[:1]) + phase[1:]
		default:
			return "uninstall::remnants", "Remnants"
		}
	case "optimize", "autofix":
		if phase := strings.TrimSpace(item.TaskPhase); phase != "" {
			return command + "::phase::" + strings.ToLower(phase), strings.ToUpper(phase[:1]) + phase[1:]
		}
		if command == "autofix" {
			return "autofix::fix", "Fix"
		}
		return command + "::task", "Task"
	default:
		return domain.ExecutionGroupKey(item), domain.ExecutionGroupLabel(item)
	}
}

func progressSectionStartDetail(command string, section *executionSection) string {
	if section == nil {
		return "starting section"
	}
	label := strings.ToLower(strings.TrimSpace(section.Label))
	if label == "" {
		label = "cleanup section"
	}
	switch command {
	case "clean":
		return fmt.Sprintf("starting %s reclaim", label)
	case "uninstall":
		switch label {
		case "native handoff":
			return "starting native handoff"
		case "remnants":
			return "starting remnant cleanup"
		default:
			return "starting " + label
		}
	case "optimize", "autofix":
		return fmt.Sprintf("starting %s phase", label)
	}
	return "starting " + label
}

func progressSectionFinishDetail(command string, section *executionSection) string {
	if section == nil {
		return "section settled"
	}
	label := strings.ToLower(strings.TrimSpace(section.Label))
	if label == "" {
		label = "cleanup section"
	}
	switch command {
	case "clean":
		return fmt.Sprintf("%s settled", label)
	case "uninstall":
		switch label {
		case "native handoff":
			return "native handoff settled"
		case "remnants":
			return "remnant cleanup settled"
		default:
			return label + " settled"
		}
	case "optimize", "autofix":
		return label + " phase settled"
	}
	return label + " settled"
}

func (s *Service) executeNativeItem(ctx context.Context, plan domain.ExecutionPlan, item domain.Finding, opts ExecuteOptions, emit func(domain.ProgressPhase, string, string)) domain.OperationResult {
	result := domain.OperationResult{
		FindingID: item.ID,
		Path:      coalesce(item.DisplayPath, item.Path),
	}
	if emit != nil {
		emit(domain.ProgressPhasePreparing, "check", progressCheckDetail(plan.Command, item))
	}
	decision := evaluatePolicy(plan.Policy, item, opts.Permanent)
	if !decision.Allowed {
		result.Status = domain.StatusProtected
		result.Reason = decision.Reason
		result.Message = decision.Message
		return result
	}
	if plan.DryRun {
		result.Status = domain.StatusSkipped
		result.Message = "dry-run"
		return result
	}
	if !opts.NativeUninstall {
		result.Status = domain.StatusSkipped
		result.Message = "native uninstall disabled; rerun with --native-uninstall"
		return result
	}
	if emit != nil {
		emit(domain.ProgressPhaseRunning, "launch", "opening native uninstall")
	}
	if err := launchNativeUninstall(ctx, item); err != nil {
		result.Status = domain.StatusFailed
		result.Reason = domain.ProtectionUnsafeCommand
		result.Message = err.Error()
		return result
	}
	if emit != nil {
		emit(domain.ProgressPhaseVerifying, "handoff", "waiting for native handoff")
	}
	result.Status = domain.StatusCompleted
	result.Message = "native uninstaller launched"
	return result
}

func (s *Service) executeManagedCommandItem(ctx context.Context, plan domain.ExecutionPlan, item domain.Finding, emit func(domain.ProgressPhase, string, string)) domain.OperationResult {
	result := domain.OperationResult{
		FindingID: item.ID,
		Path:      coalesce(item.DisplayPath, item.Path),
	}
	if emit != nil {
		emit(domain.ProgressPhasePreparing, "check", progressCheckDetail(plan.Command, item))
	}
	decision := evaluatePolicy(plan.Policy, item, false)
	if !decision.Allowed {
		result.Status = domain.StatusProtected
		result.Reason = decision.Reason
		result.Message = decision.Message
		return result
	}
	if plan.DryRun {
		result.Status = domain.StatusSkipped
		result.Message = "dry-run"
		return result
	}
	if (item.RequiresAdmin || item.CommandPath == "/usr/bin/sudo") && platform.TestModeEnabled() && !platform.LiveIntegrationEnabled() {
		result.Status = domain.StatusSkipped
		result.Message = "skipped in ci-safe test mode"
		return result
	}
	if item.CommandPath == "/usr/bin/osascript" && !platform.AllowDialogSensitiveActions() {
		result.Status = domain.StatusSkipped
		result.Message = "skipped in ci-safe test mode"
		return result
	}
	if err := validateManagedCommand(item.CommandPath, item.CommandArgs); err != nil {
		result.Status = domain.StatusFailed
		result.Reason = domain.ProtectionUnsafeCommand
		result.Message = err.Error()
		return result
	}
	runCtx := ctx
	if item.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(item.TimeoutSeconds)*time.Second)
		defer cancel()
	}
	if emit != nil {
		emit(domain.ProgressPhaseRunning, progressCommandStep(item), progressManagedTaskDetail(plan.Command, item))
	}
	if err := s.execCommand(runCtx, item.CommandPath, item.CommandArgs...); err != nil {
		result.Status = domain.StatusFailed
		result.Reason = domain.ProtectionUnsafeCommand
		result.Message = err.Error()
		return result
	}
	if emit != nil {
		emit(domain.ProgressPhaseVerifying, "verify", progressManagedVerifyDetail(plan.Command, item))
	}
	result.Status = domain.StatusCompleted
	result.Message = "command completed"
	return result
}

func progressQueueDetail(command string, item domain.Finding) string {
	switch {
	case item.Action == domain.ActionNative:
		return "queued native handoff  •  " + progressLabel(item)
	case item.Action == domain.ActionCommand:
		if phase := strings.TrimSpace(item.TaskPhase); phase != "" {
			return "queued " + phase + " task  •  " + progressLabel(item)
		}
		if command == "autofix" {
			return "queued fix  •  " + progressLabel(item)
		}
		return "queued task  •  " + progressLabel(item)
	case command == "uninstall":
		return "queued remnant  •  " + progressLabel(item)
	case command == "clean":
		return "queued reclaim  •  " + progressLabel(item)
	default:
		return "queued " + progressLabel(item)
	}
}

func progressCheckDetail(command string, item domain.Finding) string {
	switch item.Action {
	case domain.ActionCommand:
		if phase := strings.TrimSpace(item.TaskPhase); phase != "" {
			return "checking " + phase + " access"
		}
		if command == "autofix" {
			return "checking fix access"
		}
		return "checking task access"
	case domain.ActionNative:
		return "checking native uninstall access"
	default:
		if command == "uninstall" {
			return "checking remnant access"
		}
		if command == "clean" {
			return "checking reclaim target"
		}
		return "checking selected item"
	}
}

func progressApplyStep(command string, item domain.Finding, permanent bool) string {
	if item.Action == domain.ActionCommand {
		if phase := strings.TrimSpace(item.TaskPhase); phase != "" {
			return phase
		}
		if command == "autofix" {
			return "fix"
		}
		return "run"
	}
	if item.Action == domain.ActionNative {
		return "handoff"
	}
	if permanent {
		return "delete"
	}
	if command == "clean" {
		return "reclaim"
	}
	if command == "uninstall" {
		return "remove"
	}
	return "trash"
}

func progressApplyDetail(command string, item domain.Finding, permanent bool) string {
	if item.Action == domain.ActionCommand {
		return progressManagedTaskDetail(command, item)
	}
	if item.Action == domain.ActionNative {
		return "opening native uninstall  •  " + progressLabel(item)
	}
	if permanent {
		return "deleting " + progressLabel(item)
	}
	if command == "clean" {
		return "reclaiming " + progressLabel(item)
	}
	if command == "uninstall" {
		return "removing remnant  •  " + progressLabel(item)
	}
	return "moving " + progressLabel(item) + " to trash"
}

func progressVerifyDetail(command string, item domain.Finding) string {
	switch item.Action {
	case domain.ActionCommand:
		return progressManagedVerifyDetail(command, item)
	case domain.ActionNative:
		return "waiting for native handoff"
	default:
		if command == "uninstall" {
			return "checking remnant result  •  " + progressLabel(item)
		}
		if command == "clean" {
			return "checking reclaim result  •  " + progressLabel(item)
		}
		return "checking result for " + progressLabel(item)
	}
}

func progressManagedTaskDetail(command string, item domain.Finding) string {
	phase := strings.TrimSpace(item.TaskPhase)
	if phase != "" {
		return phase + " task  •  " + progressLabel(item)
	}
	if command == "autofix" {
		return "running fix  •  " + progressLabel(item)
	}
	return "running task  •  " + progressLabel(item)
}

func progressManagedVerifyDetail(command string, item domain.Finding) string {
	if len(item.TaskVerify) > 0 && strings.TrimSpace(item.TaskVerify[0]) != "" {
		return item.TaskVerify[0]
	}
	if command == "autofix" {
		return "checking fix result"
	}
	return "checking task result"
}

func progressCommandStep(item domain.Finding) string {
	phase := strings.TrimSpace(item.TaskPhase)
	if phase != "" {
		return phase
	}
	return "run"
}

func progressResultDetail(item domain.Finding, op domain.OperationResult) string {
	if strings.TrimSpace(op.Message) != "" {
		return op.Message
	}
	switch op.Status {
	case domain.StatusDeleted:
		return "done"
	case domain.StatusCompleted:
		return "completed"
	case domain.StatusSkipped:
		return "skipped"
	case domain.StatusProtected:
		return "blocked"
	case domain.StatusFailed:
		return "failed"
	default:
		return progressLabel(item)
	}
}

func progressLabel(item domain.Finding) string {
	label := strings.TrimSpace(item.DisplayPath)
	if label == "" {
		label = strings.TrimSpace(item.Path)
	}
	if label == "" {
		label = strings.TrimSpace(item.Name)
	}
	if label == "" {
		label = "item"
	}
	return label
}

func calculateTotals(items []domain.Finding) domain.Totals {
	var totals domain.Totals
	for _, item := range items {
		totals.ItemCount++
		totals.Bytes += item.Bytes
		switch item.Risk {
		case domain.RiskSafe:
			totals.SafeBytes += item.Bytes
		case domain.RiskReview:
			totals.ReviewBytes += item.Bytes
		case domain.RiskHigh:
			totals.HighBytes += item.Bytes
		}
	}
	return totals
}

func (s *Service) moveToTrash(path string) error {
	if s.MoveToTrash != nil {
		return s.MoveToTrash(path)
	}
	return wastebasket.Trash(path)
}

func verifyFingerprint(item domain.Finding) error {
	current, err := currentFingerprint(item.Path)
	if err != nil {
		return err
	}
	if current.Size != item.Fingerprint.Size || !current.ModTime.Equal(item.Fingerprint.ModTime) {
		return fmt.Errorf("preview hash mismatch for %s", item.Path)
	}
	return nil
}

func currentFingerprint(path string) (domain.Fingerprint, error) {
	info, err := os.Stat(path)
	if err != nil {
		return domain.Fingerprint{}, err
	}
	if info.IsDir() {
		size, newest, err := rules.MeasurePath(context.Background(), path)
		if err != nil {
			return domain.Fingerprint{}, err
		}
		return domain.Fingerprint{
			Mode:    uint32(info.Mode()),
			Size:    size,
			ModTime: newest,
		}, nil
	}
	return domain.Fingerprint{
		Mode:    uint32(info.Mode()),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}

func removePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}

func rulesMeasurePath(ctx context.Context, path string) (int64, time.Time, error) {
	return rules.MeasurePath(ctx, path)
}
