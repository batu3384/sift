# Contributing

## Workflow
- Create focused changes with tests.
- Run `go test ./...` before opening a pull request.
- Run `make smoke` for behavior changes that affect CLI, TUI, or platform integrations.
- Run `./hack/security_check.sh` before merge when touching command execution, filesystem operations, or policy paths.

## Expectations
- Keep destructive flows review-gated.
- Prefer typed models over stringly-typed branching.
- Preserve macOS-first product quality while keeping Windows behavior explicit.
- Add or update tests for new command contracts, UI rails, or native-action behavior.

## Pull Requests
- Use the pull request template.
- Describe user-visible behavior, validation evidence, and any platform-specific constraints.
- Call out follow-up work when a design difference is intentional.
