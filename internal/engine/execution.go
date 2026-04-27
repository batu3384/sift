package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bios-Marcel/wastebasket/v2"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/platform"
	"github.com/batu3384/sift/internal/rules"
)

func (s *Service) ExecuteWithOptions(ctx context.Context, plan domain.ExecutionPlan, opts ExecuteOptions) (domain.ExecutionResult, error) {
	return s.ExecuteWithProgress(ctx, plan, opts, nil)
}

func (s *Service) ExecuteWithProgress(ctx context.Context, plan domain.ExecutionPlan, opts ExecuteOptions, emit func(domain.ExecutionProgress)) (domain.ExecutionResult, error) {
	return newExecutionRunner(s, ctx, plan, opts, emit).run()
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
	if managedCommandBlockedInTestMode(item) {
		result.Status = domain.StatusSkipped
		result.Message = "skipped in ci-safe test mode"
		return result
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

func managedCommandBlockedInTestMode(item domain.Finding) bool {
	if !platform.TestModeEnabled() || platform.LiveIntegrationEnabled() {
		return false
	}
	if item.RequiresAdmin || isSudoManagedCommand(item.CommandPath) {
		return true
	}
	return isDialogSensitiveManagedCommand(item.CommandPath) && !platform.AllowDialogSensitiveActions()
}

func isSudoManagedCommand(path string) bool {
	base := strings.TrimSuffix(strings.ToLower(filepath.Base(path)), ".exe")
	return base == "sudo"
}

func isDialogSensitiveManagedCommand(path string) bool {
	base := strings.TrimSuffix(strings.ToLower(filepath.Base(path)), ".exe")
	return base == "osascript"
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
	if current.Mode != item.Fingerprint.Mode || current.Size != item.Fingerprint.Size || !current.ModTime.Equal(item.Fingerprint.ModTime) {
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
