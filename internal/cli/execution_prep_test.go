package cli

import (
	"context"
	"testing"

	"github.com/batu3384/sift/internal/domain"
)

func TestPlanNeedsAdminSessionOnlyForExecutableAdminCommands(t *testing.T) {
	plan := domain.ExecutionPlan{
		Items: []domain.Finding{
			{Action: domain.ActionAdvisory, Status: domain.StatusAdvisory, RequiresAdmin: true},
			{Action: domain.ActionCommand, Status: domain.StatusPlanned, CommandPath: "/usr/bin/sudo", RequiresAdmin: true},
			{Action: domain.ActionTrash, Status: domain.StatusPlanned, RequiresAdmin: true},
		},
	}
	if !planNeedsAdminSession(plan) {
		t.Fatal("expected admin session to be required for executable admin command")
	}
	if planNeedsAdminSession(domain.ExecutionPlan{
		Items: []domain.Finding{
			{Action: domain.ActionTrash, Status: domain.StatusPlanned, RequiresAdmin: true},
			{Action: domain.ActionAdvisory, Status: domain.StatusAdvisory, RequiresAdmin: true},
		},
	}) {
		t.Fatal("did not expect admin session for non-command items only")
	}
}

func TestPreparePlanExecutionWarmsAdminAndStartsKeepalive(t *testing.T) {
	originalWarm := warmAdminSession
	originalKeepalive := startAdminKeepalive
	defer func() {
		warmAdminSession = originalWarm
		startAdminKeepalive = originalKeepalive
	}()

	warmed := 0
	keepalive := 0
	warmAdminSession = func(context.Context, string) error {
		warmed++
		return nil
	}
	startAdminKeepalive = func(context.Context) {
		keepalive++
	}

	release, err := preparePlanExecution(context.Background(), domain.ExecutionPlan{
		Items: []domain.Finding{{
			Action:        domain.ActionCommand,
			Status:        domain.StatusPlanned,
			CommandPath:   "/usr/bin/sudo",
			RequiresAdmin: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	release()
	if warmed != 1 {
		t.Fatalf("expected one admin warmup, got %d", warmed)
	}
	if keepalive != 1 {
		t.Fatalf("expected one keepalive start, got %d", keepalive)
	}
}

func TestPreparePlanExecutionSkipsWarmupInCiSafeMode(t *testing.T) {
	t.Setenv("SIFT_TEST_MODE", "ci-safe")

	originalWarm := warmAdminSession
	originalKeepalive := startAdminKeepalive
	defer func() {
		warmAdminSession = originalWarm
		startAdminKeepalive = originalKeepalive
	}()

	warmed := 0
	keepalive := 0
	warmAdminSession = func(context.Context, string) error {
		warmed++
		return nil
	}
	startAdminKeepalive = func(context.Context) {
		keepalive++
	}

	release, err := preparePlanExecution(context.Background(), domain.ExecutionPlan{
		Items: []domain.Finding{{
			Action:        domain.ActionCommand,
			Status:        domain.StatusPlanned,
			CommandPath:   "/usr/bin/sudo",
			RequiresAdmin: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	release()
	if warmed != 0 || keepalive != 0 {
		t.Fatalf("expected ci-safe mode to skip admin warmup, got warmed=%d keepalive=%d", warmed, keepalive)
	}
}
