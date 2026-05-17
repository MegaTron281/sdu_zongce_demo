package service

import (
	"errors"
	"math"

	"demo1/internal/domain"

	"gorm.io/gorm"
)

func (s *Service) ActiveRule(tx *gorm.DB) (domain.RuleVersion, error) {
	var rule domain.RuleVersion
	err := tx.Where("active = ?", true).Order("created_at DESC").First(&rule).Error
	if err != nil {
		if IsNotFound(err) {
			return domain.RuleVersion{}, errors.New("active rule not found")
		}
		return domain.RuleVersion{}, err
	}
	return rule, nil
}

func CalculateScore(rule domain.RuleVersion, level, awardLevel, projectType string, rank int) float64 {
	base := rule.Matrix[level+":"+awardLevel]
	score := base
	if projectType == "集体项目" && rank > 1 {
		score = base * (1 - float64(rank-1)*0.2)
		floor := base * 0.3
		if score < floor {
			score = floor
		}
	}
	return math.Round(score*10) / 10
}

func (s *Service) PreviewScore(req domain.ScorePreviewRequest) (domain.ScorePreviewResponse, error) {
	var activity domain.Activity
	if req.ActivityID != "" {
		if err := s.db.First(&activity, "id = ?", req.ActivityID).Error; err != nil {
			if !IsNotFound(err) {
				return domain.ScorePreviewResponse{}, err
			}
		}
	}
	level := activity.Level
	if level == "" {
		level = req.ActivityLevel
	}
	if level == "" {
		return domain.ScorePreviewResponse{}, errors.New("activity level is required")
	}
	rule, err := s.ActiveRule(s.db)
	if err != nil {
		return domain.ScorePreviewResponse{}, err
	}
	return domain.ScorePreviewResponse{
		Score:    CalculateScore(rule, level, req.AwardLevel, req.ProjectType, req.Rank),
		RuleID:   rule.ID,
		RuleName: rule.Name,
	}, nil
}

func (s *Service) StudentScore(studentID string) (domain.StudentScore, error) {
	var cycle domain.AssessmentCycle
	if err := s.db.Where("active = ?", true).First(&cycle).Error; err != nil && !IsNotFound(err) {
		return domain.StudentScore{}, err
	}
	rule, err := s.ActiveRule(s.db)
	if err != nil {
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

	for _, name := range domain.ModuleNames {
		max := rule.ModuleMax[name]
		value := modules[name]
		if max > 0 && value > max {
			value = max
		}
		score.Modules = append(score.Modules, domain.ScoreModule{Name: name, Score: value, Max: max})
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
