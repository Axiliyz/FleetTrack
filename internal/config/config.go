// Package config читает конфигурацию приложения из переменных окружения
package config

import (
	"errors"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

// ErrInvalidConfig возвращается, если значение переменной окружения не удалось разобрать
var ErrInvalidConfig = errors.New("invalid config")

// Config хранит все параметры конфигурации приложения
type Config struct {
	DB       DBConfig
	API      APIConfig
	JWT      JWTConfig
	Telegram TelegramConfig
	SMTP     SMTPConfig
	Log      LogConfig
	Workers  WorkersConfig
}

// DBConfig хранит параметры подключения к PostgreSQL
type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
	MaxConns int32
}

// APIConfig хранит параметры HTTP API
type APIConfig struct {
	Port           string
	MetricsPort    string
	RequestTimeout time.Duration
}

// JWTConfig хранит параметры JWT
type JWTConfig struct {
	Secret     string
	TTL        time.Duration
	RefreshTTL time.Duration
}

// TelegramConfig хранит настройки интеграции с TG Bot API
type TelegramConfig struct {
	BotToken string
}

// SMTPConfig хранит параметры почтового соединения
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// LogConfig хранит параметры логирования
type LogConfig struct {
	Level logger.Level
}

// WorkersConfig хранит параметры фоновой обработки телеметрии и алертов
type WorkersConfig struct {
	AlertWorkers        int
	AlertQueueSize      int
	TelemetryBatchSize  int
	TelemetryBatchWait  time.Duration
	TelemetryBufferSize int
}

// DSN формирует строку подключения к PostgreSQL из параметров DBConfig
func (c DBConfig) DSN() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   c.Host + ":" + c.Port,
		Path:   c.Name,
	}
	return u.String()
}

// Load читает конфигурацию из переменных окружения.
// Обязательные переменные: DB_USER, DB_HOST, DB_PORT, DB_NAME, API_PORT, JWT_SECRET,
// JWT_ACCESS_TTL, JWT_REFRESH_TTL. Остальные имеют значения по умолчанию.
func Load() (*Config, error) {
	var errs []error
	collect := func(err error) {
		if err != nil {
			errs = append(errs, err)
		}
	}

	cfg := &Config{
		DB: DBConfig{
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			Name:     os.Getenv("DB_NAME"),
		},
		API: APIConfig{
			Port:        os.Getenv("API_PORT"),
			MetricsPort: envString("METRICS_PORT", "9091"),
		},
		JWT: JWTConfig{
			Secret: os.Getenv("JWT_SECRET"),
		},
		Telegram: TelegramConfig{
			BotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		},
		SMTP: SMTPConfig{
			Host:     os.Getenv("SMTP_HOST"),
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     os.Getenv("SMTP_FROM"),
		},
	}

	for _, name := range []string{"DB_USER", "DB_HOST", "DB_PORT", "DB_NAME"} {
		if os.Getenv(name) == "" {
			collect(fmt.Errorf("%w: %s is required", model.ErrMissingDBVars, name))
		}
	}
	if cfg.JWT.Secret == "" {
		collect(fmt.Errorf("%w: JWT_SECRET is required", model.ErrMissingJWTVars))
	}
	collect(validatePort("API_PORT", cfg.API.Port))
	collect(validatePort("METRICS_PORT", cfg.API.MetricsPort))

	var err error
	cfg.JWT.TTL, err = jwtTTL("JWT_ACCESS_TTL", time.Minute)
	collect(err)
	cfg.JWT.RefreshTTL, err = jwtTTL("JWT_REFRESH_TTL", 24*time.Hour)
	collect(err)

	cfg.DB.MaxConns, err = envInt32("DB_MAX_CONNS", 50)
	collect(err)

	cfg.API.RequestTimeout, err = envDuration("REQUEST_TIMEOUT", 5*time.Second)
	collect(err)

	cfg.SMTP.Port, err = envInt("SMTP_PORT", 587)
	collect(err)

	cfg.Log.Level, err = logger.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		collect(fmt.Errorf("%w: LOG_LEVEL: %w", ErrInvalidConfig, err))
	}

	cfg.Workers.AlertWorkers, err = envInt("ALERT_WORKERS", 16)
	collect(err)
	cfg.Workers.AlertQueueSize, err = envInt("ALERT_QUEUE_SIZE", 10000)
	collect(err)
	cfg.Workers.TelemetryBatchSize, err = envInt("TELEMETRY_BATCH_SIZE", 100)
	collect(err)
	cfg.Workers.TelemetryBatchWait, err = envDuration("TELEMETRY_BATCH_WAIT", 20*time.Millisecond)
	collect(err)
	cfg.Workers.TelemetryBufferSize, err = envInt("TELEMETRY_BUFFER_SIZE", 10000)
	collect(err)

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return cfg, nil
}

func envString(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

// envInt читает положительное целое; пустая переменная даёт значение по умолчанию
func envInt(name string, def int) (int, error) {
	v := os.Getenv(name)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%w: %s must be a positive integer, got %q", ErrInvalidConfig, name, v)
	}
	return n, nil
}

// envInt32 читает положительное целое, помещающееся в int32
func envInt32(name string, def int32) (int32, error) {
	v := os.Getenv(name)
	if v == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%w: %s must be a positive 32-bit integer, got %q", ErrInvalidConfig, name, v)
	}
	return int32(n), nil
}

// envDuration читает длительность в формате time.ParseDuration (например 5s, 20ms)
func envDuration(name string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(name)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%w: %s must be a positive duration like 5s, got %q", ErrInvalidConfig, name, v)
	}
	return d, nil
}

// jwtTTL читает обязательный TTL: длительность (15m) или целое число единиц unit
func jwtTTL(name string, unit time.Duration) (time.Duration, error) {
	v := os.Getenv(name)
	if v == "" {
		return 0, fmt.Errorf("%w: %s is required", model.ErrMissingJWTVars, name)
	}
	if d, err := time.ParseDuration(v); err == nil && d > 0 {
		return d, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%w: %s must be a duration or a positive integer, got %q", model.ErrMissingJWTVars, name, v)
	}
	return time.Duration(n) * unit, nil
}

func validatePort(name, v string) error {
	if v == "" {
		return fmt.Errorf("%w: %s is required", ErrInvalidConfig, name)
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("%w: %s must be a port number, got %q", ErrInvalidConfig, name, v)
	}
	return nil
}
