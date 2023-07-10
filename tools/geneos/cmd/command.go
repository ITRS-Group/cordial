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
	"encoding/json"
	"fmt"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"

	"github.com/spf13/cobra"
)

var commandCmdJSON bool

func init() {
	GeneosCmd.AddCommand(commandCmd)

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
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := CmdArgsParams(cmd)
		if commandCmdJSON {
			_, results, err := instance.ForAllWithResults(ct, Hostname, commandInstanceJSON, args, params)
			if err != nil {
				return err
			}
			b, err := json.MarshalIndent(results, "", "    ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}
		return instance.ForAll(ct, Hostname, commandInstance, args, params)
	},
}

func commandInstance(c geneos.Instance, params []string) (err error) {
	fmt.Printf("=== %s ===\n", c)
	cmd, env, home := instance.BuildCmd(c, true)
	if cmd != nil {
		fmt.Println("command line:")
		fmt.Println("\t", cmd.String())
		fmt.Println()
		fmt.Println("working directory:")
		fmt.Println("\t", home)
		fmt.Println()
		fmt.Println("environment:")
		for _, e := range env {
			fmt.Println("\t", e)
		}
		fmt.Println()
	}
	return
}

func commandInstanceJSON(c geneos.Instance, params []string) (result interface{}, err error) {
	cmd, env, home := instance.BuildCmd(c, true)
	command := &command{
		Instance: c.Name(),
		Type:     c.Type().Name,
		Host:     c.Host().String(),
		Path:     cmd.Path,
		Env:      env,
		Home:     home,
	}
	if len(cmd.Args) > 1 {
		command.Args = cmd.Args[1:]
	}
	return command, nil
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
