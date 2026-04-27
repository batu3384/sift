# Release Workflow

SIFT publishes a single Go application with package metadata for Homebrew,
Scoop, and Winget. The repository already contains the scripts needed for both
local dry-runs and GitHub Actions releases.

## Release prerequisites

Local release preparation may require:

- Go `1.25.0+`
- `goreleaser`
- `rg`
- `zip`
- `shasum` or `sha256sum`
- optional signing/notarization credentials; set `SIFT_SIGNING_IDENTITY` and `SIFT_NOTARY_PROFILE` if your release environment signs artifacts. When absent, release preflight reports the missing signing context explicitly.

## Versioning

- Releases are triggered by pushing a Git tag that matches `v*`.
- Packaging scripts derive the short version from that tag by removing the `v`
  prefix.
- Release URLs keep the exact tag value, for example `v1.2.3`.
- Archive names use the short version without the `v` prefix, for example
  `sift_1.2.3_darwin_arm64.tar.gz`.
- Checksums and package manifests must agree with the generated archive names.

## Local dry-run

Use this when changing release automation or package metadata:

```bash
make release-dry-run
```

That command runs:

1. `./hack/security_check.sh`
2. `go test ./...`
3. `goreleaser release --snapshot --clean --skip=publish`
4. `hack/generate_package_manifests.sh`
5. `hack/release_preflight.sh`

The result is a full local snapshot release with manifest validation.

## Recommended local release gate

Use this before touching tags or final package-manager metadata:

```bash
make quality-gate-full
```

That target runs the standard quality gate first and then attempts the local
snapshot release path. If `goreleaser` is not installed, it reports a skip
instead of failing silently. If `pwsh` is not installed, the Windows smoke step
is skipped in the same explicit way.

## Manual manifest validation

If you already have built archives, you can validate manifests without running
GoReleaser again:

```bash
make package-manifests TAG=v0.0.0-ci DIST_DIR=./.tmp/package-dist OUT_DIR=./.tmp/manifests
make release-preflight TAG=v0.0.0-ci DIST_DIR=./.tmp/package-dist MANIFEST_DIR=./.tmp/manifests
```

Expected outputs:

- `homebrew/Sift.rb`
- `scoop/sift.json`
- `winget/batu3384.SIFT.yaml`
- `winget/batu3384.SIFT.locale.en-US.yaml`
- `winget/batu3384.SIFT.installer.yaml`
- validation marker files for each package manager

## CI and tagged releases

### Continuous integration

`.github/workflows/ci.yml` validates the release path before merge by running:

- the destructive pattern guard
- `go vet ./...`
- `go test ./...`
- macOS race tests
- macOS and Windows smoke coverage
- cross-builds for supported targets
- local package-manifest generation and preflight validation

### Tagged release

`.github/workflows/release.yml` runs on tag push and:

- checks out the full git history
- reruns the security guard and tests
- executes GoReleaser with publish enabled
- generates and validates package manifests from `dist/`
- uploads Homebrew, Scoop, Winget, and combined manifest artifacts

## Artifact expectations

A healthy release should include:

- macOS archives for amd64 and arm64
- Windows zip archives for amd64 and arm64
- `checksums.txt`
- validated package manifests for Homebrew, Scoop, and Winget
- a script-installable release path via the repository root [`install.sh`](../install.sh)

## Bootstrap install contract

The release assets are expected to remain compatible with the repository root
bootstrap script:

- asset naming stays `sift_<version>_<os>_<arch>`
- macOS archives continue to contain a top-level `sift` binary
- `install.sh` can resolve the latest GitHub release and install both `sift`
  and the short alias `si`

If you change archive naming, top-level binary names, or release asset layout,
update `install.sh`, README install instructions, and this file together.

## When to update release docs

Update this file and README if you change:

- supported target platforms or architectures
- tag naming rules
- manifest file names or package manager ownership
- the commands in `hack/release_dry_run.sh`
- the required local tooling for release preparation

## GitHub publish readiness

Before creating the remote repository or opening the first public release:

1. Ensure the local worktree is clean.
2. Run `make quality-gate-full`.
3. Confirm `README.md`, `CHANGELOG.md`, `CONTRIBUTING.md`, and `SECURITY.md` are current.
4. Confirm `.github/workflows/ci.yml` and `.github/workflows/release.yml` still match the actual build and packaging contract.
5. Push the branch, verify Actions succeeds, then create the release tag from that validated commit.
