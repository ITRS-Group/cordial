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

package snow

import (
	"net/http"
	"net/url"
	"regexp"

	"github.com/labstack/echo/v4"

	"github.com/itrs-group/cordial/pkg/config"
)

type Connection struct {
	Client   *http.Client
	Instance string
	Path     string
	Username string
	Password *config.Plaintext
	Trace    bool
}

type TransitiveConnection struct {
	Connection
	Payload []byte
	Method  string
	Params  url.Values
	SysID   string
}

type Context struct {
	echo.Context
	Conf *config.Config
}

// not a complete test, but just filter characters *allowed*
var userRE = regexp.MustCompile(`^[\w\.@ ]+$`)
