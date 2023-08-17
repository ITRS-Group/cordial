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

// // Post makes a POST request to the endpoint after marshalling the body
// // as json. The caller must close resp.Body.
// //
// // TODO: detect if response is a slice or map, then stream decode
// func (icp *ICP) Post(ctx context.Context, endpoint string, request interface{}, response interface{}) (resp *http.Response, err error) {
// 	dest, err := url.JoinPath(icp.BaseURL, endpoint)
// 	if err != nil {
// 		return
// 	}
// 	// TODO: look at streaming large requests
// 	j, err := json.Marshal(request)
// 	if err != nil {
// 		return
// 	}
// 	req, err := http.NewRequestWithContext(ctx, "POST", dest, bytes.NewReader(j))
// 	if err != nil {
// 		return
// 	}
// 	if icp.token != "" {
// 		req.Header.Add("Authorization", "SUMERIAN "+icp.token)
// 	}
// 	req.Header.Add("content-type", "application/json")
// 	resp, err = icp.client.Do(req)
// 	if resp.StatusCode > 299 {
// 		b, _ := io.ReadAll(resp.Body)
// 		err = fmt.Errorf("%w: %s", ErrServerError, string(b))
// 		return
// 	}
// 	defer resp.Body.Close()
// 	if response == nil {
// 		return
// 	}
// 	err = decodeResponse(resp, response)
// 	return
// }

// // Get sends a GET request to the endpoint
// //
// // TODO: detect if response is a slice or map, then stream decode
// func (icp *ICP) Get(ctx context.Context, endpoint string, request interface{}, response interface{}) (resp *http.Response, err error) {
// 	dest, err := url.JoinPath(icp.BaseURL, endpoint)
// 	if err != nil {
// 		return
// 	}
// 	req, err := http.NewRequestWithContext(ctx, "GET", dest, nil)
// 	if err != nil {
// 		return
// 	}
// 	if icp.token != "" {
// 		req.Header.Add("Authorization", "SUMERIAN "+icp.token)
// 	}
// 	if request != nil {
// 		v, err := query.Values(request)
// 		if err != nil {
// 			return resp, err
// 		}
// 		req.URL.RawQuery = v.Encode()
// 	}
// 	resp, err = icp.client.Do(req)
// 	if err != nil {
// 		return
// 	}
// 	if resp.StatusCode > 299 {
// 		b, _ := io.ReadAll(resp.Body)
// 		err = fmt.Errorf("%w: %s", ErrServerError, string(b))
// 		return
// 	}
// 	defer resp.Body.Close()
// 	if response == nil {
// 		return
// 	}
// 	err = decodeResponse(resp, response)
// 	return
// }

// func decodeResponse(resp *http.Response, response interface{}) (err error) {
// 	d := json.NewDecoder(resp.Body)
// 	rt := reflect.TypeOf(response)
// 	switch rt.Kind() {
// 	case reflect.Slice:
// 		rv := reflect.ValueOf(response)
// 		var t json.Token
// 		t, err = d.Token()
// 		if err != nil {
// 			return
// 		}
// 		if t != "[" {
// 			err = errors.New("not an array")
// 			return
// 		}
// 		for d.More() {
// 			var s interface{}
// 			if err = d.Decode(&s); err != nil {
// 				return
// 			}
// 			rv = reflect.Append(rv, reflect.ValueOf(s))
// 		}
// 		t, err = d.Token()
// 		if err != nil {
// 			return
// 		}
// 		if t != "]" {
// 			err = errors.New("array not terminated")
// 			return
// 		}
// 	default:
// 		err = d.Decode(&response)
// 	}
// 	return
// }

// // Delete Method
// func (icp *ICP) Delete(ctx context.Context, endpoint string, request interface{}) (resp *http.Response, err error) {
// 	dest, err := url.JoinPath(icp.BaseURL, endpoint)
// 	if err != nil {
// 		return
// 	}
// 	req, err := http.NewRequestWithContext(ctx, "DELETE", dest, nil)
// 	if err != nil {
// 		return
// 	}
// 	if icp.token != "" {
// 		req.Header.Add("Authorization", "SUMERIAN "+icp.token)
// 	}
// 	if request != nil {
// 		v, err := query.Values(request)
// 		if err != nil {
// 			return resp, err
// 		}
// 		req.URL.RawQuery = v.Encode()
// 	}
// 	resp, err = icp.client.Do(req)
// 	if resp.StatusCode > 299 {
// 		err = ErrServerError
// 		return
// 	}
// 	// discard any body
// 	resp.Body.Close()
// 	return
// }
