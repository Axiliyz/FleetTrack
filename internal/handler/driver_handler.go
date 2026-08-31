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

// DriverHandler обрабатывает HTTP запросы, связанные с водителями
type DriverHandler struct {
	driverService DriverService
	logger        logger.Logger
}

// DriverService определяет контракт бизнес-логики, необходимой DriverHandler
type DriverService interface {
	// CreateDriver валидирует и сохраняет нового водителя
	CreateDriver(ctx context.Context, d model.Driver) (model.Driver, error)
	// GetDriverByID возвращает водителя по его ID
	GetDriverByID(ctx context.Context, id int) (model.Driver, error)
	// GetDriverList возвращает список водителей по фильтру
	GetDriverList(ctx context.Context, filter model.DriverFilter) ([]model.Driver, error)
	// DeleteDriverByID удаляет водителя по его ID
	DeleteDriverByID(ctx context.Context, id int) (model.Driver, error)
	// UpdateDriverByID обновляет некоторые данные водителя по ID
	UpdateDriverByID(ctx context.Context, id int, upd model.UpdateDriver) (model.Driver, error)
}

// NewDriverHandler создаёт новый DriverHandler с переданными сервисом и логгером
func NewDriverHandler(s DriverService, l logger.Logger) *DriverHandler {
	return &DriverHandler{
		driverService: s,
		logger:        l,
	}
}

// HandlePostDriver обрабатывает POST запрос на создание нового водителя
func (h *DriverHandler) HandlePostDriver(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var driverData dto.CreateDriverRequest
	err := json.NewDecoder(r.Body).Decode(&driverData)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	driver, err := h.driverService.CreateDriver(r.Context(), driverData.ToDomainModel())
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "driver created", h.logger, dto.NewDriverResponse(driver))
}

// HandleGetDriverByID получает водителя по ID
func (h *DriverHandler) HandleGetDriverByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidDriverID)
		return
	}

	driver, err := h.driverService.GetDriverByID(r.Context(), id)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "driver found", h.logger, dto.NewDriverResponse(driver))
}

// HandleGetListDriver возвращает список водителей с фильтрами
func (h *DriverHandler) HandleGetListDriver(w http.ResponseWriter, r *http.Request) {
	filter, err := dto.ParseDriverFilter(r.URL.Query())
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	drivers, err := h.driverService.GetDriverList(r.Context(), filter)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	responses := make([]dto.DriverResponse, 0, len(drivers))
	for _, d := range drivers {
		responses = append(responses, dto.NewDriverResponse(d))
	}

	respondSuccess(w, r, "drivers list", h.logger, responses)
}

// HandleDeleteDriver обрабатывает DELETE запрос на удаление водителя по ID
func (h *DriverHandler) HandleDeleteDriver(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidDriverID)
		return
	}

	deletedDriver, err := h.driverService.DeleteDriverByID(r.Context(), id)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "driver deleted", h.logger, dto.NewDriverResponse(deletedDriver))
}

// HandlePatchDriver отвечает за изменение некоторых данных водителя
func (h *DriverHandler) HandlePatchDriver(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidDriverID)
		return
	}

	var request dto.UpdateDriverRequest
	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	changed, err := h.driverService.UpdateDriverByID(r.Context(), id, request.ToDomainModel())
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	respondSuccess(w, r, "driver was changed", h.logger, dto.NewDriverResponse(changed))
}
