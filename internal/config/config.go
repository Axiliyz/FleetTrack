// package config определяет параметры подключения к БД и API
package config

import (
	"fleettrack/internal/model"
	"net/url"
	"os"
)

const RequestTimeout = 5

type Config struct {
	DB  DBConfig
	API APIConfig
}

type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
}

type APIConfig struct {
	Port string
}

func (c DBConfig) DSN() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   c.Host + ":" + c.Port,
		Path:   c.Name,
	}
	return u.String()
}

func Load() (*Config, error) {
	cfg := &Config{
		DB: DBConfig{
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Host:     "postgres",
			Port:     "5432",
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
