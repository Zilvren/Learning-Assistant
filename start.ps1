# 11408 考研学习追踪器 - 一键启动脚本

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$backend = "$root\backend"
$frontend = "$root\frontend"

Write-Host "🚀 启动 11408 考研学习追踪器..." -ForegroundColor Cyan

# 启动后端
Write-Host "📡 启动后端 API (端口 8000)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$root'; python -m uvicorn backend.api:app --host 127.0.0.1 --port 8000"

# 启动前端
Write-Host "🎨 启动前端 Vue (端口 5173)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$frontend'; node_modules\.bin\vite.cmd --host"

# 等待启动
Start-Sleep -Seconds 5

# 打开浏览器
Write-Host "🌐 打开浏览器..." -ForegroundColor Green
Start-Process "http://localhost:5173"

Write-Host ""
Write-Host "✅ 启动完成！后端 http://localhost:8000 | 前端 http://localhost:5173" -ForegroundColor Green
Write-Host "按任意键退出此窗口（不会关闭后端和前端）..."
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
