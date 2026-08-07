# 渲染 docs/diagrams 下所有 .dot 为 PNG(150dpi) 和 SVG
# 用法: 在本目录下  pwsh -File render.ps1
# 依赖: Graphviz (dot) 已安装并加入 PATH
$ErrorActionPreference = "Stop"
$dotDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$gv = Get-Command dot -ErrorAction SilentlyContinue
if (-not $gv) {
  $gvPath = "C:\Program Files\Graphviz\bin\dot.exe"
  if (Test-Path $gvPath) { $env:PATH = "C:\Program Files\Graphviz\bin;$env:PATH" }
  else { throw "Graphviz dot 未找到，请先安装: winget install --id Graphviz.Graphviz" }
}

Get-ChildItem -Path $dotDir -Filter *.dot | ForEach-Object {
  $base = $_.BaseName
  Write-Host "==> $base"
  dot -Tpng -Gdpi=150 "$_.FullName" -o (Join-Path $dotDir "$base.png")
  dot -Tsvg        "$_.FullName" -o (Join-Path $dotDir "$base.svg")
}
Write-Host "Done." -ForegroundColor Green
