#!/bin/sh
set -eu

REPO="batu3384/sift"
API_URL="https://api.github.com/repos/$REPO/releases/latest"
PREFIX="${PREFIX:-$HOME/.local/bin}"
VERSION="${SIFT_VERSION:-}"
INSTALL_COMPLETIONS="${INSTALL_COMPLETIONS:-1}"

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "install.sh: required command not found: $1" >&2
    exit 1
  }
}

detect_platform() {
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$os" in
    darwin) os="darwin" ;;
    *)
      echo "install.sh: unsupported OS: $os" >&2
      exit 1
      ;;
  esac
  case "$arch" in
    arm64|aarch64) arch="arm64" ;;
    x86_64|amd64) arch="amd64" ;;
    *)
      echo "install.sh: unsupported architecture: $arch" >&2
      exit 1
      ;;
  esac
  printf '%s %s\n' "$os" "$arch"
}

latest_version() {
  curl -fsSL "$API_URL" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1
}

download_asset() {
  version="$1"
  os="$2"
  arch="$3"
  archive="sift_${version#v}_${os}_${arch}.tar.gz"
  url="https://github.com/$REPO/releases/download/$version/$archive"
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT INT TERM
  curl -fsSL "$url" -o "$tmpdir/$archive"
  tar -xzf "$tmpdir/$archive" -C "$tmpdir"
  if [ ! -x "$tmpdir/sift" ]; then
    echo "install.sh: extracted archive does not contain sift binary" >&2
    exit 1
  fi
  mkdir -p "$PREFIX"
  install -m 0755 "$tmpdir/sift" "$PREFIX/sift"
  ln -sf "$PREFIX/sift" "$PREFIX/si"
  if [ "$INSTALL_COMPLETIONS" = "1" ] || [ "$INSTALL_COMPLETIONS" = "true" ]; then
    shell_name="$(basename "${SHELL:-}")"
    case "$shell_name" in
      bash|zsh|fish|pwsh|powershell)
        "$PREFIX/sift" completion "$shell_name" --install >/dev/null 2>&1 || true
        ;;
    esac
  fi
}

need_cmd curl
need_cmd tar
need_cmd install

platform="$(detect_platform)"
os="${platform%% *}"
arch="${platform##* }"

if [ -z "$VERSION" ]; then
  VERSION="$(latest_version)"
fi
if [ -z "$VERSION" ]; then
  echo "install.sh: failed to resolve latest release tag" >&2
  exit 1
fi

download_asset "$VERSION" "$os" "$arch"

echo "Installed sift $VERSION to $PREFIX/sift"
echo "Installed short alias to $PREFIX/si"
echo "Run '$PREFIX/sift version' to verify the install."
case ":$PATH:" in
  *":$PREFIX:"*) ;;
  *) echo "Add $PREFIX to PATH if it is not already exported in your shell profile." ;;
esac
