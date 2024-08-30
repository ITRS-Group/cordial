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
	"archive/tar"
	"compress/gzip"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var exportCmdOutput string

var exportCmdIncludeAll bool
var exportCmdIncludeAES, exportCmdIncludeCerts bool

func init() {
	configCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&exportCmdOutput, "output", "o", "", "Output file `path`. If path is a directory or has a '/' suffix then the constructed\nfile name is used in that directory. If not final file name is given then the\nfile name is of the form 'geneos[-TYPE][-NAME].tar.gz'")

	exportCmd.Flags().BoolVarP(&exportCmdIncludeAll, "include-all", "A", false, "Include all files except AES key files, certificates and associated files, in the archive.\nThis may fail for running instances")
	exportCmd.Flags().BoolVarP(&exportCmdIncludeAES, "include-aes", "S", false, "Include AES key files in the archive\n(never includes user's own keyfile)")
	exportCmd.Flags().BoolVarP(&exportCmdIncludeCerts, "include-certs", "C", false, "Include certificates, private keys and certificate chains in archive")

	exportCmd.Flags().SortFlags = false
}

//go:embed _docs/export.md
var exportCmdDescription string

var exportCmd = &cobra.Command{
	Use:   "export [flags] [TYPE] [NAME...]",
	Short: "Export Instances",
	Long:  exportCmdDescription,
	Example: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:        "true",
		cmd.CmdRequireHome:   "true",
		cmd.CmdWildcardNames: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		var archive string

		ct, names := cmd.ParseTypeNames(command)

		// h can be geneos.ALL in which case it is updated below before
		// being used
		h := geneos.GetHost(cmd.Hostname)

		switch {
		case strings.HasSuffix(exportCmdOutput, "/"):
			// user has asked for a directory, so try to create it as well
			archive = exportCmdOutput
			os.MkdirAll(archive, 0664)
		case exportCmdOutput != "":
			// check for existing file, else try to create the directories above it
			st, err := os.Stat(exportCmdOutput)
			if err == nil {
				// file (or directory) exists, use it
				archive = exportCmdOutput
				// destination may be an existing directory, in which case append a '/'
				if st.IsDir() {
					archive += "/"
				}
			} else {
				if errors.Is(err, os.ErrNotExist) {
					archive = exportCmdOutput
					os.MkdirAll(filepath.Dir(archive), 0775)
				} else {
					return err
				}
			}
		default:
			// do nothing
		}

		if archive == "" || strings.HasSuffix(archive, "/") {
			// build archive name, starting with executable
			archive += cordial.ExecutableName()

			if ct != nil {
				// add component types
				archive += "-" + ct.String()
			}

			var i geneos.Instance

			if len(names) == 1 {
				// single instance
				instances, err := instance.Instances(h, ct, instance.FilterNames(names...))
				if err != nil {
					return err
				}
				if len(instances) == 0 {
					return geneos.ErrNotExist
				}
				i = instances[0]
				archive += "-" + i.Type().String()
				archive += "-" + strings.ReplaceAll(i.Name(), " ", "_")
				h = i.Host()
			} else {
				// more than one, error if not (a) all the same type and (b) all on the same host
				instances, err := instance.Instances(h, ct, instance.FilterNames(names...))
				if err != nil {
					return err
				}
				var nct *geneos.Component
				var nh *geneos.Host
				for _, i := range instances {
					if nct == nil {
						nct = i.Type()
						nh = i.Host()
						continue
					}
					if nct != i.Type() || nh != i.Host() {
						return errors.New("all matches must be of the same TYPE and on the same host")
					}
				}
				if ct == nil {
					archive += "-" + nct.String()
				}
				h = nh
			}

			archive += ".tar.gz"
		}

		// contents is a list of files so that instances do not duplicate shared contents
		var contents []string
		instance.Do(h, ct, names, exportInstance, &contents).Write(os.Stdout)

		slices.Sort(contents)
		contents = slices.Compact(contents)
		log.Info().Msgf("contents: %v", contents)

		out, err := os.Create(archive)
		if err != nil {
			return
		}
		defer out.Close()
		gz := gzip.NewWriter(out)
		defer gz.Close()
		tw := tar.NewWriter(gz)
		defer tw.Close()

		root := h.PathTo()
		for _, f := range contents {
			p := path.Join(root, f)
			fi, err := h.Stat(p)
			if err != nil {
				return err
			}
			if !fi.Mode().IsRegular() {
				return fmt.Errorf("%s is not a regular file", p)
			}
			own := h.GetFileOwner(fi)
			th := &tar.Header{
				Format:  tar.FormatUnknown,
				Name:    f,
				Size:    fi.Size(),
				Mode:    int64(fi.Mode()),
				ModTime: fi.ModTime(),
				Uid:     own.Uid,
				Gid:     own.Gid,
			}
			tw.WriteHeader(th)
			b, err := h.ReadFile(p)
			if err != nil {
				return err
			}
			if _, err = tw.Write(b); err != nil {
				return err
			}
		}

		return nil
	},
}

func exportInstance(i geneos.Instance, params ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	cf := i.Config()

	if len(params) == 0 {
		log.Info().Msg("len(params) = 0")
		resp.Err = geneos.ErrInvalidArgs
		return
	}
	contents, ok := params[0].(*[]string)
	if !ok {
		log.Info().Msg("no contents")
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	var ignoredirs, ignorefiles []string

	var ignorelist []string
	if !exportCmdIncludeAll {
		ignorelist = strings.Split(config.GetString(i.Type().CleanList, config.Default(i.Type().ConfigAliases[i.Type().CleanList])), ":")
		ignorelist = append(ignorelist, strings.Split(config.GetString(i.Type().PurgeList, config.Default(i.Type().ConfigAliases[i.Type().PurgeList])), ":")...)

		if !exportCmdIncludeAES {
			ignorelist = append(ignorelist, "*.aes")
		}

		for _, i := range ignorelist {
			if strings.HasSuffix(i, "/") {
				ignoredirs = append(ignoredirs, strings.TrimSuffix(i, "/"))
			} else {
				ignorefiles = append(ignorefiles, i)
			}
		}
	}

	// walk the instance home directory first
	d := i.Home()
	r := strings.TrimPrefix(d, i.Host().PathTo()+"/")

	var ignoreSecure []string
	if !exportCmdIncludeAES {
		ignoreSecure = append(ignoreSecure, "keyfile", "prevkeyfile")
	}
	if !exportCmdIncludeCerts {
		ignoreSecure = append(ignoreSecure, "certificate", "privatekey", "certchain")
	}

	for _, ig := range ignoreSecure {
		if cf.IsSet(ig) {
			ignorefiles = append(ignorefiles, strings.TrimPrefix(cf.GetString(ig), i.Home()+"/"))
		}
	}

	fs.WalkDir(os.DirFS(d), ".", func(file string, di fs.DirEntry, _ error) error {
		fi, err := di.Info()
		if err != nil {
			return err
		}
		switch {
		case fi.IsDir():
			for _, ig := range ignoredirs {
				if match, err := filepath.Match(ig, file); err == nil && match {
					return fs.SkipDir
				}
			}
			return nil
		case fi.Mode()&fs.ModeSymlink != 0:
			// log.Info().Msgf("symlink %s", file)
		default:
			// log.Info().Msgf("file %s", file)
			for _, ig := range ignorefiles {
				if match, err := filepath.Match(ig, file); err == nil && match {
					return nil
				}
			}
			*contents = append(*contents, filepath.Join(r, file))
			return nil
		}
		return nil
	})

	// then wlk the shared directory, checking and updating contents
	d = i.Type().Shared(i.Host())
	r = strings.TrimPrefix(d, i.Host().GetString(cmd.Execname)+"/")

	fs.WalkDir(os.DirFS(d), ".", func(file string, di fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fi, err := di.Info()
		if err != nil {
			return err
		}
		switch {
		case fi.IsDir():
			return nil
		case fi.Mode()&fs.ModeSymlink != 0:
			// log.Info().Msgf("symlink %s", file)
		default:
			*contents = append(*contents, filepath.Join(r, file))
			return nil
		}
		return nil
	})

	return
}
