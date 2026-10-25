# PROJECT_STATE — Handoff Checkpoint

> 生成时间：2026-09-21（v4，覆盖 2026-09-20 v3）
> 用途：新对话仅读本文件 + 项目原始材料（`CLAUDE.md`/`AGENTS.md`/`README.md`/`API.md`/`FLOW.md`/`CHANGELOG.md`/`USER_GUIDE.md`/`DEPLOY.md`）即可无损续接工作。
> 本文件是「状态存档」而非对话总结：只保留事实、约定、决策、约束、Ground-truth 数值；已失效的讨论过程与无关内容已删除。

> ⚠️ **本机不是 git 仓库**（工作区根目录无 `.git`，无 commit、无分支）。所有改动都是「未提交的工作区文件」，这一事实本身就是一个约束：没有版本回退手段，改动前先备份，且无法用 `git diff` 审阅历史。v3 中「基于 commit b946074」的说法有误，已删除。

---

## 1. Project Goal

- **项目**：SDU 综合测评成绩管理平台（demo / 课程项目）。高校综测全流程数字化：
  `活动名录 → 学生申请（自填分）→ 班委初审 → 班级公示(48h) → 管理员复审 → 自动计分 → 成绩看板 → 导出造册`。
- **三种角色**：`student`（普通学生）、`reviewer`（审核学生 / 班委）、`admin`（辅导员，其中带 `is_super_admin` 标志的唯一账号是**系统管理员**）。
- **技术栈**：后端 Go（module 名 `demo1`）/ Gin / GORM / PostgreSQL（Docker Compose 本地跑）；前端 Vue 3 + Vite + TypeScript + Pinia + vue-router。前端生产构建由 Gin 托管 `frontend/dist`。
- **当前工作的具体目标**：主流程 + 数据隔离 + 管辖班级 + 系统管理员均已实现并通过验证（详见 §2）。下一步：用户做 UI 验收 → 视情况提交（注意无 git，提交指另存备份 / 打包，而非 `git commit`）。

---

## 2. Current State

### 已实现（截至 2026-09-21，全部工作区改动、未进版本控制）

1. **认证与会话**：登录写 HTTP-only session cookie；`sessions` 表存 `sha256(secret+token)`；`AuthenticateToken` 校验未撤销 + 未过期。**会话角色（`sessions.role`，迁移 004）优先于用户表角色**——班委以 student 身份登录时会话内身份不漂移；旧会话 `role` 为空时班委降级为学生视角（最小权限）。登录错误细分：`ErrAccountNotFound`/`ErrRoleMismatch`/`ErrWrongPassword`（401/400 文案区分）。审计日志 actor 取自 session，不再信任客户端。
2. **学生自填分数（规则矩阵已移除）**：`Application.ClaimedScore` 必填；班委 `review-pass` / 管理员 `admin-pass` 可改分但必须填原因。
3. **公示异议系统**：`object`（同班同学对公示中他人申请提异议，不改状态）+ `ListObjections`（按班级/管辖范围校验）。
4. **申诉系统**：`appeal`（仅申请人本人，限已驳回或被改分的公示中/已通过）/ `appeal-approve`（恢复原始分 + 重新公示 48h）/ `appeal-reject`。
5. **数据隔离 fail-closed（2026-09-20 冻结）**：
   - `ListApplications`：student 强制 `student_id=本人`；reviewer 本班；admin 按 `managed_classes`（未配置→空）。
   - `Publicity` / `StudentScoreFor`（`/api/scores/:studentId` 全角色越权校验）/ `ListObjections` / `exportScores` / `ListUsers` / `UpdateUserRole` 全部按班级或管辖范围过滤。
   - `appeal` 仅本人、`object` 仅同班——封死越权漏洞。
6. **辅导员管辖班级**：`users.managed_classes`（jsonb，迁移 005）；`PATCH /api/admin/users/:id/classes` **仅系统管理员可指定**，辅导员自改返回 403。未配置管辖班级的辅导员看不到任何班级数据。
7. **系统管理员**：`users.is_super_admin`（迁移 006）；唯一账号 `superadmin`/`123456`；不受班级限制（全校可见可操作）、唯一能分配管辖班级、唯一能任命/撤销辅导员；自身角色与管辖范围不可被接口修改。前端 `roleLabels.admin`=「辅导员」，超管紫色「系统管理员」徽标，管辖班级编辑器仅超管可见。
8. **公示墙「证明材料」列**（`PublicityWall.vue` + `MaterialList`）。
9. **CSV 身份兜底**：`StudentScore` 在学生无已通过申请/学业成绩时从 `users` 表补姓名班级（修复新学生成绩行姓名班级为空）。
10. **测试账号矩阵**：`cmd/seedclasses`（幂等，不覆盖密码）——4 班级（公信23~26）×（1 班委 + 2 学生）+ 2 辅导员（张苗管 23/24，王强管 25/26）+ 1 超管 `superadmin`。
11. **文档**：`FLOW.md`（状态机 + 隔离规则 + 账号矩阵）、`API.md`、`CHANGELOG.md`、`USER_GUIDE.md`、`DEPLOY.md`、`CLAUDE.md`/`AGENTS.md`（内容一致）。

### 验证记录（2026-09-20，仍有效）

- `go build ./...` + `go vet ./...` + `go test ./internal/service/`（15 用例）PASS；前端 `npm run typecheck` + `npm run build` PASS。
- API 黑盒隔离矩阵 `node verify_isolation.mjs`（临时文件，25 项）全部 PASS：学生互隔离、班委本班、admin 管辖班级、成绩/异议越权 403、fail-closed、PATCH classes 往返。
- ⚠️ **经验**：PowerShell 命令行里传中文会变成 `??`（控制台编码），测试脚本用 node/fetch 或从 API 响应取中文值；`Invoke-RestMethod` 对空数组 `[]` 计数有坑，验证用原始 body 判断。

### 数据库现状（2026-09-20 实测，未跑过 `go run ./cmd/seed`）

- 迁移文件共 **6 个**（见 §5）；若 `migrate up` 在加 006 之后跑过，则 `schema_migrations` = 6。
- 业务数据：3 活动（山大杯篮球/足球/排球，已上架）、**5 个申请**（金瑞开 公信23：2 公示中 + 1 已通过；吴桐 公信24、冯雪 公信25：各 1 待班委审核——为验证管辖隔离所造，可留作测试数据）、无异议。
- ⚠️ **不要跑 `go run ./cmd/seed`**（会恢复 54 个演示账号并覆盖密码）；重建测试账号用 `go run ./cmd/seedclasses`。

### 本机环境快照

- Go 1.26.2、Node v24.15.0、Docker 29.4.2；`.env`（gitignore）已存在：`DATABASE_URL=postgres://sdu:sdu123@localhost:5432/sdu_assessment?sslmode=disable`、`SERVER_ADDR=:9080`（本机 8080 被系统服务占用）。
- **gcc 已装**：TDM-GCC 10.3.0 @ `D:\TDM-GCC-64\bin`，已加用户 PATH；`go test ./...` 需用 gcc（sqlite in-memory）。Go 检测不到 C 编译器时会自动 `CGO_ENABLED=0` 导致 sqlite 测试报 stub 错误——测试前设 `$env:CGO_ENABLED="1"` + PATH。
- Postgres 容器 `sdu-assessment-postgres` healthy；后端 :9080 / 前端 :5173。
- Go module 名 `demo1`（go.mod）。

### 启动步骤

```bash
start.bat   # 一键；或 docker compose up -d postgres && go run ./cmd/migrate up && go run ./cmd/server
# 前端：cd frontend && npm run dev
# 测试账号：go run ./cmd/seedclasses（幂等，不覆盖密码）
```

---

## 3. Frozen Decisions（不应被后续 Chat 随意修改；每项附原因）

1. **计分规则矩阵已永久移除，改为学生自填分数**。`Application.ClaimedScore` 必填；审核端可改分但必须填原因。原因：矩阵覆盖不了「不同模块同一奖项加分不同」，且曾三份副本漂移。→ 勿恢复 `RuleVersion` / `rule_versions` 表 / `POST /api/scores/preview`。
2. **申诉与异议是两条独立通道**：申诉=申请人本人（被驳回/被改分）→ `申诉中`；异议=同班其他同学（公示中他人申请，不改状态）。申诉采纳=恢复原始分+重新公示 48h；驳回=维持原状态（已通过/已驳回）。
3. **数据隔离 fail-closed（2026-09-20 冻结）**：student 仅本人；reviewer 仅 `class_name=本人班级`；admin 仅 `users.managed_classes`，**未配置管辖班级=看不到任何班级数据**。**管辖班级只能由系统管理员（`is_super_admin` 唯一账号）指定，辅导员无权自改**；任命/撤销辅导员也仅系统管理员可做。系统管理员不受管辖班级限制（全校可见）。活动名录全校共享。新增数据接口必须延续此模式（用 `s.canAccessClass` / `classesContain` / `IsSuperAdmin`）。
4. **状态字符串单一来源**：后端 `domain/models.go`（7 个申请状态），前端 `src/constants/status.ts`；视图禁止中文字面量；`API.md` 是契约。新增状态：三处同步（domain 常量 + 迁移 + `constants/status.ts` 颜色）。
5. **Session 认证**：HTTP-only cookie；`sessions` 表存 `sha256(secret+token)`；**会话角色（`sessions.role`）优先于用户表角色**；旧会话 role 为空时班委降级为学生（最小权限）。审计日志 actor 取自 session。
6. **计分口径**：只累加 `已通过`；`assessment = round1(academic*0.9 + quality*0.1)`（四舍五入 1 位小数）。
7. **公示规则**：48h ticker 自动推进（启动也跑一次）；手动 `publicity-finish` 需满 1h；`待班委审核` 可 `withdraw` 撤回；特殊情况（`mark-special`）跳过公示由管理员直接处理。
8. **汇总期锁定**：`summaryPhase` 开启后 `CreateApplication` 返回 409。
9. **并发与事务**：事务 `s.db.Transaction`；行级悲观锁 `clause.Locking{Strength:"UPDATE"}`。
10. **架构分层**：`cmd/*` 入口；`internal/{domain,repository,service,httpapi,config}`；domain 是状态/角色常量唯一来源；repository 只做 `gorm.Open`。新增页面在 `frontend/src/router/index.ts` 的 `rolePages` 注册。
11. **活动截止时间不限制申报**：`deadline` 仅是活动自身时间，受控的只有汇总期锁（`summaryPhase`）。已实测过去日期可创建申请。

---

## 4. Important Agreements / Constraints

- `AGENTS.md` 与 `CLAUDE.md` 必须内容一致（2026-09-20 已同步）。
- **本机无 git**：无法提交/回退。重要改动前先手动备份（如复制目录或打包）；不要假设能用 `git` 审阅或回滚。
- 语言：注释 / UI 文案 / commit message 中文；Go 标识符英文。commit 前缀 `feat:`/`fix:`/`docs:`/`style:`/`refactor:`/`test:`（仅作书写约定，非真 git）。
- 提交前 checklist（验证用）：`go build ./...` + `go vet ./...` + `go test ./...`（gcc 已装，必须过）+ 前端 `npm run typecheck`。
- 测试放 `internal/service/*_test.go`，sqlite in-memory（需 `CGO_ENABLED=1`）；新增隔离逻辑必须带测试。
- `.ps1` 必须 UTF-8 BOM + CRLF；`.bat` 纯 ASCII（不要往 .bat 写中文）；**PowerShell 工具命令里不要传中文**（会变 `??` 写进 DB）。
- seed（`cmd/seed`）destructive：覆盖密码；测试账号一律用 `cmd/seedclasses`。
- 分页 `limit`/`offset` 默认 50 最大 200；材料上传 PDF/PNG/JPG ≤10MB。
- 本机后端 `:9080`（`.env` + vite 代理），不要改回 8080。
- Docker 部署：应用 `/app`、Alpine、`RUN_SEED=true` 仅首次、Postgres 走 `sdu-net`。
- 文档随功能同步：`API.md`、`README.md`、`USER_GUIDE.md`、`CHANGELOG.md`、`FLOW.md`、`CLAUDE.md`/`AGENTS.md`。

---

## 5. Ground-truth Numbers / Configurations（已确认，可信任）

- **迁移共 6 个**（目录 `migrations/`）：
  `001_init` / `002_add_claimed_score`（加 `claimed_score`，删 `rule_versions`）/ `003_objections`（建 `objections` 表）/ `004_session_role`（sessions.role，TEXT DEFAULT ''）/ `005_admin_classes`（`users.managed_classes`，jsonb **可空**，NULL≡未配置；json 序列化对 nil 切片写 NULL，故不能 NOT NULL）/ `006_super_admin`（`users.is_super_admin`）。
- **测试账号矩阵（密码统一 `123456`，共 15 个）**：
  - 公信23：班委 `202300202018` 巩浩；学生 `202300271018` 金瑞开、`202300271019` 刘晓；管辖辅导员 张苗 `zhangmiao`
  - 公信24：班委 `202300241001` 孙倩；学生 `202300241002` 周宇、`202300241003` 吴桐；管辖辅导员 张苗 `zhangmiao`
  - 公信25：班委 `202300251001` 郑凯；学生 `202300251002` 冯雪、`202300251003` 许亮；管辖辅导员 王强 `wangqiang`
  - 公信26：班委 `202300261001` 何静；学生 `202300261002` 卢鑫、`202300261003` 高媛；管辖辅导员 王强 `wangqiang`
  - 系统管理员：`superadmin`（唯一，不受班级限制，负责分配管辖班级与任命辅导员）
  - 重建命令：`go run ./cmd/seedclasses`（幂等，不覆盖密码，同时保障超管账号存在）
- **历史 demo 账号**（仅 `go run ./cmd/seed` 后存在，destructive 慎跑）：student `20230001` 李明 / `20230002` 陈欣(reviewer) / `20230003` 王悦 / `20230004` 赵然；密码 `123456`。
- **库内业务数据**：3 活动（山大杯篮球/足球/排球，已上架）；5 申请（金瑞开 公信23：`app-haAmo7j-dQ0v2SHa` 公示中 1.00、`app-vfXx0TkA52oa7NpC` 公示中 12345.00、`app-iIUrd62XpGO8Q3Lw` 已通过 2.00；吴桐 公信24、冯雪 公信25 各 1 待班委审核）。
- **`.env`**：`DATABASE_URL=postgres://sdu:sdu123@localhost:5432/sdu_assessment?sslmode=disable`、`SESSION_SECRET=dev-session-secret`、`SERVER_ADDR=:9080`、`FRONTEND_ORIGIN=http://localhost:5173`、`UPLOAD_DIR=uploads/materials`。
- **compose**：`postgres:16-alpine`，库 `sdu_assessment`，用户 `sdu`/`sdu123`，端口 5432，healthcheck `pg_isready`。
- **业务常量**：公示 48h；手动结束公示最短 1h；申诉采纳重新公示 48h；密码 ≥6 位；CSV 导入默认密码 `123456`；学业分占 0.9、素质分占 0.1（`assessment = round1(academic*0.9 + quality*0.1)`）；只累加「已通过」。
- **gcc**：TDM-GCC 10.3.0 @ `D:\TDM-GCC-64\bin`（用户 PATH）。
- **Go module 名**：`demo1`。

---

## 6. Rejected / Deprecated Ideas（勿再提出）

1. **自动计分规则矩阵 / `RuleVersion` / `rule_versions` 表 / `POST /api/scores/preview`** — 已删除，现行为学生自填 + 审核端人工改分（须填原因）。
2. **前端自己算分数** — 已废弃，前端不参与计分。
3. **单文件 `main.go` + 原生 JS 原型架构** — 已按 `plan.md`（Code Review）重构为分层结构；`plan.md` 仅历史参考。
4. **客户端请求体自填 actor** — 已废弃，actor 取自 session。
5. **`alert()`/`prompt()` 交互、`innerHTML` 渲染** — 已被 Vue 组件 + toast 替代。
6. **CORS `*`** — 已收紧为 `FRONTEND_ORIGIN` 白名单。
7. **把公示中的申请计入总分** — 只累加「已通过」。
8. **GBK .bat / 非 BOM UTF-8 .ps1** — 已踩坑；.bat 纯 ASCII、.ps1 UTF-8 BOM + CRLF。
9. **在 PowerShell 命令行/管道里传中文字符串写库** — 会变 `??`（实测：张苗管辖班级一度写成 `??23`）；中文数据一律走 Go 程序（如 seedclasses）或 API 响应回传值。
10. **管理员无管辖班级时看到全校数据** — 否决，fail-closed：未配置=空。
11. **辅导员自改管辖班级 / 辅导员自任命辅导员** — 否决，仅系统管理员可操作（006）。
12. **活动截止时间限制综测申报** — 否决，deadline 仅活动自身时间，受控的只有汇总期锁。
