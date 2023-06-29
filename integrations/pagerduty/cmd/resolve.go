/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// resolveCmd represents the resolve command
var resolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Send a Pagerduty resolve event",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		return sendEvent(Resolve)
	},
}

func init() {
	RootCmd.AddCommand(resolveCmd)

}
