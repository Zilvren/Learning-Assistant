param(
  [string]$Version = "",
  [string]$Spec = "Tracker.spec",
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
$Tag = "v$NormalizedVersion"
$ReleaseDir = Join-Path $Root "dist\release"
$PackageDir = Join-Path $ReleaseDir "$ProductName-$NormalizedVersion"
$ZipPath = Join-Path $ReleaseDir "$ProductName.zip"
$ExePath = Join-Path $Root "dist\$ProductName.exe"
$UpdaterPath = Join-Path $Root "dist\Updater.exe"

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

Require-Command "npm" "Install Node.js and npm, then retry."
Require-Command "python" "Install Python, then retry."
Require-Command "pyinstaller" "Install PyInstaller: python -m pip install pyinstaller"

if (-not (Test-Path (Join-Path $Root $Spec))) {
  throw "Spec file not found: $Spec"
}

if ($Clean) {
  Remove-Item -LiteralPath (Join-Path $Root "build") -Recurse -Force -ErrorAction SilentlyContinue
  Remove-Item -LiteralPath (Join-Path $Root "dist") -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Host "==> Building frontend"
Push-Location (Join-Path $Root "frontend")
npm install
Assert-NativeSuccess "npm install"
npm run build
Assert-NativeSuccess "npm run build"
Pop-Location

Write-Host "==> Building exe with PyInstaller ($Spec)"
python -m PyInstaller -y $Spec
Assert-NativeSuccess "PyInstaller"

if (-not (Test-Path $ExePath)) {
  throw "Expected exe not found: $ExePath"
}

Write-Host "==> Building updater with PyInstaller"
python -m PyInstaller -y --onefile --name Updater updater.py
Assert-NativeSuccess "Updater PyInstaller"

if (-not (Test-Path $UpdaterPath)) {
  throw "Expected updater not found: $UpdaterPath"
}

Write-Host "==> Creating release package"
New-Item -ItemType Directory -Force -Path $ReleaseDir | Out-Null
Remove-Item -LiteralPath $PackageDir -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $ZipPath -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $PackageDir | Out-Null

Copy-Item -LiteralPath $ExePath -Destination $PackageDir
Copy-Item -LiteralPath $UpdaterPath -Destination $PackageDir
Copy-Item -LiteralPath (Join-Path $Root "README.md") -Destination $PackageDir -ErrorAction SilentlyContinue

$VersionJson = [ordered]@{
  version = $NormalizedVersion
  repo = "Zilvren/Learning-Assitant"
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
  "2. The browser should open http://127.0.0.1:8000 automatically",
  "3. On first run, the app creates a data folder next to the exe",
  "",
  "Notes:",
  "- The data folder contains local study data and will be kept during automatic updates.",
  "- Automatic update downloads Tracker.zip from GitHub Releases and uses Updater.exe to replace program files.",
  "- If port 8000 is occupied, close the program using that port first."
)
Set-Content -LiteralPath $PackageReadme -Value $PackageReadmeText -Encoding UTF8

Compress-Archive -Path (Join-Path $PackageDir "*") -DestinationPath $ZipPath -Force

Write-Host "==> Release package ready:"
Write-Host "    EXE: $ExePath"
Write-Host "    UPDATER: $UpdaterPath"
Write-Host "    ZIP: $ZipPath"

if ($Upload) {
  Require-Command "gh" "Install GitHub CLI and run: gh auth login"

  $ReleaseTitle = "$ProductName $NormalizedVersion"
  $Notes = "Automated Windows build for $ProductName $NormalizedVersion."

  $oldErrorActionPreference = $ErrorActionPreference
  $ErrorActionPreference = "Continue"
  gh release view $Tag *> $null
  $releaseExists = $LASTEXITCODE -eq 0
  $ErrorActionPreference = $oldErrorActionPreference

  if ($releaseExists) {
    Write-Host "==> Uploading assets to existing GitHub release $Tag"
    gh release upload $Tag $ZipPath --clobber
    Assert-NativeSuccess "GitHub release upload"
  } else {
    Write-Host "==> Creating GitHub release $Tag"
    gh release create $Tag $ZipPath --title $ReleaseTitle --notes $Notes
    Assert-NativeSuccess "GitHub release creation"
  }
}
