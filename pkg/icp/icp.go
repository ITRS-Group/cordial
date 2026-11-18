package icp

import (
	"errors"
	"slices"

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

// DefaultBaseURL is the base URL of the ICP API used by [New] unless it
// is overridden by an option.
const DefaultBaseURL = "https://icp-api.itrsgroup.com/v2.0"

// New returns a new ICP object. BaseURL defaults to [DefaultBaseURL]
// and client, if nil, to a default http.Client. Options are applied
// after the default base URL, so a [rest.BaseURLString] or
// [rest.BaseURL] option overrides it.
func New(options ...rest.Option) (icp *ICP) {
	return &ICP{
		Client: rest.NewClient(slices.Concat([]rest.Option{rest.BaseURLString(DefaultBaseURL)}, options)...),
	}
}
