package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/st-mich43l/apexvoid-CRM/internal/app"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/config"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

func main() {
	configPath := os.Getenv("APEXVOID_CONFIG")
	if configPath == "" {
		configPath = "config/application.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("load configuration", "error", err)
		os.Exit(1)
	}
	ctx := context.Background()
	application, err := app.Bootstrap(ctx, cfg)
	if err != nil {
		slog.Error("bootstrap application", "error", err)
		os.Exit(1)
	}
	defer application.Close(context.Background())

	server, err := httpserver.New(cfg.Server, application.Logger, application.RegisterRoutes)
	if err != nil {
		application.Logger.Error("register HTTP routes", "error", err)
		os.Exit(1)
	}
	serverErrors := make(chan error, 1)
	go func() {
		application.Logger.Info("http server starting", "address", cfg.Server.Address)
		serverErrors <- server.ListenAndServe()
	}()

	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			application.Logger.Error("http server stopped", "error", err)
			os.Exit(1)
		}
	case <-shutdownCtx.Done():
		application.Logger.Info("shutdown signal received")
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			application.Logger.Error("http server shutdown", "error", err)
			os.Exit(1)
		}
	}
}
