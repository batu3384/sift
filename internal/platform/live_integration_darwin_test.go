//go:build darwin

package platform

import (
	"context"
	"testing"
)

func TestLiveIntegrationLoginItemsDiagnostic(t *testing.T) {
	t.Parallel()
	if !LiveIntegrationEnabled() {
		t.Skip("set SIFT_LIVE_INTEGRATION=1 to run live macOS integration tests")
	}

	diagnostic := darwinLoginItemsDiagnosticForMode(context.Background())
	if diagnostic.Name != "login_items" {
		t.Fatalf("expected login_items diagnostic, got %+v", diagnostic)
	}
	if diagnostic.Status == "" {
		t.Fatalf("expected login_items diagnostic status, got %+v", diagnostic)
	}
	if diagnostic.Message == "" {
		t.Fatalf("expected login_items diagnostic message, got %+v", diagnostic)
	}
}
