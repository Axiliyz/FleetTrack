package dto

import (
	"fleettrack/internal/model"
	"fleettrack/internal/validator"
	"net/url"
)

// ParseUserFilter переводит фильтр в запросе в доменную модель фильтра юзеров
func ParseUserFilter(vals url.Values) (model.UserFilter, error) {
	var f model.UserFilter
	var err error

	f.OrganizationID, err = parseIntParam(vals, "organization_id")
	if err != nil {
		return f, model.ErrInvalidOrganizationID
	}

	role, err := parseStringParam(vals, "role")
	if err != nil {
		return f, model.ErrInvalidUserRole
	}
	if role != nil {
		r := model.UserRole(*role)
		if err := validator.IsRoleValid(r); err != nil {
			return f, err
		}
		f.Role = &r
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
