/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cfgcmd

import (
	"fmt"
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
`, "|", "`"),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("config called")
	},
}

func init() {
	cmd.GeneosCmd.AddCommand(ConfigCmd)
}
