#!/usr/bin/env bash
set -euo pipefail
set -x

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

BINARY="${1:-./bin/sift}"
SMOKE_ROOT="$ROOT_DIR/.tmp/ci-smoke-macos"

rm -rf "$SMOKE_ROOT"
mkdir -p "$SMOKE_ROOT/home" "$SMOKE_ROOT/tmp" "$SMOKE_ROOT/bin" "$SMOKE_ROOT/completions"

export HOME="$SMOKE_ROOT/home"
export TMPDIR="$SMOKE_ROOT/tmp"
if [[ "${SIFT_LIVE_INTEGRATION:-}" == "1" ]]; then
  unset SIFT_TEST_MODE
else
  export SIFT_TEST_MODE="${SIFT_TEST_MODE:-ci-safe}"
fi

assert_json_command() {
  local file="$1"
  local expected="$2"
  python3 - "$file" "$expected" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)

command = data.get("command")
if command != sys.argv[2]:
    raise SystemExit(f"expected command {sys.argv[2]!r}, got {command!r}")
PY
}

assert_plan_has_item() {
  local file="$1"
  local path_fragment="$2"
  local expected_status="$3"
  python3 - "$file" "$path_fragment" "$expected_status" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)

fragment = sys.argv[2]
expected_status = sys.argv[3]
for item in data.get("items", []):
    path = item.get("path", "") or ""
    display = item.get("display_path", "") or ""
    if fragment in path or fragment in display:
        status = item.get("status")
        if status != expected_status:
            raise SystemExit(f"expected status {expected_status!r} for {fragment!r}, got {status!r}")
        raise SystemExit(0)
raise SystemExit(f"expected item containing {fragment!r}")
PY
}

assert_execution_result() {
  local file="$1"
  local path_fragment="$2"
  local expected_status="$3"
  python3 - "$file" "$path_fragment" "$expected_status" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)

fragment = sys.argv[2]
expected_status = sys.argv[3]
for item in data.get("result", {}).get("items", []):
    path = item.get("path", "") or ""
    if fragment in path:
        status = item.get("status")
        if status != expected_status:
            raise SystemExit(f"expected execution status {expected_status!r} for {fragment!r}, got {status!r}")
        raise SystemExit(0)
raise SystemExit(f"expected execution item containing {fragment!r}")
PY
}

assert_execution_warning_contains() {
  local file="$1"
  local needle="$2"
  python3 - "$file" "$needle" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)

needle = sys.argv[2]
warnings = data.get("result", {}).get("warnings", []) or []
if not any(needle in warning for warning in warnings):
    raise SystemExit(f"expected warning containing {needle!r}, got {warnings!r}")
PY
}

assert_plan_warning_contains() {
  local file="$1"
  local needle="$2"
  python3 - "$file" "$needle" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)

needle = sys.argv[2]
warnings = data.get("warnings", []) or []
if not any(needle in warning for warning in warnings):
    raise SystemExit(f"expected plan warning containing {needle!r}, got {warnings!r}")
PY
}

assert_follow_up_contains() {
  local file="$1"
  local needle="$2"
  python3 - "$file" "$needle" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)

needle = sys.argv[2]
commands = data.get("result", {}).get("follow_up_commands", []) or []
if not any(needle in command for command in commands):
    raise SystemExit(f"expected follow-up containing {needle!r}, got {commands!r}")
PY
}

assert_report_exists() {
  local file="$1"
  python3 - "$file" <<'PY'
import json
import os
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)

path = data.get("path", "")
if not path or not os.path.exists(path):
    raise SystemExit(f"expected report path to exist, got {path!r}")
PY
}

run_with_sigkill_retry() {
  local output="$1"
  shift
  local attempt=1
  local max_attempts=3
  while true; do
    if "$@" > "$output"; then
      return 0
    fi
    local rc=$?
    if [[ $rc -ne 137 || $attempt -ge $max_attempts ]]; then
      return $rc
    fi
    attempt=$((attempt + 1))
    sleep 1
  done
}

mkdir -p \
  "$HOME/Applications/Example.app" \
  "$HOME/Applications/Example.app/Contents/MacOS" \
  "$HOME/Projects/keep-me" \
  "$HOME/Library/Caches/Example Cache" \
  "$HOME/Library/Caches/Homebrew/Downloads/pkg" \
  "$HOME/Library/Application Support/Example" \
  "$HOME/Library/Application Support/Google/Chrome/Default/Code Cache/js" \
  "$HOME/Library/Logs/Example" \
  "$HOME/Downloads" \
  "$SMOKE_ROOT/analyze/cache" \
  "$SMOKE_ROOT/project/node_modules/pkg"

printf 'cache\n' > "$SMOKE_ROOT/analyze/cache/junk.txt"
printf 'payload\n' > "$HOME/Library/Caches/Example Cache/file.bin"
printf 'payload\n' > "$HOME/Library/Caches/Homebrew/Downloads/pkg/archive.tgz"
printf 'payload\n' > "$HOME/Library/Application Support/Example/state.bin"
printf 'payload\n' > "$HOME/Library/Application Support/Google/Chrome/Default/Code Cache/js/cache.bin"
printf 'installer\n' > "$HOME/Downloads/Example.pkg"
printf '{}' > "$SMOKE_ROOT/project/package.json"
printf '{}' > "$SMOKE_ROOT/project/node_modules/pkg/package.json"
cat <<EOF > "$HOME/Applications/Example.app/Contents/MacOS/uninstall"
#!/usr/bin/env bash
printf 'ok\n' > "$SMOKE_ROOT/native-uninstall-ran"
EOF
chmod +x "$HOME/Applications/Example.app/Contents/MacOS/uninstall"

run_with_sigkill_retry "$SMOKE_ROOT/help.txt" "$BINARY" --help
"$BINARY" doctor --plain > "$SMOKE_ROOT/doctor.txt"
grep -q 'report_cache' "$SMOKE_ROOT/doctor.txt"
grep -q 'audit_log' "$SMOKE_ROOT/doctor.txt"
grep -q 'purge_search_paths' "$SMOKE_ROOT/doctor.txt"
"$BINARY" protect add "$HOME/Projects/keep-me" > "$SMOKE_ROOT/protect-add.txt"
"$BINARY" protect list > "$SMOKE_ROOT/protect-list.txt"
grep -q "$HOME/Projects/keep-me" "$SMOKE_ROOT/protect-list.txt"
"$BINARY" protect family list > "$SMOKE_ROOT/protect-family-list.txt"
"$BINARY" protect family add browser_profiles > "$SMOKE_ROOT/protect-family-add.txt"
"$BINARY" protect explain --json "$HOME/Library/Application Support/Google/Chrome/Default/History" > "$SMOKE_ROOT/protect-explain-family.json"
grep -q '"state":"safe_exception"' "$SMOKE_ROOT/protect-explain-family.json"
grep -q '"family_matches":\["browser_profiles"\]' "$SMOKE_ROOT/protect-explain-family.json"
"$BINARY" protect family remove browser_profiles > "$SMOKE_ROOT/protect-family-remove.txt"
"$BINARY" protect explain --json "$HOME/Projects/keep-me" > "$SMOKE_ROOT/protect-explain-user.json"
grep -q '"state":"user_protected"' "$SMOKE_ROOT/protect-explain-user.json"
"$BINARY" protect remove "$HOME/Projects/keep-me" > "$SMOKE_ROOT/protect-remove.txt"
"$BINARY" clean --whitelist list > "$SMOKE_ROOT/clean-whitelist-list.txt"
grep -q 'No exclusions configured for clean.' "$SMOKE_ROOT/clean-whitelist-list.txt"
"$BINARY" clean --whitelist add "$HOME/Library/Caches/Homebrew" > "$SMOKE_ROOT/clean-whitelist-add.txt"
"$BINARY" clean --whitelist list > "$SMOKE_ROOT/clean-whitelist-list-after-add.txt"
grep -q "$HOME/Library/Caches/Homebrew" "$SMOKE_ROOT/clean-whitelist-list-after-add.txt"
"$BINARY" clean --whitelist remove "$HOME/Library/Caches/Homebrew" > "$SMOKE_ROOT/clean-whitelist-remove.txt"
"$BINARY" optimize --whitelist add "$HOME/Library/Caches/Homebrew" > "$SMOKE_ROOT/optimize-whitelist-add.txt"
"$BINARY" optimize --whitelist list > "$SMOKE_ROOT/optimize-whitelist-list.txt"
grep -q "$HOME/Library/Caches/Homebrew" "$SMOKE_ROOT/optimize-whitelist-list.txt"
"$BINARY" optimize --whitelist remove "$HOME/Library/Caches/Homebrew" > "$SMOKE_ROOT/optimize-whitelist-remove.txt"
"$BINARY" analyze --plain "$SMOKE_ROOT/analyze"
"$BINARY" clean --json --profile safe > "$SMOKE_ROOT/clean.json"
assert_json_command "$SMOKE_ROOT/clean.json" "clean"
"$BINARY" optimize --json > "$SMOKE_ROOT/optimize.json"
assert_json_command "$SMOKE_ROOT/optimize.json" "optimize"
"$BINARY" clean --json --profile deep > "$SMOKE_ROOT/clean-deep.json"
assert_plan_has_item "$SMOKE_ROOT/clean-deep.json" "Application Support/Google/Chrome/Default" "planned"
assert_plan_has_item "$SMOKE_ROOT/clean-deep.json" "Library/Caches/Homebrew/Downloads" "planned"
"$BINARY" protect explain --json "$HOME/Library/Application Support/Google/Chrome/Default/Code Cache/js" > "$SMOKE_ROOT/protect-explain-safe.json"
grep -q '"state":"unprotected"' "$SMOKE_ROOT/protect-explain-safe.json"
grep -q '"exception_matches":\[' "$SMOKE_ROOT/protect-explain-safe.json"
"$BINARY" purge --json "$SMOKE_ROOT/project/node_modules" > "$SMOKE_ROOT/purge.json"
assert_json_command "$SMOKE_ROOT/purge.json" "purge"
"$BINARY" purge scan --json "$SMOKE_ROOT/project" > "$SMOKE_ROOT/purge-scan.json"
assert_json_command "$SMOKE_ROOT/purge-scan.json" "purge_scan"
assert_plan_has_item "$SMOKE_ROOT/purge-scan.json" "node_modules" "planned"
"$BINARY" uninstall --json Example > "$SMOKE_ROOT/uninstall.json"
assert_json_command "$SMOKE_ROOT/uninstall.json" "uninstall"
assert_plan_has_item "$SMOKE_ROOT/uninstall.json" "Contents/MacOS/uninstall" "planned"
"$BINARY" update --plain > "$SMOKE_ROOT/update.txt"
grep -q 'Install method:' "$SMOKE_ROOT/update.txt"
grep -q 'Channel: stable' "$SMOKE_ROOT/update.txt"
"$BINARY" update --plain --channel nightly > "$SMOKE_ROOT/update-nightly.txt"
grep -q 'Channel: nightly' "$SMOKE_ROOT/update-nightly.txt"
"$BINARY" touchid --plain > "$SMOKE_ROOT/touchid.txt"
grep -q 'Supported:' "$SMOKE_ROOT/touchid.txt"
"$BINARY" touchid enable --json > "$SMOKE_ROOT/touchid-enable.json"
grep -q '"action":"enable"' "$SMOKE_ROOT/touchid-enable.json"
"$BINARY" remove --json > "$SMOKE_ROOT/remove.json"
assert_json_command "$SMOKE_ROOT/remove.json" "remove"
"$BINARY" uninstall --json --dry-run=false --yes --native-uninstall Example > "$SMOKE_ROOT/uninstall-exec.json"
assert_execution_warning_contains "$SMOKE_ROOT/uninstall-exec.json" "continued with remnant cleanup and aftercare"
assert_execution_result "$SMOKE_ROOT/uninstall-exec.json" "Contents/MacOS/uninstall" "completed"
test ! -e "$HOME/Library/Application Support/Example"
rm -rf "$HOME/Applications/Example.app"
"$BINARY" uninstall --json Example > "$SMOKE_ROOT/uninstall-rerun.json"
assert_json_command "$SMOKE_ROOT/uninstall-rerun.json" "uninstall"
assert_plan_warning_contains "$SMOKE_ROOT/uninstall-rerun.json" "No installed app or leftover files were found for Example."
"$BINARY" status --plain > "$SMOKE_ROOT/status.txt"
grep -q '^System:' "$SMOKE_ROOT/status.txt"
grep -q '^Health ' "$SMOKE_ROOT/status.txt"
grep -q '^Audit log:' "$SMOKE_ROOT/status.txt"
"$BINARY" completion bash > "$SMOKE_ROOT/completions/sift.bash"
"$BINARY" completion zsh > "$SMOKE_ROOT/completions/_sift"
"$BINARY" completion fish > "$SMOKE_ROOT/completions/sift.fish"
"$BINARY" completion powershell > "$SMOKE_ROOT/completions/sift.ps1"
"$BINARY" report --json > "$SMOKE_ROOT/report.json"
assert_report_exists "$SMOKE_ROOT/report.json"
