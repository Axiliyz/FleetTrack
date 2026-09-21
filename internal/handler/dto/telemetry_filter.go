package dto

import (
	"fleettrack/internal/model"
	"net/url"
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
