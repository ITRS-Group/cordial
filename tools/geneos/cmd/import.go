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
	"fmt"
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
Import one or multiple files to the matching instance directories.

|geneos import| may be used to add license files to an licd, scripts for
a gateway or netprobe, etc.
source files may be deifned as a file path, a URL or a |-| to read from 
STDIN.  In case the source file is defined as |-|, a destimation path must
be defined.

Files may be imported to the shared directory of a component by using
option |-c <shared_directory| (or |--common <shared_directory>|).
In such a case, the <shared _directory> is defined as |TYPE/TYPE_shared|
(e.g. |gateway/gateway_shared|).

**Note(s)**:
- To distinguish a SOURCE from an instance NAME any file in
  the current directory (without a |PATH=| prefix) **MUST** be prefixed
  with |./|.
- File imported to the shared directory of a component are imported in all 
  hosts defined.
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
		return importFiles(ct, args, params)
	},
}

// add a file to an instance, from local or URL
// overwrites without asking - use case is license files, setup files etc.
// backup / history track older files (date/time?)
// no restart or reload of components?
func importFiles(ct *geneos.Component, args []string, sources []string) (err error) {
	if importCmdCommon != "" {
		// ignore args, use ct & params
		if ct == nil {
			return fmt.Errorf("component type must be specified for common/shared directory import")
		}
		for _, r := range host.Match(importCmdHostname) {
			if _, err = instance.ImportCommons(r, ct, ct.String()+"_"+importCmdCommon, sources); err != nil {
				return
			}
		}
		return
	}

	return instance.ForAll(ct, importInstance, args, sources)
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
func importInstance(c geneos.Instance, sources []string) (err error) {
	if !c.Type().RealComponent {
		return ErrNotSupported
	}

	if len(sources) == 0 {
		log.Fatal().Msg("no file/url provided")
	}

	for _, source := range sources {
		if _, err = instance.ImportFile(c.Host(), c.Home(), c.Config().GetString("user"), source); err != nil {
			return
		}
	}
	return
}
