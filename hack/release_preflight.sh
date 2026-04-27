#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

TAG="${1:?usage: release_preflight.sh <tag> [dist-dir] [manifest-dir]}"
DIST_DIR="${2:-$ROOT_DIR/dist}"
MANIFEST_DIR="${3:-$DIST_DIR/manifests}"

SHORT_VERSION="${TAG#v}"
CHECKSUM_FILE="$DIST_DIR/checksums.txt"
DARWIN_AMD64="sift_${SHORT_VERSION}_darwin_amd64.tar.gz"
DARWIN_ARM64="sift_${SHORT_VERSION}_darwin_arm64.tar.gz"
WINDOWS_AMD64="sift_${SHORT_VERSION}_windows_amd64.zip"
WINDOWS_ARM64="sift_${SHORT_VERSION}_windows_arm64.zip"

hash_file() {
  local file="$1"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
    return
  fi
  sha256sum "$file" | awk '{print $1}'
}

require_file() {
  local file="$1"
  [[ -f "$file" ]] || { echo "missing required file: $file" >&2; exit 1; }
}

require_checksum_entry() {
  local archive="$1"
  local expected_sha="$2"
  local line
  line="$(grep "  ${archive}\$" "$CHECKSUM_FILE" || true)"
  [[ -n "$line" ]] || { echo "missing checksum entry for ${archive}" >&2; exit 1; }
  local actual_sha
  actual_sha="$(printf '%s' "$line" | awk '{print $1}')"
  [[ "$actual_sha" == "$expected_sha" ]] || {
    echo "checksum mismatch for ${archive}: expected ${expected_sha}, got ${actual_sha}" >&2
    exit 1
  }
}

for archive in "$DARWIN_AMD64" "$DARWIN_ARM64" "$WINDOWS_AMD64" "$WINDOWS_ARM64"; do
  require_file "$DIST_DIR/$archive"
done
require_file "$CHECKSUM_FILE"

darwin_amd64_sha="$(hash_file "$DIST_DIR/$DARWIN_AMD64")"
darwin_arm64_sha="$(hash_file "$DIST_DIR/$DARWIN_ARM64")"
windows_amd64_sha="$(hash_file "$DIST_DIR/$WINDOWS_AMD64")"
windows_arm64_sha="$(hash_file "$DIST_DIR/$WINDOWS_ARM64")"

require_checksum_entry "$DARWIN_AMD64" "$darwin_amd64_sha"
require_checksum_entry "$DARWIN_ARM64" "$darwin_arm64_sha"
require_checksum_entry "$WINDOWS_AMD64" "$windows_amd64_sha"
require_checksum_entry "$WINDOWS_ARM64" "$windows_arm64_sha"

require_file "$MANIFEST_DIR/validation.txt"
require_file "$MANIFEST_DIR/homebrew/Sift.rb"
require_file "$MANIFEST_DIR/homebrew/validation.txt"
require_file "$MANIFEST_DIR/scoop/sift.json"
require_file "$MANIFEST_DIR/scoop/validation.txt"
require_file "$MANIFEST_DIR/winget/batu3384.SIFT.yaml"
require_file "$MANIFEST_DIR/winget/batu3384.SIFT.locale.en-US.yaml"
require_file "$MANIFEST_DIR/winget/batu3384.SIFT.installer.yaml"
require_file "$MANIFEST_DIR/winget/validation.txt"

grep -q "version \"${SHORT_VERSION}\"" "$MANIFEST_DIR/homebrew/Sift.rb"
grep -q "\"version\": \"${SHORT_VERSION}\"" "$MANIFEST_DIR/scoop/sift.json"
grep -q "PackageVersion: ${SHORT_VERSION}" "$MANIFEST_DIR/winget/batu3384.SIFT.yaml"
grep -q "package manifests: ok" "$MANIFEST_DIR/validation.txt"
grep -q "homebrew validation: ok" "$MANIFEST_DIR/homebrew/validation.txt"
grep -q "scoop validation: ok" "$MANIFEST_DIR/scoop/validation.txt"
grep -q "winget validation: ok" "$MANIFEST_DIR/winget/validation.txt"

if [[ -n "${SIFT_SIGNING_IDENTITY:-}" && -n "${SIFT_NOTARY_PROFILE:-}" ]]; then
  echo "release preflight: signing context configured"
else
  echo "release preflight: warning - signing context missing (set SIFT_SIGNING_IDENTITY and SIFT_NOTARY_PROFILE to enable signing/notarization)" >&2
fi

echo "release preflight: ok"
