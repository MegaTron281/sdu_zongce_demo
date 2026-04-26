package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	StatusPendingReviewer = "待班委审核"
	StatusPublicity       = "公示中"
	StatusPendingAdmin    = "待管理员审核"
	StatusApproved        = "已通过"
	StatusRejected        = "已驳回"
	StatusSpecial         = "特殊情况处理中"
)

type Activity struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Organizer    string `json:"organizer"`
	Level        string `json:"level"`
	Category     string `json:"category"`
	Range        string `json:"range"`
	Deadline     string `json:"deadline"`
	Source       string `json:"source"`
	Applications int    `json:"applications"`
	Rule         string `json:"rule"`
	Status       string `json:"status"`
}

type Application struct {
	ID              string    `json:"id"`
	Student         string    `json:"student"`
	StudentID       string    `json:"studentId"`
	ClassName       string    `json:"className"`
	ActivityID      string    `json:"activityId"`
	Activity        string    `json:"activity"`
	Category        string    `json:"category"`
	AwardLevel      string    `json:"awardLevel"`
	ProjectType     string    `json:"projectType"`
	Rank            int       `json:"rank"`
	Score           float64   `json:"score"`
	Status          string    `json:"status"`
	SubmittedAt     string    `json:"submittedAt"`
	PublicityUntil  string    `json:"publicityUntil,omitempty"`
	ReviewerOpinion string    `json:"reviewerOpinion"`
	AdminOpinion    string    `json:"adminOpinion"`
	SpecialReason   string    `json:"specialReason,omitempty"`
	Materials       []string  `json:"materials"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
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

type RuleVersion struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Active      bool               `json:"active"`
	PublishedAt string             `json:"publishedAt"`
	Matrix      map[string]float64 `json:"matrix"`
	ModuleMax   map[string]float64 `json:"moduleMax"`
}

type AuditLog struct {
	ID        string    `json:"id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"createdAt"`
}

type Store struct {
	mu           sync.RWMutex
	activities   map[string]Activity
	applications map[string]Application
	rules        []RuleVersion
	logs         []AuditLog
	summaryPhase bool
	nextAppID    int
	nextLogID    int
}

type Server struct {
	store *Store
}

func main() {
	store := seedStore()
	server := &Server{store: store}
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", server.handleHealth)
	mux.HandleFunc("/api/meta", server.handleMeta)
	mux.HandleFunc("/api/activities", server.handleActivities)
	mux.HandleFunc("/api/activities/", server.handleActivityByID)
	mux.HandleFunc("/api/applications", server.handleApplications)
	mux.HandleFunc("/api/applications/", server.handleApplicationAction)
	mux.HandleFunc("/api/publicity", server.handlePublicity)
	mux.HandleFunc("/api/scores/", server.handleScores)
	mux.HandleFunc("/api/admin/rules", server.handleRules)
	mux.HandleFunc("/api/admin/export/scores.csv", server.handleExportScores)
	mux.HandleFunc("/api/admin/logs", server.handleLogs)

	mux.Handle("/", http.FileServer(http.Dir(".")))

	addr := ":8080"
	log.Printf("server listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, withCORS(mux)))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.store.mu.RLock()
		defer s.store.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"summaryPhase": s.store.summaryPhase})
	case http.MethodPatch:
		var body struct {
			SummaryPhase *bool `json:"summaryPhase"`
		}
		if err := readJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		s.store.mu.Lock()
		if body.SummaryPhase != nil {
			s.store.summaryPhase = *body.SummaryPhase
			s.store.addLogLocked("管理员", "切换综测周期", "系统", fmt.Sprintf("summaryPhase=%t", *body.SummaryPhase))
		}
		value := s.store.summaryPhase
		s.store.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"summaryPhase": value})
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleActivities(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		category := r.URL.Query().Get("category")
		level := r.URL.Query().Get("level")
		keyword := strings.TrimSpace(r.URL.Query().Get("q"))

		s.store.mu.RLock()
		defer s.store.mu.RUnlock()
		items := make([]Activity, 0, len(s.store.activities))
		for _, activity := range s.store.activities {
			if category != "" && category != "全部" && activity.Category != category {
				continue
			}
			if level != "" && level != "全部" && activity.Level != level {
				continue
			}
			if keyword != "" && !strings.Contains(activity.Name+activity.Organizer+activity.Category, keyword) {
				continue
			}
			items = append(items, activity)
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var activity Activity
		if err := readJSON(r, &activity); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if activity.ID == "" {
			activity.ID = "a" + strconv.FormatInt(time.Now().UnixNano(), 36)
		}
		if activity.Status == "" {
			activity.Status = "待审核上架"
		}
		s.store.mu.Lock()
		s.store.activities[activity.ID] = activity
		s.store.addLogLocked("学生", "申请补录活动", activity.Name, "进入待审核上架")
		s.store.mu.Unlock()
		writeJSON(w, http.StatusCreated, activity)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleActivityByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/activities/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	activity, ok := s.store.activities[id]
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("activity not found"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, activity)
	case http.MethodPatch:
		var patch Activity
		if err := readJSON(r, &patch); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		activity = mergeActivity(activity, patch)
		s.store.activities[id] = activity
		s.store.addLogLocked("管理员", "维护活动名录", activity.Name, "更新活动信息")
		writeJSON(w, http.StatusOK, activity)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleApplications(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		studentID := r.URL.Query().Get("studentId")
		className := r.URL.Query().Get("className")

		s.store.mu.RLock()
		defer s.store.mu.RUnlock()
		items := make([]Application, 0, len(s.store.applications))
		for _, app := range s.store.applications {
			if status != "" && app.Status != status {
				continue
			}
			if studentID != "" && app.StudentID != studentID {
				continue
			}
			if className != "" && app.ClassName != className {
				continue
			}
			items = append(items, app)
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var req Application
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		s.store.mu.Lock()
		defer s.store.mu.Unlock()
		if s.store.summaryPhase {
			writeError(w, http.StatusConflict, errors.New("summary phase is locked for new applications"))
			return
		}
		if err := s.store.checkDuplicateLocked(req.StudentID, req.ActivityID); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
		activity, ok := s.store.activities[req.ActivityID]
		if !ok {
			writeError(w, http.StatusBadRequest, errors.New("activity not found"))
			return
		}
		s.store.nextAppID++
		now := time.Now()
		req.ID = fmt.Sprintf("s%d", s.store.nextAppID)
		req.Activity = activity.Name
		req.Category = activity.Category
		req.Score = calculateScore(activity.Level, req.AwardLevel, req.ProjectType, req.Rank)
		req.Status = StatusPendingReviewer
		req.SubmittedAt = now.Format("2006-01-02")
		req.CreatedAt = now
		req.UpdatedAt = now
		s.store.applications[req.ID] = req
		activity.Applications++
		s.store.activities[activity.ID] = activity
		s.store.addLogLocked(req.Student, "提交申请", req.Activity, "进入班委初审")
		writeJSON(w, http.StatusCreated, req)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleApplicationAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/applications/"), "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	id, action := parts[0], parts[1]
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req struct {
		Actor  string  `json:"actor"`
		Reason string  `json:"reason"`
		Score  float64 `json:"score"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Actor == "" {
		req.Actor = "系统用户"
	}

	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	app, ok := s.store.applications[id]
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("application not found"))
		return
	}
	now := time.Now()

	switch action {
	case "review-pass":
		if app.Status != StatusPendingReviewer {
			writeError(w, http.StatusConflict, errors.New("application is not pending reviewer"))
			return
		}
		app.Status = StatusPublicity
		app.ReviewerOpinion = defaultText(req.Reason, "班委初审通过")
		app.PublicityUntil = now.Add(48 * time.Hour).Format(time.RFC3339)
		s.store.addLogLocked(req.Actor, "班委初审通过", app.Activity, app.ReviewerOpinion)
	case "review-reject":
		app.Status = StatusRejected
		app.ReviewerOpinion = requireReason(req.Reason)
		if app.ReviewerOpinion == "" {
			writeError(w, http.StatusBadRequest, errors.New("reject reason is required"))
			return
		}
		s.store.addLogLocked(req.Actor, "班委驳回", app.Activity, app.ReviewerOpinion)
	case "mark-special":
		app.Status = StatusSpecial
		app.SpecialReason = requireReason(req.Reason)
		if app.SpecialReason == "" {
			writeError(w, http.StatusBadRequest, errors.New("special reason is required"))
			return
		}
		s.store.addLogLocked(req.Actor, "标记特殊情况", app.Activity, app.SpecialReason)
	case "admin-pass":
		if app.Status != StatusPendingAdmin && app.Status != StatusSpecial && app.Status != StatusPublicity {
			writeError(w, http.StatusConflict, errors.New("application is not ready for admin review"))
			return
		}
		if req.Score > 0 {
			app.Score = req.Score
			app.AdminOpinion = requireReason(req.Reason)
			if app.AdminOpinion == "" {
				writeError(w, http.StatusBadRequest, errors.New("manual score reason is required"))
				return
			}
		} else {
			app.AdminOpinion = defaultText(req.Reason, "管理员复审通过")
		}
		app.Status = StatusApproved
		s.store.addLogLocked(req.Actor, "管理员复审通过", app.Activity, app.AdminOpinion)
	case "admin-reject":
		app.Status = StatusRejected
		app.AdminOpinion = requireReason(req.Reason)
		if app.AdminOpinion == "" {
			writeError(w, http.StatusBadRequest, errors.New("reject reason is required"))
			return
		}
		s.store.addLogLocked(req.Actor, "管理员驳回", app.Activity, app.AdminOpinion)
	case "publicity-finish":
		if app.Status != StatusPublicity {
			writeError(w, http.StatusConflict, errors.New("application is not in publicity"))
			return
		}
		app.Status = StatusPendingAdmin
		s.store.addLogLocked(req.Actor, "班级公示结束", app.Activity, "进入管理员复审")
	default:
		http.NotFound(w, r)
		return
	}

	app.UpdatedAt = now
	s.store.applications[id] = app
	writeJSON(w, http.StatusOK, app)
}

func (s *Server) handlePublicity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()
	items := make([]Application, 0)
	for _, app := range s.store.applications {
		if app.Status == StatusPublicity {
			items = append(items, app)
		}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleScores(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	studentID := strings.TrimPrefix(r.URL.Path, "/api/scores/")
	if studentID == "" {
		http.NotFound(w, r)
		return
	}

	s.store.mu.RLock()
	defer s.store.mu.RUnlock()
	score := s.store.calculateStudentScoreLocked(studentID)
	writeJSON(w, http.StatusOK, score)
}

func (s *Server) handleRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.store.mu.RLock()
		defer s.store.mu.RUnlock()
		writeJSON(w, http.StatusOK, s.store.rules)
	case http.MethodPost:
		var rule RuleVersion
		if err := readJSON(r, &rule); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if rule.ID == "" {
			rule.ID = "rule-" + strconv.FormatInt(time.Now().Unix(), 10)
		}
		rule.Active = true
		rule.PublishedAt = time.Now().Format("2006-01-02")
		s.store.mu.Lock()
		for i := range s.store.rules {
			s.store.rules[i].Active = false
		}
		s.store.rules = append(s.store.rules, rule)
		s.store.addLogLocked("管理员", "发布规则版本", rule.Name, "新规则仅对新提交申请生效")
		s.store.mu.Unlock()
		writeJSON(w, http.StatusCreated, rule)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleExportScores(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="scores.csv"`)
	writer := csv.NewWriter(w)
	defer writer.Flush()

	_ = writer.Write([]string{"学号", "姓名", "班级", "学业成绩", "素质能力总分", "综测成绩"})
	seen := map[string]bool{}
	for _, app := range s.store.applications {
		if seen[app.StudentID] {
			continue
		}
		seen[app.StudentID] = true
		score := s.store.calculateStudentScoreLocked(app.StudentID)
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

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()
	writeJSON(w, http.StatusOK, s.store.logs)
}

func seedStore() *Store {
	now := time.Now()
	store := &Store{
		activities:   map[string]Activity{},
		applications: map[string]Application{},
		rules:        []RuleVersion{},
		logs:         []AuditLog{},
		nextAppID:    5,
	}
	for _, activity := range []Activity{
		{ID: "a1", Name: "省级大学生创新创业训练计划", Organizer: "省教育厅", Level: "省级", Category: "创新素养", Range: "6-12 分", Deadline: "2026-04-28", Source: "https://example.edu/notice/innovation", Applications: 42, Rule: "按获奖等级计入成果性评价，集体项目按位次递减。", Status: "已上架"},
		{ID: "a2", Name: "校级劳动实践示范项目", Organizer: "学生工作部", Level: "校级", Category: "劳动素养", Range: "3-8 分", Deadline: "2026-05-12", Source: "https://example.edu/notice/labor", Applications: 86, Rule: "需提交实践证明和学院认定材料，模块上限 25 分。", Status: "已上架"},
		{ID: "a3", Name: "学院文艺展演优秀节目评选", Organizer: "信息工程学院", Level: "院级", Category: "文艺素养", Range: "1-5 分", Deadline: "2026-04-27", Source: "https://example.edu/notice/art", Applications: 31, Rule: "个人项目按等级计分，集体项目需填写本人位次。", Status: "已上架"},
		{ID: "a4", Name: "校运会阳光体育竞赛", Organizer: "体育教学部", Level: "校级", Category: "身心素养", Range: "2-8 分", Deadline: "2026-06-01", Source: "https://example.edu/notice/sports", Applications: 57, Rule: "比赛成绩需可在官方成绩册中核验。", Status: "已上架"},
	} {
		store.activities[activity.ID] = activity
	}
	for _, app := range []Application{
		{ID: "s1", Student: "李明", StudentID: "20230001", ClassName: "软件 2301", ActivityID: "a1", Activity: "省级大学生创新创业训练计划", Category: "创新素养", AwardLevel: "一等奖", ProjectType: "个人项目", Rank: 1, Score: 10, Status: StatusPublicity, SubmittedAt: "2026-04-22", PublicityUntil: now.Add(32 * time.Hour).Format(time.RFC3339), ReviewerOpinion: "材料齐全，建议通过。"},
		{ID: "s2", Student: "陈欣", StudentID: "20230002", ClassName: "软件 2301", ActivityID: "a2", Activity: "校级劳动实践示范项目", Category: "劳动素养", AwardLevel: "二等奖", ProjectType: "个人项目", Rank: 1, Score: 6, Status: StatusPendingReviewer, SubmittedAt: "2026-04-24"},
		{ID: "s3", Student: "王悦", StudentID: "20230003", ClassName: "软件 2301", ActivityID: "a3", Activity: "学院文艺展演优秀节目评选", Category: "文艺素养", AwardLevel: "一等奖", ProjectType: "集体项目", Rank: 2, Score: 4, Status: StatusPendingAdmin, SubmittedAt: "2026-04-20", ReviewerOpinion: "已核验官网公示。"},
		{ID: "s4", Student: "赵然", StudentID: "20230004", ClassName: "软件 2302", ActivityID: "a4", Activity: "校运会阳光体育竞赛", Category: "身心素养", AwardLevel: "二等奖", ProjectType: "个人项目", Rank: 1, Score: 5, Status: StatusApproved, SubmittedAt: "2026-04-10", ReviewerOpinion: "通过。", AdminOpinion: "计分入账。"},
		{ID: "s5", Student: "李明", StudentID: "20230001", ClassName: "软件 2301", Activity: "学院志愿服务活动补录", Category: "劳动素养", Score: 0, Status: StatusRejected, SubmittedAt: "2026-04-18", ReviewerOpinion: "缺少主办方盖章证明。"},
	} {
		app.CreatedAt = now
		app.UpdatedAt = now
		store.applications[app.ID] = app
	}
	store.rules = append(store.rules, RuleVersion{
		ID:          "2026-v1",
		Name:        "2026 V1",
		Active:      true,
		PublishedAt: "2026-04-01",
		Matrix: map[string]float64{
			"国家级:一等奖": 16, "国家级:二等奖": 12, "国家级:三等奖": 8, "国家级:其他名次": 5,
			"省级:一等奖": 12, "省级:二等奖": 10, "省级:三等奖": 7, "省级:其他名次": 4,
			"校级:一等奖": 8, "校级:二等奖": 6, "校级:三等奖": 4, "校级:其他名次": 2,
			"院级:一等奖": 5, "院级:二等奖": 4, "院级:三等奖": 3, "院级:其他名次": 1,
		},
		ModuleMax: map[string]float64{"身心素养": 25, "文艺素养": 25, "劳动素养": 25, "创新素养": 25},
	})
	store.addLogLocked("系统", "初始化数据", "演示环境", "加载活动、申请、规则样例")
	return store
}

func (s *Store) checkDuplicateLocked(studentID, activityID string) error {
	for _, app := range s.applications {
		if app.StudentID == studentID && app.ActivityID == activityID && app.Status != StatusRejected {
			return errors.New("duplicate active application")
		}
	}
	return nil
}

func (s *Store) calculateStudentScoreLocked(studentID string) StudentScore {
	modules := map[string]float64{"身心素养": 0, "文艺素养": 0, "劳动素养": 0, "创新素养": 0}
	student := StudentScore{StudentID: studentID, AcademicScore: 86.4, Locked: s.summaryPhase}
	for _, app := range s.applications {
		if app.StudentID != studentID {
			continue
		}
		if student.Student == "" {
			student.Student = app.Student
			student.ClassName = app.ClassName
		}
		if app.Status == StatusApproved || app.Status == StatusPublicity {
			modules[app.Category] += app.Score
		}
	}
	maxMap := map[string]float64{"身心素养": 25, "文艺素养": 25, "劳动素养": 25, "创新素养": 25}
	if len(s.rules) > 0 {
		maxMap = s.rules[len(s.rules)-1].ModuleMax
	}
	for _, name := range []string{"身心素养", "文艺素养", "劳动素养", "创新素养"} {
		max := maxMap[name]
		score := modules[name]
		if max > 0 && score > max {
			score = max
		}
		student.Modules = append(student.Modules, ScoreModule{Name: name, Score: score, Max: max})
		student.QualityScore += score
	}
	student.AssessmentScore = student.AcademicScore*0.9 + student.QualityScore*0.1
	return student
}

func (s *Store) addLogLocked(actor, action, target, note string) {
	s.nextLogID++
	s.logs = append(s.logs, AuditLog{
		ID:        fmt.Sprintf("log-%d", s.nextLogID),
		Actor:     actor,
		Action:    action,
		Target:    target,
		Note:      note,
		CreatedAt: time.Now(),
	})
}

func calculateScore(level, awardLevel, projectType string, rank int) float64 {
	matrix := map[string]float64{
		"国家级:一等奖": 16, "国家级:二等奖": 12, "国家级:三等奖": 8, "国家级:其他名次": 5,
		"省级:一等奖": 12, "省级:二等奖": 10, "省级:三等奖": 7, "省级:其他名次": 4,
		"校级:一等奖": 8, "校级:二等奖": 6, "校级:三等奖": 4, "校级:其他名次": 2,
		"院级:一等奖": 5, "院级:二等奖": 4, "院级:三等奖": 3, "院级:其他名次": 1,
	}
	score := matrix[level+":"+awardLevel]
	if projectType == "集体项目" && rank > 1 {
		score *= 1 - float64(rank-1)*0.2
		if score < matrix[level+":"+awardLevel]*0.3 {
			score = matrix[level+":"+awardLevel] * 0.3
		}
	}
	return score
}

func mergeActivity(current, patch Activity) Activity {
	if patch.Name != "" {
		current.Name = patch.Name
	}
	if patch.Organizer != "" {
		current.Organizer = patch.Organizer
	}
	if patch.Level != "" {
		current.Level = patch.Level
	}
	if patch.Category != "" {
		current.Category = patch.Category
	}
	if patch.Range != "" {
		current.Range = patch.Range
	}
	if patch.Deadline != "" {
		current.Deadline = patch.Deadline
	}
	if patch.Source != "" {
		current.Source = patch.Source
	}
	if patch.Rule != "" {
		current.Rule = patch.Rule
	}
	if patch.Status != "" {
		current.Status = patch.Status
	}
	return current
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func defaultText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func requireReason(value string) string {
	return strings.TrimSpace(value)
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 1, 64)
}
