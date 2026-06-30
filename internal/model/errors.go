// Package model содержит основные сущности логики
package model

import "errors"

// Определяем набор возможных ошибок
var (
	ErrInvalidMethod      = errors.New("invalid method")
	ErrInvalidTelemetryID = errors.New("invalid telemetry id")
	ErrInvalidDeviceID    = errors.New("invalid  device id")
	ErrInvalidVehicleID   = errors.New("invalid vehicle id")
	ErrInvalidCoords      = errors.New("invalid coordinates")
	ErrInvalidFuel        = errors.New("invalid fuel")
	ErrInvalidTimestamp   = errors.New("invalid timestamp")
	ErrDecoding           = errors.New("decoding failed")
	ErrEncoding           = errors.New("encoding failed")
	ErrInvalidJSON        = errors.New("invalid JSON")
	ErrNotFound           = errors.New("telemetry not found")
)
