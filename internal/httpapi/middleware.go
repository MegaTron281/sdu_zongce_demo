package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"demo1/internal/domain"
	"demo1/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const userContextKey = "authUser"

// bearerToken 从 Authorization: Bearer <token> 头中取出会话令牌。
// 浏览器对每个同源请求只会自动携带一个 sdu_session cookie，多个标签页共用同一份
// cookie 会导致"后登录的账号顶掉先登录的账号"。改用 Bearer 头后，每个标签页可
// 在 sessionStorage 中保存各自独立的令牌，从而实现同一浏览器内的并行多账号操作。
func bearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.HasPrefix(h, prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}

func (s *Server) authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先使用 Bearer 头（标签页级独立令牌），其次回退到 cookie（兼容单账号场景）
		token := bearerToken(c)
		if token == "" {
			if ck, err := c.Cookie(service.SessionCookieName); err == nil {
				token = ck
			}
		}
		if token == "" {
			writeError(c, service.ErrUnauthorized)
			c.Abort()
			return
		}
		user, err := s.svc.AuthenticateToken(token)
		if err != nil {
			writeError(c, err)
			c.Abort()
			return
		}
		c.Set(userContextKey, user)
		c.Next()
	}
}

func (s *Server) requireRole(roles ...string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, role := range roles {
		allowed[role] = true
	}
	return func(c *gin.Context) {
		user := currentUser(c)
		if !allowed[user.Role] {
			writeError(c, service.ErrForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}

func currentUser(c *gin.Context) domain.User {
	value, ok := c.Get(userContextKey)
	if !ok {
		return domain.User{}
	}
	user, _ := value.(domain.User)
	return user
}

func writeError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, service.ErrUnauthorized),
		errors.Is(err, service.ErrAccountNotFound),
		errors.Is(err, service.ErrRoleMismatch):
		status = http.StatusUnauthorized
	case errors.Is(err, service.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, gorm.ErrRecordNotFound), service.IsNotFound(err):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrDuplicateApply),
		errors.Is(err, service.ErrSummaryPhaseLocked),
		errors.Is(err, service.ErrInvalidTransition):
		status = http.StatusConflict
	case errors.Is(err, service.ErrReasonRequired),
		errors.Is(err, service.ErrWrongPassword):
		status = http.StatusBadRequest
	}
	c.JSON(status, gin.H{"error": err.Error()})
}
