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
	App      AppConfig      `yaml:"app"`
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Logging  LoggingConfig  `yaml:"logging"`
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
		App:      AppConfig{Name: "apexvoid-crm", Environment: "development"},
		Server:   ServerConfig{Address: ":6868", ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: time.Minute, RequestTimeout: 30 * time.Second, ShutdownTimeout: 10 * time.Second, CORSOrigins: []string{"http://localhost:8386"}},
		Database: DatabaseConfig{URL: "", MaxConns: 10, MinConns: 2, MaxConnLifetime: time.Hour, MaxConnIdleTime: 30 * time.Minute},
		Logging:  LoggingConfig{Level: "INFO"},
	}
}

func applyEnv(c *Config) {
	setString(&c.App.Name, "APP_NAME")
	setString(&c.App.Environment, "APP_ENV")
	setString(&c.Server.Address, "SERVER_ADDRESS")
	setString(&c.Database.URL, "DATABASE_URL")
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
	return nil
}
