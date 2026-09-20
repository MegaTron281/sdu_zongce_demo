Set-Location $PSScriptRoot
cmd /c "docker compose stop"
Write-Host "PostgreSQL 已停止（数据保留在卷中，下次 start.bat 会自动再启动）。" -ForegroundColor Green
Read-Host "按回车键退出"
