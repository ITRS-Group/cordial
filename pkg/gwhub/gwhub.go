package gwhub

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/google/go-querystring/query"
)

// Hub holds connection details for a GW Hub
type Hub struct {
	BaseURL string
	client  *http.Client
	token   string
}

// ErrServerError makes it a little easier for the caller to check the
// underlying HTTP response
var ErrServerError = errors.New("Error from server (HTTP Status > 299)")

func New(options ...Options) *Hub {
	opts := evalOptions(options...)
	return &Hub{
		BaseURL: opts.baseURL,
		client:  opts.client,
	}
}

// Get method. On successful return the response body will be closed.
func (hub *Hub) Get(ctx context.Context, endpoint string, request interface{}, response interface{}) (resp *http.Response, err error) {
	dest, err := url.JoinPath(hub.BaseURL, endpoint)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, "GET", dest, nil)
	if err != nil {
		return
	}
	if hub.token != "" {
		req.Header.Add("Authorization", "Bearer "+hub.token)
	}
	if request != nil {
		v, err := query.Values(request)
		if err != nil {
			return resp, err
		}
		req.URL.RawQuery = v.Encode()
	}
	resp, err = hub.client.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("%w: %s", ErrServerError, string(b))
		return
	}
	defer resp.Body.Close()
	if response == nil {
		return
	}
	err = decodeResponse(resp, response)
	return
}

// Post method
func (hub *Hub) Post(ctx context.Context, endpoint string, request interface{}, response interface{}) (resp *http.Response, err error) {
	dest, err := url.JoinPath(hub.BaseURL, endpoint)
	if err != nil {
		return
	}
	j, err := json.Marshal(request)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, "POST", dest, bytes.NewReader(j))
	if err != nil {
		return
	}
	if hub.token != "" {
		req.Header.Add("Authorization", "Bearer "+hub.token)
	}
	req.Header.Add("content-type", "application/json")
	resp, err = hub.client.Do(req)
	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("%w: %s", ErrServerError, string(b))
		return
	}
	defer resp.Body.Close()
	if response == nil {
		return
	}
	err = decodeResponse(resp, response)
	return
}

func decodeResponse(resp *http.Response, response interface{}) (err error) {
	d := json.NewDecoder(resp.Body)
	rt := reflect.TypeOf(response)
	switch rt.Kind() {
	case reflect.Slice:
		rv := reflect.ValueOf(response)
		var t json.Token
		t, err = d.Token()
		if err != nil {
			return
		}
		if t != "[" {
			err = errors.New("not an array")
			return
		}
		for d.More() {
			var s interface{}
			if err = d.Decode(&s); err != nil {
				return
			}
			rv = reflect.Append(rv, reflect.ValueOf(s))
		}
		t, err = d.Token()
		if err != nil {
			return
		}
		if t != "]" {
			err = errors.New("array not terminated")
			return
		}
	default:
		err = d.Decode(&response)
	}
	return
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
