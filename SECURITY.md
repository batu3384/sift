# Security Policy

## Reporting a Vulnerability

Please use GitHub Security Advisories for private reports:

- [Report a vulnerability](https://github.com/batu3384/sift/security/advisories/new)

Do not open public issues for suspected vulnerabilities.

## Secure-by-default Expectations

- Destructive actions stay review-gated.
- Native/system commands remain capability-checked and test-covered.
- Policy, protection, and whitelist behavior must stay explicit and auditable.

## Security-Sensitive Validation

Run `./hack/security_check.sh` when changing command execution, filesystem
operations, policy decisions, uninstall behavior, or release packaging. For
release candidates, pair that guard with `go test ./...`, platform smoke tests,
and the release verification matrix in [`docs/RELEASE.md`](docs/RELEASE.md).

CI-safe macOS smoke intentionally avoids unattended privilege prompts. Treat
live macOS prompt and admin-session behavior as verified only after
`make integration-live-macos` or `make smoke-live-macos` passes on a prepared
test host.
