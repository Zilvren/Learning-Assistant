param(
  [string]$Version = "",
  [switch]$Clean,
  [switch]$SkipGit
)

$ErrorActionPreference = "Stop"

$Root = $PSScriptRoot
Set-Location $Root

if (-not $Version) {
  $Version = Get-Date -Format "yyyy.MM.dd-HHmm"
}

function Assert-NativeSuccess($Step) {
  if ($LASTEXITCODE -ne 0) {
    throw "$Step failed with exit code $LASTEXITCODE."
  }
}

if (-not $SkipGit) {
  if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    throw "Git is not installed or not available in PATH."
  }

  $Branch = git branch --show-current
  Assert-NativeSuccess "git branch --show-current"
  if (-not $Branch) {
    throw "Could not determine current git branch."
  }

  git status --short
  Assert-NativeSuccess "git status"

  $HasChanges = (git status --porcelain) -ne $null
  Assert-NativeSuccess "git status --porcelain"

  if ($HasChanges) {
    Write-Host "Committing project changes on branch $Branch" -ForegroundColor Cyan
    git add -A
    Assert-NativeSuccess "git add"
    git commit -m "release $Version"
    Assert-NativeSuccess "git commit"
    git push origin $Branch
    Assert-NativeSuccess "git push"
  } else {
    Write-Host "No project changes to commit." -ForegroundColor Green
  }
}

$ReleaseScript = Join-Path $Root "scripts\build-release.ps1"
if (-not (Test-Path $ReleaseScript)) {
  throw "Release script not found: $ReleaseScript"
}

$args = @(
  "-NoProfile",
  "-ExecutionPolicy", "Bypass",
  "-File", $ReleaseScript,
  "-Version", $Version
)

$CanUpload = $false
if (Get-Command gh -ErrorAction SilentlyContinue) {
  gh auth status *> $null
  if ($LASTEXITCODE -eq 0) {
    $CanUpload = $true
  } else {
    Write-Host "GitHub CLI is installed but not authenticated. Run: gh auth login" -ForegroundColor Yellow
  }
} else {
  Write-Host "GitHub CLI is not installed. The script will build EXE/ZIP only." -ForegroundColor Yellow
  Write-Host "Install it later with: winget install --id GitHub.cli" -ForegroundColor Yellow
  Write-Host "Then run: gh auth login" -ForegroundColor Yellow
}

if ($CanUpload) {
  $args += "-Upload"
}

if ($Clean) {
  $args += "-Clean"
}

if ($CanUpload) {
  Write-Host "Publishing release version $Version" -ForegroundColor Cyan
} else {
  Write-Host "Building local release package version $Version" -ForegroundColor Cyan
}
& powershell @args
if ($LASTEXITCODE -ne 0) {
  throw "Release script failed with exit code $LASTEXITCODE."
}

Write-Host ""
if ($CanUpload) {
  Write-Host "Release finished." -ForegroundColor Green
} else {
  Write-Host "Local package finished. ZIP is under dist\release." -ForegroundColor Green
  Write-Host "GitHub upload was skipped because GitHub CLI is unavailable or not logged in." -ForegroundColor Yellow
}
Read-Host "Press Enter to exit"
