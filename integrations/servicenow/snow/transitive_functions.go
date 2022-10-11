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
		if strings.HasPrefix(passwordfile, "~/") {
			home, _ := os.UserHomeDir()
			passwordfile = strings.Replace(passwordfile, "~", home, 1)
		}
		if pw, err = os.ReadFile(passwordfile); err != nil {
			log.Fatalf("cannot read password from file %q", passwordfile)
		}
	}
	password := strings.TrimSpace(string(pw))

	username := vc.GetString("servicenow.username")
	clientid := vc.GetString("servicenow.clientid")
	clientsecret := vc.GetString("servicenow.clientsecret")
	instance := vc.GetString("servicenow.instance")

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
		}
		return cachedConnection
	}

	cachedConnection = &Connection{
		Client:   http.DefaultClient,
		Instance: instance,
		Username: username,
		Password: password,
	}
	return cachedConnection
}

func AssembleRequest(t RequestTransitive, table string) (req *http.Request) {
	host := t.Connection.Instance
	if !strings.Contains(host, ".") {
		host += ".service-now.com"
	}
	Url, err := url.Parse("https://" + host)
	if err != nil {
		fmt.Printf("Error%s\n", err)
	}

	if t.SysID != "" {
		Url.Path += "/api/now/v2/table/" + table + "/" + t.SysID
	} else {
		Url.Path += "/api/now/v2/table/" + table
	}

	z := t.Params.Encode()
	z = strings.ReplaceAll(z, "+", "%20") // XXX ?

	Url.RawQuery = z

	payload := bytes.NewReader(t.Payload)
	if req, err = http.NewRequest(t.Method, Url.String(), payload); err != nil {
		fmt.Printf("Error %s\n", err)
	}
	if t.Connection.Client == http.DefaultClient {
		req.SetBasicAuth(t.Connection.Username, t.Connection.Password)
	}
	req.Header.Add("Accept", "application/json")

	return req
}
