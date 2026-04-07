# Clean Flow Controller Design

## Summary

SIFT will adopt a `clean`-specific orchestration layer that keeps the current safety model but replaces the current generic review-first feel with a more continuous, Mole-inspired live operation flow.

The target experience is:

1. `Home`
2. `Clean Console`
3. live scan ledger
4. frozen review on the same visual surface
5. permissions only if needed
6. reclaim on the same ledger language
7. result with settled lanes and residual review

This is explicitly **not** a Mole clone. We will borrow the strengths of Mole's scan and reclaim pacing, but the presentation, copy, routing model, and safety gates stay SIFT-specific.

## Goals

- Make `clean` feel live, item-by-item, and operational instead of static and route-heavy.
- Preserve SIFT's safety character:
  - review stays in the flow
  - permission gates stay explicit
  - destructive work still requires the same approval path
- Keep the existing engine, policy, and permission machinery.
- Establish a reusable orchestration pattern that can later be applied to `uninstall` and `analyze`.

## Non-Goals

- Replacing the shared execution engine.
- Rewriting all command flows in one pass.
- Removing the review or permission gates.
- Changing public CLI command names, flags, or JSON contracts.

## Design Thesis

The `clean` experience becomes a **cinematic live ledger**:

- the left side behaves like a flowing operation stream
- category boundaries stay visible as lanes
- the right side keeps orientation stable with totals, focus, and gate state
- scan, review, permissions, reclaim, and result look like phases of one operation rather than unrelated screens

The visual tone should feel warmer and more authored than the current SIFT shell, but still disciplined enough for a safety-oriented utility.

## Chosen Direction

### Visual direction

Selected direction: `Cinematic Utility`

- warmer accent palette
- authored, premium terminal presentation
- lane-based grouping
- live item stream
- clear focus box and reclaim totals

### Flow direction

Selected direction: `review-first but more alive`

This keeps SIFT's safety posture while removing the dead feeling between:

- scan
- review
- permissions
- reclaim
- result

### Surface model

Selected direction: `Live Ledger`

Every real finding appears as an item row when possible:

- item name or path
- byte size
- current state

Lane headers remain visible so the stream does not collapse into noise.

## Product Behavior

### 1. Home

`Clean` on home should stop feeling like a plain launcher.

The selected detail area should show:

- last sweep summary
- last reclaimed size if available
- a short promise of the live-ledger behavior

`enter` opens the `Clean Console`.

### 2. Clean Console

This is the new orchestration surface for `clean`.

It owns:

- initial scan startup
- live ledger rendering
- lane summary rendering
- focus state
- transition into review
- transition into permissions
- transition into reclaim
- transition into result

The console opens immediately and starts scanning. The user should not feel bounced through an extra loading screen unless startup meaningfully stalls.

### 3. Live Scan Ledger

The scan phase streams findings as rows.

Each row should carry:

- icon or activity marker
- item label
- bytes
- state

Example states:

- `queued`
- `scanning`
- `focus`
- `review`
- `protected`
- `settled`

The stream must remain readable under terminal constraints:

- short, operational copy
- minimal per-row decoration
- stable column rhythm
- lane breaks that reset visual attention

### 4. Frozen Review

When scan completes, the same console shifts into review instead of jumping to a visually unrelated route.

What changes:

- row movement stops
- reviewable rows are now inspectable and toggleable
- right panel becomes more explicit about:
  - safe reclaim
  - review-required reclaim
  - protected items
  - permission requirements

This should feel like the scan "settled into review" instead of "opening another app screen."

### 5. Permissions

Permissions remain conditional.

If the selected reclaim batch needs:

- admin
- dialogs
- native handoff

the user moves into a `cleanPermissions` state using the same shell language.

Permissions must visually read as part of the same operation, not as a foreign interrupt.

### 6. Reclaim

Reclaim reuses the ledger language from scan.

The same rows should now express execution lifecycle:

- `queued`
- `reclaiming`
- `settled`
- `failed`

Lanes remain useful because users should still understand where reclaimed space is coming from.

### 7. Result

Result continues the same visual grammar:

- settled lanes
- total reclaimed bytes
- leftover review or failed rows
- retry or recovery affordances

The result must answer:

- what lanes finished
- how much space was reclaimed
- what still needs attention

## State Model

The `clean` command gets a dedicated controller state machine.

### States

1. `cleanIdle`
2. `cleanScanning`
3. `cleanReviewReady`
4. `cleanPermissions`
5. `cleanReclaiming`
6. `cleanResult`

### Transitions

- `cleanIdle -> cleanScanning`
  - on route entry
- `cleanScanning -> cleanReviewReady`
  - after scan stream finishes and reviewable plan is ready
- `cleanReviewReady -> cleanPermissions`
  - only when permission manifest requires it
- `cleanReviewReady -> cleanReclaiming`
  - when reclaim can start without additional gate
- `cleanPermissions -> cleanReclaiming`
  - after preflight acceptance and warmup
- `cleanReclaiming -> cleanResult`
  - when execution settles
- `cleanResult -> cleanReviewReady`
  - optional recovery or retry path
- `cleanResult -> cleanIdle`
  - optional return-home style reset

## Architecture

### Reused systems

These stay in place:

- scan plan generation
- policy and protection logic
- permission manifest logic
- admin warmup logic
- execution engine
- result and recovery primitives

### New orchestration seam

Add a `clean`-specific controller layer inside TUI.

Proposed files:

- `internal/tui/clean_flow.go`
- `internal/tui/clean_flow_update.go`
- `internal/tui/clean_flow_view.go`
- `internal/tui/clean_flow_viewmodel.go`

Responsibilities:

- own `clean`-specific state
- coordinate route-local transitions
- consume scan progress callbacks
- build live ledger view models
- adapt existing review, permission, and result data into the new surface

### Existing files that should be reused, not replaced

- `internal/tui/app.go`
- `internal/tui/app_runtime.go`
- `internal/tui/plan_runtime.go`
- `internal/tui/preflight.go`
- `internal/engine/scan_plans.go`
- `internal/engine/execution_runner.go`
- `internal/engine/execution_progress.go`

## View Model Design

### Lane

A lane is a visual category grouping in the live ledger.

Fields:

- `key`
- `label`
- `totalBytes`
- `visibleItems`
- `status`

Possible lane sources:

- browser caches
- dev artifacts
- package manager residues
- app support leftovers
- logs or transient data

Lane naming must be curated and stable. We should not expose raw rule internals as user-facing category labels.

### Ledger row

A ledger row is a single reclaim candidate or scan focus row.

Fields:

- `id`
- `label`
- `path`
- `bytes`
- `state`
- `tone`
- `laneKey`
- `reviewRequired`
- `protected`

Rows may be collapsed or truncated for narrow terminals, but the logic should still be item-first.

### Focus panel

The right panel carries:

- current focus item
- total reclaimable bytes
- safe vs review split
- permission summary
- next-step hints

The focus panel prevents the stream from feeling visually uncontrolled.

## Copy Strategy

The copy must stay operational and short.

Good:

- `scanning`
- `queued`
- `review`
- `reclaiming`
- `settled`
- `protected`

Avoid:

- long prose in the live stream
- marketing-like narration
- excessive mascot narration

Mascot usage should be limited to:

- initial clean console hero note
- state shift acknowledgements
- empty or recovery moments

The mascot should not narrate every row.

## Visual System Rules

- Keep one warm accent family for `clean`.
- Maintain high structural contrast.
- Use motion only to clarify:
  - active item
  - lane transitions
  - state transitions
- Respect reduced motion and no-color paths.
- Narrow widths must degrade to:
  - compact lane headers
  - fewer visible rows
  - abbreviated focus panel

## Error Handling

### Scan errors

If scan emits partial or path-specific errors:

- keep the console alive
- mark the affected row or lane
- route details into review or warning summary

Do not dump the user into a generic failure screen unless the whole scan fails.

### Permission refusal

If the user declines permissions:

- keep the ledger intact
- return to review-ready state
- show what remains blocked

### Execution failures

If reclaim fails for some rows:

- move them into failed or review-needed result states
- preserve retry path
- do not discard settled lane context

## Testing Strategy

### Unit tests

Add controller tests for:

- state transitions
- lane aggregation
- live row promotion
- review freeze behavior
- permission gate branching
- reclaim-to-result transitions

### Render tests

Add render contracts for:

- live scan ledger
- frozen review state
- permission-integrated clean view
- reclaim ledger
- result lane summary

### Behavior tests

Add app-level tests for:

- `home -> clean console`
- scan completion into review-ready
- permission-required reclaim
- reclaim settle into result
- reduced-motion behavior
- narrow-width layout

### Smoke tests

Smoke should confirm:

- clean can enter the new console
- scan summary and result summary remain machine-checkable where needed
- no-color and non-TTY flows still behave safely

## Rollout Plan

### Phase 1

Build the `clean` controller only.

Preserve:

- existing shared routes for other commands
- existing engine contracts

### Phase 2

Evaluate whether the controller pattern should be reused for:

- `uninstall`
- `analyze`

### Phase 3

Extend the same orchestration pattern to the remaining commands if the `clean` pilot proves stable.

## Risks

### Risk 1: too much theatrical motion

Mitigation:

- keep row copy short
- animate only active row and lane transitions
- keep the right panel stable

### Risk 2: category noise

Mitigation:

- cap per-lane visible rows
- keep lane labels curated
- do not over-segment into too many small categories

### Risk 3: safety ambiguity

Mitigation:

- review freeze must be visually explicit
- permission manifest must remain visible before reclaim
- destructive execution must still require the same approval semantics

## Decision

Proceed with a `clean`-specific `Clean Flow Controller` that introduces a live, item-by-item ledger for scan and reclaim while preserving SIFT's review and permission model.
