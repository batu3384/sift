# Testing and Quality Gate

This project already has a strong repository-native quality workflow. Use these
commands instead of inventing ad hoc checks so local validation stays aligned
with CI.

## Fast local loop

For most code changes:

```bash
make quality-gate
```

If you only need the pinned lint pass:

```bash
make lint
```

If you want the explicit command-by-command sequence instead:

```bash
./hack/security_check.sh
go test ./...
go vet ./...
go build ./cmd/sift
```

Default test posture uses the CI-safe contract layer. Dialog-sensitive macOS
operations can be suppressed by exporting `SIFT_TEST_MODE=ci-safe`. Live host
service validation is re-enabled only when `SIFT_LIVE_INTEGRATION=1` is set.

What each step covers:

- `./hack/security_check.sh`: guards against forbidden destructive or shell
  invocation patterns in `internal/` and `cmd/`
- `make lint`: installs pinned `staticcheck` and `shellcheck` builds into
  `.tmp/tools/bin`, then runs Go and shell lint passes
- `go test ./...`: package-level unit and behavior tests across the codebase
- `go vet ./...`: common correctness issues
- `go build ./cmd/sift`: binary still compiles end-to-end
- `go test -race ./...`: concurrency regressions in runtime probes and long-lived flows

## Smoke coverage

Smoke tests exercise the built binary against isolated fixture roots instead of
your real machine state.

They intentionally prefer structured JSON assertions over fragile text matching
for plan and execution contracts. Plain-text checks stay focused on stable human
headers and summary lines.

### macOS smoke

```bash
make smoke
```

This runs the contract layer and exports `SIFT_TEST_MODE=ci-safe` unless you
explicitly opt into live integration.

In `ci-safe` mode, admin-managed follow-up commands are intentionally skipped
instead of prompting for `sudo`. That keeps smoke runs unattended-safe while
`make integration-live-macos` continues to cover the real privilege path.

For the live macOS validation layer:

```bash
make smoke-live-macos
```

For the focused Go integration layer that exercises host services directly:

```bash
make integration-live-macos
```

`hack/macos_smoke.sh` validates:

- doctor diagnostics
- protect add/list/explain/remove flows
- protect family list/add/remove flows
- analyze output
- clean plans for safe and deep coverage
- purge and purge scan behavior
- uninstall planning plus same-session native uninstall handoff, remnant cleanup, and aftercare
- optimize/update/remove/touchid command surfaces
- status output and suggested commands
- shell completion generation
- report bundle export
- CI-safe guards for login item enumeration/removal and desktop open/reveal
- permission preflight and command-specific execution flow labels

`make integration-live-macos` validates:

- AppleScript command execution in live mode
- login item enumeration in live mode
- Finder open/reveal flows in live mode

### Windows smoke

```bash
make smoke-windows
```

Requirements:

- `pwsh`
- ability to cross-build the Windows binary locally

The PowerShell smoke script mirrors the same product-level checks with Windows
paths and uninstall behavior.

## TUI regression focus

When you touch the full-screen interface, also run the focused render and route
contract sweep:

```bash
go test ./internal/tui/... -run 'Snapshot|Render|Help|Preflight|Home|Status|Analyze|Progress|Result|Motion|Plan' -count=1
```

The TUI also supports `SIFT_REDUCED_MOTION=1`. Use it when you need
deterministic captures or want to verify the accessibility fallback:

```bash
SIFT_REDUCED_MOTION=1 ./sift
```

## Cross-platform verification

Run this when you touch platform adapters, CLI behavior used by smoke tests, or
packaging:

```bash
make cross-build
```

This compiles:

- Darwin amd64
- Darwin arm64
- Windows amd64
- Windows arm64

## Release and packaging checks

When modifying `hack/`, `.goreleaser.yml`, or packaging metadata:

```bash
make package-manifests TAG=v0.0.0-ci DIST_DIR=./.tmp/package-dist OUT_DIR=./.tmp/manifests
make release-preflight TAG=v0.0.0-ci DIST_DIR=./.tmp/package-dist MANIFEST_DIR=./.tmp/manifests
```

If GoReleaser is installed and you need the full local snapshot path:

```bash
make release-dry-run
```

For the complete local gate, including race detection:

```bash
make quality-gate-full
```

The preflight script validates:

- required archives exist
- `checksums.txt` entries match generated artifacts
- Homebrew formula output, release URLs, and SHA values
- Scoop manifest output, release URLs, and SHA values
- Winget manifests, release URLs, SHA values, and validation markers

## Recommended quality gate by change type

### Docs-only changes

```bash
./hack/security_check.sh
go test ./...
```

Also run a Markdown link/reference sanity check for changed docs. At minimum,
confirm every relative Markdown link points at an existing file or anchor target
that is intentionally managed by GitHub.

### Engine, rules, CLI, or config changes

```bash
make quality-gate
```

### Platform or release changes

```bash
make quality-gate
make smoke-windows
make cross-build
```

## CI mapping

GitHub Actions already enforces most of this:

- `ci.yml` runs the security guard on macOS
- `ci.yml` runs govulncheck on Ubuntu
- `ci.yml` runs `go vet ./...` and `go test ./...` on macOS and Windows
- `ci.yml` runs race tests on macOS
- `ci.yml` runs macOS and Windows smoke flows
- `ci.yml` validates generated package manifests on macOS
- `codeql.yml` runs CodeQL on push, pull request, and weekly schedule
- `scorecard.yml` publishes OpenSSF Scorecard SARIF on `main` pushes, weekly, and on manual dispatch
- `release.yml` reruns tests and executes GoReleaser for tags

Local results are necessary but not enough for a release candidate. After
pushing, verify GitHub Actions directly before treating a commit as
release-ready.

## Validation Matrix

| Scope | Minimum command | What it proves | What it does not prove |
| --- | --- | --- | --- |
| Docs-only change | `go test ./...` plus Markdown link/reference sanity | The repo still compiles/tests and docs links are coherent | Remote CI, screenshots, or release artifacts |
| TUI rendering change | Focused `internal/tui` test sweep | Route snapshots and reduced-motion render contracts | Live terminal behavior on every OS |
| CLI/engine behavior change | `make quality-gate` | Security guard, lint, tests, vet, smoke, completions, cross-build | Tagged release publishing |
| Windows behavior change | `make smoke-windows` | Runtime behavior through the Windows smoke script | macOS native prompt behavior |
| macOS native integration change | `make integration-live-macos` or `make smoke-live-macos` | Host-service and prompt-sensitive behavior | Windows behavior |
| Packaging/release change | `make quality-gate-full` | Full local gate plus release dry-run when GoReleaser is available | Remote Actions success until pushed |

## Troubleshooting

- If `make smoke-windows` fails immediately, verify `pwsh` is installed.
- If `make release-dry-run` fails, confirm `goreleaser` is available in `PATH`.
- `make quality-gate` skips the Windows smoke step when `pwsh` is missing.
- `make quality-gate-full` skips `release-dry-run` when `goreleaser` is missing.
- If smoke tests fail after touching platform paths, inspect `.tmp/ci-smoke-*`
  outputs before changing the scripts.
- Do not skip `./hack/security_check.sh` when adding native execution or delete
  code paths.
- If a macOS test unexpectedly opens Finder, System Events, or a permission
  prompt, rerun under `SIFT_TEST_MODE=ci-safe` and fix the missing test guard.
