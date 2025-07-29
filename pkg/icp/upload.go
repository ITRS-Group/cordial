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
func (i *ICP) Upload(ctx context.Context, request UploadRequest, filename string, body io.ReadCloser) (resp *http.Response, err error) {
	// this does not use normal Post method, it uses query parameters
	// and sends a file as the body

	if icp.token == "" {
		err = errors.New("auth token required")
		return
	}

	dest := icp.BaseURL.JoinPath(UploadEndpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", dest.String(), body)
	if err != nil {
		return
	}
	req.Header.Add("Authorization", "SUMERIAN "+icp.token)
	req.Header.Add("Context-Type", "application/binary")
	req.Header.Add("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	v, err := query.Values(request)
	if err != nil {
		return resp, err
	}
	req.URL.RawQuery = v.Encode()

	return icp.Client.HTTPClient.Do(req)
}
