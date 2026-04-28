package cli

import (
	"context"
	"testing"

	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/engine"
	"github.com/batu3384/sift/internal/platform"
)

type cliUninstallCaptureAdapter struct {
	allowAdminCalls []bool
}

func (a *cliUninstallCaptureAdapter) Name() string { return "test" }
func (a *cliUninstallCaptureAdapter) CuratedRoots() platform.CuratedRoots {
	return platform.CuratedRoots{}
}
func (a *cliUninstallCaptureAdapter) ProtectedPaths() []string { return nil }
func (a *cliUninstallCaptureAdapter) ResolveTargets(in []string) []string {
	return append([]string{}, in...)
}
func (a *cliUninstallCaptureAdapter) ListApps(_ context.Context, allowAdmin bool) ([]domain.AppEntry, error) {
	a.allowAdminCalls = append(a.allowAdminCalls, allowAdmin)
	return []domain.AppEntry{{Name: "Example", DisplayName: "Example"}}, nil
}
func (a *cliUninstallCaptureAdapter) DiscoverRemnants(context.Context, domain.AppEntry) ([]string, []string, error) {
	return nil, nil, nil
}
func (a *cliUninstallCaptureAdapter) MaintenanceTasks(context.Context) []domain.MaintenanceTask {
	return nil
}
func (a *cliUninstallCaptureAdapter) Diagnostics(context.Context) []platform.Diagnostic { return nil }
func (a *cliUninstallCaptureAdapter) IsAdminPath(string) bool                           { return false }
func (a *cliUninstallCaptureAdapter) IsFileInUse(context.Context, string) bool          { return false }
func (a *cliUninstallCaptureAdapter) IsProcessRunning(...string) bool                   { return false }

func TestUninstallCommandRespectsGlobalAdminFlag(t *testing.T) {
	adapter := &cliUninstallCaptureAdapter{}
	state := &runtimeState{
		cfg: config.Default(),
		service: &engine.Service{
			Adapter: adapter,
			Config:  config.Default(),
		},
		flags: globalOptions{DryRun: true, Plain: true},
	}
	cmd := newUninstallCommand(state)
	cmd.SetContext(context.Background())
	if err := cmd.RunE(cmd, []string{"Example"}); err != nil {
		t.Fatal(err)
	}
	if len(adapter.allowAdminCalls) != 1 || adapter.allowAdminCalls[0] {
		t.Fatalf("expected uninstall preview without --admin to call ListApps(false), got %v", adapter.allowAdminCalls)
	}

	state.flags.Admin = true
	if err := cmd.RunE(cmd, []string{"Example"}); err != nil {
		t.Fatal(err)
	}
	if len(adapter.allowAdminCalls) != 2 || !adapter.allowAdminCalls[1] {
		t.Fatalf("expected uninstall preview with --admin to call ListApps(true), got %v", adapter.allowAdminCalls)
	}
}
