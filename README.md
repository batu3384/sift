# SIFT

[![CI](https://github.com/batu3384/sift/actions/workflows/ci.yml/badge.svg)](https://github.com/batu3384/sift/actions/workflows/ci.yml)
[![Release](https://github.com/batu3384/sift/actions/workflows/release.yml/badge.svg)](https://github.com/batu3384/sift/actions/workflows/release.yml)
[![License](https://img.shields.io/github/license/batu3384/sift)](LICENSE)

SIFT is a review-first terminal cleaner for macOS and Windows. It keeps the
workflow fast and terminal-native, but replaces shell-script cleanup logic with
a typed Go core, explicit safety policy, permission preflight, audit history,
and a single full-screen TUI that carries destructive work from selection to
execution without context switching.

Running `sift` opens the application shell. From there you can move through
`Home`, `Clean`, `Uninstall`, `Analyze`, `Status`, `Review`, `Permissions`,
`Progress`, and `Result` in one routed interface.

## Screenshots

<table>
  <tr>
    <td width="33%">
      <img src="docs/assets/screenshots/home.png" alt="SIFT home screen" />
    </td>
    <td width="33%">
      <img src="docs/assets/screenshots/analyze.png" alt="SIFT analyze screen" />
    </td>
    <td width="33%">
      <img src="docs/assets/screenshots/review.png" alt="SIFT review screen" />
    </td>
  </tr>
  <tr>
    <td valign="top">
      <strong>Home</strong><br />
      Primary workflows, live state, and fast entry into cleanup or status.
    </td>
    <td valign="top">
      <strong>Analyze</strong><br />
      Explorer-style disk analysis with staged review handoff.
    </td>
    <td valign="top">
      <strong>Review</strong><br />
      Planned deletions, protected findings, and explicit execution control.
    </td>
  </tr>
</table>

## Why SIFT

- Review-first destructive flows. Cleanup is previewed before it is applied.
- Dry-run by default. Destructive non-interactive runs require both `--dry-run=false` and `--yes`.
- Cross-platform core. One Go binary with platform adapters for macOS and Windows.
- Explicit permission model. Admin, dialog, and native handoff requirements are shown before execution.
- Auditable behavior. Plans, executions, diagnostics, and reports are written to local state and audit logs.
- Task-native TUI. `clean`, `uninstall`, and staged `analyze` runs preload real preview plans so the selected work is visible before full review opens.

## Core Workflows

### Clean

Choose a cleanup scope, review planned findings, then execute through
`Review -> Permissions -> Progress -> Result`.

Profiles:

- `safe`: temp files, logs, obvious stale caches
- `developer`: safe plus developer and package-manager caches
- `deep`: broader cleanup with stronger review warnings

### Uninstall

Search installed apps, review remnants, optionally launch a native uninstall,
and continue in the same session through remnant cleanup and aftercare.

### Analyze

Inspect large directories and files, drill into folders, stage findings, and
send them into the standard cleanup review flow.

### Optimize and Autofix

Use the same reviewed execution model for safe maintenance actions and
autofixable posture findings.

## Install

### Install Script

```bash
curl -fsSL https://raw.githubusercontent.com/batu3384/sift/main/install.sh | sh
```

This installs `sift` and the short wrapper `si` into `~/.local/bin` by default.
Set `PREFIX=/custom/bin` to override the install location.

### Go Install

```bash
go install github.com/batu3384/sift/cmd/sift@latest
```

### Build From Source

```bash
git clone https://github.com/batu3384/sift.git
cd sift
go build -o ./sift ./cmd/sift
```

## Quick Start

```bash
# Launch the full-screen application shell
sift

# Open a reviewed cleanup plan for the safe profile
sift clean safe

# Analyze a path and stage items into cleanup review
sift analyze ~/Downloads

# Live status in plain text or JSON
sift status --plain
sift status --json

# Posture audit and reviewed autofix flow
sift check
sift autofix
```

## Safety Model

- `clean`, `purge`, `uninstall`, `optimize`, `autofix`, `remove`, and `touchid` preview first.
- Interactive destructive flows stay in the TUI and require explicit confirmation.
- Permission preflight summarizes admin, dialog, and native handoff requirements before execution.
- Protected paths, protected data families, and command-scoped exclusions are enforced by the same policy engine used during execution.
- JSON and non-interactive destructive runs never proceed unless the intent is explicit.

## Command Surface

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
sift report [scan-id]
sift version
sift completion [shell]
sift touchid
```

## Output and Automation Notes

- `status`, `analyze`, and `check` automatically emit JSON when stdout is piped. Use `--plain` to force human-readable output.
- `doctor --json` emits the same diagnostic set used by the TUI.
- `analyze --json` emits a regular `ExecutionPlan`, matching interactive review.
- `status --json` emits a structured `StatusReport`.
- Set `SIFT_REDUCED_MOTION=1` to keep the TUI interactive while disabling spinner and pulse animation.

## Configuration

SIFT writes its user config to the platform config directory. See
[config.example.toml](config.example.toml) for the supported shape.

Important keys:

- `interaction_mode`: `auto`, `plain`, or `tui`
- `trash_mode`: `trash_first` or `permanent`
- `confirm_level`: `strict` or `balanced`
- `disabled_rules`: suppress specific built-in rule IDs
- `protected_paths`: never delete below these roots
- `protected_families`: enable broader built-in protection groups
- `command_excludes`: command-scoped exclusions such as `clean = ["~/Projects/keep-me/build"]`
- `purge_search_paths`: default roots for `sift purge scan`
- `diagnostics.redaction`: redact `$HOME` paths in debug bundles

## Development

```bash
go test ./...
make smoke
make quality-gate-full
./hack/security_check.sh
```

Other useful targets:

- `make integration-live-macos`
- `make cross-build`
- `make completions`
- `make release-dry-run`
- `make package-manifests TAG=v0.0.0-ci DIST_DIR=./.tmp/package-dist OUT_DIR=./.tmp/manifests`

README screenshots are generated from deterministic fixture roots with:

```bash
./hack/capture_readme_screens.sh
```

## Documentation

- [CHANGELOG.md](CHANGELOG.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- [SECURITY.md](SECURITY.md)
- [SECURITY_AUDIT.md](SECURITY_AUDIT.md)
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- [docs/TESTING.md](docs/TESTING.md)
- [docs/RELEASE.md](docs/RELEASE.md)
- [docs/MOLE_GAP_REPORT.md](docs/MOLE_GAP_REPORT.md)

## License

[MIT](LICENSE)
