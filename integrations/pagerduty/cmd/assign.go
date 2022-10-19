/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

// assignCmd represents the assign command
var assignCmd = &cobra.Command{
	Use:   "assign",
	Short: "Send a Pagerduty trigger event",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	RunE: func(cmd *cobra.Command, args []string) error {
		return sendEvent(Assign)
	},
}

func init() {
	rootCmd.AddCommand(assignCmd)

}
