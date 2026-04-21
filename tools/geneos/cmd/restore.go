/*
/*
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
	"os"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos/restore"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var restoreCmdCompression string
var restoreCmdShared, restoreCmdList bool

func init() {
	GeneosCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().BoolVarP(&restoreCmdShared, "shared", "s", false, "include shared files")

	restoreCmd.Flags().StringVarP(&restoreCmdCompression, "decompress", "z", "", "use decompression `TYPE`, one of `gzip`, `bzip2` or `none`\nif not given then the file name is used to guess the type\nMUST be supplied if the source is stdin (`-`)")

	restoreCmd.Flags().BoolVarP(&restoreCmdList, "list", "l", false, "list the contents of the archive(s)")

	restoreCmd.Flags().SortFlags = false
}

var fileTypes = map[string]string{
	".tar.gz":  "gzip",
	".tgz":     "gzip",
	".tar.bz2": "bz2",
	".tar":     "none",
}

//go:embed _docs/restore.md
var restoreCmdDescription string

var restoreCmd = &cobra.Command{
	Use:     "restore [flags] [TYPE] [[DEST=]NAME...]",
	Aliases: []string{"load"},
	GroupID: CommandGroupConfig,
	Short:   "Restore instances from archive",
	Long:    restoreCmdDescription,
	Example: strings.ReplaceAll(`
geneos restore backup.tgz
geneos restore gateway ABC x.tgz
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:        "false",
		CmdRequireHome:   "true",
		CmdWildcardNames: "false",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		ct, names, params, err := FetchArgs(command)
		if err != nil {
			return
		}

		names = append(names, params...)

		if !restoreCmdList && len(args) == 0 {
			return command.Usage()
		}

		// specific host or local, never all
		h := geneos.NewHost(Hostname)
		if h == geneos.ALL {
			h = geneos.LOCAL
		}

		// extract any args that look like archives
		files := []string{}
		names = slices.DeleteFunc(names, func(name string) bool {
			for n := range fileTypes {
				if strings.HasSuffix(name, n) {
					files = append(files, name)
					return true
				}
			}
			return false
		})

		log.Debug().Msgf("files: %v", files)

		// remove host suffixes from args
		// TODO rewrite as a slices in-place delete/replace
		var newnames []string
		for _, n := range names {
			h2, _, n2 := instance.ParseName(n, h)
			if h2 != h {
				// skip any args that don't match destination host, just in case
				continue
			}
			newnames = append(newnames, n2)
		}
		names = newnames

		if len(names) == 0 {
			switch {
			case restoreCmdList || restoreCmdShared:
				names = []string{"all"}
			default:
				return fmt.Errorf("`restore` requires specific instances to restore or `--shared` and/or the wildcard name `all`")
			}

		}

		// if no file names are found in args then assume STDIN. later
		// we could look for named file patterns, if it proves useful
		if len(files) == 0 {
			// if err = restoreFromFile(h, ct, "-", names); err != nil {
			if err = restore.Restore("-",
				restore.Host(h),
				restore.Component(ct),
				restore.Names(names...),
				restore.Compression(restoreCmdCompression),
				restore.Shared(restoreCmdShared),
				restore.List(restoreCmdList),
				restore.ProgressTo(os.Stdout),
			); err != nil {
				return
			}
		} else {
			for _, f := range files {
				// process file
				log.Debug().Msgf("checking file %s for ct %s and names %v", f, ct, names)
				if err = restore.Restore(f,
					restore.Host(h),
					restore.Component(ct),
					restore.Names(names...),
					restore.Compression(restoreCmdCompression),
					restore.Shared(restoreCmdShared),
					restore.List(restoreCmdList),
					restore.ProgressTo(os.Stdout),
				); err != nil {
					return
				}
			}
		}

		return
	},
}
