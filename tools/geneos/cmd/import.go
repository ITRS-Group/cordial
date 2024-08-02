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
		CmdNoneMeansAll: "true",
		CmdRequireHome:  "true",
		CmdGlobNames:    "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, names, params := ParseTypeNamesParams(cmd)
		return ImportFiles(ct, names, params)
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

	instance.Do(geneos.GetHost(Hostname), ct, args, importInstance, sources)
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
func importInstance(i geneos.Instance, params ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	if len(params) == 0 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}
	sources, ok := params[0].([]string)
	if !ok {
		panic("wront type")
	}

	if i.Type() == &geneos.RootComponent {
		resp.Err = geneos.ErrNotSupported
		return
	}

	if len(sources) == 0 {
		log.Fatal().Msg("no file/url provided")
	}

	for _, source := range sources {
		if _, resp.Err = geneos.ImportFile(i.Host(), i.Home(), source); resp.Err != nil && resp.Err != geneos.ErrExists {
			return
		}
	}
	resp.Err = nil
	return
}
