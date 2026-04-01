# SIFT

SIFT is a cross-platform terminal cleaner for macOS and Windows. It keeps the
command surface direct and terminal-first while replacing shell-driven logic
with a typed Go core, a scan-plan-execute pipeline, explicit permission
preflight, and a safety-first audit trail.

Running `sift` without a subcommand opens a single full-screen, route-driven
terminal app so you can move between Home, Clean, Uninstall, Analyze, Status,
Review, Permissions, Progress, and Result screens without tearing down the UI.

## Highlights

- One Go binary with platform adapters for macOS and Windows
- Safety-first execution: dry-run by default, trash-first deletion, audit logs
- Bubble Tea/Lip Gloss TUI for interactive scan and cleanup flows
- Single alt-screen terminal app with a shared router, route-local loading, a 5-item home menu, compact help bar, and consistent screens across all interactive flows
- Task-native preview loading for `clean`, `uninstall`, and staged `analyze` review so selected work is summarized before you commit to review
- Explicit permission preflight for destructive runs, with admin/dialog/native access summarized before execution starts
- Sectioned execution progress for clean, uninstall, optimize, and autofix flows
- Stable JSON output for automation and support workflows
- Local SQLite state store for scan history and debug bundles
- Daily NDJSON audit trail for plans, executions, and report exports
- Expanded curated coverage for browser caches, package managers, developer tooling, installer roots, and stale app-support locations on both macOS and Windows
- Quick launcher setup for Raycast and Alfred via [`hack/setup-quick-launchers.sh`](hack/setup-quick-launchers.sh)
- Bootstrap installer for tagged releases via [`install.sh`](install.sh)
- Optional short wrapper command via [`si`](si)

## Install

### Script install

```bash
curl -fsSL https://raw.githubusercontent.com/batuhanyuksel/sift/main/install.sh | sh
```

This installs `sift` plus the short alias `si` into `~/.local/bin` by default.
Set `PREFIX=/custom/bin` to override the target directory.

### Homebrew

```bash
brew install batuhanyuksel/tap/sift
```

## Documentation

- [CONTRIBUTING.md](CONTRIBUTING.md): contributor workflow, safety expectations, and local checks
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md): package layout and scan-plan-review-execute flow
- [docs/MOLE_GAP_REPORT.md](docs/MOLE_GAP_REPORT.md): latest Mole fresh-clone baseline, compare range, and parity notes
- [docs/TESTING.md](docs/TESTING.md): local quality gate, smoke tests, and CI mapping
- [docs/RELEASE.md](docs/RELEASE.md): local release dry-runs, manifest generation, and tagged release flow
- [SECURITY_AUDIT.md](SECURITY_AUDIT.md): destructive-safety and security boundary notes
- [SECURITY.md](SECURITY.md): disclosure policy

## Commands

```text
sift analyze [targets...]
sift check
sift clean [profile]
sift clean --whitelist [list|add <path>|remove <path>]
sift autofix
sift installer
sift purge <rule-or-path>
sift purge scan [roots...]
sift protect list
sift protect add <path>
sift protect remove <path>
sift protect explain <path>
sift protect family list
sift protect family add <family>
sift protect family remove <family>
sift protect scope list [command]
sift protect scope add <command> <path>
sift protect scope remove <command> <path>
sift uninstall <app>
sift optimize
sift optimize --whitelist [list|add <path>|remove <path>]
sift update
sift remove
sift status
sift doctor
sift version
sift report [scan-id]
sift completion [shell]
sift touchid
```

## Profiles

- `safe`: temp files, logs, obvious stale caches
- `developer`: safe + developer and package-manager caches
- `deep`: broad cleanup with extra review warnings

Profiles remain part of the CLI and config model. In the interactive TUI,
`Clean` is a single top-level workflow and these presets are presented as
cleanup depth choices instead of raw profile names. The TUI home screen keeps
five primary flows: `Clean`, `Uninstall`, `Analyze`, `Status`, and `Optimize`.
Secondary workflows such as `Check`, `Autofix`, `Protect Paths`, `Purge Scan`,
and `Doctor` stay under `Tools` via the tools shortcut.

The interactive `Uninstall` screen now includes a searchable installed-app list
with a visible distinction between apps that expose a native uninstall command
and apps where SIFT can only offer remnant review.

## Command Notes

- `sift`, `sift status`, `sift analyze`, and `sift check` prefer the interactive TUI on a real terminal. When `status`, `analyze`, or `check` are piped, SIFT automatically emits JSON unless `--plain` is set.
- `sift check`: actionable posture audit across `security`, `updates`, `config`, and `health`. It is read-only by design and acts as the intake stage for `autofix`.
- `sift autofix`: turns autofixable `check` findings into a reviewed execution plan. It uses the same review/progress/result flow as `optimize`, keeps `--dry-run=true` by default, and requires `--dry-run=false --yes` to apply.
- `sift analyze`: read-only disk usage analysis. Without a target it analyzes your home directory, separates the output into "largest children" and "large files", lets you drill into directories with `enter` and `backspace`, and in TUI mode supports a staged cleanup queue: `space` to stage, `u` to unstage, `o` to open/reveal the selected path, and `x` to hand staged items into the normal cleanup review flow.
- `sift status`: shows live CPU, per-core load, memory, swap, disk, disk I/O, network, uptime, process/user counts, virtualization hints, optional battery/power source, optional proxy details, a health score, top processes, then recent scan history and execution deltas.
- `sift installer`: direct cleanup flow for stale installer packages across common download locations, with zip archives checked for installer payloads before they are flagged.
- `sift installer` now also covers broader macOS drop zones such as `Documents`, `Public`, `/Users/Shared`, iCloud Downloads, and Telegram Desktop, and it recognizes `.mpkg` and `.xip` alongside the existing installer archive types.
- `sift installer` also flags stale incomplete downloads (`.download`, `.crdownload`, `.part`) but skips files that are still actively held open by the system.
- `sift clean`, `sift purge`, `sift uninstall`: plan first, execute only when `--dry-run=false` is explicitly set. In interactive terminals the TUI becomes a real review gate, shows a permission preflight when access changes are required, and requires explicit `y` to continue.
- `sift clean --whitelist ...` and `sift optimize --whitelist ...` are command-scoped exclusion managers. They let you block paths from one workflow without globally protecting those paths from every command.
- `sift purge <path>` only accepts known project artifact directories such as `node_modules`, `dist`, `build`, `target`, `.next`, `venv`, and similar cache/build outputs.
- `sift purge scan [roots...]` discovers known project artifact directories under one or more search roots and returns a preview-only purge plan.
- `sift protect ...` manages user-defined protected paths, protected data families, and command-scoped exclusions. `sift protect explain <path> --command clean` shows whether a path is blocked by user policy, a protected family, a command scope, built-in policy, or allowed as a safe curated cache exception.
- `sift protect family ...` also includes `launcher_state` to protect Raycast, Alfred, and related local automation state.
- `sift optimize` uses the same review/progress/result flow as cleanup commands and now mixes real safe cache resets with managed maintenance commands such as DNS flush, LaunchServices refresh, Quick Look reload, Spotlight rebuild, Dock refresh, and Bluetooth reset.
- `sift optimize` and `sift autofix` now expose a task-board style review with phase, suggested-by checks, verification hints, and impact metadata instead of a generic cleanup list.
- `sift update` previews the detected install-method update command by default and applies it only with `--dry-run=false --yes`. It now supports `--channel nightly` for manual installs and `--force` to request a reinstall-style update where the install method supports it.
- `sift version` and `sift --version` show the current build, install method, update channel, shell context, executable path, disk-free summary, and SIP posture.
- `sift remove` reviews and deletes only SIFT-owned local state. Binary/package-manager uninstall remains an explicit manual step.
- `sift touchid` reports Touch ID sudo status, while `sift touchid enable|disable` provides a dry-run preview by default and applies changes only with `--dry-run=false --yes`. On non-macOS platforms it returns an explicit unsupported message.
- `sift doctor` now reports config/store health plus report cache, audit log, purge discovery defaults, protection policy summary, command-scoped exclusions, a tracked Mole parity matrix summary, and the current upstream baseline compare range.
- `sift uninstall <app>` includes a native uninstall step when the platform exposes one. It launches only when `--native-uninstall` is set; after the handoff, SIFT continues in the same reviewed run with remnant cleanup and aftercare guidance instead of forcing a second uninstall pass.
- `sift uninstall <app>` also checks whether the target app still appears to be running. If it is, SIFT protects the whole uninstall plan and tells you to close the app first.
- `sift uninstall <app>` now also adds platform-aware aftermath guidance in review/result, including LaunchServices refresh and Homebrew follow-up on macOS when relevant.
- `sift uninstall <app>` now also offers managed launch-agent unload follow-ups so stale per-user or system daemons can be unloaded from the review flow instead of being left as manual cleanup.

## Output Contracts

- `status`, `analyze`, and `check` automatically emit JSON when stdout is piped. Use `--plain` to force human-readable output in piped workflows.
- `doctor --json` emits the same diagnostic set the TUI uses, including `parity_matrix` and `upstream_baseline`.
- Destructive non-interactive or JSON flows require both `--dry-run=false` and `--yes`.
- Destructive `--json` flows emit a single envelope with both `plan` and `result` instead of printing multiple JSON documents.
- `check --json` emits a structured `CheckReport`.
- `status --json` emits a `StatusReport` with live telemetry and recent history.
- `analyze --json` emits a regular `ExecutionPlan`, which is the same schema used by interactive review.

## Quick Launchers

Install Raycast script commands and Alfred workflows for the main SIFT flows:

```bash
./hack/setup-quick-launchers.sh
```

Pass an explicit binary path if `sift` is not already on your `PATH`:

```bash
./hack/setup-quick-launchers.sh /absolute/path/to/sift
```

## Completion

Generate completion to stdout:

```bash
sift completion zsh > ~/.zfunc/_sift
```

Or let SIFT install it and update the matching shell profile:

```bash
sift completion --install
```

## Build

```bash
go build ./cmd/sift
```

## Test

```bash
go test ./...
```

## Local Verification

```bash
make quality-gate
make security-check
make vet
make test
make smoke
make smoke-live-macos
make integration-live-macos
make smoke-windows
make completions
make cross-build
make refresh-mole-upstream
make release-dry-run
make package-manifests TAG=v0.0.0-ci DIST_DIR=./.tmp/package-dist OUT_DIR=./.tmp/manifests
make release-preflight TAG=v0.0.0-ci DIST_DIR=./.tmp/package-dist MANIFEST_DIR=./.tmp/manifests
```

## Configuration

SIFT writes its user config to the platform config directory on first `sift doctor` run.
See `config.example.toml` for the supported shape.

- `interaction_mode`: `auto`, `plain`, or `tui`
- `trash_mode`: `trash_first` or `permanent`
- `confirm_level`: `strict` or `balanced`
- `disabled_rules`: suppress specific built-in rule IDs
- `protected_paths`: never delete below these roots
- `protected_families`: optional protected data families such as `browser_profiles`, `password_managers`, `vpn_proxy`, `developer_identity`, `mail_accounts`, `ai_workspaces`, and `ide_settings`
- `command_excludes`: command-scoped exclusion roots such as `clean = ["~/Projects/keep-me/build"]`
- `purge_search_paths`: roots used by `sift purge scan` when no paths are passed
- `diagnostics.redaction`: redact `$HOME` paths in debug bundles

## Operational Notes

- Destructive runs require `--dry-run=false`.
- Non-interactive or JSON destructive runs also require `--yes`.
- Destructive `--json` flows now emit a single envelope with both `plan` and `result` instead of printing two separate JSON documents.
- `interaction_mode` is now enforced: `plain` disables Bubble Tea, `tui` prefers the interactive review flow on supported terminals.
- `confirm_level = "balanced"` skips extra confirmation only for trash-first, safe-only `clean` and `installer` runs; `purge` and `uninstall` still require review.
- `sift purge` rejects shallow or ambiguous paths and raises recent artifacts to high risk instead of deleting them silently.
- `sift purge scan` never executes deletion itself; it discovers candidate artifact directories and emits the same plan schema used by other scan commands.
- `sift clean` scans the largest immediate cleanup candidates inside curated roots instead of targeting whole cache roots as a single delete action.
- `sift protect add/remove/list` persists normalized absolute paths into config and feeds the same protection policy used during execution.
- `sift protect family add/remove/list` activates broader built-in protection families without requiring hand-managed path lists.
- `sift protect scope add/remove/list` manages command-scoped exclusions so paths can be blocked from `clean` or `optimize` without becoming globally protected.
- `sift uninstall` now expands beyond the app bundle into discovered remnant locations using platform-specific support path discovery.
- Native uninstall execution is explicit: `sift uninstall "<App>" --dry-run=false --native-uninstall` launches the parsed vendor command without a shell, keeps the reviewed execution session alive, and continues with remnant cleanup plus aftercare when the native handoff returns.
- `sift optimize` now uses the shared review/progress/result flow for real safe maintenance resets plus advisory guidance, instead of acting as a read-only placeholder.
- `sift doctor` includes a parity matrix summary generated from the tracked Mole feature inventory; remaining `missing` rows stay visible until SIFT reaches parity.
- `sift remove` builds a reviewable plan for SIFT-owned config, report cache, and audit log paths, then leaves binary uninstall to the printed package-manager guidance.
- `sift status` shows recent scans, delta versus the previous scan, and the current audit log path.
- Interactive destructive flows now stage through `Review -> Permissions -> Progress -> Result` only when access changes are required; already-accepted permission profiles skip the extra stop and go straight into execution.
- `Clean`, `Uninstall`, and staged `Analyze` runs now preload task-native preview plans so selection state, module/target coverage, and reclaim estimates are visible before full review opens.
- Browser/profile identity roots, password material, VPN/proxy state, mail/account data, IDE settings, and key app identity paths are now protected by built-in policy and can also be elevated via protected families.
- `sift report` exports a zip bundle with the saved plan, config snapshot, diagnostics, status summary, recent scans, latest execution summary, and recent audit records.
- Interactive execution now flows through `Review -> Progress -> Result`, so destructive work stays visible while it is running and failed/protected items can be reopened from Result with `x`.
- CI runs on both macOS and Windows, including native smoke commands, completion generation, purge scan/protect coverage, and cross-build checks for the other supported platform.
- CI smoke is now environment-isolated on both macOS and Windows, so `clean`, `purge`, report generation, shell completions, and native uninstall follow-up flows run against deterministic fixture roots instead of the host machine state.
- Release dry-runs and `make package-manifests` now generate validated Homebrew formula output, Scoop manifest output, and Winget manifests from built archives. `make release-preflight` verifies archives, `checksums.txt`, and generated manifests together, and CI uploads macOS/Windows smoke evidence as artifacts.

## Security

- Security boundaries and destructive-safety notes: [SECURITY_AUDIT.md](SECURITY_AUDIT.md)
- Disclosure policy: [SECURITY.md](SECURITY.md)
- Release artifacts are configured to publish a `checksums.txt` manifest through GoReleaser.
