/*
Copyright © 2024 ITRS Group

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
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"
)

var daemon bool

var logfile string

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().BoolVarP(&daemon, "daemon", "D", false, "Run as a daemon")
	serverCmd.Flags().StringVarP(&logfile, "logfile", "L", "", "Override configured log file path")

}

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run server for config request",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if daemon {
			process.Daemon(os.Stdout, process.RemoveArgs, "-D", "--daemon")
		}

		if logfile == "" {
			logfile = config.Get[string](cf, "server.logs.path")
		}

		logfile = config.ResolveHome(logfile)

		done := make(chan bool)

		if logfile == "-" {
			logfile = ""
		}
		initConfig(cmd)

		var usetls string
		if config.Get[bool](cf, "server.tls.enable") {
			usetls = "s"
		}
		log.Info().Msgf("starting %s version %s. listening for %s connections on %s:%d", cordial.ExecutableName(), cordial.VERSION, "http"+usetls, config.Get[string](cf, "server.host"), config.Get[uint16](cf, "server.port"))
		cs, e := initServer(cf)
		go cs.startServer(e)
		<-done
		return
	},
}

func initServer(cf *config.Config) (cs *ConfigServer, e *echo.Echo) {
	e = echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.Recover())
	// log requests
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log.Debug().Str("URI", v.URI).Int("status", v.Status).Msg("request")
			return nil
		},
	}))

	cs = &ConfigServer{conf: cf, hosts: map[string]HostMappings{}}

	return
}

// ConfigServer encapsulates the data required to server config (and
// connection) requests
type ConfigServer struct {
	sync.RWMutex
	conf     *config.Config
	gateways []string                // live gateways
	hosts    map[string]HostMappings // known host mappings - always Clone HostMappings before changing!
}

func (cs *ConfigServer) startServer(e *echo.Echo) {
	var err error

	// initialise both at least once before starting listener, mutex not
	// yet required
	for {
		cs.hosts, err = LoadHosts(cs.conf)
		if err == nil {
			break
		}
		log.Error().Err(err).Msg("retrying until first inventory load(s) succeeds")
		check := config.Get[time.Duration](cs.conf, "inventory.check-interval")
		if check == 0 {
			check = 60 * time.Second
		}
		time.Sleep(check)
	}
	cs.gateways = CheckGateways(cs.conf)

	// check for live gateways
	go func(cs *ConfigServer) {
		for {
			check := config.Get[time.Duration](cs.conf, "geneos.check-interval")
			if check == 0 {
				check = 60 * time.Second
			}

			time.Sleep(check)

			cs.Lock()
			cs.gateways = CheckGateways(cs.conf)
			cs.Unlock()
		}
	}(cs)

	// load inventories
	go func(cs *ConfigServer) {
		for {
			// update each time in case configuration has changed
			check := config.Get[time.Duration](cs.conf, "inventory.check-interval")
			if check == 0 {
				check = 60 * time.Second
			}
			time.Sleep(check)

			cs.Lock()
			cs.hosts, err = LoadHosts(cs.conf)
			cs.Unlock()
		}
	}(cs)

	cf := cs.conf

	if !cf.IsSet("server.config-path") {
		log.Fatal().Msg("no configuration path (`server.config-path`) set, exiting")
	}

	e.GET(config.Get[string](cf, "server.config-path")+"/:hostname", cs.ServeConfig)
	e.GET(config.Get[string](cf, "server.config-path")+"/:hostname/:type", cs.ServeConfig)

	if cf.IsSet("server.connections-path") {
		e.GET(config.Get[string](cf, "server.connections-path"), cs.ServeConnection)
	}

	// loop forever trying to listen on configured port
	for {
		if !config.Get[bool](cf, "server.tls.enable") {
			err = e.Start(fmt.Sprintf("%s:%d", config.Get[string](cf, "server.host"), config.Get[uint16](cf, "server.port")))
			if err != nil {
				log.Error().Err(err).Msg("retrying in 5 seconds")
				time.Sleep(5 * time.Second)
			}
			continue
		}

		// if we can decode the provided cert or key then pass them as
		// []byte to StartTLS to be used directly otherwise pass the
		// string in as a file path
		var cert, key any

		certstr := config.Get[string](cf, "server.tls.certificate")
		cert = []byte(certstr)

		if certpem, _ := pem.Decode([]byte(certstr)); certpem == nil {
			cert = config.ResolveHome(certstr)
		}

		keystr := config.Get[string](cf, "server.tls.privatekey")
		key = []byte(keystr)

		if keypem, _ := pem.Decode([]byte(keystr)); keypem == nil {
			key = config.ResolveHome(keystr)
		}

		listen := fmt.Sprintf("%s:%d", config.Get[string](cf, cf.Join("server", "host"), config.DefaultValue("0.0.0.0")), config.Get[uint16](cf, cf.Join("server", "port"), config.DefaultValue(6543)))
		err = e.StartTLS(listen, cert, key)

		if err != nil {
			log.Error().Err(err).Msg("retrying in 5 seconds")
			time.Sleep(5 * time.Second)
		}
	}
}

// ServeConfig is the main handler to return a SAN XML config
func (cs *ConfigServer) ServeConfig(c echo.Context) (err error) {
	hosttype := c.Param("type")
	hostname := c.Param("hostname")
	if hostname == "" {
		return c.String(http.StatusBadRequest, "hostname not given")
	}
	log.Debug().Msgf("serve: hostname %s type %s", hostname, hosttype)

	np, finalHosttype := cs.NetprobeConfig(hostname, hosttype)

	if len(np.SelfAnnounce.Gateways) == 0 {
		return echo.ErrInternalServerError
	}
	log.Info().Msgf("sending config for '%s' type '%s'", hostname, finalHosttype)
	return c.XMLPretty(http.StatusOK, np, "    ")
}

// ServeConnection supplies a connection file with all live gateways
//
// The output format is (from docs):
//
//		<hostname>~<port>~<description>~<Secondary Host>~<Secondary Port>~<Logon Method>~<Connection Security>
//	 thinkpad  ~7038  ~test1        ~ubuntu          ~7038            ~*             ~SECURE
//		...
//
// The description, Secondary Host and port and logon methods can be replaced with '*' for undefined
func (cs *ConfigServer) ServeConnection(c echo.Context) (err error) {
	var lines strings.Builder

	cs.Lock()
	gwlist := cs.gateways
	cs.Unlock()

	sort.Strings(gwlist)
	gwlist = slices.Compact(gwlist)

	// only lookup allGateways once
	allGateways := config.Get[[]map[string]string](cs.conf, "geneos.gateways")

	for _, gw := range gwlist {
		gateway := GatewayDetails(gw, allGateways)

		lines.WriteString(fmt.Sprintf("%s~%d~%s~", gateway.Primary, gateway.PrimaryPort, gw))
		if gateway.Standby != "" && gateway.StandbyPort != 0 {
			lines.WriteString(fmt.Sprintf("%s~%d~*~", gateway.Standby, gateway.StandbyPort))
		} else {
			lines.WriteString("~~*~")
		}
		if !gateway.Secure {
			lines.WriteString("LM_IN")
		}
		lines.WriteString("SECURE\n")
	}

	return c.String(http.StatusOK, lines.String())
}
