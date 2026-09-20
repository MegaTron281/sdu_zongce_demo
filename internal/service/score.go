package service

import (
	"math"

	"demo1/internal/domain"
)

// studentClass 查询某学号所属班级：优先账号表，其次学业成绩表，最后申请记录。
func (s *Service) studentClass(studentID string) string {
	if studentID == "" {
		return ""
	}
	var user domain.User
	if err := s.db.Select("class_name").First(&user, "student_id = ? AND student_id <> ''", studentID).Error; err == nil && user.ClassName != "" {
		return user.ClassName
	}
	var academic domain.AcademicScore
	if err := s.db.Select("class_name").First(&academic, "student_id = ?", studentID).Error; err == nil && academic.ClassName != "" {
		return academic.ClassName
	}
	var app domain.Application
	if err := s.db.Select("class_name").First(&app, "student_id = ?", studentID).Error; err == nil {
		return app.ClassName
	}
	return ""
}

// StudentScoreFor 带越权校验的成绩查询：
// student 只能查本人；reviewer 只能查本班学生；admin 只能查管辖班级学生。
func (s *Service) StudentScoreFor(user domain.User, studentID string) (domain.StudentScore, error) {
	switch user.Role {
	case domain.RoleStudent:
		if user.StudentID != studentID {
			return domain.StudentScore{}, ErrForbidden
		}
	case domain.RoleReviewer:
		if studentID != user.StudentID {
			if targetClass := s.studentClass(studentID); targetClass != user.ClassName {
				return domain.StudentScore{}, ErrForbidden
			}
		}
	case domain.RoleAdmin:
		// 系统管理员可查全校；辅导员只能查管辖班级学生
		if !IsSuperAdmin(user) && !classesContain(user.ManagedClasses, s.studentClass(studentID)) {
			return domain.StudentScore{}, ErrForbidden
		}
	default:
		return domain.StudentScore{}, ErrForbidden
	}
	return s.StudentScore(studentID)
}

func (s *Service) StudentScore(studentID string) (domain.StudentScore, error) {
	var cycle domain.AssessmentCycle
	if err := s.db.Where("active = ?", true).First(&cycle).Error; err != nil && !IsNotFound(err) {
		return domain.StudentScore{}, err
	}

	modules := map[string]float64{}
	for _, name := range domain.ModuleNames {
		modules[name] = 0
	}
	var apps []domain.Application
	if err := s.db.Where("student_id = ? AND status = ?", studentID, domain.StatusApproved).Find(&apps).Error; err != nil {
		return domain.StudentScore{}, err
	}

	score := domain.StudentScore{StudentID: studentID, Locked: cycle.SummaryPhase}
	for _, app := range apps {
		if score.Student == "" {
			score.Student = app.Student
			score.ClassName = app.ClassName
		}
		modules[app.Category] += app.Score
	}

	var academic domain.AcademicScore
	if err := s.db.First(&academic, "student_id = ?", studentID).Error; err == nil {
		score.Student = firstNonEmpty(score.Student, academic.Student)
		score.ClassName = firstNonEmpty(score.ClassName, academic.ClassName)
		score.AcademicScore = academic.Score
	} else if !IsNotFound(err) {
		return domain.StudentScore{}, err
	}

	// 学生还没有已通过申请和学业成绩记录时，从用户表兜底补齐姓名班级
	if score.Student == "" || score.ClassName == "" {
		var u domain.User
		if err := s.db.Where("student_id = ?", studentID).First(&u).Error; err == nil {
			score.Student = firstNonEmpty(score.Student, u.DisplayName)
			score.ClassName = firstNonEmpty(score.ClassName, u.ClassName)
		}
	}

	for _, name := range domain.ModuleNames {
		value := modules[name]
		score.Modules = append(score.Modules, domain.ScoreModule{Name: name, Score: value, Max: value})
		score.QualityScore += value
	}
	score.AssessmentScore = math.Round((score.AcademicScore*0.9+score.QualityScore*0.1)*10) / 10
	return score, nil
}

func firstNonEmpty(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}
