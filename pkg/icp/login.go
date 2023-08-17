package icp

import (
	"context"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/rest"
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
func Login(username string, password *config.Plaintext, options ...rest.Options) (icp *ICP, err error) {
	creds := &LoginRequest{
		Username: username,
		Password: password.String(),
	}
	icp = New(options...)
	var token string
	_, err = icp.Post(context.Background(), LoginEndpoint, creds, &token)
	if err != nil {
		return
	}
	icp.SetAuth("Authorization", "SUMERIAN "+token)
	return
}
