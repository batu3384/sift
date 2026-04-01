package engine

import (
	"strings"
	"testing"

	"github.com/batu3384/sift/internal/domain"
)

func TestProgressHelpersUseCommandSpecificLanguage(t *testing.T) {
	t.Parallel()

	cleanItem := domain.Finding{DisplayPath: "/tmp/cache", Action: domain.ActionTrash}
	if got := progressQueueDetail("clean", cleanItem); !strings.Contains(got, "queued reclaim") {
		t.Fatalf("expected clean queue detail, got %q", got)
	}
	if got := progressCheckDetail("clean", cleanItem); got != "checking reclaim target" {
		t.Fatalf("expected clean check detail, got %q", got)
	}
	if got := progressApplyStep("clean", cleanItem, false); got != "reclaim" {
		t.Fatalf("expected clean apply step, got %q", got)
	}
	if got := progressApplyDetail("clean", cleanItem, false); !strings.Contains(got, "reclaiming /tmp/cache") {
		t.Fatalf("expected clean apply detail, got %q", got)
	}
	if got := progressVerifyDetail("clean", cleanItem); !strings.Contains(got, "checking reclaim result") {
		t.Fatalf("expected clean verify detail, got %q", got)
	}

	uninstallItem := domain.Finding{DisplayPath: "/tmp/example", Action: domain.ActionTrash}
	if got := progressQueueDetail("uninstall", uninstallItem); !strings.Contains(got, "queued remnant") {
		t.Fatalf("expected uninstall queue detail, got %q", got)
	}
	if got := progressApplyStep("uninstall", uninstallItem, false); got != "remove" {
		t.Fatalf("expected uninstall apply step, got %q", got)
	}
	if got := progressApplyDetail("uninstall", uninstallItem, false); !strings.Contains(got, "removing remnant") {
		t.Fatalf("expected uninstall apply detail, got %q", got)
	}
	if got := progressVerifyDetail("uninstall", uninstallItem); !strings.Contains(got, "checking remnant result") {
		t.Fatalf("expected uninstall verify detail, got %q", got)
	}

	nativeItem := domain.Finding{DisplayPath: "Example", Action: domain.ActionNative}
	if got := progressQueueDetail("uninstall", nativeItem); !strings.Contains(got, "queued native handoff") {
		t.Fatalf("expected native queue detail, got %q", got)
	}
	if got := progressApplyStep("uninstall", nativeItem, false); got != "handoff" {
		t.Fatalf("expected native handoff step, got %q", got)
	}
}

func TestManagedProgressHelpersUsePhaseAndFixLanguage(t *testing.T) {
	t.Parallel()

	optimizeItem := domain.Finding{DisplayPath: "/tmp/maintenance", Action: domain.ActionCommand, TaskPhase: "repair"}
	if got := progressQueueDetail("optimize", optimizeItem); !strings.Contains(got, "queued repair task") {
		t.Fatalf("expected optimize queue detail, got %q", got)
	}
	if got := progressCheckDetail("optimize", optimizeItem); got != "checking repair access" {
		t.Fatalf("expected optimize check detail, got %q", got)
	}
	if got := progressApplyStep("optimize", optimizeItem, false); got != "repair" {
		t.Fatalf("expected optimize apply step, got %q", got)
	}
	if got := progressManagedTaskDetail("optimize", optimizeItem); !strings.Contains(got, "repair task") {
		t.Fatalf("expected optimize managed task detail, got %q", got)
	}

	autofixItem := domain.Finding{DisplayPath: "/tmp/firewall", Action: domain.ActionCommand}
	if got := progressQueueDetail("autofix", autofixItem); !strings.Contains(got, "queued fix") {
		t.Fatalf("expected autofix queue detail, got %q", got)
	}
	if got := progressCheckDetail("autofix", autofixItem); got != "checking fix access" {
		t.Fatalf("expected autofix check detail, got %q", got)
	}
	if got := progressApplyStep("autofix", autofixItem, false); got != "fix" {
		t.Fatalf("expected autofix apply step, got %q", got)
	}
	if got := progressManagedTaskDetail("autofix", autofixItem); !strings.Contains(got, "running fix") {
		t.Fatalf("expected autofix managed task detail, got %q", got)
	}
	if got := progressManagedVerifyDetail("autofix", autofixItem); got != "checking fix result" {
		t.Fatalf("expected autofix verify detail, got %q", got)
	}
}
