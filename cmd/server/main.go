package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"livechat-server/internal/config"
	httpserver "livechat-server/internal/http"
	"livechat-server/internal/http/middleware"
	"livechat-server/internal/store"
	"livechat-server/internal/ws"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	store := store.New(pool)
	if _, err := store.EnsureDefaultRoom(ctx); err != nil {
		log.Printf("seed default room failed: %v", err)
	}

	hub := ws.NewHub(store, cfg.JWTSecret)
	go hub.Run()

	authMiddleware := middleware.AuthMiddleware{
		Secret:   cfg.JWTSecret,
		Store:    store,
		TokenTTL: cfg.TokenTTL,
	}

	router := httpserver.NewRouter(httpserver.RouterDeps{
		Store:       store,
		Auth:        authMiddleware,
		WSHub:       hub,
		UploadDir:   cfg.UploadDir,
		MaxUpload:   cfg.MaxUploadMB,
		CORSOrigins: cfg.CORSOrigins,
	})

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("livechat server listening on %s", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
