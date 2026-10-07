package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/st-mich43l/apexvoid-CRM/internal/framework/runtime"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/core"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/config"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

func main() {
	configPath := os.Getenv("APEXVOID_CONFIG")
	if configPath == "" {
		configPath = "config/application.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		fail(err)
	}
	ctx := context.Background()
	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		fail(err)
	}
	defer db.Close()
	framework := runtime.New()
	if err := framework.Modules.Register(core.New(core.Dependencies{Metadata: framework.Metadata})); err != nil {
		fail(err)
	}
	if err := framework.Initialize(ctx); err != nil {
		fail(err)
	}
	migrations, err := framework.Modules.Migrations()
	if err != nil {
		fail(err)
	}
	runner := database.NewMigrationRunner(db, migrations)
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	switch command {
	case "up":
		err = runner.Up(ctx)
	case "down":
		err = runner.Down(ctx)
	default:
		fail(fmt.Errorf("unknown migration command %q", command))
	}
	if errors.Is(err, database.ErrNoMigrationChange) {
		fmt.Println("no module migrations registered")
		return
	}
	if err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, "migration:", err); os.Exit(1) }
