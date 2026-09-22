package service

import (
	"fmt"
	"strings"

	"demo1/internal/domain"

	"gorm.io/gorm"
)

func (s *Service) Meta() (domain.MetaResponse, error) {
	var cycle domain.AssessmentCycle
	if err := s.db.Where("active = ?", true).First(&cycle).Error; err != nil && !IsNotFound(err) {
		return domain.MetaResponse{}, err
	}
	return domain.MetaResponse{
		SummaryPhase:        cycle.SummaryPhase,
		ApplicationStatuses: domain.ApplicationStatuses,
		ActivityStatuses:    domain.ActivityStatuses,
		Modules:             domain.ModuleNames,
	}, nil
}

func (s *Service) SetSummaryPhase(user domain.User, value bool) (domain.MetaResponse, error) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var cycle domain.AssessmentCycle
		if err := tx.Where("active = ?", true).First(&cycle).Error; err != nil {
			return err
		}
		cycle.SummaryPhase = value
		cycle.UpdatedAt = s.now()
		if err := tx.Save(&cycle).Error; err != nil {
			return err
		}
		return s.AddLog(tx, user, "切换综测周期", "系统", fmt.Sprintf("summaryPhase=%t", value))
	})
	if err != nil {
		return domain.MetaResponse{}, err
	}
	return s.Meta()
}

func (s *Service) ListActivities(keyword, category, level string) ([]domain.ActivityWithCount, error) {
	query := s.db.Model(&domain.Activity{}).Order("created_at ASC")
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name ILIKE ? OR organizer ILIKE ? OR category ILIKE ?", like, like, like)
	}
	if category != "" && category != "全部" {
		query = query.Where("category = ?", category)
	}
	if level != "" && level != "全部" {
		query = query.Where("level = ?", level)
	}
	var activities []domain.Activity
	if err := query.Find(&activities).Error; err != nil {
		return nil, err
	}
	items := make([]domain.ActivityWithCount, 0, len(activities))
	for _, activity := range activities {
		var count int64
		if err := s.db.Model(&domain.Application{}).
			Where("activity_id = ? AND status <> ?", activity.ID, domain.StatusRejected).
			Count(&count).Error; err != nil {
			return nil, err
		}
		items = append(items, domain.ActivityWithCount{Activity: activity, Applications: count})
	}
	return items, nil
}

func (s *Service) CreateActivity(user domain.User, activity domain.Activity) (domain.ActivityWithCount, error) {
	if user.Role != domain.RoleStudent && user.Role != domain.RoleAdmin {
		return domain.ActivityWithCount{}, ErrForbidden
	}
	name := strings.TrimSpace(activity.Name)
	if name == "" {
		return domain.ActivityWithCount{}, fmt.Errorf("活动名称不能为空")
	}
	// 名录里不允许出现同名活动：重复条目对学生和审核都没有意义
	var sameName int64
	if err := s.db.Model(&domain.Activity{}).Where("name = ?", name).Count(&sameName).Error; err != nil {
		return domain.ActivityWithCount{}, err
	}
	if sameName > 0 {
		return domain.ActivityWithCount{}, fmt.Errorf("活动名录中已存在同名活动「%s」，请勿重复创建", name)
	}
	activity.Name = name

	now := s.now()
	if activity.ID == "" {
		activity.ID = NewID("act")
	}
	if strings.TrimSpace(activity.Status) == "" {
		activity.Status = domain.ActivityStatusPending
	}
	activity.CreatedAt = now
	activity.UpdatedAt = now
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&activity).Error; err != nil {
			return err
		}
		action := "申请补录活动"
		note := "进入待审核上架"
		if user.Role == domain.RoleAdmin {
			action = "创建活动名录"
			note = "管理员创建"
		}
		return s.AddLog(tx, user, action, activity.Name, note)
	})
	if err != nil {
		return domain.ActivityWithCount{}, err
	}
	return domain.ActivityWithCount{Activity: activity, Applications: 0}, nil
}

func (s *Service) PatchActivity(user domain.User, id string, patch domain.ActivityPatch) (domain.ActivityWithCount, error) {
	if user.Role != domain.RoleAdmin {
		return domain.ActivityWithCount{}, ErrForbidden
	}
	var activity domain.Activity
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&activity, "id = ?", id).Error; err != nil {
			return err
		}
		if patch.Name != nil {
			activity.Name = *patch.Name
		}
		if patch.Organizer != nil {
			activity.Organizer = *patch.Organizer
		}
		if patch.Level != nil {
			activity.Level = *patch.Level
		}
		if patch.Category != nil {
			activity.Category = *patch.Category
		}
		if patch.Range != nil {
			activity.Range = *patch.Range
		}
		if patch.Deadline != nil {
			activity.Deadline = *patch.Deadline
		}
		if patch.Source != nil {
			activity.Source = *patch.Source
		}
		if patch.Rule != nil {
			activity.Rule = *patch.Rule
		}
		if patch.Status != nil {
			activity.Status = *patch.Status
		}
		activity.UpdatedAt = s.now()
		if err := tx.Save(&activity).Error; err != nil {
			return err
		}
		return s.AddLog(tx, user, "维护活动名录", activity.Name, "更新活动信息")
	})
	if err != nil {
		return domain.ActivityWithCount{}, err
	}
	var count int64
	if err := s.db.Model(&domain.Application{}).Where("activity_id = ? AND status <> ?", id, domain.StatusRejected).Count(&count).Error; err != nil {
		return domain.ActivityWithCount{}, err
	}
	return domain.ActivityWithCount{Activity: activity, Applications: count}, nil
}

func (s *Service) Logs(user domain.User) ([]domain.AuditLog, error) {
	if user.Role != domain.RoleAdmin {
		return nil, ErrForbidden
	}
	var logs []domain.AuditLog
	err := s.db.Order("created_at ASC").Find(&logs).Error
	return logs, err
}
