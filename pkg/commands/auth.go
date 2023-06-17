/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package commands

import (
	"net/http"

	"github.com/itrs-group/cordial/pkg/config"
)

// Authentication types. SSO is not currently implemented.
const (
	None = iota
	Basic
	SSO
)

// SSOAuth is a placeholder struct for SSO authentication in the command
// Connection struct
type SSOAuth struct {
	AccessToken string `json:"access_token,omitempty"`
	Expires     int64  `json:"expires,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
}

// AuthSSO is a placeholder for future SSO authentication
func AuthSSO() {
	const endpoint = "/rest/authorize"

}

// AuthBasic sets up HTTP Basic Authentication on client c using the
// plaintext username and password pw. The password is a
// config.Plaintext enclave
func AuthBasic(c *http.Request, username string, password *config.Plaintext) (err error) {
	if c != nil {
		c.SetBasicAuth(username, password.String())
	}
	return
}
