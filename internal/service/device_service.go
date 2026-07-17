package service

import (
	"context"
	"errors"
	"fleettrack/internal/database"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fleettrack/internal/repository/factory"
	"fleettrack/internal/transaction"
	"fmt"
	"strings"
)

// DeviceService реализует бизнес-логику работы с устройствами.
type DeviceService struct {
	repository  repository.DeviceRepository
	logger      logger.Logger
	txManager   transaction.TransactionManager
	repoFactory factory.RepositoryFactory
}

// NewDeviceService создаёт новый сервис устройств.
func NewDeviceService(r repository.DeviceRepository, l logger.Logger, tm transaction.TransactionManager, rf factory.RepositoryFactory) *DeviceService {
	return &DeviceService{
		repository:  r,
		logger:      l,
		txManager:   tm,
		repoFactory: rf,
	}
}

// ProcessDevice валидирует и сохраняет новое устройство.
func (s *DeviceService) ProcessDevice(ctx context.Context, d model.Device) (model.Device, error) {
	if strings.TrimSpace(d.SerialNumber) == "" {
		return model.Device{}, model.ErrInvalidSerialNumber
	}

	if err := s.repository.Create(ctx, &d); err != nil {
		return model.Device{}, err
	}

	s.logger.Info(fmt.Sprintf("Device %d created", d.ID))
	return d, nil
}

// GetDeviceByID возвращает устройство по его ID
func (s *DeviceService) GetDeviceByID(ctx context.Context, id int) (model.Device, error) {
	d, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return model.Device{}, err
	}
	return d, nil
}

// DeleteDevice удаляет устройство по ID.
// Перед удалением завершает активную связь устройства с автомобилем, если она есть —
// обе операции выполняются в одной транзакции.
func (s *DeviceService) DeleteDevice(ctx context.Context, id int) (model.Device, error) {
	var deleted model.Device

	err := s.txManager.WithTx(ctx, func(tx database.DBTX) error {
		repos := s.repoFactory.New(tx)

		if err := repos.Assignment.EndAssignment(ctx, id); err != nil && !errors.Is(err, model.ErrNotFound) {
			return err
		}

		d, err := repos.Device.Delete(ctx, id)
		if err != nil {
			return err
		}
		deleted = d
		return nil
	})

	return deleted, err
}
