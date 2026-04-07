package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	infrastructurepostgres "example.com/taskservice/internal/infrastructure/postgres"
	postgresrepo "example.com/taskservice/internal/repository/postgres"
	transporthttp "example.com/taskservice/internal/transport/http"
	swaggerdocs "example.com/taskservice/internal/transport/http/docs"
	httphandlers "example.com/taskservice/internal/transport/http/handlers"
	transportui "example.com/taskservice/internal/transport/http/ui"
	recurringtaskusecase "example.com/taskservice/internal/usecase/recurringtask"
	"example.com/taskservice/internal/usecase/task"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := loadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := infrastructurepostgres.Open(ctx, cfg.DatabaseDSN)
	if err != nil {
		logger.Error("open postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	taskRepo := postgresrepo.New(pool)
	recurringTaskRepo := postgresrepo.NewRecurringTaskRepository(pool)
	taskUsecase := task.NewService(taskRepo)
	recurringTaskUsecase := recurringtaskusecase.NewService(recurringTaskRepo, taskRepo)
	taskHandler := httphandlers.NewTaskHandler(taskUsecase)
	recurringTaskHandler := httphandlers.NewRecurringTaskHandler(recurringTaskUsecase)
	docsHandler := swaggerdocs.NewHandler()
	uiHandler := transportui.NewHandler()
	router := transporthttp.NewRouter(taskHandler, recurringTaskHandler, docsHandler, uiHandler)

	if err := recurringTaskUsecase.RunOnce(ctx); err != nil {
		logger.Error("generate recurring tasks on startup", "error", err)
		os.Exit(1)
	}

	go runRecurringGeneration(ctx, logger, recurringTaskUsecase)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown http server", "error", err)
		}
	}()

	logger.Info("http server started", "addr", cfg.HTTPAddr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("listen and serve", "error", err)
		os.Exit(1)
	}
}

type config struct {
	HTTPAddr    string
	DatabaseDSN string
}

func loadConfig() config {
	cfg := config{
		HTTPAddr:    envOrDefault("HTTP_ADDR", ":8080"),
		DatabaseDSN: envOrDefault("DATABASE_DSN", "postgres://postgres:postgres@localhost:5432/taskservice?sslmode=disable"),
	}

	if cfg.DatabaseDSN == "" {
		panic(fmt.Errorf("DATABASE_DSN is required"))
	}

	return cfg
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func runRecurringGeneration(ctx context.Context, logger *slog.Logger, recurringTaskUsecase recurringtaskusecase.Usecase) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := recurringTaskUsecase.RunOnce(context.Background()); err != nil {
				logger.Error("generate recurring tasks", "error", err)
			}
		}
	}
}
