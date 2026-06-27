param(
  [string]$Version = "",
  [string]$ProductName = "Tracker",
  [switch]$Upload,
  [switch]$Clean
)

$ErrorActionPreference = "Stop"

$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $Root

if (-not $Version) {
  $Version = Get-Date -Format "yyyy.MM.dd-HHmm"
}

$NormalizedVersion = if ($Version.StartsWith("v")) { $Version.Substring(1) } else { $Version }
$Repo = "Zilvren/Learning-Assitant"
$DistDir = Join-Path $Root "dist"
$ReleaseDir = Join-Path $DistDir "release"
$PackageDir = Join-Path $ReleaseDir "$ProductName-$NormalizedVersion"
$ZipPath = Join-Path $ReleaseDir "$ProductName.zip"
$TrackerPath = Join-Path $DistDir "$ProductName.exe"
$UpdaterPath = Join-Path $DistDir "Updater.exe"

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

if (-not (Test-Path (Join-Path $Root "frontend\dist\index.html"))) {
  throw "Missing frontend/dist/index.html. Build or copy the frontend dist before packaging."
}

if ($Clean) {
  Remove-Item -LiteralPath $DistDir -Recurse -Force -ErrorAction SilentlyContinue
}

New-Item -ItemType Directory -Force -Path $DistDir | Out-Null
New-Item -ItemType Directory -Force -Path $ReleaseDir | Out-Null

Write-Host "==> Building $ProductName.exe"
go build -o $TrackerPath .
Assert-NativeSuccess "$ProductName build"

Write-Host "==> Building Updater.exe"
go build -o $UpdaterPath ./cmd/updater
Assert-NativeSuccess "Updater build"

if (-not (Test-Path $TrackerPath)) {
  throw "Expected exe not found: $TrackerPath"
}
if (-not (Test-Path $UpdaterPath)) {
  throw "Expected updater not found: $UpdaterPath"
}

Write-Host "==> Creating release package"
Remove-Item -LiteralPath $PackageDir -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $ZipPath -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $PackageDir | Out-Null

Copy-Item -LiteralPath $TrackerPath -Destination (Join-Path $PackageDir "$ProductName.exe")
Copy-Item -LiteralPath $UpdaterPath -Destination (Join-Path $PackageDir "Updater.exe")

$VersionJson = [ordered]@{
  version = $NormalizedVersion
  repo = $Repo
  asset_name = "$ProductName.zip"
  app_exe = "$ProductName.exe"
} | ConvertTo-Json
$Utf8NoBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllText((Join-Path $PackageDir "version.json"), $VersionJson, $Utf8NoBom)

$PackageReadme = Join-Path $PackageDir "README-release.txt"
$PackageReadmeText = @(
  "$ProductName $NormalizedVersion",
  "",
  "How to run:",
  "1. Double-click $ProductName.exe",
  "2. Open http://127.0.0.1:8000 if the browser does not open automatically.",
  "3. Local data is stored in the data folder next to the exe.",
  "",
  "Notes:",
  "- The data folder is preserved during automatic updates.",
  "- Automatic update downloads $ProductName.zip from GitHub Releases and uses Updater.exe to replace program files.",
  "- If port 8000 is occupied, close the program using that port first."
)
Set-Content -LiteralPath $PackageReadme -Value $PackageReadmeText -Encoding UTF8

Compress-Archive -Path (Join-Path $PackageDir "*") -DestinationPath $ZipPath -Force

Write-Host "==> Release package ready:"
Write-Host "    EXE: $TrackerPath"
Write-Host "    UPDATER: $UpdaterPath"
Write-Host "    ZIP: $ZipPath"

if ($Upload) {
  Require-Command "gh" "Install GitHub CLI and run: gh auth login"

  $Tag = "v$NormalizedVersion"
  $ReleaseTitle = "$ProductName $NormalizedVersion"
  $Notes = "Automated Windows build for $ProductName $NormalizedVersion."

  $oldErrorActionPreference = $ErrorActionPreference
  $ErrorActionPreference = "Continue"
  gh release view $Tag -R $Repo *> $null
  $releaseExists = $LASTEXITCODE -eq 0
  $ErrorActionPreference = $oldErrorActionPreference

  if ($releaseExists) {
    Write-Host "==> Uploading assets to existing GitHub release $Tag"
    gh release upload $Tag $ZipPath --clobber -R $Repo
    Assert-NativeSuccess "GitHub release upload"
  } else {
    Write-Host "==> Creating GitHub release $Tag"
    gh release create $Tag $ZipPath --title $ReleaseTitle --notes $Notes -R $Repo
    Assert-NativeSuccess "GitHub release creation"
  }
}
