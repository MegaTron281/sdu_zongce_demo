# PROJECT_STATE — Handoff Checkpoint

> 生成时间：2026-09-20（第 3 版，覆盖 2026-09-19 版）
> 基于 commit `b946074`（main）+ **两批未提交的工作区改动**（2026-09-19 登录错误细分/双身份/启动脚本 + 2026-09-20 数据隔离强化/管辖班级）。
> 用途：新对话仅读本文件 + 项目原始材料（CLAUDE.md / README.md / API.md / FLOW.md / CHANGELOG.md / USER_GUIDE.md / DEPLOY.md）即可无损续接工作。

---

## 1. Project Goal

- **项目**：SDU 综合测评成绩管理平台（demo/课程项目），高校综测全流程数字化：`活动名录 → 学生申请 → 班委初审 → 班级公示(48h) → 管理员复审 → 自动计分 → 成绩看板 → 导出造册`。
- **三种角色**：`student`（普通学生）、`reviewer`（审核学生/班委）、`admin`（辅导员）。
- **技术栈**：Go 1.20+ / Gin / GORM / PostgreSQL（Docker Compose 本地跑）；前端 Vue 3 + Vite + TypeScript + Pinia + vue-router。
- **当前工作的具体目标**：主流程已实现。2026-09-20 按用户要求完成了**数据隔离强化 + 辅导员管辖班级 + 4 班级测试矩阵**（已实测 25/25 项隔离验证通过），改动未提交。下一步：用户继续 UI 验收 → 提交。

## 2. Current State

### 2026-09-20 本轮已完成（未提交）
1. **修复"我的申请"隔离 bug**（用户报告：巩浩以学生身份登录能看到金瑞开的申请）。根因：旧会话无 `sessions.role` 时回退到用户表角色（reviewer）→ 按班级过滤返回全班数据。修复：迁移 004 + `AuthenticateToken` 会话角色优先；旧会话（role 为空）按最小权限把班委降级为学生视角。
2. **数据隔离全面强化（fail-closed）**：
   - `ListApplications`：student 强制 `student_id=本人`；reviewer 本班；admin 按 `managed_classes`（未配置→空）。
   - `Publicity`：同上班级过滤。
   - `StudentScoreFor`（新）：student 仅本人、reviewer 仅本班学生、admin 仅管辖班级学生（`/api/scores/:studentId` 之前任何登录用户可查任意人）。
   - `ListObjections(user, appID)`（签名变更）：按班级/管辖范围校验。
   - `appeal` 动作：之前**任何学生可对任何申请申诉**（越权漏洞）→ 现仅申请人本人 + 仅限 已驳回 或 分数被改的公示中/已通过。
   - `object` 动作：仅同班同学。
   - `ensureApplicationActor`：admin 侧也校验管辖班级。
   - `exportScores` / `ListUsers` / `UpdateUserRole`：按管辖班级过滤/校验。
3. **辅导员管辖班级功能**：`users.managed_classes`（jsonb，迁移 005）；`PATCH /api/admin/users/:id/classes`（初版只能改自己，当晚改为**仅系统管理员可指定**，见第 9 条）；AccountsAdmin.vue 管辖班级编辑；未配置管辖班级的管理员看不到任何班级数据。
4. **公示墙增加「证明材料」列**（PublicityWall.vue + MaterialList）。
5. **测试账号矩阵**：新命令 `cmd/seedclasses`（幂等，不覆盖密码）——4 班级（公信23~26）×（1 班委+2 学生）+ 2 辅导员（张苗管 23/24，王强管 25/26）。账号清单见 FLOW.md 第四节。
6. **FLOW.md**：全流程 Mermaid 状态机 + 特殊情况/复审/申诉/异议说明 + 隔离规则。
7. **测试**：`applications_test.go` 新增 9 个隔离测试（学生隔离/班委跨班/admin 管辖/申诉越权/异议越权/成绩越权/管辖班级设置），连同旧用例 15/15 PASS。
8. **CSV 身份兜底**（本轮补）：`StudentScore` 在学生无已通过申请/学业成绩时从 `users` 表补姓名班级——修复"新学生成绩行姓名班级为空"（202300241003 等验证通过）。
9. **系统管理员（2026-09-20 晚追加）**：`users.is_super_admin`（迁移 006）；唯一账号 `superadmin`/`123456`（seedclasses 创建）。权限：不受管辖班级限制（申请/公示/用户/CSV/成绩全校可见可操作）、唯一能 `PATCH /api/admin/users/:id/classes` 指定辅导员管辖班级、唯一能任命/撤销辅导员（admin 角色）；辅导员自改管辖班级已封死（403），系统管理员自身角色/管辖班级不可被修改。前端 `roleLabels.admin` 改为「辅导员」，超管显示紫色「系统管理员」徽标，管辖班级编辑器仅超管可见。**活动截止时间不限制申报**（deadline 仅是活动自身时间，已实测过去日期可创建 201）。测试改写：`TestUpdateUserClasses_SuperOnly`。

### 2026-09-19 那轮已完成（未提交，详见上一版 checkpoint）
- 登录错误细分（`ErrAccountNotFound`/`ErrRoleMismatch`/`ErrWrongPassword` 文案区分，401/400）；`loadDotEnv`（`.env` 自动加载，根因修复）；vite 代理 8090；LoginView UI 中文化；活动广场只显示已上架；start/stop 脚本；`sessions.role` 双身份会话修复（迁移 004）。

### 数据库现状（2026-09-20 实测）
- `schema_migrations` = **5**（001_init / 002_add_claimed_score / 003_objections / 004_session_role / 005_admin_classes）。
- 14 个用户（4 班级矩阵 + 2 辅导员）、3 个活动（山大杯篮球/足球/排球，已上架）、5 个申请（金瑞开 公信23：2 公示中 + 1 已通过；吴桐 公信24、冯雪 公信25：各 1 待班委审核——为验证管辖隔离所造，可留作测试数据）、无异议。
- **⚠️ 不要跑 `go run ./cmd/seed`**（会恢复 54 个演示账号并覆盖密码）；重建测试账号用 `go run ./cmd/seedclasses`。

### 本机环境快照
- Go 1.26.2、Node v24.15.0、Docker 29.4.2；`.env`（gitignore）已存在：`DATABASE_URL=postgres://sdu:sdu123@localhost:5432/sdu_assessment?sslmode=disable`、`SERVER_ADDR=:8090`（本机 8080 被系统服务占用）等。
- **gcc 已装**：TDM-GCC 10.3.0 @ `D:\TDM-GCC-64\bin`，已加用户 PATH；`go test ./...` 可用（新 shell 若找不到 gcc 先刷 PATH）。Go 检测不到 C 编译器时会自动 CGO_ENABLED=0 导致 sqlite 测试报 stub 错误——测试命令前设置 `$env:CGO_ENABLED="1"` + PATH。
- Postgres 容器 `sdu-assessment-postgres` healthy；后端 :8090 / 前端 :5173。
- Go module 名为 `demo1`（go.mod）。

### 验证记录（2026-09-20）
- `go build ./...` + `go vet ./...` + `go test ./internal/service/`（15 用例）PASS；前端 `npm run typecheck` + `npm run build` PASS。
- API 黑盒隔离矩阵 `node verify_isolation.mjs`（临时文件，25 项）全部 PASS：学生互隔离、班委本班、admin 管辖班级、成绩/异议越权 403、fail-closed、PATCH classes 往返。
- ⚠️ 经验：**PowerShell 命令行里传中文会变 `??`**（控制台编码），测试脚本用 node/fetch 或从 API 响应取中文值；`Invoke-RestMethod` 对空数组 `[]` 的计数有坑，验证用原始 body 判断。

### 启动步骤
```bash
start.bat   # 一键；或 docker compose up -d postgres && go run ./cmd/migrate up && go run ./cmd/server
# 前端：cd frontend && npm run dev
# 测试账号：go run ./cmd/seedclasses（幂等，不覆盖密码）
```

## 3. Frozen Decisions（不应被后续修改）

1. **计分规则矩阵已永久移除，改为学生自填分数**。`Application.ClaimedScore` 必填；班委 `review-pass` / 管理员 `admin-pass` 可改分但必须填原因。原因：矩阵覆盖不了"不同模块同一奖项加分不同"，且曾三份副本漂移。
2. **申诉与异议是两条独立通道**：申诉=申请人本人（被驳回/被改分）→ `申诉中`；异议=同班其他同学（公示中他人申请，不改状态）。申诉采纳=恢复原始分+重新公示 48h；驳回=分数未改回已驳回/改过回已通过。
3. **数据隔离 fail-closed（2026-09-20 冻结）**：student 仅本人；reviewer 仅 `class_name=本人班级`；admin 仅 `users.managed_classes`，**未配置管辖班级=看不到任何班级数据**。**管辖班级只能由系统管理员（`is_super_admin` 唯一账号）指定，辅导员无权自改**；任命/撤销辅导员也仅系统管理员可做。系统管理员不受管辖班级限制（全校可见）。活动名录全校共享。新增数据接口必须延续此模式（用 `s.canAccessClass` / `classesContain` / `IsSuperAdmin`）。
4. **状态字符串单一来源**：后端 `domain/models.go`（7 个申请状态），前端 `src/constants/status.ts`；视图禁止中文字面量；`API.md` 是契约。
5. **Session 认证**：HTTP-only cookie；`sessions` 表存 `sha256(secret+token)`；**会话角色（`sessions.role`，迁移 004）优先于用户表角色**——班委以 student 登录的会话内身份不漂移；旧会话 role 为空时班委降级为学生（最小权限）。审计日志 actor 取自 session。
6. **计分口径**：只累加 `已通过`；`assessment = round1(academic*0.9 + quality*0.1)`。
7. **公示规则**：48h ticker 自动推进（启动也跑）；手动 `publicity-finish` 需满 1h；`待班委审核` 可撤回；特殊情况跳过公示由管理员直接处理。
8. **汇总期锁定**：`summaryPhase` 开启后 `CreateApplication` 409。
9. **并发与事务**：事务 `s.db.Transaction`；行级锁 `clause.Locking{Strength:"UPDATE"}`。
10. **架构分层**：`cmd/*` 入口；`internal/{domain,repository,service,httpapi,config}`；domain 是状态唯一来源；repository 只做 `gorm.Open`。

## 4. Important Agreements / Constraints

- `AGENTS.md` 与 `CLAUDE.md` 内容必须保持一致（2026-09-20 已重新同步，之前欠账已清）。
- 语言：注释/UI 文案/commit message 中文；Go 标识符英文。commit 前缀 `feat:`/`fix:`/`docs:`/`style:`/`refactor:`/`test:`。
- 提交前 checklist：`go build ./...` + `go vet ./...` + `go test ./...`（gcc 已装，必须过）+ 前端 `npm run typecheck`。
- 测试放 `internal/service/*_test.go`，sqlite in-memory；新增隔离逻辑要带测试。
- `.ps1` 必须 UTF-8 BOM + CRLF；`.bat` 纯 ASCII；**PowerShell 工具命令里不要传中文**（会变 `??` 写进 DB）。
- seed（`cmd/seed`）destructive：覆盖密码；测试账号一律用 `cmd/seedclasses`。
- 新增页面：`frontend/src/router/index.ts` 的 `rolePages` 注册；新增状态：`domain/models.go` + 迁移 + `constants/status.ts` 三处同步。
- 分页 `limit`/`offset` 默认 50 最大 200；材料上传 PDF/PNG/JPG ≤10MB。
- 本机后端 :8090（`.env` + vite 代理），不要改回 8080。
- Docker 部署：应用 `/app`、Alpine、`RUN_SEED=true` 仅首次、Postgres 走 `sdu-net`。
- 文档随功能同步：`API.md`、`README.md`、`USER_GUIDE.md`、`CHANGELOG.md`、`FLOW.md`、`CLAUDE.md`/`AGENTS.md`。

## 5. Ground-truth Numbers / Configurations

- 迁移 5 版：`001_init`、`002_add_claimed_score`、`003_objections`、`004_session_role`（sessions.role，TEXT DEFAULT ''）、`005_admin_classes`（users.managed_classes，jsonb **可空**，NULL≡未配置；json 序列化字段对 nil 切片写 NULL，故不能 NOT NULL）。
- 测试账号（密码 `123456`）：见 FLOW.md 第四节表；张苗管辖 公信23/24，王强管辖 公信25/26；系统管理员 `superadmin`（不受限）。
- 库内业务数据：3 活动（山大杯篮球/足球/排球，已上架）、3 申请（app-haAmo7j-dQ0v2SHa 公示中 1.00、app-vfXx0TkA52oa7NpC 公示中 12345.00、app-iIUrd62XpGO8Q3Lw 已通过 2.00，均为金瑞开 公信23）。
- `.env`：`DATABASE_URL=postgres://sdu:sdu123@localhost:5432/sdu_assessment?sslmode=disable`、`SESSION_SECRET=dev-session-secret`、`SERVER_ADDR=:8090`、`FRONTEND_ORIGIN=http://localhost:5173`、`UPLOAD_DIR=uploads/materials`。
- compose：`postgres:16-alpine`，库 `sdu_assessment`，用户 `sdu`/`sdu123`，端口 5432，healthcheck `pg_isready`。
- 公示 48h；手动结束公示最短 1h；申诉采纳重新公示 48h；密码 ≥6 位；CSV 导入默认密码 `123456`。
- gcc：TDM-GCC 10.3.0 @ `D:\TDM-GCC-64\bin`（用户 PATH）。

## 6. Rejected / Deprecated Ideas（勿再提出）

1. **自动计分规则矩阵 / RuleVersion / `POST /api/scores/preview`** — 已删除。现行为：学生自填 + 审核端人工改分（须填原因）。
2. **前端自己算分数** — 已废弃，前端不参与计分。
3. **单文件 main.go + 原生 JS 原型架构** — 已按 `plan.md` 重构；`plan.md` 仅历史参考。
4. **客户端请求体自填 actor** — 已废弃，actor 取自 session。
5. **`alert()`/`prompt()` 交互、innerHTML 渲染** — 已被 Vue 组件 + toast 替代。
6. **CORS `*`** — 已收紧为 `FRONTEND_ORIGIN` 白名单。
7. **把公示中的申请计入总分** — 只累加「已通过」。
8. **GBK .bat / 非 BOM UTF-8 .ps1** — 已踩坑；.bat 纯 ASCII、.ps1 UTF-8 BOM + CRLF。
9. **在 PowerShell 命令行/管道里传中文字符串写库** — 会变 `??`（2026-09-20 实测踩坑：张苗管辖班级一度写成 `??23`）；中文数据一律走 Go 程序（如 seedclasses）或 API 响应回传值。
10. **管理员无管辖班级时看到全校数据** — 否决，fail-closed：未配置=空。
