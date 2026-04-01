# Changelog

All notable changes to this project will be documented in this file.

This changelog is intentionally release-oriented so GitHub releases can reuse
the same summary with minimal editing.

## Unreleased

### Added
- Reduced-motion TUI mode via `SIFT_REDUCED_MOTION=1`.
- Deterministic app-level width and resize regression coverage for the full TUI router.
- Explicit permission preflight and accepted-profile reuse across interactive execution flows.
- Task-native preview loading for `clean`, `uninstall`, and staged `analyze` review.
- Command-specific progress and result tracks for `clean`, `uninstall`, `optimize`, and `autofix`.

### Changed
- Tightened CI-safe execution so admin-managed commands stay unattended-safe in smoke coverage.
- Hardened macOS smoke coverage for native uninstall handoff, remnant cleanup, and aftercare.
- Simplified route help, footer hints, and review/result wording to keep the interface more glanceable.
- Refined Home, Status, Analyze, Review, Progress, and Result layouts for compact and wide terminal sizes.

### Fixed
- Route-local loading and transition timing issues that previously made parts of the TUI feel stalled.
- Darwin process snapshot race conditions in system health collection.
- Non-interactive native-uninstall smoke behavior.
- Window size propagation gaps affecting progress rendering after terminal resizes.

## 0.1.0

### Added
- Initial public baseline for the typed Go implementation of SIFT.
- Cross-platform CLI and TUI flows for scan, review, execute, report, protect, uninstall, optimize, and diagnostics.
- GitHub Actions CI, release automation, package manifest generation, and repository contribution scaffolding.
