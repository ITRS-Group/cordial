/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// protectCmd represents the protect command
var protectCmd = &cobra.Command{
	Use:   "protect",
	Short: "Mark instances as protected",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		fmt.Println("protect called")
		return
	},
}

func init() {
	RootCmd.AddCommand(protectCmd)

}
