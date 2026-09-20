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
		&domain.Objection{},
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
	// 管理员必须配置管辖班级后才可见数据（fail-closed）
	return domain.User{ID: "u-adm", Role: domain.RoleAdmin, DisplayName: "辅导员", ManagedClasses: []string{"C1"}}
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

func TestApplicationAction_AdminCannotCrossManagedClass(t *testing.T) {
	svc := newTestService(t)
	seedFixture(t, svc)
	app := makeApp(t, svc, domain.StatusPendingAdmin)
	outOfScope := domain.User{ID: "u-adm2", Role: domain.RoleAdmin, DisplayName: "别的辅导员", ManagedClasses: []string{"C2"}}

	_, err := svc.ApplicationAction(outOfScope, app.ID, "admin-pass", domain.ApplicationActionRequest{Reason: "ok"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestApplicationAction_AppealOwnershipAndStatus(t *testing.T) {
	svc := newTestService(t)
	seedFixture(t, svc)
	app := makeApp(t, svc, domain.StatusRejected)

	// 其他学生不能替别人申诉
	other := domain.User{ID: "u-stu2", Role: domain.RoleStudent, DisplayName: "王悦", StudentID: "20230002", ClassName: "C1"}
	if _, err := svc.ApplicationAction(other, app.ID, "appeal", domain.ApplicationActionRequest{Reason: "越权申诉"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for cross-student appeal, got %v", err)
	}

	// 申请人本人从已驳回状态可以申诉
	owner := domain.User{ID: "u-stu", Role: domain.RoleStudent, DisplayName: "李明", StudentID: "20230001", ClassName: "C1"}
	got, err := svc.ApplicationAction(owner, app.ID, "appeal", domain.ApplicationActionRequest{Reason: "材料其实齐全"})
	if err != nil {
		t.Fatalf("own appeal: %v", err)
	}
	if got.Status != domain.StatusAppealing {
		t.Fatalf("status=%s, want %s", got.Status, domain.StatusAppealing)
	}

	// 待班委审核状态不允许申诉
	app2 := makeApp(t, svc, domain.StatusPendingReviewer)
	if _, err := svc.ApplicationAction(owner, app2.ID, "appeal", domain.ApplicationActionRequest{Reason: "太早了"}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestApplicationAction_ObjectSameClassOnly(t *testing.T) {
	svc := newTestService(t)
	seedFixture(t, svc)
	past := time.Now().Add(-time.Hour)
	app := domain.Application{
		ID: "app-obj", Student: "李明", StudentID: "20230001", ClassName: "C1",
		ActivityID: "act1", Activity: "省级创新", Category: "创新素养",
		AwardLevel: "一等奖", ProjectType: "个人项目", Rank: 1, Score: 12, ClaimedScore: 12,
		Status: domain.StatusPublicity, SubmittedAt: "2026-04-22", CycleID: "c1",
		PublicityUntil: &past, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := svc.db.Create(&app).Error; err != nil {
		t.Fatalf("seed publicity app: %v", err)
	}

	// 他班学生不能提异议
	other := domain.User{ID: "u-stu2", Role: domain.RoleStudent, DisplayName: "王悦", StudentID: "20230002", ClassName: "C2"}
	if _, err := svc.ApplicationAction(other, app.ID, "object", domain.ApplicationActionRequest{Reason: "他班越权"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for cross-class object, got %v", err)
	}

	// 同班学生可以提异议，且不改变状态
	same := domain.User{ID: "u-stu3", Role: domain.RoleStudent, DisplayName: "赵然", StudentID: "20230003", ClassName: "C1"}
	got, err := svc.ApplicationAction(same, app.ID, "object", domain.ApplicationActionRequest{Reason: "分数存疑"})
	if err != nil {
		t.Fatalf("same-class object: %v", err)
	}
	if got.Status != domain.StatusPublicity {
		t.Fatalf("status=%s, want %s", got.Status, domain.StatusPublicity)
	}
}

func TestListApplications_IsolationMatrix(t *testing.T) {
	svc := newTestService(t)
	seedFixture(t, svc)
	makeApp(t, svc, domain.StatusPendingReviewer) // app-待班委审核：李明 C1
	now := time.Now()
	otherClassApp := domain.Application{
		ID: "app-c2", Student: "王悦", StudentID: "20230002", ClassName: "C2",
		ActivityID: "act1", Activity: "省级创新", Category: "创新素养",
		Status: domain.StatusPendingReviewer, SubmittedAt: "2026-04-23", CycleID: "c1",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := svc.db.Create(&otherClassApp).Error; err != nil {
		t.Fatalf("seed other class app: %v", err)
	}
	sameClassOtherStudent := domain.Application{
		ID: "app-c1b", Student: "赵然", StudentID: "20230003", ClassName: "C1",
		ActivityID: "act1", Activity: "省级创新", Category: "创新素养",
		Status: domain.StatusPendingReviewer, SubmittedAt: "2026-04-23", CycleID: "c1",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := svc.db.Create(&sameClassOtherStudent).Error; err != nil {
		t.Fatalf("seed same class other student app: %v", err)
	}

	// 学生只看自己的申请（同班其他人的也看不到）
	student := domain.User{ID: "u-stu", Role: domain.RoleStudent, DisplayName: "李明", StudentID: "20230001", ClassName: "C1"}
	apps, err := svc.ListApplications(student, "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("student list: %v", err)
	}
	if len(apps) != 1 || apps[0].StudentID != "20230001" {
		t.Fatalf("student should only see own apps, got %d", len(apps))
	}

	// 班委只看本班申请
	apps, err = svc.ListApplications(reviewer(), "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("reviewer list: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("reviewer should see 2 same-class apps, got %d", len(apps))
	}
	for _, app := range apps {
		if app.ClassName != "C1" {
			t.Fatalf("reviewer saw cross-class app %s", app.ID)
		}
	}

	// 管理员只看管辖班级申请
	apps, err = svc.ListApplications(admin(), "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("admin list: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("admin should see 2 managed-class apps, got %d", len(apps))
	}

	// 未配置管辖班级的管理员看不到任何申请
	blind := domain.User{ID: "u-adm3", Role: domain.RoleAdmin, DisplayName: "无辖区辅导员"}
	apps, err = svc.ListApplications(blind, "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("blind admin list: %v", err)
	}
	if len(apps) != 0 {
		t.Fatalf("admin without managed classes should see nothing, got %d", len(apps))
	}
}

func TestStudentScoreFor_Scoping(t *testing.T) {
	svc := newTestService(t)
	seedFixture(t, svc)
	makeApp(t, svc, domain.StatusApproved)
	if err := svc.db.Create(&domain.User{ID: "u-stu", Role: domain.RoleStudent, DisplayName: "李明", StudentID: "20230001", ClassName: "C1"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	otherClassStudent := domain.User{ID: "u-stu2", Role: domain.RoleStudent, DisplayName: "王悦", StudentID: "20230002", ClassName: "C2"}
	if err := svc.db.Create(&otherClassStudent).Error; err != nil {
		t.Fatalf("seed other user: %v", err)
	}

	// 学生查本人 OK
	if _, err := svc.StudentScoreFor(domain.User{ID: "u-stu", Role: domain.RoleStudent, StudentID: "20230001", ClassName: "C1"}, "20230001"); err != nil {
		t.Fatalf("own score should pass: %v", err)
	}
	// 学生查他人 403
	if _, err := svc.StudentScoreFor(domain.User{ID: "u-stu", Role: domain.RoleStudent, StudentID: "20230001", ClassName: "C1"}, "20230002"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	// 班委查本班学生 OK
	if _, err := svc.StudentScoreFor(reviewer(), "20230001"); err != nil {
		t.Fatalf("reviewer same class score should pass: %v", err)
	}
	// 班委查他班学生 403
	if _, err := svc.StudentScoreFor(reviewer(), "20230002"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	// 管理员查管辖班级学生 OK
	if _, err := svc.StudentScoreFor(admin(), "20230001"); err != nil {
		t.Fatalf("admin managed class score should pass: %v", err)
	}
	// 管理员查辖区外学生 403
	outOfScope := domain.User{ID: "u-adm2", Role: domain.RoleAdmin, ManagedClasses: []string{"C2"}}
	if _, err := svc.StudentScoreFor(outOfScope, "20230001"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestListObjections_Scoping(t *testing.T) {
	svc := newTestService(t)
	seedFixture(t, svc)
	app := makeApp(t, svc, domain.StatusPublicity)
	if err := svc.db.Create(&domain.Objection{
		ID: "obj-1", ApplicationID: app.ID, Student: "赵然", StudentID: "20230003", Reason: "分数存疑",
	}).Error; err != nil {
		t.Fatalf("seed objection: %v", err)
	}

	// 同班班委可见
	list, err := svc.ListObjections(reviewer(), app.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("reviewer should see objections: %v %d", err, len(list))
	}
	// 他班班委 403
	other := domain.User{ID: "u-rev2", Role: domain.RoleReviewer, DisplayName: "他班班委", ClassName: "C2"}
	if _, err := svc.ListObjections(other, app.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	// 管辖班级外管理员 403
	outOfScope := domain.User{ID: "u-adm2", Role: domain.RoleAdmin, ManagedClasses: []string{"C2"}}
	if _, err := svc.ListObjections(outOfScope, app.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpdateUserClasses_SuperOnly(t *testing.T) {
	svc := newTestService(t)
	if err := svc.db.Create(&domain.User{ID: "u-adm", Role: domain.RoleAdmin, DisplayName: "辅导员"}).Error; err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	if err := svc.db.Create(&domain.User{ID: "u-super", Role: domain.RoleAdmin, DisplayName: "系统管理员", IsSuperAdmin: true}).Error; err != nil {
		t.Fatalf("seed super admin: %v", err)
	}

	// 辅导员不能自改管辖班级
	counselor := domain.User{ID: "u-adm", Role: domain.RoleAdmin, DisplayName: "辅导员"}
	if _, err := svc.UpdateUserClasses(counselor, "u-adm", []string{"C1"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("counselor self-edit should be forbidden, got %v", err)
	}

	// 系统管理员可以指定任意辅导员的管辖班级（去重去空）
	super := domain.User{ID: "u-super", Role: domain.RoleAdmin, DisplayName: "系统管理员", IsSuperAdmin: true}
	updated, err := svc.UpdateUserClasses(super, "u-adm", []string{"公信23", " 公信24 ", "公信23", ""})
	if err != nil {
		t.Fatalf("super set classes: %v", err)
	}
	if len(updated.ManagedClasses) != 2 || updated.ManagedClasses[0] != "公信23" {
		t.Fatalf("managedClasses=%v, want [公信23 公信24]", updated.ManagedClasses)
	}

	// 不能对非辅导员账号设置管辖班级
	if _, err := svc.UpdateUserClasses(super, "u-super", []string{"C1"}); err == nil {
		t.Fatalf("super admin target should be rejected")
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
