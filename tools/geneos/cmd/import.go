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

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var importCmdCommon, importCmdHostname string

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&importCmdCommon, "common", "c", "", "Import into a common directory instead of matching instances.	For example, if TYPE is 'gateway' and NAME is 'shared' then this common directory is 'gateway/gateway_shared'")
	importCmd.Flags().StringVarP(&importCmdHostname, "host", "H", "all", "Import only to named host, default is all")

	importCmd.Flags().SortFlags = false
}

var importCmd = &cobra.Command{
	Use:   "import [flags] [TYPE] [NAME...] [PATH=]SOURCE...",
	Short: "Import files to an instance or a common directory",
	Long: strings.ReplaceAll(`
Import one or more files to matching instance directories, or with
|--common| flag to a component shared directory. This can be used to
add configuration or license files or scripts for gateways and
netprobes to run. The SOURCE can be a local path, a URL or a |-| for
stdin. PATH is local pathname ending in either a filename or a
directory separator. Is SOURCE is |-| then a destination PATH must be
given. If PATH includes a directory separator then it must be
relative to the instance directory and cannot contain a parent
reference |..|.

Only the base filename of SOURCE is used and if SOURCE contains
parent directories these are stripped and if required should be
provided in PATH.

**Note**: To distinguish a SOURCE from an instance NAME any file in
the current directory (without a |PATH=| prefix) **MUST** be prefixed
with |./|. Any SOURCE that is not a valid instance name is treated as
SOURCE and no immediate error is raised. Directories are created as required.
If run as root, directories and files ownership is set to the user in
the instance configuration or the default user.

Currently only files can be imported and if the SOURCE is a directory
then this is an error.
`, "|", "`"),
	Example: strings.ReplaceAll(`
geneos import gateway example1 https://example.com/myfiles/gateway.setup.xml
geneos import licd example2 geneos.lic=license.txt
geneos import netprobe example3 scripts/=myscript.sh
geneos import san localhost ./netprobe.setup.xml
geneos import gateway -c shared common_include.xml
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return commandImport(ct, args, params)
	},
}

// add a file to an instance, from local or URL
// overwrites without asking - use case is license files, setup files etc.
// backup / history track older files (date/time?)
// no restart or reload of components?

func commandImport(ct *geneos.Component, args []string, params []string) (err error) {
	if importCmdCommon != "" {
		// ignore args, use ct & params
		for _, r := range host.Match(importCmdHostname) {
			if _, err = instance.ImportCommons(r, ct, ct.String()+"_"+importCmdCommon, params); err != nil {
				return
			}
		}
		return
	}

	return instance.ForAll(ct, importInstance, args, params)
}

// args are instance [file...]
// file can be a local path or a url
// destination is basename of source in the home directory
// file can also be DEST=SOURCE where dest must be a relative path (with
// no ../) to home area, anding in / means subdir, e.g.:
//
// 'geneos import gateway example1 https://example.com/myfiles/gateway.setup.xml'
// 'geneos import licd example2 geneos.lic=license.txt'
// 'geneos import netprobe example3 scripts/=myscript.sh'
//
// local directories are created
func importInstance(c geneos.Instance, params []string) (err error) {
	if !c.Type().RealComponent {
		return ErrNotSupported
	}

	if len(params) == 0 {
		log.Fatal().Msg("no file/url provided")
	}

	for _, source := range params {
		if _, err = instance.ImportFile(c.Host(), c.Home(), c.Config().GetString("user"), source); err != nil {
			return
		}
	}
	return
}
