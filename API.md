# 综测成绩管理平台 Go 后端接口

启动方式：

```bash
docker compose up -d postgres
go run ./cmd/migrate up
go run ./cmd/seed
go run ./cmd/server
```

默认后端地址：`http://localhost:8080`。

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
- `GET /api/applications?status=&studentId=&className=`：申请列表
- `POST /api/applications`：提交申请
- `POST /api/applications/{id}/review-pass`：班委初审通过，进入 2 天班级公示
- `POST /api/applications/{id}/review-reject`：班委驳回
- `POST /api/applications/{id}/mark-special`：标记一事一议
- `POST /api/applications/{id}/publicity-finish`：班级公示结束，进入管理员复审
- `POST /api/applications/{id}/admin-pass`：管理员复审通过，可手动改分
- `POST /api/applications/{id}/admin-reject`：管理员驳回
- `GET /api/publicity`：公示墙
- `GET /api/scores/{studentId}`：学生成绩
- `POST /api/scores/preview`：按当前 active rule 预览分值
- `GET /api/admin/rules`：规则版本列表
- `POST /api/admin/rules`：发布新规则版本
- `GET /api/admin/export/scores.csv`：导出成绩册 CSV
- `GET /api/admin/logs`：操作日志

除 `GET /api/health` 和 `POST /api/auth/login` 外，接口均需要 session cookie。角色权限由服务端校验：student 只能操作本人，reviewer 只能审核本班，admin 才能管理规则、复审和导出。

## 登录示例

```bash
curl -c /tmp/sdu-cookie.txt -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"role":"student","account":"20230001","password":"123456"}'
```

可用角色：`student`（普通学生）、`reviewer`（审核学生）、`admin`（管理员）。

## 提交申请示例

```bash
curl -X POST http://localhost:8080/api/applications \
  -b /tmp/sdu-cookie.txt \
  -H 'Content-Type: application/json' \
  -d '{
    "activityId":"a2",
    "awardLevel":"二等奖",
    "projectType":"个人项目",
    "rank":1,
    "materials":["证书图片","官网公示链接"]
  }'
```

## 审核示例

```bash
curl -X POST http://localhost:8080/api/applications/s2/review-pass \
  -b /tmp/sdu-cookie.txt \
  -H 'Content-Type: application/json' \
  -d '{"reason":"材料齐全"}'
```
