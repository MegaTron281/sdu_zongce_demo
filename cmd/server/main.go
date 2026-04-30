package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"demo1/internal/config"
	"demo1/internal/httpapi"
	"demo1/internal/repository"
	"demo1/internal/service"
)

const publicityTickInterval = time.Minute

func main() {
	cfg := config.Load()
	db, err := repository.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	svc := service.New(db, cfg.SessionSecret)

	if err := svc.AdvanceExpiredPublicity(); err != nil {
		log.Printf("publicity advance on startup: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	go runPublicityTicker(ctx, svc)

	router := httpapi.NewRouter(cfg, svc)
	server := &http.Server{Addr: cfg.ServerAddr, Handler: router}
	go func() {
		log.Printf("server listening on http://localhost%s", cfg.ServerAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown: %v", err)
	}
}

func runPublicityTicker(ctx context.Context, svc *service.Service) {
	ticker := time.NewTicker(publicityTickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := svc.AdvanceExpiredPublicity(); err != nil {
				log.Printf("publicity advance: %v", err)
			}
		}
	}
}
