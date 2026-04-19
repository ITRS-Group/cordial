/*
Copyright © 2023 ITRS Group

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

package cmd

import (
	"fmt"
	"net/url"

	"github.com/itrs-group/cordial/pkg/commands"
	"github.com/itrs-group/cordial/pkg/config"
)

func dialGateway(cf *config.Config) (gw *commands.Connection, err error) {
	u := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", config.Get[string](cf, "gateway.host"), config.Get[uint16](cf, cf.Join("gateway", "port"))),
	}

	if config.Get[bool](cf, cf.Join("gateway", "use-tls")) {
		u.Scheme = "https"
	}

	username := config.Get[string](cf, cf.Join("gateway", "username"))
	gateway := config.Get[string](cf, cf.Join("gateway", "name"))

	password := &config.Secret{}

	if username != "" {
		password = config.Get[*config.Secret](cf, cf.Join("gateway", "password"))
	}

	if username == "" {
		var creds *config.Config
		if gateway != "" {
			creds = config.FindCreds("gateway:"+gateway, config.SetAppName("geneos"))
		} else {
			creds = config.FindCreds("gateway", config.SetAppName("geneos"))
		}
		if creds != nil {
			username = config.Get[string](creds, "username")
			password = config.Get[*config.Secret](creds, "password")
		}
	}

	return commands.DialGateway(u,
		commands.SetBasicAuth(username, password),
		commands.AllowInsecureCertificates(config.Get[bool](cf, "gateway.allow-insecure")),
	)
}
