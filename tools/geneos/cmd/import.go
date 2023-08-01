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
	_ "embed"
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

	importCmd.Flags().StringVarP(&importCmdCommon, "common", "c", "", "Import files to a component directory named TYPE_`SUFFIX`")

	importCmd.Flags().SortFlags = false
}

//go:embed _docs/import.md
var importCmdDescription string

var importCmd = &cobra.Command{
	Use:     "import [flags] [TYPE] [NAME...] [DEST=]SOURCE...",
	GroupID: CommandGroupConfig,
	Short:   "Import Files To Instances Or Components",
	Long:    importCmdDescription,
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
			if _, err = geneos.ImportCommons(r, ct, ct.String()+"_"+importCmdCommon, sources); err != nil {
				return
			}
		}
		return nil
	}

	_, err = instance.ForAllWithParamStringSlice(geneos.GetHost(Hostname), ct, importInstance, args, sources)
	return
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
func importInstance(c geneos.Instance, sources []string) (result any, err error) {
	if !c.Type().RealComponent {
		err = geneos.ErrNotSupported
		return
	}

	if len(sources) == 0 {
		log.Fatal().Msg("no file/url provided")
	}

	for _, source := range sources {
		if _, err = geneos.ImportFile(c.Host(), c.Home(), source); err != nil && err != geneos.ErrExists {
			return
		}
	}
	err = nil
	return
}
