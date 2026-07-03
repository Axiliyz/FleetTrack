// Package dto для транспортировки данных
package dto

import (
	"fleettrack/internal/model"
	"net/url"
	"strconv"
	"time"
)

// ParseTelemetryFilter переводит фильтр в запросе в доменную модель фильтра
func ParseTelemetryFilter(vals url.Values) (model.TelemetryFilter, error) {
	var f model.TelemetryFilter
	var err error
	f.VehicleID, err = parseIntParam(vals, "vehicle_id")
	if err != nil {
		return f, model.ErrInvalidVehicleID
	}
	f.DeviceID, err = parseIntParam(vals, "device_id")
	if err != nil {
		return f, model.ErrInvalidDeviceID
	}
	f.TripID, err = parseIntParam(vals, "trip_id")
	if err != nil {
		return f, model.ErrInvalidTripID
	}
	f.DriverID, err = parseIntParam(vals, "driver_id")
	if err != nil {
		return f, model.ErrInvalidDriverID
	}
	f.OrganizationID, err = parseIntParam(vals, "organization_id")
	if err != nil {
		return f, model.ErrInvalidOrganizationID
	}
	f.LatMin, err = parseFloat64Param(vals, "lat_min")
	if err != nil {
		return f, model.ErrInvalidCoords
	}
	f.LatMax, err = parseFloat64Param(vals, "lat_max")
	if err != nil {
		return f, model.ErrInvalidCoords
	}
	f.LonMin, err = parseFloat64Param(vals, "lon_min")
	if err != nil {
		return f, model.ErrInvalidCoords
	}
	f.LonMax, err = parseFloat64Param(vals, "lon_max")
	if err != nil {
		return f, model.ErrInvalidCoords
	}
	f.FuelMin, err = parseFloat32Param(vals, "fuel_min")
	if err != nil {
		return f, model.ErrInvalidFuel
	}
	f.FuelMax, err = parseFloat32Param(vals, "fuel_max")
	if err != nil {
		return f, model.ErrInvalidFuel
	}
	f.From, err = parseTimeParam(vals, "from")
	if err != nil {
		return f, model.ErrInvalidTimestamp
	}
	f.To, err = parseTimeParam(vals, "to")
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

// parseIntParam возвращает nil, если параметр key не передан в query.
func parseIntParam(q url.Values, key string) (*int, error) {
	raw := q.Get(key)
	if raw == "" {
		return nil, nil
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return nil, err
	}
	return &val, nil
}

// parseFloat64Param возвращает nil, если параметр key не передан в query.
func parseFloat64Param(q url.Values, key string) (*float64, error) {
	raw := q.Get(key)
	if raw == "" {
		return nil, nil
	}
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, err
	}
	return &val, nil
}

// parseFloat32Param возвращает nil, если параметр key не передан в query.
func parseFloat32Param(q url.Values, key string) (*float32, error) {
	val, err := parseFloat64Param(q, key)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil
	}
	res := float32(*val)
	return &res, nil
}

// parseLimitParam читает limit: def, если параметр пуст; ошибка, если <= 0; обрезает до max.
func parseLimitParam(q url.Values, key string, max int) (int, error) {
	const def = 100
	res, err := parseIntParam(q, key)
	if err != nil {
		return 0, err
	}
	if res == nil {
		return def, nil
	}
	if *res <= 0 {
		return 0, model.ErrInvalidLimit
	}
	if *res > max {
		return max, nil
	}
	return *res, nil
}

// parseOffsetParam читает offset: 0, если параметр пуст; ошибка, если < 0.
func parseOffsetParam(q url.Values, key string) (int, error) {
	res, err := parseIntParam(q, key)
	if err != nil {
		return 0, err
	}
	if res == nil {
		return 0, nil
	}
	if *res < 0 {
		return 0, model.ErrInvalidOffset
	}
	return *res, nil
}

// parseTimeParam возвращает nil, если параметр key не передан в query.
func parseTimeParam(q url.Values, key string) (*time.Time, error) {
	raw := q.Get(key)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
