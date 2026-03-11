/*
Copyright © 2025 ITRS Group

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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"github.com/itrs-group/cordial/pkg/process"

	_ "github.com/itrs-group/cordial/tools/ims-gateway/internal/sdp"
	_ "github.com/itrs-group/cordial/tools/ims-gateway/internal/snow"
)

var daemon bool

func init() {
	rootCmd.AddCommand(routerCmd)

	routerCmd.Flags().BoolVarP(&daemon, "daemon", "D", false, "Daemonise the proxy process")
	routerCmd.PersistentFlags().StringVarP(&logFile, "logfile", "l", "-", "Write logs to `file`. Use '-' for console or "+os.DevNull+" for none")

	routerCmd.Flags().SortFlags = false
}

// routerCmd represents the proxy command
var routerCmd = &cobra.Command{
	Use:   "start",
	Short: "Run an ims-gateway",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Run: func(command *cobra.Command, args []string) {
		if daemon {
			var logArgs []string

			if logFile == "-" {
				logArgs = append(logArgs, "--logfile", cordial.ExecutableName()+".proxy.log")
			}

			if err := process.Daemon2(os.Stdout, logArgs, nil, "-D", "--daemon"); err != nil {
				log.Fatal().Err(err).Msg("failed to daemonise process")
			}
		}

		var l slog.Level = slog.LevelInfo
		if Debug {
			l = slog.LevelDebug
		}

		cf := LoadConfigFile()

		cordial.LogInit(cordial.ExecutableName(),
			cordial.LogLevel(l),
			cordial.SetLogfile(logFile),
			cordial.LumberjackOptions(&lumberjack.Logger{
				Filename:   logFile,
				MaxSize:    cf.GetInt("server.log.max-size"),
				MaxBackups: cf.GetInt("server.log.max-backups"),
				MaxAge:     cf.GetInt("server.log.stale-after"),
				Compress:   cf.GetBool("server.log.compress"),
			}),
			cordial.RotateOnStart(cf.GetBool("server.log.rotate-on-start")),
		)
		startGateway(cf)
	},
}

type ctxKey string

const startTimeKey ctxKey = "starttime"

func startGateway(cf *config.Config) {
	listen := cf.GetString(cf.Join("server", "listen"))
	basePath := cf.GetString(cf.Join("server", "path"))

	log.Debug().Msgf("starting proxy with configuration: listen=%s, path=%s", listen, basePath)

	// init connection or fail early
	// snow.NewClient(cf.Sub("snow"))

	mux := http.NewServeMux()

	for _, endpoint := range ims.Endpoints {
		mux.HandleFunc(endpoint.Method+" "+basePath+endpoint.Path, func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), ims.ContextKeyConfig, cf))
			endpoint.Handler(w, r)
		})
		log.Debug().Msgf("registered %s %s endpoint", endpoint.Method, basePath+endpoint.Path)
	}

	var handler http.Handler = mux
	handler = withKeyAuth(cf, handler)
	handler = withRequestLog(cf, handler)
	handler = withTimestamp(handler)

	log.Debug().Msg("starting HTTP server")

	if err := startHTTPServer(cf, listen, handler); err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
	}
}

func withTimestamp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), startTimeKey, time.Now())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func withKeyAuth(cf *config.Config, next http.Handler) http.Handler {
	expected := cf.GetString(cf.Join("server", "authentication", "token"))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := extractAPIKey(r)
		if key != expected {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withRequestLog(cf *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBody, _ := readAndRestoreRequestBody(r)

		rr := &responseRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(rr, r)
		requestLog(cf, r, reqBody, rr.body.Bytes(), rr.status, rr.size)
	})
}

func requestLog(cf *config.Config, r *http.Request, reqBody, resBody []byte, resStatus, resSize int64) {
	_ = reqBody // kept for parity with prior middleware signature

	message := ""
	var result map[string]any
	if err := json.Unmarshal(resBody, &result); err == nil {
		if v, ok := result["result"].(string); ok {
			message = v
		}
	}

	bytesIn := r.Header.Get("Content-Length")
	if bytesIn == "" {
		bytesIn = "0"
	}

	start, _ := r.Context().Value(startTimeKey).(time.Time)
	latency := time.Since(start)

	log.Info().Msgf("%s %s %3d %s/%d %.3fs %s %s %s %q",
		"URL", // cf.GetString(cf.Join("snow", "url")),
		r.Proto,
		resStatus,
		bytesIn,
		resSize,
		float64(latency.Milliseconds())/1000.0,
		realIP(r),
		r.Method,
		r.URL.String(),
		message,
	)
}

func startHTTPServer(cf *config.Config, listen string, handler http.Handler) error {
	if !cf.GetBool(cf.Join("server", "tls", "enabled")) {
		log.Debug().Msgf("starting server without TLS on %s", listen)
		return http.ListenAndServe(listen, handler)
	}

	certPEM := config.GetBytes(cf.Join("server", "tls", "certificate"))
	keyPEM := config.GetBytes(cf.Join("server", "tls", "private-key"))

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:    listen,
		Handler: handler,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}

	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return err
	}

	log.Debug().Msgf("starting server on %s", listen)
	return srv.Serve(tls.NewListener(ln, srv.TLSConfig))
}

func extractAPIKey(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return strings.TrimSpace(auth)
}

func readAndRestoreRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	b, err := ioReadAll(r.Body)
	r.Body.Close()
	r.Body = ioNopCloser(bytes.NewReader(b))
	return b, err
}

func realIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if p := strings.Split(xff, ","); len(p) > 0 {
			return strings.TrimSpace(p[0])
		}
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

type responseRecorder struct {
	http.ResponseWriter
	status int64
	size   int64
	body   bytes.Buffer
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = int64(code)
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	if n > 0 {
		r.size += int64(n)
		_, _ = r.body.Write(b[:n])
	}
	return n, err
}

// tiny wrappers to keep this file stdlib-only for HTTP server concerns.
func ioReadAll(body io.Reader) ([]byte, error) { return io.ReadAll(body) }
func ioNopCloser(r io.Reader) io.ReadCloser    { return io.NopCloser(r) }
