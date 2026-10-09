package config

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "application.yaml")
	contents := []byte("app:\n  name: from-file\n  environment: development\nserver:\n  address: :6868\ndatabase:\n  url: postgres://file\n")
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

func TestLoadBuildsDatabaseURLFromDiscreteValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "application.yaml")
	if err := os.WriteFile(path, []byte("database:\n  url: postgres://ignored\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_HOST", "postgres")
	t.Setenv("DATABASE_PORT", "5432")
	t.Setenv("DATABASE_NAME", "apexvoid")
	t.Setenv("DATABASE_USER", "apexvoid")
	t.Setenv("DATABASE_PASSWORD", "pa:ss/word?#@ value")
	t.Setenv("DATABASE_SSLMODE", "disable")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(cfg.Database.URL)
	if err != nil {
		t.Fatal(err)
	}
	password, present := parsed.User.Password()
	if parsed.Host != "postgres:5432" || parsed.Path != "/apexvoid" || !present || password != "pa:ss/word?#@ value" || parsed.Query().Get("sslmode") != "disable" {
		t.Fatalf("unexpected generated database URL components: host=%q path=%q password-present=%t sslmode=%q", parsed.Host, parsed.Path, present, parsed.Query().Get("sslmode"))
	}
}

func TestLoadFailsWithoutRequiredInfrastructureURLs(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoadUsesDefaultBootstrapAdministrator(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test")
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Bootstrap.AdminUsername != "admin" || cfg.Bootstrap.AdminPassword != "admin" {
		t.Fatalf("unexpected default bootstrap administrator: %+v", cfg.Bootstrap)
	}
}

func TestValidateRejectsUnsafeProductionBootstrapDefaults(t *testing.T) {
	cfg := defaultConfig()
	cfg.Database.URL = "postgres://test"
	cfg.App.Environment = "production"
	cfg.Auth.CookieSecure = true
	cfg.Server.CORSOrigins = []string{"https://crm.example.test"}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "default bootstrap") {
		t.Fatalf("expected unsafe production bootstrap error, got %v", err)
	}
}

func TestValidateAcceptsExplicitSecureProductionConfiguration(t *testing.T) {
	cfg := defaultConfig()
	cfg.Database.URL = "postgres://test"
	cfg.App.Environment = "production"
	cfg.Auth.CookieSecure = true
	cfg.Server.CORSOrigins = []string{"https://crm.example.test"}
	cfg.Bootstrap = BootstrapConfig{AdminEmail: "administrator@example.test", AdminUsername: "platform-admin", AdminPassword: "a-very-long-bootstrap-secret"}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadReadsCommaSeparatedCORSOriginsAndBootstrapUsername(t *testing.T) {
	path := filepath.Join(t.TempDir(), "application.yaml")
	if err := os.WriteFile(path, []byte("database:\n  url: postgres://test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SERVER_CORS_ORIGINS", "https://one.example.test, https://two.example.test")
	t.Setenv("APEXVOID_BOOTSTRAP_ADMIN_USERNAME", "configured-admin")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Server.CORSOrigins) != 2 || cfg.Server.CORSOrigins[1] != "https://two.example.test" || cfg.Bootstrap.AdminUsername != "configured-admin" {
		t.Fatalf("environment configuration was not applied: %+v", cfg)
	}
}
