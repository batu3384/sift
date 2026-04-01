# Mole Fresh Clone Gap Report

Baseline:

- Upstream repo: `https://github.com/tw93/mole`
- Canonical clone: `.tmp/mole-upstream.latest`
- Previous baseline: `ea4cd9d0e30e0f3563a82d4fe64e60de509faa8b`
- Current baseline: `13274154d40251be00bcd5cd4379efe04ee5bb1d`
- Compare range: `ea4cd9d0e30e0f3563a82d4fe64e60de509faa8b...13274154d40251be00bcd5cd4379efe04ee5bb1d`

Latest upstream delta:

1. `.github/workflows/release.yml`: `actions/download-artifact` bump from `8.0.0` to `8.0.1`.

Current SIFT position:

- Tracked parity surfaces remain closed for cleanup, installer, purge, optimize,
  uninstall, analyze, status, protect, launcher, bootstrap, completion, and
  version workflows.
- SIFT still holds the structural advantage in typed policy, review/progress/result
  flow, command-scoped exclusions, cross-platform packaging, and evidence-backed
  parity tracking.
- This refresh closes the remaining Mole differences Jarvis reported around
  install/bootstrap, short alias ergonomics, completion auto-install, richer
  version output, and repository/community scaffolding.

Area comparison:

| Area | Mole surface | SIFT surface | Status | Notes |
| --- | --- | --- | --- | --- |
| Bootstrap install | `install.sh`, `README.md` | `install.sh`, `README.md` | Covered | SIFT now ships a curl/bootstrap installer that downloads the latest release and installs `sift`. |
| Short alias | `README.md`, `tests/cli.bats` | `install.sh`, `si`, `README.md` | Covered | Bootstrap installs the short `si` alias and the repo now includes a local `si` wrapper. |
| Completion install | `bin/completion.sh`, `tests/completion.bats` | `internal/cli/commands_system.go`, `internal/cli/completion_install.go`, `README.md` | Covered | `sift completion --install` writes completion files and wires shell config for bash, zsh, fish, and PowerShell. |
| Rich version shell | `mole` | `internal/cli/version_info.go`, `internal/cli/root.go`, `README.md` | Covered | `sift version` and `sift --version` now expose install method, channel, shell context, executable path, disk free, and SIP posture. |
| Repository maturity | `.github` | `.github/CODEOWNERS`, `.github/dependabot.yml`, `.github/ISSUE_TEMPLATE`, `.github/pull_request_template.md`, `.github/workflows/codeql.yml` | Covered | Contributor automation, ownership, and security scanning now match the tracked Mole baseline. |
| Touch ID lifecycle | `bin/touchid.sh` | `internal/engine/touchid.go`, `internal/cli/commands_maintenance.go` | Covered | Mole’s modal quit-key fix remains a non-gap because SIFT uses a non-modal reviewed CLI flow. |
| Dialog-sensitive test isolation | `lib/check/all.sh`, `lib/uninstall/batch.sh` | `internal/platform/testmode.go`, `internal/platform/current_darwin.go`, `internal/engine/execution.go`, `internal/tui/open_darwin.go` | Covered | `SIFT_TEST_MODE=ci-safe` and `SIFT_LIVE_INTEGRATION=1` keep CI deterministic without hiding native behavior. |
| Analyze cache layering | `cmd/analyze/cache.go`, `cmd/analyze/scanner.go` | `internal/analyze/scan.go`, `internal/analyze/preview.go` | Covered | Preview and scan caching now live in shared backend modules with in-flight coalescing. |
| Status operator alerts | `cmd/status/view.go` | `internal/engine/health.go`, `internal/tui/status_render.go`, `internal/tui/home_render.go` | Covered | Live telemetry drives operator alerts in the home and status rails. |

Watchlist:

- Mole’s upstream churn is currently release-automation only, but future comparison
  passes should continue to watch `cmd/status/*` render flavor and
  `cmd/analyze/*` scanner ergonomics for non-functional drift.
- SIFT now exceeds Mole’s test-file count, so the important watch item is no
  longer raw breadth but keeping live integration and CI-safe contract layers in
  sync when desktop-facing behavior changes.
