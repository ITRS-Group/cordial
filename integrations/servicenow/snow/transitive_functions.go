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
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/itrs-group/cordial/pkg/config"
	"golang.org/x/oauth2/clientcredentials"
)

var snowMutex sync.RWMutex
var snowConnection *Connection

func InitializeConnection(vc *config.Config) *Connection {
	var err error

	snowMutex.RLock()
	if snowConnection != nil {
		snowMutex.RUnlock()
		return snowConnection
	}
	snowMutex.RUnlock()

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
	path := vc.GetString("servicenow.path", config.Default("/api/now/v2"))
	trace := vc.GetBool(config.Join("servicenow", "trace"))

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

		snowMutex.Lock()
		// with OAuth we don't need to store the username and password
		snowConnection = &Connection{
			Client:   conf.Client(context.Background()),
			Instance: instance,
			Path:     path,
			Trace:    trace,
		}
		snowMutex.Unlock()
		return snowConnection
	}

	snowMutex.Lock()
	snowConnection = &Connection{
		Client:   http.DefaultClient,
		Instance: instance,
		Path:     path,
		Username: username,
		Password: password,
		Trace:    trace,
	}
	snowMutex.Unlock()
	return snowConnection
}

func AssembleRequest(snow TransitiveConnection, table string) (req *http.Request, err error) {
	snowMutex.RLock()
	defer snowMutex.RUnlock()

	host := snow.Instance
	if !strings.Contains(host, ".") {
		host += ".service-now.com"
	}
	u, err := url.Parse("https://" + host)
	if err != nil {
		return
	}

	if snow.SysID != "" {
		u.Path += path.Join(snow.Path, "table", table, snow.SysID)
	} else {
		u.Path += path.Join(snow.Path, "table", table)
	}

	z := snow.Params.Encode()
	z = strings.ReplaceAll(z, "+", "%20") // XXX ?

	u.RawQuery = z

	if req, err = http.NewRequest(snow.Method, u.String(), bytes.NewReader(snow.Payload)); err != nil {
		return
	}
	if snow.Client == http.DefaultClient {
		req.SetBasicAuth(snow.Username, snow.Password)
	}
	req.Header.Add("Accept", "application/json")

	return
}
