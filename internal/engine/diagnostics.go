package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/batu3384/sift/internal/parity"
	"github.com/batu3384/sift/internal/platform"
	"github.com/batu3384/sift/internal/report"
)

func (s *Service) Diagnostics(ctx context.Context) []platform.Diagnostic {
	diagnostics := append([]platform.Diagnostic{}, s.Adapter.Diagnostics(ctx)...)
	diagnostics = append(diagnostics,
		platform.Diagnostic{Name: "interaction_mode", Status: "ok", Message: s.Config.InteractionMode},
		platform.Diagnostic{Name: "trash_mode", Status: "ok", Message: s.Config.TrashMode},
		platform.Diagnostic{Name: "confirm_level", Status: "ok", Message: s.Config.ConfirmLevel},
		platform.Diagnostic{Name: "test_policy", Status: "ok", Message: testPolicyMessage()},
		platform.Diagnostic{Name: "live_integration", Status: "ok", Message: liveIntegrationMessage()},
		platform.Diagnostic{Name: "protected_paths", Status: "ok", Message: func() string {
			n := len(s.Config.ProtectedPaths)
			return fmt.Sprintf("%d user-defined %s", n, map[bool]string{true: "root", false: "roots"}[n == 1])
		}()},
		platform.Diagnostic{Name: "command_excludes", Status: "ok", Message: func() string {
			n := len(s.Config.CommandExcludes)
			return fmt.Sprintf("%d command %s", n, map[bool]string{true: "scope", false: "scopes"}[n == 1])
		}()},
		platform.Diagnostic{Name: "built_in_protected_paths", Status: "ok", Message: func() string {
			n := len(normalizePolicyPaths(s.Adapter.ProtectedPaths()))
			return fmt.Sprintf("%d built-in %s", n, map[bool]string{true: "root", false: "roots"}[n == 1])
		}()},
		platform.Diagnostic{Name: "protected_families", Status: "ok", Message: fmt.Sprintf("%d active of %d available", len(dedupeLower(s.Config.ProtectedFamilies)), len(availableProtectedFamilies(s.Adapter)))},
		platform.Diagnostic{Name: "safe_exceptions", Status: "ok", Message: func() string {
			n := len(normalizePolicyPaths(safeProtectedExceptions(s.Adapter.CuratedRoots())))
			return fmt.Sprintf("%d curated cache %s", n, map[bool]string{true: "root", false: "roots"}[n == 1])
		}()},
	)
	if len(s.Config.PurgeSearchPaths) == 0 {
		diagnostics = append(diagnostics, platform.Diagnostic{Name: "purge_search_paths", Status: "warn", Message: "no default purge discovery roots configured"})
	} else {
		n := len(s.Config.PurgeSearchPaths)
		diagnostics = append(diagnostics, platform.Diagnostic{Name: "purge_search_paths", Status: "ok", Message: fmt.Sprintf("%d configured %s", n, map[bool]string{true: "root", false: "roots"}[n == 1])})
	}
	if reportDir, err := report.Dir(); err != nil {
		diagnostics = append(diagnostics, platform.Diagnostic{Name: "report_cache", Status: "warn", Message: err.Error()})
	} else if err := os.MkdirAll(reportDir, 0o755); err != nil {
		diagnostics = append(diagnostics, platform.Diagnostic{Name: "report_cache", Status: "warn", Message: err.Error()})
	} else {
		diagnostics = append(diagnostics, platform.Diagnostic{Name: "report_cache", Status: "ok", Message: reportDir})
	}
	if s.Store != nil {
		diagnostics = append(diagnostics, platform.Diagnostic{Name: "audit_log", Status: "ok", Message: s.Store.AuditLogPath()})
	}
	diagnostics = append(diagnostics,
		s.toolDiagnostic("pwsh", "required for make smoke-windows"),
		s.toolDiagnostic("goreleaser", "required for make release-dry-run"),
	)
	if matrix, err := parity.Load(); err != nil {
		diagnostics = append(diagnostics, platform.Diagnostic{Name: "parity_matrix", Status: "warn", Message: err.Error()})
	} else {
		summary := parity.Summarize(matrix)
		status := "ok"
		if summary.Missing > 0 || summary.RegressionRisk > 0 {
			status = "warn"
		}
		diagnostics = append(diagnostics, platform.Diagnostic{
			Name:   "parity_matrix",
			Status: status,
			Message: fmt.Sprintf(
				"%d covered, %d partial, %d missing, %d regression-risk, %d better-than-mole",
				summary.Covered,
				summary.Partial,
				summary.Missing,
				summary.RegressionRisk,
				summary.BetterThanMole,
			),
		})
		baselineMessage := matrix.Upstream.BaselineCommit
		if compare := matrix.Upstream.CompareRange; compare != "" {
			baselineMessage = fmt.Sprintf("%s  •  %s", shortCommit(matrix.Upstream.BaselineCommit), compare)
		} else if matrix.Upstream.BaselineCommit != "" {
			baselineMessage = shortCommit(matrix.Upstream.BaselineCommit)
		}
		if changed := len(matrix.Upstream.ChangedFiles); changed > 0 {
			fileWord := map[bool]string{true: "file", false: "files"}[changed == 1]
			baselineMessage = fmt.Sprintf("%s  •  %d changed %s", baselineMessage, changed, fileWord)
		}
		diagnostics = append(diagnostics, platform.Diagnostic{
			Name:    "upstream_baseline",
			Status:  "ok",
			Message: baselineMessage,
		})
	}
	touchID := s.TouchIDStatus()
	status := "ok"
	if !touchID.Supported || !touchID.Enabled || touchID.MigrationNeeded {
		status = "warn"
	}
	message := touchID.Message
	if touchID.ActivePAMPath != "" {
		message = fmt.Sprintf("%s (%s)", message, touchID.ActivePAMPath)
	} else if touchID.PAMPath != "" {
		message = fmt.Sprintf("%s (%s)", message, touchID.PAMPath)
	}
	diagnostics = append(diagnostics, platform.Diagnostic{Name: "touchid", Status: status, Message: message})
	return diagnostics
}

func shortCommit(commit string) string {
	if len(commit) <= 8 {
		return commit
	}
	return commit[:8]
}

func testPolicyMessage() string {
	switch {
	case platform.TestModeEnabled() && platform.LiveIntegrationEnabled():
		return "ci-safe guards overridden by live integration"
	case platform.TestModeEnabled():
		return "ci-safe guards active"
	default:
		return "host mode active"
	}
}

func liveIntegrationMessage() string {
	if platform.LiveIntegrationEnabled() {
		return "enabled via SIFT_LIVE_INTEGRATION=1"
	}
	return "disabled by default; use make integration-live-macos or make smoke-live-macos"
}

func (s *Service) toolDiagnostic(name string, purpose string) platform.Diagnostic {
	lookPath := s.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	path, err := lookPath(name)
	if err != nil {
		return platform.Diagnostic{
			Name:    name,
			Status:  "warn",
			Message: fmt.Sprintf("missing from PATH; %s", purpose),
		}
	}
	return platform.Diagnostic{
		Name:    name,
		Status:  "ok",
		Message: path,
	}
}
