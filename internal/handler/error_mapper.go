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

// errorMappings связывает доменные ошибки с HTTP-статусом и сообщением для клиента.
// Порядок важен: побеждает первая ошибка, совпавшая через errors.Is.
var errorMappings = []struct {
	err     error
	status  int
	message string
}{
	{model.ErrDecoding, http.StatusInternalServerError, "error decoding"},
	{model.ErrEncoding, http.StatusInternalServerError, "error encoding"},
	{model.ErrInvalidCoords, http.StatusBadRequest, "invalid coords"},
	{model.ErrInvalidMethod, http.StatusMethodNotAllowed, "unsupported method"},
	{model.ErrInvalidFuel, http.StatusBadRequest, "invalid fuel"},
	{model.ErrInvalidTimestamp, http.StatusBadRequest, "invalid timestamp"},
	{model.ErrInvalidDeviceID, http.StatusBadRequest, "invalid device id"},
	{model.ErrInvalidVehicleID, http.StatusBadRequest, "invalid vehicle id"},
	{model.ErrInvalidJSON, http.StatusBadRequest, "invalid json"},
	{model.ErrNotFound, http.StatusNotFound, "record not found"},
	{model.ErrInvalidTelemetryID, http.StatusBadRequest, "invalid telemetry id"},
	{model.ErrInvalidDriverID, http.StatusBadRequest, "invalid driver id"},
	{model.ErrInvalidTripID, http.StatusBadRequest, "invalid trip id"},
	{model.ErrInvalidOrganizationID, http.StatusBadRequest, "invalid organization id"},
	{model.ErrInvalidLimit, http.StatusBadRequest, "invalid limit"},
	{model.ErrInvalidOffset, http.StatusBadRequest, "invalid offset"},
	{model.ErrInvalidInteger, http.StatusBadRequest, "invalid integer(must be > 0)"},
	{model.ErrInvalidFloat, http.StatusBadRequest, "invalid float(must be > 0)"},
	{model.ErrMissingDBVars, http.StatusServiceUnavailable, "missing required DB env vars"},
	{model.ErrConnectingDB, http.StatusServiceUnavailable, "error connecting to DB"},
	{model.ErrInvalidVIN, http.StatusBadRequest, "invalid vin"},
	{model.ErrDuplicateVIN, http.StatusConflict, "vehicle with this vin already exists"},
	{model.ErrDuplicatePlate, http.StatusConflict, "vehicle with this number plate already exists"},
	{model.ErrInvalidNumberPlate, http.StatusBadRequest, "invalid number plate"},
	{model.ErrInvalidStatus, http.StatusBadRequest, "invalid status"},
	{model.ErrInvalidModel, http.StatusBadRequest, "invalid car model"},
	{model.ErrDeviceAlreadyAssigned, http.StatusConflict, "device is already assigned"},
	{model.ErrVehicleIsBusy, http.StatusConflict, "vehicle is busy"},
	{model.ErrDeviceIsBusy, http.StatusConflict, "device is active or on maintenance"},
	{model.ErrDeviceNotAssigned, http.StatusConflict, "device is not assigned to this vehicle"},
	{model.ErrInvalidSerialNumber, http.StatusBadRequest, "invalid serial number"},
	{model.ErrDuplicateSerialNumber, http.StatusConflict, "device with this serial number already exists"},
	{model.ErrInvalidOrgName, http.StatusBadRequest, "invalid organization name"},
	{model.ErrTripAlreadyFinished, http.StatusConflict, "trip is already finished"},
	{model.ErrInvalidDriverName, http.StatusBadRequest, "invalid driver name"},
	{model.ErrDriverHasActiveTrips, http.StatusConflict, "driver has trips and can't be deleted"},
	{model.ErrCalculating, http.StatusBadRequest, "can't calculate motion"},
	{model.ErrInvalidTime, http.StatusBadRequest, "invalid time"},
	{model.ErrNoValue, http.StatusBadRequest, "no previous value to calculate"},
	{model.ErrNoActiveTrip, http.StatusConflict, "no active trip"},
	{model.ErrInvalidSpeed, http.StatusBadRequest, "invalid speed"},
	{model.ErrInvalidDistance, http.StatusBadRequest, "invalid distance"},
	{model.ErrDuplicateOrgName, http.StatusConflict, "organization with this name already exists"},
	{model.ErrInvalidName, http.StatusBadRequest, "invalid user name"},
	{model.ErrInvalidEmail, http.StatusBadRequest, "invalid email"},
	{model.ErrDuplicateEmail, http.StatusConflict, "user with this email already exists"},
	{model.ErrInvalidPassword, http.StatusBadRequest, "invalid password"},
	{model.ErrForbiddenPassword, http.StatusConflict, "forbidden password"},
	{model.ErrInvalidUserID, http.StatusBadRequest, "invalid user id"},
	{model.ErrInvalidUserRole, http.StatusBadRequest, "invalid user role"},
	{model.ErrInvalidToken, http.StatusUnauthorized, "invalid token"},
	{model.ErrInvalidSigningMethod, http.StatusUnauthorized, "invalid token"},
	{model.ErrMissingToken, http.StatusUnauthorized, "missing bearer token"},
	{model.ErrInvalidCredentials, http.StatusUnauthorized, "invalid credentials"},
	{model.ErrDriverAlreadyLinked, http.StatusConflict, "driver is already linked to another user"},
	{model.ErrForbidden, http.StatusForbidden, "forbidden"},
	{model.ErrDriverNotLinked, http.StatusForbidden, "user account has no linked driver"},
	{model.ErrInvalidThreshold, http.StatusBadRequest, "invalid threshold"},
	{model.ErrInvalidRuleID, http.StatusBadRequest, "invalid rule id"},
	{model.ErrDuplicateAlert, http.StatusConflict, "alert is already exist"},
	{model.ErrServiceUnavailable, http.StatusServiceUnavailable, "service is unavailable, try later"},
}

// mapError преобразует внутреннюю ошибку приложения в HTTP ошибку с кодом статуса.
// Неизвестные ошибки отдаются как 500 без деталей.
func mapError(err error) HTTPError {
	for _, m := range errorMappings {
		if errors.Is(err, m.err) {
			return HTTPError{Message: m.message, Status: m.status}
		}
	}
	return HTTPError{Message: "unknown error", Status: http.StatusInternalServerError}
}
