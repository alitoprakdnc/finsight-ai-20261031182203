package config

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/yourorg/finSight/internal/secret"
)

// Config holds all runtime configuration for the application.
// Values are loaded from environment variables, optional config files
// and can be overridden via Viper. Secrets referenced with the `sm://`
// scheme are fetched from Google Cloud Secret Manager at startup.
type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Redis         RedisConfig         `mapstructure:"redis"`
	MinIO         MinIOConfig         `mapstructure:"minio"`
	JWT           JWTConfig           `mapstructure:"jwt"`
	OIDC          OIDCConfig          `mapstructure:"oidc"`
	TaskQueue     TaskQueueConfig     `mapstructure:"task_queue"`
	SecretManager SecretManagerConfig `mapstructure:"secret_manager"`
	Log           LogConfig           `mapstructure:"log"`
}

// ServerConfig groups HTTP server settings.
type ServerConfig struct {
	Host string `mapstructure:"host"` // default 0.0.0.0
	Port int    `mapstructure:"port"` // default 8080
	TLS  TLSConfig `mapstructure:"tls"`
}

// TLSConfig holds optional TLS configuration.
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// DatabaseConfig groups PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	Name           string `mapstructure:"name"`
	User           string `mapstructure:"user"`
	Password       string `mapstructure:"password"` // may be a secret reference
	SSLMode        string `mapstructure:"ssl_mode"` // disable, require, verify-ca, verify-full
	MaxOpenConns   int    `mapstructure:"max_open_conns"`
	MaxIdleConns   int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime string `mapstructure:"conn_max_lifetime"` // e.g. "30m"
}

// RedisConfig groups Redis connection parameters.
type RedisConfig struct {
	Addr     string `mapstructure:"addr"` // host:port
	Password string `mapstructure:"password"` // may be a secret reference
	DB       int    `mapstructure:"db"`
}

// MinIOConfig groups S3‑compatible storage parameters.
type MinIOConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"` // may be a secret reference
	SecretKey string `mapstructure:"secret_key"` // may be a secret reference
	UseSSL    bool   `mapstructure:"use_ssl"`
	Bucket    string `mapstructure:"bucket"`
}

// JWTConfig holds JWT signing settings.
type JWTConfig struct {
	PrivateKey        string `mapstructure:"private_key"` // PEM or secret reference
	PublicKey         string `mapstructure:"public_key"`  // optional PEM
	ExpirationMinutes int    `mapstructure:"expiration_minutes"`
	Issuer            string `mapstructure:"issuer"`
}

// OIDCConfig holds OpenID Connect client configuration.
type OIDCConfig struct {
	Issuer       string `mapstructure:"issuer"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"` // may be a secret reference
}

// TaskQueueConfig selects and configures the background task system.
type TaskQueueConfig struct {
	Provider   string `mapstructure:"provider"` // "gcloud" or "bullmq"
	ProjectID  string `mapstructure:"project_id"` // for GCP
	Location   string `mapstructure:"location"`   // for GCP
	QueueName  string `mapstructure:"queue_name"` // for GCP
	RedisAddr  string `mapstructure:"redis_addr"` // optional, overrides global Redis
}

// SecretManagerConfig toggles secret manager usage.
type SecretManagerConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	ProjectID string `mapstructure:"project_id"`
}

// LogConfig groups logging preferences.
type LogConfig struct {
	Level string `mapstructure:"level"` // debug, info, warn, error
}

// Load loads configuration from environment variables, optional config files
// and resolves any secret references using Google Cloud Secret Manager.
// It returns a fully populated Config ready for consumption.
func Load(ctx context.Context) (*Config, error) {
	v := viper.New()

	// -------------------------------------------------------------------------
	// Defaults
	// -------------------------------------------------------------------------
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.tls.enabled", false)

	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.name", "finsight")
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "30m")

	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)

	v.SetDefault("minio.endpoint", "localhost:9000")
	v.SetDefault("minio.use_ssl", false)
	v.SetDefault("minio.bucket", "finsight-docs")

	v.SetDefault("jwt.expiration_minutes", 60)
	v.SetDefault("jwt.issuer", "finSight")

	v.SetDefault("task_queue.provider", "bullmq")

	v.SetDefault("log.level", "info")

	// -------------------------------------------------------------------------
	// Environment handling
	// -------------------------------------------------------------------------
	// Convert nested keys to SCREAMING_SNAKE_CASE (e.g. SERVER_PORT)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Optional config file (e.g. config.yaml) – ignored if not present.
	v.SetConfigName("config")
	v.AddConfigPath(".")
	_ = v.ReadInConfig() // ignore error – env vars are primary source

	// -------------------------------------------------------------------------
	// Unmarshal into struct
	// -------------------------------------------------------------------------
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode configuration: %w", err)
	}

	// -------------------------------------------------------------------------
	// Secret manager integration
	// -------------------------------------------------------------------------
	if cfg.SecretManager.Enabled {
		secretMgr, err := secret.NewManager(ctx, cfg.SecretManager.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("initialising secret manager: %w", err)
		}
		// Helper to resolve a possible secret reference.
		resolve := func(value string) (string, error) {
			if strings.HasPrefix(value, "sm://") {
				secretName := strings.TrimPrefix(value, "sm://")
				val, err := secretMgr.AccessSecret(ctx, secretName)
				if err != nil {
					return "", fmt.Errorf("retrieving secret %s: %w", secretName, err)
				}
				return val, nil
			}
			return value, nil
		}

		// Database password
		if cfg.Database.Password, err = resolve(cfg.Database.Password); err != nil {
			return nil, err
		}
		// Redis password
		if cfg.Redis.Password, err = resolve(cfg.Redis.Password); err != nil {
			return nil, err
		}
		// MinIO credentials
		if cfg.MinIO.AccessKey, err = resolve(cfg.MinIO.AccessKey); err != nil {
			return nil, err
		}
		if cfg.MinIO.SecretKey, err = resolve(cfg.MinIO.SecretKey); err != nil {
			return nil, err
		}
		// JWT private key
		if cfg.JWT.PrivateKey, err = resolve(cfg.JWT.PrivateKey); err != nil {
			return nil, err
		}
		// OIDC client secret
		if cfg.OIDC.ClientSecret, err = resolve(cfg.OIDC.ClientSecret); err != nil {
			return nil, err
		}
	}

	// -------------------------------------------------------------------------
	// Post‑processing & validation
	// -------------------------------------------------------------------------
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// Initialise logger according to config level (optional, but useful early)
	if err := initLogger(cfg.Log.Level); err != nil {
		// logger failure should not crash the app, just return the config
		zap.L().Warn("failed to initialise logger", zap.Error(err))
	}

	return &cfg, nil
}

// validate checks required fields and returns a descriptive error if any are missing.
func (c *Config) validate() error {
	missing := []string{}

	if c.Server.Port == 0 {
		missing = append(missing, "SERVER_PORT")
	}
	if c.Database.Host == "" {
		missing = append(missing, "DATABASE_HOST")
	}
	if c.Database.Name == "" {
		missing = append(missing, "DATABASE_NAME")
	}
	if c.Database.User == "" {
		missing = append(missing, "DATABASE_USER")
	}
	if c.Database.Password == "" {
		missing = append(missing, "DATABASE_PASSWORD")
	}
	if c.Redis.Addr == "" {
		missing = append(missing, "REDIS_ADDR")
	}
	if c.MinIO.Endpoint == "" {
		missing = append(missing, "MINIO_ENDPOINT")
	}
	if c.MinIO.Bucket == "" {
		missing = append(missing, "MINIO_BUCKET")
	}
	if c.JWT.PrivateKey == "" {
		missing = append(missing, "JWT_PRIVATE_KEY")
	}
	if c.JWT.Issuer == "" {
		missing = append(missing, "JWT_ISSUER")
	}
	if c.OIDC.Issuer == "" {
		missing = append(missing, "OIDC_ISSUER")
	}
	if c.OIDC.ClientID == "" {
		missing = append(missing, "OIDC_CLIENT_ID")
	}
	if c.TaskQueue.Provider == "" {
		missing = append(missing, "TASK_QUEUE_PROVIDER")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}
	return nil
}

// initLogger configures zap global logger based on the supplied level.
func initLogger(level string) error {
	var zapLevel zap.AtomicLevel
	switch strings.ToLower(level) {
	case "debug":
		zapLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapLevel = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = zapLevel
	logger, err := cfg.Build()
	if err != nil {
		return err
	}
	zap.ReplaceGlobals(logger)
	return nil
}

// Helper functions to expose derived values ----------------------------------------------------

// DSN returns the PostgreSQL data source name constructed from the config.
func (c *Config) DSN() string {
	ssl := c.Database.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	// Example: postgres://user:pass@host:port/dbname?sslmode=disable
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		ssl,
	)
}

// RedisOptions returns the address and password for redis clients.
func (c *Config) RedisOptions() (addr, password string, db int) {
	return c.Redis.Addr, c.Redis.Password, c.Redis.DB
}

// MinIOOptions returns the connection parameters required by the MinIO SDK.
func (c *Config) MinIOOptions() (endpoint, accessKey, secretKey string, useSSL bool, bucket string) {
	return c.MinIO.Endpoint, c.MinIO.AccessKey, c.MinIO.SecretKey, c.MinIO.UseSSL, c.MinIO.Bucket
}

// JWTExpiration returns the token expiration as a time.Duration.
func (c *Config) JWTExpiration() time.Duration {
	return time.Duration(c.JWT.ExpirationMinutes) * time.Minute
}

// TaskQueueSettings returns a struct with the selected provider and its parameters.
func (c *Config) TaskQueueSettings() TaskQueueConfig {
	if c.TaskQueue.RedisAddr != "" {
		c.TaskQueue.RedisAddr = c.TaskQueue.RedisAddr
	} else {
		c.TaskQueue.RedisAddr = c.Redis.Addr
	}
	return c.TaskQueue
}