// Package config определяет параметры подключения к БД и API
package config

import (
	"fleettrack/internal/model"
	"net/url"
	"os"
	"strconv"
	"time"
)

// RequestTimeout — таймаут, применяемый к каждому HTTP-запросу
const RequestTimeout = 5 * time.Second

// Config хранит все параметры конфигурации приложения
type Config struct {
	DB       DBConfig
	API      APIConfig
	JWT      JWTConfig
	Telegram TelegramConfig
	SMTP     SMTPConfig
}

// DBConfig хранит параметры подключения к PostgreSQL
type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
}

// APIConfig хранит параметры HTTP API
type APIConfig struct {
	Port string
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

// Load читает конфигурацию из переменных окружения
func Load() (*Config, error) {
	cfg := &Config{
		DB: DBConfig{
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			Name:     os.Getenv("DB_NAME"),
		},
		API: APIConfig{
			Port: os.Getenv("API_PORT"),
		},
	}

	if cfg.DB.User == "" {
		return nil, model.ErrMissingDBVars
	}

	accessTTL, err := parseDurationEnv(os.Getenv("JWT_ACCESS_TTL"), time.Minute)
	if err != nil {
		return nil, model.ErrMissingJWTVars
	}

	refreshTTL, err := parseDurationEnv(os.Getenv("JWT_REFRESH_TTL"), 24*time.Hour)
	if err != nil {
		return nil, model.ErrMissingJWTVars
	}

	cfg.JWT = JWTConfig{
		Secret:     os.Getenv("JWT_SECRET"),
		TTL:        time.Duration(accessTTL),
		RefreshTTL: time.Duration(refreshTTL),
	}

	if cfg.JWT.Secret == "" {
		return nil, model.ErrMissingJWTVars
	}

	smtpPort, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if smtpPort == 0 {
		smtpPort = 587
	}
	cfg.Telegram = TelegramConfig{
		BotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
	}

	cfg.SMTP = SMTPConfig{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     smtpPort,
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     os.Getenv("SMTP_FROM"),
	}

	return cfg, nil
}

func parseDurationEnv(val string, defaultUnit time.Duration) (time.Duration, error) {
	if val == "" {
		return 0, model.ErrMissingJWTVars
	}
	if d, err := time.ParseDuration(val); err == nil {
		return d, nil
	}

	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, model.ErrMissingJWTVars
	}
	return time.Duration(n) * defaultUnit, nil
}
