# 综测成绩管理平台 Go 后端接口

启动方式：

```bash
go run .
```

默认地址：`http://localhost:8080`。后端同时托管当前目录下的静态前端文件，打开 `http://localhost:8080/` 即可访问原型。

## 核心接口

- `GET /api/health`：健康检查
- `GET /api/meta`：获取综测周期状态
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
- `GET /api/admin/rules`：规则版本列表
- `POST /api/admin/rules`：发布新规则版本
- `GET /api/admin/export/scores.csv`：导出成绩册 CSV
- `GET /api/admin/logs`：操作日志

## 提交申请示例

```bash
curl -X POST http://localhost:8080/api/applications \
  -H 'Content-Type: application/json' \
  -d '{
    "student":"李明",
    "studentId":"20230001",
    "className":"软件 2301",
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
  -H 'Content-Type: application/json' \
  -d '{"actor":"班委 陈欣","reason":"材料齐全"}'
```
