#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TOOL_ROOT="${TOOL_ROOT:-$ROOT_DIR/.tmp/tools}"
BIN_DIR="$TOOL_ROOT/bin"
CACHE_DIR="$TOOL_ROOT/cache"

STATICCHECK_VERSION="${STATICCHECK_VERSION:-v0.7.0}"
SHELLCHECK_VERSION="${SHELLCHECK_VERSION:-0.10.0}"

mkdir -p "$BIN_DIR" "$CACHE_DIR"

install_staticcheck() {
	if [[ -x "$BIN_DIR/staticcheck" ]] && "$BIN_DIR/staticcheck" -version 2>/dev/null | grep -q "staticcheck ${STATICCHECK_VERSION}"; then
		return
	fi

	GOBIN="$BIN_DIR" go install "honnef.co/go/tools/cmd/staticcheck@${STATICCHECK_VERSION}"
}

shellcheck_platform() {
	case "$(uname -s):$(uname -m)" in
	Darwin:arm64)
		printf '%s\n' "darwin.aarch64"
		;;
	Darwin:x86_64)
		printf '%s\n' "darwin.x86_64"
		;;
	Linux:arm64 | Linux:aarch64)
		printf '%s\n' "linux.aarch64"
		;;
	Linux:x86_64)
		printf '%s\n' "linux.x86_64"
		;;
	*)
		printf 'unsupported platform for shellcheck bootstrap: %s/%s\n' "$(uname -s)" "$(uname -m)" >&2
		exit 1
		;;
	esac
}

install_shellcheck() {
	if [[ -x "$BIN_DIR/shellcheck" ]] && "$BIN_DIR/shellcheck" --version 2>/dev/null | grep -q "version: ${SHELLCHECK_VERSION}"; then
		return
	fi

	local platform archive url extract_dir temp_dir
	platform="$(shellcheck_platform)"
	archive="$CACHE_DIR/shellcheck-v${SHELLCHECK_VERSION}.${platform}.tar.xz"
	url="https://github.com/koalaman/shellcheck/releases/download/v${SHELLCHECK_VERSION}/shellcheck-v${SHELLCHECK_VERSION}.${platform}.tar.xz"
	if [[ ! -f "$archive" ]]; then
		curl -fsSL "$url" -o "$archive"
	fi

	temp_dir="$(mktemp -d)"
	trap 'rm -rf "$temp_dir"' RETURN
	tar -xJf "$archive" -C "$temp_dir"
	extract_dir="$temp_dir/shellcheck-v${SHELLCHECK_VERSION}"
	cp "$extract_dir"/shellcheck "$BIN_DIR/shellcheck"
	chmod +x "$BIN_DIR/shellcheck"
	rm -rf "$temp_dir"
	trap - RETURN
}

usage() {
	cat <<'EOF'
usage: hack/install_dev_tools.sh [staticcheck|shellcheck|all]
EOF
}

case "${1:-all}" in
staticcheck)
	install_staticcheck
	;;
shellcheck)
	install_shellcheck
	;;
all)
	install_staticcheck
	install_shellcheck
	;;
*)
	usage
	exit 1
	;;
esac
