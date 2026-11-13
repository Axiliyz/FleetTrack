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
	// make test экспортирует .env, поэтому опциональные переменные явно обнуляем
	for _, k := range []string{
		"METRICS_PORT", "DB_MAX_CONNS", "REQUEST_TIMEOUT", "SMTP_PORT", "LOG_LEVEL",
		"ALERT_WORKERS", "ALERT_QUEUE_SIZE", "TELEMETRY_BATCH_SIZE", "TELEMETRY_BATCH_WAIT", "TELEMETRY_BUFFER_SIZE",
		"LOG_FORMAT", "SWAGGER_ENABLED", "STREAMS_PER_USER", "TELEMETRY_MAX_CLOCK_AHEAD",
	} {
		t.Setenv(k, "")
	}
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
	if cfg.Workers.TelemetryMaxClockAhead != 2*time.Minute {
		t.Errorf("clock ahead default = %v, want 2m: дни вперёд отравляют расчёт скорости", cfg.Workers.TelemetryMaxClockAhead)
	}
	if cfg.Log.Format != logger.FormatJSON || cfg.API.SwaggerEnabled || cfg.API.StreamsPerUser != 5 {
		t.Errorf("format=%q swagger=%v streams=%d", cfg.Log.Format, cfg.API.SwaggerEnabled, cfg.API.StreamsPerUser)
	}
}

func TestLoad_ReportsEveryInvalidVariable(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DB_HOST", "")
	t.Setenv("SMTP_PORT", "abc")
	t.Setenv("LOG_LEVEL", "loud")
	t.Setenv("JWT_ACCESS_TTL", "-1")
	t.Setenv("SWAGGER_ENABLED", "maybe")
	t.Setenv("LOG_FORMAT", "xml")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
	for _, name := range []string{"DB_HOST", "SMTP_PORT", "LOG_LEVEL", "JWT_ACCESS_TTL", "SWAGGER_ENABLED", "LOG_FORMAT"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error does not mention %s: %v", name, err)
		}
	}
	if !errors.Is(err, model.ErrMissingDBVars) || !errors.Is(err, ErrInvalidConfig) || !errors.Is(err, model.ErrMissingJWTVars) {
		t.Errorf("error does not wrap expected sentinels: %v", err)
	}
}
