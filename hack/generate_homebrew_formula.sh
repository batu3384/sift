#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

TAG="${1:?usage: generate_homebrew_formula.sh <tag> [dist-dir] [out-dir]}"
DIST_DIR="${2:-$ROOT_DIR/dist}"
OUT_DIR="${3:-$DIST_DIR/homebrew}"

SHORT_VERSION="${TAG#v}"
FORMULA_NAME="Sift"
FORMULA_CLASS="Sift"
PACKAGE_NAME="sift"
HOMEPAGE="https://github.com/batuhanyuksel/sift"
DESCRIPTION="Safety-first terminal cleaner for macOS and Windows"
LICENSE="MIT"

amd64_archive="$DIST_DIR/sift_${TAG}_darwin_amd64.tar.gz"
arm64_archive="$DIST_DIR/sift_${TAG}_darwin_arm64.tar.gz"

hash_file() {
  local file="$1"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
    return
  fi
  sha256sum "$file" | awk '{print $1}'
}

for archive in "$amd64_archive" "$arm64_archive"; do
  [[ -f "$archive" ]] || { echo "missing archive: $archive" >&2; exit 1; }
done

mkdir -p "$OUT_DIR"

amd64_sha="$(hash_file "$amd64_archive")"
arm64_sha="$(hash_file "$arm64_archive")"
release_base="https://github.com/batuhanyuksel/sift/releases/download/${TAG}"
formula_path="$OUT_DIR/${FORMULA_NAME}.rb"

cat > "$formula_path" <<EOF
class ${FORMULA_CLASS} < Formula
  desc "${DESCRIPTION}"
  homepage "${HOMEPAGE}"
  version "${SHORT_VERSION}"
  license "${LICENSE}"

  on_macos do
    if Hardware::CPU.arm?
      url "${release_base}/sift_${TAG}_darwin_arm64.tar.gz"
      sha256 "${arm64_sha}"
    else
      url "${release_base}/sift_${TAG}_darwin_amd64.tar.gz"
      sha256 "${amd64_sha}"
    end
  end

  def install
    bin.install "${PACKAGE_NAME}"
  end

  test do
    assert_match "Safety-first terminal cleaner", shell_output("\#{bin}/${PACKAGE_NAME} --help")
  end
end
EOF

validation_file="$OUT_DIR/validation.txt"
{
  echo "homebrew validation: ok"
  echo "formula=${formula_path}"
  echo "package=${PACKAGE_NAME}"
  echo "version=${SHORT_VERSION}"
  echo "amd64_archive=${amd64_archive}"
  echo "arm64_archive=${arm64_archive}"
  echo "amd64_sha=${amd64_sha}"
  echo "arm64_sha=${arm64_sha}"
} > "$validation_file"

[[ -f "$formula_path" ]] || { echo "missing formula: $formula_path" >&2; exit 1; }
grep -q "sha256 \"${amd64_sha}\"" "$formula_path"
grep -q "sha256 \"${arm64_sha}\"" "$formula_path"
if command -v ruby >/dev/null 2>&1; then
  ruby -c "$formula_path" >/dev/null
fi

echo "homebrew formula output written to $OUT_DIR"
