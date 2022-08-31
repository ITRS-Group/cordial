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

	"github.com/itrs-group/cordial/pkg/commands"
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

var values, severities, snoozes, userAssignments bool

func init() {
	rootCmd.AddCommand(snapshotCmd)

	snapshotCmd.Flags().SortFlags = false
	snapshotCmd.Flags().BoolVarP(&values, "value", "V", true, "Request cell values")
	snapshotCmd.Flags().BoolVarP(&severities, "severity", "S", false, "Request cell severities")
	snapshotCmd.Flags().BoolVarP(&snoozes, "snooze", "Z", false, "Request cell snooze info")
	snapshotCmd.Flags().BoolVarP(&userAssignments, "userassignment", "U", false, "Request cell user assignment info")
}

func snapshotInstance(c geneos.Instance, params []string) (err error) {
	logDebug.Println("snapshot on", c)
	for _, path := range params {
		x, err := xpath.Parse(path)
		if err != nil {
			logError.Printf("%s: %q", err, path)
			continue
		}
		d := x.ResolveTo(&xpath.Dataview{})
		log.Println("dataview:", d)
		conn, err := commands.DialGateway(gatewayURL(c), commands.AllowInsecureCertificates(true), commands.SetBasicAuth("test", "abc123"))
		if err != nil {
			logError.Println(err)
			return err
		}
		views, err := conn.Match(d, 0)
		if err != nil {
			logError.Println(err)
			return err
		}
		for _, view := range views {
			data, err := conn.Snapshot(view)
			if err != nil {
				logError.Println(err)
				return err
			}
			j, _ := json.MarshalIndent(data, "", "    ")
			log.Printf("data: %s", string(j))
		}
	}
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
	port := c.GetConfig().GetInt("port")
	u.Host = fmt.Sprintf("%s:%d", host, port)
	u.Scheme = "http"
	if c.GetConfig().GetString("certificate") != "" {
		u.Scheme = "https"
	}
	return
}
