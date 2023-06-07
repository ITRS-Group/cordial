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
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/itrs-group/cordial/pkg/commands"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/xpath"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
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

	snapshotCmd.Flags().StringVarP(&snapshotCmdUsername, "username", "u", "", "Username")
	snapshotCmd.Flags().StringVarP(&snapshotCmdPwFile, "pwfile", "P", "", "Password")

	snapshotCmd.Flags().IntVarP(&snapshotCmdMaxitems, "limit", "l", 0, "limit matching items to display. default is unlimited. results unsorted.")
	snapshotCmd.Flags().BoolVarP(&snapshotCmdXpathsonly, "xpaths", "x", false, "just show matching xpaths")

	snapshotCmd.Flags().SortFlags = false
}

//go:embed _docs/snapshot.md
var snapshotCmdDescription string

var snapshotCmd = &cobra.Command{
	Use:          "snapshot [flags] [gateway] [NAME] XPATH...",
	Short:        "Capture a snapshot of each matching dataview",
	Long:         snapshotCmdDescription,
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
			snapshotCmdPassword, err = config.ReadPasswordInput(false, 0)
			if err == config.ErrNotInteractive {
				err = fmt.Errorf("%w and password required", err)
				return
			}
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
		// or credentials file
		username := c.Config().GetString(config.Join("snapshot", "username"))
		password := c.Config().GetPassword(config.Join("snapshot", "password"))

		if username == "" {
			username = snapshotCmdUsername
		}

		if password.IsNil() {
			password = snapshotCmdPassword
		}

		// if username is still unset then look for credentials
		if username == "" {
			creds := config.FindCreds(c.Type().String()+":"+c.Name(), config.SetAppName(Execname))
			if creds != nil {
				username = creds.GetString("username")
				password = creds.GetPassword("password")
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
	if c.Type().String() != "gateway" {
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
