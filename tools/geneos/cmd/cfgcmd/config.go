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

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure geneos command environment",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("config called")
	},
}

func init() {
	cmd.RootCmd.AddCommand(configCmd)
}
