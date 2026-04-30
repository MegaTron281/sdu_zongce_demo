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
