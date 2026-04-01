package cli

import (
	"context"

	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/platform"
)

var (
	warmAdminSession    = platform.WarmAdminSession
	startAdminKeepalive = platform.StartAdminKeepalive
)

func preparePlanExecution(ctx context.Context, plan domain.ExecutionPlan) (func(), error) {
	if !planNeedsAdminSession(plan) {
		return func() {}, nil
	}
	if platform.TestModeEnabled() && !platform.LiveIntegrationEnabled() {
		return func() {}, nil
	}
	if err := warmAdminSession(ctx, "SIFT needs admin access to continue this run."); err != nil {
		return nil, err
	}
	keepaliveCtx, cancel := context.WithCancel(ctx)
	startAdminKeepalive(keepaliveCtx)
	return cancel, nil
}

func planNeedsAdminSession(plan domain.ExecutionPlan) bool {
	for _, item := range plan.Items {
		if item.Status == domain.StatusProtected || item.Action == domain.ActionAdvisory {
			continue
		}
		if item.Action == domain.ActionCommand && (item.RequiresAdmin || item.CommandPath == "/usr/bin/sudo") {
			return true
		}
	}
	return false
}
