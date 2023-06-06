package icp

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/itrs-group/cordial/pkg/config"
)

// LoginRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-login
type LoginRequest struct {
	Username string `json:"Username"`
	Password string `json:"Password"`
}

var icp *ICP

// Login sends a login request to the http endpoint and returns a token
// or an error
func Login(username string, password config.Plaintext, options ...Options) (icp *ICP, err error) {
	creds := &LoginRequest{
		Username: username,
		Password: password.String(),
	}
	icp = New(options...)
	resp, err := icp.Post(context.Background(), LoginEndpoint, creds)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		err = fmt.Errorf("%s", resp.Status)
		return
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	icp.token, _ = strconv.Unquote(string(b))
	return
}
