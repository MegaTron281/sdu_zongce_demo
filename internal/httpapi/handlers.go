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
	c.JSON(http.StatusOK, domain.LoginResponse{User: domain.ToAuthUser(user)})
}

func (s *Server) me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"user": domain.ToAuthUser(currentUser(c))})
}

func (s *Server) logout(c *gin.Context) {
	token, _ := c.Cookie(service.SessionCookieName)
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
	items, err := s.svc.ListApplications(currentUser(c), c.Query("status"), c.Query("studentId"), c.Query("className"))
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
	user := currentUser(c)
	studentID := c.Param("studentId")
	if user.Role == domain.RoleStudent && user.StudentID != studentID {
		writeError(c, service.ErrForbidden)
		return
	}
	score, err := s.svc.StudentScore(studentID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, score)
}

func (s *Server) previewScore(c *gin.Context) {
	var req domain.ScorePreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}
	resp, err := s.svc.PreviewScore(req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Server) listRules(c *gin.Context) {
	rules, err := s.svc.ListRules()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, rules)
}

func (s *Server) createRule(c *gin.Context) {
	var rule domain.RuleVersion
	if err := c.ShouldBindJSON(&rule); err != nil {
		writeError(c, err)
		return
	}
	created, err := s.svc.CreateRule(currentUser(c), rule)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
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
	ids := map[string]bool{}
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
