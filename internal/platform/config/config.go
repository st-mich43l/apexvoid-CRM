package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App       AppConfig       `yaml:"app"`
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Auth      AuthConfig      `yaml:"auth"`
	Contacts  ContactsConfig  `yaml:"contacts"`
	Bootstrap BootstrapConfig `yaml:"bootstrap"`
	Logging   LoggingConfig   `yaml:"logging"`
}

type AppConfig struct {
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
}

type ServerConfig struct {
	Address         string        `yaml:"address"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	RequestTimeout  time.Duration `yaml:"request_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	CORSOrigins     []string      `yaml:"cors_origins"`
}

type DatabaseConfig struct {
	URL             string        `yaml:"url"`
	MaxConns        int32         `yaml:"max_conns"`
	MinConns        int32         `yaml:"min_conns"`
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `yaml:"max_conn_idle_time"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type AuthConfig struct {
	AccessTokenTTL  time.Duration `yaml:"access_token_ttl"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl"`
	CookieSecure    bool          `yaml:"cookie_secure"`
	CookieSameSite  string        `yaml:"cookie_same_site"`
	PasswordMinLen  int           `yaml:"password_min_length"`
	PasswordMaxLen  int           `yaml:"password_max_length"`
}

type BootstrapConfig struct {
	AdminEmail    string `yaml:"admin_email"`
	AdminUsername string `yaml:"admin_username"`
	AdminPassword string `yaml:"admin_password"`
}

type ContactsConfig struct {
	UploadDir      string `yaml:"upload_dir"`
	MaxUploadBytes int64  `yaml:"max_upload_bytes"`
}

func Load(path string) (Config, error) {
	cfg := defaultConfig()
	if path != "" {
		contents, err := os.ReadFile(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
		if err == nil {
			if err := yaml.Unmarshal(contents, &cfg); err != nil {
				return Config{}, fmt.Errorf("parse config: %w", err)
			}
		}
	}
	applyEnv(&cfg)
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		App:       AppConfig{Name: "apexvoid-crm", Environment: "development"},
		Server:    ServerConfig{Address: ":6868", ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: time.Minute, RequestTimeout: 30 * time.Second, ShutdownTimeout: 10 * time.Second, CORSOrigins: []string{"http://localhost:8386"}},
		Database:  DatabaseConfig{URL: "", MaxConns: 10, MinConns: 2, MaxConnLifetime: time.Hour, MaxConnIdleTime: 30 * time.Minute},
		Auth:      AuthConfig{AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 720 * time.Hour, CookieSameSite: "lax", PasswordMinLen: 12, PasswordMaxLen: 128},
		Contacts:  ContactsConfig{UploadDir: "/var/lib/apexvoid/attachments", MaxUploadBytes: 25 * 1024 * 1024},
		Bootstrap: BootstrapConfig{AdminEmail: "admin@localhost", AdminUsername: "admin", AdminPassword: "admin"},
		Logging:   LoggingConfig{Level: "INFO"},
	}
}

func applyEnv(c *Config) {
	setString(&c.App.Name, "APP_NAME")
	setString(&c.App.Environment, "APP_ENV")
	setString(&c.Server.Address, "SERVER_ADDRESS")
	setString(&c.Database.URL, "DATABASE_URL")
	setString(&c.Contacts.UploadDir, "CONTACTS_UPLOAD_DIR")
	setInt64(&c.Contacts.MaxUploadBytes, "CONTACTS_MAX_UPLOAD_BYTES")
	setDuration(&c.Auth.AccessTokenTTL, "AUTH_ACCESS_TOKEN_TTL")
	setDuration(&c.Auth.RefreshTokenTTL, "AUTH_REFRESH_TOKEN_TTL")
	setBool(&c.Auth.CookieSecure, "AUTH_COOKIE_SECURE")
	setString(&c.Auth.CookieSameSite, "AUTH_COOKIE_SAMESITE")
	setInt(&c.Auth.PasswordMinLen, "AUTH_PASSWORD_MIN_LENGTH")
	setInt(&c.Auth.PasswordMaxLen, "AUTH_PASSWORD_MAX_LENGTH")
	setString(&c.Bootstrap.AdminEmail, "APEXVOID_BOOTSTRAP_ADMIN_EMAIL")
	setString(&c.Bootstrap.AdminPassword, "APEXVOID_BOOTSTRAP_ADMIN_PASSWORD")
	setString(&c.Logging.Level, "LOG_LEVEL")
	setDuration(&c.Server.ReadTimeout, "SERVER_READ_TIMEOUT")
	setDuration(&c.Server.WriteTimeout, "SERVER_WRITE_TIMEOUT")
	setDuration(&c.Server.IdleTimeout, "SERVER_IDLE_TIMEOUT")
	setDuration(&c.Server.RequestTimeout, "SERVER_REQUEST_TIMEOUT")
	setDuration(&c.Server.ShutdownTimeout, "SERVER_SHUTDOWN_TIMEOUT")
	setInt32(&c.Database.MaxConns, "DATABASE_MAX_CONNS")
	setInt32(&c.Database.MinConns, "DATABASE_MIN_CONNS")
}

func setString(target *string, key string) {
	if value, ok := os.LookupEnv(key); ok {
		*target = value
	}
}
func setDuration(target *time.Duration, key string) {
	if value, ok := os.LookupEnv(key); ok {
		if parsed, err := time.ParseDuration(value); err == nil {
			*target = parsed
		}
	}
}
func setInt32(target *int32, key string) {
	if value, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.ParseInt(value, 10, 32); err == nil {
			*target = int32(parsed)
		}
	}
}

func setInt(target *int, key string) {
	if value, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.Atoi(value); err == nil {
			*target = parsed
		}
	}
}

func setInt64(target *int64, key string) {
	if value, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			*target = parsed
		}
	}
}

func setBool(target *bool, key string) {
	if value, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.ParseBool(value); err == nil {
			*target = parsed
		}
	}
}

func (c Config) Validate() error {
	var missing []string
	if strings.TrimSpace(c.App.Name) == "" {
		missing = append(missing, "app.name")
	}
	if strings.TrimSpace(c.Server.Address) == "" {
		missing = append(missing, "server.address")
	}
	if strings.TrimSpace(c.Database.URL) == "" {
		missing = append(missing, "database.url (or DATABASE_URL)")
	}
	if len(missing) > 0 {
		return fmt.Errorf("invalid configuration: missing %s", strings.Join(missing, ", "))
	}
	if c.Database.MinConns < 0 || c.Database.MaxConns < 1 || c.Database.MinConns > c.Database.MaxConns {
		return fmt.Errorf("invalid configuration: database pool bounds")
	}
	if c.Auth.AccessTokenTTL <= 0 || c.Auth.RefreshTokenTTL <= c.Auth.AccessTokenTTL {
		return fmt.Errorf("invalid configuration: auth token lifetimes")
	}
	if c.Auth.PasswordMinLen < 12 || c.Auth.PasswordMaxLen < c.Auth.PasswordMinLen {
		return fmt.Errorf("invalid configuration: password policy")
	}
	if c.Auth.CookieSameSite != "lax" && c.Auth.CookieSameSite != "strict" && c.Auth.CookieSameSite != "none" {
		return fmt.Errorf("invalid configuration: auth cookie same-site must be lax, strict, or none")
	}
	if strings.TrimSpace(c.Contacts.UploadDir) == "" || c.Contacts.MaxUploadBytes <= 0 {
		return fmt.Errorf("invalid configuration: contacts attachment storage")
	}
	return nil
}
