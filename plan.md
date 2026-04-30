Code Review
一、严重问题
安全（核心 demo 性质，但任何上线计划都必须修）
认证形同虚设 — main.go:168-174:
账号空字符串直接通过校验；密码空字符串也直接通过。if strings.TrimSpace(req.Account) != "" 这种"非空才检查"的写法是经典反模式。
即使输入正确，登录后也不下发任何 token/session — 服务端没有任何后续接口能识别"我是谁"。
所有写操作完全裸奔 — main.go:124-135:任何人都能 POST /api/applications/s2/admin-pass 把任何申请改通过、改分。actor 字段由客户端自填（app.js:1013 state.role === "admin" ? "管理员" : "班委 陈欣"），审计日志没有可信度。
CORS * + 无认证 = 任何站点的 JS 都能调你的接口（main.go:728）。
明文密码 123456 同时硬编码在前后端（main.go:172、app.js:32-39）。
业务正确性
规则版本是装饰品 — POST /api/admin/rules 把新版本标 Active=true，但实际计分函数 calculateScore（main.go:657-672）使用的是写死的 matrix，根本不读 s.rules。"发布新规则"对计分零影响。
加分矩阵复制了三份：main.go:657-663 calculateScore、main.go:592-597 seedStore rule、app.js:504-521 calculatePreviewScore。任意一份被改都会出现"预览/落库/规则面板"三个分数不一致。
状态字符串硬编码 5+ 处：main.go:17-23 常量、app.js:135-144 statusTag 配色、app.js:296 renderShell 计数、app.js:646 renderScores 过滤器、app.js:709-710 renderReviewDesk…… 改名一处必漏。
admin-pass 不能改成 0 分 — main.go:417 if req.Score > 0，无法用"扣零"处理违规申请。
公示中 申请被算入分数 — main.go:624 calculateStudentScoreLocked 把 StatusPublicity 也累加。流程要求"公示期满无异议后才入账"，与文档矛盾。
班级公示无自动到期 — PublicityUntil 设了 48h，但状态推进只能由人手动调 publicity-finish。没人点就永远卡住。
Activity.Applications 只增不减（main.go:343）— 驳回不回滚，活动卡片上的"申报人数"会越来越虚高。
AcademicScore: 86.4 写死给所有学生（main.go:615）。
架构 / 工程
main.go 770 行单文件：handlers、types、业务逻辑、内存持久化、seed 数据、CSV 导出、辅助函数全在一起。无法单元测试。
app.js 1297 行单全局 state：UI 状态（filter/sort/draft/toast）和服务端数据混在同一个 state.data 里；currentStudent 又是另一个独立 let — 两个事实来源。
没有任何测试。Go 没有 *_test.go，前端没有任何 assertion。
render() 全量替换 innerHTML，每次渲染重新 querySelectorAll 绑定事件（app.js:1052-1241）。表单焦点、滚动位置、未提交输入都会丢。
二、次要问题
持锁写网络 — main.go:377-451 handleApplicationAction 用 defer Unlock，整个 json.NewEncoder 写客户端期间都持有写锁。慢客户端可阻塞所有写请求。
mergeActivity 不能清空字段（main.go:674-703）— 用空串就忽略，导致 PATCH 无法把活动撤架到无理由状态。
renderShell 全局计数错位（app.js:295-301）— 学生看到的"待处理"是全校所有班级的待办数，不是自己的。
alert() / prompt() 当 UI — app.js:1017、app.js:1064、app.js:1211、app.js:1238 等。原型可，但产品里这是阻塞主线程的退路。
readJSON 用 DisallowUnknownFields（main.go:708）— 前后端字段稍不同步立即 400，对演进很不友好。
重复申请校验过严 — main.go:604-611 不区分综测周期，跨学年同一活动也会被判重。
CSV 导出只覆盖"曾申请过的学生"（main.go:531-547），从没申请的学生不会出现在成绩册上。
重构计划
原则：先稳住地基，再拆分，再加功能。原型阶段最痛的是"改一处漏五处"，所以前两阶段重点是消除重复与建立测试。

阶段 1：止血（1-2 天，零功能新增）
目标：消除"改一处漏五处"的硬编码重复，建立最小测试网。

任务	位置	产出
1.1 状态字符串集中	新建 internal/domain/status.go + 前端 src/constants/status.js，前端从 /api/meta 或常量文件读取	单一来源
1.2 加分矩阵唯一来源	calculateScore 改为读取 store.activeRule()；前端 calculatePreviewScore 调用新增的 POST /api/score/preview	三份归一
1.3 admin-pass 改用 *float64 指针	区分"未提供"和"显式 0 分"	可正确扣零
1.4 公示中 不计分	calculateStudentScoreLocked 只累加 StatusApproved	与流程一致
1.5 持锁写网络修正	在临界区内 copy 出快照后 Unlock，再 writeJSON	锁竞争消除
1.6 最小 Go 测试	至少 calculateScore、checkDuplicateLocked、状态机迁移（handleApplicationAction 每个 action 一例）	重构有保障
阶段 2：拆分单文件（3-5 天）
后端按职责切包：


demo1/
├── cmd/server/main.go            # 仅 wire 路由
├── internal/
│   ├── domain/                    # 类型、状态常量、业务规则
│   ├── store/                     # 现 Store，接口化为 Repository
│   ├── score/                     # calculateScore + 模块上限 + 规则版本
│   ├── http/                      # handlers，每个资源一个文件
│   └── auth/                      # 阶段 3 用
└── web/                           # 现根目录 html/css/js
前端按视图切：


web/
├── src/
│   ├── api.js                     # api()、错误归一
│   ├── state.js                   # 单一 store + subscribe
│   ├── pages/                     # 每个 render* 一个文件
│   ├── components/                # statusTag、filterRow、table 等
│   └── utils/                     # esc/format/countdown
└── index.html, styles.css
不引入 React/Vue — 维持"原生 + 零构建"的项目定位，但用 ES module 拆文件即可（<script type="module">）。

阶段 3：补真正的认证（2-3 天）
登录返回 HTTP-only Cookie 承载的随机 session token，服务端 map[token]AuthUser。
增加 requireAuth(role...) 中间件包住所有非公开 handler。
从客户端请求体里彻底删除 actor，服务端从 session 拿；审计日志才有意义。
修复"空账号/空密码即通过"的 bypass。
CORS 从 * 收紧到允许的 origin，并打开 credentials。
密码仍然可以 demo 用 123456，但走 bcrypt 比对，且不再写在前端。

阶段 4：业务漏洞收尾（2 天）
公示到期后台轮询：每分钟扫一次 StatusPublicity 且 PublicityUntil < now，自动迁移到 StatusPendingAdmin（或者前端轮询触发 publicity-finish，二选一，前者更可靠）。
Activity.Applications 改为按需 count() 计算，不再持久化计数器。
mergeActivity 改为 map[string]any PATCH，区分"未提供"和"清空"。
重复申请校验加入"周期"维度（store.summaryPhase 切换时清算）。
AcademicScore 至少做成可配置的 mock 数据接口 /api/academic/{studentId}，为接 SDU 教务系统留口子。
阶段 5（可选，看是否要落生产）
SQLite/Postgres 持久化（Repository 接口已就位）
文件上传（OSS / 本地 /uploads，materials 改成 []File）
列表分页（公示墙、申请列表）
操作日志的不可篡改（hash chain）
建议落地顺序
如果时间紧只能做一个阶段：做阶段 1。它对外行为零变化，但消除了后续所有改动的"漏改风险"，并把测试架起来 — 这是其余阶段的前提。

要我从阶段 1 的某一项开始动手吗？比较合适的起点是 1.1（状态常量集中）+ 1.6（最小测试） — 一次提交，几乎不可能出回归，但收益立竿见影。

