# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

SDU 综合测评成绩管理平台 prototype — 高校综合测评工作台（student 申报 → 班委初审 → 班级公示 → 管理员复审 → 学院公示 → 计分 → 导出造册）。三种角色：`student`、`reviewer`（班委）、`admin`（辅导员）；`admin` 中带 `is_super_admin` 标志的唯一账号是**系统管理员**（不受管辖班级限制，唯一能分配辅导员管辖班级与任命辅导员的身份）。数据隔离 fail-closed：student 仅本人、reviewer 仅本班、辅导员仅 `users.managed_classes` 管辖班级。

## Run / build

一键启动（Windows）：双击 `start.bat` — 自动拉起 Docker Desktop（如未运行）→ 起 Postgres 并等健康检查 → 数据库表结构迁移（migrate 只建表，不改动业务数据）→ 开两个窗口分别运行后端 `:9080` 和前端 `:5173`。**纯启动工具，不含 seed**；演示数据只能手动 `go run ./cmd/seed` 恢复（destructive，会覆盖账号密码并恢复全部演示数据）。停止数据库用 `stop.bat`。逻辑在 `start.ps1` / `stop.ps1`（UTF-8 BOM + CRLF 编码，编辑时必须保持，否则中文乱码；`start.bat` / `stop.bat` 是纯 ASCII 启动器，不要往里写中文）。

手动启动：
后端：

```bash
docker compose up -d postgres        # 起 Postgres
cp .env.example .env                 # 首次需要复制
go run ./cmd/migrate up              # 建表
go run ./cmd/seedclasses             # 测试账号矩阵（幂等）；go run ./cmd/seed 为完整演示数据（覆盖密码）
go run ./cmd/server                  # 监听 :9080（.env 的 SERVER_ADDR）
go build ./...                       # 编译检查
go vet ./...
go test ./...
```

前端（开发）：

```bash
cd frontend
npm install
npm run dev    # 监听 :5173，/api 代理到 :9080
```

前端（生产）：执行 `npm run build`，Gin 会自动从 `frontend/dist` 提供静态文件并把非 `/api/*` 的路由 fallback 到 `index.html`。

Demo 账号（密码统一 `123456`）：

**含历史数据的演示账号（仅 `go run ./cmd/seed` 后存在，destructive 慎跑）：**
- student: `20230001`（李明，有申请记录和成绩）
- reviewer: `20230002`（陈欣，有申请记录和成绩）
- student: `20230003`（王悦）
- student: `20230004`（赵然）

**空白测试账号（4 班级隔离测试矩阵，详见 FLOW.md；`go run ./cmd/seedclasses` 幂等重建）：**
- student: `202300271018`（金瑞开，公信23）/ `202300271019`（刘晓，公信23）
- reviewer: `202300202018`（班委 巩浩，公信23，可用 student 身份登录）
- 公信24：reviewer `202300241001` 孙倩；student `202300241002` 周宇、`202300241003` 吴桐
- 公信25：reviewer `202300251001` 郑凯；student `202300251002` 冯雪、`202300251003` 许亮
- 公信26：reviewer `202300261001` 何静；student `202300261002` 卢鑫、`202300261003` 高媛
- admin: `zhangmiao`（张苗，管辖公信23/24）、`wangqiang`（王强，管辖公信25/26）
- super admin: `superadmin`（系统管理员，不受班级限制，唯一能分配管辖班级/任命辅导员）

测试账号可用 `go run ./cmd/seedclasses` 幂等重建（只校正角色/班级/管辖班级，**不覆盖密码**）。

## Architecture

### Backend

按职责分层，均在 `internal/` 下：

- [cmd/server/main.go](cmd/server/main.go) — 仅 wire 配置、连库、起 Gin
- [cmd/migrate/main.go](cmd/migrate/main.go) — 调用 `golang-migrate` 跑 `migrations/*.sql`
- [cmd/seed/main.go](cmd/seed/main.go) — 通过 GORM `OnConflict.UpdateAll` 灌入 demo 数据。**注意 destructive**：再次执行会覆盖账号密码哈希
- [cmd/seedclasses/main.go](cmd/seedclasses/main.go) — 幂等重建 4 班级隔离测试矩阵（1 班委 + ≥2 学生 × 4 班 + 2 辅导员管辖 2 班），不覆盖已有密码
- [internal/config](internal/config/) — 启动时自动加载工作目录的 `.env`（真实环境变量优先），读取 `DATABASE_URL` / `SERVER_ADDR` / `SESSION_SECRET` / `FRONTEND_ORIGIN` / `COOKIE_SECURE`
- [internal/repository](internal/repository/) — `gorm.Open(postgres.Open(...))`
- [internal/domain](internal/domain/) — GORM 模型 + 状态/角色常量 + DTO。**所有状态字符串的唯一来源**
- [internal/service](internal/service/) — 业务逻辑，按资源切文件（auth / applications / catalog / score）。事务用 `s.db.Transaction`；行级悲观锁用 `clause.Locking{Strength: "UPDATE"}`
- [internal/httpapi](internal/httpapi/) — Gin handlers + 中间件（`authRequired` / `requireRole` / CORS）。错误用 sentinel error（`ErrUnauthorized` / `ErrForbidden` / `ErrConflict` / `ErrNotFound`）+ `errors.Is` 判等映射 HTTP 状态码

**Session 认证**：登录返回 HTTP-only cookie；服务端在 `sessions` 表存 `sha256(secret + token)` 哈希，`AuthenticateToken` 用未撤销 + 未过期校验。所有非 `/api/health` / `/api/auth/login` 的接口都走 `authRequired`。审计日志的 actor 来自 session，不再信任客户端。

**计分**：无规则矩阵（已永久移除）。学生提交申请时自填分数（`ClaimedScore` 必填）；班委/管理员复审时可改分但必须填原因。成绩聚合见 [internal/service/score.go](internal/service/score.go)。

**数据隔离（fail-closed）**：student 只能看本人数据；reviewer 按 `class_name = 本人班级`；admin 按 `users.managed_classes`（管辖班级，jsonb 列，迁移 005）。辅导员未配置管辖班级则看不到任何班级数据；管辖班级**只能由系统管理员指定**（`PATCH /api/admin/users/{id}/classes`，仅 `is_super_admin` 账号可调用），辅导员无权自改；任命/撤销辅导员也仅系统管理员可做。活动名录全校共享，不受班级隔离影响。

**状态机**（`internal/service/applications.go::ApplicationAction`）：`待班委审核` → `公示中` → `待管理员审核` → `已通过`/`已驳回`，外加 `特殊情况处理中`（班委标记后跳过公示由管理员直接处理）与 `申诉中`（申请人本人可对被驳回/被改分申诉）。`公示中` 到期由后台 ticker 自动迁移到 `待管理员审核`（[internal/service/applications.go::AdvanceExpiredPublicity](internal/service/applications.go)），并写审计日志；启动时也会跑一次。完整流程图见 [FLOW.md](FLOW.md)。

**计分聚合**只累加 `已通过`，`公示中` 不计入。最终 `assessment = academic*0.9 + quality*0.1`（四舍五入 1 位小数）。

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
