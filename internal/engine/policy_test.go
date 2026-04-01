package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func TestEvaluatePolicyRejectsOutsideAllowedRoots(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "other", "cache.bin")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	item := domain.Finding{
		Path:   path,
		Action: domain.ActionTrash,
	}
	decision := evaluatePolicy(domain.ProtectionPolicy{
		AllowedRoots:  []string{filepath.Join(root, "allowed")},
		BlockSymlinks: true,
	}, item, false)
	if decision.Allowed {
		t.Fatal("expected outside allowed roots to be rejected")
	}
	if decision.Reason != domain.ProtectionOutsideAllowedRoots {
		t.Fatalf("expected outside allowed roots reason, got %s", decision.Reason)
	}
}

func TestEvaluatePolicyRejectsCriticalRoot(t *testing.T) {
	t.Parallel()
	item := domain.Finding{
		Path:   string(filepath.Separator),
		Action: domain.ActionTrash,
	}
	decision := evaluatePolicy(domain.ProtectionPolicy{BlockSymlinks: true}, item, false)
	if decision.Allowed {
		t.Fatal("expected critical root to be rejected")
	}
	if decision.Reason != domain.ProtectionCriticalRoot {
		t.Fatalf("expected critical root reason, got %s", decision.Reason)
	}
}

func TestEvaluatePolicyRejectsUnsafeNativeCommand(t *testing.T) {
	t.Parallel()
	item := domain.Finding{
		Action:        domain.ActionNative,
		NativeCommand: "cleanup-helper --silent",
	}
	decision := evaluatePolicy(domain.ProtectionPolicy{}, item, false)
	if decision.Allowed {
		t.Fatal("expected unsafe native command to be rejected")
	}
	if decision.Reason != domain.ProtectionUnsafeCommand {
		t.Fatalf("expected unsafe command reason, got %s", decision.Reason)
	}
}

func TestEvaluatePolicyAllowsManagedCommand(t *testing.T) {
	t.Parallel()
	item := domain.Finding{
		Action:      domain.ActionCommand,
		CommandPath: "/usr/bin/true",
		CommandArgs: []string{"--version"},
	}
	decision := evaluatePolicy(domain.ProtectionPolicy{}, item, false)
	if !decision.Allowed {
		t.Fatalf("expected managed command to be allowed, got %+v", decision)
	}
}

func TestEvaluatePolicyRejectsUnsafeManagedCommand(t *testing.T) {
	t.Parallel()
	item := domain.Finding{
		Action:      domain.ActionCommand,
		CommandPath: "true",
	}
	decision := evaluatePolicy(domain.ProtectionPolicy{}, item, false)
	if decision.Allowed {
		t.Fatal("expected bare managed command to be rejected")
	}
	if decision.Reason != domain.ProtectionUnsafeCommand {
		t.Fatalf("expected unsafe command reason, got %s", decision.Reason)
	}
}

func TestEvaluatePolicyRejectsCommandScopedExclude(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "cache", "blob.bin")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	item := domain.Finding{
		Path:   path,
		Action: domain.ActionTrash,
	}
	decision := evaluatePolicy(domain.ProtectionPolicy{
		Command:               "clean",
		CommandProtectedPaths: []string{filepath.Join(root, "cache")},
		AllowedRoots:          []string{root},
		BlockSymlinks:         true,
	}, item, false)
	if decision.Allowed {
		t.Fatal("expected command-scoped exclude to be rejected")
	}
	if decision.Reason != domain.ProtectionCommandExcluded {
		t.Fatalf("expected command excluded reason, got %s", decision.Reason)
	}
}
