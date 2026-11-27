package instance

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos/commands"
	"github.com/itrs-group/cordial/pkg/geneos/xpath"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// SnapshotDataviews TODO: Document its behavior, inputs, and effects.
func SnapshotDataviews(i geneos.Instance, paths []string, options ...SnapshotOptions) (dataviews []*commands.Dataview, err error) {
	if i != nil && CompareVersion(i, "5.14") < 0 {
		err = fmt.Errorf("%s is too old (5.14 or above required)", i)
		return
	}
	if i != nil {
		i.Log().Debug("snapshot", slog.Any("paths", paths))
	}

	opts := evalSnapshotOptions(options...)

	// always try to use auth details in per-instance config, default to
	// from the command line or user/global config or credentials file.
	//
	// only clear passwords that are copies made here, never the one
	// passed in by the caller, which may be used for other instances
	var username string
	var password config.Secret
	ownPassword := true
	if i != nil {
		username = config.Get[string](i.Config(), config.Join("snapshot", "username"))
		password = config.Get[config.Secret](i.Config(), config.Join("snapshot", "password"))
	}
	defer func() {
		if ownPassword {
			clear(password)
		}
	}()

	if username == "" {
		username = opts.username
	}

	if len(password) == 0 {
		password = opts.password
		ownPassword = false
	}

	// if username is still unset then look for credentials
	//
	// credential domain is gateway:NAME or gateway:* for wildcard
	if username == "" && i != nil {
		creds := config.FindCreds(i.Type().String()+":"+i.Name(), config.AppName(cordial.ExecutableName()))
		if creds == nil {
			creds = config.FindCreds(i.Type().String()+":*", config.AppName(cordial.ExecutableName()))
		}
		if creds != nil {
			username = config.Get[string](creds, "username")
			password = config.Get[config.Secret](creds, "password")
			ownPassword = true
		}
	}
	if username != "" && len(password) == 0 && !opts.passwordSet && opts.prompt {
		password, err = config.ReadPasswordInput(false, 0)
		if err != nil {
			if errors.Is(err, config.ErrNotInteractive) {
				err = fmt.Errorf("%w and password required", err)
			}
			return
		}
		ownPassword = true
	}

	var parseErrs error
	urls, autoFallback, err := snapshotGatewayURLs(i, opts.endpoint)
	if err != nil {
		return nil, err
	}

	var gw *commands.Connection
	for _, path := range paths {
		var x *xpath.XPath
		x, err = xpath.Parse(path)
		if err != nil {
			if i != nil {
				i.Log().Error("failed to parse xpath", slog.Any("error", err), slog.String("path", path))
			}
			parseErrs = errors.Join(parseErrs, fmt.Errorf("invalid xpath %q: %w", path, err))
			continue
		}

		if gw == nil {
			for n, u := range urls {
				if i != nil {
					i.Log().Debug("dialling", slog.Any("url", u))
				}
				gw, err = commands.DialGateway(
					u,
					commands.AllowInsecureCertificates(i != nil),
					commands.SetBasicAuth(username, password),
				)
				if err == nil || !autoFallback || n == len(urls)-1 || !snapshotTLSUnsupported(err) {
					break
				}
			}
			if err != nil {
				return
			}
		}
		d := x.ResolveTo(&xpath.Dataview{})
		if i != nil {
			i.Log().Debug("matching xpath", slog.Any("xpath", d))
		}
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

	return dataviews, parseErrs
}

func snapshotGatewayURLs(i geneos.Instance, endpoint string) (urls []*url.URL, autoFallback bool, err error) {
	if endpoint == "" {
		if i == nil {
			return nil, false, geneos.ErrInvalidArgs
		}
		return []*url.URL{gatewayURL(i)}, false, nil
	}

	if strings.Contains(endpoint, "://") {
		u, parseErr := url.Parse(endpoint)
		if parseErr != nil {
			return nil, false, fmt.Errorf("invalid connect value %q: %w", endpoint, parseErr)
		}
		scheme := strings.ToLower(u.Scheme)
		if scheme != "http" && scheme != "https" {
			return nil, false, fmt.Errorf("connect URL scheme %q is not supported", u.Scheme)
		}
		if u.Host == "" {
			return nil, false, fmt.Errorf("connect URL %q has no host", endpoint)
		}
		u.Scheme = scheme
		return []*url.URL{u}, false, nil
	}

	host, port, splitErr := net.SplitHostPort(endpoint)
	if splitErr != nil {
		return nil, false, fmt.Errorf("connect value %q must be HOST:PORT or an http(s) URL: %w", endpoint, splitErr)
	}
	if host == "" || port == "" {
		return nil, false, fmt.Errorf("connect value %q must include a host and port", endpoint)
	}
	address := net.JoinHostPort(host, port)
	httpsURL := &url.URL{Scheme: "https", Host: address}
	httpURL := &url.URL{Scheme: "http", Host: address}
	return []*url.URL{httpsURL, httpURL}, true, nil
}

func snapshotTLSUnsupported(err error) bool {
	if err == nil {
		return false
	}
	var recordHeaderError tls.RecordHeaderError
	if errors.As(err, &recordHeaderError) {
		return true
	}
	message := err.Error()
	return strings.Contains(message, "server gave HTTP response to HTTPS client") ||
		strings.Contains(message, "first record does not look like a TLS handshake")
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

// SnapshotOptions TODO: Describe its purpose and intended use.
type SnapshotOptions func(*snapshotOptions)

type snapshotOptions struct {
	username    string
	password    config.Secret
	passwordSet bool
	prompt      bool
	endpoint    string
	scopes      commands.Scope
	maxitems    int
	xpathOnly   bool
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

// SnapshotUsername TODO: Document its behavior, inputs, and effects.
func SnapshotUsername(username string) SnapshotOptions {
	return func(opts *snapshotOptions) {
		// set the username in the snapshot options
		opts.username = username
	}
}

// SnapshotPassword TODO: Document its behavior, inputs, and effects.
func SnapshotPassword(password config.Secret) SnapshotOptions {
	return func(opts *snapshotOptions) {
		// set the password in the snapshot options
		opts.password = password
	}
}

// SnapshotPasswordSet records whether an explicitly empty password was supplied.
func SnapshotPasswordSet(set bool) SnapshotOptions {
	return func(opts *snapshotOptions) {
		opts.passwordSet = set
	}
}

// SnapshotPrompt enables prompting when a username is available without a password.
func SnapshotPrompt(prompt bool) SnapshotOptions {
	return func(opts *snapshotOptions) {
		opts.prompt = prompt
	}
}

// SnapshotEndpoint selects a direct Commands API connection instead of an instance.
func SnapshotEndpoint(endpoint string) SnapshotOptions {
	return func(opts *snapshotOptions) {
		opts.endpoint = endpoint
	}
}

// SnapshotScopes TODO: Document its behavior, inputs, and effects.
func SnapshotScopes(scopes commands.Scope) SnapshotOptions {
	return func(opts *snapshotOptions) {
		// set the scopes in the snapshot options
		opts.scopes = scopes
	}
}

// SnapshotMaxItems TODO: Document its behavior, inputs, and effects.
func SnapshotMaxItems(maxitems int) SnapshotOptions {
	return func(opts *snapshotOptions) {
		// set the maxitems in the snapshot options
		opts.maxitems = maxitems
	}
}

// SnapshotXPathOnly TODO: Document its behavior, inputs, and effects.
func SnapshotXPathOnly(xpathOnly bool) SnapshotOptions {
	return func(opts *snapshotOptions) {
		// set the xpathOnly flag in the snapshot options
		opts.xpathOnly = xpathOnly
	}
}
