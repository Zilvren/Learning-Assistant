Set-Location (Split-Path -Parent $MyInvocation.MyCommand.Path)
pip install -r requirements.txt 2>&1 | Out-Null

if (-not (Test-Path "frontend\dist")) {
    Write-Host "Building frontend..." -ForegroundColor Yellow
    Set-Location frontend
    npm install 2>&1 | Out-Null
    npm run build 2>&1 | Out-Null
    Set-Location ..
}

Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$pwd'; python backend/api.py"
