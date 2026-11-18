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
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// requestTimeout is the longest a request to ServiceNow can take
const requestTimeout = 30 * time.Second

var snowMutex sync.RWMutex
var snowConnection *Connection

// InitializeConnection TODO: Document its behavior, inputs, and effects.
func InitializeConnection(vc *config.Config) *Connection {
	var err error

	snowMutex.RLock()
	if snowConnection != nil {
		snowMutex.RUnlock()
		return snowConnection
	}
	snowMutex.RUnlock()

	pw := []byte(config.Get[string](vc, "servicenow.password"))

	// XXX - deprecated. Use above with expansion options
	if len(pw) == 0 {
		var passwordfile = config.Get[string](vc, "servicenow.passwordfile")
		if len(passwordfile) == 0 {
			log.Error("no password or password file configured")
			os.Exit(1)
		}
		passwordfile = config.ResolveHome(passwordfile)
		if pw, err = os.ReadFile(passwordfile); err != nil {
			log.Error("cannot read password from file", slog.Any("error", err), slog.String("file", passwordfile))
			os.Exit(1)
		}
	}
	password := strings.TrimSpace(string(pw))

	username := config.Get[string](vc, vc.Join("servicenow", "username"))
	clientid := config.Get[string](vc, vc.Join("servicenow", "clientid"))
	clientsecret := config.Get[string](vc, vc.Join("servicenow", "clientsecret"))
	instance := config.Get[string](vc, vc.Join("servicenow", "instance"))
	path := config.Get[string](vc, vc.Join("servicenow", "path"), config.DefaultValue("/api/now/v2"))
	trace := config.Get[bool](vc, vc.Join("servicenow", "trace"))

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

		client := oauthClient(context.Background(), conf, instanceURL(instance))

		snowMutex.Lock()
		// with OAuth we don't need to store the username and password
		snowConnection = &Connection{
			Client:   client,
			Instance: instance,
			Path:     path,
			Trace:    trace,
		}
		snowMutex.Unlock()
		return snowConnection
	}

	snowMutex.Lock()
	snowConnection = &Connection{
		Client:    basicClient(instanceURL(instance)),
		Instance:  instance,
		Path:      path,
		Username:  username,
		Password:  password,
		Trace:     trace,
		BasicAuth: true,
	}
	snowMutex.Unlock()
	return snowConnection
}

// instanceHost returns the host name of a ServiceNow instance, which is
// either the name of a service-now.com instance or a host name
func instanceHost(instance string) string {
	if !strings.Contains(instance, ".") {
		return instance + ".service-now.com"
	}
	return instance
}

// instanceURL returns the base URL of a ServiceNow instance
func instanceURL(instance string) *url.URL {
	return &url.URL{Scheme: "https", Host: instanceHost(instance)}
}

// oauthClient returns a client that sends the OAuth token to the
// instance only, and not to any other origin that it is redirected to.
// The token is requested using the HTTP client in ctx, if any.
func oauthClient(ctx context.Context, conf *clientcredentials.Config, instance *url.URL) *http.Client {
	client := conf.Client(ctx)
	plain := http.DefaultTransport
	if hc, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); ok && hc.Transport != nil {
		plain = hc.Transport
	}
	client.Transport = ims.OriginOnly(instance, client.Transport, plain)
	ims.SameOriginRedirects(client, instance)
	client.Timeout = requestTimeout
	return client
}

// basicClient returns a client for requests that have a username and
// password, which are not sent to another origin than the instance
func basicClient(instance *url.URL) *http.Client {
	client := &http.Client{Timeout: requestTimeout}
	ims.SameOriginRedirects(client, instance)
	return client
}

// AssembleRequest TODO: Document its behavior, inputs, and effects.
func AssembleRequest(snow TransitiveConnection, table string) (req *http.Request, err error) {
	snowMutex.RLock()
	defer snowMutex.RUnlock()

	u, err := url.Parse("https://" + instanceHost(snow.Instance))
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

	// there is no context for the request
	if req, err = http.NewRequestWithContext(context.Background(), snow.Method, u.String(), bytes.NewReader(snow.Payload)); err != nil {
		return
	}
	if snow.BasicAuth {
		req.SetBasicAuth(snow.Username, snow.Password)
	}
	req.Header.Add("Accept", "application/json")
	if len(snow.Payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	return
}
