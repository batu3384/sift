# Roadmap and Known Limitations

SIFT is intentionally release-gated. The sections below separate implemented
behavior from validation still required before stronger public claims.

## Near-Term Release Gates

- Keep the protected `main` branch green in GitHub Actions across `ci`, CodeQL,
  OpenSSF Scorecard, macOS smoke, and Windows smoke.
- Re-run `go test ./...` on the release candidate.
- Run `make smoke` on macOS with CI-safe behavior.
- Run `make smoke-windows` on Windows or a PowerShell-capable runner.
- Run `make integration-live-macos` or `make smoke-live-macos` on a host where live macOS service prompts are acceptable.
- Run package-manifest generation and release preflight for candidate artifacts.
- Capture the missing Permissions, Progress, and Result screenshots listed in `docs/SCREENSHOTS.md`.

## Known Limitations

- Remote CI is passing on `main`, but every release candidate still needs a
  fresh local and remote verification pass before tagging.
- macOS CI-safe smoke intentionally skips live prompts and admin-managed follow-up commands.
- Windows support is validated in GitHub Actions on a Windows runner; local
  Windows smoke is still required when changing platform-specific behavior.
- Live macOS integration can interact with host services and should be run only on a prepared test machine.
- Package-manager install paths depend on validated release artifacts. Do not claim Homebrew, Scoop, or Winget availability until the corresponding manifests have passed preflight for the release.
- Screenshot coverage is incomplete until Permissions, Progress, and Result captures are added.

## Product Roadmap

- Keep destructive flows review-first and permission-aware.
- Keep Windows parity explicit by documenting unsupported or skipped native actions.
- Expand screenshot coverage only from deterministic fixture data.
- Improve release evidence by attaching smoke artifacts and manifest validation outputs to tagged releases.
- Keep progress/result screens readable before adding visual complexity.
