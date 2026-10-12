# 更新日志

## 2026-10-12 修复：前端 `npm run dev` 启动报 `EACCES: permission denied 0.0.0.0:5173`

---

### 一、问题

| 现象 | 根因 |
|------|------|
| 运行 `npm run dev` 直接崩溃：`Error: listen EACCES: permission denied 0.0.0.0:5173` | 旧 `dev` 脚本为 `vite --host 0.0.0.0`（通配绑定）。在 Windows 上端口 5173 落在系统保留/排除段（Hyper-V / WSL / WinNAT，`netsh int ipv4 show excludedportrange tcp` 可见 `5122–5221` 等），通配绑定命中保留段被系统拒绝。且 Vite 只在 `EADDRINUSE`（端口被占用）时自动顺延端口，**不会**从 `EACCES`（端口被系统保留）恢复，因此首端口即崩溃 |

### 二、变更

| 位置 | 说明 |
|------|------|
| `frontend/package.json` | `dev` 脚本由 `vite --host 0.0.0.0` 改回 `vite`（默认仅绑定回环 `localhost`） |
| `frontend/vite.config.ts` | 新增 `findAvailablePort()`：启动时主动探测可用端口，**同时跳过 `EACCES`（保留段）与 `EADDRINUSE`（已被占用）**，彻底规避保留段导致的崩溃；`server.host` 默认 `localhost`，`strictPort: true` 使用探测到的端口；通过 `npm run dev -- --host` 仍可按需开放局域网访问（该场景探测改用 `0.0.0.0`） |
| `frontend/package.json` / `package-lock.json` | 新增开发依赖 `@types/node`（config 用到 `node:net` 类型，保证 `vue-tsc` 通过） |

### 三、验证

- 本机（端口 5173 处于保留段）启动成功，自动选用首个可用端口 `5322`，`GET /` 返回 200（SPA 外壳），`/api/health` 代理返回 200，后端存活。
- `vue-tsc --noEmit` 类型检查通过。

---

## 2026-10-12 修复：同一浏览器多账号并行登录（后登录顶掉先登录）

---

### 一、问题

| 现象 | 根因 |
|------|------|
| 在一个浏览器里先登录账号 A，再登录账号 B，A 就"掉线"，只能使用最新的 B | 后端每次登录都会**新建一条 session 记录且不会吊销旧 session**（服务端其实早已支持并发会话）；"掉线"纯粹是因为浏览器对每个同源请求只自动携带**一个同名 cookie**（`sdu_session`），B 登录后该 cookie 被覆盖为 B 的令牌，所有标签页都改发 B 的令牌，A 标签页也随之变成 B |

### 二、变更

| 位置 | 说明 |
|------|------|
| `internal/httpapi/middleware.go` | `authRequired` 优先读取 `Authorization: Bearer <token>` 请求头，其次回退到 cookie；新增 `bearerToken` 辅助函数提取令牌 |
| `internal/httpapi/handlers.go` | 登录接口在响应体里额外返回 `token`；退出接口优先用 Bearer 头里的令牌（按标签页精确吊销） |
| `internal/domain/dto.go` | `LoginResponse` 增加 `Token` 字段 |
| `frontend/src/api/client.ts` | 新增 `setSessionToken` / `getSessionToken`；每个请求自动附带本标签页独立保存的 `Authorization: Bearer` 头；`login` 返回 `token` |
| `frontend/src/stores/auth.ts` | 登录成功把令牌存入 `sessionStorage`（**按标签页隔离**，关闭标签页即失效），登出时清除 |

### 三、效果与用法

- 现在可在**同一浏览器的不同标签页 / 窗口**分别登录不同账号（学生 / 班委 / 辅导员 / 超管），各自保持独立登录态，互不顶替，方便走完整流程（申报 → 初审 → 公示 → 复审）。
- 单账号场景完全兼容：未携带 Bearer 头时仍走 cookie，刷新页面、新开空白标签页等行为不变。
- 权衡：`sessionStorage` 中的令牌是 JS 可读的（不像 HttpOnly cookie 那样防 XSS 读取），本原型可接受；若日后渲染不可信内容，建议配合 CSP 与令牌短期过期。

### 四、验证

- `go build ./...`、`go vet ./...`、`vue-tsc --noEmit` 全部通过。
- 真实接口并行验证（需先启动 Docker/Postgres 与前后端）：用 Bearer 头分别携带 A、B 两个令牌请求 `/api/auth/me`，均返回各自账号，证明并行会话互不干扰。

---

## 2026-09-22 修复：学生申报会重复生成同名活动

---

### 一、问题

| 现象 | 根因 |
|------|------|
| 学生从「活动广场」选一个已有活动去申报，辅导员「活动名录管理 → 学生申报待审」里又冒出同名活动；点「确认上架」后活动名录出现重复条目 | 学生申报页只把活动信息**预填**到表单，提交时**没有回传 `activityId`**；后端收到空 `activityId` 就按「学生补录活动」处理，新建了一条「待审核上架」活动 |
| 活动名录里出现同名活动，既无意义又容易选错 | `CreateActivity` 对活动名没有任何唯一性约束 |

### 二、变更

| 位置 | 说明 |
|------|------|
| `frontend/src/views/student/StudentSubmit.vue` | 从「活动广场」带入的 `activityId` 会随申报一起提交；此时活动名称/主办方/级别/类别/日期/来源/规则**锁定不可改**，只填获奖情况与申报分数，页面标题变为「申报活动（活动信息取自活动名录）」 |
| `internal/service/applications.go` | 新增 `resolveApplicationActivity`：① 带 `activityId` → 直接复用该活动；② 未带但活动名在名录中命中同名且未驳回的活动 → **复用**该活动；③ 名录中确实没有同名活动 → 才新建「待审核上架」活动 |
| `internal/service/catalog.go` | `CreateActivity` 增加同名校验，重复创建返回 400「活动名录中已存在同名活动「X」，请勿重复创建」 |
| `internal/service/applications.go` | 重复申报校验（同一学生 + 同一活动 + 同一周期，排除已驳回）统一在活动确定之后执行，补录路径同样受保护 |

### 三、数据修复

- 清理了库里历史遗留的 5 条重复活动条目：先把挂在重复条目上的 6 条申请重定向到名录中的规范活动（活动名不变），再删除重复条目。
- 清理后：活动 5 条、无同名；申请 6 条，全部指向存在的活动。
- 备份：`.workbuddy/backup/activities_applications_20260922.sql`（`--column-inserts`，可直接回灌）。

### 四、验证

- `go build ./...`、`go vet ./...` 通过；前端 `vue-tsc --noEmit` 通过。
- 真实接口端到端 **16/16 通过**：带 `activityId` 申报不再新建活动、自填同名活动会复用、重复申报仍返回 409、新建同名活动被拒、新建唯一名活动成功。

---

## 2026-09-20 数据隔离强化 + 管辖班级

---

### 一、数据隔离（fail-closed）

| 规则 | 说明 |
|------|------|
| 学生隔离 | `我的申请` 后端强制 `student_id = 登录人`，任何账号（含班委的学生身份）只能看到本人申请 |
| 班委隔离 | 申请列表/审核操作/公示墙/成绩查询全部按 `class_name = 本人班级` 过滤，跨班操作 403 |
| 管理员管辖班级 | 新增 `users.managed_classes`（迁移 005）；复审/申诉/成绩总览/导出/账号管理只展示管辖班级数据；未配置则什么都看不到 |
| 系统管理员 | 新增 `users.is_super_admin`（迁移 006）；唯一账号 `superadmin` 不受班级限制，是唯一能分配管辖班级、任命/撤销辅导员的身份；辅导员自改管辖班级已封死 |
| 越权修复 | 申诉只能由申请人本人发起（且仅限被驳回或分数被改）；异议仅限同班同学；异议列表按班级/管辖范围校验 |

### 二、新增/变更 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `PATCH` | `/api/admin/users/{id}/classes` | **仅系统管理员**可指定任意辅导员的管辖班级（辅导员无权自改） |
| 变更 | `PATCH /api/admin/users/{id}/role` | 辅导员只能在本班 student ↔ reviewer 间调整；任命/撤销辅导员仅系统管理员 |
| 变更 | `GET /api/applications` | admin 增加管辖班级过滤（系统管理员全校）；student 强制本人 |
| 变更 | `GET /api/scores/{studentId}` | 三种角色全部加越权校验 |
| 变更 | `GET /api/applications/{id}/objections` | 按班级/管辖范围校验 |

### 三、其他

- 公示墙新增「证明材料」列（`PublicityWall.vue`）
- 「账号与班级管理」页：管辖班级编辑器仅系统管理员可见；辅导员角色显示为「辅导员」，超管显示「系统管理员」徽标
- 明确：活动截止时间仅为活动自身时间，**不限制综测申报**（已实测过去日期可提交）
- 登录错误细分：`该账号不存在` / `密码错误` / `该账号与所选登录身份不符`；登录页去预填、密码提示改「默认密码：123456」
- 修复：学生无已通过申请时导出 CSV/成绩总览姓名班级为空（`StudentScore` 从用户表兜底）
- 新增 `cmd/seedclasses`：幂等重建 4 班级 ×（1 班委 + 2 学生）+ 2 辅导员（各管辖 2 班）+ 系统管理员，不覆盖密码
- 新增 `FLOW.md`：系统全流程图（状态机 + 特殊情况/复审/申诉/异议流程 + 隔离规则）
- 修复：旧会话无角色信息时班委降级为学生视角（最小权限），需重新登录恢复班委身份
- 迁移：`004_session_role`（sessions.role）、`005_admin_classes`（users.managed_classes）、`006_super_admin`（users.is_super_admin）

---

## 2026-06-06

---

### 一、Docker 部署支持

**新增文件**

| 文件 | 说明 |
|------|------|
| `Dockerfile` | 三阶段构建（Node 编前端 → Go 编后端 → Alpine 运行时），工作目录 `/app`，持久化数据目录 `/data` |
| `docker-entrypoint.sh` | 容器入口：自动运行数据库迁移 → 可选灌入种子数据（`RUN_SEED=true`） → 启动服务 |
| `DEPLOY.md` | 逐步部署指南，含每条命令和参数说明 |

**关键设计**
- 应用文件放 `/app`（平台挂载 `/data` 不覆盖）
- 构建与运行统一使用 Alpine（避免 glibc/musl 跨链接问题）
- PostgreSQL 容器独立部署，通过 `sdu-net` 桥接网络通信

---

### 二、账号管理

**新增功能**

| 功能 | 说明 |
|------|------|
| CSV 批量导入学生 | 管理员上传 CSV（学号,姓名,班级），批量创建账号，默认密码 `123456`，已存在自动跳过 |
| 修改密码 | 所有角色登录后可修改密码（旧密码验证 → 新密码 ≥ 6 位） |
| 角色管理 | 管理员可将学生提升为班委 / 取消班委 |

**新增 API**

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/auth/change-password` | 修改当前用户密码 |
| `GET` | `/api/admin/users` | 用户列表（分页） |
| `POST` | `/api/admin/users/import` | CSV 导入学生 |
| `PATCH` | `/api/admin/users/:id/role` | 修改用户角色 |

**后端文件**

| 文件 | 改动 |
|------|------|
| `internal/service/users.go` | 新建 — `ListUsers`、`ImportUsersCSV`、`ChangePassword`、`UpdateUserRole` |
| `internal/service/auth.go` | 新增 `ErrWrongPassword` |
| `internal/domain/dto.go` | 新增 `ChangePasswordRequest`、`UserListItem`、`UserImportResult`、`UpdateRoleRequest` |
| `internal/httpapi/handlers.go` | 新增 4 个 handler |
| `internal/httpapi/router.go` | 新增 4 条路由 |
| `internal/httpapi/middleware.go` | `ErrWrongPassword` → 400 |

**前端文件**

| 文件 | 改动 |
|------|------|
| `frontend/src/api/types.ts` | 新增 `UserListItem`、`UserImportResult` |
| `frontend/src/api/client.ts` | 新增 `changePassword`、`listUsers`、`importUsers`、`updateUserRole` |
| `frontend/src/components/ChangePassword.vue` | 新建 — 修改密码弹窗（居中卡片，带入场动画） |
| `frontend/src/layouts/ShellLayout.vue` | 顶栏加「修改密码」按钮和弹窗入口 |
| `frontend/src/views/admin/AccountsAdmin.vue` | 重写 — CSV 导入区 + 用户列表 + 角色管理 + 周期切换 |
| `frontend/src/assets/styles.css` | 新增弹窗样式（`.pwd-modal-overlay`、`.pwd-modal-card` 等） |

---

### 三、学生自填分数 + 申诉系统

**需求背景**

原计分规则矩阵无法覆盖不同模块同一奖项加分不同的情况，改为学生自行填写申报分数。

**删除的模块**

| 模块 | 说明 |
|------|------|
| `domain.RuleVersion` 模型 | 加分规则矩阵已移除 |
| `service/score.go` 中 `ActiveRule`、`CalculateScore`、`PreviewScore` | 不再自动计分 |
| `service/catalog.go` 中 `CreateRule`、`ListRules` | 不再维护规则 |
| 前端 `RuleAdmin.vue`、`ScorePreview.vue` | 规则配置页和计分预览已删除 |
| 前端路由 `/admin/rules` | 已移除 |
| 数据库 `rule_versions` 表 | 迁移 `002` 已删除 |

**新增内容**

| 新增 | 说明 |
|------|------|
| `Application.ClaimedScore` 字段 | 记录学生原始申报分数 |
| `StatusAppealing = "申诉中"` | 新增申请状态 |
| 学生自填分数 | `ApplicationCreateRequest.Score`（必填） |
| 班委改分 | `review-pass` 接受 `Score` 参数，改分需填原因 |
| 申诉 action | `appeal`（学生）、`appeal-approve`（管理员采纳）、`appeal-reject`（管理员驳回） |

**申诉流程**

```
学生申诉 → 申诉中 → 管理员采纳 → 重新公示(48h) → 自动送复审 → 通过
                  → 管理员驳回 → 恢复原状态（已通过/已驳回）
```

- 被驳回 → 可申诉
- 公示中分数被改 → 可申诉（不需要等到复审结束）
- 已通过分数被改 → 可申诉
- 管理员采纳后重新进入公示，其他同学可提异议

**后端文件**

| 文件 | 改动 |
|------|------|
| `internal/domain/models.go` | 删 `RuleVersion`，加 `ClaimedScore`、`StatusAppealing` |
| `internal/domain/dto.go` | 删 `RuleSummary`、`ScorePreview*`，`ApplicationCreateRequest` 加 `Score` |
| `internal/service/score.go` | 删计分相关，只保留 `StudentScore` |
| `internal/service/catalog.go` | 删规则相关，`Meta()` 移除 `ActiveRule` |
| `internal/service/applications.go` | `CreateApplication` 用自填分，`review-pass` 支持改分，加 `appeal` 三个 action，`appeal-approve` 重新公示 |
| `internal/service/applications_test.go` | 适配改动 |
| `internal/service/score_test.go` | 删除 |
| `internal/httpapi/router.go` | 移除 rules/preview 路由 |
| `internal/httpapi/handlers.go` | 移除规则相关 handler |
| `cmd/seed/main.go` | 移除 `RuleVersion` 种子数据 |
| `migrations/002_add_claimed_score.up.sql` | 加 `claimed_score` 列，删 `rule_versions` 表 |

**前端文件**

| 文件 | 改动 |
|------|------|
| `frontend/src/components/ScorePreview.vue` | 删除 |
| `frontend/src/views/admin/RuleAdmin.vue` | 删除 |
| `frontend/src/router/index.ts` | 移除 `admin.rules` |
| `frontend/src/api/types.ts` | 删 `RuleVersion`、`ScorePreview`，`Application` 加 `claimedScore` |
| `frontend/src/api/client.ts` | 删 `preview`、`rules`、`createRule`，`createApplication` 加 `score` |
| `frontend/src/stores/workspace.ts` | 移除 `rules` 状态 |
| `frontend/src/constants/status.ts` | 新增 `Appealing: "申诉中"`（amber） |
| `frontend/src/views/student/StudentSubmit.vue` | 移除 `ScorePreview`，加自填分数输入框 |
| `frontend/src/views/student/StudentApplications.vue` | 显示申报分/最终分，申诉弹窗 |
| `frontend/src/views/reviewer/ReviewDesk.vue` | 加分数修改，驳回/特殊情况弹窗 |
| `frontend/src/views/admin/RecheckDesk.vue` | 分数修改 + 异议展示 + 驳回弹窗 |

---

### 四、公示异议系统

**新增功能**

| 功能 | 说明 |
|------|------|
| 提出异议 | 公示期间，学生对**别人的**申请提出异议，填写理由 |
| 管理员查看异议 | 复审工作台显示每条申请的异议列表 |

**与申诉的区别**

| | 申诉 | 异议 |
|------|------|------|
| 谁提 | 申请人自己 | 同班其他同学 |
| 针对 | 自己被驳回/改分 | 别人公示中的申请 |
| 影响状态 | 有（→ 申诉中） | 不影响 |
| 入口 | 「我的申请」页 | 「公示墙」页 |

**新增内容**

| 文件 | 改动 |
|------|------|
| `internal/domain/models.go` | 新增 `Objection` 模型 |
| `internal/domain/dto.go` | 新增 `ObjectionRequest` |
| `internal/service/applications.go` | 新增 `object` action + `ListObjections` 方法 |
| `internal/httpapi/router.go` | 新增 `GET /api/applications/:id/objections` |
| `internal/httpapi/handlers.go` | 新增 `listObjections` handler |
| `migrations/003_objections.up.sql` | 新建 `objections` 表 |
| `frontend/src/api/types.ts` | 新增 `Objection` 类型 |
| `frontend/src/api/client.ts` | 新增 `objections` 方法 |
| `frontend/src/views/student/PublicityWall.vue` | 重写 — 显示申报分/最终分、异议弹窗（仅他人可见） |
| `frontend/src/views/admin/RecheckDesk.vue` | 展示异议列表 |

---

### 五、班委双重身份支持

**问题**：班委被设为学生→班委后只能以班委身份登录，无法访问学生端。

**修复**：班委账号可用两种身份分别登录，互不干扰。

| 登录身份 | 适用 | 看到的页面 |
|----------|------|-----------|
| 普通学生 | 班委以 student 登录 | 活动广场、提交申请、我的申请、我的成绩、公示墙 |
| 审核学生 | 班委以 reviewer 登录 | 审核工作台、班级申请历史 |

**实现**：`internal/service/auth.go` Login 方法中，以 `student` 身份登录时按账号查找（不限制 role），Session 中 role 记为 `student`。

---

### 六、管理员申诉工作台

**问题**：申诉处理与复审混在同一页面，逻辑不清。

**修复**：拆分为两个独立工作台。

| 工作台 | 处理 | 状态 |
|--------|------|------|
| 复审工作台 | 正常复审 | 待管理员审核、特殊情况处理中 |
| 申诉工作台 | 学生申诉 | 申诉中 |

**新增文件**

| 文件 | 说明 |
|------|------|
| `frontend/src/views/admin/AppealDesk.vue` | 申诉工作台 — 查看申诉理由、采纳（恢复分数+重新公示）/ 驳回申诉 |

---

### 七、Bug 修复与优化

| 问题 | 修复 |
|------|------|
| **种子数据不匹配** | 新增 `20230001`~`20230004` 四个 demo 账号，与历史申请/成绩数据对应 |
| **成绩页进度条除零** | 改用四列统计卡片展示各模块得分 |
| **文档过时** | CLAUDE.md、README.md 的 demo 账号更新为实际种子数据 |
| **侧边栏角色切换死按钮** | 改为角色标签，清晰显示当前身份 |
| **撤回申请** | 新增 `withdraw` action，学生在「待班委审核」阶段可撤回 |
| **公示提前结束** | `publicity-finish` 加校验：公示至少满 1 小时 |
| **列表无分页** | `ListApplications`、`ListUsers` 加 `limit`/`offset`（默认 50，最大 200） |
| **班级历史太简陋** | 从 12 行薄壳扩展为完整表格 + 搜索框 |
| **计分 Module.Max=0** | 改为当前得分 |
| **前端残留类型** | 移除 `MetaResponse.activeRule`、`RuleVersion`、`ScorePreview` 等已废弃类型 |

---

### 八、数据库迁移汇总

| 迁移 | 内容 |
|------|------|
| `002_add_claimed_score.up.sql` | `applications` 表加 `claimed_score` 列，删除 `rule_versions` 表 |
| `003_objections.up.sql` | 新建 `objections` 表 |

---

### 当前状态机总览

```
待班委审核 ──review-pass(可改分)──→ 公示中 ──publicity-finish──→ 待管理员审核
    │              │                    │        (≥1h后可手动)       │
    ├──review-reject──→ 已驳回 ←──── object(异议)                    │
    │       ↑              │                                        │
    │       │              ├──appeal──→ 申诉中 ──appeal-approve──→ 重新公示
    │       │              │            │         (恢复分数+48h)
    │       │              │            └──appeal-reject──→ 已通过/已驳回
    │       │              │
    │       │              │            ┌────────────────────────────┘
    │       │              │            │  admin-pass(可改分)
    │       │              │            │
    │       │              ↓            ↓
    └──mark-special──→ 特殊情况处理中 ──admin-pass──→ 已通过 ──appeal──→ 申诉中
                       (同待管理员审核)  ──admin-reject──→ 已驳回
```

- 学生可随时**撤回**「待班委审核」状态的申请
- 公示期间可对**他人**申请提异议
- 申诉采纳后重新进入公示
