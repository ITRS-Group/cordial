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
		Host:   fmt.Sprintf("%s:%d", cf.GetString("gateway.host"), cf.GetInt("gateway.port")),
	}

	if cf.GetBool("gateway.use-tls") {
		u.Scheme = "https"
	}

	username := cf.GetString("gateway.username")
	gateway := cf.GetString("gateway.name")

	password := &config.Plaintext{}

	if username != "" {
		password = cf.GetPassword("gateway.password")
	}

	if username == "" {
		var creds *config.Config
		if gateway != "" {
			creds = config.FindCreds("gateway:"+gateway, config.SetAppName("geneos"))
		} else {
			creds = config.FindCreds("gateway", config.SetAppName("geneos"))
		}
		if creds != nil {
			username = creds.GetString("username")
			password = creds.GetPassword("password")
		}
	}

	return commands.DialGateway(u,
		commands.SetBasicAuth(username, password),
		commands.AllowInsecureCertificates(cf.GetBool("gateway.allow-insecure")),
	)
}
