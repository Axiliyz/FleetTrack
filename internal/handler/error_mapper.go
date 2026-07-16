package handler

import (
	"errors"
	"fleettrack/internal/model"
	"net/http"
)

// HTTPError определяет структуру HTTP ответа при ошибке
type HTTPError struct {
	Message string
	Status  int
}

// mapError преобразует внутреннюю ошибку приложения в HTTP ошибку с кодом статуса
// Возвращает HTTPError
func mapError(err error) HTTPError {
	switch {
	case errors.Is(err, model.ErrDecoding):
		return HTTPError{
			Message: "error decoding",
			Status:  http.StatusInternalServerError,
		}

	case errors.Is(err, model.ErrEncoding):
		return HTTPError{
			Message: "error encoding",
			Status:  http.StatusInternalServerError,
		}

	case errors.Is(err, model.ErrInvalidCoords):
		return HTTPError{
			Message: "invalid coords",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrInvalidMethod):
		return HTTPError{
			Message: "unsupported method",
			Status:  http.StatusMethodNotAllowed,
		}

	case errors.Is(err, model.ErrInvalidFuel):
		return HTTPError{
			Message: "invalid fuel",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrInvalidTimestamp):
		return HTTPError{
			Message: "invalid timestamp",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrInvalidDeviceID):
		return HTTPError{
			Message: "invalid device id",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrInvalidVehicleID):
		return HTTPError{
			Message: "invalid vehicle id",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrInvalidJSON):
		return HTTPError{
			Message: "invalid json",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrNotFound):
		return HTTPError{
			Message: "record not found",
			Status:  http.StatusNotFound,
		}

	case errors.Is(err, model.ErrInvalidTelemetryID):
		return HTTPError{
			Message: "invalid telemetry id",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrInvalidDriverID):
		return HTTPError{
			Message: "invalid driver id",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrInvalidTripID):
		return HTTPError{
			Message: "invalid trip id",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrInvalidOrganizationID):
		return HTTPError{
			Message: "invalid organization id",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrInvalidLimit):
		return HTTPError{
			Message: "invalid limit",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrInvalidOffset):
		return HTTPError{
			Message: "invalid offset",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrInvalidInteger):
		return HTTPError{
			Message: "invalid integer(must be > 0)",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrInvalidFloat):
		return HTTPError{
			Message: "invalid float(must be > 0)",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrMissingDBVars):
		return HTTPError{
			Message: "missing required DB env vars",
			Status:  http.StatusServiceUnavailable,
		}

	case errors.Is(err, model.ErrConnectingDB):
		return HTTPError{
			Message: "error connecting to DB",
			Status:  http.StatusServiceUnavailable,
		}

	case errors.Is(err, model.ErrInvalidVIN):
		return HTTPError{
			Message: "invalid vin",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrDuplicateVIN):
		return HTTPError{
			Message: "vehicle with this vin already exists",
			Status:  http.StatusConflict,
		}

	case errors.Is(err, model.ErrDuplicatePlate):
		return HTTPError{
			Message: "vehicle with this number plate already exists",
			Status:  http.StatusConflict,
		}

	case errors.Is(err, model.ErrInvalidNumberPlate):
		return HTTPError{
			Message: "invalid number plate",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrInvalidStatus):
		return HTTPError{
			Message: "invalid vehicle status",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrInvalidModel):
		return HTTPError{
			Message: "invalid car model",
			Status:  http.StatusBadRequest,
		}

	case errors.Is(err, model.ErrDeviceAlreadyAssigned):
		return HTTPError{
			Message: "device is already assigned",
			Status:  http.StatusConflict,
		}

	case errors.Is(err, model.ErrVehicleIsBusy):
		return HTTPError{
			Message: "vehicle is busy",
			Status:  http.StatusConflict,
		}

	case errors.Is(err, model.ErrDeviceIsBusy):
		return HTTPError{
			Message: "device is active or on maintenance",
			Status:  http.StatusConflict,
		}

	default:
		return HTTPError{
			Message: "unknown error",
			Status:  http.StatusInternalServerError,
		}
	}
}
