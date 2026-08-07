package model

import (
	"errors"
)

// PositionData validation errors.
var (
	ErrEmptyTerminalID   = errors.New("terminal_id is empty")
	ErrEmptyPersonID     = errors.New("person_id is empty")
	ErrInvalidCoordinate = errors.New("coordinate is NaN or Inf")
	ErrBatteryOutOfRange = errors.New("battery must be in range [0, 100]")
	ErrInvalidTimestamp  = errors.New("timestamp must be > 0")
)
