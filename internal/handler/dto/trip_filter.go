package dto

import (
	"fleettrack/internal/model"
	"fleettrack/internal/validator"
	"net/url"
)

// ParseTripFilter переводит фильтр в запросе в доменную модель фильтра рейсов
func ParseTripFilter(vals url.Values) (model.TripFilter, error) {
	var f model.TripFilter
	var err error

	f.DriverID, err = parseIntParam(vals, "driver_id")
	if err != nil {
		return f, model.ErrInvalidDriverID
	}
	f.VehicleID, err = parseIntParam(vals, "vehicle_id")
	if err != nil {
		return f, model.ErrInvalidVehicleID
	}
	f.Status, err = parseTripStatusParam(vals, "status")
	if err != nil {
		return f, model.ErrInvalidStatus
	}
	f.StartedFrom, err = parseTimeParam(vals, "started_from")
	if err != nil {
		return f, model.ErrInvalidTimestamp
	}
	f.StartedTo, err = parseTimeParam(vals, "started_to")
	if err != nil {
		return f, model.ErrInvalidTimestamp
	}
	f.Limit, err = parseLimitParam(vals, "limit", 500)
	if err != nil {
		return f, model.ErrInvalidLimit
	}
	f.Offset, err = parseOffsetParam(vals, "offset")
	if err != nil {
		return f, model.ErrInvalidOffset
	}
	return f, nil
}

// parseTripStatusParam возвращает nil, если параметр key не передан в query,
// и ошибку, если переданное значение не входит в enum TripStatus.
func parseTripStatusParam(q url.Values, key string) (*model.TripStatus, error) {
	raw := q.Get(key)
	if raw == "" {
		return nil, nil
	}
	status := model.TripStatus(raw)
	if !validator.IsTripStatusValid(status) {
		return nil, model.ErrInvalidStatus
	}
	return &status, nil
}
