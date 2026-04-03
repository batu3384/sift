# SIFT Architecture

## Overview

SIFT is a cross-platform cleaner built around a single Go binary. The important
design choice is that scanning, policy evaluation, review, execution, and
reporting are separate steps. That keeps destructive behavior observable and
testable.

High-level flow:

```text
CLI/TUI entrypoint
  -> config + store bootstrap
  -> engine scan
  -> policy evaluation
  -> review output (plain, json, or TUI)
  -> optional execution
  -> store + audit persistence
  -> optional report bundle
```

## Main components

### Command surface

- `cmd/sift/main.go` starts the binary.
- `internal/cli` defines the Cobra command tree and shared flags.
- Running `sift` without a subcommand enters the Bubble Tea application when the
  terminal supports it.
- Output policy is centralized so the same rules decide whether a command stays
  plain, switches to JSON, or opens the TUI. That keeps `--json`, `--plain`,
  piped `status/analyze/check`, and destructive `--yes` requirements aligned.

### TUI shell

- `internal/tui` contains the route-driven full-screen app.
- Routes cover Home, Clean, Tools, Protect, Uninstall, Status, Doctor, Analyze,
  Permissions, Review, Progress, and Result screens.
- Home keeps the five primary workflows (`Clean`, `Uninstall`, `Analyze`,
  `Status`, `Optimize`) while secondary maintenance and policy workflows stay
  under `Tools`.
- The TUI never bypasses the engine. It requests plans, shows task-native
  previews, optionally runs permission preflight, and then executes an approved
  plan.
- The TUI shell is intentionally split into seams:
  - `app.go` keeps the top-level Bubble Tea router
  - `app_bootstrap.go` seeds initial state and callbacks
  - `app_plan_flow.go` owns review, preflight, and execution handoff
  - `app_runtime.go` owns shared runtime event handling
  - `app_view.go` focuses route-level rendering dispatch
- Render files such as `status_render.go`, `progress_render.go`, and
  `result_render.go` are expected to stay presentation-focused while route
  models carry the view state.

### Engine and policy

- `internal/engine.Service` is the center of the workflow.
- `Scan` collects findings from rule definitions, normalizes them, applies
  protection policy, sorts them, and produces an `ExecutionPlan`.
- Execution converts reviewed plan items into an `ExecutionResult`, emits
  section/phase events for task-native progress, and persists outcomes.
- Execution orchestration is isolated behind an internal runner layer so the
  service entrypoint stays small while queue/check/apply/verify events remain a
  single engine-owned contract.
- Native command parsing and launch are also split so trust/validation rules
  stay separate from process start behavior.
- Health and telemetry collection are also split into focused probe helpers so
  live snapshot assembly stays readable without changing the `StatusReport`
  contract.
- Policy evaluation decides whether an item is planned, protected, skipped, or
  requires stronger confirmation.

### Rules and discovery

- `internal/rules` owns the curated cleanup catalog.
- Each rule definition provides metadata, candidate roots, and a scanner
  function.
- Rule families cover temp files, logs, developer caches, package manager
  caches, browser data, installer leftovers, app leftovers, and project
  artifact discovery for `purge`.

### Platform adapters

- `internal/platform` isolates OS-specific behavior behind the `Adapter`
  interface.
- Adapters provide curated roots, diagnostics, installed app discovery,
  remnant discovery, native uninstall hints, admin session handling, and
  platform-specific protection.
- Larger Darwin and Windows responsibilities are intentionally split by
  concern: roots, diagnostics, maintenance task catalogs, app discovery, and
  remnant discovery live in separate files so platform-specific behavior can
  evolve without turning one adapter file into a monolith.
- Admin session orchestration is also split between a public warmup/keepalive
  layer and runtime primitives so TTY, GUI prompt, and `sudo` refresh behavior
  can be tested independently.
- This is what keeps the rest of the application largely platform-agnostic.

### Persistence and reporting

- `internal/store` persists plans, executions, and exported reports in SQLite.
- The same package also writes an append-only NDJSON audit trail.
- `internal/report` exports support bundles that include the saved plan, config,
  diagnostics, recent scans, latest execution, status summary, and audit
  records.
- When diagnostics redaction is enabled, bundle content is sanitized before it
  is written.

## Data model

The core workflow revolves around two values from `internal/domain`:

- `ExecutionPlan`: preview of candidate items, totals, warnings, targets, and
  active protection policy
- `ExecutionResult`: executed item outcomes, warnings, and follow-up commands

This separation matters because the app can:

- render a safe preview without deleting anything
- use the same plan in plain text, JSON, or TUI review
- persist what was proposed independently from what actually ran

## Safety model

Safety constraints are part of the architecture, not a UI convention:

- `--dry-run=true` is the default
- destructive JSON and non-interactive runs require explicit confirmation flags
- protected paths can come from user config, built-in platform policy, or safe
  exception logic
- destructive interactive runs can stop in a dedicated permission preflight
  route before execution begins
- admin access is managed through a reusable session warmup + keepalive layer
- native uninstall execution is explicit and now continues into remnant cleanup
  and aftercare within the same reviewed run
- shell-driven deletion patterns are guarded by `hack/security_check.sh`

## Testing strategy

SIFT relies on multiple test layers:

- package tests for rules, engine logic, config, store, report, platform, CLI,
  and TUI behavior
- macOS and Windows smoke scripts that run the built binary against isolated
  fixture roots
- cross-build and package-manifest validation for release readiness
- JSON-first smoke assertions for plan/result contracts, with plain-text checks
  reserved for stable headings and human summaries

See [`docs/TESTING.md`](TESTING.md) for the concrete commands.

## Release architecture

Release automation is split across three parts:

- `.github/workflows/ci.yml` for security checks, tests, smoke runs, and
  manifest validation
- `.github/workflows/release.yml` for tagged releases with GoReleaser
- `hack/` scripts for local dry-runs, manifest generation, and preflight
  validation

See [`docs/RELEASE.md`](RELEASE.md) for the release path in detail.
