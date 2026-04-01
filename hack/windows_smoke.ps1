param(
  [string]$Binary = ".\bin\sift.exe"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$Root = Join-Path (Get-Location) ".tmp\ci-smoke-windows"
if (Test-Path $Root) {
  Remove-Item -Recurse -Force $Root
}

$paths = @(
  $Root,
  (Join-Path $Root "home"),
  (Join-Path $Root "home\AppData"),
  (Join-Path $Root "home\AppData\Local"),
  (Join-Path $Root "home\AppData\Roaming"),
  (Join-Path $Root "ProgramData"),
  (Join-Path $Root "Temp"),
  (Join-Path $Root "home\Projects\keep-me"),
  (Join-Path $Root "analyze\cache"),
  (Join-Path $Root "project\node_modules\pkg"),
  (Join-Path $Root "home\AppData\Local\Programs\Example App"),
  (Join-Path $Root "home\AppData\Local\Google\Chrome\User Data\Default\Code Cache\js"),
  (Join-Path $Root "ProgramData\chocolatey\cache\pkg"),
  (Join-Path $Root "home\Downloads"),
  (Join-Path $Root "completions")
)
foreach ($path in $paths) {
  New-Item -ItemType Directory -Force -Path $path | Out-Null
}

$env:USERPROFILE = Join-Path $Root "home"
$env:HOME = $env:USERPROFILE
$env:LOCALAPPDATA = Join-Path $env:USERPROFILE "AppData\Local"
$env:APPDATA = Join-Path $env:USERPROFILE "AppData\Roaming"
$env:ProgramData = Join-Path $Root "ProgramData"
$env:TEMP = Join-Path $Root "Temp"
$env:TMP = $env:TEMP
$NativeSentinel = Join-Path $Root "native-uninstall-ran"
$HelperSource = Join-Path $Root "uninstall_helper.go"

New-Item -ItemType Directory -Force -Path (Join-Path $env:LOCALAPPDATA "Temp"), (Join-Path $env:LOCALAPPDATA "npm-cache"), (Join-Path $env:APPDATA "Example App"), (Join-Path $env:ProgramData "Example Logs") | Out-Null
Set-Content -Path (Join-Path $Root "analyze\cache\junk.txt") -Value 'cache'
Set-Content -Path (Join-Path $env:LOCALAPPDATA "Temp\cache.txt") -Value 'cache'
Set-Content -Path (Join-Path $env:LOCALAPPDATA "Google\Chrome\User Data\Default\Code Cache\js\cache.bin") -Value 'cache'
Set-Content -Path (Join-Path $env:ProgramData "chocolatey\cache\pkg\archive.nupkg") -Value 'pkg'
Set-Content -Path (Join-Path $env:LOCALAPPDATA "Programs\Example App\app.exe") -Value 'binary'
Set-Content -Path (Join-Path $env:USERPROFILE "Downloads\setup.msi") -Value 'installer'
Set-Content -Path (Join-Path $Root "project\package.json") -Value '{}'
Set-Content -Path (Join-Path $Root "project\node_modules\pkg\package.json") -Value '{}'
Set-Content -Path (Join-Path $env:APPDATA "Example App\state.bin") -Value 'payload'
$helperLiteral = '"' + $NativeSentinel.Replace('\', '\\') + '"'
$helperContent = @'
package main

import (
  "os"
  "path/filepath"
)

func main() {
  target := __TARGET__
  _ = os.MkdirAll(filepath.Dir(target), 0o755)
  _ = os.WriteFile(target, []byte("ok\n"), 0o644)
}
'@
$helperContent = $helperContent.Replace('__TARGET__', $helperLiteral)
Set-Content -Path $HelperSource -Value $helperContent
go build -o (Join-Path $env:LOCALAPPDATA "Programs\Example App\uninstall.exe") $HelperSource

& $Binary --help | Out-Null
$doctor = & $Binary doctor --plain
$doctor | Out-File -Encoding utf8 (Join-Path $Root "doctor.txt")
if (-not ($doctor -match 'report_cache')) {
  throw "expected report_cache diagnostic in doctor output"
}
if (-not ($doctor -match 'audit_log')) {
  throw "expected audit_log diagnostic in doctor output"
}
if (-not ($doctor -match 'purge_search_paths')) {
  throw "expected purge_search_paths diagnostic in doctor output"
}
& $Binary protect add (Join-Path $env:USERPROFILE "Projects\keep-me") | Out-Null
$protectList = & $Binary protect list
if (-not ($protectList -match 'Projects[\\/]keep-me')) {
  throw "expected protected path in protect list output"
}
$protectFamilyList = & $Binary protect family list
if (-not ($protectFamilyList -match 'browser_profiles')) {
  throw "expected browser_profiles in protected family list"
}
& $Binary protect family add browser_profiles | Out-Null
$protectExplainFamily = & $Binary protect explain --json (Join-Path $env:LOCALAPPDATA "Google\Chrome\User Data\Default\History") | ConvertFrom-Json
if ($protectExplainFamily.state -ne 'user_protected') {
  throw "expected user_protected family explanation state"
}
if (-not ($protectExplainFamily.family_matches -contains 'browser_profiles')) {
  throw "expected browser_profiles family match"
}
& $Binary protect family remove browser_profiles | Out-Null
$protectExplainUser = & $Binary protect explain --json (Join-Path $env:USERPROFILE "Projects\keep-me") | ConvertFrom-Json
if ($protectExplainUser.state -ne 'user_protected') {
  throw "expected user protected explanation state"
}
& $Binary protect remove (Join-Path $env:USERPROFILE "Projects\keep-me") | Out-Null
& $Binary analyze --plain (Join-Path $Root "analyze")

$optimize = & $Binary optimize --json | ConvertFrom-Json
if ($optimize.command -ne 'optimize') {
  throw "expected optimize command in JSON output"
}

$clean = & $Binary clean --json --profile safe | ConvertFrom-Json
if ($clean.command -ne 'clean') {
  throw "expected clean command in JSON output"
}
$deepClean = & $Binary clean --json --profile deep | ConvertFrom-Json
$chromeCache = $deepClean.items | Where-Object { $_.path -match 'Google[\\/]+Chrome[\\/]+User Data[\\/]+Default[\\/]+Code Cache(?:$|[\\/])' } | Select-Object -First 1
if (-not $chromeCache) {
  throw "expected Chrome code cache finding in deep clean output: $($deepClean.items | ConvertTo-Json -Compress)"
}
if ($chromeCache.status -ne 'planned') {
  throw "expected planned Chrome code cache finding, got $($chromeCache | ConvertTo-Json -Compress)"
}
$chocoCache = $deepClean.items | Where-Object { $_.path -match 'chocolatey[\\/]+cache(?:$|[\\/])' } | Select-Object -First 1
if (-not $chocoCache) {
  throw "expected chocolatey cache finding in deep clean output: $($deepClean.items | ConvertTo-Json -Compress)"
}
if ($chocoCache.status -ne 'protected') {
  throw "expected protected chocolatey cache finding, got $($chocoCache | ConvertTo-Json -Compress)"
}
if (-not $chocoCache.requires_admin -or $chocoCache.policy.reason -ne 'admin_required') {
  throw "expected admin-required chocolatey cache policy, got $($chocoCache | ConvertTo-Json -Compress)"
}
$protectExplainSafe = & $Binary protect explain --json (Join-Path $env:LOCALAPPDATA "Google\Chrome\User Data\Default\Code Cache\js") | ConvertFrom-Json
if ($protectExplainSafe.state -ne 'safe_exception') {
  throw "expected safe_exception explanation state"
}

$purge = & $Binary purge --json (Join-Path $Root "project\node_modules") | ConvertFrom-Json
if ($purge.command -ne 'purge') {
  throw "expected purge command in JSON output"
}
$purgeScan = & $Binary purge scan --json (Join-Path $Root "project") | ConvertFrom-Json
if ($purgeScan.command -ne 'purge_scan') {
  throw "expected purge scan command in JSON output"
}

$uninstallRaw = & $Binary uninstall --json "Example App"
$uninstallRaw | Out-File -Encoding utf8 (Join-Path $Root "uninstall-plan.json")
$uninstall = $uninstallRaw | ConvertFrom-Json
if ($uninstall.command -ne 'uninstall') {
  throw "expected uninstall command in JSON output"
}
if (-not ($uninstallRaw -match 'uninstall.native_step')) {
  throw "expected native uninstall step in uninstall plan"
}

$update = & $Binary update --json | ConvertFrom-Json
if (-not $update.install_method) {
  throw "expected install method in update output"
}

$touchID = & $Binary touchid --json | ConvertFrom-Json
if ($touchID.supported -ne $false) {
  throw "expected unsupported touchid status on Windows"
}
$touchIDEnable = & $Binary touchid enable --json | ConvertFrom-Json
if ($touchIDEnable.supported -ne $false -or $touchIDEnable.action -ne 'enable') {
  throw "expected unsupported touchid enable preview on Windows"
}

$remove = & $Binary remove --json | ConvertFrom-Json
if ($remove.command -ne 'remove') {
  throw "expected remove command in JSON output"
}

$uninstallExecRaw = & $Binary uninstall --json --dry-run=false --yes --native-uninstall "Example App"
$uninstallExecRaw | Out-File -Encoding utf8 (Join-Path $Root "uninstall-exec.json")
$uninstallExec = $uninstallExecRaw | ConvertFrom-Json
if ($uninstallExec.plan.command -ne 'uninstall') {
  throw "expected uninstall command in execution envelope"
}
if (-not ($uninstallExec.result.items | Where-Object { $_.status -eq 'completed' } | Select-Object -First 1)) {
  throw "expected completed native uninstall result"
}
if (-not ($uninstallExec.result.warnings -match 'Native uninstaller launched')) {
  throw "expected native uninstall follow-up warning"
}
if ($uninstallExec.result.follow_up_commands) {
  throw "expected no follow-up commands after native uninstall continuation"
}
for ($i = 0; $i -lt 100 -and -not (Test-Path $NativeSentinel); $i++) {
  Start-Sleep -Milliseconds 100
}
if (-not (Test-Path $NativeSentinel)) {
  throw "expected native uninstall helper to create sentinel file"
}
Remove-Item -Recurse -Force (Join-Path $env:LOCALAPPDATA "Programs\Example App")
$uninstallRerunRaw = & $Binary uninstall --json "Example App"
$uninstallRerunRaw | Out-File -Encoding utf8 (Join-Path $Root "uninstall-rerun.json")
$uninstallRerun = $uninstallRerunRaw | ConvertFrom-Json
if ($uninstallRerun.command -ne 'uninstall') {
  throw "expected uninstall command in rerun plan"
}
if (-not ($uninstallRerunRaw -match 'No installed app or leftover files were found for Example\.')) {
  throw "expected empty uninstall rerun after native uninstall"
}

$status = & $Binary status --plain
$status | Out-File -Encoding utf8 (Join-Path $Root "status.txt")
if (-not ($status -match '^System:')) {
  throw "expected status system output"
}
if (-not ($status -match '^Operator alerts:')) {
  throw "expected operator alerts in status output"
}
& $Binary completion bash | Out-File -Encoding utf8 (Join-Path $Root "completions\sift.bash")
& $Binary completion zsh | Out-File -Encoding utf8 (Join-Path $Root "completions\_sift")
& $Binary completion fish | Out-File -Encoding utf8 (Join-Path $Root "completions\sift.fish")
& $Binary completion powershell | Out-File -Encoding utf8 (Join-Path $Root "completions\sift.ps1")
foreach ($completion in @("completions\sift.bash", "completions\_sift", "completions\sift.fish", "completions\sift.ps1")) {
  if (-not (Test-Path (Join-Path $Root $completion))) {
    throw "expected completion output $completion"
  }
}
$report = & $Binary report --json | ConvertFrom-Json
if (-not (Test-Path $report.path)) {
  throw "expected report bundle at $($report.path)"
}
