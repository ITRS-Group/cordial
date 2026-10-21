/*
Copyright © 2026 ITRS Group

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

package profile

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"src.elv.sh/pkg/md"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

//go:embed profile.md
var profileDescription string

func init() {
	cmd.Cmd.AddCommand(profileHelpCmd)

	profileHelpCmd.SetHelpFunc(profileHelp)
}

// profileHelpCmd represents the profile component
var profileHelpCmd = &cobra.Command{
	Use:                   "profile [name]",
	GroupID:               cmd.CommandGroupComponents,
	Short:                 "Profile",
	Long:                  profileDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Run:                   cmd.Cmd.HelpFunc(),
}

// profileHelp is the custom help function for the profile command. It
// checks if a profile name is provided as an argument and displays its
// description if available. Otherwise, it shows the general help text.
func profileHelp(command *cobra.Command, _ []string) {
	// if given a profile name as an argumnent, see if the profile
	// has a `description` field and display it as help text using
	// the same markdown renderer

	// pull out profile name from the command-line arguments
	args := os.Args[3:]

	if len(args) > 0 {
		// check if the profile exists and has a description
		// display the description as help text
		// (implementation would go here)
		pf := cmd.Profiles
		name := args[0]
		components, found := config.Lookup[map[string]any](pf, pf.Join("profiles", name))
		if found {
			if desc, ok := components["description"].(string); ok {
				fmt.Println(md.RenderString(desc, &md.TTYCodec{Width: 76}))
			}
			return
		}
	}

	// no profile name is given, just display the general help text
	command.SetHelpFunc(nil)
	command.Help()
}
