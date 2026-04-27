package engine

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/batu3384/sift/internal/domain"
)

type executionRunner struct {
	service         *Service
	ctx             context.Context
	plan            domain.ExecutionPlan
	opts            ExecuteOptions
	emit            func(domain.ExecutionProgress)
	result          domain.ExecutionResult
	sectionsByIndex []*executionSection
}

func newExecutionRunner(service *Service, ctx context.Context, plan domain.ExecutionPlan, opts ExecuteOptions, emit func(domain.ExecutionProgress)) *executionRunner {
	return &executionRunner{
		service: service,
		ctx:     ctx,
		plan:    plan,
		opts:    opts,
		emit:    emit,
		result: domain.ExecutionResult{
			ID:         uuid.NewString(),
			ScanID:     plan.ScanID,
			PlanDigest: domain.PlanDigest(plan),
			StartedAt:  time.Now().UTC(),
		},
		sectionsByIndex: buildExecutionSections(plan),
	}
}

func (r *executionRunner) run() (domain.ExecutionResult, error) {
	for idx, item := range r.plan.Items {
		if err := r.ensureContext(); err != nil {
			return r.cancel(err)
		}
		r.executeItem(idx, item)
	}
	r.result.FinishedAt = time.Now().UTC()
	if r.plan.Command == "uninstall" && hasCompletedUninstallWork(r.result.Items) {
		r.result.FollowUpCommands = append(r.result.FollowUpCommands, uninstallAftermathCommands(r.plan)...)
	}
	r.result.Warnings = dedupe(r.result.Warnings)
	r.result.FollowUpCommands = dedupe(r.result.FollowUpCommands)
	if r.service.Store != nil {
		_ = r.service.Store.SaveExecution(r.ctx, r.result)
	}
	return r.result, nil
}

func (r *executionRunner) ensureContext() error {
	select {
	case <-r.ctx.Done():
		return r.ctx.Err()
	default:
		return nil
	}
}

func (r *executionRunner) cancel(err error) (domain.ExecutionResult, error) {
	r.result.FinishedAt = time.Now().UTC()
	if r.service.Store != nil {
		_ = r.service.Store.SaveExecution(r.ctx, r.result)
	}
	return r.result, err
}

func (r *executionRunner) executeItem(idx int, item domain.Finding) {
	ordinal := idx + 1
	section := r.sectionForIndex(idx)
	r.beginSection(section, item, ordinal)
	r.emitItemProgress(item, domain.OperationResult{}, domain.ProgressPhaseStarting, "queue", progressQueueDetail(r.plan.Command, item), ordinal)

	switch {
	case item.Status == domain.StatusProtected || item.Action == domain.ActionAdvisory:
		r.appendResult(item, domain.OperationResult{
			FindingID: item.ID,
			Path:      coalesce(item.DisplayPath, item.Path),
			Status:    item.Status,
			Reason:    item.Policy.Reason,
			Message:   item.Recovery.Message,
		}, ordinal)
	case item.Action == domain.ActionNative:
		nativeResult := r.service.executeNativeItem(r.ctx, r.plan, item, r.opts, func(phase domain.ProgressPhase, step, detail string) {
			r.emitItemProgress(item, domain.OperationResult{}, phase, step, detail, ordinal)
		})
		r.appendResult(item, nativeResult, ordinal)
		if nativeResult.Status == domain.StatusCompleted {
			r.result.Warnings = append(r.result.Warnings, nativeContinuationWarning(r.plan))
		}
	case item.Action == domain.ActionCommand:
		commandResult := r.service.executeManagedCommandItem(r.ctx, r.plan, item, func(phase domain.ProgressPhase, step, detail string) {
			r.emitItemProgress(item, domain.OperationResult{}, phase, step, detail, ordinal)
		})
		r.appendResult(item, commandResult, ordinal)
	default:
		r.executePathItem(item, ordinal)
	}

	r.finishSection(section, item, ordinal)
}

func (r *executionRunner) executePathItem(item domain.Finding, ordinal int) {
	decision := evaluatePolicy(r.plan.Policy, item, r.opts.Permanent)
	r.emitItemProgress(item, domain.OperationResult{}, domain.ProgressPhasePreparing, "check", progressCheckDetail(r.plan.Command, item), ordinal)
	if !decision.Allowed {
		r.appendResult(item, domain.OperationResult{
			FindingID: item.ID,
			Path:      item.Path,
			Status:    domain.StatusProtected,
			Reason:    decision.Reason,
			Message:   decision.Message,
		}, ordinal)
		return
	}
	if err := verifyFingerprint(item); err != nil {
		r.appendResult(item, domain.OperationResult{
			FindingID: item.ID,
			Path:      item.Path,
			Status:    domain.StatusFailed,
			Reason:    domain.ProtectionMissingPath,
			Message:   err.Error(),
		}, ordinal)
		return
	}
	if r.plan.DryRun {
		r.appendResult(item, domain.OperationResult{
			FindingID: item.ID,
			Path:      item.Path,
			Status:    domain.StatusSkipped,
			Message:   "dry-run",
		}, ordinal)
		return
	}

	r.emitItemProgress(item, domain.OperationResult{}, domain.ProgressPhaseRunning, progressApplyStep(r.plan.Command, item, r.opts.Permanent), progressApplyDetail(r.plan.Command, item, r.opts.Permanent), ordinal)
	err := r.applyPathItem(item)
	r.emitItemProgress(item, domain.OperationResult{}, domain.ProgressPhaseVerifying, "verify", progressVerifyDetail(r.plan.Command, item), ordinal)

	status := domain.StatusDeleted
	message := "moved to trash"
	if r.opts.Permanent {
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
	r.appendResult(item, domain.OperationResult{
		FindingID: item.ID,
		Path:      item.Path,
		Status:    status,
		Reason:    item.Policy.Reason,
		Message:   message,
		Bytes:     freedBytes,
	}, ordinal)
}

func (r *executionRunner) applyPathItem(item domain.Finding) error {
	if r.opts.Permanent {
		return removePath(item.Path)
	}
	return r.service.moveToTrash(item.Path)
}

func (r *executionRunner) sectionForIndex(idx int) *executionSection {
	if idx < 0 || idx >= len(r.sectionsByIndex) {
		return nil
	}
	return r.sectionsByIndex[idx]
}

func (r *executionRunner) beginSection(section *executionSection, item domain.Finding, ordinal int) {
	if section == nil || section.Completed != 0 {
		return
	}
	r.emitSectionProgress(item, section, domain.ProgressPhaseStarting, "section", progressSectionStartDetail(r.plan.Command, section), ordinal)
}

func (r *executionRunner) finishSection(section *executionSection, item domain.Finding, ordinal int) {
	if section == nil {
		return
	}
	section.Completed++
	if section.Completed >= section.Items {
		r.emitSectionProgress(item, section, domain.ProgressPhaseFinished, "section", progressSectionFinishDetail(r.plan.Command, section), ordinal)
	}
}

func (r *executionRunner) appendResult(item domain.Finding, op domain.OperationResult, ordinal int) {
	r.result.Items = append(r.result.Items, op)
	r.emitItemProgress(item, op, domain.ProgressPhaseFinished, "settle", progressResultDetail(item, op), ordinal)
}

func (r *executionRunner) emitItemProgress(item domain.Finding, op domain.OperationResult, phase domain.ProgressPhase, step, detail string, ordinal int) {
	if r.emit == nil {
		return
	}
	r.emit(domain.ExecutionProgress{
		ScanID:    r.plan.ScanID,
		StartedAt: r.result.StartedAt,
		Current:   ordinal,
		Completed: len(r.result.Items),
		Total:     len(r.plan.Items),
		Event:     domain.ProgressEventItem,
		Phase:     phase,
		Step:      step,
		Detail:    detail,
		Item:      item,
		Result:    op,
	})
}

func (r *executionRunner) emitSectionProgress(item domain.Finding, section *executionSection, phase domain.ProgressPhase, step, detail string, ordinal int) {
	if r.emit == nil || section == nil {
		return
	}
	r.emit(domain.ExecutionProgress{
		ScanID:       r.plan.ScanID,
		StartedAt:    r.result.StartedAt,
		Current:      ordinal,
		Completed:    len(r.result.Items),
		Total:        len(r.plan.Items),
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
