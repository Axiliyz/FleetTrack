package handler

import (
	"context"
	"encoding/json"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// DeviceHandler обрабатывает HTTP запросы, связанные с устройствами
type DeviceHandler struct {
	deviceService DeviceService
	logger        logger.Logger
}

// DeviceService определяет контракт бизнес-логики, необходимой DeviceHandler
type DeviceService interface {
	// ProcessDevice валидирует и сохраняет новое устройство
	ProcessDevice(ctx context.Context, d model.Device) (model.Device, error)
	// GetDeviceByID возвращает устройство по его ID
	GetDeviceByID(ctx context.Context, id int) (model.Device, error)
	// DeleteDevice удаляет устройство по его ID
	DeleteDevice(ctx context.Context, id int) (model.Device, error)
}

// NewDeviceHandler создаёт новый DeviceHandler с переданными сервисом и логгером
func NewDeviceHandler(s DeviceService, l logger.Logger) *DeviceHandler {
	return &DeviceHandler{
		deviceService: s,
		logger:        l,
	}
}

// HandlePostDevice обрабатывает POST запрос на создание нового устройства
func (h *DeviceHandler) HandlePostDevice(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var deviceData dto.CreateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&deviceData); err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	device, err := h.deviceService.ProcessDevice(r.Context(), deviceData.ToDomainModel())
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "device created", h.logger, dto.NewDeviceResponse(device))
}

// HandleGetDeviceByID обрабатывает GET запрос на получение устройства по ID
func (h *DeviceHandler) HandleGetDeviceByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidDeviceID)
		return
	}

	device, err := h.deviceService.GetDeviceByID(r.Context(), id)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "device found", h.logger, dto.NewDeviceResponse(device))
}

// HandleDeleteDeviceByID обрабатывает DELETE запрос на удаление устройства по ID
func (h *DeviceHandler) HandleDeleteDeviceByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidDeviceID)
		return
	}

	device, err := h.deviceService.DeleteDevice(r.Context(), id)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "device deleted", h.logger, dto.NewDeviceResponse(device))
}
