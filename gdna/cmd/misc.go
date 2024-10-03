/*
Copyright © 2024 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import "github.com/spf13/cobra"

var addCmdUser, addCmdComment, addCmdOrigin string

func init() {
	GDNACmd.AddCommand(addCmd)
	GDNACmd.AddCommand(removeCmd)

	addCmd.PersistentFlags().StringVar(&addCmdUser, "user", "", "user adding these items")
	addCmd.PersistentFlags().StringVar(&addCmdComment, "comment", "", "comment for these items")
	addCmd.PersistentFlags().StringVar(&addCmdOrigin, "origin", "", "origin for these items")

	removeCmd.PersistentFlags().BoolVarP(&removeCmdAll, "all", "A", false, "remove all filters or group for a category")

}

var addCmdDescription string
var removeCmdDescription string

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "",
	Long:  addCmdDescription,
	Args:  cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	// no action
}

var removeCmd = &cobra.Command{
	Use:     "remove",
	Short:   "",
	Long:    removeCmdDescription,
	Aliases: []string{"rm", "delete"},
	Args:    cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	// no action
}

// listCmd is in list.go
