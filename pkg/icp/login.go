package icp

import (
	"fmt"
	"io"
	"strconv"

	"github.com/itrs-group/cordial/pkg/config"
)

// Credentials type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-login
type Credentials struct {
	Username string `json:"Username"`
	Password string `json:"Password"`
}

var icp *ICP

// Login sends a login request to the http endpoint and returns a token
// or an error
func Login(projectID int, username string, password config.Plaintext, options ...Options) (icp *ICP, err error) {
	creds := &Credentials{
		Username: username,
		Password: password.String(),
	}
	icp = New(projectID, options...)
	resp, err := icp.Post("api/login", creds)
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
	icp.Token, _ = strconv.Unquote(string(b))
	return
}
