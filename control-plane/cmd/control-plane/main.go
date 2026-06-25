package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/samaymehar/nolan/control-plane/internal/api"
	"github.com/samaymehar/nolan/control-plane/internal/config"
	"github.com/samaymehar/nolan/control-plane/internal/db"
	"github.com/samaymehar/nolan/control-plane/internal/queue"
	"github.com/samaymehar/nolan/control-plane/internal/reaper"
	"github.com/samaymehar/nolan/control-plane/internal/storage"
	"github.com/valkey-io/valkey-go"
)

func main() {
	// 1. Initialize logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 2. Load Config
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// 3. Connect to Valkey
	valkeyClient, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{cfg.ValkeyAddr},
		Password:    cfg.ValkeyPassword,
		Username:    cfg.ValkeyUser,
	})
	if err != nil {
		logger.Error("Failed to connect to Valkey", "error", err)
		os.Exit(1)
	}
	defer valkeyClient.Close()

	// 4. Connect to Postgres
	dbPool, err := pgxpool.New(context.Background(), cfg.PostgresDSN)
	if err != nil {
		logger.Error("Failed to connect to Postgres", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// 5. Connect to MinIO
	minioClient, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: cfg.MinIOUseSSL,
	})
	if err != nil {
		logger.Error("Failed to connect to MinIO", "error", err)
		os.Exit(1)
	}

	// 6. Initialize components
	repo := db.NewRepository(dbPool)
	producer := queue.NewProducer(valkeyClient)
	storageClient, err := storage.NewStorage(minioClient)
	if err != nil {
		logger.Error("Failed to initialize storage", "error", err)
		os.Exit(1)
	}

	// Initialize buckets
	if err := storageClient.EnsureBuckets(context.Background()); err != nil {
		logger.Error("Failed to ensure buckets", "error", err)
		os.Exit(1)
	}

	handler := api.NewHandler(storageClient, producer, repo, cfg)

	// 7. Start Reaper
	reaperObj := reaper.NewReaper(valkeyClient, repo, cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go reaperObj.Run(ctx)

	// 8. Register HTTP routes
	mux := http.NewServeMux()
	api.RegisterRoutes(mux, handler)

	// Wrap mux with middleware
	serverHandler := api.LoggingMiddleware(logger)(
		api.RecoveryMiddleware(logger)(
			api.RequestIDMiddleware(mux),
		),
	)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: serverHandler,
	}

	// 9. Start server & handle graceful shutdown
	go func() {
		logger.Info("Starting server", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	cancel() // Stop reaper

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(ctxShutdown); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exiting")
}
