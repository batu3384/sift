#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TMP_DIR="$ROOT_DIR/.tmp"
TARGET_DIR="$TMP_DIR/mole-upstream.latest"
UPSTREAM_URL="https://github.com/tw93/mole"

mkdir -p "$TMP_DIR"
find "$TMP_DIR" -maxdepth 1 -mindepth 1 -type d -name 'mole-upstream*' -prune -exec rm -rf {} +
git clone --depth=1 "$UPSTREAM_URL" "$TARGET_DIR" >/dev/null

printf 'Canonical clone: %s\n' "$TARGET_DIR"
printf 'HEAD: %s\n' "$(git -C "$TARGET_DIR" rev-parse HEAD)"
printf 'Date: %s\n' "$(git -C "$TARGET_DIR" log -1 --format=%cI)"
