package gwhub

import (
	"encoding/json"
	"time"
)

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

type Filter struct {
	Entities    string       `json:"entities,omitempty"`
	Range       TimeRange    `json:"range"`
	TimeWindows []TimeWindow `json:"timeWindows,omitempty"`
}

type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// MarshalJSON is needed as Hub only accepts short form ISO times
func (t TimeRange) MarshalJSON() ([]byte, error) {
	timerange := struct {
		From string `json:"from"`
		To   string `json:"to"`
	}{
		From: t.From.Format(time.RFC3339),
		To:   t.To.Format(time.RFC3339),
	}
	return json.Marshal(timerange)
}

// TimeWindow is only time-of-day
type TimeWindow struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
	On   []string  `json:"on,omitempty"`
}

func (t TimeWindow) MarshalJSON() ([]byte, error) {
	timewindow := struct {
		From string   `json:"from"`
		To   string   `json:"to"`
		On   []string `json:"on,omitempty"`
	}{
		From: t.From.Format("15:04:05"),
		To:   t.To.Format("15:04:05"),
		On:   t.On,
	}
	return json.Marshal(timewindow)
}
