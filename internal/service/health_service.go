package service

import (
	"context"
	"fleettrack/internal/model"
	"fmt"
	"time"
)

// readinessCheckTimeout ограничивает время проверки БД в CheckReadiness
const readinessCheckTimeout = 3 * time.Second

// HealthService реализует проверки состояния приложения
type HealthService struct {
	db Pinger
}

// Pinger описывает зависимость, доступность которой можно проверить
type Pinger interface {
	// Ping возвращает ошибку, если зависимость недоступна
	Ping(ctx context.Context) error
}

// NewHealthService создаёт новый сервис проверки состояния приложения
func NewHealthService(db Pinger) *HealthService {
	return &HealthService{
		db: db,
	}
}

// CheckHealth выполняет liveness-проверку процесса
// Возвращает ошибку, только если процесс нужно перезапустить
func (s *HealthService) CheckHealth(ctx context.Context) error {
	return nil
}

// CheckReadiness проверяет готовность приложения принимать трафик
// Возвращает ошибку, если БД недоступна или не ответила вовремя
func (s *HealthService) CheckReadiness(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, readinessCheckTimeout)
	defer cancel()
	if err := s.db.Ping(checkCtx); err != nil {
		return fmt.Errorf("%w: ping postgres: %w", model.ErrServiceUnavailable, err)
	}
	return nil
}
