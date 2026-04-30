package service

import (
	"time"

	"demo1/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AdvanceExpiredPublicity 把所有公示到期的申请推进到待管理员审核，并写入审计日志。
// 仅由 server 启动时和后台 ticker 调用 —— 不要在读路径上触发。
func (s *Service) AdvanceExpiredPublicity() error {
	now := s.now()
	system := s.systemUser()
	return s.db.Transaction(func(tx *gorm.DB) error {
		var apps []domain.Application
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ? AND publicity_until IS NOT NULL AND publicity_until <= ?", domain.StatusPublicity, now).
			Find(&apps).Error; err != nil {
			return err
		}
		if len(apps) == 0 {
			return nil
		}
		ids := make([]string, 0, len(apps))
		for _, app := range apps {
			ids = append(ids, app.ID)
		}
		if err := tx.Model(&domain.Application{}).
			Where("id IN ?", ids).
			Updates(map[string]any{"status": domain.StatusPendingAdmin, "updated_at": now}).Error; err != nil {
			return err
		}
		for _, app := range apps {
			if err := s.AddLog(tx, system, "公示到期自动送审", app.Activity, "公示期满无异议，进入管理员复审"); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) ListApplications(user domain.User, status, studentID, className string) ([]domain.Application, error) {
	query := s.db.Model(&domain.Application{}).Order("created_at DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if studentID != "" {
		query = query.Where("student_id = ?", studentID)
	}
	if className != "" {
		query = query.Where("class_name = ?", className)
	}
	switch user.Role {
	case domain.RoleStudent:
		query = query.Where("student_id = ?", user.StudentID)
	case domain.RoleReviewer:
		query = query.Where("class_name = ?", user.ClassName)
	case domain.RoleAdmin:
	default:
		return nil, ErrForbidden
	}
	var apps []domain.Application
	err := query.Find(&apps).Error
	return apps, err
}

func (s *Service) Publicity(user domain.User) ([]domain.Application, error) {
	query := s.db.Where("status = ?", domain.StatusPublicity).Order("publicity_until ASC")
	if user.Role == domain.RoleStudent || user.Role == domain.RoleReviewer {
		query = query.Where("class_name = ?", user.ClassName)
	}
	var apps []domain.Application
	err := query.Find(&apps).Error
	return apps, err
}

func (s *Service) CreateApplication(user domain.User, req domain.ApplicationCreateRequest) (domain.Application, error) {
	if user.Role != domain.RoleStudent {
		return domain.Application{}, ErrForbidden
	}
	if req.Rank < 1 {
		req.Rank = 1
	}
	var app domain.Application
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var cycle domain.AssessmentCycle
		if err := tx.Where("active = ?", true).First(&cycle).Error; err != nil {
			return err
		}
		if cycle.SummaryPhase {
			return ErrSummaryPhaseLocked
		}
		var duplicate int64
		if err := tx.Model(&domain.Application{}).
			Where("student_id = ? AND activity_id = ? AND cycle_id = ? AND status <> ?", user.StudentID, req.ActivityID, cycle.ID, domain.StatusRejected).
			Count(&duplicate).Error; err != nil {
			return err
		}
		if duplicate > 0 {
			return ErrDuplicateApply
		}
		var activity domain.Activity
		if err := tx.First(&activity, "id = ?", req.ActivityID).Error; err != nil {
			return err
		}
		rule, err := s.ActiveRule(tx)
		if err != nil {
			return err
		}
		now := s.now()
		app = domain.Application{
			ID:          NewID("app"),
			Student:     user.DisplayName,
			StudentID:   user.StudentID,
			ClassName:   user.ClassName,
			ActivityID:  activity.ID,
			Activity:    activity.Name,
			Category:    activity.Category,
			AwardLevel:  req.AwardLevel,
			ProjectType: req.ProjectType,
			Rank:        req.Rank,
			Score:       CalculateScore(rule, activity.Level, req.AwardLevel, req.ProjectType, req.Rank),
			Status:      domain.StatusPendingReviewer,
			SubmittedAt: now.Format("2006-01-02"),
			Materials:   req.Materials,
			CycleID:     cycle.ID,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := tx.Create(&app).Error; err != nil {
			return err
		}
		return s.AddLog(tx, user, "提交申请", app.Activity, "进入班委初审")
	})
	return app, err
}

func (s *Service) ApplicationAction(user domain.User, id, action string, req domain.ApplicationActionRequest) (domain.Application, error) {
	var app domain.Application
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&app, "id = ?", id).Error; err != nil {
			return err
		}
		if err := s.ensureApplicationActor(user, app, action); err != nil {
			return err
		}
		now := s.now()
		switch action {
		case "review-pass":
			if app.Status != domain.StatusPendingReviewer {
				return ErrInvalidTransition
			}
			app.Status = domain.StatusPublicity
			app.ReviewerOpinion = DefaultText(req.Reason, "班委初审通过")
			publicityUntil := now.Add(48 * time.Hour)
			app.PublicityUntil = &publicityUntil
			if err := s.AddLog(tx, user, "班委初审通过", app.Activity, app.ReviewerOpinion); err != nil {
				return err
			}
		case "review-reject":
			if app.Status != domain.StatusPendingReviewer {
				return ErrInvalidTransition
			}
			reason, err := RequireReason(req.Reason, "reject reason is required")
			if err != nil {
				return err
			}
			app.Status = domain.StatusRejected
			app.ReviewerOpinion = reason
			if err := s.AddLog(tx, user, "班委驳回", app.Activity, reason); err != nil {
				return err
			}
		case "mark-special":
			if app.Status != domain.StatusPendingReviewer {
				return ErrInvalidTransition
			}
			reason, err := RequireReason(req.Reason, "special reason is required")
			if err != nil {
				return err
			}
			app.Status = domain.StatusSpecial
			app.SpecialReason = reason
			if err := s.AddLog(tx, user, "标记特殊情况", app.Activity, reason); err != nil {
				return err
			}
		case "publicity-finish":
			if app.Status != domain.StatusPublicity {
				return ErrInvalidTransition
			}
			app.Status = domain.StatusPendingAdmin
			if err := s.AddLog(tx, user, "班级公示结束", app.Activity, "进入管理员复审"); err != nil {
				return err
			}
		case "admin-pass":
			if user.Role != domain.RoleAdmin {
				return ErrForbidden
			}
			if app.Status != domain.StatusPendingAdmin && app.Status != domain.StatusSpecial {
				return ErrInvalidTransition
			}
			if req.Score != nil {
				reason, err := RequireReason(req.Reason, "manual score reason is required")
				if err != nil {
					return err
				}
				app.Score = *req.Score
				app.AdminOpinion = reason
			} else {
				app.AdminOpinion = DefaultText(req.Reason, "管理员复审通过")
			}
			app.Status = domain.StatusApproved
			if err := s.AddLog(tx, user, "管理员复审通过", app.Activity, app.AdminOpinion); err != nil {
				return err
			}
		case "admin-reject":
			if user.Role != domain.RoleAdmin {
				return ErrForbidden
			}
			if app.Status != domain.StatusPendingAdmin && app.Status != domain.StatusSpecial {
				return ErrInvalidTransition
			}
			reason, err := RequireReason(req.Reason, "reject reason is required")
			if err != nil {
				return err
			}
			app.Status = domain.StatusRejected
			app.AdminOpinion = reason
			if err := s.AddLog(tx, user, "管理员驳回", app.Activity, reason); err != nil {
				return err
			}
		default:
			return gorm.ErrRecordNotFound
		}
		app.UpdatedAt = now
		return tx.Save(&app).Error
	})
	return app, err
}

func (s *Service) ensureApplicationActor(user domain.User, app domain.Application, action string) error {
	switch action {
	case "review-pass", "review-reject", "mark-special", "publicity-finish":
		if user.Role == domain.RoleAdmin {
			return nil
		}
		if user.Role != domain.RoleReviewer {
			return ErrForbidden
		}
		if user.ClassName != app.ClassName {
			return ErrForbidden
		}
		return nil
	case "admin-pass", "admin-reject":
		if user.Role != domain.RoleAdmin {
			return ErrForbidden
		}
		return nil
	default:
		return gorm.ErrRecordNotFound
	}
}
