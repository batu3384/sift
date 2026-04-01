#!/bin/bash

set -euo pipefail

SCRIPT_NAME="SIFT Quick Launchers"
COMMAND_SPECS=(
  "clean|SIFT Clean|Run the interactive cleanup review"
  "uninstall|SIFT Uninstall|Open the uninstall workflow"
  "optimize|SIFT Optimize|Review maintenance and optimize tasks"
  "analyze|SIFT Analyze|Open the disk analysis explorer"
  "status|SIFT Status|Open the live status dashboard"
)

detect_sift() {
  if [[ $# -gt 0 && -x "$1" ]]; then
    printf '%s\n' "$1"
    return 0
  fi
  if command -v sift >/dev/null 2>&1; then
    command -v sift
    return 0
  fi
  printf 'Unable to find a SIFT binary. Pass the absolute path as the first argument.\n' >&2
  exit 1
}

uuid_value() {
  if command -v uuidgen >/dev/null 2>&1; then
    uuidgen | tr '[:upper:]' '[:lower:]'
    return 0
  fi
  local hex
  hex=$(openssl rand -hex 16)
  printf '%s-%s-%s-%s-%s\n' "${hex:0:8}" "${hex:8:4}" "${hex:12:4}" "${hex:16:4}" "${hex:20:12}"
}

install_raycast() {
  local sift_bin="$1"
  local dir="${RAYCAST_SCRIPT_DIR:-$HOME/Library/Application Support/Raycast/script-commands}"
  mkdir -p "$dir"
  local spec subcommand title description target
  for spec in "${COMMAND_SPECS[@]}"; do
    IFS='|' read -r subcommand title description <<<"$spec"
    target="$dir/sift-${subcommand}.sh"
    cat >"$target" <<EOF
#!/bin/bash

# Required parameters:
# @raycast.schemaVersion 1
# @raycast.title ${title}
# @raycast.mode fullOutput
# @raycast.packageName SIFT
# @raycast.description ${description}
# @raycast.icon terminal

exec "$sift_bin" "$subcommand"
EOF
    chmod +x "$target"
  done
}

install_alfred() {
  local sift_bin="$1"
  local prefs_dir="${ALFRED_PREFS_DIR:-$HOME/Library/Application Support/Alfred/Alfred.alfredpreferences}"
  local workflows_dir="$prefs_dir/workflows"
  [[ -d "$workflows_dir" ]] || return 0

  local spec subcommand title description bundle workflow_uid input_uid action_uid dir
  for spec in "${COMMAND_SPECS[@]}"; do
    IFS='|' read -r subcommand title description <<<"$spec"
    bundle="com.batu3384.sift.${subcommand}"
    workflow_uid="user.workflow.$(uuid_value)"
    input_uid="$(uuid_value)"
    action_uid="$(uuid_value)"
    dir="$workflows_dir/$workflow_uid"
    mkdir -p "$dir"
    cat >"$dir/info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>bundleid</key>
  <string>${bundle}</string>
  <key>createdby</key>
  <string>SIFT</string>
  <key>name</key>
  <string>${title}</string>
  <key>objects</key>
  <array>
    <dict>
      <key>config</key>
      <dict>
        <key>argumenttype</key>
        <integer>2</integer>
        <key>keyword</key>
        <string>${subcommand}</string>
        <key>subtext</key>
        <string>${description}</string>
        <key>text</key>
        <string>${title}</string>
        <key>withspace</key>
        <true/>
      </dict>
      <key>type</key>
      <string>alfred.workflow.input.keyword</string>
      <key>uid</key>
      <string>${input_uid}</string>
      <key>version</key>
      <integer>1</integer>
    </dict>
    <dict>
      <key>config</key>
      <dict>
        <key>concurrently</key>
        <true/>
        <key>escaping</key>
        <integer>102</integer>
        <key>script</key>
        <string>#!/bin/bash
PATH="/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin"
exec "$sift_bin" "$subcommand"</string>
        <key>scriptargtype</key>
        <integer>1</integer>
        <key>scriptfile</key>
        <string></string>
        <key>type</key>
        <integer>0</integer>
      </dict>
      <key>type</key>
      <string>alfred.workflow.action.script</string>
      <key>uid</key>
      <string>${action_uid}</string>
      <key>version</key>
      <integer>2</integer>
    </dict>
  </array>
  <key>connections</key>
  <dict>
    <key>${input_uid}</key>
    <array>
      <dict>
        <key>destinationuid</key>
        <string>${action_uid}</string>
        <key>modifiers</key>
        <integer>0</integer>
        <key>modifiersubtext</key>
        <string></string>
      </dict>
    </array>
  </dict>
  <key>uid</key>
  <string>${workflow_uid}</string>
  <key>version</key>
  <integer>1</integer>
</dict>
</plist>
EOF
  done
}

main() {
  local sift_bin
  sift_bin="$(detect_sift "${1:-}")"
  printf '%s\n' "$SCRIPT_NAME"
  printf 'Detected SIFT at %s\n' "$sift_bin"
  install_raycast "$sift_bin"
  install_alfred "$sift_bin"
  printf 'Installed Raycast command scripts and Alfred workflows for clean, uninstall, optimize, analyze, and status.\n'
  printf 'Raycast may still need "Reload Script Directories" after first install.\n'
}

main "$@"
