package service

import (
	"errors"
	"net/http"
	"time"

	"demo1/internal/domain"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrAccountNotFound    = errors.New("该账号不存在")
	ErrRoleMismatch       = errors.New("该账号与所选登录身份不符")
	ErrDuplicateApply     = errors.New("duplicate active application")
	ErrSummaryPhaseLocked = errors.New("summary phase is locked for new applications")
	ErrInvalidTransition  = errors.New("invalid status transition")
	ErrReasonRequired     = errors.New("reason is required")
	ErrWrongPassword      = errors.New("密码错误")
)

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func (s *Service) Login(req domain.LoginRequest) (domain.User, string, time.Time, error) {
	account := Trim(req.Account)
	password := Trim(req.Password)
	role := Trim(req.Role)
	if account == "" || password == "" || role == "" {
		return domain.User{}, "", time.Time{}, ErrUnauthorized
	}

	var user domain.User
	// 先按账号查人，把"账号不存在 / 身份不符 / 密码错误"三种失败区分开
	if err := s.db.Where("account = ?", account).First(&user).Error; err != nil {
		if IsNotFound(err) {
			return domain.User{}, "", time.Time{}, ErrAccountNotFound
		}
		return domain.User{}, "", time.Time{}, err
	}

	// 班委本身也是学生，允许以 student 身份登录（有学号即可，包括班委账号）
	if role == domain.RoleStudent {
		if user.StudentID == "" {
			return domain.User{}, "", time.Time{}, ErrRoleMismatch
		}
	} else if user.Role != role {
		return domain.User{}, "", time.Time{}, ErrRoleMismatch
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return domain.User{}, "", time.Time{}, ErrWrongPassword
	}

	// 班委以 student 身份登录时，session 中角色记为 student
	if role == domain.RoleStudent && user.StudentID != "" {
		user.Role = domain.RoleStudent
	}

	token, err := NewToken()
	if err != nil {
		return domain.User{}, "", time.Time{}, err
	}
	expiresAt := s.now().Add(7 * 24 * time.Hour)
	session := domain.Session{
		ID:        NewID("sess"),
		UserID:    user.ID,
		TokenHash: s.TokenHash(token),
		Role:      user.Role,
		ExpiresAt: expiresAt,
		CreatedAt: s.now(),
		UpdatedAt: s.now(),
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&session).Error; err != nil {
			return err
		}
		return s.AddLog(tx, user, "登录系统", roleLabel(user.Role), "账号密码登录")
	})
	if err != nil {
		return domain.User{}, "", time.Time{}, err
	}
	return user, token, expiresAt, nil
}

func (s *Service) AuthenticateToken(token string) (domain.User, error) {
	if Trim(token) == "" {
		return domain.User{}, ErrUnauthorized
	}
	var session domain.Session
	err := s.db.Preload("User").
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", s.TokenHash(token), s.now()).
		First(&session).Error
	if err != nil {
		if IsNotFound(err) {
			return domain.User{}, ErrUnauthorized
		}
		return domain.User{}, err
	}
	// 会话角色优先于用户表默认角色（双身份登录）
	if session.Role != "" {
		session.User.Role = session.Role
	} else if session.User.Role == domain.RoleReviewer {
		// 兼容修复：旧会话没有记录登录身份，班委按最小权限降级为学生视角，
		// 避免"我的申请"按班级展示全班数据；重新登录即可恢复班委身份。
		session.User.Role = domain.RoleStudent
	}
	return session.User, nil
}

func (s *Service) Logout(token string, user domain.User) error {
	if Trim(token) == "" {
		return nil
	}
	now := s.now()
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.Session{}).
			Where("token_hash = ? AND revoked_at IS NULL", s.TokenHash(token)).
			Updates(map[string]any{"revoked_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		return s.AddLog(tx, user, "退出系统", roleLabel(user.Role), "主动退出")
	})
}

func SessionCookie(token string, expiresAt time.Time, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	}
}

func ClearSessionCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	}
}

func roleLabel(role string) string {
	switch role {
	case domain.RoleStudent:
		return "普通学生"
	case domain.RoleReviewer:
		return "审核学生"
	case domain.RoleAdmin:
		return "管理员"
	default:
		return role
	}
}
