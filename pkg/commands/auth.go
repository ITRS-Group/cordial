/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
