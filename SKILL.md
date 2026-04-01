---
name: sift-go-cli
description: Use when working in the SIFT repository, a Go-based macOS and Windows cleaner with Cobra CLI, Bubble Tea TUI, platform adapters, safety-first execution, SQLite state, smoke scripts, and release manifest generation. Apply for feature work, bug fixes, docs updates, tests, packaging, and repo-native quality checks in this codebase.
---

# SIFT Repo Skill

Use this skill for changes inside this repository.

## Quick map

- `cmd/sift`: binary entrypoint
- `internal/cli`: Cobra commands and output orchestration
- `internal/tui`: Bubble Tea full-screen app
- `internal/engine`: scan, policy, execution, native uninstall flow
- `internal/rules`: cleanup rule catalog and purge discovery
- `internal/platform`: macOS and Windows adapters
- `internal/config`: config defaults, load, normalize, save
- `internal/store`: SQLite state and audit records
- `internal/report`: exported zip support bundles
- `hack`: security, smoke, packaging, and release scripts

## Default workflow

1. Inspect the smallest package that owns the behavior.
2. Preserve the scan -> review -> execute model.
3. Prefer repo-native commands over ad hoc validation.
4. Update tests and docs when behavior or workflows change.

## Safety rules

- Destructive behavior must stay opt-in behind `--dry-run=false`.
- Do not bypass review gates in TUI or non-interactive flows.
- Protected path logic is part of the product contract.
- Native uninstall execution must remain explicit and must not be shell-wrapped.
- Keep `hack/security_check.sh` passing when touching delete or process-launch code.

## Quality gate

Run the narrowest useful checks first:

```bash
./hack/security_check.sh
go test ./...
go vet ./...
go build ./cmd/sift
```

Broaden when needed:

```bash
make smoke
make cross-build
```

Optional when the environment supports it:

```bash
make smoke-windows
make release-dry-run
```

## When editing docs

- Keep [`README.md`](README.md) aligned with shipped commands and flags.
- Use [`CONTRIBUTING.md`](CONTRIBUTING.md) for contributor workflow updates.
- Use [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for package and flow changes.
- Use [`docs/TESTING.md`](docs/TESTING.md) for local/CI quality changes.
- Use [`docs/RELEASE.md`](docs/RELEASE.md) for packaging or release flow changes.

## When editing product logic

- Rules belong in `internal/rules`.
- Scan/execution policy belongs in `internal/engine`.
- OS-specific behavior belongs in `internal/platform`.
- CLI wiring belongs in `internal/cli`.
- Route/view behavior belongs in `internal/tui`.

## Release and packaging

If the task touches packaging, manifests, or GoReleaser, also inspect:

- [`hack/release_dry_run.sh`](hack/release_dry_run.sh)
- [`hack/release_preflight.sh`](hack/release_preflight.sh)
- [`.github/workflows/ci.yml`](.github/workflows/ci.yml)
- [`.github/workflows/release.yml`](.github/workflows/release.yml)

## Extra references

Open only as needed:

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)
- [`docs/TESTING.md`](docs/TESTING.md)
- [`docs/RELEASE.md`](docs/RELEASE.md)
- [`SECURITY_AUDIT.md`](SECURITY_AUDIT.md)
