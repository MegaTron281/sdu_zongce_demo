package main

import (
	"errors"
	"log"
	"os"

	"demo1/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	cfg := config.Load()
	direction := "up"
	if len(os.Args) > 1 {
		direction = os.Args[1]
	}

	m, err := migrate.New("file://migrations", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("create migrator: %v", err)
	}
	defer m.Close()

	switch direction {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	default:
		log.Fatalf("unknown migration direction %q", direction)
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("run migrations: %v", err)
	}
	if errors.Is(err, migrate.ErrNoChange) {
		log.Println("migrations: no change")
		return
	}
	log.Printf("migrations %s complete", direction)
}
