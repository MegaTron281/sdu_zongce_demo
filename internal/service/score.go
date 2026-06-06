package service

import (
	"math"

	"demo1/internal/domain"
)

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
