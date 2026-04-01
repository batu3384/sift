#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

command -v goreleaser >/dev/null 2>&1 || {
  echo "goreleaser is required for release-dry-run" >&2
  exit 1
}

./hack/security_check.sh
go test ./...
goreleaser release --snapshot --clean --skip=publish
archive="$(find "$ROOT_DIR/dist" -maxdepth 1 -name 'sift_*_windows_amd64.zip' | head -n 1)"
[[ -n "$archive" ]] || {
  echo "missing windows amd64 archive after goreleaser snapshot" >&2
  exit 1
}
tag="$(basename "$archive")"
tag="${tag#sift_}"
tag="${tag%_windows_amd64.zip}"
./hack/generate_package_manifests.sh "$tag" "$ROOT_DIR/dist" "$ROOT_DIR/dist/manifests"
./hack/release_preflight.sh "$tag" "$ROOT_DIR/dist" "$ROOT_DIR/dist/manifests"
