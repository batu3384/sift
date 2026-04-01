#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

TAG="${1:?usage: generate_winget_manifest.sh <tag> [dist-dir] [out-dir]}"
DIST_DIR="${2:-$ROOT_DIR/dist}"
OUT_DIR="${3:-$DIST_DIR/winget}"

PACKAGE_IDENTIFIER="batuhanyuksel.SIFT"
PACKAGE_NAME="SIFT"
PUBLISHER="batuhanyuksel"
LICENSE="MIT"
SHORT_VERSION="${TAG#v}"

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
release_base="https://github.com/batuhanyuksel/sift/releases/download/${TAG}"

cat > "$OUT_DIR/${PACKAGE_IDENTIFIER}.yaml" <<EOF
PackageIdentifier: ${PACKAGE_IDENTIFIER}
PackageVersion: ${SHORT_VERSION}
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.6.0
EOF

cat > "$OUT_DIR/${PACKAGE_IDENTIFIER}.locale.en-US.yaml" <<EOF
PackageIdentifier: ${PACKAGE_IDENTIFIER}
PackageVersion: ${SHORT_VERSION}
PackageLocale: en-US
Publisher: ${PUBLISHER}
PackageName: ${PACKAGE_NAME}
ShortDescription: Safety-first terminal cleaner for macOS and Windows
License: ${LICENSE}
ManifestType: defaultLocale
ManifestVersion: 1.6.0
EOF

cat > "$OUT_DIR/${PACKAGE_IDENTIFIER}.installer.yaml" <<EOF
PackageIdentifier: ${PACKAGE_IDENTIFIER}
PackageVersion: ${SHORT_VERSION}
Installers:
  - Architecture: x64
    InstallerType: zip
    NestedInstallerType: portable
    NestedInstallerFiles:
      - RelativeFilePath: sift.exe
        PortableCommandAlias: sift
    InstallerUrl: ${release_base}/sift_${TAG}_windows_amd64.zip
    InstallerSha256: ${amd64_sha}
  - Architecture: arm64
    InstallerType: zip
    NestedInstallerType: portable
    NestedInstallerFiles:
      - RelativeFilePath: sift.exe
        PortableCommandAlias: sift
    InstallerUrl: ${release_base}/sift_${TAG}_windows_arm64.zip
    InstallerSha256: ${arm64_sha}
ManifestType: installer
ManifestVersion: 1.6.0
EOF

validation_file="$OUT_DIR/validation.txt"
{
  echo "winget validation: ok"
  echo "package_identifier=${PACKAGE_IDENTIFIER}"
  echo "package_version=${SHORT_VERSION}"
  echo "amd64_archive=${amd64_zip}"
  echo "arm64_archive=${arm64_zip}"
  echo "amd64_sha=${amd64_sha}"
  echo "arm64_sha=${arm64_sha}"
} > "$validation_file"

for file in \
  "$OUT_DIR/${PACKAGE_IDENTIFIER}.yaml" \
  "$OUT_DIR/${PACKAGE_IDENTIFIER}.locale.en-US.yaml" \
  "$OUT_DIR/${PACKAGE_IDENTIFIER}.installer.yaml" \
  "$validation_file"; do
  [[ -f "$file" ]] || { echo "missing generated file: $file" >&2; exit 1; }
done

grep -q "PackageIdentifier: ${PACKAGE_IDENTIFIER}" "$OUT_DIR/${PACKAGE_IDENTIFIER}.installer.yaml"
grep -q "InstallerSha256: ${amd64_sha}" "$OUT_DIR/${PACKAGE_IDENTIFIER}.installer.yaml"
grep -q "InstallerSha256: ${arm64_sha}" "$OUT_DIR/${PACKAGE_IDENTIFIER}.installer.yaml"
echo "winget manifest output written to $OUT_DIR"
