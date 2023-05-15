/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	dbg "runtime/debug"
	"strings"

	"github.com/itrs-group/cordial"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(versionCmd)

	// versionCmd.Flags().SortFlags = false
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show program version details",
	Long: strings.ReplaceAll(`
Show program version details
`, "|", "`"),
	SilenceUsage: true,
	Version:      cordial.VERSION,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s version %s\n", Execname, cmd.Version)
		if debug {
			info, ok := dbg.ReadBuildInfo()
			if ok {
				fmt.Println("additional info:")
				fmt.Println(info)
			}
		}
	},
}
