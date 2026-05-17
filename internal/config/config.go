package config

import (
	"os"
	"strings"
)

type Config struct {
	DatabaseURL    string
	ServerAddr     string
	SessionSecret  string
	FrontendOrigin string
	CookieSecure   bool
	UploadDir      string
}

func Load() Config {
	cfg := Config{
		DatabaseURL:    getenv("DATABASE_URL", "postgres://sdu:sdu123@localhost:5432/sdu_assessment?sslmode=disable"),
		ServerAddr:     getenv("SERVER_ADDR", ":8080"),
		SessionSecret:  getenv("SESSION_SECRET", "dev-session-secret"),
		FrontendOrigin: getenv("FRONTEND_ORIGIN", "http://localhost:5173"),
		UploadDir:      getenv("UPLOAD_DIR", "uploads/materials"),
	}
	cfg.CookieSecure = strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true")
	return cfg
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
