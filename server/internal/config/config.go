package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds the full application configuration loaded from the
// environment (and an optional .env file) via Viper.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Email    EmailConfig
	WhatsApp WhatsAppConfig
	SMS      SMSConfig
	Queue    QueueConfig
}

type ServerConfig struct {
	Host string
	Port string
}

type DatabaseConfig struct {
	URL         string
	MaxConns    int32
	MinConns    int32
	MaxConnIdle time.Duration
}

type RedisConfig struct {
	URL          string
	PoolSize     int
	MinIdleConns int
}

type JWTConfig struct {
	Secret string
	Expiry time.Duration
}

type EmailConfig struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

type WhatsAppConfig struct {
	APIURL string
	APIKey string
}

type SMSConfig struct {
	TwilioSID   string
	TwilioToken string
	TwilioFrom  string
}

type QueueConfig struct {
	Workers          int
	MaxRetryAttempts int
	RetryBackoffBase int // seconds
}

// Load reads configuration from environment variables and an optional
// .env file in the working directory. Environment variables always win.
func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("./server")

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Defaults
	v.SetDefault("HOST", "0.0.0.0")
	v.SetDefault("PORT", "8080")
	v.SetDefault("DATABASE_URL", "postgres://pushnotify:pushnotify@localhost:5432/pushnotify?sslmode=disable")
	v.SetDefault("DATABASE_MAX_CONNS", 20)
	v.SetDefault("DATABASE_MIN_CONNS", 2)
	v.SetDefault("DATABASE_MAX_CONN_IDLE", "5m")
	v.SetDefault("REDIS_URL", "redis://localhost:6379/0")
	v.SetDefault("REDIS_POOL_SIZE", 20)
	v.SetDefault("REDIS_MIN_IDLE_CONNS", 2)
	v.SetDefault("JWT_SECRET", "change-me-in-production")
	v.SetDefault("JWT_EXPIRY", "24h")
	v.SetDefault("SMTP_HOST", "")
	v.SetDefault("SMTP_PORT", 587)
	v.SetDefault("SMTP_USER", "")
	v.SetDefault("SMTP_PASS", "")
	v.SetDefault("SMTP_FROM", "PushNotify <no-reply@pushnotify.dev>")
	v.SetDefault("EVOLUTION_API_URL", "")
	v.SetDefault("EVOLUTION_API_KEY", "")
	v.SetDefault("TWILIO_SID", "")
	v.SetDefault("TWILIO_TOKEN", "")
	v.SetDefault("TWILIO_FROM", "")
	v.SetDefault("QUEUE_WORKERS", 10)
	v.SetDefault("MAX_RETRY_ATTEMPTS", 3)
	v.SetDefault("RETRY_BACKOFF_BASE", 30)

	// Reading the file is best-effort; missing file is not an error.
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only ignore "not found"; surface real parse errors.
			if !strings.Contains(err.Error(), "no such file") {
				return nil, err
			}
		}
	}

	cfg := &Config{
		Server: ServerConfig{
			Host: v.GetString("HOST"),
			Port: v.GetString("PORT"),
		},
		Database: DatabaseConfig{
			URL:         v.GetString("DATABASE_URL"),
			MaxConns:    int32(v.GetInt("DATABASE_MAX_CONNS")),
			MinConns:    int32(v.GetInt("DATABASE_MIN_CONNS")),
			MaxConnIdle: v.GetDuration("DATABASE_MAX_CONN_IDLE"),
		},
		Redis: RedisConfig{
			URL:          v.GetString("REDIS_URL"),
			PoolSize:     v.GetInt("REDIS_POOL_SIZE"),
			MinIdleConns: v.GetInt("REDIS_MIN_IDLE_CONNS"),
		},
		JWT: JWTConfig{
			Secret: v.GetString("JWT_SECRET"),
			Expiry: v.GetDuration("JWT_EXPIRY"),
		},
		Email: EmailConfig{
			Host: v.GetString("SMTP_HOST"),
			Port: v.GetInt("SMTP_PORT"),
			User: v.GetString("SMTP_USER"),
			Pass: v.GetString("SMTP_PASS"),
			From: v.GetString("SMTP_FROM"),
		},
		WhatsApp: WhatsAppConfig{
			APIURL: v.GetString("EVOLUTION_API_URL"),
			APIKey: v.GetString("EVOLUTION_API_KEY"),
		},
		SMS: SMSConfig{
			TwilioSID:   v.GetString("TWILIO_SID"),
			TwilioToken: v.GetString("TWILIO_TOKEN"),
			TwilioFrom:  v.GetString("TWILIO_FROM"),
		},
		Queue: QueueConfig{
			Workers:          v.GetInt("QUEUE_WORKERS"),
			MaxRetryAttempts: v.GetInt("MAX_RETRY_ATTEMPTS"),
			RetryBackoffBase: v.GetInt("RETRY_BACKOFF_BASE"),
		},
	}

	return cfg, nil
}
