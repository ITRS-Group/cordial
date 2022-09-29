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
	"regexp"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show runtime, global, user or instance configuration is JSON format",
	Long: `Show the runtime or instance configuration. The loaded
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
to prevent visibility in casual viewing.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) == 0 {
			// running config
			rc := config.GetConfig().ExpandAllSettings()
			if showCmdRaw {
				rc = config.GetConfig().AllSettings()
			}
			j, _ := json.MarshalIndent(rc, "", "    ")
			j = opaqueJSONSecrets(j)
			fmt.Println(string(j))
			return nil
		}

		return commandShow(cmdArgsParams(cmd))
	},
}

var showCmdRaw bool

func init() {
	rootCmd.AddCommand(showCmd)

	showCmd.Flags().BoolVarP(&showCmdRaw, "raw", "r", false, "Show raw (unexpanded) configuration values")
	showCmd.Flags().SortFlags = false
}

// var showCmdYAML bool

func commandShow(ct *geneos.Component, args []string, params []string) (err error) {
	return instance.ForAll(ct, showInstance, args, params)
}

type showCmdConfig struct {
	Name   string      `json:"name,omitempty"`
	Host   string      `json:"host,omitempty"`
	Type   string      `json:"type,omitempty"`
	Config interface{} `json:"config,omitempty"`
}

func showInstance(c geneos.Instance, params []string) (err error) {
	var buffer []byte

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
	cf := &showCmdConfig{Name: c.Name(), Host: c.Host().String(), Type: c.Type().String(), Config: as}

	if buffer, err = json.MarshalIndent(cf, "", "    "); err != nil {
		return
	}
	buffer = opaqueJSONSecrets(buffer)
	fmt.Println(string(buffer))

	return
}

// XXX redact passwords - any field matching some regexp ?
var red1 = regexp.MustCompile(`"(.*((?i)pass|password|secret))": "(.*)"`)
var red2 = regexp.MustCompile(`"(.*((?i)pass|password|secret))=(.*)"`)

func opaqueJSONSecrets(j []byte) []byte {
	// simple redact - and left field with "Pass" in it gets the right replaced
	j = red1.ReplaceAll(j, []byte(`"$1": "********"`))
	j = red2.ReplaceAll(j, []byte(`"$1=********"`))
	return j
}
