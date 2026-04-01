# SIFT Security Audit

This document captures the destructive-safety boundaries SIFT enforces in code
today. It is meant to be read alongside the implementation, tests, and CI
guards, not as a substitute for them.

## Core Boundaries

SIFT defaults to preview mode.

- All cleanup commands start as plans. Destructive execution requires
  `--dry-run=false`.
- Non-interactive and JSON destructive execution also require `--yes`.
- Permanent deletion requires `--force`; otherwise the default path is
  Trash/Recycle Bin first.

The engine denies unsafe targets before deletion.

- Empty, relative, traversal-style, control-character, and critical-root paths
  are blocked.
- Symlink targets are blocked for destructive actions.
- Protected roots and configured protected paths are blocked.
- Items outside allowed cleanup roots are blocked.
- Admin-only items stay blocked unless `--admin` is explicitly enabled.
- A preview fingerprint is verified again at execution time.

References:

- [internal/engine/policy.go](internal/engine/policy.go)
- [internal/engine/service.go](internal/engine/service.go)
- [internal/domain/path.go](internal/domain/path.go)

## Rule Safety

SIFT does not full-disk crawl by default.

- Cleanup rules scan curated user-space roots only.
- `purge` accepts only known project artifact directories.
- Recent purge artifacts are raised to `high` risk instead of being silently
  deleted.
- `analyze` is advisory only.
- `uninstall` includes a native uninstall step before remnant cleanup. It stays reviewable by default, only launches when `--native-uninstall` is explicitly enabled, and does not delete remnants in the same run after launch.
- Running apps are detected during uninstall planning. If the target still appears to be running, SIFT protects the uninstall plan instead of racing the live process.

References:

- [internal/rules/catalog.go](internal/rules/catalog.go)
- [internal/platform/current_darwin.go](internal/platform/current_darwin.go)
- [internal/platform/current_windows.go](internal/platform/current_windows.go)

## Auditability

Every saved plan, execution, and report export is written to the local state
store and appended to a daily NDJSON audit log.

- Reports bundle plan/config/diagnostics plus status summary and recent audit
  records.
- Path redaction is applied when diagnostics redaction is enabled.

References:

- [internal/store/store.go](internal/store/store.go)
- [internal/report/report.go](internal/report/report.go)

## CI Guards

CI enforces static destructive-pattern checks before build/test.

- `os.RemoveAll` is allowed only in the guarded delete helper.
- `exec.Command` is allowed only in the macOS bundle identifier reader and the native uninstall launcher.
- Native uninstall commands are parsed without a shell, reject metacharacters, reject shell executables, and accept only absolute paths or a small trusted allowlist such as `msiexec.exe`.
- Shell escape patterns like `sh -c`, `bash -c`, `cmd.exe`, `powershell -Command`,
  raw `rm -rf`, and forced recursive `Remove-Item` are blocked.

References:

- [hack/security_check.sh](hack/security_check.sh)
- [.github/workflows/ci.yml](.github/workflows/ci.yml)
- [.github/workflows/release.yml](.github/workflows/release.yml)

## Remaining Gaps

- Windows runtime behavior still needs repeated smoke validation on real Windows
  environments, not just cross-builds and CI setup.
- Signing and provenance publishing are not yet wired into release artifacts.
