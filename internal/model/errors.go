package model

import "errors"

// Определяем набор возможных ошибок
var (
	ErrInvalidMethod         = errors.New("invalid method")
	ErrInvalidTelemetryID    = errors.New("invalid telemetry id")
	ErrInvalidDeviceID       = errors.New("invalid device id")
	ErrInvalidVehicleID      = errors.New("invalid vehicle id")
	ErrInvalidTripID         = errors.New("invalid trip id")
	ErrInvalidDriverID       = errors.New("invalid driver id")
	ErrInvalidOrganizationID = errors.New("invalid organization id")
	ErrInvalidCoords         = errors.New("invalid coordinates")
	ErrInvalidFuel           = errors.New("invalid fuel")
	ErrInvalidTimestamp      = errors.New("invalid timestamp")
	ErrDecoding              = errors.New("decoding failed")
	ErrEncoding              = errors.New("encoding failed")
	ErrInvalidJSON           = errors.New("invalid JSON")
	ErrNotFound              = errors.New("telemetry not found")
	ErrInvalidLimit          = errors.New("invalid limit")
	ErrInvalidOffset         = errors.New("invalid offset")
	ErrInvalidInteger        = errors.New("invalid integer(must be > 0)")
	ErrInvalidFloat          = errors.New("invalid float(must be > 0)")
	ErrMissingDBVars         = errors.New("missing required DB env vars")
	ErrConnectingDB          = errors.New("error connecting to DB")

	// Vehicle errors

	ErrInvalidVIN         = errors.New("invalid vin")
	ErrInvalidStatus      = errors.New("invalid status")
	ErrInvalidModel       = errors.New("invalid car model")
	ErrDuplicateVIN       = errors.New("vehicle with this vin already exists")
	ErrDuplicatePlate     = errors.New("vehicle with this number plate already exists")
	ErrInvalidNumberPlate = errors.New("invalid number plate")
	ErrVehicleIsBusy      = errors.New("vehicle is busy")

	// Devices errors

	ErrDeviceAlreadyAssigned = errors.New("device is already assigned")
	ErrDeviceIsBusy          = errors.New("device is active or on maintenance")
	ErrInvalidSerialNumber   = errors.New("invalid serial number")
	ErrDuplicateSerialNumber = errors.New("device with this serial number already exists")

	// Organization errors

	ErrInvalidOrgName   = errors.New("invalid organization name")
	ErrDuplicateOrgName = errors.New("organization with this name is already exists")

	// Trip errors

	ErrTripAlreadyFinished = errors.New("trip is already finished")

	// Driver errors

	ErrInvalidDriverName    = errors.New("invalid driver name")
	ErrDriverHasActiveTrips = errors.New("driver has trips and can't be deleted")

	// Calculation errors

	ErrCalculating     = errors.New("can't calculate motion")
	ErrInvalidTime     = errors.New("invalid time")
	ErrNoValue         = errors.New("no value to calculate")
	ErrNoActiveTrip    = errors.New("no trip in status RUNNING")
	ErrInvalidDistance = errors.New("invalid distance")
	ErrInvalidSpeed    = errors.New("invalid speed")
)
