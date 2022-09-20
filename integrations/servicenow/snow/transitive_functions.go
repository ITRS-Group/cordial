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
	"net/http"
	"net/url"
	"strings"

	"github.com/itrs-group/cordial/integrations/servicenow/settings"
	"github.com/itrs-group/cordial/pkg/config"
	"golang.org/x/oauth2/clientcredentials"
)

func InitializeConnection(cf settings.Settings, password string) *Connection {
	var client *http.Client = http.DefaultClient

	if cf.ServiceNow.ClientID != "" && cf.ServiceNow.ClientSecret != "" && !strings.Contains(cf.ServiceNow.Instance, ".") {
		params := make(url.Values)
		params.Set("grant_type", "password")
		params.Set("username", cf.ServiceNow.Username)
		params.Set("password", password)
		conf := &clientcredentials.Config{
			ClientID:       cf.ServiceNow.ClientID,
			ClientSecret:   config.GetConfig().ExpandString(cf.ServiceNow.ClientSecret, nil),
			EndpointParams: params,
			TokenURL:       "https://" + cf.ServiceNow.Instance + ".service-now.com/oauth_token.do",
		}

		client = conf.Client(context.Background())
	}
	return &Connection{
		client,
		cf.ServiceNow.Instance,
		cf.ServiceNow.Username,
		password,
	}
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

	z := t.Parms.Encode()
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
