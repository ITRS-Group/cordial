/*
Copyright © 2023 ITRS Group

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

import (
	"errors"

	"github.com/spf13/cobra"
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push file data using either XML or REST APIs (not yet implemented)",
	Long: `Push mode is not yet implemented. The command is reserved for a future release,
and currently returns an error. Use the default toolkit mode to output a Dataview.`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		return errors.New("push mode is not yet implemented")
	},
}

func init() {
	FILE2DVCmd.AddCommand(pushCmd)
}
