param(
  [switch]$SkipFrontendBuild,
  [switch]$NoBrowser
)

$ErrorActionPreference = "Stop"

$Root = $PSScriptRoot
Set-Location $Root

function Require-Command($Name, $InstallHint) {
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "Missing command '$Name'. $InstallHint"
  }
}

function Assert-NativeSuccess($Step) {
  if ($LASTEXITCODE -ne 0) {
    throw "$Step failed with exit code $LASTEXITCODE."
  }
}

Require-Command "go" "Install Go, then retry."

if (-not $SkipFrontendBuild) {
  Require-Command "npm" "Install Node.js and npm, then retry."
  Push-Location (Join-Path $Root "frontend")
  npm install
  Assert-NativeSuccess "npm install"
  npm run build
  Assert-NativeSuccess "npm run build"
  Pop-Location
}

$goArgs = @("run", ".")
if ($NoBrowser) {
  $goArgs += "--"
  $goArgs += "--no-browser"
}

Write-Host "Starting study tracker at http://127.0.0.1:8000" -ForegroundColor Cyan
& go @goArgs
Assert-NativeSuccess "go run"
