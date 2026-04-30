package main

import (
	"log"
	"time"

	"demo1/internal/config"
	"demo1/internal/domain"
	"demo1/internal/repository"
	"demo1/internal/service"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func main() {
	cfg := config.Load()
	db, err := repository.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	if err := seed(db); err != nil {
		log.Fatalf("seed database: %v", err)
	}
	log.Println("seed data loaded")
}

func seed(db *gorm.DB) error {
	hash, err := service.HashPassword("123456")
	if err != nil {
		return err
	}
	now := time.Now()

	return db.Transaction(func(tx *gorm.DB) error {
		if err := upsert(tx, &domain.AssessmentCycle{
			ID:           "2026",
			Name:         "2026 综测周期",
			SummaryPhase: false,
			Active:       true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}); err != nil {
			return err
		}

		users := []domain.User{
			{ID: "user-student", Role: domain.RoleStudent, DisplayName: "李明", StudentID: "20230001", ClassName: "软件 2301", Account: "20230001", PasswordHash: hash, CreatedAt: now, UpdatedAt: now},
			{ID: "user-reviewer", Role: domain.RoleReviewer, DisplayName: "班委 陈欣", StudentID: "20230002", ClassName: "软件 2301", Account: "20230002", PasswordHash: hash, CreatedAt: now, UpdatedAt: now},
			{ID: "user-admin", Role: domain.RoleAdmin, DisplayName: "辅导员 王老师", ClassName: "信息工程学院", Account: "coach-wang", PasswordHash: hash, CreatedAt: now, UpdatedAt: now},
		}
		// 用户表只在 ID 不存在时插入，避免重新跑 seed 把用户改过的密码覆盖掉。
		for _, user := range users {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&user).Error; err != nil {
				return err
			}
		}

		activities := []domain.Activity{
			{ID: "a1", Name: "省级大学生创新创业训练计划", Organizer: "省教育厅", Level: "省级", Category: "创新素养", Range: "6-12 分", Deadline: "2026-04-28", Source: "https://example.edu/notice/innovation", Rule: "按获奖等级计入成果性评价，集体项目按位次递减。", Status: domain.ActivityStatusListed, CreatedAt: now, UpdatedAt: now},
			{ID: "a2", Name: "校级劳动实践示范项目", Organizer: "学生工作部", Level: "校级", Category: "劳动素养", Range: "3-8 分", Deadline: "2026-05-12", Source: "https://example.edu/notice/labor", Rule: "需提交实践证明和学院认定材料，模块上限 25 分。", Status: domain.ActivityStatusListed, CreatedAt: now, UpdatedAt: now},
			{ID: "a3", Name: "学院文艺展演优秀节目评选", Organizer: "信息工程学院", Level: "院级", Category: "文艺素养", Range: "1-5 分", Deadline: "2026-04-27", Source: "https://example.edu/notice/art", Rule: "个人项目按等级计分，集体项目需填写本人位次。", Status: domain.ActivityStatusListed, CreatedAt: now, UpdatedAt: now},
			{ID: "a4", Name: "校运会阳光体育竞赛", Organizer: "体育教学部", Level: "校级", Category: "身心素养", Range: "2-8 分", Deadline: "2026-06-01", Source: "https://example.edu/notice/sports", Rule: "比赛成绩需可在官方成绩册中核验。", Status: domain.ActivityStatusListed, CreatedAt: now, UpdatedAt: now},
			{ID: "a5", Name: "学院志愿服务活动补录", Organizer: "信息工程学院", Level: "院级", Category: "劳动素养", Range: "待确认", Deadline: "2026-04-18", Source: "", Rule: "补录活动，材料缺失已驳回。", Status: domain.ActivityStatusReject, CreatedAt: now, UpdatedAt: now},
		}
		for _, activity := range activities {
			if err := upsert(tx, &activity); err != nil {
				return err
			}
		}

		rule := domain.RuleVersion{
			ID:          "2026-v1",
			Name:        "2026 V1",
			Active:      true,
			PublishedAt: "2026-04-01",
			Matrix: map[string]float64{
				"国家级:一等奖": 16, "国家级:二等奖": 12, "国家级:三等奖": 8, "国家级:其他名次": 5,
				"省级:一等奖": 12, "省级:二等奖": 10, "省级:三等奖": 7, "省级:其他名次": 4,
				"校级:一等奖": 8, "校级:二等奖": 6, "校级:三等奖": 4, "校级:其他名次": 2,
				"院级:一等奖": 5, "院级:二等奖": 4, "院级:三等奖": 3, "院级:其他名次": 1,
			},
			ModuleMax: map[string]float64{"身心素养": 25, "文艺素养": 25, "劳动素养": 25, "创新素养": 25},
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := upsert(tx, &rule); err != nil {
			return err
		}

		publicityUntil := now.Add(32 * time.Hour)
		applications := []domain.Application{
			{ID: "s1", Student: "李明", StudentID: "20230001", ClassName: "软件 2301", ActivityID: "a1", Activity: "省级大学生创新创业训练计划", Category: "创新素养", AwardLevel: "一等奖", ProjectType: "个人项目", Rank: 1, Score: 12, Status: domain.StatusPublicity, SubmittedAt: "2026-04-22", PublicityUntil: &publicityUntil, ReviewerOpinion: "材料齐全，建议通过。", Materials: []string{"证书图片", "官网公示链接"}, CycleID: "2026", CreatedAt: now, UpdatedAt: now},
			{ID: "s2", Student: "陈欣", StudentID: "20230002", ClassName: "软件 2301", ActivityID: "a2", Activity: "校级劳动实践示范项目", Category: "劳动素养", AwardLevel: "二等奖", ProjectType: "个人项目", Rank: 1, Score: 6, Status: domain.StatusPendingReviewer, SubmittedAt: "2026-04-24", Materials: []string{"实践证明"}, CycleID: "2026", CreatedAt: now, UpdatedAt: now},
			{ID: "s3", Student: "王悦", StudentID: "20230003", ClassName: "软件 2301", ActivityID: "a3", Activity: "学院文艺展演优秀节目评选", Category: "文艺素养", AwardLevel: "一等奖", ProjectType: "集体项目", Rank: 2, Score: 4, Status: domain.StatusPendingAdmin, SubmittedAt: "2026-04-20", ReviewerOpinion: "已核验官网公示。", Materials: []string{"节目单", "公示截图"}, CycleID: "2026", CreatedAt: now, UpdatedAt: now},
			{ID: "s4", Student: "赵然", StudentID: "20230004", ClassName: "软件 2302", ActivityID: "a4", Activity: "校运会阳光体育竞赛", Category: "身心素养", AwardLevel: "二等奖", ProjectType: "个人项目", Rank: 1, Score: 6, Status: domain.StatusApproved, SubmittedAt: "2026-04-10", ReviewerOpinion: "通过。", AdminOpinion: "计分入账。", Materials: []string{"成绩册"}, CycleID: "2026", CreatedAt: now, UpdatedAt: now},
			{ID: "s5", Student: "李明", StudentID: "20230001", ClassName: "软件 2301", ActivityID: "a5", Activity: "学院志愿服务活动补录", Category: "劳动素养", Score: 0, Status: domain.StatusRejected, SubmittedAt: "2026-04-18", ReviewerOpinion: "缺少主办方盖章证明。", Materials: []string{"说明材料"}, CycleID: "2026", CreatedAt: now, UpdatedAt: now},
		}
		for _, app := range applications {
			if err := upsert(tx, &app); err != nil {
				return err
			}
		}

		academics := []domain.AcademicScore{
			{StudentID: "20230001", Student: "李明", ClassName: "软件 2301", Score: 86.4, CreatedAt: now, UpdatedAt: now},
			{StudentID: "20230002", Student: "陈欣", ClassName: "软件 2301", Score: 88.2, CreatedAt: now, UpdatedAt: now},
			{StudentID: "20230003", Student: "王悦", ClassName: "软件 2301", Score: 90.1, CreatedAt: now, UpdatedAt: now},
			{StudentID: "20230004", Student: "赵然", ClassName: "软件 2302", Score: 84.8, CreatedAt: now, UpdatedAt: now},
		}
		for _, academic := range academics {
			if err := upsert(tx, &academic); err != nil {
				return err
			}
		}

		return upsert(tx, &domain.AuditLog{
			ID:        "log-seed",
			Actor:     "系统",
			ActorRole: "system",
			Action:    "初始化数据",
			Target:    "演示环境",
			Note:      "加载活动、申请、规则样例",
			CreatedAt: now,
		})
	})
}

func upsert[T any](tx *gorm.DB, value *T) error {
	return tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(value).Error
}
