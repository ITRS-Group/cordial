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
	"os"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

type showCmdConfig struct {
	Name      string      `json:"name,omitempty"`
	Host      string      `json:"host,omitempty"`
	Type      string      `json:"type,omitempty"`
	Disabled  bool        `json:"disabled"`
	Protected bool        `json:"protected"`
	Config    interface{} `json:"config,omitempty"`
}

var showCmdRaw bool

func init() {
	RootCmd.AddCommand(showCmd)

	showCmd.Flags().BoolVarP(&showCmdRaw, "raw", "r", false, "Show raw (unexpanded) configuration values")

	showCmd.Flags().SortFlags = false
}

var showCmd = &cobra.Command{
	Use:   "show [flags] [TYPE] [NAME...]",
	Short: "Show runtime, global, user or instance configuration is JSON format",
	Long: strings.ReplaceAll(`
Show the runtime or instance configuration. The loaded
global or user configurations can be seen through the show global
and show user sub-commands, respectively.

With no arguments show the full runtime configuration that
results from environment variables, loading built-in defaults and the
global and user configurations.

If a component TYPE and/or instance NAME(s) are given then the
configuration for those instances are output as JSON. This is
regardless of the instance using a legacy .rc file or a native JSON
configuration.

Passwords and secrets are redacted in a very simplistic manner simply
to prevent visibility in casual viewing.
`, "|", "`"),
	Aliases:      []string{"details"},
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) == 0 {
			// running config
			rc := config.GetConfig().ExpandAllSettings()
			if showCmdRaw {
				rc = config.GetConfig().AllSettings()
			}
			j, _ := json.MarshalIndent(rc, "", "    ")
			fmt.Println(string(j))
			return nil
		}

		ct, args, params := CmdArgsParams(cmd)
		results, err := instance.ForAllWithResults(ct, showInstance, args, params)
		if err != nil {
			if err == os.ErrNotExist {
				return fmt.Errorf("no matching instance found")
			}
			return
		}
		b, _ := json.MarshalIndent(results, "", "    ")
		fmt.Println(string(b))
		return
	},
}

func showInstance(c geneos.Instance, params []string) (result interface{}, err error) {
	// remove aliases
	nv := config.New()
	for _, k := range c.Config().AllKeys() {
		if _, ok := c.Type().Aliases[k]; !ok {
			nv.Set(k, c.Config().Get(k))
		}
	}

	// XXX wrap in location and type
	as := nv.ExpandAllSettings()
	if showCmdRaw {
		as = nv.AllSettings()
	}
	cf := &showCmdConfig{
		Name:      c.Name(),
		Host:      c.Host().String(),
		Type:      c.Type().String(),
		Disabled:  instance.IsDisabled(c),
		Protected: instance.IsProtected(c),
		Config:    as,
	}

	result = cf
	return
}
