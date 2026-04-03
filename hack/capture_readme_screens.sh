#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

OUT_DIR="${1:-$ROOT_DIR/docs/assets/screenshots}"
FIXTURE_ROOT="$ROOT_DIR/.tmp/readme-fixture"
FIXTURE_HOME="$FIXTURE_ROOT/home"
FIXTURE_TMP="$FIXTURE_ROOT/tmp"
ANALYZE_ROOT="$FIXTURE_ROOT/analyze"

mkdir -p "$OUT_DIR" \
	"$FIXTURE_HOME/Library/Caches/SiftDemo" \
	"$FIXTURE_TMP" \
	"$ANALYZE_ROOT/cache" \
	"$ANALYZE_ROOT/logs"

printf 'cache\n' > "$FIXTURE_HOME/Library/Caches/SiftDemo/data.tmp"
printf 'alpha\n' > "$ANALYZE_ROOT/cache/a.tmp"
printf 'beta\n' > "$ANALYZE_ROOT/logs/b.log"

open_terminal_screen() {
	local title="$1"
	local wait_seconds="$2"
	local sift_command="$3"

	local shell_command
	shell_command=$(
		cat <<EOF
cd "$ROOT_DIR"
export HOME="$FIXTURE_HOME" TMPDIR="$FIXTURE_TMP" SIFT_TEST_MODE=ci-safe SIFT_REDUCED_MOTION=1 TERM=xterm-256color
mkdir -p "$FIXTURE_TMP"
stty cols 120 rows 34
clear
$sift_command
EOF
	)

	osascript \
		-e 'on run argv' \
		-e 'set cmdText to item 1 of argv' \
		-e 'tell application "Terminal" to activate' \
		-e 'tell application "Terminal" to do script cmdText' \
		-e 'end run' \
		-- "$shell_command" >/dev/null

	sleep "$wait_seconds"
}

capture_front_window() {
	local out_file="$1"
	local bounds
	bounds="$(osascript -e 'tell application "Terminal" to get bounds of front window')"

	local x y right bottom width height
	IFS=', ' read -r x y right bottom <<<"$bounds"
	width=$((right - x))
	height=$((bottom - y))
	screencapture -R"${x},${y},${width},${height}" "$out_file"
}

close_front_window() {
	osascript -e 'tell application "System Events" to keystroke "q"' >/dev/null
	sleep 1
	osascript -e 'tell application "Terminal" to close front window saving no' >/dev/null
	sleep 1
}

capture_route() {
	local slug="$1"
	local title="$2"
	local wait_seconds="$3"
	local sift_command="$4"

	open_terminal_screen "$title" "$wait_seconds" "$sift_command"
	capture_front_window "$OUT_DIR/$slug.png"
	close_front_window
}

capture_route "home" "SIFT README Home" 2 "./sift"
capture_route "analyze" "SIFT README Analyze" 2 "./sift analyze \"$ANALYZE_ROOT\""
capture_route "review" "SIFT README Review" 4 "./sift clean safe"

echo "Wrote screenshots to $OUT_DIR"
