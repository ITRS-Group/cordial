/*
Copyright © 2022 ITRS Group

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
	_ "embed"
	"fmt"
	"net/url"
	"os"

	"github.com/itrs-group/cordial"
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
	snapshotCmd.Flags().StringVarP(&snapshotCmdPwFile, "pwfile", "P", "", "File containing cleartext password (deprecated)")
	snapshotCmd.Flags().MarkHidden("pwfile")

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
		// CmdComponent:   "gateway",
		CmdGlobal:        "false",
		CmdRequireHome:   "true",
		CmdWildcardNames: "true",
	},
	Run: func(cmd *cobra.Command, _ []string) {
		var err error
		ct, names, params := ParseTypeNamesParams(cmd)
		if ct == nil {
			ct = geneos.ParseComponent("gateway")
		} else if !ct.IsA("gateway") {
			fmt.Println("snapshots are only valid for gateways")
			return
		}

		if len(names) == 0 {
			fmt.Println(`no gateway name(s) supplied. Use a NAME of "all" as an explicit wildcard`)
			return
		}
		if len(params) == 0 {
			fmt.Printf("no dataview xpath(s) supplied\n")
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

		instance.Do(geneos.GetHost(Hostname), ct, names, snapshotInstance, params).Write(os.Stdout, instance.WriterIndent(true))
	},
}

func snapshotInstance(i geneos.Instance, params ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	if len(params) == 0 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	paths, ok := params[0].([]string)
	if !ok {
		panic("wrong type")
	}

	if instance.CompareVersion(i, "5.14") <= 0 {
		resp.Err = fmt.Errorf("%s is too old (5.14 or above required)", i)
		return
	}
	values := []any{}
	log.Debug().Msgf("snapshot on %s", i)
	for _, path := range paths {
		var x *xpath.XPath
		x, resp.Err = xpath.Parse(path)
		if resp.Err != nil {
			log.Error().Msgf("%s: %q", resp.Err, path)
			continue
		}

		// always use auth details in per-instance config, default to
		// from the command line or user/global config or credentials
		// file
		username := i.Config().GetString(config.Join("snapshot", "username"))
		password := i.Config().GetPassword(config.Join("snapshot", "password"))

		if username == "" {
			username = snapshotCmdUsername
		}

		if password.IsNil() {
			password = snapshotCmdPassword
		}

		// if username is still unset then look for credentials
		//
		// credential domain is gateway:NAME or gateway:* for wildcard
		if username == "" {
			creds := config.FindCreds(i.Type().String()+":"+i.Name(), config.SetAppName(cordial.ExecutableName()))
			if creds != nil {
				username = creds.GetString("username")
				password = creds.GetPassword("password")
			} else {
				if creds = config.FindCreds(i.Type().String()+":*", config.SetAppName(cordial.ExecutableName())); creds != nil {
					username = creds.GetString("username")
					password = creds.GetPassword("password")
				}
			}
		}

		log.Debug().Msgf("dialling %s", gatewayURL(i))
		var gw *commands.Connection
		gw, resp.Err = commands.DialGateway(gatewayURL(i),
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
				data, resp.Err = gw.Snapshot(view, "")
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

func gatewayURL(i geneos.Instance) (u *url.URL) {
	if !instance.IsA(i, "gateway") {
		return
	}
	u = &url.URL{}
	hostname := i.Host().GetString("hostname")
	if hostname == "" {
		hostname = "localhost"
	}
	port := i.Config().GetInt("port")
	u.Host = fmt.Sprintf("%s:%d", hostname, port)
	u.Scheme = "http"
	if instance.FileOf(i, "certificate") != "" {
		u.Scheme = "https"
	}
	return
}
