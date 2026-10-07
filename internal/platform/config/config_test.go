package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "application.yaml")
	contents := []byte("app:\n  name: from-file\n  environment: development\nserver:\n  address: :8080\ndatabase:\n  url: postgres://file\n")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_NAME", "from-env")
	t.Setenv("DATABASE_URL", "postgres://env")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.App.Name != "from-env" || cfg.Database.URL != "postgres://env" {
		t.Fatalf("environment overrides not applied: %+v", cfg)
	}
}

func TestLoadFailsWithoutRequiredInfrastructureURLs(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("expected validation error")
	}
}
