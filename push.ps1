param(
  [string]$Version = "",
  [switch]$Clean,
  [switch]$SkipGit,
  [switch]$NoUpload
)

$ErrorActionPreference = "Stop"

$Root = $PSScriptRoot
Set-Location $Root

if (-not $Version) {
  $Version = Get-Date -Format "yyyy.MM.dd-HHmm"
}
$NormalizedVersion = if ($Version.StartsWith("v")) { $Version.Substring(1) } else { $Version }

function Assert-NativeSuccess($Step) {
  if ($LASTEXITCODE -ne 0) {
    throw "$Step failed with exit code $LASTEXITCODE."
  }
}

function Write-VersionFile($TargetVersion) {
  $VersionJson = [ordered]@{
    version = $TargetVersion
    repo = "Zilvren/Learning-Assitant"
    asset_name = "Tracker.zip"
    app_exe = "Tracker.exe"
  } | ConvertTo-Json
  $Utf8NoBom = New-Object System.Text.UTF8Encoding($false)
  [System.IO.File]::WriteAllText((Join-Path $Root "version.json"), $VersionJson, $Utf8NoBom)
}

Write-VersionFile $NormalizedVersion

$InsideGitRepo = $false
if (Get-Command git -ErrorAction SilentlyContinue) {
  $oldErrorActionPreference = $ErrorActionPreference
  $ErrorActionPreference = "Continue"
  git rev-parse --is-inside-work-tree *> $null
  $InsideGitRepo = $LASTEXITCODE -eq 0
  $ErrorActionPreference = $oldErrorActionPreference
}

if (-not $SkipGit -and $InsideGitRepo) {
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
    git commit -m "release $NormalizedVersion"
    Assert-NativeSuccess "git commit"
    git push origin $Branch
    Assert-NativeSuccess "git push"
  } else {
    Write-Host "No project changes to commit." -ForegroundColor Green
  }
} elseif ($SkipGit) {
  Write-Host "Git step skipped by -SkipGit." -ForegroundColor Yellow
} else {
  Write-Host "Not inside a git repository. Git commit/push skipped." -ForegroundColor Yellow
}

$ReleaseScript = Join-Path $Root "scripts\build-release.ps1"
if (-not (Test-Path $ReleaseScript)) {
  throw "Release script not found: $ReleaseScript"
}

$args = @(
  "-NoProfile",
  "-ExecutionPolicy", "Bypass",
  "-File", $ReleaseScript,
  "-Version", $NormalizedVersion
)

$CanUpload = $false
if (-not $NoUpload -and (Get-Command gh -ErrorAction SilentlyContinue)) {
  $oldErrorActionPreference = $ErrorActionPreference
  $ErrorActionPreference = "Continue"
  gh auth status *> $null
  if ($LASTEXITCODE -eq 0) {
    $CanUpload = $true
  } else {
    Write-Host "GitHub CLI is installed but not authenticated. Run: gh auth login" -ForegroundColor Yellow
  }
  $ErrorActionPreference = $oldErrorActionPreference
} elseif ($NoUpload) {
  Write-Host "GitHub upload skipped by -NoUpload." -ForegroundColor Yellow
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
  Write-Host "Publishing release version $NormalizedVersion" -ForegroundColor Cyan
} else {
  Write-Host "Building local release package version $NormalizedVersion" -ForegroundColor Cyan
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
  Write-Host "GitHub upload was skipped." -ForegroundColor Yellow
}

Read-Host "Press Enter to exit"
