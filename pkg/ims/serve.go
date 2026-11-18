/*
Copyright © 2026 ITRS Group

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

package ims

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"
)

// shutdownTimeout is how long in-flight requests are given to complete
// when the server is stopped, or when a server that has been replaced
// by a reload drains
const shutdownTimeout = 15 * time.Second

// ServeOptions are the optional settings for [ServeWith]
type ServeOptions struct {
	// Context stops the server, gracefully, when it is done. If nil the
	// server runs until it is signalled.
	Context context.Context

	// PIDFile is the path of a PID file to hold, and write once the
	// server is listening, while the server runs. If empty there is
	// none. If another process holds the file the server is not started.
	PIDFile string

	// ConfigFile is recorded in the PID file, so that the control
	// commands can find the configuration to validate for a reload
	ConfigFile string

	// Load reads the configuration again, for a reload. If nil the
	// server cannot be reloaded. It must return a new configuration
	// each time, as the one in use is not changed so that requests in
	// progress are not affected.
	Load func() (*config.Config, error)

	// Check validates a configuration that has been read for a reload,
	// in addition to the checks of the server settings. A configuration
	// that fails is not used and the server continues unchanged.
	Check func(cf *config.Config) error

	// Apply is called, with a configuration that has passed all the
	// checks, just before it is used. It can be used to change things
	// that are not part of the server, such as the logging.
	Apply func(cf *config.Config)

	// Ready is called, with the address, each time the server is
	// listening, which is at start-up and when a reload changes the
	// address
	Ready func(addr net.Addr)
}

// Serve serves the endpoints on the address in the configuration, with
// TLS if enabled, until the process is signalled, when it shuts down
// gracefully. It returns an error if the configuration is not usable,
// the server cannot be started or it stops unexpectedly.
//
// See [ServeWith] for the configuration keys used and for reloading.
func Serve(cf *config.Config, endpoints []Endpoint) error {
	return ServeWith(cf, endpoints, ServeOptions{})
}

// ServeWith is like [Serve] but with options, for example to hold a PID
// file and to reload the configuration.
//
// The configuration keys used are all in the `server` section:
// `listen`, `path` (the prefix for all endpoint paths),
// `authentication.token` (required, requests must present it as a
// bearer token) and `tls.enabled`, `tls.certificate` and
// `tls.private-key`.
//
// If a Load function is given then the server reloads its
// configuration when it receives the signal [process.SigReload], which
// is SIGUSR1 (reloading is not available on Windows). A new
// configuration is only used if it can be loaded and passes the checks,
// otherwise the error is logged and the server continues as it was. The
// new configuration is used for all requests received after that,
// including for the token, path, tables and ServiceNow settings, while
// requests already in progress complete using the one they started with.
// The certificate is replaced without any interruption, and if the
// listen address has changed then the new address is bound before the
// old one is closed, which then stops listening and has up to 15 seconds
// to complete its requests. Turning TLS on or off needs a restart.
func ServeWith(cf *config.Config, endpoints []Endpoint, opts ServeOptions) (err error) {
	settings, err := readSettings(cf)
	if err != nil {
		return err
	}

	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// catch the signals before there is anything to signal, as the
	// default for the reload signal is to terminate the process
	sigs := make(chan os.Signal, 4)
	reloadSigs := reloadSignals()
	signal.Notify(sigs, append([]os.Signal{os.Interrupt, syscall.SIGTERM}, reloadSigs...)...)
	defer signal.Stop(sigs)

	// hold the PID file before listening so that a second instance
	// fails with a clear error, but only write it once we are ready
	var pf *process.PIDFile
	if opts.PIDFile != "" {
		if pf, err = process.LockPIDFile(opts.PIDFile); err != nil {
			return err
		}
		defer pf.Release()
	}

	log.Debug("starting server with configuration",
		slog.String("listen", settings.listen),
		slog.String("path", settings.path),
	)

	errs := make(chan error, 8)
	var draining sync.WaitGroup

	cur, err := newGeneration(settings, NewHandler(cf, endpoints, settings.token))
	if err != nil {
		return err
	}
	go cur.serve(errs)
	log.Info("listening", slog.String("address", cur.ln.Addr().String()), slog.Bool("tls", cur.tls))
	if opts.Ready != nil {
		opts.Ready(cur.ln.Addr())
	}

	if pf != nil {
		info := process.PIDInfo{Args: os.Args[1:], ConfigFile: opts.ConfigFile}
		info.Executable, _ = os.Executable()
		info.Dir, _ = os.Getwd()
		if err = pf.Write(info); err != nil {
			cur.shutdown()
			return fmt.Errorf("cannot write pid file: %w", err)
		}
	}

	stop := func() {
		cur.shutdown()
		draining.Wait()
	}

	// reload reads and checks a new configuration and, if it is good,
	// starts using it
	reload := func() error {
		if opts.Load == nil {
			return errors.New("reloading is not supported")
		}
		ncf, err := opts.Load()
		if err != nil {
			return fmt.Errorf("cannot load configuration: %w", err)
		}
		ns, err := readSettings(ncf)
		if err != nil {
			return err
		}
		if opts.Check != nil {
			if err = opts.Check(ncf); err != nil {
				return err
			}
		}
		if ns.tls != cur.tls {
			return errors.New("turning TLS on or off needs a restart")
		}

		handler := NewHandler(ncf, endpoints, ns.token)

		var next *generation
		// the address is unchanged if it is the same as configured or as
		// actually bound, which differ if the port was 0
		if ns.listen != cur.listen && ns.listen != cur.ln.Addr().String() {
			// bind the new address before anything is changed, so that
			// failing to leaves the server as it was
			if next, err = newGeneration(ns, handler); err != nil {
				return err
			}
		}

		if opts.Apply != nil {
			opts.Apply(ncf)
		}

		if next == nil {
			cur.cert.Store(ns.cert)
			cur.setHandler(handler)
			return nil
		}

		old := cur
		cur = next
		go cur.serve(errs)
		log.Info("listening", slog.String("address", cur.ln.Addr().String()), slog.Bool("tls", cur.tls))
		if opts.Ready != nil {
			opts.Ready(cur.ln.Addr())
		}
		draining.Go(old.shutdown)
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("shutting down")
			stop()
			return nil

		case s := <-sigs:
			if slices.Contains(reloadSigs, s) {
				log.Info("reloading configuration", slog.String("signal", s.String()))
				if err := reload(); err != nil {
					log.Error("configuration not reloaded, continuing with the current configuration", slog.Any("error", err))
				} else {
					log.Info("configuration reloaded")
				}
				continue
			}
			log.Info("shutting down", slog.String("signal", s.String()))
			stop()
			return nil

		case err := <-errs:
			stop()
			return err
		}
	}
}

// serverSettings are the settings from the `server` section that can
// be changed by a reload
type serverSettings struct {
	listen string
	token  string
	path   string
	tls    bool
	cert   *tls.Certificate
}

// readSettings reads, and checks, the settings from the `server` section
func readSettings(cf *config.Config) (s serverSettings, err error) {
	s.listen = config.Get[string](cf, cf.Join("server", "listen"))
	if s.listen == "" {
		return s, fmt.Errorf("no listen address configured (%s)", cf.Join("server", "listen"))
	}

	s.token = config.Get[string](cf, cf.Join("server", "authentication", "token"))
	if s.token == "" {
		return s, fmt.Errorf("no authentication token configured (%s)", cf.Join("server", "authentication", "token"))
	}

	s.path = config.Get[string](cf, cf.Join("server", "path"))

	if s.tls = config.Get[bool](cf, cf.Join("server", "tls", "enabled")); s.tls {
		cert, err := tls.X509KeyPair(
			config.Get[[]byte](cf, cf.Join("server", "tls", "certificate")),
			[]byte(config.Get[config.Secret](cf, cf.Join("server", "tls", "private-key"))),
		)
		if err != nil {
			return s, fmt.Errorf("cannot load TLS certificate and private key: %w", err)
		}
		s.cert = &cert
	}
	return
}

// CheckServer returns an error if the configuration cannot be used to
// run the server, in the same way as the server checks it at start-up and
// when reloading.
func CheckServer(cf *config.Config) error {
	_, err := readSettings(cf)
	return err
}

// generation is a listener and the HTTP server using it. There is only
// one unless the listen address is changed by a reload, in which case the
// old one drains while the new one takes over.
type generation struct {
	srv     *http.Server
	ln      net.Listener
	listen  string
	tls     bool
	handler atomic.Pointer[http.Handler]
	cert    atomic.Pointer[tls.Certificate]
}

// newGeneration listens on the address and returns a server, which is
// not yet serving
func newGeneration(s serverSettings, handler http.Handler) (g *generation, err error) {
	// listen explicitly so that failures are reported before serving
	ln, err := net.Listen("tcp", s.listen)
	if err != nil {
		return nil, err
	}

	g = &generation{ln: ln, listen: s.listen, tls: s.tls}
	g.setHandler(handler)
	// there is no WriteTimeout as how long a handler takes is limited by
	// the timeouts of the requests it makes, which can be longer than any
	// one timeout that is right for all handlers
	g.srv = &http.Server{
		Addr: s.listen,
		// requests are passed to whichever handler is current
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			(*g.handler.Load()).ServeHTTP(w, r)
		}),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
	}
	if s.tls {
		g.cert.Store(s.cert)
		g.srv.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
				return g.cert.Load(), nil
			},
		}
	}
	return
}

func (g *generation) setHandler(h http.Handler) { g.handler.Store(&h) }

// serve serves until the server is shut down, and sends any other
// error to errs
func (g *generation) serve(errs chan<- error) {
	var err error
	if g.tls {
		err = g.srv.ServeTLS(g.ln, "", "")
	} else {
		err = g.srv.Serve(g.ln)
	}
	if !errors.Is(err, http.ErrServerClosed) {
		errs <- err
	}
}

// shutdown stops listening and waits for requests in progress to
// complete, for up to the shutdown timeout
func (g *generation) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := g.srv.Shutdown(ctx); err != nil {
		log.Error("error shutting down", slog.String("address", g.listen), slog.Any("error", err))
	}
}
