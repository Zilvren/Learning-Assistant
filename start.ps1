# 错题追踪器 - 一键启动脚本

$root = Split-Path -Parent $MyInvocation.MyCommand.Path

Write-Host "Starting server..." -ForegroundColor Cyan
Write-Host ""

# Check if built frontend exists
if (-not (Test-Path "$root\frontend\dist")) {
    Write-Host "Building frontend first..." -ForegroundColor Yellow
    Set-Location "$root\frontend"
    npm install 2>&1 | Out-Null
    npm run build 2>&1 | Out-Null
    Set-Location $root
    Write-Host ""
}

Write-Host "Server: http://127.0.0.1:8000" -ForegroundColor Green
Write-Host "Press Ctrl+C to stop" -ForegroundColor Gray
Write-Host ""

Set-Location $root
python backend/api.py
