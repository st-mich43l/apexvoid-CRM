package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App          AppConfig          `yaml:"app"`
	Server       ServerConfig       `yaml:"server"`
	Database     DatabaseConfig     `yaml:"database"`
	Auth         AuthConfig         `yaml:"auth"`
	Contacts     ContactsConfig     `yaml:"contacts"`
	Integrations IntegrationsConfig `yaml:"integrations"`
	Bootstrap    BootstrapConfig    `yaml:"bootstrap"`
	Logging      LoggingConfig      `yaml:"logging"`
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
	Host            string        `yaml:"-"`
	Port            string        `yaml:"-"`
	Name            string        `yaml:"-"`
	User            string        `yaml:"-"`
	Password        string        `yaml:"-"`
	SSLMode         string        `yaml:"-"`
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

// IntegrationsConfig controls trusted Docker-network applications. Production
// deployments must provide both an assertion secret and explicit service hosts.
type IntegrationsConfig struct {
	AssertionSecret     string   `yaml:"assertion_secret"`
	AllowedServiceHosts []string `yaml:"allowed_service_hosts"`
	ProvisioningURL     string   `yaml:"provisioning_url"`
	ProvisioningKey     string   `yaml:"provisioning_key"`
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
		App:          AppConfig{Name: "apexvoid-crm", Environment: "development"},
		Server:       ServerConfig{Address: ":6868", ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: time.Minute, RequestTimeout: 30 * time.Second, ShutdownTimeout: 10 * time.Second, CORSOrigins: []string{"http://localhost:8386"}},
		Database:     DatabaseConfig{URL: "", MaxConns: 10, MinConns: 2, MaxConnLifetime: time.Hour, MaxConnIdleTime: 30 * time.Minute},
		Auth:         AuthConfig{AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 720 * time.Hour, CookieSameSite: "lax", PasswordMinLen: 12, PasswordMaxLen: 128},
		Contacts:     ContactsConfig{UploadDir: "/var/lib/apexvoid/attachments", MaxUploadBytes: 25 * 1024 * 1024},
		Integrations: IntegrationsConfig{},
		Bootstrap:    BootstrapConfig{AdminEmail: "admin@localhost", AdminUsername: "admin", AdminPassword: "admin"},
		Logging:      LoggingConfig{Level: "INFO"},
	}
}

func applyEnv(c *Config) {
	setString(&c.App.Name, "APP_NAME")
	setString(&c.App.Environment, "APP_ENV")
	setString(&c.Server.Address, "SERVER_ADDRESS")
	setStringSlice(&c.Server.CORSOrigins, "SERVER_CORS_ORIGINS")
	setString(&c.Database.URL, "DATABASE_URL")
	setString(&c.Database.Host, "DATABASE_HOST")
	setString(&c.Database.Port, "DATABASE_PORT")
	setString(&c.Database.Name, "DATABASE_NAME")
	setString(&c.Database.User, "DATABASE_USER")
	setString(&c.Database.Password, "DATABASE_PASSWORD")
	setString(&c.Database.SSLMode, "DATABASE_SSLMODE")
	if c.Database.Host != "" {
		c.Database.URL = postgresURL(c.Database)
	}
	setString(&c.Contacts.UploadDir, "CONTACTS_UPLOAD_DIR")
	setInt64(&c.Contacts.MaxUploadBytes, "CONTACTS_MAX_UPLOAD_BYTES")
	setString(&c.Integrations.AssertionSecret, "INTEGRATIONS_ASSERTION_SECRET")
	setStringSlice(&c.Integrations.AllowedServiceHosts, "INTEGRATIONS_ALLOWED_SERVICE_HOSTS")
	setString(&c.Integrations.ProvisioningURL, "DATABASE_PROVISIONING_URL")
	setString(&c.Integrations.ProvisioningKey, "DATABASE_PROVISIONING_KEY")
	setDuration(&c.Auth.AccessTokenTTL, "AUTH_ACCESS_TOKEN_TTL")
	setDuration(&c.Auth.RefreshTokenTTL, "AUTH_REFRESH_TOKEN_TTL")
	setBool(&c.Auth.CookieSecure, "AUTH_COOKIE_SECURE")
	setString(&c.Auth.CookieSameSite, "AUTH_COOKIE_SAMESITE")
	setInt(&c.Auth.PasswordMinLen, "AUTH_PASSWORD_MIN_LENGTH")
	setInt(&c.Auth.PasswordMaxLen, "AUTH_PASSWORD_MAX_LENGTH")
	setString(&c.Bootstrap.AdminEmail, "APEXVOID_BOOTSTRAP_ADMIN_EMAIL")
	setString(&c.Bootstrap.AdminUsername, "APEXVOID_BOOTSTRAP_ADMIN_USERNAME")
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

// postgresURL builds a connection string from discrete runtime values. Using
// url.UserPassword prevents URI-reserved password characters from changing the
// connection semantics, while keeping the secret out of Compose interpolation.
func postgresURL(database DatabaseConfig) string {
	port := database.Port
	if port == "" {
		port = "5432"
	}
	sslMode := database.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	connection := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(database.User, database.Password),
		Host:   net.JoinHostPort(database.Host, port),
		Path:   database.Name,
	}
	query := connection.Query()
	query.Set("sslmode", sslMode)
	connection.RawQuery = query.Encode()
	return connection.String()
}

func setString(target *string, key string) {
	if value, ok := os.LookupEnv(key); ok {
		*target = value
	}
}

func setStringSlice(target *[]string, key string) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return
	}
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	*target = result
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
	if len(c.Integrations.AllowedServiceHosts) > 0 {
		if len(strings.TrimSpace(c.Integrations.AssertionSecret)) < 32 {
			return fmt.Errorf("invalid configuration: integrations.assertion_secret must be at least 32 characters when external service hosts are enabled")
		}
	}
	if strings.TrimSpace(c.Integrations.ProvisioningURL) != "" {
		if len(c.Integrations.ProvisioningKey) < 32 {
			return fmt.Errorf("invalid configuration: database provisioning requires a stable encryption key of at least 32 characters")
		}
		parsed, err := url.Parse(c.Integrations.ProvisioningURL)
		if err != nil || parsed.Scheme != "postgres" || parsed.Host == "" || parsed.User == nil {
			return fmt.Errorf("invalid configuration: integrations.provisioning_url must be a PostgreSQL URL with explicit credentials")
		}
	}
	if err := c.validateProductionSecurity(); err != nil {
		return err
	}
	return nil
}

func (c Config) validateProductionSecurity() error {
	if !strings.EqualFold(strings.TrimSpace(c.App.Environment), "production") {
		return nil
	}
	if !c.Auth.CookieSecure {
		return fmt.Errorf("invalid production configuration: AUTH_COOKIE_SECURE must be true")
	}
	if len(c.Server.CORSOrigins) == 0 {
		return fmt.Errorf("invalid production configuration: SERVER_CORS_ORIGINS must declare trusted origins")
	}
	for _, origin := range c.Server.CORSOrigins {
		if strings.TrimSpace(origin) == "" || origin == "*" {
			return fmt.Errorf("invalid production configuration: wildcard or empty CORS origins are not allowed")
		}
	}
	if strings.TrimSpace(c.Bootstrap.AdminEmail) == "" || strings.TrimSpace(c.Bootstrap.AdminUsername) == "" || strings.TrimSpace(c.Bootstrap.AdminPassword) == "" {
		return fmt.Errorf("invalid production configuration: explicit bootstrap administrator credentials are required")
	}
	if c.Bootstrap.AdminEmail == "admin@localhost" || c.Bootstrap.AdminUsername == "admin" || c.Bootstrap.AdminPassword == "admin" {
		return fmt.Errorf("invalid production configuration: default bootstrap administrator credentials are not allowed")
	}
	if len(c.Bootstrap.AdminPassword) < c.Auth.PasswordMinLen {
		return fmt.Errorf("invalid production configuration: bootstrap administrator password does not meet password policy")
	}
	return nil
}
