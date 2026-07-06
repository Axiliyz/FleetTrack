// Package dto для транспортировки данных
package dto

import (
	"fleettrack/internal/model"
	"net/url"
)

// ParseVehicleFilter переводит фильтр в запросе в доменную модель фильтра
func ParseVehicleFilter(vals url.Values) (model.VehicleFilter, error) {
	var f model.VehicleFilter
	var err error
	f.OrganizationID, err = parseIntParam(vals, "organization_id")
	if err != nil {
		return f, model.ErrInvalidOrganizationID
	}
	f.VIN, err = parseStringParam(vals, "vin")
	if err != nil {
		return f, model.ErrInvalidVIN
	}
	f.NumberPlate, err = parseStringParam(vals, "number_plate")
	if err != nil {
		return f, model.ErrInvalidNumberPlate
	}
	f.Model, err = parseStringParam(vals, "model")
	if err != nil {
		return f, model.ErrInvalidModel
	}
	f.Status, err = parseVehicleStatusParam(vals, "status")
	if err != nil {
		return f, model.ErrInvalidStatus
	}
	f.CreatedFrom, err = parseTimeParam(vals, "created_from")
	if err != nil {
		return f, model.ErrInvalidTimestamp
	}
	f.CreatedTo, err = parseTimeParam(vals, "created_to")
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

// parseStringParam возвращает nil, если параметр key не передан в query.
func parseStringParam(q url.Values, key string) (*string, error) {
	raw := q.Get(key)
	if raw == "" {
		return nil, nil
	}
	return &raw, nil
}

// parseVehicleStatusParam возвращает nil, если параметр key не передан в query,
// и ошибку, если переданное значение не входит в enum VehicleStatus.
func parseVehicleStatusParam(q url.Values, key string) (*model.VehicleStatus, error) {
	raw := q.Get(key)
	if raw == "" {
		return nil, nil
	}
	status := model.VehicleStatus(raw)
	if !model.IsStatusValid(status) {
		return nil, model.ErrInvalidStatus
	}
	return &status, nil
}
