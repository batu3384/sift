//go:build darwin

package engine

import (
	"context"
	"testing"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/platform"
)

func TestLiveIntegrationExecuteManagedCommandItemRunsAppleScript(t *testing.T) {
	t.Parallel()
	if !platform.LiveIntegrationEnabled() {
		t.Skip("set SIFT_LIVE_INTEGRATION=1 to run live macOS integration tests")
	}

	service := &Service{}
	result := service.executeManagedCommandItem(context.Background(), domain.ExecutionPlan{}, domain.Finding{
		ID:          "live-osascript",
		RuleID:      "integration.live.osascript",
		Name:        "AppleScript live probe",
		Action:      domain.ActionCommand,
		Path:        "/usr/bin/osascript",
		DisplayPath: "/usr/bin/osascript -e return \"sift-live\"",
		CommandPath: "/usr/bin/osascript",
		CommandArgs: []string{"-e", `return "sift-live"`},
	}, nil)

	if result.Status != domain.StatusCompleted {
		t.Fatalf("expected live AppleScript command to complete, got %+v", result)
	}
}
