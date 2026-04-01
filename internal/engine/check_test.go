package engine

import (
	"strings"
	"testing"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/platform"
)

func TestCheckItemFromDiagnosticMapsSecurityAndConfigFindings(t *testing.T) {
	service := &Service{}

	firewall, ok := service.checkItemFromDiagnostic(platform.Diagnostic{
		Name:    "firewall",
		Status:  "warn",
		Message: "Firewall is disabled.",
	})
	if !ok {
		t.Fatal("expected firewall diagnostic to map to check item")
	}
	if firewall.ID != checkIDFirewall || firewall.Group != domain.CheckGroupSecurity || !firewall.AutofixAvailable {
		t.Fatalf("unexpected firewall item: %+v", firewall)
	}

	touchID, ok := service.checkItemFromDiagnostic(platform.Diagnostic{
		Name:    "touchid",
		Status:  "warn",
		Message: "Touch ID is disabled for sudo.",
	})
	if !ok {
		t.Fatal("expected touchid diagnostic to map to check item")
	}
	if touchID.ID != checkIDTouchID || touchID.Group != domain.CheckGroupConfig || !touchID.AutofixAvailable {
		t.Fatalf("unexpected touchid item: %+v", touchID)
	}
}

func TestAutofixItemsForCheckUsesMaintenanceTaskForMemoryPressure(t *testing.T) {
	service := &Service{}
	task := domain.MaintenanceTask{
		ID:              "macos.optimize.memory-relief",
		Title:           "Relieve memory pressure",
		Description:     "Drop reclaimable inactive pages.",
		Risk:            domain.RiskReview,
		Action:          domain.ActionCommand,
		CommandPath:     "/usr/bin/sudo",
		CommandArgs:     []string{"purge"},
		TimeoutSeconds:  60,
		Capability:      "purge available",
		Phase:           "repair",
		EstimatedImpact: "Drops reclaimable inactive pages.",
		Verification:    []string{"Check memory pressure in sift status"},
	}

	items, warnings := service.autofixItemsForCheck(domain.CheckItem{
		ID:     checkIDMemoryPressure,
		Status: "warn",
		Name:   "Memory pressure",
	}, map[string]domain.MaintenanceTask{task.ID: task})
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(items) != 1 {
		t.Fatalf("expected one autofix item, got %v", items)
	}
	if items[0].RuleID != task.ID || items[0].CommandPath != "/usr/bin/sudo" || items[0].TaskPhase != "repair" {
		t.Fatalf("unexpected maintenance autofix item: %+v", items[0])
	}
}

func TestCheckItemFromUpdateNoticeRequiresAvailableRelease(t *testing.T) {
	if _, ok := checkItemFromUpdateNotice(UpdateNotice{}); ok {
		t.Fatal("expected unavailable update notice to be skipped")
	}

	item, ok := checkItemFromUpdateNotice(UpdateNotice{
		Available: true,
		Message:   "Update available.",
		Commands:  []string{"sift update"},
	})
	if !ok {
		t.Fatal("expected available update notice to become a check item")
	}
	if item.ID != checkIDSiftUpdate || item.Group != domain.CheckGroupUpdates || !item.AutofixAvailable {
		t.Fatalf("unexpected update check item: %+v", item)
	}
}

func TestHealthCheckItemsEmitsWarningsForPressureSignals(t *testing.T) {
	items := healthCheckItems(&SystemSnapshot{
		MemoryUsedPercent: 84.5,
		SwapUsedPercent:   24.0,
		DiskFreeBytes:     10 * 1024 * 1024 * 1024,
		HealthScore:       62,
		HealthLabel:       "warm",
	})
	if len(items) < 4 {
		t.Fatalf("expected multiple health items, got %v", items)
	}

	joined := make([]string, 0, len(items))
	for _, item := range items {
		joined = append(joined, item.ID+":"+item.Status)
	}
	summary := strings.Join(joined, ",")
	for _, expected := range []string{
		checkIDMemoryPressure + ":warn",
		checkIDSwapPressure + ":warn",
		checkIDDiskPressure + ":warn",
		checkIDHealthScore + ":warn",
	} {
		if !strings.Contains(summary, expected) {
			t.Fatalf("expected %q in %q", expected, summary)
		}
	}
}
