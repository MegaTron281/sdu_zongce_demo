# 综测成绩管理平台 Go 后端接口

启动方式：

```bash
docker compose up -d postgres
go run ./cmd/migrate up
go run ./cmd/seedclasses   # 测试账号矩阵（幂等，不覆盖密码）
go run ./cmd/server
```

后端地址取自 `.env` 的 `SERVER_ADDR`（本机开发为 `http://localhost:8090`，未配置时默认 `:8080`）。

前端开发：

```bash
cd frontend
npm install
npm run dev
```

Vue 开发地址：`http://localhost:5173`，Vite 会把 `/api` 代理到 Gin 后端。生产模式下先执行 `npm run build`，Gin 会托管 `frontend/dist`。

## 核心接口

- `GET /api/health`：健康检查
- `POST /api/auth/login`：账号密码登录，写入 HTTP-only session cookie
- `GET /api/auth/me`：获取当前登录用户
- `POST /api/auth/logout`：退出登录并撤销 session
- `GET /api/meta`：获取综测周期状态、状态枚举和当前规则摘要
- `PATCH /api/meta`：切换周期，例如 `{"summaryPhase":true}`
- `GET /api/activities?q=&category=&level=`：活动名录列表
- `POST /api/activities`：学生申请补录活动
- `GET /api/activities/{id}`：活动详情
- `PATCH /api/activities/{id}`：管理员维护活动
- `GET /api/applications?status=&studentId=&className=`：申请列表。**数据隔离**：student 只能看本人申请；reviewer 只能看本班；admin 只能看 `managed_classes` 管辖班级（未配置则返回空）
- `POST /api/applications`：提交申请。学生可直接填写活动信息申报；如果传 `activityId`，则沿用已有活动名录信息
- `POST /api/applications/{id}/review-pass`：班委初审通过，进入 2 天班级公示（仅限本班班委或管辖班级管理员）
- `POST /api/applications/{id}/review-reject`：班委驳回（须填原因）
- `POST /api/applications/{id}/mark-special`：标记一事一议/特殊情况（须填原因），跳过公示由管理员直接处理
- `POST /api/applications/{id}/publicity-finish`：班级公示结束（需已公示满 1 小时），进入管理员复审
- `POST /api/applications/{id}/admin-pass`：管理员复审通过，可手动改分（仅管辖班级；`特殊情况处理中` 也可处理）
- `POST /api/applications/{id}/admin-reject`：管理员驳回（须填原因）
- `POST /api/applications/{id}/object`：同班同学对公示中申请提异议（不改变状态）
- `POST /api/applications/{id}/appeal`：申请人本人申诉（仅限已驳回，或分数被改动的公示中/已通过）
- `POST /api/applications/{id}/appeal-approve` / `appeal-reject`：管理员处理申诉
- `GET /api/applications/{id}/objections`：申请的异议列表（student/reviewer 限本班，admin 限管辖班级）
- `POST /api/materials/upload`：学生上传证明材料，支持 PDF、PNG、JPG，单文件最大 10MB，返回可放入 `materials` 的文件地址
- `GET /api/materials/{name}`：已登录用户查看上传材料
- `GET /api/publicity`：公示墙（student/reviewer 本班，admin 管辖班级）
- `GET /api/scores/{studentId}`：学生成绩。**越权校验**：student 仅本人；reviewer 仅本班学生；admin 仅管辖班级学生
- `GET /api/admin/users`：用户列表。系统管理员可见全部；辅导员只见管辖班级学生/班委 + 自己
- `POST /api/admin/users/import`：CSV 导入学生（学号,姓名,班级）
- `PATCH /api/admin/users/{id}/role`：辅导员可把管辖班级内账号设为/取消班委；**任命/撤销辅导员（admin）仅系统管理员可操作**；系统管理员自身角色不可修改
- `PATCH /api/admin/users/{id}/classes`：**仅系统管理员可操作**，指定任意辅导员的管辖班级，如 `{"classes":["公信23","公信24"]}`；辅导员无权自改
- `GET /api/admin/export/scores.csv`：导出成绩册 CSV（系统管理员全校，辅导员仅管辖班级）
- `GET /api/admin/logs`：操作日志

除 `GET /api/health` 和 `POST /api/auth/login` 外，接口均需要 session cookie。角色权限由服务端校验：student 只能操作本人，reviewer 只能审核本班，admin 只能操作管辖班级；`admin` 角色中带 `isSuperAdmin` 标志的账号为**系统管理员**，不受管辖班级限制（可见/可操作全校数据），且是唯一能分配管辖班级、任命辅导员的身份。

活动截止时间（`deadline`）仅表示活动本身的举办/报名截止，**不限制综测加分申报**——学生随时可对已上架活动提交综测申报，受控的只有汇总期锁（`summaryPhase`）。

## 登录示例

```bash
curl -c /tmp/sdu-cookie.txt -X POST http://localhost:8090/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"role":"student","account":"202300271018","password":"123456"}'
```

可用角色：`student`（普通学生）、`reviewer`（审核学生）、`admin`（辅导员；含唯一 `isSuperAdmin` 系统管理员账号）。

## 提交申请示例

```bash
curl -X POST http://localhost:8090/api/applications \
  -b /tmp/sdu-cookie.txt \
  -H 'Content-Type: application/json' \
  -d '{
    "activityName":"校运会阳光体育竞赛",
    "organizer":"体育教学部",
    "level":"校级",
    "category":"身心素养",
    "deadline":"2026-06-01",
    "source":"https://example.edu/notice/sports",
    "rule":"比赛成绩需可在官方成绩册中核验。",
    "awardLevel":"二等奖",
    "projectType":"个人项目",
    "rank":1,
    "materials":["证书图片","官网公示链接"]
  }'
```

## 审核示例

```bash
curl -X POST http://localhost:8090/api/applications/s2/review-pass \
  -b /tmp/sdu-cookie.txt \
  -H 'Content-Type: application/json' \
  -d '{"reason":"材料齐全"}'
```
