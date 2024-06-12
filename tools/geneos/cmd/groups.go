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

package cmd

import "github.com/spf13/cobra"

// Available command groups for Cobra command set-up. This influences
// the display of the help text for the top-level `geneos` command.
const (
	CommandGroupConfig      = "config"
	CommandGroupComponents  = "components"
	CommandGroupCredentials = "credentials"
	CommandGroupManage      = "manage"
	CommandGroupOther       = "other"
	CommandGroupProcess     = "process"
	CommandGroupSubsystems  = "subsystems"
	CommandGroupView        = "view"
)

func init() {
	GeneosCmd.AddGroup(&cobra.Group{
		ID:    CommandGroupSubsystems,
		Title: "Subsystems",
	})
	GeneosCmd.AddGroup(&cobra.Group{
		ID:    CommandGroupProcess,
		Title: "Control Instances",
	})
	GeneosCmd.AddGroup(&cobra.Group{
		ID:    CommandGroupView,
		Title: "Inspect Instances",
	})
	GeneosCmd.AddGroup(&cobra.Group{
		ID:    CommandGroupManage,
		Title: "Manage Instances",
	})
	GeneosCmd.AddGroup(&cobra.Group{
		ID:    CommandGroupConfig,
		Title: "Configure Instances",
	})
	GeneosCmd.AddGroup(&cobra.Group{
		ID:    CommandGroupCredentials,
		Title: "Manage Credentials",
	})
	GeneosCmd.AddGroup(&cobra.Group{
		ID:    CommandGroupOther,
		Title: "Miscellaneous",
	})
	GeneosCmd.AddGroup(&cobra.Group{
		ID:    CommandGroupComponents,
		Title: "Component Types",
	})
}
