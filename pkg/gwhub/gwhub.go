package gwhub

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/itrs-group/cordial/pkg/rest"
)

// Hub holds connection details for a GW Hub
type Hub struct {
	*rest.Client
}

// ErrServerError makes it a little easier for the caller to check the
// underlying HTTP response
var ErrServerError = errors.New("error from server (HTTP Status > 299)")

// New creates a Hub client using the supplied REST client options.
func New(options ...rest.Option) *Hub {
	c := rest.NewClient(options...)
	return &Hub{Client: c}
}

// Ping checks whether the GW Hub service is reachable.
func (hub *Hub) Ping(ctx context.Context) (resp *http.Response, err error) {
	return hub.Get(ctx, PingEndpoint, nil, nil)
}

// durationRE matches the ISO 8601 durations understood by
// [ParseDuration]
var durationRE = regexp.MustCompile(`^P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)W)?(?:(\d+)D)?T?(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?$`)

// ParseDuration parses an ISO 8601 string representing a duration, and
// returns the resultant golang time.Duration instance.
//
// From https://github.com/spatialtime/iso8601 but "fixed" to allow
// durations without 'T'
//
// Years and months have no fixed length, so a duration that includes
// them, such as "P1Y" or "P1M", is an error, as is a duration with no
// elements such as "P" or "PT". Note that "PT1M" is one minute.
func ParseDuration(isoDuration string) (time.Duration, error) {
	matches := durationRE.FindStringSubmatch(isoDuration)
	if matches == nil {
		return 0, errors.New("duration string is of incorrect format")
	}

	if matches[1] != "" || matches[2] != "" {
		return 0, errors.New("duration with years or months is not supported")
	}

	if !slices.ContainsFunc(matches[3:], func(s string) bool { return s != "" }) {
		return 0, errors.New("duration string has no elements")
	}

	// the seconds in a week, day, hour and minute, and one second for
	// the last element, which can have a fraction
	scales := [...]float64{7 * 24 * 60 * 60, 24 * 60 * 60, 60 * 60, 60, 1}

	seconds := 0.0
	for i, scale := range scales {
		if matches[3+i] == "" {
			continue
		}
		f, err := strconv.ParseFloat(matches[3+i], 64)
		if err != nil {
			return 0, err
		}
		seconds += f * scale
	}

	return time.ParseDuration(strconv.FormatFloat(seconds, 'f', -1, 64) + "s")
}

const (
	// Day is the duration of one day, used when converting Hub durations.
	Day = 24 * time.Hour
	// Week is the duration of one week, used when converting Hub durations.
	Week = 7 * Day
)

// FormatDuration converts a time.Duration to an ISO8601 duration up to
// weeks. Months and years are not supported.
//
// A zero duration is "PT0S". d should not be negative.
func FormatDuration(d time.Duration) (iso string) {
	if d == 0 {
		return "PT0S"
	}
	iso = "P"

	weeks := d.Truncate(Week)
	if weeks > 0 {
		iso += fmt.Sprintf("%dW", weeks/Week)
		d -= weeks
		if d == 0 {
			return
		}
	}

	days := d.Truncate(Day)
	if days > 0 {
		iso += fmt.Sprintf("%dD", days/Day)
		d -= days
		if d == 0 {
			return
		}
	}

	iso += "T"

	hours := d.Truncate(time.Hour)
	if hours > 0 {
		iso += fmt.Sprintf("%dH", hours/time.Hour)
		d -= hours
		if d == 0 {
			return
		}
	}

	minutes := d.Truncate(time.Minute)
	if minutes > 0 {
		iso += fmt.Sprintf("%dM", minutes/time.Minute)
		d -= minutes
		if d == 0 {
			return
		}
	}

	iso += fmt.Sprintf("%fS", d.Seconds())
	return
}
