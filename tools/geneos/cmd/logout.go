/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
)

func init() {
	RootCmd.AddCommand(logoutCmd)
}

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout (remove credentials)",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	Run: func(cmd *cobra.Command, args []string) {
		config.DeleteCreds("x", config.SetAppName(Execname))
		fmt.Println("logout called")
	},
}
