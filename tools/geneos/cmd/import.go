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

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var importCmdCommon string

func init() {
	GeneosCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&importCmdCommon, "common", "c", "", "Import file(s) to a components top-level directory with suffix TYPE_`SUFFIX`")

	importCmd.Flags().SortFlags = false
}

var importCmd = &cobra.Command{
	Use:     "import [flags] [TYPE] [NAME...] [DEST=]SOURCE...",
	GroupID: GROUP_CONFIG,
	Short:   "Import files to an instance or a common directory",
	Long: strings.ReplaceAll(`
Import each |SOURCE| to instance directories. With the
|--common|/|-c| option the imports are to a TYPE component
sub-directory |TYPE_| suffixed with the value to the |--common|/|-c|
option. See examples below.

The |SOURCE| can be the path to a local file, a URL or '-' for STDIN.
|SOURCE| may not be a directory.

If the |SOURCE| is a file in the current directory then it must be
prefixed with |"./"| to avoid being seen as an instance NAME to
search for. Any file path with a directory separator already present
does not need this precaution. The program will read from STDIN if
the |SOURCE| '-' is given but this can only be used once and a
destination DEST must be defined.

If |DEST| is given with a |SOURCE| then it must either be a plain file
name or a descending relative path. An absolute or ascending path is
an error.

Without an explicit |DEST| for the destination file only the base name
of the |SOURCE| is used. If |SOURCE| is a URL then the file name for
the resource from the remote web server is preferred over the last
part of the URL.

If the |--common|/|-c| option is used then a TYPE must also be
specified. Each component of TYPE has a base directory. That
directory may contain, in addition to instances of that TYPE, a
number of other directories that can be used for shared resources.
These may be scripts, include files and so on. Using a TYPE |gateway|
as an example and using a |--common config| option the destination
for |SOURCE| would be |gateway/gateway_config|

Future release may add support for directories and.or unarchiving of
|tar.gz|/|zip| and other file archives.
`, "|", "`"),
	Example: strings.ReplaceAll(`
# import a gateway setup file from a web server
geneos import gateway example1 https://example.com/myfiles/gateway.setup.xml

# import the "license.txt" file to the licd instance example2 but
# with a filename of geneos.lic
geneos import licd example2 geneos.lic=license.txt

# import the "myscript.sh" file into the scripts directory under the
# netprobe example3's working directory
#
# Note: the file will not be made executable
geneos import netprobe example3 scripts/=myscript.sh

# import the file "netprobe.setup.xml" from the current directory to
# the SAN localhost
# 
# Note the leading "./" to disambiguate the file name from an instance
# to match
geneos import san localhost ./netprobe.setup.xml

# import "common_include" into the gateway_shared directory under the gateway are of the installation directory
geneos import gateway -c shared common_include.xml
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := CmdArgsParams(cmd)
		return ImportFiles(ct, args, params)
	},
}

// ImportFiles add a file to an instance, from local or URL overwrites
// without asking - use case is license files, setup files etc. backup /
// history track older files (date/time?) no restart or reload of
// components?
func ImportFiles(ct *geneos.Component, args []string, sources []string) (err error) {
	if importCmdCommon != "" {
		// ignore args, use ct & params
		if ct == nil {
			return fmt.Errorf("component type must be specified for common/shared directory import")
		}
		for _, r := range geneos.Match(Hostname) {
			if _, err = instance.ImportCommons(r, ct, ct.String()+"_"+importCmdCommon, sources); err != nil {
				return
			}
		}
		return
	}

	return instance.ForAll(ct, Hostname, importInstance, args, sources)
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
		if _, err = instance.ImportFile(c.Host(), c.Home(), source); err != nil {
			return
		}
	}
	return
}
