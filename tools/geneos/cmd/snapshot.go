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
	"strings"

	"github.com/itrs-group/cordial/pkg/commands"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/xpath"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/spf13/cobra"
)

// snapshotCmd represents the snapshot command
var snapshotCmd = &cobra.Command{
	Use:   "snapshot [gateway] [NAME] XPATH [XPATH...]",
	Short: "Capture a snapshot of each matching dataview",
	Long: `Using the Dataview Snapshot REST endpoint in GA5.14+ Gateways,
capture each dataview matching to given XPATH(s). Options to select
what data to request and authentication.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"ct":       "gateway",
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return instance.ForAll(ct, snapshotInstance, args, params)
	},
}

var values, severities, snoozes, userAssignments, xpathsonly bool
var maxitems int

func init() {
	rootCmd.AddCommand(snapshotCmd)

	snapshotCmd.Flags().SortFlags = false
	snapshotCmd.Flags().BoolVarP(&values, "value", "V", true, "Request cell values")
	snapshotCmd.Flags().BoolVarP(&severities, "severity", "S", false, "Request cell severities")
	snapshotCmd.Flags().BoolVarP(&snoozes, "snooze", "Z", false, "Request cell snooze info")
	snapshotCmd.Flags().BoolVarP(&userAssignments, "userassignment", "U", false, "Request cell user assignment info")

	snapshotCmd.Flags().IntVarP(&maxitems, "limit", "l", 0, "limit matching items to display. default is unlimited. results unsorted.")
	snapshotCmd.Flags().BoolVarP(&xpathsonly, "xpaths", "x", false, "just show matching xpaths")
}

func snapshotInstance(c geneos.Instance, params []string) (err error) {
	dvs := []string{}
	logDebug.Println("snapshot on", c)
	for _, path := range params {
		x, err := xpath.Parse(path)
		if err != nil {
			logError.Printf("%s: %q", err, path)
			continue
		}
		logDebug.Println("dialling", gatewayURL(c))
		gw, err := commands.DialGateway(gatewayURL(c),
			commands.AllowInsecureCertificates(true),
			commands.SetBasicAuth(config.GetString("download.username"), config.GetString("download.password")))
		if err != nil {
			return err
		}
		d := x.ResolveTo(&xpath.Dataview{})
		logDebug.Println("matching xpath", d)
		views, err := gw.Match(d, 0)
		if err != nil {
			return err
		}
		if maxitems > 0 && len(views) > maxitems {
			views = views[0:maxitems]
		}
		if xpathsonly {
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
	log.Printf("[\n    %s\n]\n", strings.Join(dvs, ",\n    "))
	return
}

func gatewayURL(c geneos.Instance) (u *url.URL) {
	if c.Type() != &gateway.Gateway {
		return
	}
	u = &url.URL{}
	host := c.Host().GetString("hostname")
	if host == "" {
		host = "localhost"
	}
	port := c.Config().GetInt("port")
	u.Host = fmt.Sprintf("%s:%d", host, port)
	u.Scheme = "http"
	if instance.Filename(c, "certificate") != "" {
		u.Scheme = "https"
	}
	return
}
