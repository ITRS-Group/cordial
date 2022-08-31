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
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

// commandCmd represents the command command
var commandCmd = &cobra.Command{
	Use:   "command [TYPE] [NAME...]",
	Short: "Show command line and environment for launching instances",
	Long: `Show the command line for the matching instances
along with any environment variables explicitly set for
execution.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return instance.ForAll(ct, commandInstance, args, params)
	},
}

func init() {
	rootCmd.AddCommand(commandCmd)
	commandCmd.Flags().SortFlags = false
}

func commandInstance(c geneos.Instance, params []string) (err error) {
	log.Printf("=== %s ===", c)
	cmd, env := instance.BuildCmd(c)
	if cmd != nil {
		log.Println("command line:")
		log.Println("\t", cmd.String())
		log.Println()
		log.Println("environment:")
		for _, e := range env {
			log.Println("\t", e)
		}
		log.Println()
	}
	return
}
