// Package config определяет параметры подключения к БД и API
package config

import (
	"fleettrack/internal/model"
	"net/url"
	"os"
	"time"
)

// RequestTimeout — таймаут, применяемый к каждому HTTP-запросу.
const RequestTimeout = 5 * time.Second

// Config хранит все параметры конфигурации приложения.
type Config struct {
	DB  DBConfig
	API APIConfig
}

// DBConfig хранит параметры подключения к PostgreSQL.
type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
}

// APIConfig хранит параметры HTTP API.
type APIConfig struct {
	Port string
}

// DSN формирует строку подключения к PostgreSQL из параметров DBConfig.
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
	return cfg, nil
}
