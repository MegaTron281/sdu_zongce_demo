package httpapi

import (
	"encoding/csv"
	"net/http"
	"sort"
	"strconv"

	"demo1/internal/domain"
	"demo1/internal/service"

	"github.com/gin-gonic/gin"
)

func pagination(c *gin.Context) (limit int, offset int) {
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return
}

func (s *Server) login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}
	user, token, expiresAt, err := s.svc.Login(req)
	if err != nil {
		writeError(c, err)
		return
	}
	http.SetCookie(c.Writer, service.SessionCookie(token, expiresAt, s.cfg.CookieSecure))
	// 同时把令牌返回给前端，前端存入 sessionStorage 并通过 Bearer 头使用，
	// 使同一浏览器内的多个标签页可各自保持独立登录状态（并行多账号操作）。
	c.JSON(http.StatusOK, domain.LoginResponse{User: domain.ToAuthUser(user), Token: token})
}

func (s *Server) me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"user": domain.ToAuthUser(currentUser(c))})
}

func (s *Server) logout(c *gin.Context) {
	// 优先用 Bearer 头里的令牌（标签页级），否则回退到 cookie 令牌
	token := bearerToken(c)
	if token == "" {
		token, _ = c.Cookie(service.SessionCookieName)
	}
	if err := s.svc.Logout(token, currentUser(c)); err != nil {
		writeError(c, err)
		return
	}
	http.SetCookie(c.Writer, service.ClearSessionCookie(s.cfg.CookieSecure))
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) meta(c *gin.Context) {
	meta, err := s.svc.Meta()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, meta)
}

func (s *Server) patchMeta(c *gin.Context) {
	var req domain.SummaryPhaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}
	meta, err := s.svc.SetSummaryPhase(currentUser(c), *req.SummaryPhase)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, meta)
}

func (s *Server) listActivities(c *gin.Context) {
	items, err := s.svc.ListActivities(c.Query("q"), c.Query("category"), c.Query("level"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) createActivity(c *gin.Context) {
	var activity domain.Activity
	if err := c.ShouldBindJSON(&activity); err != nil {
		writeError(c, err)
		return
	}
	item, err := s.svc.CreateActivity(currentUser(c), activity)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (s *Server) getActivity(c *gin.Context) {
	var activity domain.Activity
	if err := s.svc.DB().First(&activity, "id = ?", c.Param("id")).Error; err != nil {
		writeError(c, err)
		return
	}
	var count int64
	_ = s.svc.DB().Model(&domain.Application{}).
		Where("activity_id = ? AND status <> ?", activity.ID, domain.StatusRejected).
		Count(&count).Error
	c.JSON(http.StatusOK, domain.ActivityWithCount{Activity: activity, Applications: count})
}

func (s *Server) patchActivity(c *gin.Context) {
	var patch domain.ActivityPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		writeError(c, err)
		return
	}
	item, err := s.svc.PatchActivity(currentUser(c), c.Param("id"), patch)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) listApplications(c *gin.Context) {
	limit, offset := pagination(c)
	items, err := s.svc.ListApplications(currentUser(c), c.Query("status"), c.Query("studentId"), c.Query("className"), limit, offset)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) createApplication(c *gin.Context) {
	var req domain.ApplicationCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}
	app, err := s.svc.CreateApplication(currentUser(c), req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, app)
}

func (s *Server) applicationAction(c *gin.Context) {
	var req domain.ApplicationActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}
	app, err := s.svc.ApplicationAction(currentUser(c), c.Param("id"), c.Param("action"), req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, app)
}

func (s *Server) publicity(c *gin.Context) {
	items, err := s.svc.Publicity(currentUser(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) score(c *gin.Context) {
	score, err := s.svc.StudentScoreFor(currentUser(c), c.Param("studentId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, score)
}

func (s *Server) logs(c *gin.Context) {
	logs, err := s.svc.Logs(currentUser(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (s *Server) exportScores(c *gin.Context) {
	actor := currentUser(c)
	ids := map[string]bool{}
	// 系统管理员导出全校；辅导员只导出管辖班级；未配置管辖班级则导出空表（fail-closed）
	if service.IsSuperAdmin(actor) {
		var academics []domain.AcademicScore
		if err := s.svc.DB().Find(&academics).Error; err != nil {
			writeError(c, err)
			return
		}
		for _, item := range academics {
			ids[item.StudentID] = true
		}
		var apps []domain.Application
		if err := s.svc.DB().Select("student_id").Find(&apps).Error; err != nil {
			writeError(c, err)
			return
		}
		for _, app := range apps {
			ids[app.StudentID] = true
		}
	} else if len(actor.ManagedClasses) > 0 {
		var academics []domain.AcademicScore
		if err := s.svc.DB().Where("class_name IN ?", actor.ManagedClasses).Find(&academics).Error; err != nil {
			writeError(c, err)
			return
		}
		for _, item := range academics {
			ids[item.StudentID] = true
		}
		var apps []domain.Application
		if err := s.svc.DB().Select("student_id").
			Where("class_name IN ?", actor.ManagedClasses).
			Find(&apps).Error; err != nil {
			writeError(c, err)
			return
		}
		for _, app := range apps {
			ids[app.StudentID] = true
		}
	}

	sortedIDs := make([]string, 0, len(ids))
	for id := range ids {
		sortedIDs = append(sortedIDs, id)
	}
	sort.Strings(sortedIDs)

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="scores.csv"`)
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()
	_ = writer.Write([]string{"学号", "姓名", "班级", "学业成绩", "素质能力总分", "综测成绩"})
	for _, id := range sortedIDs {
		score, err := s.svc.StudentScore(id)
		if err != nil {
			writeError(c, err)
			return
		}
		_ = writer.Write([]string{
			score.StudentID,
			score.Student,
			score.ClassName,
			formatFloat(score.AcademicScore),
			formatFloat(score.QualityScore),
			formatFloat(score.AssessmentScore),
		})
	}
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 1, 64)
}

func (s *Server) changePassword(c *gin.Context) {
	var req domain.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}
	if err := s.svc.ChangePassword(currentUser(c), req.OldPassword, req.NewPassword); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) listUsers(c *gin.Context) {
	limit, offset := pagination(c)
	items, err := s.svc.ListUsers(currentUser(c), limit, offset)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) importUsers(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		writeError(c, err)
		return
	}
	f, err := file.Open()
	if err != nil {
		writeError(c, err)
		return
	}
	defer f.Close()
	result := s.svc.ImportUsersCSV(currentUser(c), f)
	c.JSON(http.StatusOK, result)
}

func (s *Server) updateUserRole(c *gin.Context) {
	var req domain.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}
	if err := s.svc.UpdateUserRole(currentUser(c), c.Param("id"), req.Role); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) listObjections(c *gin.Context) {
	list, err := s.svc.ListObjections(currentUser(c), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (s *Server) updateUserClasses(c *gin.Context) {
	var req domain.UpdateClassesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}
	user, err := s.svc.UpdateUserClasses(currentUser(c), c.Param("id"), req.Classes)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": domain.ToAuthUser(user)})
}
