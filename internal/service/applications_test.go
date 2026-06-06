package service

import (
	"errors"
	"testing"
	"time"

	"demo1/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_pragma=foreign_keys(1)"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Session{},
		&domain.AssessmentCycle{},
		&domain.Activity{},
		&domain.Application{},
		&domain.AcademicScore{},
		&domain.AuditLog{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return New(db, "test-secret")
}

func seedFixture(t *testing.T, svc *Service) {
	t.Helper()
	now := time.Now()
	if err := svc.db.Create(&domain.AssessmentCycle{ID: "c1", Name: "test", Active: true, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed cycle: %v", err)
	}
	if err := svc.db.Create(&domain.Activity{
		ID: "act1", Name: "省级创新", Level: "省级", Category: "创新素养", Status: domain.ActivityStatusListed,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("seed activity: %v", err)
	}
}

func reviewer() domain.User {
	return domain.User{ID: "u-rev", Role: domain.RoleReviewer, DisplayName: "班委", ClassName: "C1"}
}

func admin() domain.User {
	return domain.User{ID: "u-adm", Role: domain.RoleAdmin, DisplayName: "辅导员"}
}

func makeApp(t *testing.T, svc *Service, status string) domain.Application {
	t.Helper()
	now := time.Now()
	app := domain.Application{
		ID: "app-" + status, Student: "李明", StudentID: "20230001", ClassName: "C1",
		ActivityID: "act1", Activity: "省级创新", Category: "创新素养",
		AwardLevel: "一等奖", ProjectType: "个人项目", Rank: 1, Score: 12,
		Status: status, SubmittedAt: "2026-04-22", CycleID: "c1",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := svc.db.Create(&app).Error; err != nil {
		t.Fatalf("create app %s: %v", status, err)
	}
	return app
}

func TestApplicationAction_ReviewPass_MovesToPublicity(t *testing.T) {
	svc := newTestService(t)
	seedFixture(t, svc)
	app := makeApp(t, svc, domain.StatusPendingReviewer)

	got, err := svc.ApplicationAction(reviewer(), app.ID, "review-pass", domain.ApplicationActionRequest{Reason: "ok"})
	if err != nil {
		t.Fatalf("review-pass: %v", err)
	}
	if got.Status != domain.StatusPublicity {
		t.Fatalf("status=%s, want %s", got.Status, domain.StatusPublicity)
	}
	if got.PublicityUntil == nil || got.PublicityUntil.Before(time.Now()) {
		t.Fatalf("publicityUntil should be set in the future, got %v", got.PublicityUntil)
	}
}

func TestApplicationAction_ReviewReject_RequiresReason(t *testing.T) {
	svc := newTestService(t)
	seedFixture(t, svc)
	app := makeApp(t, svc, domain.StatusPendingReviewer)

	_, err := svc.ApplicationAction(reviewer(), app.ID, "review-reject", domain.ApplicationActionRequest{Reason: ""})
	if !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("expected ErrReasonRequired, got %v", err)
	}

	got, err := svc.ApplicationAction(reviewer(), app.ID, "review-reject", domain.ApplicationActionRequest{Reason: "材料不全"})
	if err != nil {
		t.Fatalf("review-reject with reason: %v", err)
	}
	if got.Status != domain.StatusRejected {
		t.Fatalf("status=%s, want %s", got.Status, domain.StatusRejected)
	}
}

func TestApplicationAction_AdminPass_AcceptsZeroScore(t *testing.T) {
	svc := newTestService(t)
	seedFixture(t, svc)
	app := makeApp(t, svc, domain.StatusPendingAdmin)

	zero := 0.0
	got, err := svc.ApplicationAction(admin(), app.ID, "admin-pass", domain.ApplicationActionRequest{Score: &zero, Reason: "违规扣零"})
	if err != nil {
		t.Fatalf("admin-pass with zero score: %v", err)
	}
	if got.Status != domain.StatusApproved {
		t.Fatalf("status=%s, want %s", got.Status, domain.StatusApproved)
	}
	if got.Score != 0 {
		t.Fatalf("score=%v, want 0", got.Score)
	}
}

func TestApplicationAction_RejectsInvalidTransition(t *testing.T) {
	svc := newTestService(t)
	seedFixture(t, svc)
	app := makeApp(t, svc, domain.StatusApproved)

	_, err := svc.ApplicationAction(reviewer(), app.ID, "review-pass", domain.ApplicationActionRequest{})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestApplicationAction_ReviewerCannotCrossClass(t *testing.T) {
	svc := newTestService(t)
	seedFixture(t, svc)
	app := makeApp(t, svc, domain.StatusPendingReviewer)
	otherClass := domain.User{ID: "u-rev2", Role: domain.RoleReviewer, DisplayName: "他班班委", ClassName: "C2"}

	_, err := svc.ApplicationAction(otherClass, app.ID, "review-pass", domain.ApplicationActionRequest{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateApplication_BlocksDuplicateInSameCycle(t *testing.T) {
	svc := newTestService(t)
	seedFixture(t, svc)
	makeApp(t, svc, domain.StatusPendingReviewer)
	student := domain.User{ID: "u-stu", Role: domain.RoleStudent, DisplayName: "李明", StudentID: "20230001", ClassName: "C1"}

	_, err := svc.CreateApplication(student, domain.ApplicationCreateRequest{
		ActivityID: "act1", AwardLevel: "一等奖", ProjectType: "个人项目", Rank: 1, Score: 12,
	})
	if !errors.Is(err, ErrDuplicateApply) {
		t.Fatalf("expected ErrDuplicateApply, got %v", err)
	}
}

func TestCreateApplication_AllowsStudentFilledActivity(t *testing.T) {
	svc := newTestService(t)
	seedFixture(t, svc)
	student := domain.User{ID: "u-stu", Role: domain.RoleStudent, DisplayName: "李明", StudentID: "20230001", ClassName: "C1"}

	app, err := svc.CreateApplication(student, domain.ApplicationCreateRequest{
		ActivityName: "学生自填活动",
		Organizer:    "学院团委",
		Level:        "校级",
		Category:     "劳动素养",
		AwardLevel:   "二等奖",
		ProjectType:  "个人项目",
		Rank:         1,
		Score:        6,
		Materials:    []string{"证明材料"},
	})
	if err != nil {
		t.Fatalf("create application with filled activity: %v", err)
	}
	if app.Activity != "学生自填活动" {
		t.Fatalf("activity=%q, want 学生自填活动", app.Activity)
	}
	if app.ActivityID == "" {
		t.Fatal("expected generated activity id")
	}
	if app.ClaimedScore != 6 {
		t.Fatalf("claimedScore=%v, want 6", app.ClaimedScore)
	}

	var activity domain.Activity
	if err := svc.db.First(&activity, "id = ?", app.ActivityID).Error; err != nil {
		t.Fatalf("created activity not found: %v", err)
	}
	if activity.Status != domain.ActivityStatusPending {
		t.Fatalf("activity status=%s, want %s", activity.Status, domain.ActivityStatusPending)
	}
}

func TestCreateApplication_BlockedBySummaryPhase(t *testing.T) {
	svc := newTestService(t)
	seedFixture(t, svc)
	if err := svc.db.Model(&domain.AssessmentCycle{}).Where("id = ?", "c1").Update("summary_phase", true).Error; err != nil {
		t.Fatalf("flip phase: %v", err)
	}
	student := domain.User{ID: "u-stu", Role: domain.RoleStudent, DisplayName: "李明", StudentID: "20230001", ClassName: "C1"}

	_, err := svc.CreateApplication(student, domain.ApplicationCreateRequest{
		ActivityID: "act1", AwardLevel: "一等奖", ProjectType: "个人项目", Rank: 1, Score: 12,
	})
	if !errors.Is(err, ErrSummaryPhaseLocked) {
		t.Fatalf("expected ErrSummaryPhaseLocked, got %v", err)
	}
}

func TestAdvanceExpiredPublicity_PromotesAndLogs(t *testing.T) {
	svc := newTestService(t)
	seedFixture(t, svc)
	past := time.Now().Add(-time.Hour)
	app := domain.Application{
		ID: "app-pub", Student: "李明", StudentID: "20230001", ClassName: "C1",
		ActivityID: "act1", Activity: "省级创新", Category: "创新素养",
		AwardLevel: "一等奖", ProjectType: "个人项目", Rank: 1, Score: 12,
		Status: domain.StatusPublicity, SubmittedAt: "2026-04-22", CycleID: "c1",
		PublicityUntil: &past,
		CreatedAt:      time.Now(), UpdatedAt: time.Now(),
	}
	if err := svc.db.Create(&app).Error; err != nil {
		t.Fatalf("seed publicity app: %v", err)
	}

	if err := svc.AdvanceExpiredPublicity(); err != nil {
		t.Fatalf("advance: %v", err)
	}

	var got domain.Application
	if err := svc.db.First(&got, "id = ?", "app-pub").Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Status != domain.StatusPendingAdmin {
		t.Fatalf("status=%s, want %s", got.Status, domain.StatusPendingAdmin)
	}

	var logCount int64
	if err := svc.db.Model(&domain.AuditLog{}).Where("action = ?", "公示到期自动送审").Count(&logCount).Error; err != nil {
		t.Fatalf("count logs: %v", err)
	}
	if logCount != 1 {
		t.Fatalf("expected 1 audit log, got %d", logCount)
	}
}
