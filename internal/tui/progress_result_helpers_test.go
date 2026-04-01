package tui

import (
	"strings"
	"testing"

	"github.com/batu3384/sift/internal/domain"
)

func TestProgressStageBucketsAndInfoTrackModuleFlow(t *testing.T) {
	t.Parallel()

	progress := progressModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", Category: domain.CategoryBrowserData, Source: "Chrome code cache", Bytes: 100, Action: domain.ActionTrash},
				{ID: "b", Path: "/tmp/b", Category: domain.CategoryBrowserData, Source: "Chrome code cache", Bytes: 200, Action: domain.ActionTrash},
				{ID: "c", Path: "/tmp/c", Category: domain.CategoryLogs, Source: "Application logs", Bytes: 300, Action: domain.ActionTrash},
			},
		},
		items: []domain.OperationResult{
			{FindingID: "a", Path: "/tmp/a", Status: domain.StatusDeleted},
		},
		cursor: 1,
	}

	order, buckets := progressStageBuckets(progress)
	if len(order) != 2 {
		t.Fatalf("expected 2 stage buckets, got %v", order)
	}
	first := buckets[order[0]]
	second := buckets[order[1]]
	if first == nil || first.label != "Chrome code cache" || first.total != 2 || first.done != 1 || first.bytes != 300 {
		t.Fatalf("unexpected first bucket: %+v", first)
	}
	if second == nil || second.label != "Application logs" || second.total != 1 || second.done != 0 || second.bytes != 300 {
		t.Fatalf("unexpected second bucket: %+v", second)
	}

	stage := progressStageInfo(progress)
	if stage.Group != "Chrome code cache" || stage.Index != 1 || stage.Total != 2 || stage.Done != 1 || stage.Items != 2 {
		t.Fatalf("unexpected stage info: %+v", stage)
	}
}

func TestProgressFreedBytesMatchesByIDAndPath(t *testing.T) {
	t.Parallel()

	plan := domain.ExecutionPlan{
		Items: []domain.Finding{
			{ID: "id-a", Path: "/a/path", Bytes: 100},
			{ID: "id-b", Path: "/b/path", Bytes: 200},
			{ID: "id-c", Path: "/c/path", Bytes: 50},
		},
	}
	// id-a matched by ID, id-c matched by path only (no FindingID on result)
	progress := progressModel{
		plan: plan,
		items: []domain.OperationResult{
			{FindingID: "id-a", Path: "/a/path", Status: domain.StatusDeleted},
			{FindingID: "", Path: "/c/path", Status: domain.StatusCompleted},
			{FindingID: "id-b", Path: "/b/path", Status: domain.StatusFailed},
		},
	}
	got := progressFreedBytes(progress)
	// 100 (id-a, deleted) + 50 (id-c, completed by path) = 150
	if got != 150 {
		t.Fatalf("expected 150 freed bytes, got %d", got)
	}
}

func TestResultFreedBytesCountsDeletedAndCompleted(t *testing.T) {
	t.Parallel()

	plan := domain.ExecutionPlan{
		Items: []domain.Finding{
			{ID: "r1", Path: "/x", Bytes: 1000},
			{ID: "r2", Path: "/y", Bytes: 500},
			{ID: "r3", Path: "/z", Bytes: 200},
		},
	}
	result := domain.ExecutionResult{
		Items: []domain.OperationResult{
			{FindingID: "r1", Path: "/x", Status: domain.StatusDeleted},
			{FindingID: "r2", Path: "/y", Status: domain.StatusFailed},
			{FindingID: "r3", Path: "/z", Status: domain.StatusCompleted},
		},
	}
	got := resultFreedBytes(plan, result)
	if got != 1200 {
		t.Fatalf("expected 1200 freed bytes, got %d", got)
	}
}

func TestProgressModuleFlowLinesShowDoneNowNext(t *testing.T) {
	t.Parallel()

	progress := progressModel{
		plan: domain.ExecutionPlan{
			Command: "clean",
			Items: []domain.Finding{
				{ID: "a", Path: "/tmp/a", Category: domain.CategoryBrowserData, Source: "Chrome code cache", Bytes: 100, Action: domain.ActionTrash},
				{ID: "b", Path: "/tmp/b", Category: domain.CategoryLogs, Source: "Application logs", Bytes: 200, Action: domain.ActionTrash},
				{ID: "c", Path: "/tmp/c", Category: domain.CategoryDeveloperCaches, Source: "Developer cache", Bytes: 300, Action: domain.ActionTrash},
			},
		},
		items: []domain.OperationResult{
			{FindingID: "a", Path: "/tmp/a", Status: domain.StatusDeleted},
		},
		current: &domain.Finding{ID: "b", Path: "/tmp/b", Category: domain.CategoryLogs, Source: "Application logs"},
	}

	lines := progressModuleFlowLines(progress, 140)
	joined := strings.Join(lines, "\n")
	for _, needle := range []string{"Done", "Chrome code cache", "Now", "Application logs", "Next", "Developer cache"} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected %q in module flow lines, got %s", needle, joined)
		}
	}
}

func TestResultRecoveryCandidatesResolveByIDAndFilterStatuses(t *testing.T) {
	t.Parallel()

	plan := domain.ExecutionPlan{
		Command: "clean",
		Items: []domain.Finding{
			{ID: "a", Path: "/tmp/a", Category: domain.CategoryBrowserData, Source: "Chrome code cache", Bytes: 100, Action: domain.ActionTrash},
			{ID: "b", Path: "/tmp/b", Category: domain.CategoryLogs, Source: "Application logs", Bytes: 200, Action: domain.ActionTrash},
		},
	}
	result := domain.ExecutionResult{
		Items: []domain.OperationResult{
			{FindingID: "a", Path: "/tmp/a", Status: domain.StatusDeleted},
			{FindingID: "b", Path: "/tmp/b", Status: domain.StatusFailed},
			{FindingID: "missing", Path: "/tmp/unknown", Status: domain.StatusProtected},
		},
	}

	candidates := resultRecoveryCandidatesForStatuses(plan, result, resultFilterAll, domain.StatusFailed, domain.StatusProtected)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 recovery candidates, got %+v", candidates)
	}
	if candidates[0].Path != "/tmp/b" {
		t.Fatalf("expected failed known path first, got %+v", candidates)
	}
	if candidates[1].Path != "/tmp/unknown" || candidates[1].Action != domain.ActionSkip {
		t.Fatalf("expected fallback candidate for unknown protected path, got %+v", candidates[1])
	}
}

func TestResultCurrentGroupRecoveryCandidatesStayWithinModule(t *testing.T) {
	t.Parallel()

	plan := domain.ExecutionPlan{
		Command: "clean",
		Items: []domain.Finding{
			{ID: "a", Path: "/tmp/a", Category: domain.CategoryBrowserData, Source: "Chrome code cache", Bytes: 100, Action: domain.ActionTrash},
			{ID: "b", Path: "/tmp/b", Category: domain.CategoryBrowserData, Source: "Chrome code cache", Bytes: 200, Action: domain.ActionTrash},
			{ID: "c", Path: "/tmp/c", Category: domain.CategoryLogs, Source: "Application logs", Bytes: 300, Action: domain.ActionTrash},
		},
	}
	model := resultModel{
		plan: plan,
		result: domain.ExecutionResult{
			Items: []domain.OperationResult{
				{FindingID: "a", Path: "/tmp/a", Status: domain.StatusFailed},
				{FindingID: "b", Path: "/tmp/b", Status: domain.StatusProtected},
				{FindingID: "c", Path: "/tmp/c", Status: domain.StatusFailed},
			},
		},
		cursor: 0,
		filter: resultFilterAll,
	}

	group := resultCurrentGroupRecoveryCandidates(model)
	if len(group) != 2 {
		t.Fatalf("expected 2 candidates for current module, got %+v", group)
	}
	for _, item := range group {
		if groupedItemLabel(item) != "Chrome code cache" {
			t.Fatalf("expected only current module candidates, got %+v", group)
		}
	}
}
