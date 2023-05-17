/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cfgcmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

// ConfigCmd represents the config command
var ConfigCmd = &cobra.Command{
	Use:     "config",
	GroupID: cmd.GROUP_SUBSYSTEMS,
	Short:   "Configure geneos command environment",
	Long: strings.ReplaceAll(`
The commands in the |config| subsystem allow you to control the
environment of the |geneos| program itself. Please see the
descriptions of the commands below for more information.

If you run this command directly then you will either be shown the
output of |geneos config show| or |geneos config set [ARGS]| if you
supply any further arguments.
`, "|", "`"),
	Example: `
geneos config
geneos config geneos=/opt/itrs
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	DisableFlagParsing: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		if len(args) > 0 {
			return cmd.RunE(command.Root(), []string{"config", "set"}, args)
		}
		return cmd.RunE(command.Root(), []string{"config", "show"}, args)
	},
}

func init() {
	cmd.GeneosCmd.AddCommand(ConfigCmd)
}
