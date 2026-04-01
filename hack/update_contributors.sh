#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

tmp_file="$(mktemp)"
contributors="$(git shortlog -sne HEAD 2>/dev/null || git shortlog -sne 2>/dev/null || true)"
{
  echo "# Contributors"
  echo
  echo "Generated from git history. Run \`./hack/update_contributors.sh\` to refresh."
  echo
  if [[ -n "${contributors//[[:space:]]/}" ]]; then
    while IFS= read -r line; do
      [[ -z "$line" ]] && continue
      count="${line%%$'\t'*}"
      identity="${line#*$'\t'}"
      identity="${identity#"${identity%%[![:space:]]*}"}"
      [[ -z "$identity" || "$identity" == "$line" ]] && continue
      echo "- ${identity} (${count} commit(s))"
    done <<<"$contributors"
  else
    echo "- No git history available in this workspace snapshot."
  fi
  echo
} >"$tmp_file"

mv "$tmp_file" CONTRIBUTORS.md
