package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"demo1/internal/config"
	"demo1/internal/service"

	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg *config.Config
	svc *service.Service
}

func NewRouter(cfg config.Config, svc *service.Service) *gin.Engine {
	server := &Server{cfg: &cfg, svc: svc}
	router := gin.New()
	router.MaxMultipartMemory = 16 << 20
	router.Use(gin.Logger(), gin.Recovery(), server.cors())

	api := router.Group("/api")
	api.GET("/health", server.health)
	api.POST("/auth/login", server.login)

	auth := api.Group("")
	auth.Use(server.authRequired())
	auth.GET("/auth/me", server.me)
	auth.POST("/auth/logout", server.logout)
	auth.POST("/auth/change-password", server.changePassword)
	auth.GET("/meta", server.meta)
	auth.PATCH("/meta", server.requireRole("admin"), server.patchMeta)
	auth.GET("/activities", server.listActivities)
	auth.POST("/activities", server.requireRole("admin"), server.createActivity)
	auth.GET("/activities/:id", server.getActivity)
	auth.PATCH("/activities/:id", server.requireRole("admin"), server.patchActivity)
	auth.GET("/applications", server.listApplications)
	auth.GET("/applications/:id/objections", server.listObjections)
	auth.POST("/applications", server.requireRole("student"), server.createApplication)
	auth.POST("/applications/:id/:action", server.applicationAction)
	auth.POST("/materials/upload", server.requireRole("student"), server.uploadMaterial)
	auth.GET("/materials/:name", server.downloadMaterial)
	auth.GET("/publicity", server.publicity)
	auth.GET("/scores/:studentId", server.score)
	auth.GET("/admin/users", server.requireRole("admin"), server.listUsers)
	auth.POST("/admin/users/import", server.requireRole("admin"), server.importUsers)
	auth.PATCH("/admin/users/:id/role", server.requireRole("admin"), server.updateUserRole)
	auth.PATCH("/admin/users/:id/classes", server.requireRole("admin"), server.updateUserClasses)
	auth.GET("/admin/export/scores.csv", server.requireRole("admin"), server.exportScores)
	auth.GET("/admin/logs", server.requireRole("admin"), server.logs)

	registerStatic(router)
	return router
}

func registerStatic(router *gin.Engine) {
	dist := filepath.Join("frontend", "dist")
	if _, err := os.Stat(filepath.Join(dist, "index.html")); err != nil {
		return
	}
	router.Static("/assets", filepath.Join(dist, "assets"))
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.File(filepath.Join(dist, "index.html"))
	})
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && (origin == s.cfg.FrontendOrigin || strings.HasPrefix(origin, "http://localhost:")) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Accept")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
