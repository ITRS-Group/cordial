package instance

import (
	"fmt"
	"log/slog"
	"net/url"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos/commands"
	"github.com/itrs-group/cordial/pkg/geneos/xpath"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func SnapshotDataviews(i geneos.Instance, paths []string, options ...SnapshotOptions) (dataviews []*commands.Dataview, err error) {
	if CompareVersion(i, "5.14") <= 0 {
		err = fmt.Errorf("%s is too old (5.14 or above required)", i)
		return
	}
	i.Log().Debug("snapshot", slog.Any("paths", paths))

	opts := evalSnapshotOptions(options...)

	for _, path := range paths {
		var x *xpath.XPath
		x, err = xpath.Parse(path)
		if err != nil {
			i.Log().Error("failed to parse xpath", slog.Any("error", err), slog.String("path", path))
			continue
		}

		// always try to use auth details in per-instance config,
		// default to from the command line or user/global config or
		// credentials file
		username := config.Get[string](i.Config(), config.Join("snapshot", "username"))
		password := config.Get[config.Secret](i.Config(), config.Join("snapshot", "password"))
		defer clear(password)

		if username == "" {
			username = opts.username
		}

		if password == nil {
			password = opts.password
		}

		// if username is still unset then look for credentials
		//
		// credential domain is gateway:NAME or gateway:* for wildcard
		if username == "" {
			creds := config.FindCreds(i.Type().String()+":"+i.Name(), config.AppName(cordial.ExecutableName()))
			if creds != nil {
				username = config.Get[string](creds, "username")
				password = config.Get[config.Secret](creds, "password")
				defer clear(password)
			} else {
				if creds = config.FindCreds(i.Type().String()+":*", config.AppName(cordial.ExecutableName())); creds != nil {
					username = config.Get[string](creds, "username")
					password = config.Get[config.Secret](creds, "password")
					defer clear(password)
				}
			}
		}

		i.Log().Debug("dialling", slog.Any("url", gatewayURL(i)))
		var gw *commands.Connection
		gw, err = commands.DialGateway(
			gatewayURL(i),
			commands.AllowInsecureCertificates(true),
			commands.SetBasicAuth(username, password),
		)
		if err != nil {
			return
		}
		d := x.ResolveTo(&xpath.Dataview{})
		i.Log().Debug("matching xpath", slog.Any("xpath", d))
		var views []*xpath.XPath
		views, err = gw.Match(d, 0)
		if err != nil {
			return
		}
		if opts.maxitems > 0 && len(views) > opts.maxitems {
			views = views[0:opts.maxitems]
		}
		if opts.xpathOnly {
			for _, x := range views {
				dataviews = append(dataviews, &commands.Dataview{XPath: x})
			}
		} else {
			for _, view := range views {
				var data *commands.Dataview
				data, err = gw.Snapshot(view, "", opts.scopes)
				if err != nil {
					return
				}
				dataviews = append(dataviews, data)
			}
		}
	}

	return
}

func gatewayURL(i geneos.Instance) (u *url.URL) {
	if !IsA(i, "gateway") {
		return
	}
	u = &url.URL{}
	hostname := config.Get[string](i.Host().Config, "hostname")
	if hostname == "" {
		hostname = "localhost"
	}
	port := config.Get[uint16](i.Config(), "port")
	u.Host = fmt.Sprintf("%s:%d", hostname, port)
	u.Scheme = "http"
	if IsTLSCapable(i) {
		u.Scheme = "https"
	}
	return
}

type SnapshotOptions func(*snapshotOptions)

type snapshotOptions struct {
	username  string
	password  config.Secret
	scopes    commands.Scope
	maxitems  int
	xpathOnly bool
}

func evalSnapshotOptions(options ...SnapshotOptions) *snapshotOptions {
	opts := &snapshotOptions{
		scopes: commands.Scope{Value: true},
	}
	for _, option := range options {
		option(opts)
	}
	return opts
}

func SnapshotUsername(username string) SnapshotOptions {
	return func(opts *snapshotOptions) {
		// set the username in the snapshot options
		opts.username = username
	}
}

func SnapshotPassword(password config.Secret) SnapshotOptions {
	return func(opts *snapshotOptions) {
		// set the password in the snapshot options
		opts.password = password
	}
}

func SnapshotScopes(scopes commands.Scope) SnapshotOptions {
	return func(opts *snapshotOptions) {
		// set the scopes in the snapshot options
		opts.scopes = scopes
	}
}

func SnapshotMaxItems(maxitems int) SnapshotOptions {
	return func(opts *snapshotOptions) {
		// set the maxitems in the snapshot options
		opts.maxitems = maxitems
	}
}

func SnapshotXPathOnly(xpathOnly bool) SnapshotOptions {
	return func(opts *snapshotOptions) {
		// set the xpathOnly flag in the snapshot options
		opts.xpathOnly = xpathOnly
	}
}
