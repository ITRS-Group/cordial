package api

import (
	"net/http"
)

// APIClient is the method set required for any API sending data into a Geneos Netprobe
type APIClient interface {
	Healthy() bool
	CreateDataview(entity, sampler, name string) error
	UpdateDataview(entity, sampler, name string, values [][]string) error
	DeleteDataview(entity, sampler, name string) error
	CreateRow(entity, sampler, view, name string) error
	UpdateRow(entity, sampler, view, name string, values []string) error
	DeleteRow(entity, sampler, view, name string) error
	CreateStream(entity, sampler, name string) error
	UpdateStream(entity, sampler, name, message string) error

	DataviewExists(entity, sampler, name string) (bool, error)
}

type roundTripper struct {
	transport http.RoundTripper
}

func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return r.transport.RoundTrip(req)
}

// compile time check for interface validity
var _ http.RoundTripper = (*roundTripper)(nil)
