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
	"fmt"
	"net/url"
	"os"

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
var snapshotCmdPassword *config.Plaintext

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
	GroupID:      CommandGroupOther,
	Short:        "Capture a snapshot of each matching dataview",
	Long:         snapshotCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationComponent: "gateway",
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
	},
	Run: func(cmd *cobra.Command, _ []string) {
		var err error
		ct, names, params := TypeNamesParams(cmd)
		if len(params) == 0 {
			fmt.Printf("no dataview xpath(s) supplied")
			return
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
				fmt.Printf("not running interactive and password required")
				return
			}
		}

		// at this point snapshotCmdUsername/Password contain global or
		// command line values. These can be overridden per-instance.
		responses := instance.Do(geneos.GetHost(Hostname), ct, names, snapshotInstance, params)
		responses.Write(os.Stdout, instance.WriterIndent(true))
	},
}

func snapshotInstance(c geneos.Instance, params ...any) (resp *instance.Response) {
	resp = instance.NewResponse(c)

	if len(params) == 0 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	paths, ok := params[0].([]string)
	if !ok {
		panic("wrong type")
	}

	if !instance.AtLeastVersion(c, "5.14") {
		resp.Err = fmt.Errorf("%s is too old (5.14 or above required)", c)
		return
	}
	values := []any{}
	log.Debug().Msgf("snapshot on %s", c)
	for _, path := range paths {
		var x *xpath.XPath
		x, resp.Err = xpath.Parse(path)
		if resp.Err != nil {
			log.Error().Msgf("%s: %q", resp.Err, path)
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
		var gw *commands.Connection
		gw, resp.Err = commands.DialGateway(gatewayURL(c),
			commands.AllowInsecureCertificates(true),
			commands.SetBasicAuth(username, password))
		if resp.Err != nil {
			return
		}
		d := x.ResolveTo(&xpath.Dataview{})
		log.Debug().Msgf("matching xpath %s", d)
		var views []*xpath.XPath
		views, resp.Err = gw.Match(d, 0)
		if resp.Err != nil {
			return
		}
		if snapshotCmdMaxitems > 0 && len(views) > snapshotCmdMaxitems {
			views = views[0:snapshotCmdMaxitems]
		}
		if snapshotCmdXpathsonly {
			for _, x := range views {
				values = append(values, x)
			}
		} else {
			for _, view := range views {
				var data *commands.Dataview
				data, resp.Err = gw.Snapshot(view)
				if resp.Err != nil {
					return
				}
				values = append(values, data)
			}
		}
	}
	if len(values) > 0 {
		resp.Value = values
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
	if instance.FileOf(c, "certificate") != "" {
		u.Scheme = "https"
	}
	return
}
