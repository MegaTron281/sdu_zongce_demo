package domain

import "time"

type AuthUser struct {
	ID             string   `json:"id"`
	Role           string   `json:"role"`
	DisplayName    string   `json:"displayName"`
	StudentID      string   `json:"studentId,omitempty"`
	ClassName      string   `json:"className,omitempty"`
	ManagedClasses []string `json:"managedClasses,omitempty"`
	IsSuperAdmin   bool     `json:"isSuperAdmin,omitempty"`
}

func ToAuthUser(user User) AuthUser {
	return AuthUser{
		ID:             user.ID,
		Role:           user.Role,
		DisplayName:    user.DisplayName,
		StudentID:      user.StudentID,
		ClassName:      user.ClassName,
		ManagedClasses: user.ManagedClasses,
		IsSuperAdmin:   user.IsSuperAdmin,
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
	SummaryPhase        bool     `json:"summaryPhase"`
	ApplicationStatuses []string `json:"applicationStatuses"`
	ActivityStatuses    []string `json:"activityStatuses"`
	Modules             []string `json:"modules"`
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
	ActivityID   string   `json:"activityId"`
	ActivityName string   `json:"activityName"`
	Organizer    string   `json:"organizer"`
	Level        string   `json:"level"`
	Category     string   `json:"category"`
	Range        string   `json:"range"`
	Deadline     string   `json:"deadline"`
	Source       string   `json:"source"`
	Rule         string   `json:"rule"`
	AwardLevel   string   `json:"awardLevel" binding:"required"`
	ProjectType  string   `json:"projectType" binding:"required"`
	Rank         int      `json:"rank"`
	Score        float64  `json:"score" binding:"required"`
	Materials    []string `json:"materials"`
}

type ApplicationActionRequest struct {
	Reason string   `json:"reason"`
	Score  *float64 `json:"score"`
}

type ObjectionRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

type UserListItem struct {
	ID             string    `json:"id"`
	Role           string    `json:"role"`
	DisplayName    string    `json:"displayName"`
	StudentID      string    `json:"studentId,omitempty"`
	ClassName      string    `json:"className,omitempty"`
	Account        string    `json:"account"`
	ManagedClasses []string  `json:"managedClasses,omitempty"`
	IsSuperAdmin   bool      `json:"isSuperAdmin,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

type UserImportResult struct {
	Created int      `json:"created"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
}

type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

type UpdateClassesRequest struct {
	Classes []string `json:"classes" binding:"required"`
}
