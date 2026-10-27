package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"aba/internal/api"
	"aba/internal/mlclient"
	"aba/internal/pipeline"
	"aba/internal/storage"
	"aba/internal/store"
)

func main() {
	ctx := context.Background()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/data"
	}
	if u := os.Getenv("PUBLIC_URL"); u != "" {
		api.PublicURL = strings.TrimRight(u, "/")
	}
	mlURL := os.Getenv("ML_URL")
	if mlURL == "" {
		log.Fatal("ML_URL is required")
	}

	st, err := store.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	files, err := storage.New(dataDir)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}

	ml := mlclient.New(mlURL)
	pipe := pipeline.New(st, ml, dataDir)
	pipe.Start(ctx, 2)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(api.CORS())

	botUsername := os.Getenv("BOT_USERNAME")
	if botUsername == "" {
		botUsername = "poisklkbot"
	}
	api.RegisterRoutes(r, st, files, pipe, ml, jwtSecret, os.Getenv("BOT_SECRET"), botUsername)

	addr := ":8000"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("bye")
}
