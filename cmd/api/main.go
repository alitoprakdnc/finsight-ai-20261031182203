package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"finSight/internal/config"
	"finSight/internal/middleware"
	"finSight/internal/router"
	"finSight/internal/secret"
	"finSight/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// -------------------------------------------------------------------------
	// Load configuration
	// -------------------------------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// -------------------------------------------------------------------------
	// Initialise logger
	// -------------------------------------------------------------------------
	logger, err := util.NewLogger(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialise logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = logger.Sync()
	}()

	// -------------------------------------------------------------------------
	// Initialise PostgreSQL (GORM)
	// -------------------------------------------------------------------------
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		logger.Fatal("unable to connect to PostgreSQL", zap.Error(err))
	}
	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatal("unable to obtain generic DB from GORM", zap.Error(err))
	}
	// Ensure the DB connection is closed on shutdown
	defer func() {
		_ = sqlDB.Close()
	}()

	// -------------------------------------------------------------------------
	// Initialise Redis client
	// -------------------------------------------------------------------------
	redisOpts := &redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	rdb := redis.NewClient(redisOpts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		logger.Fatal("unable to connect to Redis", zap.Error(err))
	}
	defer func() {
		_ = rdb.Close()
	}()

	// -------------------------------------------------------------------------
	// Initialise Secret Manager abstraction
	// -------------------------------------------------------------------------
	secretMgr, err := secret.NewManager(cfg)
	if err != nil {
		logger.Fatal("failed to initialise secret manager", zap.Error(err))
	}

	// -------------------------------------------------------------------------
	// Setup Gin router
	// -------------------------------------------------------------------------
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()

	// Global middlewares
	engine.Use(middleware.Logger(logger))
	engine.Use(middleware.Recovery())
	engine.Use(middleware.Metrics())
	engine.Use(middleware.CORSMiddleware())

	// Register all routes (health, auth, documents, webhooks, etc.)
	if err := router.SetupRoutes(engine, cfg, db, rdb, secretMgr, logger); err != nil {
		logger.Fatal("failed to register routes", zap.Error(err))
	}

	// -------------------------------------------------------------------------
	// HTTP server with graceful shutdown
	// -------------------------------------------------------------------------
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: engine,
	}

	// Run server in a goroutine
	go func() {
		logger.Info("starting HTTP server", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("unexpected server error", zap.Error(err))
		}
	}()

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutdown signal received")

	// Context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server exiting")
}