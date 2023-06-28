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
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

type showCmdInstanceConfig struct {
	Name      string `json:"name,omitempty"`
	Host      string `json:"host,omitempty"`
	Type      string `json:"type,omitempty"`
	Disabled  bool   `json:"disabled"`
	Protected bool   `json:"protected"`
}

type showCmdConfig struct {
	Instance      showCmdInstanceConfig `json:"instance"`
	Configuration interface{}           `json:"configuration,omitempty"`
}

var showCmdRaw, showCmdSetup, showCmdMerge, showCmdValidate bool
var showCmdOutput, showCmdHooksDir string

func init() {
	GeneosCmd.AddCommand(showCmd)

	showCmd.Flags().StringVarP(&showCmdOutput, "output", "o", "", "Output file, default stdout")

	showCmd.Flags().BoolVarP(&showCmdRaw, "raw", "r", false, "Show raw (unexpanded) configuration values")

	showCmd.Flags().BoolVarP(&showCmdSetup, "setup", "s", false, "Show the instance Geneos configuration file, if any")
	showCmd.Flags().BoolVarP(&showCmdMerge, "merge", "m", false, "Merge Gateway configurations using the Gateway -dump-xml flag")

	showCmd.Flags().BoolVarP(&showCmdValidate, "validate", "v", false, "Validate Gateway configurations using the Gateway -validate flag")
	showCmd.Flags().StringVar(&showCmdHooksDir, "hooks", "", "Hooks directory (may clash with instance options)")

	showCmd.Flags().SortFlags = false
}

//go:embed _docs/show.md
var showCmdDescription string

var showCmd = &cobra.Command{
	Use:          "show [flags] [TYPE] [NAME...]",
	GroupID:      CommandGroupView,
	Short:        "Show Instance Configuration",
	Long:         showCmdDescription,
	Aliases:      []string{"details"},
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ct, args, params := CmdArgsParams(cmd)
		output := os.Stdout
		if showCmdOutput != "" {
			output, err = os.Create(showCmdOutput)
			if err != nil {
				return
			}
		}

		if showCmdMerge && showCmdValidate {
			return errors.New("cannot validate and merge at the same time")
		}

		if showCmdMerge {
			showCmdSetup = true
		}

		if showCmdValidate {
			if showCmdHooksDir != "" {
				params = append(params, showCmdHooksDir)
			}

			var results []interface{}
			results, err = instance.ForAllWithResults(ct, Hostname, showValidateInstance, args, params)

			if err != nil {
				if err == os.ErrNotExist {
					return fmt.Errorf("no matching instance found")
				}
			}
			for _, r := range results {
				result, ok := r.(showConfig)
				if !ok {
					return
				}
				fmt.Fprintf(output, "### validation results for %s\n\n%s\n\n", result.c, result.file)
			}
			return
		}

		if showCmdSetup {
			params = []string{}
			if showCmdMerge {
				params = append(params, "merge")
			}
			var results []interface{}
			results, err = instance.ForAllWithResults(ct, Hostname, showInstanceConfig, args, params)

			if err != nil {
				if err == os.ErrNotExist {
					return fmt.Errorf("no matching instance found")
				}
			}
			for _, r := range results {
				result, ok := r.(showConfig)
				if !ok {
					return
				}
				fmt.Fprintf(output, "<!-- configuration for %s -->\n\n%s\n\n", result.c, result.file)
			}
			return
		}
		results, err := instance.ForAllWithResults(ct, Hostname, showInstance, args, params)
		if err != nil {
			if err == os.ErrNotExist {
				return fmt.Errorf("no matching instance found")
			}
			return
		}
		b, _ := json.MarshalIndent(results, "", "    ")
		fmt.Fprintln(output, string(b))
		return
	},
}

func showValidateInstance(c geneos.Instance, params []string) (result interface{}, err error) {
	setup := c.Config().GetString("setup")
	if setup == "" {
		return
	}
	if c.Type().String() == "gateway" {
		// temp file for JSON output
		tempfile := filepath.Join(c.Host().TempDir(), "validate-"+c.Name()+".json")
		defer c.Host().Remove(tempfile)

		// run a gateway with -dump-xml and consume the result, discard the heading
		cmd, env, home := instance.BuildCmd(c)
		// replace args with a more limited set
		cmd.Args = []string{
			cmd.Path,
			"-resources-dir",
			path.Join(instance.BaseVersion(c), "resources"),
			"-nolog",
			"-skip-cache",
			"-setup",
			c.Config().GetString("setup"),
			"-validate-json-output",
			tempfile,
			"-silent",
			"-hub-validation-rules",
		}
		cmd.Args = append(cmd.Args, instance.SetSecureArgs(c)...)
		if len(params) > 0 {
			cmd.Args = append(cmd.Args, "-hooks-dir", params[0])
		}

		var output []byte
		// we don't care about errors, just the output
		_, err = c.Host().Run(cmd, env, home, "errors.txt")
		if err != nil {
			log.Debug().Msgf("error: %s", output)
		}
		output, err = os.ReadFile(tempfile)
		if err != nil {
			return
		}

		result = showConfig{
			c:    c,
			file: output,
		}
		return
	}
	return
}

type showConfig struct {
	c    geneos.Instance
	file []byte
}

// showInstanceConfig returns a slice of showConfig structs per instance
func showInstanceConfig(c geneos.Instance, params []string) (result interface{}, err error) {
	setup := c.Config().GetString("setup")
	if setup == "" {
		return
	}
	if c.Type().String() == "gateway" && len(params) > 0 && params[0] == "merge" {
		// run a gateway with -dump-xml and consume the result, discard the heading
		cmd, env, home := instance.BuildCmd(c)
		// replace args with a more limited set
		cmd.Args = []string{
			cmd.Path,
			"-resources-dir",
			path.Join(instance.BaseVersion(c), "resources"),
			"-nolog",
			"-skip-cache",
			"-setup",
			c.Config().GetString("setup"),
			"-dump-xml",
		}
		cmd.Args = append(cmd.Args, instance.SetSecureArgs(c)...)
		var output []byte
		// we don't care about errors, just the output
		output, err = c.Host().Run(cmd, env, home, "errors.txt")
		if err != nil {
			log.Debug().Msgf("error: %s", output)
		}
		i := bytes.Index(output, []byte("<?xml"))
		if i == -1 {
			return
		}
		result = showConfig{
			c:    c,
			file: output[i:],
		}
		return
	}
	file, err := os.ReadFile(setup)
	if err != nil {
		return
	}
	result = showConfig{
		c:    c,
		file: file,
	}

	return
}

func showInstance(c geneos.Instance, params []string) (result interface{}, err error) {
	// remove aliases
	nv := config.New()
	aliases := c.Type().Aliases
	for _, k := range c.Config().AllKeys() {
		// skip any names in the alias table
		log.Debug().Msgf("checking %s", k)
		if _, ok := aliases[k]; !ok {
			log.Debug().Msgf("setting %s", k)
			nv.Set(k, c.Config().Get(k))
		}
	}

	as := nv.ExpandAllSettings(config.NoDecode())
	if showCmdRaw {
		as = nv.AllSettings()
	}
	cf := &showCmdConfig{
		Instance: showCmdInstanceConfig{
			Name:      c.Name(),
			Host:      c.Host().String(),
			Type:      c.Type().String(),
			Disabled:  instance.IsDisabled(c),
			Protected: instance.IsProtected(c),
		},
		Configuration: as,
	}

	result = cf
	return
}
