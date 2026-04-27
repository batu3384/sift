#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

SEARCH_PATHS=(internal cmd hack .github/workflows)

fail() {
  echo "security-check: $1" >&2
  exit 1
}

allow_only_in() {
  local pattern="$1"
  local allowed_file="$2"
  local matches
  matches="$(rg -n "$pattern" "${SEARCH_PATHS[@]}" || true)"
  if [[ -z "$matches" ]]; then
    return 0
  fi
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    local file="${line%%:*}"
    if [[ "$file" != "$allowed_file" ]]; then
      fail "unexpected match for '$pattern': $line"
    fi
  done <<<"$matches"
}

allow_only_in_any() {
  local pattern="$1"
  shift
  local matches
  matches="$(rg -n "$pattern" "${SEARCH_PATHS[@]}" || true)"
  if [[ -z "$matches" ]]; then
    return 0
  fi
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    local file="${line%%:*}"
    local allowed=0
    for allowed_file in "$@"; do
      if [[ "$file" == "$allowed_file" ]]; then
        allowed=1
        break
      fi
    done
    if [[ "$allowed" -ne 1 ]]; then
      fail "unexpected match for '$pattern': $line"
    fi
  done <<<"$matches"
}

reject_any() {
  local pattern="$1"
  local matches
  matches="$(rg -n "$pattern" "${SEARCH_PATHS[@]}" || true)"
  if [[ -n "$matches" ]]; then
    fail "forbidden pattern '$pattern' found:\n$matches"
  fi
}

reject_any_except() {
  local pattern="$1"
  shift
  local matches
  matches="$(rg -n "$pattern" "${SEARCH_PATHS[@]}" || true)"
  if [[ -z "$matches" ]]; then
    return 0
  fi
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    local file="${line%%:*}"
    local allowed=0
    for allowed_file in "$@"; do
      if [[ "$file" == "$allowed_file" ]]; then
        allowed=1
        break
      fi
    done
    if [[ "$allowed" -ne 1 ]]; then
      fail "forbidden pattern '$pattern' found: $line"
    fi
  done <<<"$matches"
}

allow_only_in_any 'os\.RemoveAll' 'internal/engine/service.go' 'internal/engine/execution.go'
allow_only_in_any 'exec\.Command(Context)?' 'internal/platform/current_darwin.go' 'internal/platform/current_darwin_apps.go' 'internal/platform/current_darwin_diagnostics.go' 'internal/platform/admin_session.go' 'internal/platform/admin_session_runtime.go' 'internal/engine/native.go' 'internal/engine/health_darwin.go' 'internal/tui/open_darwin.go' 'internal/tui/open_windows.go' 'internal/tui/preflight.go'

reject_any 'exec\.Command(Context)?\([^)]*"cmd(\.exe)?"'
reject_any 'exec\.Command(Context)?\([^)]*"powershell(\.exe)?"'
reject_any 'exec\.Command(Context)?\([^)]*"pwsh(\.exe)?"'
reject_any 'exec\.Command(Context)?\([^)]*"sh"'
reject_any 'exec\.Command(Context)?\([^)]*"bash"'
reject_any 'exec\.Command(Context)?\([^)]*"zsh"'
reject_any_except 'rm -rf' 'hack/security_check.sh' 'hack/install_dev_tools.sh' 'hack/macos_smoke.sh' 'hack/refresh_mole_upstream.sh'
reject_any_except 'Remove-Item\s+-Recurse\s+-Force' 'hack/windows_smoke.ps1'

echo "security-check: ok"
