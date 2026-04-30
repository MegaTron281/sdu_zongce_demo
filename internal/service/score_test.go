package service

import (
	"testing"

	"demo1/internal/domain"
)

func TestCalculateScoreUsesRuleMatrixAndRankDecay(t *testing.T) {
	rule := domain.RuleVersion{
		Matrix: map[string]float64{
			"省级:一等奖": 12,
		},
	}

	got := CalculateScore(rule, "省级", "一等奖", "集体项目", 3)
	if got != 7.2 {
		t.Fatalf("expected rank-decayed score 7.2, got %.1f", got)
	}
}

func TestCalculateScoreFloor(t *testing.T) {
	rule := domain.RuleVersion{
		Matrix: map[string]float64{
			"省级:一等奖": 12,
		},
	}

	got := CalculateScore(rule, "省级", "一等奖", "集体项目", 8)
	if got != 3.6 {
		t.Fatalf("expected floor score 3.6, got %.1f", got)
	}
}
