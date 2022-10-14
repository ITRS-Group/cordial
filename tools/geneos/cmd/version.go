/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/itrs-group/cordial/pkg/cordial"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)

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
		"wildcard": "false",
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s version %s\n", rootCmd.Use, cmd.Version)
	},
}
