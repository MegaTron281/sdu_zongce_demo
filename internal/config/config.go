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
	loadDotEnv(".env")
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

// loadDotEnv 解析 .env 键值对写入进程环境变量；已存在的真实环境变量优先，不被覆盖。
func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}
