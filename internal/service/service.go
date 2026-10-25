package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"demo1/internal/domain"

	"gorm.io/gorm"
)

const SessionCookieName = "sdu_session"

type Service struct {
	db            *gorm.DB
	sessionSecret string
	now           func() time.Time
}

func New(db *gorm.DB, sessionSecret string) *Service {
	return &Service{db: db, sessionSecret: sessionSecret, now: time.Now}
}

func (s *Service) DB() *gorm.DB {
	return s.db
}

func NewID(prefix string) string {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + base64.RawURLEncoding.EncodeToString(raw)
}

func NewToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (s *Service) TokenHash(token string) string {
	sum := sha256.Sum256([]byte(s.sessionSecret + "." + token))
	return hex.EncodeToString(sum[:])
}

func Trim(value string) string {
	return strings.TrimSpace(value)
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func (s *Service) AddLog(tx *gorm.DB, actor domain.User, action, target, note string) error {
	return tx.Create(&domain.AuditLog{
		ID:        NewID("log"),
		Actor:     actor.DisplayName,
		ActorRole: actor.Role,
		Action:    action,
		Target:    target,
		Note:      strings.TrimSpace(note),
		ClassName: actor.ClassName,
		CreatedAt: s.now(),
	}).Error
}

func (s *Service) systemUser() domain.User {
	return domain.User{ID: "system", Role: "system", DisplayName: "系统"}
}

func RequireReason(reason, message string) (string, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "", fmt.Errorf("%w: %s", ErrReasonRequired, message)
	}
	return reason, nil
}

func DefaultText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

// classesContain 判断班级是否在管辖/可见班级集合内。
func classesContain(classes []string, className string) bool {
	className = strings.TrimSpace(className)
	if className == "" {
		return false
	}
	for _, item := range classes {
		if strings.TrimSpace(item) == className {
			return true
		}
	}
	return false
}

// visibleClasses 返回用户可见的班级集合：
// student/reviewer 只看本班；admin 只看管辖班级（未配置则不可见任何班级数据，fail-closed）。
func (s *Service) visibleClasses(user domain.User) []string {
	switch user.Role {
	case domain.RoleStudent, domain.RoleReviewer:
		if user.ClassName == "" {
			return nil
		}
		return []string{user.ClassName}
	case domain.RoleAdmin:
		return user.ManagedClasses
	default:
		return nil
	}
}

// IsSuperAdmin 系统管理员：admin 角色 + 超管标志，不受管辖班级限制，唯一能指定辅导员管辖班级的身份。
func IsSuperAdmin(user domain.User) bool {
	return user.Role == domain.RoleAdmin && user.IsSuperAdmin
}

// canAccessClass 判断用户能否访问某班级数据：系统管理员不受限，其余按 visibleClasses。
func (s *Service) canAccessClass(user domain.User, className string) bool {
	if IsSuperAdmin(user) {
		return true
	}
	return classesContain(s.visibleClasses(user), className)
}
