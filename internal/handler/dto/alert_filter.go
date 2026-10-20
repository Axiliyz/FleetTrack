package dto

import (
	"fleettrack/internal/model"
	"fleettrack/internal/validator"
	"net/url"
)

// ParseAlertFilter переводит query-параметры GET /alerts в доменный фильтр.
// organization_id из запроса не читается: его всегда выставляет хендлер по токену
func ParseAlertFilter(vals url.Values) (model.AlertFilter, error) {
	var f model.AlertFilter
	var err error

	f.VehicleID, err = parseIntParam(vals, "vehicle_id")
	if err != nil {
		return f, model.ErrInvalidVehicleID
	}
	if raw := vals.Get("status"); raw != "" {
		status := model.AlertStatus(raw)
		if !validator.IsAlertStatusValid(status) {
			return f, model.ErrInvalidStatus
		}
		f.Status = &status
	}
	if raw := vals.Get("severity"); raw != "" {
		severity := model.SeverityLevel(raw)
		if !validator.IsSeverityValid(severity) {
			return f, model.ErrInvalidSeverity
		}
		f.Severity = &severity
	}
	f.CreatedAt, err = parseTimeParam(vals, "created_from")
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
