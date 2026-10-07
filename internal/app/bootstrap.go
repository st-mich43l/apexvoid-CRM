package app

import (
	"context"
	"fmt"

	"github.com/st-mich43l/apexvoid-CRM/internal/framework/runtime"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/core"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/config"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/health"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/logging"
)

func Bootstrap(ctx context.Context, cfg config.Config) (*App, error) {
	logger := logging.New(cfg.App.Environment, cfg.Logging.Level)
	postgres, err := database.New(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}
	framework := runtime.New()
	if err := framework.Modules.Register(core.New(core.Dependencies{Metadata: framework.Metadata})); err != nil {
		postgres.Close()
		return nil, fmt.Errorf("register built-in modules: %w", err)
	}
	if err := framework.Initialize(ctx); err != nil {
		postgres.Close()
		return nil, fmt.Errorf("initialize framework: %w", err)
	}
	application := &App{Logger: logger, Database: postgres, Transactions: database.NewTxManager(postgres), Health: health.NewChecker(postgres), Runtime: framework}
	framework.LogStartup(logger)
	logger.Info("application initialized")
	return application, nil
}
