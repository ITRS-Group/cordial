/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
)

var logoutCmdAll bool

func init() {
	GeneosCmd.AddCommand(logoutCmd)

	logoutCmd.Flags().BoolVarP(&logoutCmdAll, "all", "A", false, "remove all credentials")
}

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:     "logout [flags] [NAME...]",
	GroupID: GROUP_CREDENTIALS,
	Short:   "Logout (remove credentials)",
	Long: strings.ReplaceAll(`
The logout command removes the credentials for the names given. If no
names are set then the default credentials are removed.

If the |-A| options is given then all credentials are removed.
`, "|", "`"),
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	Run: func(cmd *cobra.Command, args []string) {
		if logoutCmdAll {
			config.DeleteAllCreds(config.SetAppName(Execname))
			return
		}
		if len(args) == 0 {
			for _, d := range args {
				config.DeleteCreds(d, config.SetAppName(Execname))
			}
		}
	},
}
