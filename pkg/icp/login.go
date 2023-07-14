package icp

import (
	"context"

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
func Login(username string, password *config.Plaintext, options ...Options) (icp *ICP, err error) {
	creds := &LoginRequest{
		Username: username,
		Password: password.String(),
	}
	icp = New(options...)
	_, err = icp.Post(context.Background(), LoginEndpoint, creds, &icp.token)
	if err != nil {
		return
	}
	return
}
