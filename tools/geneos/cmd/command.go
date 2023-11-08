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
	_ "embed"
	"fmt"
	"os"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"

	"github.com/spf13/cobra"
)

var commandCmdJSON bool
var commandCmdExtras string
var commandCmdEnvs instance.NameValues

func init() {
	GeneosCmd.AddCommand(commandCmd)

	commandCmd.Flags().StringVarP(&commandCmdExtras, "extras", "x", "", "Extra args passed to process, split on spaces and quoting ignored")
	commandCmd.Flags().VarP(&commandCmdEnvs, "env", "e", "Extra environment variable (Repeat as required)")
	commandCmd.Flags().BoolVarP(&commandCmdJSON, "json", "j", false, "JSON formatted output")
}

//go:embed _docs/command.md
var commandCmdDescription string

var commandCmd = &cobra.Command{
	Use:          "command [TYPE] [NAME...]",
	GroupID:      CommandGroupView,
	Short:        "Show Instance Start-up Details",
	Long:         commandCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
		AnnotationExpand:    "true",
	},
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, names := ParseTypeNames(cmd)
		if commandCmdJSON {
			results := instance.Do(geneos.GetHost(Hostname), ct, names, commandInstanceJSON)
			results.Write(os.Stdout, instance.WriterIndent(true))
			return nil
		}
		results := instance.Do(geneos.GetHost(Hostname), ct, names, commandInstance)
		results.Write(os.Stdout)
		return
	},
}

func commandInstance(i geneos.Instance, params ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	lines := []string{fmt.Sprintf("=== %s ===", i)}

	cmd, env, home := instance.BuildCmd(i, true, instance.StartingExtras(commandCmdExtras), instance.StartingEnvs(commandCmdEnvs))
	lines = append(lines,
		"command line:",
		fmt.Sprint("\t", cmd.String()),
		"",
		"working directory:",
		fmt.Sprint("\t", home),
		"",
		"environment:",
	)

	for _, e := range env {
		lines = append(lines, fmt.Sprint("\t", e))
	}
	lines = append(lines, "")
	resp.Lines = lines
	return
}

type command struct {
	Instance string   `json:"instance"`
	Type     string   `json:"type"`
	Host     string   `json:"host"`
	Path     string   `json:"path"`
	Args     []string `json:"arguments,omitempty"`
	Env      []string `json:"environment,omitempty"`
	Home     string   `json:"directory"`
}

func commandInstanceJSON(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	cmd, env, home := instance.BuildCmd(i, true, instance.StartingExtras(commandCmdExtras), instance.StartingEnvs(commandCmdEnvs))
	command := &command{
		Instance: i.Name(),
		Type:     i.Type().Name,
		Host:     i.Host().String(),
		Path:     cmd.Path,
		Env:      env,
		Home:     home,
	}
	if len(cmd.Args) > 1 {
		command.Args = cmd.Args[1:]
	}
	resp.Value = command
	return
}
