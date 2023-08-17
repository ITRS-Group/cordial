package gwhub

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
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

func New(options ...rest.Options) *Hub {
	c := rest.NewClient(options...)
	return &Hub{Client: c}
}

func (hub *Hub) Ping(ctx context.Context) (resp *http.Response, err error) {
	return hub.Get(ctx, PingEndpoint, nil, nil)
}

// ParseDuration parses an ISO 8601 string representing a duration, and
// returns the resultant golang time.Duration instance.
//
// From https://github.com/spatialtime/iso8601 but "fixed" to allow
// durations without 'T'
func ParseDuration(isoDuration string) (time.Duration, error) {
	re := regexp.MustCompile(`^P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)D)?T?(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:.\d+)?)S)?$`)
	matches := re.FindStringSubmatch(isoDuration)
	if matches == nil {
		return 0, errors.New("duration string is of incorrect format")
	}

	seconds := 0.0

	//skipping years and months

	//days
	if matches[3] != "" {
		f, err := strconv.ParseFloat(matches[3], 32)
		if err != nil {
			return 0, err
		}

		seconds += (f * 24 * 60 * 60)
	}
	//hours
	if matches[4] != "" {
		f, err := strconv.ParseFloat(matches[4], 32)
		if err != nil {
			return 0, err
		}

		seconds += (f * 60 * 60)
	}
	//minutes
	if matches[5] != "" {
		f, err := strconv.ParseFloat(matches[5], 32)
		if err != nil {
			return 0, err
		}

		seconds += (f * 60)
	}
	//seconds & milliseconds
	if matches[6] != "" {
		f, err := strconv.ParseFloat(matches[6], 32)
		if err != nil {
			return 0, err
		}

		seconds += f
	}

	goDuration := strconv.FormatFloat(seconds, 'f', -1, 32) + "s"
	return time.ParseDuration(goDuration)
}

const (
	Day  = 24 * time.Hour
	Week = 7 * Day
)

// FormatDuration converts a time.Duration to an ISO8601 duration up to
// weeks. Months and years are not supported.
func FormatDuration(d time.Duration) (iso string) {
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
		iso += fmt.Sprintf("%dH", minutes/time.Minute)
		d -= minutes
		if d == 0 {
			return
		}
	}

	iso += fmt.Sprintf("%fS", d.Seconds())
	return
}
