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
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"golang.org/x/oauth2/clientcredentials"
)

var cachedConnection *Connection

func InitializeConnection(vc *config.Config) *Connection {
	var err error

	if cachedConnection != nil {
		return cachedConnection
	}

	pw := []byte(vc.GetString("servicenow.password"))

	// XXX - deprecated. Use above with expansion options
	if len(pw) == 0 {
		var passwordfile = vc.GetString("servicenow.passwordfile")
		if len(passwordfile) == 0 {
			log.Fatalln("no password or password file configured")
		}
		passwordfile = config.ExpandHome(passwordfile)
		if pw, err = os.ReadFile(passwordfile); err != nil {
			log.Fatalf("cannot read password from file %q", passwordfile)
		}
	}
	password := strings.TrimSpace(string(pw))

	username := vc.GetString("servicenow.username")
	clientid := vc.GetString("servicenow.clientid")
	clientsecret := vc.GetString("servicenow.clientsecret")
	instance := vc.GetString("servicenow.instance")
	trace := vc.GetBool(config.Join("servicenow", "trace"))

	fmt.Println("trace enabled:", trace)

	if clientid != "" && clientsecret != "" && !strings.Contains(instance, ".") {
		params := make(url.Values)
		params.Set("grant_type", "password")
		params.Set("username", username)
		params.Set("password", password)

		conf := &clientcredentials.Config{
			ClientID:       clientid,
			ClientSecret:   clientsecret,
			EndpointParams: params,
			TokenURL:       "https://" + instance + ".service-now.com/oauth_token.do",
		}

		// with OAuth we don't need to store the username and password
		cachedConnection = &Connection{
			Client:   conf.Client(context.Background()),
			Instance: instance,
			Trace:    trace,
		}
		return cachedConnection
	}

	cachedConnection = &Connection{
		Client:   http.DefaultClient,
		Instance: instance,
		Username: username,
		Password: password,
		Trace:    trace,
	}
	return cachedConnection
}

func AssembleRequest(t RequestTransitive, table string) (req *http.Request, err error) {
	host := t.Connection.Instance
	if !strings.Contains(host, ".") {
		host += ".service-now.com"
	}
	u, err := url.Parse("https://" + host)
	if err != nil {
		return
	}

	if t.SysID != "" {
		u.Path += "/api/now/v2/table/" + table + "/" + t.SysID
	} else {
		u.Path += "/api/now/v2/table/" + table
	}

	z := t.Params.Encode()
	z = strings.ReplaceAll(z, "+", "%20") // XXX ?

	u.RawQuery = z

	if req, err = http.NewRequest(t.Method, u.String(), bytes.NewReader(t.Payload)); err != nil {
		return
	}
	if t.Connection.Client == http.DefaultClient {
		req.SetBasicAuth(t.Connection.Username, t.Connection.Password)
	}
	req.Header.Add("Accept", "application/json")

	return
}
