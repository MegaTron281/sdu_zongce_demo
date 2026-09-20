package httpapi

import (
	"errors"
	"net/http"

	"demo1/internal/domain"
	"demo1/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const userContextKey = "authUser"

func (s *Server) authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(service.SessionCookieName)
		if err != nil {
			writeError(c, service.ErrUnauthorized)
			c.Abort()
			return
		}
		user, err := s.svc.AuthenticateToken(cookie)
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
