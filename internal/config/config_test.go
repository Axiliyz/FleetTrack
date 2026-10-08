package config

import (
	"errors"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"strings"
	"testing"
	"time"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"DB_USER": "u", "DB_HOST": "h", "DB_PORT": "5432", "DB_NAME": "n",
		"API_PORT": "8080", "JWT_SECRET": "s", "JWT_ACCESS_TTL": "15", "JWT_REFRESH_TTL": "30",
	} {
		t.Setenv(k, v)
	}
}

func TestLoad_Defaults(t *testing.T) {
	setRequiredEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.JWT.TTL != 15*time.Minute || cfg.JWT.RefreshTTL != 30*24*time.Hour {
		t.Errorf("ttl = %v / %v", cfg.JWT.TTL, cfg.JWT.RefreshTTL)
	}
	if cfg.API.RequestTimeout != 5*time.Second || cfg.API.MetricsPort != "9091" {
		t.Errorf("api defaults = %+v", cfg.API)
	}
	if cfg.Log.Level != logger.InfoLevel || cfg.DB.MaxConns != 50 || cfg.SMTP.Port != 587 {
		t.Errorf("defaults: level=%v maxConns=%d smtp=%d", cfg.Log.Level, cfg.DB.MaxConns, cfg.SMTP.Port)
	}
	if cfg.Workers.TelemetryBatchWait != 20*time.Millisecond || cfg.Workers.AlertWorkers != 16 {
		t.Errorf("workers defaults = %+v", cfg.Workers)
	}
}

func TestLoad_ReportsEveryInvalidVariable(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DB_HOST", "")
	t.Setenv("SMTP_PORT", "abc")
	t.Setenv("LOG_LEVEL", "loud")
	t.Setenv("JWT_ACCESS_TTL", "-1")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
	for _, name := range []string{"DB_HOST", "SMTP_PORT", "LOG_LEVEL", "JWT_ACCESS_TTL"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error does not mention %s: %v", name, err)
		}
	}
	if !errors.Is(err, model.ErrMissingDBVars) || !errors.Is(err, ErrInvalidConfig) || !errors.Is(err, model.ErrMissingJWTVars) {
		t.Errorf("error does not wrap expected sentinels: %v", err)
	}
}
