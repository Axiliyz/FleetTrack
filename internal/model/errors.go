// Package model содержит основные сущности логики
package model

import "errors"

// Определяем набор возможных ошибок
var (
	ErrInvalidMethod    = errors.New("invalid method")
	ErrInvalidDeviceID  = errors.New("invalid  deviceid")
	ErrInvalidVehicleID = errors.New("invalid vehicle id")
	ErrInvalidCoords    = errors.New("invalid coordinates")
	ErrInvalidFuel      = errors.New("invalid fuel")
	ErrInvalidTimestamp = errors.New("invalid timestamp")
	ErrDecoding         = errors.New("decoding failed")
	ErrEncoding         = errors.New("encoding failed")
	ErrInvalidJSON      = errors.New("Invalid JSON")
)
