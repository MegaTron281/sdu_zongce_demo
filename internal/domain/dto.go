package domain

type AuthUser struct {
	ID          string `json:"id"`
	Role        string `json:"role"`
	DisplayName string `json:"displayName"`
	StudentID   string `json:"studentId,omitempty"`
	ClassName   string `json:"className,omitempty"`
}

func ToAuthUser(user User) AuthUser {
	return AuthUser{
		ID:          user.ID,
		Role:        user.Role,
		DisplayName: user.DisplayName,
		StudentID:   user.StudentID,
		ClassName:   user.ClassName,
	}
}

type LoginRequest struct {
	Role     string `json:"role" binding:"required"`
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	User AuthUser `json:"user"`
}

type MetaResponse struct {
	SummaryPhase        bool        `json:"summaryPhase"`
	ApplicationStatuses []string    `json:"applicationStatuses"`
	ActivityStatuses    []string    `json:"activityStatuses"`
	Modules             []string    `json:"modules"`
	ActiveRule          RuleSummary `json:"activeRule"`
}

type RuleSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SummaryPhaseRequest struct {
	SummaryPhase *bool `json:"summaryPhase" binding:"required"`
}

type ActivityPatch struct {
	Name      *string `json:"name"`
	Organizer *string `json:"organizer"`
	Level     *string `json:"level"`
	Category  *string `json:"category"`
	Range     *string `json:"range"`
	Deadline  *string `json:"deadline"`
	Source    *string `json:"source"`
	Rule      *string `json:"rule"`
	Status    *string `json:"status"`
}

type ApplicationCreateRequest struct {
	ActivityID  string   `json:"activityId" binding:"required"`
	AwardLevel  string   `json:"awardLevel" binding:"required"`
	ProjectType string   `json:"projectType" binding:"required"`
	Rank        int      `json:"rank"`
	Materials   []string `json:"materials"`
}

type ApplicationActionRequest struct {
	Reason string   `json:"reason"`
	Score  *float64 `json:"score"`
}

type ScorePreviewRequest struct {
	ActivityID    string `json:"activityId" binding:"required"`
	AwardLevel    string `json:"awardLevel" binding:"required"`
	ProjectType   string `json:"projectType" binding:"required"`
	Rank          int    `json:"rank"`
	ActivityLevel string `json:"activityLevel"`
}

type ScorePreviewResponse struct {
	Score    float64 `json:"score"`
	RuleID   string  `json:"ruleId"`
	RuleName string  `json:"ruleName"`
}
