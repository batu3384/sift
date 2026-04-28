package engine

import (
	"context"
	"fmt"

	"github.com/batu3384/sift/internal/domain"
)

func (s *Service) persistPlan(ctx context.Context, plan *domain.ExecutionPlan) {
	if s == nil || s.Store == nil || plan == nil {
		return
	}
	if err := s.Store.SavePlan(ctx, *plan); err != nil {
		plan.Warnings = dedupe(append(plan.Warnings, auditPersistenceWarning(err)))
	}
}

func (r *executionRunner) persistExecution() {
	if r == nil || r.service == nil || r.service.Store == nil {
		return
	}
	if err := r.service.Store.SaveExecution(r.ctx, r.result); err != nil {
		r.result.Warnings = dedupe(append(r.result.Warnings, auditPersistenceWarning(err)))
	}
}

func auditPersistenceWarning(err error) string {
	return fmt.Sprintf("audit persistence unavailable: %v", err)
}
