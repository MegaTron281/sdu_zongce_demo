# SDU 综测平台 — Docker 部署指南

## 架构说明

```
镜像内：
  /app/                    ← 应用文件（不被覆盖）
    ├── server             ← HTTP 服务
    ├── migrate            ← 数据库迁移
    ├── seed               ← 种子数据
    ├── migrations/        ← SQL 迁移脚本
    ├── frontend/dist/     ← 前端静态文件
    └── docker-entrypoint.sh

平台挂载：
  宿主机目录 → /data        ← 持久化数据（MCSManager 自动挂载）
    └── materials/          ← 上传的证明材料
```

> 注意：MCSManager 会将工作目录挂载到容器的 `/data`，因此**应用文件不能放 `/data`**，否则会被覆盖。

---

## 前置条件

- 本机已安装 Docker 且 Docker Engine 运行中
- 本项目源码已上传到服务器

---

## 1. 构建镜像

```bash
docker build --no-cache -t sdu-assessment .
```

---

## 2. 创建容器间网络

```bash
docker network create sdu-net
```

---

## 3. 启动 PostgreSQL 容器

```bash
docker run -d \
  --name sdu-postgres \
  --network sdu-net \
  --restart unless-stopped \
  -e POSTGRES_DB=sdu_assessment \
  -e POSTGRES_USER=sdu \
  -e POSTGRES_PASSWORD=sdu123 \
  -e PGDATA=/data \
  -v sdu-pgdata:/data \
  postgres:16-alpine
```

| 参数 | 说明 |
|------|------|
| `--name sdu-postgres` | 容器名，app 通过此名称连接 |
| `--network sdu-net` | 加入容器间网络 |
| `-e PGDATA=/data` | 数据库文件持久化目录 |
| `-v sdu-pgdata:/data` | Docker 卷持久化 |



## 4. 等待数据库就绪

```bash
docker run --rm --network sdu-net postgres:16-alpine \
  pg_isready -h sdu-postgres -U sdu -d sdu_assessment
```

输出 `accepting connections` 表示就绪。

---

## 5. 启动应用容器

> **首次部署**带 `RUN_SEED=true` 写入 demo 数据。后续重启去掉。

```bash
docker run -d \
  --name sdu-app \
  --network sdu-net \
  --restart unless-stopped \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://sdu:sdu123@sdu-postgres:5432/sdu_assessment?sslmode=disable" \
  -e SESSION_SECRET="请更换为随机字符串" \
  -e RUN_SEED=true \
  sdu-assessment
```

| 参数 | 说明 |
|------|------|
| `--network sdu-net` | 与数据库同一网络 |
| `-p 8080:8080` | **对外端口** |
| `DATABASE_URL` | PostgreSQL 连接串，`sdu-postgres` 即数据库容器名 |
| `SESSION_SECRET` | 会话加密密钥，务必修改 |
| `RUN_SEED=true` | 仅首次启用，写入 demo 账号 |


### MCSManager 平台部署

平台会自动处理工作目录挂载和端口映射，只需在面板中配置以下环境变量：

| 环境变量 | 值 |
|----------|-----|
| `DATABASE_URL` | `postgres://sdu:sdu123@<数据库IP>:5432/sdu_assessment?sslmode=disable` |
| `SESSION_SECRET` | 随机字符串 |
| `RUN_SEED` | `true`（首次，后续删除） |

---

## 6. 验证

```bash
curl http://localhost:8080/api/health
# → {"status":"ok"}
```

浏览器访问 `http://<服务器IP>:8080`。

---

## 常用管理

```bash
docker logs -f sdu-app          # 查看日志
docker restart sdu-app           # 重启
docker stop sdu-app sdu-postgres # 停止
docker rm -f sdu-app sdu-postgres # 删除容器
docker volume rm sdu-pgdata      # 删除数据库卷
docker network rm sdu-net        # 删除网络
```

---

## Demo 账号（密码 123456）

| 角色 | 账号 |
|------|------|
| 学生 | `20230001` |
| 班委 | `20230002` |
| 管理员 | `coach-wang` |
