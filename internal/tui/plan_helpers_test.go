package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/batu3384/sift/internal/domain"
)

func TestCurrentGroupSummaryTracksIncludedBytes(t *testing.T) {
	t.Parallel()

	model := planModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", Action: domain.ActionTrash, Status: domain.StatusPlanned, Category: domain.CategoryBrowserData, Source: "Chrome code cache", Bytes: 1024},
				{ID: "b", Path: "/tmp/b", Action: domain.ActionTrash, Status: domain.StatusPlanned, Category: domain.CategoryBrowserData, Source: "Chrome code cache", Bytes: 2048},
				{ID: "c", Path: "/tmp/c", Action: domain.ActionTrash, Status: domain.StatusPlanned, Category: domain.CategoryLogs, Source: "Application logs", Bytes: 512},
			},
		},
		excluded: map[string]bool{"b": true},
	}

	summary := model.currentGroupSummary()
	if summary.label != "Chrome code cache" {
		t.Fatalf("expected group label to use execution/source label, got %+v", summary)
	}
	if summary.total != 2 || summary.included != 1 || summary.bytes != 1024 {
		t.Fatalf("unexpected group summary: %+v", summary)
	}
}

func TestPlanModuleCountIgnoresAdvisoryItems(t *testing.T) {
	t.Parallel()

	count := planModuleCount(domain.ExecutionPlan{
		Command: "clean",
		Items: []domain.Finding{
			{Path: "/tmp/a", Action: domain.ActionTrash, Category: domain.CategoryBrowserData, Source: "Chrome code cache"},
			{Path: "/tmp/b", Action: domain.ActionTrash, Category: domain.CategoryBrowserData, Source: "Chrome code cache"},
			{Path: "/tmp/c", Action: domain.ActionTrash, Category: domain.CategoryLogs, Source: "Application logs"},
			{Path: "/tmp/d", Action: domain.ActionAdvisory, Category: domain.CategoryDiskUsage, Source: "Immediate child of /tmp"},
		},
	})

	if count != 2 {
		t.Fatalf("expected 2 actionable modules, got %d", count)
	}
}

func TestCalculatePlanTotalsOnlyCountsActionableBytes(t *testing.T) {
	t.Parallel()

	totals := calculatePlanTotals([]domain.Finding{
		{Action: domain.ActionTrash, Status: domain.StatusPlanned, Bytes: 1024, Risk: domain.RiskSafe},
		{Action: domain.ActionCommand, Status: domain.StatusPlanned, Bytes: 2048, Risk: domain.RiskReview},
		{Action: domain.ActionAdvisory, Status: domain.StatusAdvisory, Bytes: 4096, Risk: domain.RiskReview},
		{Action: domain.ActionTrash, Status: domain.StatusProtected, Bytes: 8192, Risk: domain.RiskHigh},
		{Action: domain.ActionSkip, Status: domain.StatusSkipped, Bytes: 16384, Risk: domain.RiskSafe},
	})

	if totals.ItemCount != 5 {
		t.Fatalf("expected item count to preserve all rows, got %+v", totals)
	}
	if totals.Bytes != 3072 || totals.SafeBytes != 1024 || totals.ReviewBytes != 2048 || totals.HighBytes != 0 {
		t.Fatalf("expected only actionable bytes in totals, got %+v", totals)
	}
}

func TestPlanDisplayBytesOnlyCountsActionableCleanupBytes(t *testing.T) {
	t.Parallel()

	plan := domain.ExecutionPlan{
		Command: "clean",
		Totals:  domain.Totals{Bytes: 15 * 1024},
		Items: []domain.Finding{
			{Action: domain.ActionTrash, Status: domain.StatusPlanned, Bytes: 1024},
			{Action: domain.ActionCommand, Status: domain.StatusPlanned, Bytes: 2048},
			{Action: domain.ActionAdvisory, Status: domain.StatusAdvisory, Bytes: 4096},
			{Action: domain.ActionTrash, Status: domain.StatusProtected, Bytes: 8192},
		},
	}
	if got := planDisplayBytes(plan); got != 3072 {
		t.Fatalf("expected display bytes to use actionable cleanup bytes, got %d", got)
	}

	plan.Command = "analyze"
	if got := planDisplayBytes(plan); got != 15*1024 {
		t.Fatalf("expected analyze display bytes to keep total inspected bytes, got %d", got)
	}
}

func TestAnalyzeSummaryLinesIncludeTopChildAndTopFile(t *testing.T) {
	t.Parallel()

	lines := analyzeSummaryLines(domain.ExecutionPlan{
		Command: "analyze",
		Totals:  domain.Totals{Bytes: 9 * 1024},
		Items: []domain.Finding{
			{Name: "cache", Bytes: 6 * 1024, Category: domain.CategoryDiskUsage, LastModified: time.Now()},
			{Name: "archive.zip", Bytes: 3 * 1024, Category: domain.CategoryLargeFiles, LastModified: time.Now()},
		},
	})
	joined := strings.Join(lines, "\n")
	for _, needle := range []string{
		"Summary  children 1",
		"files 1",
		"Top  child cache",
		"file archive.zip",
	} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected %q in analyze summary, got %s", needle, joined)
		}
	}
}

func TestAnalyzeLineUsesReadableNaturalOrder(t *testing.T) {
	t.Parallel()

	line := analyzeLine(domain.Finding{
		Name:         "slack-cache",
		DisplayPath:  "/tmp/slack-cache",
		Bytes:        2 * 1024 * 1024,
		Risk:         domain.RiskReview,
		LastModified: time.Now().Add(-2 * time.Hour),
	})

	for _, needle := range []string{"slack-cache", "2.0 MB", "review"} {
		if !strings.Contains(line, needle) {
			t.Fatalf("expected %q in analyze line, got %q", needle, line)
		}
	}
	if !strings.Contains(line, " • ") {
		t.Fatalf("expected analyze line to use bullet-separated flow, got %q", line)
	}
}

func TestUninstallTargetSummaryLinesSummarizeTargetsAndOverflow(t *testing.T) {
	t.Parallel()

	lines := uninstallTargetSummaryLines(domain.ExecutionPlan{
		Command: "uninstall",
		Targets: []string{"Arc", "Slack", "Notion", "Discord"},
		Totals:  domain.Totals{Bytes: 8 * 1024},
		Items: []domain.Finding{
			{Action: domain.ActionNative},
			{Action: domain.ActionTrash},
			{Action: domain.ActionTrash, Status: domain.StatusProtected},
		},
	}, 120)

	joined := strings.Join(lines, "\n")
	for _, needle := range []string{
		"4 apps",
		"1 native step",
		"2 remnants",
		"1 protected item",
		"Arc",
		"Slack",
		"Notion",
		"+1 more target",
	} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected %q in uninstall summary, got %s", needle, joined)
		}
	}
}
