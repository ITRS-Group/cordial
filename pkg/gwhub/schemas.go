package gwhub

import "time"

// ISO8601 durations used by GW Hub
const (
	PT1M  = 1 * time.Minute
	PT5M  = 5 * time.Minute
	PT15M = 15 * time.Minute
	PT1H  = 1 * time.Hour
	PT3H  = 3 * time.Hour
	PT12H = 12 * time.Hour
	P1D   = 24 * time.Hour
	P1W   = 7 * 24 * time.Hour
	//
	// A month is not possible, as it is calendar based. Instead use time.AddDate(0, -1, 0) and similar.
	//
	// P1M = ...
	//
)

// Time Windows
const (
	Monday    = "monday"
	Tuesday   = "tuesday"
	Wednesday = "wednesday"
	// ...
)
