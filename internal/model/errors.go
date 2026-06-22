package model

import "errors"

var (
	ErrInvalidMethod    = errors.New("invalid method")
	ErrInvalidID        = errors.New("invalid id")
	ErrInvalidCoords    = errors.New("invalid coordinates")
	ErrInvalidFuel      = errors.New("invalid fuel")
	ErrInvalidTimestamp = errors.New("invalid timestamp")
)
