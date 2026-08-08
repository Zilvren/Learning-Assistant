param(
  [string]$DatabaseUrl = $env:TRACKER_DATABASE_URL,
  [switch]$SkipFrontendBuild,
  [switch]$NoBrowser,
  [int]$Port = 8000
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($DatabaseUrl)) {
  throw "请提供 -DatabaseUrl，或先设置 TRACKER_DATABASE_URL。"
}

$env:TRACKER_STORAGE = "postgres"
$env:TRACKER_REQUIRE_POSTGRES = "true"
$env:TRACKER_DATABASE_URL = $DatabaseUrl

$startArgs = @{
  Port = $Port
}
if ($SkipFrontendBuild) { $startArgs.SkipFrontendBuild = $true }
if ($NoBrowser) { $startArgs.NoBrowser = $true }

& (Join-Path $PSScriptRoot "start.ps1") @startArgs
exit $LASTEXITCODE
