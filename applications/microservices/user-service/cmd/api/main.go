package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gorilla/mux"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "go.uber.org/zap"

    "user-service/internal/handlers"
    "user-service/internal/repository"
    "user-service/internal/services"
    "user-service/pkg/config"
    "user-service/pkg/database"
    "user-service/pkg/logger"
    "user-service/pkg/metrics"
)

var (
    version = "dev"
    commit  = "unknown"
)

func main() {
    // Initialize configuration
    cfg, err := config.Load()
    if err != nil {
        panic(fmt.Sprintf("Failed to load configuration: %v", err))
    }

    // Initialize logger
    if err := logger.Init(cfg.ServiceName, version, cfg.Environment); err != nil {
        panic(fmt.Sprintf("Failed to initialize logger: %v", err))
    }
    defer logger.Sync()

    logger.Info("Starting service",
        zap.String("version", version),
        zap.String("commit", commit),
        zap.String("environment", cfg.Environment),
    )

    // Initialize database
    db, err := database.Connect(cfg.DatabaseURL)
    if err != nil {
        logger.Fatal("Failed to connect to database", zap.Error(err))
    }
    defer db.Close()

    // Run migrations
    if err := database.Migrate(db); err != nil {
        logger.Fatal("Failed to run migrations", zap.Error(err))
    }

    // Initialize repository layer
    userRepo := repository.NewUserRepository(db)

    // Initialize service layer
    userService := services.NewUserService(userRepo)

    // Initialize handlers
    userHandler := handlers.NewUserHandler(userService)
    healthHandler := handlers.NewHealthHandler(db)

    // Setup router
    r := mux.NewRouter()

    // Health endpoints
    r.HandleFunc("/health", healthHandler.Health).Methods("GET")
    r.HandleFunc("/ready", healthHandler.Ready).Methods("GET")

    // Metrics endpoint
    r.Handle("/metrics", promhttp.Handler())

    // API v1 routes
    api := r.PathPrefix("/api/v1").Subrouter()
    api.Use(logger.HTTPMiddleware)
    api.Use(metrics.HTTPMiddleware)

    // User routes
    api.HandleFunc("/users", userHandler.List).Methods("GET")
    api.HandleFunc("/users", userHandler.Create).Methods("POST")
    api.HandleFunc("/users/{id}", userHandler.Get).Methods("GET")
    api.HandleFunc("/users/{id}", userHandler.Update).Methods("PUT")
    api.HandleFunc("/users/{id}", userHandler.Delete).Methods("DELETE")

    // Create HTTP server
    srv := &http.Server{
        Addr:         fmt.Sprintf(":%d", cfg.Port),
        Handler:      r,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Start server in goroutine
    go func() {
        logger.Info("Server starting", zap.Int("port", cfg.Port))
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("Server failed to start", zap.Error(err))
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("Server shutting down")

    // Graceful shutdown with 30 second timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        logger.Error("Server forced to shutdown", zap.Error(err))
    }

    logger.Info("Server exited")
}
