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
	"errors"
	"fmt"
	"os"
	"path"

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

	showCmd.Flags().BoolVarP(&showCmdValidate, "validate", "V", false, "Validate Gateway configurations using the Gateway -validate flag")
	showCmd.Flags().StringVar(&showCmdHooksDir, "hooks", "", "Hooks directory\n(may clash with instance parameters if set for normal execution)")

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
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
		AnnotationExpand:    "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, names := ParseTypeNames(command)
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
			instance.Do(geneos.GetHost(Hostname), ct, names, showValidateInstance, showCmdHooksDir).Write(output,
				instance.WriterPrefix("### validation results for %s\n\n"),
				instance.WriterSuffix("\n\n"),
				instance.WriterPlainValue(),
			)
			return
		}

		if showCmdSetup {
			instance.Do(geneos.GetHost(Hostname), ct, names, showInstanceConfig, showCmdMerge).Write(output,
				instance.WriterPrefix("<!-- configuration for %s -->\n\n"),
				instance.WriterSuffix("\n\n"),
				instance.WriterPlainValue(),
			)
			return
		}
		instance.Do(geneos.GetHost(Hostname), ct, names, showInstance).Write(os.Stdout, instance.WriterIndent(true))
		return
	},
}

func showValidateInstance(i geneos.Instance, params ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	setup := i.Config().GetString("setup")
	if setup == "" {
		return
	}
	if instance.IsA(i, "gateway") {
		// temp file for JSON output
		tempfile := path.Join(i.Host().TempDir(), "validate-"+i.Name()+".json")
		defer i.Host().Remove(tempfile)

		// run a gateway with -dump-xml and consume the result, discard the heading
		cmd, env, home := instance.BuildCmd(i, false)
		// replace args with a more limited set
		cmd.Args = []string{
			cmd.Path,
			"-resources-dir",
			path.Join(instance.BaseVersion(i), "resources"),
			"-nolog",
			"-skip-cache",
			"-setup",
			i.Config().GetString("setup"),
			"-validate-json-output",
			tempfile,
			"-silent",
			"-hub-validation-rules",
		}
		cmd.Args = append(cmd.Args, instance.SetSecureArgs(i)...)
		if len(params) > 0 {
			cmd.Args = append(cmd.Args, fmt.Sprintf("-hooks-dir %s", params[0]))
		}

		var output []byte
		// we don't care about errors, just the output
		_, err := i.Host().Run(cmd, env, home, "errors.txt")
		if err != nil {
			log.Debug().Msgf("error: %s", output)
		}
		output, resp.Err = os.ReadFile(tempfile)
		if resp.Err != nil {
			return
		}

		resp.Value = output
		return
	}
	return
}

// showInstanceConfig returns a slice of showConfig structs per instance
func showInstanceConfig(i geneos.Instance, params ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	if len(params) == 0 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	merge, ok := params[0].(bool)
	if !ok {
		panic("wrong type")
	}

	setup := i.Config().GetString("setup")
	if setup == "" {
		return
	}
	if instance.IsA(i, "gateway") && merge {
		// run a gateway with -dump-xml and consume the result, discard the heading
		cmd, env, home := instance.BuildCmd(i, false)
		// replace args with a more limited set
		cmd.Args = []string{
			cmd.Path,
			"-resources-dir",
			path.Join(instance.BaseVersion(i), "resources"),
			"-nolog",
			"-skip-cache",
			"-setup",
			i.Config().GetString("setup"),
			"-dump-xml",
		}
		cmd.Args = append(cmd.Args, instance.SetSecureArgs(i)...)
		var output []byte
		// we don't care about errors, just the output
		output, err := i.Host().Run(cmd, env, home, "errors.txt")
		if err != nil {
			log.Debug().Msgf("error: %s", output)
		}
		idx := bytes.Index(output, []byte("<?xml"))
		if idx == -1 {
			return
		}
		resp.Value = output[idx:]
		return
	}
	file, err := i.Host().ReadFile(setup)
	if err != nil {
		resp.Err = err
		return
	}

	resp.Value = file
	return
}

func showInstance(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	// remove aliases
	nv := config.New()
	aliases := i.Type().LegacyParameters
	for _, k := range i.Config().AllKeys() {
		// skip any names in the alias table
		log.Debug().Msgf("checking %s", k)
		if _, ok := aliases[k]; !ok {
			log.Debug().Msgf("setting %s", k)
			nv.Set(k, i.Config().Get(k))
		}
	}

	as := nv.ExpandAllSettings(config.NoDecode(true))
	if showCmdRaw {
		as = nv.AllSettings()
	}
	cf := &showCmdConfig{
		Instance: showCmdInstanceConfig{
			Name:      i.Name(),
			Host:      i.Host().String(),
			Type:      i.Type().String(),
			Disabled:  instance.IsDisabled(i),
			Protected: instance.IsProtected(i),
		},
		Configuration: as,
	}

	resp.Value = cf
	return
}
