//go:build !darwin && !windows

package platform

import (
	"context"

	"github.com/batu3384/sift/internal/domain"
)

type unsupportedAdapter struct{}

func Current() Adapter {
	return unsupportedAdapter{}
}

func (unsupportedAdapter) Name() string { return "unsupported" }
func (unsupportedAdapter) CuratedRoots() CuratedRoots {
	return CuratedRoots{}
}
func (unsupportedAdapter) ProtectedPaths() []string { return nil }
func (unsupportedAdapter) ResolveTargets(in []string) []string {
	return in
}
func (unsupportedAdapter) ListApps(context.Context, bool) ([]domain.AppEntry, error) {
	return nil, nil
}
func (unsupportedAdapter) DiscoverRemnants(context.Context, domain.AppEntry) ([]string, []string, error) {
	return nil, nil, nil
}
func (unsupportedAdapter) MaintenanceTasks(context.Context) []domain.MaintenanceTask {
	return nil
}
func (unsupportedAdapter) Diagnostics(context.Context) []Diagnostic {
	return []Diagnostic{{Name: "platform", Status: "warn", Message: "unsupported platform"}}
}
func (unsupportedAdapter) IsAdminPath(string) bool { return false }
func (unsupportedAdapter) IsFileInUse(context.Context, string) bool { return false }
func (unsupportedAdapter) IsProcessRunning(...string) bool { return false }
