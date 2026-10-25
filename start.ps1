$root = $PSScriptRoot
Set-Location $root

function Abort($msg) {
    Write-Host "[错误] $msg" -ForegroundColor Red
    Read-Host "按回车键退出"
    exit 1
}

function Test-Docker {
    cmd /c "docker info >nul 2>&1"
    return ($LASTEXITCODE -eq 0)
}

Write-Host "[1/3] 检查 Docker 引擎..." -ForegroundColor Cyan
if (-not (Test-Docker)) {
    $desktop = "C:\Program Files\Docker\Docker\Docker Desktop.exe"
    if (-not (Test-Path $desktop)) {
        Abort "找不到 Docker Desktop，请先安装或手动启动后重试。"
    }
    Write-Host "未检测到 Docker 引擎，正在启动 Docker Desktop（首次可能需要一两分钟）..."
    Start-Process $desktop | Out-Null
    $ready = $false
    foreach ($i in 1..36) {
        Start-Sleep -Seconds 5
        if (Test-Docker) { $ready = $true; break }
    }
    if (-not $ready) { Abort "Docker 引擎 180 秒内未就绪，请手动打开 Docker Desktop 后重试。" }
}
Write-Host "Docker 就绪。"

Write-Host "[2/3] 启动 PostgreSQL（等待健康检查通过）..." -ForegroundColor Cyan
cmd /c "docker compose up -d --wait postgres"
if ($LASTEXITCODE -ne 0) { Abort "PostgreSQL 启动失败，请查看上方 docker compose 输出。" }

Write-Host "[3/3] 数据库表结构检查（不改动任何业务数据）..." -ForegroundColor Cyan
cmd /c "go run ./cmd/migrate up"
if ($LASTEXITCODE -ne 0) { Abort "数据库迁移失败。" }

Write-Host "启动后端（:9080）与前端（:5173）..." -ForegroundColor Cyan
if (-not (Test-Path "$root\frontend\node_modules")) {
    Start-Process cmd -ArgumentList "/k", "title sdu-frontend && npm install && npm run dev" -WorkingDirectory "$root\frontend" | Out-Null
} else {
    Start-Process cmd -ArgumentList "/k", "title sdu-frontend && npm run dev" -WorkingDirectory "$root\frontend" | Out-Null
}
Start-Process cmd -ArgumentList "/k", "title sdu-backend && go run ./cmd/server" -WorkingDirectory $root | Out-Null

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  启动完成！（本脚本不修改任何业务数据）" -ForegroundColor Green
Write-Host "  前端： http://localhost:5173"
Write-Host "  后端： http://localhost:9080"
Write-Host "  停止服务： 关闭 sdu-backend / sdu-frontend 两个窗口，再运行 stop.bat"
Write-Host "========================================" -ForegroundColor Green
Start-Sleep -Seconds 5
