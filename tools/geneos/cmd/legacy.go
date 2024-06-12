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

import (
	"github.com/spf13/cobra"
)

//
// all superseded, legacy commands are in this file and call their replacements
//

func init() {
	addCmd.AddCommand(addHostCmd)
	deleteCmd.AddCommand(deleteHostCmd)
	listCmd.AddCommand(lsHostCmd)
	setCmd.AddCommand(setHostCmd)
	showCmd.AddCommand(showHostCmd)

	updateCmd.AddCommand(updateLsCmd)
	GeneosCmd.AddCommand(installCmd)
	GeneosCmd.AddCommand(updateCmd)

	showCmd.AddCommand(showGlobalCmd)
	setCmd.AddCommand(setGlobalCmd)
	unsetCmd.AddCommand(unsetGlobalCmd)

	setCmd.AddCommand(setUserCmd)
	unsetCmd.AddCommand(unsetUserCmd)
	showCmd.AddCommand(showUserCmd)
}

// RunPlaceholder is an empty function for commands that have to run but no do anything
//
// Used to allow PersistentPreRun to check for aliases for legacy commands
var RunPlaceholder = func(command *cobra.Command, args []string) {}

var addHostCmd = &cobra.Command{
	Use:          "host [flags] [NAME] [SSHURL]",
	Aliases:      []string{"remote"},
	Short:        "Alias for `host add`",
	SilenceUsage: true,
	Args:         cobra.RangeArgs(1, 2),
	Annotations: map[string]string{
		AnnotationWildcard:   "false",
		AnnotationNeedsHome:  "false",
		AnnotationReplacedBy: "host add",
	},
	Hidden:             true,
	DisableFlagParsing: true,
	Run:                RunPlaceholder,
}

var deleteHostCmd = &cobra.Command{
	Use:          "host [flags] NAME...",
	Aliases:      []string{"hosts", "remote", "remotes"},
	Short:        "Alias for `host delete`",
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:   "false",
		AnnotationNeedsHome:  "false",
		AnnotationReplacedBy: "host delete",
	},
	Hidden:             true,
	DisableFlagParsing: true,
	Run:                RunPlaceholder,
}

var lsHostCmd = &cobra.Command{
	Use:          "host [flags] [TYPE] [NAME...]",
	Aliases:      []string{"hosts", "remote", "remotes"},
	Short:        "Alias for `host ls`",
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:   "false",
		AnnotationNeedsHome:  "false",
		AnnotationReplacedBy: "host ls",
	},
	Hidden:             true,
	DisableFlagParsing: true,
	Run:                func(cmd *cobra.Command, args []string) {},
}

var setHostCmd = &cobra.Command{
	Use:                   "host [flags] [NAME...] [KEY=VALUE...]",
	Short:                 "Alias for 'host set'",
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		AnnotationWildcard:   "false",
		AnnotationNeedsHome:  "false",
		AnnotationReplacedBy: "host set",
	},
	Hidden:             true,
	DisableFlagParsing: true,
	Run:                RunPlaceholder,
}

var showHostCmd = &cobra.Command{
	Use:          "host [flags] [NAME...]",
	Short:        "Alias for `show host`",
	Aliases:      []string{"hosts"},
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:   "false",
		AnnotationNeedsHome:  "false",
		AnnotationReplacedBy: "host show",
	},
	Hidden:             true,
	DisableFlagParsing: true,
	Run:                RunPlaceholder,
}

var installCmd = &cobra.Command{
	Use:          "install [flags] [TYPE] [FILE|URL...]",
	Short:        "Alias for `package install`",
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:   "false",
		AnnotationNeedsHome:  "true",
		AnnotationReplacedBy: "package install",
	},
	Hidden:             true,
	DisableFlagParsing: true,
	Run:                RunPlaceholder,
}

var updateLsCmd = &cobra.Command{
	Use:          "ls [flags] [TYPE]",
	Short:        "Alias for `package ls`",
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:   "false",
		AnnotationNeedsHome:  "true",
		AnnotationReplacedBy: "package ls",
	},
	Hidden:             true,
	DisableFlagParsing: true,
	Run:                RunPlaceholder,
}

var updateCmd = &cobra.Command{
	Use:          "update [flags] [TYPE] [VERSION]",
	Short:        "Alias for `package update`",
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:   "false",
		AnnotationNeedsHome:  "true",
		AnnotationReplacedBy: "package update",
	},
	Args:               cobra.RangeArgs(0, 2),
	Hidden:             true,
	DisableFlagParsing: true,
	Run:                RunPlaceholder,
}

var setUserCmd = &cobra.Command{
	Use:          "user [KEY=VALUE...]",
	Short:        "Set user configuration parameters",
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:   "false",
		AnnotationNeedsHome:  "false",
		AnnotationReplacedBy: "config set",
	},
	Hidden:             true,
	DisableFlagParsing: true,
	Run:                RunPlaceholder,
}

var unsetUserCmd = &cobra.Command{
	Use:          "user",
	Short:        "Unset a user parameter",
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:   "false",
		AnnotationNeedsHome:  "false",
		AnnotationReplacedBy: "config unset",
	},
	Hidden:             true,
	DisableFlagParsing: true,
	Run:                RunPlaceholder,
}

var showUserCmd = &cobra.Command{
	Use:          "user",
	Short:        "user",
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:   "false",
		AnnotationNeedsHome:  "false",
		AnnotationReplacedBy: "config show",
	},
	Hidden:             true,
	DisableFlagParsing: true,
	Run:                RunPlaceholder,
}

var showGlobalCmd = &cobra.Command{
	Use:          "global",
	Short:        "set global is deprecated",
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "false",
		AnnotationNeedsHome: "false",
	},
	Hidden:     true,
	Deprecated: "please view the global config file directly if required.",
}

var setGlobalCmd = &cobra.Command{
	Use:                   "global [KEY=VALUE...]",
	Short:                 "Set global configuration parameters",
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "false",
		AnnotationNeedsHome: "false",
	},
	Hidden:     true,
	Deprecated: "please edit the global config file directly if required.",
}

var unsetGlobalCmd = &cobra.Command{
	Use:          "global",
	Short:        "Unset a global parameter",
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "false",
		AnnotationNeedsHome: "false",
	},
	Hidden:     true,
	Deprecated: "please edit the global config file directly if required",
}
