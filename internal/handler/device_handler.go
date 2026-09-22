package handler

import (
	"context"
	"encoding/json"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
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
	DeleteDevice(ctx context.Context, id int, organizationID *int) (model.Device, error)
}

// NewDeviceHandler создаёт новый DeviceHandler с переданными сервисом и логгером
func NewDeviceHandler(s DeviceService, l logger.Logger) *DeviceHandler {
	return &DeviceHandler{
		deviceService: s,
		logger:        l,
	}
}

// HandlePostDevice обрабатывает POST запрос на создание нового устройства
// @Summary Зарегистрировать GPS-трекер
// @Tags devices
// @Accept json
// @Produce json
// @Param request body dto.CreateDeviceRequest true "Данные устройства"
// @Success 201 {object} dto.DeviceResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Router /devices [post]
// @Security BearerAuth
func (h *DeviceHandler) HandlePostDevice(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var deviceData dto.CreateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&deviceData); err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	newDevice := deviceData.ToDomainModel()
	newDevice.OrganizationID = authCtx.OrganizationID
	device, err := h.deviceService.ProcessDevice(r.Context(), newDevice)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondCreated(w, r, "device created", h.logger, dto.NewDeviceResponse(device))
}

// HandleGetDeviceByID обрабатывает GET запрос на получение устройства по ID
// @Summary Получить трекер по ID
// @Tags devices
// @Produce json
// @Param id path int true "ID устройства"
// @Success 200 {object} dto.DeviceResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /devices/{id} [get]
// @Security BearerAuth
func (h *DeviceHandler) HandleGetDeviceByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidDeviceID)
		return
	}

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	device, err := h.deviceService.GetDeviceByID(r.Context(), id)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	if err := requireOwnOrg(authCtx, device.OrganizationID); err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "device found", h.logger, dto.NewDeviceResponse(device))
}

// HandleDeleteDeviceByID обрабатывает DELETE запрос на удаление устройства по ID
// @Summary Удалить трекер
// @Tags devices
// @Produce json
// @Param id path int true "ID устройства"
// @Success 200 {object} dto.DeviceResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /devices/{id} [delete]
// @Security BearerAuth
func (h *DeviceHandler) HandleDeleteDeviceByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidDeviceID)
		return
	}

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}
	orgScope := scopeOrganizationID(authCtx)

	device, err := h.deviceService.DeleteDevice(r.Context(), id, orgScope)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "device deleted", h.logger, dto.NewDeviceResponse(device))
}
