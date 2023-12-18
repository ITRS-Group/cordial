/*
Copyright © 2023 ITRS Group

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
