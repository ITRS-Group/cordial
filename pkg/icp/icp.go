package icp

import (
	"errors"

	"github.com/itrs-group/cordial/pkg/rest"
)

// ICP holds the projectID, endpoint and http client for the configured
// ICP instance. If token is set it is sent with each request as a auth header.
type ICP struct {
	*rest.Client
	token string
}

// ErrServerError makes it a little easier for the caller to check the
// underlying HTTP response
var ErrServerError = errors.New("error from server (HTTP status > 299)")

// New returns a new ICP object. BaseURL defaults to
// "https://icp-api.itrsgroup.com/v2.0" and client, if nil, to a default
// http.Client
func New(options ...rest.Options) (icp *ICP) {
	return &ICP{
		Client: rest.NewClient(options...),
	}
}
