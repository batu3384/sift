# Clean Flow Controller Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first `clean`-specific orchestration layer so SIFT can stop treating clean as a generic menu-to-review hop and instead host a dedicated clean controller surface.

**Architecture:** Introduce a new `cleanFlowModel` that wraps the current clean profiles and preview data, then wire `RouteClean` through that controller instead of the generic `menuModel`. In this first tranche, the controller will own route-local state, first-pass view rendering, cached-preview review handoff, and a stable path for later live scan/review/reclaim phases.

**Tech Stack:** Go, Bubble Tea, Lip Gloss, Cobra-backed TUI runtime, existing SIFT engine callbacks and plan-loading helpers.

---

## File Structure

- Create: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/clean_flow.go`
  - New model types, clean flow state, and construction helpers.
- Create: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/clean_flow_view.go`
  - `View()` and first-pass cinematic clean shell rendering.
- Create: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/clean_flow_viewmodel.go`
  - Summary, lane, selected action, and preview-to-ledger display helpers.
- Modify: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/app.go`
  - Replace `menuModel` usage for `clean` with `cleanFlowModel`.
- Modify: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/app_bootstrap.go`
  - Seed the new clean controller instead of the old menu model.
- Modify: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/app_runtime.go`
  - Keep width/height and preview loading state in sync with the clean controller.
- Modify: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/app_view.go`
  - Route `RouteClean` to the new controller view.
- Modify: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/app_routes_primary.go`
  - Update `updateClean` and preview loading to target `cleanFlowModel`.
- Modify: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/app_test.go`
  - Add app-level failing tests for clean controller activation, preview, and cached review behavior.
- Optionally modify: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/app_help_viewmodel.go`
  - Only if clean route help labels must change to match the new surface.

### Task 1: Add failing app-level tests for the clean controller

**Files:**
- Modify: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/app_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestCleanRouteUsesDedicatedFlowView(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteClean)

	view := model.View()

	for _, needle := range []string{"Live Ledger", "Current focus", "review gate on"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected %q in clean flow view, got %s", needle, view)
		}
	}
}

func TestCleanFlowPreviewLoadUsesControllerState(t *testing.T) {
	t.Parallel()

	model := newTestAppModel(RouteClean)
	model.callbacks.LoadCleanProfile = func(profile string) (domain.ExecutionPlan, error) {
		return domain.ExecutionPlan{
			Command: "clean",
			Profile: profile,
			DryRun:  true,
		}, nil
	}

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected clean controller to start loading review preview")
	}

	if !model.cleanFlow.preview.loading {
		t.Fatalf("expected clean flow preview loading, got %+v", model.cleanFlow.preview)
	}
}

func TestCleanFlowUsesCachedPreviewForImmediateReview(t *testing.T) {
	t.Parallel()

	cachedPlan := domain.ExecutionPlan{Command: "clean", Profile: "safe", DryRun: true}

	model := newTestAppModel(RouteClean)
	model.cleanFlow.applyPreview("safe", cachedPlan, nil)

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected cached preview to bypass reload")
	}
	if next.(appModel).route != RouteReview {
		t.Fatalf("expected review route, got %s", next.(appModel).route)
	}
}
```

- [ ] **Step 2: Run the targeted test slice to verify it fails**

Run:

```bash
source ~/.zprofile >/dev/null 2>&1; source ~/.zshrc >/dev/null 2>&1
env GOCACHE=/Users/batuhanyuksel/Documents/cleaner/.tmp/gocache \
go test ./internal/tui/... -run 'TestCleanRouteUsesDedicatedFlowView|TestCleanFlowPreviewLoadUsesControllerState|TestCleanFlowUsesCachedPreviewForImmediateReview' -count=1
```

Expected:

- fail with missing `cleanFlow` field or missing clean flow output strings

- [ ] **Step 3: Commit the red test state only if you need a checkpoint**

```bash
git add /Users/batuhanyuksel/Documents/cleaner/internal/tui/app_test.go
git commit -m "test: define clean flow controller expectations"
```

### Task 2: Create the clean controller model and first-pass view

**Files:**
- Create: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/clean_flow.go`
- Create: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/clean_flow_view.go`
- Create: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/clean_flow_viewmodel.go`

- [ ] **Step 1: Add the model shell and preview state**

```go
type cleanFlowPhase string

const (
	cleanFlowIdle        cleanFlowPhase = "idle"
	cleanFlowScanning    cleanFlowPhase = "scanning"
	cleanFlowReviewReady cleanFlowPhase = "review_ready"
)

type cleanFlowModel struct {
	title    string
	subtitle string
	actions  []homeAction
	cursor   int
	width    int
	height   int
	hint     string
	phase    cleanFlowPhase
	preview  menuPreviewState
}

func newCleanFlowModel() cleanFlowModel {
	return cleanFlowModel{
		title:    "Clean",
		subtitle: "live reclaim console",
		hint:     "Scan first, review before reclaim.",
		actions:  buildCleanActions(),
		phase:    cleanFlowIdle,
	}
}
```

- [ ] **Step 2: Add minimal preview helpers compatible with the old menu preview loader**

```go
func (m *cleanFlowModel) setPreviewLoading(key string) {
	m.preview = menuPreviewState{key: key, loading: strings.TrimSpace(key) != ""}
	if m.preview.loading {
		m.phase = cleanFlowScanning
	}
}

func (m *cleanFlowModel) applyPreview(key string, plan domain.ExecutionPlan, err error) {
	preview := menuPreviewState{key: key}
	if err != nil {
		preview.err = err.Error()
		m.preview = preview
		return
	}
	preview.plan = plan
	preview.loaded = true
	m.preview = preview
	m.phase = cleanFlowReviewReady
}
```

- [ ] **Step 3: Add a first-pass cinematic view**

```go
func (m cleanFlowModel) View() string {
	width, _ := effectiveSize(m.width, m.height)
	body := renderCleanFlowBody(m, width-4)
	return renderChrome(
		"SIFT / Clean",
		"live reclaim console",
		cleanFlowStats(m, width),
		body,
		nil,
		width,
		false,
		m.height,
	)
}
```

- [ ] **Step 4: Run the targeted tests and keep them green**

Run:

```bash
source ~/.zprofile >/dev/null 2>&1; source ~/.zshrc >/dev/null 2>&1
env GOCACHE=/Users/batuhanyuksel/Documents/cleaner/.tmp/gocache \
go test ./internal/tui/... -run 'TestCleanRouteUsesDedicatedFlowView|TestCleanFlowPreviewLoadUsesControllerState|TestCleanFlowUsesCachedPreviewForImmediateReview' -count=1
```

Expected:

- PASS

### Task 3: Rewire `appModel` to use the clean controller

**Files:**
- Modify: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/app.go`
- Modify: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/app_bootstrap.go`
- Modify: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/app_runtime.go`
- Modify: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/app_view.go`
- Modify: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/app_routes_primary.go`

- [ ] **Step 1: Replace the old clean menu field**

```go
type appModel struct {
	// ...
	home      homeModel
	cleanFlow cleanFlowModel
	tools     menuModel
	// ...
}
```

- [ ] **Step 2: Seed the controller in bootstrap and tests**

```go
cleanFlow: newCleanFlowModel(),
```

- [ ] **Step 3: Route the clean view and size sync through the new model**

```go
m.cleanFlow.width, m.cleanFlow.height = msg.Width, msg.Height
```

and

```go
case RouteClean:
	m.cleanFlow.width, m.cleanFlow.height = m.width, m.height
	return m.cleanFlow.View()
```

- [ ] **Step 4: Update preview loading and cached review logic**

```go
if previewPlan, ok := m.cleanFlow.previewPlanForSelected(); ok {
	m.setReviewPlan(previewPlan, shouldExecutePlan(previewPlan))
	m.route = RouteReview
	m.reviewReturnRoute = RouteClean
	m.resultReturnRoute = RouteClean
	return m, nil
}
```

and

```go
m.cleanFlow.setPreviewLoading(key)
```

- [ ] **Step 5: Run the broader clean-focused TUI tests**

Run:

```bash
source ~/.zprofile >/dev/null 2>&1; source ~/.zshrc >/dev/null 2>&1
env GOCACHE=/Users/batuhanyuksel/Documents/cleaner/.tmp/gocache \
go test ./internal/tui/... -run 'Clean|Home|PlanLoad|Help' -count=1
```

Expected:

- PASS

### Task 4: Build the first live-ledger view model

**Files:**
- Modify: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/clean_flow_viewmodel.go`
- Modify: `/Users/batuhanyuksel/Documents/cleaner/internal/tui/clean_flow_view.go`

- [ ] **Step 1: Derive lane rows from the preview plan**

```go
type cleanFlowLane struct {
	Label string
	Bytes int64
	Rows  []cleanFlowRow
}

type cleanFlowRow struct {
	Label string
	Bytes int64
	State string
}
```

- [ ] **Step 2: Map the preview plan into live-ledger rows**

```go
func cleanFlowLanes(m cleanFlowModel) []cleanFlowLane {
	if !m.preview.loaded {
		return placeholderCleanLanes(m.actions, m.cursor)
	}
	return lanesFromPlan(m.preview.plan)
}
```

- [ ] **Step 3: Render the left live stream and right focus panel**

```go
func renderCleanFlowBody(m cleanFlowModel, width int) string {
	left := renderCleanLiveLedger(m, width)
	right := renderCleanFocusPanel(m, width)
	return joinPanels(left, right, width)
}
```

- [ ] **Step 4: Add output assertions for the new visual contract**

```go
for _, needle := range []string{
	"Live Ledger",
	"Current focus",
	"review gate on",
	"Quick Clean",
} {
	if !strings.Contains(view, needle) {
		t.Fatalf("expected %q in clean flow view, got %s", needle, view)
	}
}
```

- [ ] **Step 5: Run clean-focused render tests**

Run:

```bash
source ~/.zprofile >/dev/null 2>&1; source ~/.zshrc >/dev/null 2>&1
env GOCACHE=/Users/batuhanyuksel/Documents/cleaner/.tmp/gocache \
go test ./internal/tui/... -run 'Clean|Render|Snapshot|Home' -count=1
```

Expected:

- PASS

### Task 5: Full verification and commit

**Files:**
- Modify: all files from Tasks 1-4

- [ ] **Step 1: Run focused clean verification**

```bash
source ~/.zprofile >/dev/null 2>&1; source ~/.zshrc >/dev/null 2>&1
env GOCACHE=/Users/batuhanyuksel/Documents/cleaner/.tmp/gocache \
go test ./internal/tui/... -run 'Clean|Home|PlanLoad|Render|Snapshot|Help' -count=1
```

- [ ] **Step 2: Run full repo tests**

```bash
source ~/.zprofile >/dev/null 2>&1; source ~/.zshrc >/dev/null 2>&1
env GOCACHE=/Users/batuhanyuksel/Documents/cleaner/.tmp/gocache go test ./...
```

- [ ] **Step 3: Run lint and smoke**

```bash
source ~/.zprofile >/dev/null 2>&1; source ~/.zshrc >/dev/null 2>&1
env GOCACHE=/Users/batuhanyuksel/Documents/cleaner/.tmp/gocache make lint
env GOCACHE=/Users/batuhanyuksel/Documents/cleaner/.tmp/gocache make smoke
```

- [ ] **Step 4: Commit**

```bash
git add \
  /Users/batuhanyuksel/Documents/cleaner/internal/tui/clean_flow.go \
  /Users/batuhanyuksel/Documents/cleaner/internal/tui/clean_flow_view.go \
  /Users/batuhanyuksel/Documents/cleaner/internal/tui/clean_flow_viewmodel.go \
  /Users/batuhanyuksel/Documents/cleaner/internal/tui/app.go \
  /Users/batuhanyuksel/Documents/cleaner/internal/tui/app_bootstrap.go \
  /Users/batuhanyuksel/Documents/cleaner/internal/tui/app_runtime.go \
  /Users/batuhanyuksel/Documents/cleaner/internal/tui/app_view.go \
  /Users/batuhanyuksel/Documents/cleaner/internal/tui/app_routes_primary.go \
  /Users/batuhanyuksel/Documents/cleaner/internal/tui/app_test.go \
  /Users/batuhanyuksel/Documents/cleaner/docs/superpowers/plans/2026-04-07-clean-flow-controller-foundation.md
git commit -m "Build clean flow controller foundation"
```

## Self-Review

### Spec coverage

- Covered: clean-specific controller, clean route-local state, first-pass live ledger shell, cached preview review handoff, and verification.
- Deferred intentionally: full scan streaming callbacks, permission-integrated clean controller states, reclaim-phase ledger reuse, and clean-specific result integration. Those belong in the next plan tranche.

### Placeholder scan

- No `TBD`, `TODO`, or generic “handle appropriately” placeholders remain.
- Commands, file paths, and code targets are explicit.

### Type consistency

- The plan consistently uses `cleanFlowModel`, `menuPreviewState`, and the existing `domain.ExecutionPlan` preview loading contract.
