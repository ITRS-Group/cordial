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
	"archive/tar"
	"compress/gzip"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/alecthomas/units"
	"github.com/dsnet/compress/bzip2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var exportCmdOutput string

var exportCmdIncludeAll, exportCmdIncludeShared bool
var exportCmdIncludeAES, exportCmdIncludeCerts bool
var exportCmdLimitSize, exportCmdCompression string
var maxsize int64

func init() {
	GeneosCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&exportCmdOutput, "output", "o", "", "Output file `path`. If path is a directory or has a '/' suffix then the constructed\nfile name is used in that directory. If not final file name is given then the\nfile name is of the form 'geneos[-TYPE][-NAME].tar.gz'")

	exportCmd.Flags().StringVarP(&exportCmdCompression, "compress", "c", "gzip", "Compressions type. One of gzip, bzip2 or none.")

	exportCmd.Flags().StringVarP(&exportCmdLimitSize, "size", "s", "1MiB", "Ignore files larger than this size (in bytes) unless --all is used\nAccepts suffixes i=with both B and iB units")
	exportCmd.Flags().BoolVarP(&exportCmdIncludeAll, "all", "A", false, "Include all files except AES key files, certificates and associated files, in the archive.\nThis may fail for running instances")

	exportCmd.Flags().BoolVar(&exportCmdIncludeShared, "shared", false, "Include shared directory contents in the archive\n(also use --all to include files that are filtered by clean/purge lists)")

	exportCmd.Flags().BoolVar(&exportCmdIncludeAES, "aes", false, "Include AES key files in the archive\n(never includes user's own keyfile)")
	exportCmd.Flags().BoolVar(&exportCmdIncludeCerts, "tls", false, "Include certificates, private keys and certificate chains in archive")

	exportCmd.Flags().SortFlags = false
}

//go:embed _docs/export.md
var exportCmdDescription string

var compression = map[string]string{
	"gzip":  "gz",
	"bzip2": "bz2",
	"none":  "",
}

var exportCmd = &cobra.Command{
	Use:   "export [flags] [TYPE] [NAME...]",
	Short: "Export Instances",
	Long:  exportCmdDescription,
	Example: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:        "true",
		CmdRequireHome:   "true",
		CmdWildcardNames: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		var archive string

		ct, names := ParseTypeNames(command)

		// h can be geneos.ALL in which case it is updated below before
		// being used
		h := geneos.GetHost(Hostname)

		suffix, ok := compression[exportCmdCompression]
		if !ok {
			return fmt.Errorf("invalid compression type, select one of: %s", strings.Join(slices.Collect(maps.Keys(compression)), ", "))
		}

		if !exportCmdIncludeAll {
			maxsize, err = units.ParseStrictBytes(exportCmdLimitSize)
			if err != nil {
				return fmt.Errorf("invalid size: %w", err)
			}
		}

		if exportCmdOutput != "" {
			if strings.HasSuffix(exportCmdOutput, "/") {
				// user has asked for a directory, so try to create it as well
				archive = exportCmdOutput
				os.MkdirAll(archive, 0664)
			} else {
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
			}
		}

		if archive == "" || strings.HasSuffix(archive, "/") {
			var i geneos.Instance

			// build archive name, starting with executable
			archive += cordial.ExecutableName()

			if ct != nil {
				// add component types
				archive += "-" + ct.String()
			}

			if len(names) == 1 {
				// single instance
				instances := instance.Instances(h, ct, instance.FilterNames(names...))
				if len(instances) == 0 {
					return geneos.ErrNotExist
				}
				i = instances[0]
				archive += "-" + i.Type().String()
				archive += "-" + strings.ReplaceAll(i.Name(), " ", "_")
				h = i.Host()
			} else {
				// more than one, error if not (a) all the same type and (b) all on the same host
				var nct *geneos.Component
				var nh *geneos.Host
				for _, i := range instance.Instances(h, ct, instance.FilterNames(names...)) {
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

			archive += ".tar"
			if suffix != "" {
				archive += "." + suffix
			}
		}

		// contents is a list of files so that instances do not duplicate shared contents
		var contents []string
		resp := instance.Do(h, ct, names, exportInstance, &contents)
		if archive != "-" {
			resp.Write(os.Stdout)
		} else {
			resp.Write(os.Stderr)
		}

		slices.Sort(contents)
		contents = slices.Compact(contents)
		log.Debug().Msgf("contents: %v", contents)

		var out *os.File
		if archive == "-" {
			out = os.Stdout
			if h == geneos.ALL {
				h = geneos.LOCAL
			}
		} else {
			out, err = os.Create(archive)
			if err != nil {
				return
			}
			defer out.Close()
		}

		var cout io.WriteCloser

		switch exportCmdCompression {
		case "gzip":
			cout, err = gzip.NewWriterLevel(out, gzip.BestCompression)
			if err != nil {
				return
			}
			defer cout.Close()
		case "bzip2":
			cout, err = bzip2.NewWriter(out, &bzip2.WriterConfig{Level: 9})
			if err != nil {
				return
			}
			defer cout.Close()
		case "none":
			cout = out
		}

		tw := tar.NewWriter(cout)
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
			uid, gid := h.GetFileOwner(fi)
			th := &tar.Header{
				Format:  tar.FormatUnknown,
				Name:    f,
				Size:    fi.Size(),
				Mode:    int64(fi.Mode()),
				ModTime: fi.ModTime(),
				Uid:     uid,
				Gid:     gid,
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
	var ignoredirs, ignorefiles []string

	resp = instance.NewResponse(i)

	cf := i.Config()
	ct := i.Type()

	if len(params) == 0 {
		log.Debug().Msg("len(params) = 0")
		resp.Err = geneos.ErrInvalidArgs
		return
	}
	contents, ok := params[0].(*[]string)
	if !ok {
		log.Debug().Msgf("invalid contents parameter (%T and not *[]string)", contents)
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	if !exportCmdIncludeAll {
		ignorelist := strings.Split(config.GetString(ct.CleanList, config.Default(ct.ConfigAliases[ct.CleanList])), ":")
		ignorelist = append(ignorelist, strings.Split(config.GetString(ct.PurgeList, config.Default(ct.ConfigAliases[ct.PurgeList])), ":")...)

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
			for _, ig := range ignoredirs {
				if match, err := filepath.Match(ig, file); err == nil && match {
					return fs.SkipDir
				}
			}
			return nil
		case fi.Mode()&fs.ModeSymlink != 0:
			log.Debug().Msgf("ignoring symlink %s", file)
		default:
			if !exportCmdIncludeAll {
				for _, ig := range ignorefiles {
					if match, err := filepath.Match(ig, file); err == nil && match {
						return nil
					}
				}
				if fi.Size() > maxsize {
					log.Debug().Msgf("skipping large file %s", filepath.Join(r, file))
					return nil
				}
			}
			*contents = append(*contents, filepath.Join(r, file))
			return nil
		}
		return nil
	})

	if !exportCmdIncludeShared {
		return
	}

	// then walk the shared directory, checking and updating contents
	d = ct.Shared(i.Host())
	r = strings.TrimPrefix(d, i.Host().GetString(cordial.ExecutableName())+"/")

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
			if !exportCmdIncludeAll {
				for _, ig := range ignoredirs {
					if match, err := filepath.Match(ig, file); err == nil && match {
						return fs.SkipDir
					}
				}
			}
			return nil
		case fi.Mode()&fs.ModeSymlink != 0:
			log.Debug().Msgf("ignoring symlink %s", file)
		default:
			if !exportCmdIncludeAll {
				for _, ig := range ignorefiles {
					if match, err := filepath.Match(ig, file); err == nil && match {
						return nil
					}
				}
				if fi.Size() > maxsize {
					log.Debug().Msgf("skipping large file %s", filepath.Join(r, file))
					return nil
				}
			}
			*contents = append(*contents, filepath.Join(r, file))
			return nil
		}
		return nil
	})

	return
}
