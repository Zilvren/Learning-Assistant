param(
  [switch]$SkipFrontendBuild,
  [switch]$NoBrowser,
  [int]$Port = 8000
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

function Test-PortAvailable($Port) {
  $listener = $null
  try {
    $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Parse("127.0.0.1"), $Port)
    $listener.Start()
    return $true
  } catch {
    return $false
  } finally {
    if ($listener) {
      $listener.Stop()
    }
  }
}

function Find-AvailablePort($PreferredPort) {
  for ($candidate = $PreferredPort; $candidate -lt ($PreferredPort + 20); $candidate++) {
    if (Test-PortAvailable $candidate) {
      return $candidate
    }
  }
  throw "No available port found from $PreferredPort to $($PreferredPort + 19)."
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

$actualPort = Find-AvailablePort $Port
if ($actualPort -ne $Port) {
  Write-Host "Port $Port is already in use. Starting on port $actualPort instead." -ForegroundColor Yellow
}

$goArgs = @("run", ".")
$goArgs += "--"
$goArgs += "--port"
$goArgs += "$actualPort"
if ($NoBrowser) {
  $goArgs += "--no-browser"
}

Write-Host "Starting study tracker at http://127.0.0.1:$actualPort" -ForegroundColor Cyan
& go @goArgs
Assert-NativeSuccess "go run"
