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

package client

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/pkg/rest"

	"github.com/itrs-group/cordial/integrations/servicenow2/cmd"
	"github.com/itrs-group/cordial/integrations/servicenow2/cmd/proxy"
)

// flags
var clientCmdProfile, clientCmdTable string
var clientCmdQuiet bool

var log = logger.Logger

func init() {
	cmd.Cmd.AddCommand(clientCmd)

	clientCmd.Flags().StringVarP(&clientCmdProfile, "profile", "p", "", "profile to use for field creation")
	clientCmd.Flags().StringVarP(&clientCmdTable, "table", "t", "", "servicenow table, defaults typically to incident")
	clientCmd.Flags().BoolVarP(&clientCmdQuiet, "quiet", "q", false, "quiet mode. supress all non-error messages")

	clientCmd.Flags().SortFlags = false
}

var clientCmd = &cobra.Command{
	Use:   "client [FLAGS] [field=value ...]",
	Short: "Create or update a ServiceNow incident",
	Long: strings.ReplaceAll(`
	Create or update a ServiceNow incident from ITRS Geneos.
	
	This command is the client-side of the ITRS Geneos to ServiceNow
	incident integration. The program takes command line flags, arguments
	and environment variables to create a submission to the proxy
	which is responsible for sending the request to the
	ServiceNow API.

	The exit code is zero only if a proxy accepted the request and
	reported that the incident was created, updated or deliberately
	ignored. In all other cases an error is written to standard error
	and the exit code is non-zero.
	`, "|", "`"),
	SilenceUsage: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		cf, err := cmd.LoadConfigFile("client")
		if err != nil {
			return err
		}

		return submit(command.Context(), cf, submitOptions{
			profile: clientCmdProfile,
			table:   clientCmdTable,
			quiet:   clientCmdQuiet,
		}, args, os.Stdout)
	},
}

type submitOptions struct {
	profile string
	table   string
	quiet   bool
}

// submit builds an incident from the configuration, environment and
// arguments and sends it to the first proxy that gives an answer. It
// returns an error unless a proxy reports that the incident was
// created, updated or deliberately ignored.
//
// The `defaults` and profile action groups, and the expansion functions
// available to them, are those of [ims.ProcessActionGroup].
//
// All keys with a leading "_" are passed to the proxy but the proxy
// then removes them in addition to other configuration settings. The
// expected fields are:
//
// correlation_id - correlation ID, which is left unchanged before use, or if not defined,
// _correlation_id - correlation ID, which is SHA1 checksummed before use
//
// cmdb_ci or
// _cmdb_search - search query for cmdb_ci sys_id - or
// _cmdb_ci_default
//
// _subject - short description
// _text - long text
func submit(ctx context.Context, cf *config.Config, opts submitOptions, args []string, out io.Writer) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	incident := make(ims.Values)

	if err = ims.BuildIncident(cf, incident, opts.profile, proxy.PROFILE); err != nil {
		return err
	}

	// command line args can replace defaults and config file settings.
	// parse key value pairs as fields for the request
	for _, arg := range args {
		k, v, found := strings.Cut(arg, "=")
		if !found || k == "" {
			log.Warn("ignoring argument that is not in the form field=value", slog.String("argument", arg))
			continue
		}
		if v == "" {
			delete(incident, k)
		} else {
			incident[k] = v
		}
	}

	// check correlation ID, prefer a "raw" ID
	if _, ok := incident[proxy.SNOW_CORRELATION_ID]; !ok {
		if id, ok := incident[proxy.RAW_CORRELATION_ID]; ok {
			incident[proxy.SNOW_CORRELATION_ID] = fmt.Sprintf("%x", sha1.Sum([]byte(id)))
		}
	}
	// drop internal string either way
	delete(incident, proxy.RAW_CORRELATION_ID)

	table := opts.table
	if table == "" {
		var ok bool
		if table, ok = incident[proxy.INCIDENT_TABLE]; !ok {
			table = proxy.SNOW_INCIDENT_TABLE
		}
	}

	var result proxy.Response
	err = ims.Try(cf.Sub("proxy"), "", func(rc *rest.Client) error {
		result = proxy.Response{}
		if _, err := rc.Post(ctx, table, incident, &result); err != nil {
			return err
		}
		return checkResponse(result)
	})
	if err != nil {
		return noProxyError(err)
	}

	if !opts.quiet {
		fmt.Fprintln(out, result.Result)
	}

	return nil
}

// noProxyError adds the name of the configuration key to the error if
// there are no proxies configured
func noProxyError(err error) error {
	if errors.Is(err, ims.ErrNoURL) {
		return fmt.Errorf("no proxy URL configured (%s)", config.Join("proxy", "url"))
	}
	return err
}

// checkResponse validates a response, with a successful HTTP status,
// from the proxy. Anything other than a clear indication that the
// request was handled is an error.
func checkResponse(r proxy.Response) error {
	switch r.Action {
	case proxy.Created, proxy.Updated, proxy.Ignored:
		return nil
	case proxy.Failed:
		msg := r.Error
		if msg == "" {
			msg = r.Result
		}
		return fmt.Errorf("proxy reported failure: %s", msg)
	default:
		return fmt.Errorf("unexpected response from proxy (action %q)", r.Action)
	}
}
