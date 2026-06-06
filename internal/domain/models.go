package domain

import "time"

const (
	RoleStudent  = "student"
	RoleReviewer = "reviewer"
	RoleAdmin    = "admin"
)

const (
	StatusPendingReviewer = "待班委审核"
	StatusPublicity       = "公示中"
	StatusPendingAdmin    = "待管理员审核"
	StatusApproved        = "已通过"
	StatusRejected        = "已驳回"
	StatusSpecial         = "特殊情况处理中"
	StatusAppealing       = "申诉中"
)

const (
	ActivityStatusListed  = "已上架"
	ActivityStatusPending = "待审核上架"
	ActivityStatusReject  = "已驳回"
)

var ApplicationStatuses = []string{
	StatusPendingReviewer,
	StatusPublicity,
	StatusPendingAdmin,
	StatusApproved,
	StatusRejected,
	StatusSpecial,
	StatusAppealing,
}

var ActivityStatuses = []string{
	ActivityStatusListed,
	ActivityStatusPending,
	ActivityStatusReject,
}

var ModuleNames = []string{"身心素养", "文艺素养", "劳动素养", "创新素养"}

type User struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	Role         string    `json:"role"`
	DisplayName  string    `json:"displayName"`
	StudentID    string    `json:"studentId,omitempty"`
	ClassName    string    `json:"className,omitempty"`
	Account      string    `json:"account"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Session struct {
	ID        string     `gorm:"primaryKey" json:"id"`
	UserID    string     `json:"userId"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expiresAt"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	User      User       `json:"user"`
}

type AssessmentCycle struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	Name         string    `json:"name"`
	SummaryPhase bool      `json:"summaryPhase"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Activity struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	Organizer string    `json:"organizer"`
	Level     string    `json:"level"`
	Category  string    `json:"category"`
	Range     string    `json:"range"`
	Deadline  string    `json:"deadline"`
	Source    string    `json:"source"`
	Rule      string    `json:"rule"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Application struct {
	ID              string     `gorm:"primaryKey" json:"id"`
	Student         string     `json:"student"`
	StudentID       string     `json:"studentId"`
	ClassName       string     `json:"className"`
	ActivityID      string     `json:"activityId"`
	Activity        string     `json:"activity"`
	Category        string     `json:"category"`
	AwardLevel      string     `json:"awardLevel"`
	ProjectType     string     `json:"projectType"`
	Rank            int        `json:"rank"`
	Score           float64    `json:"score"`
	ClaimedScore    float64    `json:"claimedScore"`
	Status          string     `json:"status"`
	SubmittedAt     string     `json:"submittedAt"`
	PublicityUntil  *time.Time `json:"publicityUntil,omitempty"`
	ReviewerOpinion string     `json:"reviewerOpinion"`
	AdminOpinion    string     `json:"adminOpinion"`
	SpecialReason   string     `json:"specialReason,omitempty"`
	Materials       []string   `gorm:"serializer:json;type:jsonb" json:"materials"`
	CycleID         string     `json:"cycleId"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type AcademicScore struct {
	StudentID string    `gorm:"primaryKey" json:"studentId"`
	Student   string    `json:"student"`
	ClassName string    `json:"className"`
	Score     float64   `json:"score"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Objection struct {
	ID            string    `gorm:"primaryKey" json:"id"`
	ApplicationID string    `json:"applicationId"`
	Student       string    `json:"student"`
	StudentID     string    `json:"studentId"`
	Reason        string    `json:"reason"`
	CreatedAt     time.Time `json:"createdAt"`
}

type AuditLog struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	Actor     string    `json:"actor"`
	ActorRole string    `json:"actorRole"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"createdAt"`
}

type ActivityWithCount struct {
	Activity
	Applications int64 `json:"applications"`
}

type ScoreModule struct {
	Name  string  `json:"name"`
	Score float64 `json:"score"`
	Max   float64 `json:"max"`
}

type StudentScore struct {
	Student         string        `json:"student"`
	StudentID       string        `json:"studentId"`
	ClassName       string        `json:"className"`
	AcademicScore   float64       `json:"academicScore"`
	QualityScore    float64       `json:"qualityScore"`
	AssessmentScore float64       `json:"assessmentScore"`
	Modules         []ScoreModule `json:"modules"`
	Locked          bool          `json:"locked"`
}
