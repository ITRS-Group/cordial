package icp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/go-querystring/query"
)

// UploadRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-Upload_version_taskname_machinename_selectedProjectId_testOnly_log
type UploadRequest struct {
	Version           string `url:"version"`
	TaskName          string `url:"taskname"`
	MachineName       string `url:"machinename"`
	SelectedProjectID int    `url:"selectedProjectId"`
	TestOnly          bool   `url:"testOnly"`
	Log               int    `url:"log"`
}

// Upload request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-Upload_version_taskname_machinename_selectedProjectId_testOnly_log
//
// The client must have been authenticated with [Login]. body is closed
// before Upload returns. As for the other requests, a status outside
// 200-299 is returned as a [*rest.HTTPError] and the response body is
// closed.
func (i *ICP) Upload(ctx context.Context, request UploadRequest, filename string, body io.ReadCloser) (resp *http.Response, err error) {
	// this does not use the normal Post method, it uses query parameters
	// and sends a file as the body

	if i.token == "" {
		if body != nil {
			_ = body.Close()
		}
		return nil, errors.New("auth token required")
	}

	v, err := query.Values(request)
	if err != nil {
		if body != nil {
			_ = body.Close()
		}
		return nil, err
	}

	dest := i.BaseURL.JoinPath(UploadEndpoint)
	dest.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dest.String(), body)
	if err != nil {
		if body != nil {
			_ = body.Close()
		}
		return nil, err
	}
	req.Header.Set("Content-Type", "application/binary")
	req.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	return i.Send(req, nil)
}
