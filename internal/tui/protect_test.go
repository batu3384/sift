package tui

import (
	"strings"
	"testing"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func TestProtectViewShowsFamilyAndScopeMatrices(t *testing.T) {
	t.Parallel()

	model := newProtectModel([]string{"/tmp/keep"})
	model.syncFamilies([]string{"browser_profiles", "ai_workspaces"})
	model.syncScopes(map[string][]string{
		"clean":     {"/tmp/keep", "/tmp/cache"},
		"purge_scan": {"/Users/test/dev"},
	})
	model.width = 132
	model.height = 30

	view := model.View()
	for _, needle := range []string{"Family Matrix", "Scope Matrix", "browser_profiles", "ai_workspaces", "clean", "purge_scan"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in protect view, got %s", needle, view)
		}
	}
}

func TestProtectDetailShowsDecisionPath(t *testing.T) {
	t.Parallel()

	model := newProtectModel([]string{"/tmp/keep"})
	model.syncFamilies([]string{"browser_profiles"})
	model.syncScopes(map[string][]string{"clean": {"/tmp/keep"}})
	model.explanation = &domain.ProtectionExplanation{
		Path:             "/tmp/keep",
		Command:          "clean",
		State:            domain.ProtectionStateUserProtected,
		Message:          "Blocked by a user-configured protected path.",
		UserMatches:      []string{"/tmp/keep"},
		FamilyMatches:    []string{"browser_profiles"},
		ExceptionMatches: []string{"safe cache"},
	}

	view := model.detailView(72, 20)
	for _, needle := range []string{"Decision Path", "command scopes", "Command  clean", "State    user protected", "Family Matrix", "Scope Matrix"} {
		if !strings.Contains(strings.ToLower(view), strings.ToLower(needle)) {
			t.Fatalf("expected %q in protect detail, got %s", needle, view)
		}
	}
}

func TestProtectViewRenderMatrix(t *testing.T) {
	t.Parallel()

	base := newProtectModel([]string{"/tmp/keep", "/tmp/cache"})
	base.syncFamilies([]string{"browser_profiles", "launcher_state"})
	base.syncScopes(map[string][]string{
		"clean":    {"/tmp/keep"},
		"optimize": {"/tmp/cache"},
	})

	cases := []struct {
		width  int
		height int
	}{
		{80, 24},
		{100, 30},
		{144, 40},
	}

	for _, tc := range cases {
		model := base
		model.width = tc.width
		model.height = tc.height
		view := model.View()
		for _, needle := range []string{"PROTECT", "Family Matrix"} {
			if !strings.Contains(strings.ToLower(view), strings.ToLower(needle)) {
				t.Fatalf("expected %q in protect view for %dx%d, got %s", needle, tc.width, tc.height, view)
			}
		}
		if tc.width >= 100 && !strings.Contains(strings.ToLower(view), "command scopes") {
			t.Fatalf("expected command scopes summary in wider protect view for %dx%d, got %s", tc.width, tc.height, view)
		}
		if tc.width >= 140 && !strings.Contains(strings.ToLower(view), "scope matrix") {
			t.Fatalf("expected scope matrix in widest protect view for %dx%d, got %s", tc.width, tc.height, view)
		}
		if got := len(strings.Split(view, "\n")); got > tc.height {
			t.Fatalf("expected protect view to fit within %d lines, got %d", tc.height, got)
		}
	}
}
