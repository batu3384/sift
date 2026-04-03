param(
  [string]$Binary = ".\bin\sift.exe"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Assert-PlanCommand {
  param(
    [Parameter(Mandatory = $true)]$Plan,
    [Parameter(Mandatory = $true)][string]$Expected
  )

  if ($Plan.command -ne $Expected) {
    throw "expected command '$Expected', got '$($Plan.command)'"
  }
}

function Assert-PlanHasItem {
  param(
    [Parameter(Mandatory = $true)]$Plan,
    [Parameter(Mandatory = $true)][string]$PathPattern,
    [Parameter(Mandatory = $true)][string]$ExpectedStatus
  )

  $item = $Plan.items | Where-Object {
    (($_.path -as [string]) -match $PathPattern) -or
    (($_.display_path -as [string]) -match $PathPattern)
  } | Select-Object -First 1
  if (-not $item) {
    throw "expected item matching '$PathPattern'"
  }
  if ($item.status -ne $ExpectedStatus) {
    throw "expected status '$ExpectedStatus' for '$PathPattern', got '$($item.status)'"
  }
}

function Assert-ExecutionResult {
  param(
    [Parameter(Mandatory = $true)]$Execution,
    [Parameter(Mandatory = $true)][string]$PathPattern,
    [Parameter(Mandatory = $true)][string]$ExpectedStatus
  )

  $item = $Execution.result.items | Where-Object { ($_.path -as [string]) -match $PathPattern } | Select-Object -First 1
  if (-not $item) {
    throw "expected execution item matching '$PathPattern'"
  }
  if ($item.status -ne $ExpectedStatus) {
    throw "expected execution status '$ExpectedStatus' for '$PathPattern', got '$($item.status)'"
  }
}

function Assert-WarningContains {
  param(
    [Parameter(Mandatory = $true)]$Warnings,
    [Parameter(Mandatory = $true)][string]$Expected
  )

  if (-not ($Warnings | Where-Object { ($_ -as [string]) -match [regex]::Escape($Expected) })) {
    throw "expected warning containing '$Expected'"
  }
}

function Assert-FollowUpContains {
  param(
    [Parameter(Mandatory = $true)]$Commands,
    [Parameter(Mandatory = $true)][string]$Expected
  )

  if (-not ($Commands | Where-Object { ($_ -as [string]) -match [regex]::Escape($Expected) })) {
    throw "expected follow-up containing '$Expected'"
  }
}

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
Assert-PlanCommand -Plan $optimize -Expected 'optimize'

$clean = & $Binary clean --json --profile safe | ConvertFrom-Json
Assert-PlanCommand -Plan $clean -Expected 'clean'
$deepClean = & $Binary clean --json --profile deep | ConvertFrom-Json
Assert-PlanHasItem -Plan $deepClean -PathPattern 'Google[\\/]+Chrome[\\/]+User Data[\\/]+Default[\\/]+Code Cache(?:$|[\\/])' -ExpectedStatus 'planned'
$chocoCache = $deepClean.items | Where-Object { $_.path -match 'chocolatey[\\/]+cache(?:$|[\\/])' } | Select-Object -First 1
if (-not $chocoCache) {
  throw "expected chocolatey cache finding in deep clean output"
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
Assert-PlanCommand -Plan $purge -Expected 'purge'
$purgeScan = & $Binary purge scan --json (Join-Path $Root "project") | ConvertFrom-Json
Assert-PlanCommand -Plan $purgeScan -Expected 'purge_scan'
Assert-PlanHasItem -Plan $purgeScan -PathPattern 'node_modules' -ExpectedStatus 'planned'

$uninstallRaw = & $Binary uninstall --json "Example App"
$uninstallRaw | Out-File -Encoding utf8 (Join-Path $Root "uninstall-plan.json")
$uninstall = $uninstallRaw | ConvertFrom-Json
Assert-PlanCommand -Plan $uninstall -Expected 'uninstall'
Assert-PlanHasItem -Plan $uninstall -PathPattern 'uninstall\.exe' -ExpectedStatus 'planned'

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
Assert-PlanCommand -Plan $remove -Expected 'remove'

$uninstallExecRaw = & $Binary uninstall --json --dry-run=false --yes --native-uninstall "Example App"
$uninstallExecRaw | Out-File -Encoding utf8 (Join-Path $Root "uninstall-exec.json")
$uninstallExec = $uninstallExecRaw | ConvertFrom-Json
Assert-PlanCommand -Plan $uninstallExec.plan -Expected 'uninstall'
if (-not $uninstallExec.result.items) {
  throw "expected uninstall execution results"
}
Assert-WarningContains -Warnings $uninstallExec.result.warnings -Expected 'Native uninstaller launched'
Assert-FollowUpContains -Commands $uninstallExec.result.follow_up_commands -Expected 'Settings > Apps > Startup'
Assert-ExecutionResult -Execution $uninstallExec -PathPattern 'uninstall\.exe' -ExpectedStatus 'completed'
for ($i = 0; $i -lt 100 -and -not (Test-Path $NativeSentinel); $i++) {
  Start-Sleep -Milliseconds 100
}
if (-not (Test-Path $NativeSentinel)) {
  throw "expected native uninstall helper to create sentinel file"
}
$installedAppPath = Join-Path $env:LOCALAPPDATA "Programs\Example App"
if (Test-Path $installedAppPath) {
  Remove-Item -Recurse -Force $installedAppPath
}
$uninstallRerunRaw = & $Binary uninstall --json "Example App"
$uninstallRerunRaw | Out-File -Encoding utf8 (Join-Path $Root "uninstall-rerun.json")
$uninstallRerun = $uninstallRerunRaw | ConvertFrom-Json
Assert-PlanCommand -Plan $uninstallRerun -Expected 'uninstall'
Assert-WarningContains -Warnings $uninstallRerun.warnings -Expected 'No installed app or leftover files were found for Example App.'

$status = & $Binary status --plain
$status | Out-File -Encoding utf8 (Join-Path $Root "status.txt")
if (-not ($status -match '(?m)^System:')) {
  throw "expected status system output"
}
if (-not ($status -match '(?m)^Health ')) {
  throw "expected status health output"
}
if (-not ($status -match '(?m)^Audit log:')) {
  throw "expected audit log in status output"
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
