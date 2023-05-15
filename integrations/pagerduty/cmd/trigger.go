/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// triggerCmd represents the trigger command
var triggerCmd = &cobra.Command{
	Use:   "trigger",
	Short: "Send a Pagerduty trigger event",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		return sendEvent(Trigger)
	},
}

func init() {
	RootCmd.AddCommand(triggerCmd)

}
