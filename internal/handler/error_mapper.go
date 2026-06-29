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
			Status:  http.StatusNotFound,
		}

	case errors.Is(err, model.ErrInvalidVehicleID):
		return HTTPError{
			Message: "invalid vehicle id",
			Status:  http.StatusNotFound,
		}

	case errors.Is(err, model.ErrInvalidJSON):
		return HTTPError{
			Message: "invalid json",
			Status:  http.StatusBadRequest,
		}

	default:
		return HTTPError{
			Message: "unknown error",
			Status:  http.StatusInternalServerError,
		}
	}
}
