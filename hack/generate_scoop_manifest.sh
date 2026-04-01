#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

TAG="${1:?usage: generate_scoop_manifest.sh <tag> [dist-dir] [out-dir]}"
DIST_DIR="${2:-$ROOT_DIR/dist}"
OUT_DIR="${3:-$DIST_DIR/scoop}"

SHORT_VERSION="${TAG#v}"
PACKAGE_NAME="sift"
HOMEPAGE="https://github.com/batu3384/sift"
DESCRIPTION="Safety-first terminal cleaner for macOS and Windows"
LICENSE="MIT"

amd64_zip="$DIST_DIR/sift_${TAG}_windows_amd64.zip"
arm64_zip="$DIST_DIR/sift_${TAG}_windows_arm64.zip"

hash_file() {
  local file="$1"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
    return
  fi
  sha256sum "$file" | awk '{print $1}'
}

for archive in "$amd64_zip" "$arm64_zip"; do
  [[ -f "$archive" ]] || { echo "missing archive: $archive" >&2; exit 1; }
done

mkdir -p "$OUT_DIR"

amd64_sha="$(hash_file "$amd64_zip")"
arm64_sha="$(hash_file "$arm64_zip")"
release_base="https://github.com/batu3384/sift/releases/download/${TAG}"
manifest_path="$OUT_DIR/${PACKAGE_NAME}.json"

cat > "$manifest_path" <<EOF
{
  "version": "${SHORT_VERSION}",
  "description": "${DESCRIPTION}",
  "homepage": "${HOMEPAGE}",
  "license": "${LICENSE}",
  "architecture": {
    "64bit": {
      "url": "${release_base}/sift_${TAG}_windows_amd64.zip",
      "hash": "${amd64_sha}"
    },
    "arm64": {
      "url": "${release_base}/sift_${TAG}_windows_arm64.zip",
      "hash": "${arm64_sha}"
    }
  },
  "bin": "sift.exe",
  "checkver": {
    "github": "${HOMEPAGE}"
  },
  "autoupdate": {
    "architecture": {
      "64bit": {
        "url": "${release_base}/sift_${TAG}_windows_amd64.zip"
      },
      "arm64": {
        "url": "${release_base}/sift_${TAG}_windows_arm64.zip"
      }
    }
  }
}
EOF

validation_file="$OUT_DIR/validation.txt"
{
  echo "scoop validation: ok"
  echo "manifest=${manifest_path}"
  echo "package=${PACKAGE_NAME}"
  echo "version=${SHORT_VERSION}"
  echo "amd64_archive=${amd64_zip}"
  echo "arm64_archive=${arm64_zip}"
  echo "amd64_sha=${amd64_sha}"
  echo "arm64_sha=${arm64_sha}"
} > "$validation_file"

[[ -f "$manifest_path" ]] || { echo "missing scoop manifest: $manifest_path" >&2; exit 1; }
if command -v python3 >/dev/null 2>&1; then
  python3 -m json.tool "$manifest_path" >/dev/null
elif command -v python >/dev/null 2>&1; then
  python -m json.tool "$manifest_path" >/dev/null
fi
grep -q "\"hash\": \"${amd64_sha}\"" "$manifest_path"
grep -q "\"hash\": \"${arm64_sha}\"" "$manifest_path"

echo "scoop manifest output written to $OUT_DIR"
