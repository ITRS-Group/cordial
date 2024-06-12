/*
Copyright © 2023 ITRS Group

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
	"errors"
	"net/url"
	"strconv"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/xmlrpc"
	"github.com/spf13/cobra"
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push file data using either XML or REST APIs",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		dataviews := []*config.Config{}
		for i := 0; ; i++ {
			pi := config.Join("dataviews", strconv.Itoa(i))
			if !cf.IsSet(pi) {
				break
			}
			dataviews = append(dataviews, cf.Sub(pi))
		}

		if len(dataviews) == 0 {
			return
		}

		gurl, err := url.Parse(cf.GetString("push.netprobe"))
		if err != nil {
			return err
		}
		client := xmlrpc.NewClient(gurl, xmlrpc.Secure(cf.GetBool("push.secure")))
		if !client.Connected() {
			return errors.New("gateway not connected to netprobe")
		}

		return
	},
}

func init() {
	FILE2DVCmd.AddCommand(pushCmd)
}
