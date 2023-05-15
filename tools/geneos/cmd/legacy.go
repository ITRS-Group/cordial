/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

//
// all superseded, legacy commands are in this file and call their replacements
//

func init() {
	addCmd.AddCommand(addHostCmd)
	deleteCmd.AddCommand(deleteHostCmd)
	lsCmd.AddCommand(lsHostCmd)
	setCmd.AddCommand(setHostCmd)
	showCmd.AddCommand(showHostCmd)

	updateCmd.AddCommand(updateLsCmd)
	RootCmd.AddCommand(installCmd)
	RootCmd.AddCommand(updateCmd)

	showCmd.AddCommand(showGlobalCmd)
	setCmd.AddCommand(setGlobalCmd)
	unsetCmd.AddCommand(unsetGlobalCmd)

	setCmd.AddCommand(setUserCmd)
	unsetCmd.AddCommand(unsetUserCmd)
	showCmd.AddCommand(showUserCmd)
}

var addHostCmd = &cobra.Command{
	Use:     "host [flags] [NAME] [SSHURL]",
	Aliases: []string{"remote"},
	Short:   "Alias for `host add`",
	Long: strings.ReplaceAll(`
Alias for |host add|. Please use |geneos host add| in the future as
this alias will be removed in an upcoming release.
`, "|", "`"),
	SilenceUsage: true,
	Args:         cobra.RangeArgs(1, 2),
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	DisableFlagParsing: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		return RunE(command.Root(), []string{"host", "add"}, args)
	},
}

var deleteHostCmd = &cobra.Command{
	Use:     "host [flags] NAME...",
	Aliases: []string{"hosts", "remote", "remotes"},
	Short:   "Alias for `host delete`",
	Long: strings.ReplaceAll(`
Alias for |host delete|. Please use |geneos host delete| in the
future as this alias will be removed in an upcoming release.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	DisableFlagParsing: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		return RunE(command.Root(), []string{"host", "delete"}, args)
	},
}

var lsHostCmd = &cobra.Command{
	Use:     "host [flags] [TYPE] [NAME...]",
	Aliases: []string{"hosts", "remote", "remotes"},
	Short:   "Alias for `host ls`",
	Long: strings.ReplaceAll(`
Alias for |host ls|. Please use |geneos host ls| in the future as
this alias will be removed in an upcoming release.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	DisableFlagParsing: true,
	Deprecated:         "Use `geneos host ls` instead.",
	RunE: func(command *cobra.Command, args []string) (err error) {
		return RunE(command.Root(), []string{"host", "ls"}, args)
	},
}

var setHostCmd = &cobra.Command{
	Use:   "host [flags] [NAME...] [KEY=VALUE...]",
	Short: "Alias for 'host set'",
	Long: strings.ReplaceAll(`

`, "|", "`"),
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	DisableFlagParsing: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		return RunE(command.Root(), []string{"host", "set"}, args)
	},
}

var showHostCmd = &cobra.Command{
	Use:   "host [flags] [NAME...]",
	Short: "Alias for `show host`",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	Aliases:      []string{"hosts"},
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	DisableFlagParsing: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		return RunE(command.Root(), []string{"host", "show"}, args)
	},
}

var installCmd = &cobra.Command{
	Use:   "install [flags] [TYPE] [FILE|URL...]",
	Short: "Alias for `package install`",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "true",
	},
	DisableFlagParsing: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		return RunE(command.Root(), []string{"package", "install"}, args)
	},
}

var updateLsCmd = &cobra.Command{
	Use:   "ls [flags] [TYPE]",
	Short: "Alias for `package ls`",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "true",
	},
	DisableFlagParsing: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		return RunE(command.Root(), []string{"package", "ls"}, args)
	},
}

var updateCmd = &cobra.Command{
	Use:   "update [flags] [TYPE] [VERSION]",
	Short: "Alias for `package update`",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "true",
	},
	Args:               cobra.RangeArgs(0, 2),
	DisableFlagParsing: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		return RunE(command.Root(), []string{"package", "update"}, args)
	},
}

var setUserCmd = &cobra.Command{
	Use:   "user [KEY=VALUE...]",
	Short: "Set user configuration parameters",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	DisableFlagParsing: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		return RunE(command.Root(), []string{"config", "set"}, args)
	},
}

var unsetUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Unset a user parameter",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	DisableFlagParsing: true,
	RunE: func(command *cobra.Command, args []string) error {
		return RunE(command.Root(), []string{"config", "unset"}, args)
	},
}

var showUserCmd = &cobra.Command{
	Use:   "user",
	Short: "A brief description of your command",
	Long: strings.ReplaceAll(`
A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	DisableFlagParsing: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		return RunE(command.Root(), []string{"config", "show"}, args)
	},
}

var showGlobalCmd = &cobra.Command{
	Use:   "global",
	Short: "set global is deprecated",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	Deprecated: "please view the global config file directly if required.",
}

var setGlobalCmd = &cobra.Command{
	Use:   "global [KEY=VALUE...]",
	Short: "Set global configuration parameters",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	Deprecated: "please edit the global config file directly if required.",
}

var unsetGlobalCmd = &cobra.Command{
	Use:   "global",
	Short: "Unset a global parameter",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	Deprecated: "please edit the global config file directly if required",
}
