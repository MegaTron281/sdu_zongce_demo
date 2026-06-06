package service

import (
	"errors"
	"fmt"
	"strings"
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

func (s *Service) ListApplications(user domain.User, status, studentID, className string, limit, offset int) ([]domain.Application, error) {
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
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
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
		var activity domain.Activity
		activityID := strings.TrimSpace(req.ActivityID)
		if activityID != "" {
			var duplicate int64
			if err := tx.Model(&domain.Application{}).
				Where("student_id = ? AND activity_id = ? AND cycle_id = ? AND status <> ?", user.StudentID, activityID, cycle.ID, domain.StatusRejected).
				Count(&duplicate).Error; err != nil {
				return err
			}
			if duplicate > 0 {
				return ErrDuplicateApply
			}
			if err := tx.First(&activity, "id = ?", activityID).Error; err != nil {
				return err
			}
		} else {
			var err error
			activity, err = s.createActivityFromApplicationLocked(tx, user, req, cycle.ID)
			if err != nil {
				return err
			}
		}
		now := s.now()
		app = domain.Application{
			ID:           NewID("app"),
			Student:      user.DisplayName,
			StudentID:    user.StudentID,
			ClassName:    user.ClassName,
			ActivityID:   activity.ID,
			Activity:     activity.Name,
			Category:     activity.Category,
			AwardLevel:   req.AwardLevel,
			ProjectType:  req.ProjectType,
			Rank:         req.Rank,
			Score:        req.Score,
			ClaimedScore: req.Score,
			Status:       domain.StatusPendingReviewer,
			SubmittedAt:  now.Format("2006-01-02"),
			Materials:    req.Materials,
			CycleID:      cycle.ID,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.Create(&app).Error; err != nil {
			return err
		}
		return s.AddLog(tx, user, "提交申请", app.Activity, "进入班委初审")
	})
	return app, err
}

func (s *Service) createActivityFromApplicationLocked(tx *gorm.DB, user domain.User, req domain.ApplicationCreateRequest, cycleID string) (domain.Activity, error) {
	name := strings.TrimSpace(req.ActivityName)
	level := strings.TrimSpace(req.Level)
	category := strings.TrimSpace(req.Category)
	if name == "" || level == "" || category == "" {
		return domain.Activity{}, errors.New("activity name, level and category are required")
	}

	var duplicate int64
	if err := tx.Model(&domain.Application{}).
		Where("student_id = ? AND cycle_id = ? AND activity = ? AND status <> ?", user.StudentID, cycleID, name, domain.StatusRejected).
		Count(&duplicate).Error; err != nil {
		return domain.Activity{}, err
	}
	if duplicate > 0 {
		return domain.Activity{}, ErrDuplicateApply
	}

	now := s.now()
	activity := domain.Activity{
		ID:        NewID("act"),
		Name:      name,
		Organizer: strings.TrimSpace(req.Organizer),
		Level:     level,
		Category:  category,
		Range:     DefaultText(req.Range, "待管理员确认"),
		Deadline:  strings.TrimSpace(req.Deadline),
		Source:    strings.TrimSpace(req.Source),
		Rule:      DefaultText(req.Rule, "学生自填活动，待管理员复核。"),
		Status:    domain.ActivityStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tx.Create(&activity).Error; err != nil {
		return domain.Activity{}, err
	}
	if err := s.AddLog(tx, user, "填报活动信息", activity.Name, "随申请进入待审核活动"); err != nil {
		return domain.Activity{}, err
	}
	return activity, nil
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
			if req.Score != nil {
				reason, err := RequireReason(req.Reason, "修改分数需要填写原因")
				if err != nil {
					return err
				}
				app.Score = *req.Score
				app.ReviewerOpinion = reason
			} else {
				app.ReviewerOpinion = DefaultText(req.Reason, "班委初审通过")
			}
			app.Status = domain.StatusPublicity
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
		case "withdraw":
			if user.Role != domain.RoleStudent || user.StudentID != app.StudentID {
				return ErrForbidden
			}
			if app.Status != domain.StatusPendingReviewer {
				return ErrInvalidTransition
			}
			app.Status = domain.StatusRejected
			app.ReviewerOpinion = "学生自行撤回"
			if err := s.AddLog(tx, user, "撤回申请", app.Activity, "学生自行撤回"); err != nil {
				return err
			}
		case "publicity-finish":
			if app.Status != domain.StatusPublicity {
				return ErrInvalidTransition
			}
			// 公示至少已过 1 小时才允许手动结束
			if app.PublicityUntil != nil && app.PublicityUntil.After(now.Add(47*time.Hour)) {
				return fmt.Errorf("公示至少需满 1 小时才能手动结束")
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
		case "object":
			if user.Role != domain.RoleStudent {
				return ErrForbidden
			}
			if user.StudentID == app.StudentID {
				return fmt.Errorf("不能对自己的申请提出异议")
			}
			if app.Status != domain.StatusPublicity {
				return ErrInvalidTransition
			}
			reason, err := RequireReason(req.Reason, "objection reason is required")
			if err != nil {
				return err
			}
			obj := domain.Objection{
				ID:            NewID("obj"),
				ApplicationID: app.ID,
				Student:       user.DisplayName,
				StudentID:     user.StudentID,
				Reason:        reason,
				CreatedAt:     now,
			}
			if err := tx.Create(&obj).Error; err != nil {
				return err
			}
			if err := s.AddLog(tx, user, "提出异议", app.Activity, reason); err != nil {
				return err
			}
			// 异议不改变申请状态，直接返回当前状态
			app.UpdatedAt = now
			return tx.Save(&app).Error
		case "appeal":
			reason, err := RequireReason(req.Reason, "appeal reason is required")
			if err != nil {
				return err
			}
			app.Status = domain.StatusAppealing
			app.SpecialReason = reason
			if err := s.AddLog(tx, user, "学生申诉", app.Activity, reason); err != nil {
				return err
			}
		case "appeal-approve":
			if user.Role != domain.RoleAdmin {
				return ErrForbidden
			}
			if app.Status != domain.StatusAppealing {
				return ErrInvalidTransition
			}
			// 恢复原始申报分
			if app.Score != app.ClaimedScore {
				app.Score = app.ClaimedScore
			}
			// 重新进入公示期，让其他同学有机会提异议
			publicityUntil := now.Add(48 * time.Hour)
			app.PublicityUntil = &publicityUntil
			app.Status = domain.StatusPublicity
			app.AdminOpinion = DefaultText(req.Reason, "管理员采纳申诉，重新进入公示")
			if err := s.AddLog(tx, user, "采纳申诉", app.Activity, "恢复原始分数并重新公示"); err != nil {
				return err
			}
		case "appeal-reject":
			if user.Role != domain.RoleAdmin {
				return ErrForbidden
			}
			if app.Status != domain.StatusAppealing {
				return ErrInvalidTransition
			}
			reason, err := RequireReason(req.Reason, "reject reason is required")
			if err != nil {
				return err
			}
			// 分数未被改 → 申诉针对的是驳回 → 回到已驳回
			// 分数被改过 → 申诉针对的是改分 → 回到已通过（维持改后分数）
			if app.ClaimedScore == app.Score {
				app.Status = domain.StatusRejected
			} else {
				app.Status = domain.StatusApproved
			}
			app.AdminOpinion = reason
			if err := s.AddLog(tx, user, "驳回申诉", app.Activity, reason); err != nil {
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

func (s *Service) ListObjections(applicationID string) ([]domain.Objection, error) {
	var list []domain.Objection
	err := s.db.Where("application_id = ?", applicationID).Order("created_at DESC").Find(&list).Error
	return list, err
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
	case "object", "withdraw":
		return nil // 权限在 ApplicationAction 中判断
	case "appeal":
		return nil // 权限在 ApplicationAction 中自行判断
	case "admin-pass", "admin-reject", "appeal-approve", "appeal-reject":
		if user.Role != domain.RoleAdmin {
			return ErrForbidden
		}
		return nil
	default:
		return gorm.ErrRecordNotFound
	}
}
