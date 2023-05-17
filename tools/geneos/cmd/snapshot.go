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

package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/awnumar/memguard"
	"github.com/itrs-group/cordial/pkg/commands"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/xpath"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var snapshotCmdValues, snapshotCmdSeverities, snapshotCmdSnoozes, snapshotCmdUserAssignments, snapshotCmdXpathsonly bool
var snapshotCmdMaxitems int
var snapshotCmdUsername, snapshotCmdPwFile string
var snapshotCmdPassword config.Plaintext

func init() {
	GeneosCmd.AddCommand(snapshotCmd)

	snapshotCmd.Flags().SortFlags = false
	snapshotCmd.Flags().BoolVarP(&snapshotCmdValues, "value", "V", true, "Request cell values")
	snapshotCmd.Flags().BoolVarP(&snapshotCmdSeverities, "severity", "S", false, "Request cell severities")
	snapshotCmd.Flags().BoolVarP(&snapshotCmdSnoozes, "snooze", "Z", false, "Request cell snooze info")
	snapshotCmd.Flags().BoolVarP(&snapshotCmdUserAssignments, "userassignment", "U", false, "Request cell user assignment info")

	snapshotCmd.Flags().StringVarP(&snapshotCmdUsername, "username", "u", "", "Username for snapshot, defaults to configuration value in snapshot.username")
	snapshotCmd.Flags().StringVarP(&snapshotCmdPwFile, "pwfile", "P", "", "Password file to read for snapshots, defaults to configuration value in snapshot.password or otherwise prompts")

	snapshotCmd.Flags().IntVarP(&snapshotCmdMaxitems, "limit", "l", 0, "limit matching items to display. default is unlimited. results unsorted.")
	snapshotCmd.Flags().BoolVarP(&snapshotCmdXpathsonly, "xpaths", "x", false, "just show matching xpaths")

	snapshotCmd.Flags().SortFlags = false
}

// snapshotCmd represents the snapshot command
var snapshotCmd = &cobra.Command{
	Use:   "snapshot [flags] [gateway] [NAME] XPATH...",
	Short: "Capture a snapshot of each matching dataview",
	Long: strings.ReplaceAll(`
Snapshot one or more dataviews using the REST Commands API endpoint
introduced in GA5.14. The TYPE, if given, must be |gateway|.

Authentication to the Gateway is through a combination of command
line flags and configuration parameters. If either of the parameters
|snapshot.username| or |snapshot.password| is defined for the Gateway
or globally then this is used as a default unless overridden on the
command line by the |-u| and |-P| options. The user is only prompted
for a password if it cannot be located in either of the previous
places.

The output is in JSON format as an array of dataviews, where each
dataview is in the format defined in the Gateway documentation at

	https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_commands_tr.html#fetch_dataviews

Flags to select which properties of data items are available: |-V|,
|-S|, |-Z|, |-U| for value, severity, snooze and user-assignment
respectively. If none is given then the default is to fetch values
only.

To help capture diagnostic information the |-x| option can be used to
capture matching xpaths without the dataview contents. |-l| can be
used to limit the number of dataviews (or xpaths) but the limit is
not applied in any defined order.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"ct":           "gateway",
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, args, params := CmdArgsParams(cmd)
		if len(params) == 0 {
			return fmt.Errorf("no dataview xpath(s) supplied")
		}

		if snapshotCmdUsername == "" {
			snapshotCmdUsername = config.GetString(config.Join("snapshot", "username"))
		}

		if snapshotCmdPwFile != "" {
			var sp []byte
			if sp, err = os.ReadFile(snapshotCmdPwFile); err != nil {
				return
			}
			snapshotCmdPassword = config.NewPlaintext(sp)
		} else {
			snapshotCmdPassword = config.GetPassword(config.Join("snapshot", "password"))
		}

		if snapshotCmdUsername != "" && (snapshotCmdPassword.IsNil() || snapshotCmdPassword.Size() == 0) {
			snapshotCmdPassword, _ = config.ReadPasswordInput(false, 0)
		}

		// at this point snapshotCmdUsername/Password contain global or
		// command line values. These can be overridden per-instance.
		return instance.ForAll(ct, Hostname, snapshotInstance, args, params)
	},
}

func snapshotInstance(c geneos.Instance, params []string) (err error) {
	if !instance.AtLeastVersion(c, "5.14") {
		return fmt.Errorf("%s is too old (5.14 or above required)", c)
	}
	dvs := []string{}
	log.Debug().Msgf("snapshot on %s", c)
	for _, path := range params {
		x, err := xpath.Parse(path)
		if err != nil {
			log.Error().Msgf("%s: %q", err, path)
			continue
		}

		// always use auth details in per-instance config, but if not
		// given use those from the command line or user/global config
		username, password := c.Config().GetString(config.Join("snapshot", "username")), c.Config().GetString(config.Join("snapshot", "password"))
		if username == "" {
			username = snapshotCmdUsername
		}

		if len(password) == 0 {
			password = snapshotCmdPassword.String()
		}

		// if username is still unset then look for credentials
		if username == "" {
			var pwb *memguard.Enclave
			creds := config.FindCreds(c.Type().String()+":"+c.Name(), config.SetAppName(Execname))
			if creds != nil {
				username = creds.GetString("username")
				password = fmt.Sprint(creds.GetPassword("password"))
			}

			if pwb != nil {
				pw, _ := pwb.Open()
				pws := config.ExpandLockedBuffer(pw.String())
				password = strings.Clone(pws.String())
				pw.Destroy()
				pws.Destroy()
			}
		}

		log.Debug().Msgf("dialling %s", gatewayURL(c))
		gw, err := commands.DialGateway(gatewayURL(c),
			commands.AllowInsecureCertificates(true),
			commands.SetBasicAuth(username, password))
		if err != nil {
			return err
		}
		d := x.ResolveTo(&xpath.Dataview{})
		log.Debug().Msgf("matching xpath %s", d)
		views, err := gw.Match(d, 0)
		if err != nil {
			return err
		}
		if snapshotCmdMaxitems > 0 && len(views) > snapshotCmdMaxitems {
			views = views[0:snapshotCmdMaxitems]
		}
		if snapshotCmdXpathsonly {
			for _, x := range views {
				dvs = append(dvs, fmt.Sprintf("%q", x))
			}
		} else {
			for _, view := range views {
				data, err := gw.Snapshot(view)
				if err != nil {
					return err
				}
				j, _ := json.MarshalIndent(data, "    ", "    ")
				dvs = append(dvs, string(j))
			}
		}
	}
	if len(dvs) > 0 {
		fmt.Printf("[\n    %s\n]\n", strings.Join(dvs, ",\n    "))
	}
	return
}

func gatewayURL(c geneos.Instance) (u *url.URL) {
	if c.Type() != &gateway.Gateway {
		return
	}
	u = &url.URL{}
	hostname := c.Host().GetString("hostname")
	if hostname == "" {
		hostname = "localhost"
	}
	port := c.Config().GetInt("port")
	u.Host = fmt.Sprintf("%s:%d", hostname, port)
	u.Scheme = "http"
	if instance.Filename(c, "certificate") != "" {
		u.Scheme = "https"
	}
	return
}
