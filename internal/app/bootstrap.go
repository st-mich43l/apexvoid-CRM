package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/runtime"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/access"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/core"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/customization"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/organization"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/users"
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
	usersModule := users.New(users.Dependencies{Pool: postgres, Transactions: database.NewTxManager(postgres), Auth: cfg.Auth})
	organizationModule := organization.New(organization.Dependencies{Pool: postgres, Transactions: database.NewTxManager(postgres), Authenticator: usersModule.Service(), Users: usersModule.Service(), Directory: usersModule.Service(), Events: framework.Events, Logger: logger})
	accessModule := access.New(access.Dependencies{Pool: postgres, Transactions: database.NewTxManager(postgres), Permissions: framework.Permissions, Users: usersModule.Service(), Authenticator: usersModule.Service()})
	contactsModule, err := contacts.New(contacts.Dependencies{Pool: postgres, Transactions: database.NewTxManager(postgres), Permissions: framework.Permissions, Access: accessModule.Service(), Workspace: organizationModule.Service(), Authenticator: usersModule.Service(), Events: framework.Events, Logger: logger, UploadDir: cfg.Contacts.UploadDir, MaxUploadBytes: cfg.Contacts.MaxUploadBytes})
	if err != nil {
		postgres.Close()
		return nil, fmt.Errorf("create contacts module: %w", err)
	}
	customizationModule := customization.New(customization.Dependencies{Pool: postgres, Transactions: database.NewTxManager(postgres), Metadata: framework.Metadata, Contacts: contactsModule.Service(), Access: accessModule.Service(), Workspace: organizationModule.Service(), Authenticator: usersModule.Service()})
	organizationModule.SetAccess(accessModule.Service())
	usersModule.SetAuthorizer(accessModule.Service())
	usersModule.SetStatusGuard(accessModule.Service())
	if err := framework.Modules.Register(core.New(core.Dependencies{Metadata: framework.Metadata})); err != nil {
		postgres.Close()
		return nil, fmt.Errorf("register built-in modules: %w", err)
	}
	if err := framework.Modules.Register(usersModule); err != nil {
		postgres.Close()
		return nil, fmt.Errorf("register users module: %w", err)
	}
	if err := framework.Modules.Register(organizationModule); err != nil {
		postgres.Close()
		return nil, fmt.Errorf("register organization module: %w", err)
	}
	if err := framework.Modules.Register(accessModule); err != nil {
		postgres.Close()
		return nil, fmt.Errorf("register access module: %w", err)
	}
	if err := framework.Modules.Register(contactsModule); err != nil {
		postgres.Close()
		return nil, fmt.Errorf("register contacts module: %w", err)
	}
	if err := framework.Modules.Register(customizationModule); err != nil {
		postgres.Close()
		return nil, fmt.Errorf("register customization module: %w", err)
	}
	if err := framework.Initialize(ctx); err != nil {
		postgres.Close()
		return nil, fmt.Errorf("initialize framework: %w", err)
	}
	migrations, err := framework.Modules.Migrations()
	if err != nil {
		postgres.Close()
		return nil, fmt.Errorf("collect migrations: %w", err)
	}
	if err := database.NewMigrationRunner(postgres, migrations).Up(ctx); err != nil && err != database.ErrNoMigrationChange {
		postgres.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}
	adminRole, err := accessModule.Service().EnsureAdministrator(ctx)
	if err != nil {
		postgres.Close()
		return nil, fmt.Errorf("ensure administrator role: %w", err)
	}
	if cfg.Bootstrap.AdminEmail != "" {
		user, created, err := usersModule.Service().BootstrapAdminWithUsername(ctx, cfg.Bootstrap.AdminEmail, cfg.Bootstrap.AdminUsername, cfg.Bootstrap.AdminPassword)
		if err != nil {
			postgres.Close()
			return nil, fmt.Errorf("bootstrap administrator user: %w", err)
		}
		if created {
			if err := accessModule.Service().ReplaceUserRoles(ctx, user.ID, []uuid.UUID{adminRole.ID}); err != nil {
				postgres.Close()
				return nil, fmt.Errorf("assign administrator role: %w", err)
			}
			logger.Info("bootstrap administrator created", "email", user.Email)
		}
	}
	application := &App{Logger: logger, Database: postgres, Transactions: database.NewTxManager(postgres), Health: health.NewChecker(postgres), Runtime: framework}
	framework.LogStartup(logger)
	logger.Info("application initialized")
	return application, nil
}
