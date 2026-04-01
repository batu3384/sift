#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

TAG="${1:?usage: generate_package_manifests.sh <tag> [dist-dir] [out-dir]}"
DIST_DIR="${2:-$ROOT_DIR/dist}"
OUT_DIR="${3:-$DIST_DIR/manifests}"

mkdir -p "$OUT_DIR"

./hack/generate_homebrew_formula.sh "$TAG" "$DIST_DIR" "$OUT_DIR/homebrew"
./hack/generate_scoop_manifest.sh "$TAG" "$DIST_DIR" "$OUT_DIR/scoop"
./hack/generate_winget_manifest.sh "$TAG" "$DIST_DIR" "$OUT_DIR/winget"

validation_file="$OUT_DIR/validation.txt"
{
  echo "package manifests: ok"
  echo "tag=${TAG}"
  echo "dist_dir=${DIST_DIR}"
  echo "output_dir=${OUT_DIR}"
} > "$validation_file"

echo "package manifests written to $OUT_DIR"
