// Package routes provides route-specific update handlers for the SIFT TUI.
package routes

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/batu3384/sift/internal/domain"
)

// UpdateHandler handles route-specific keyboard updates.
type UpdateHandler func(msg tea.KeyMsg) (tea.Model, tea.Cmd)

// RouteUpdater is the interface for route update handlers.
type RouteUpdater interface {
	Update(msg tea.KeyMsg) (tea.Model, tea.Cmd)
}

// HomeActions returns the home screen actions.
type HomeAction struct {
	ID          string
	Label       string
	Description string
	ProfileKey  string
	Enabled     bool
}

// ScanProgressCallback is called during scanning progress.
type ScanProgressCallback func(ruleID, ruleName string, itemsFound int, bytesFound int64)

// ScanFindingCallback is called when a scan finding is discovered.
type ScanFindingCallback func(ruleID, ruleName string, item domain.Finding)

// PlanLoader is a function that loads an execution plan.
type PlanLoader func() (domain.ExecutionPlan, error)

// PreviewLoader is a function that loads a menu preview.
type PreviewLoader func() (domain.ExecutionPlan, error)

// MenuPreviewCallback is called when a menu preview is loaded.
type MenuPreviewCallback func(key string, plan domain.ExecutionPlan, err error)

// ExecutionResultCallback is called when execution completes.
type ExecutionResultCallback func(plan domain.ExecutionPlan, result domain.ExecutionResult)

// DashboardLoader loads dashboard data.
type DashboardLoader func() (DashboardData, error)

// DashboardData represents the home dashboard state.
type DashboardData struct {
	Health        int
	DiskFree      int64
	LastRun       string
	WatchStatus   string
	Alerts        int
	Recommendations []string
}

// UninstallItem represents an installed application.
type UninstallItem struct {
	Name    string
	Path    string
	Size    int64
	Origin  string
	Staged  bool
}

// UninstallBatchLoader loads a batch uninstall plan.
type UninstallBatchLoader func(names []string) (domain.ExecutionPlan, error)

// ProtectPath represents a protected path entry.
type ProtectPath struct {
	Path      string
	Protected bool
	Reason    string
}

// AutofixLoader loads an autofix plan.
type AutofixLoader func() (domain.ExecutionPlan, error)

// OptimizeLoader loads an optimize plan.
type OptimizeLoader func() (domain.ExecutionPlan, error)