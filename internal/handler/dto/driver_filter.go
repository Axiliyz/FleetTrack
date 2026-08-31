package dto

import (
	"fleettrack/internal/model"
	"net/url"
)

// ParseDriverFilter переводит фильтр в запросе в доменную модель фильтра водителей
func ParseDriverFilter(vals url.Values) (model.DriverFilter, error) {
	var f model.DriverFilter
	var err error
	f.OrganizationID, err = parseIntParam(vals, "organization_id")
	if err != nil {
		return f, model.ErrInvalidOrganizationID
	}
	f.Name, err = parseStringParam(vals, "name")
	if err != nil {
		return f, model.ErrInvalidDriverName
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
