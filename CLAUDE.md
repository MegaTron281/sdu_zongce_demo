# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

SDU 综合测评成绩管理平台 prototype — 高校综合测评工作台（student 申报 → 班委初审 → 班级公示 → 管理员复审 → 学院公示 → 计分 → 导出造册）。三种角色：`student`、`reviewer`（班委）、`admin`（辅导员）。

## Run / build

后端：

```bash
docker compose up -d postgres        # 起 Postgres
cp .env.example .env                 # 首次需要复制
go run ./cmd/migrate up              # 建表
go run ./cmd/seed                    # 写入 demo 数据
go run ./cmd/server                  # 监听 :8080
go build ./...                       # 编译检查
go vet ./...
go test ./...
```

前端（开发）：

```bash
cd frontend
npm install
npm run dev    # 监听 :5173，/api 代理到 :8080
```

前端（生产）：执行 `npm run build`，Gin 会自动从 `frontend/dist` 提供静态文件并把非 `/api/*` 的路由 fallback 到 `index.html`。

Demo 账号（密码统一 `123456`）：
- student: `20230001`
- reviewer: `20230002`
- admin: `coach-wang`（显示名"辅导员 王老师"）

## Architecture

### Backend

按职责分层，均在 `internal/` 下：

- [cmd/server/main.go](cmd/server/main.go) — 仅 wire 配置、连库、起 Gin
- [cmd/migrate/main.go](cmd/migrate/main.go) — 调用 `golang-migrate` 跑 `migrations/*.sql`
- [cmd/seed/main.go](cmd/seed/main.go) — 通过 GORM `OnConflict.UpdateAll` 灌入 demo 数据。**注意 destructive**：再次执行会覆盖账号密码哈希
- [internal/config](internal/config/) — 从环境变量加载 `DATABASE_URL` / `SESSION_SECRET` / `FRONTEND_ORIGIN` / `COOKIE_SECURE`
- [internal/repository](internal/repository/) — `gorm.Open(postgres.Open(...))`
- [internal/domain](internal/domain/) — GORM 模型 + 状态/角色常量 + DTO。**所有状态字符串的唯一来源**
- [internal/service](internal/service/) — 业务逻辑，按资源切文件（auth / applications / catalog / score）。事务用 `s.db.Transaction`；行级悲观锁用 `clause.Locking{Strength: "UPDATE"}`
- [internal/httpapi](internal/httpapi/) — Gin handlers + 中间件（`authRequired` / `requireRole` / CORS）。错误用 sentinel error（`ErrUnauthorized` / `ErrForbidden` / `ErrConflict` / `ErrNotFound`）+ `errors.Is` 判等映射 HTTP 状态码

**Session 认证**：登录返回 HTTP-only cookie；服务端在 `sessions` 表存 `sha256(secret + token)` 哈希，`AuthenticateToken` 用未撤销 + 未过期校验。所有非 `/api/health` / `/api/auth/login` 的接口都走 `authRequired`。审计日志的 actor 来自 session，不再信任客户端。

**计分**：单一来源是数据库里的 active rule（[internal/service/score.go](internal/service/score.go) 的 `CalculateScore`）。前端通过 `POST /api/scores/preview` 拿预览分，不再自己算。集体项目按位次衰减、最低 30%。

**状态机**（`internal/service/applications.go::ApplicationAction`）：`待班委审核` → `公示中` → `待管理员审核` → `已通过`/`已驳回`，外加 `特殊情况处理中`。`公示中` 到期由后台 ticker 自动迁移到 `待管理员审核`（[internal/service/applications.go::AdvanceExpiredPublicity](internal/service/applications.go)），并写审计日志；启动时也会跑一次。

**计分聚合**只累加 `已通过`，`公示中` 不计入。模块分受 `RuleVersion.ModuleMax` 上限封顶，最终 `assessment = academic*0.9 + quality*0.1`。

`summaryPhase` 在 `assessment_cycles.summary_phase` 上；启用后 `CreateApplication` 返回 409。

### Frontend

`frontend/`：**Vue 3 + Vite + TypeScript + Pinia + vue-router**。

- [src/api/client.ts](frontend/src/api/client.ts) — `fetch` 封装，`credentials: "include"` 带 cookie；非 2xx 抛 `ApiError`
- [src/api/types.ts](frontend/src/api/types.ts) — 后端 DTO 的 TS 镜像
- [src/api/format.ts](frontend/src/api/format.ts) — 数字/日期/倒计时格式化
- [src/constants/status.ts](frontend/src/constants/status.ts) — **状态字符串和颜色映射的唯一来源**。所有视图通过 `import { ApplicationStatus, ActivityStatus, statusColor } from "@/constants/status"` 使用，不要在视图里写中文字面量
- [src/stores/auth.ts](frontend/src/stores/auth.ts) — 当前用户 + meta（包括 `applicationStatuses` / `activityStatuses` 枚举，从后端 `/api/meta` 读）
- [src/stores/workspace.ts](frontend/src/stores/workspace.ts) — 列表数据、loading、toast
- [src/router/index.ts](frontend/src/router/index.ts) — 路由 + `beforeEach` 强制 `auth.ensureMe()`，按 `meta.roles` 限角色
- [src/views/](frontend/src/views/) — 一个角色一个目录
- [src/components/](frontend/src/components/) — `StatusTag`、`ScorePreview` 等

新增页面：在 `src/router/index.ts` 的 `rolePages` 加入条目；新增状态：改 [internal/domain/models.go](internal/domain/models.go) 的常量 + [migrations/](migrations/) + 前端 [src/constants/status.ts](frontend/src/constants/status.ts) 的颜色映射。

### Data flow

[API.md](API.md) 是前后端契约，新增/修改接口时同步。状态枚举从后端 meta 拉，避免再次硬编码漂移。

## Conventions

- 代码、UI 文案、注释、commit message 使用中文；Go 标识符使用英文
- Commit message 前缀：`feat:` / `fix:` / `docs:` / `style:` / `refactor:` / `test:`
- 测试位置：`internal/service/*_test.go`，sqlite in-memory 模式，跑 `go test ./...`
- 修改 seed 数据要保持 idempotent，不要破坏 admin 改过的密码
