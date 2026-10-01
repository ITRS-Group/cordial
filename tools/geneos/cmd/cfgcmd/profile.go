/*
Copyright © 2022 ITRS Group

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

package cfgcmd

import (
	_ "embed"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/profiles"
)

func init() {
	configCmd.AddCommand(profileCmd)

	profileCmd.Flags().SortFlags = false
}

//go:embed _docs/profile.md
var profileCmdDescription string

var profileCmd = &cobra.Command{
	Use:   "profile [flags] [TYPE] [NAME...]",
	Short: "set-up profiles",
	Long:  profileCmdDescription,
	Example: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		log := cordial.Logger.With("command", "config profile")

		if len(args) < 1 {
			return fmt.Errorf("profile type is required")
		}

		// Initialise profiles, in case this is first use
		profiles.Initialise(cordial.ExecutableName())

		profileType := args[0]

		pf, err := profiles.Load(cordial.ExecutableName())
		if err != nil {
			log.Error("failed to load profiles", slog.Any("err", err))
			return err
		}

		log.Debug("applying profile", slog.String("type", profileType))
		return profiles.Apply(pf, profileType)
	},
}
