package dto

import (
	"fleettrack/internal/model"
	"net/url"
	"strconv"
	"time"
)

// parseStringParam возвращает nil, если параметр key не передан в query.
func parseStringParam(q url.Values, key string) (*string, error) {
	raw := q.Get(key)
	if raw == "" {
		return nil, nil
	}
	return &raw, nil
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
