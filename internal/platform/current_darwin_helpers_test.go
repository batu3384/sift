//go:build darwin

package platform

import (
	"errors"
	"strings"
	"testing"
)

func TestDarwinResolveTargetsExpandsHomeShortcuts(t *testing.T) {
	t.Parallel()

	adapter := darwinAdapter{home: "/Users/tester"}
	got := adapter.ResolveTargets([]string{"~", "~/Library/Caches", "/tmp/file"})
	if len(got) != 3 {
		t.Fatalf("expected 3 resolved targets, got %v", got)
	}
	if got[0] != "/Users/tester" || got[1] != "/Users/tester/Library/Caches" || got[2] != "/tmp/file" {
		t.Fatalf("unexpected resolved targets: %v", got)
	}
}

func TestDarwinIsAdminPathRecognizesProtectedPrefixes(t *testing.T) {
	t.Parallel()

	adapter := darwinAdapter{}
	if !adapter.IsAdminPath("/Applications/Example.app") {
		t.Fatal("expected /Applications path to require admin")
	}
	if !adapter.IsAdminPath("/Library/Application Support/Example") {
		t.Fatal("expected /Library path to require admin")
	}
	if adapter.IsAdminPath("/Users/tester/Library/Caches/Example") {
		t.Fatal("did not expect user cache path to require admin")
	}
}

func TestDarwinGitIdentityDiagnosticHandlesPartialIdentity(t *testing.T) {
	t.Parallel()

	diag := darwinGitIdentityDiagnostic("Batuhan", nil, "", nil)
	if diag.Name != "git_identity" || diag.Status != "warn" {
		t.Fatalf("expected partial git identity warning, got %+v", diag)
	}
	if !strings.Contains(diag.Message, "user.email") {
		t.Fatalf("expected missing user.email message, got %+v", diag)
	}
}

func TestDarwinLoginItemsDiagnosticWarnsAndTruncatesPreview(t *testing.T) {
	t.Parallel()

	output := "One, Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten, Eleven, Twelve, Thirteen, Fourteen, Fifteen, Sixteen"
	diag := darwinLoginItemsDiagnostic(output, nil)
	if diag.Status != "warn" {
		t.Fatalf("expected login items warning for large set, got %+v", diag)
	}
	if !strings.Contains(diag.Message, "16 apps") || !strings.Contains(diag.Message, "One, Two, Three +13") {
		t.Fatalf("expected truncated preview, got %+v", diag)
	}
}

func TestDarwinFirewallDiagnosticUnavailableFallsBackToStatusUnavailable(t *testing.T) {
	t.Parallel()

	diag := darwinFirewallDiagnostic("", errors.New("socketfilterfw missing"))
	if diag.Name != "firewall" || diag.Status != "warn" || diag.Message != "status unavailable" {
		t.Fatalf("expected unavailable firewall diagnostic, got %+v", diag)
	}
}

func TestFirstDiagnosticLineUsesFallbackForBlankOutput(t *testing.T) {
	t.Parallel()

	if got := firstDiagnosticLine("\n \n", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback line, got %q", got)
	}
	if got := firstDiagnosticLine("\nsecond\nthird", "fallback"); got != "second" {
		t.Fatalf("expected first non-empty line, got %q", got)
	}
}
