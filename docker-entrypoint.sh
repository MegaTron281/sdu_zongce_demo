#!/bin/sh
set -e

echo "=== SDU 综测平台 容器启动 ==="

# 运行数据库迁移
echo "[1/3] 执行数据库迁移..."
/app/migrate up

# 可选：灌入种子数据（设 RUN_SEED=true 启用）
if [ "$RUN_SEED" = "true" ]; then
    echo "[2/3] 灌入种子数据..."
    /app/seed
else
    echo "[2/3] 跳过种子数据（设 RUN_SEED=true 启用）"
fi

# 启动应用服务
echo "[3/3] 启动应用服务..."
exec /app/server
