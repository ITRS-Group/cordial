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
	"time"

	"github.com/alecthomas/units"
	"github.com/dsnet/compress/bzip2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var saveCmdOutput string

var saveCmdIncludeAll, saveCmdIncludeShared bool
var saveCmdIncludeAES, saveCmdIncludeTLS bool
var saveCmdIncludeDatetime bool
var saveCmdLimitSize, saveCmdCompression string

var maxsize int64

func init() {
	GeneosCmd.AddCommand(saveCmd)

	saveCmd.Flags().StringVarP(&saveCmdOutput, "output", "o", "", "Output file `path`. If path is a directory or has a '/' suffix then the constructed\nfile name is used in that directory. If not final file name is given then the\nfile name is of the form 'geneos[-TYPE][-NAME].tar.gz'")

	// archive name options, if not a fixed name
	saveCmd.Flags().BoolVarP(&saveCmdIncludeDatetime, "datetime", "D", false, "include a datetime string the in the auto-generated archive name")

	saveCmd.Flags().StringVarP(&saveCmdCompression, "compress", "z", "gzip", "Compression `type`. One of `gzip`, `bzip2` or `none`.")

	saveCmd.Flags().StringVarP(&saveCmdLimitSize, "size", "s", "1MiB", "Ignore files larger than this size (in bytes) unless --all is used\nAccepts suffixes i=with both B and iB units")
	saveCmd.Flags().BoolVarP(&saveCmdIncludeAll, "all", "A", false, "Include all files except AES key files, certificates and associated files, in the archive.\nThis may fail for running instances")

	saveCmd.Flags().BoolVar(&saveCmdIncludeShared, "shared", false, "Include shared directory contents in the archive\n(also use --all to include files that are filtered by clean/purge lists)")

	saveCmd.Flags().BoolVar(&saveCmdIncludeAES, "aes", false, "Include AES key files in the archive\n(never includes user's own keyfile)")
	saveCmd.Flags().BoolVar(&saveCmdIncludeTLS, "tls", false, "Include certificates, private keys and certificate chains in archive")

	saveCmd.Flags().SortFlags = false
}

//go:embed _docs/save.md
var saveCmdDescription string

var compression = map[string]string{
	"gzip":  ".gz",
	"bzip2": ".bz2",
	"none":  "",
}

var saveCmd = &cobra.Command{
	Use:     "save [flags] [TYPE] [NAME...]",
	Aliases: []string{"backup"},
	Short:   "Save Instances",
	Long:    saveCmdDescription,
	Example: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:        "true",
		CmdRequireHome:   "true",
		CmdWildcardNames: "true",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		var archive string

		ct, names := ParseTypeNames(command)

		// if no host is given, make it local only
		h := geneos.GetHost(Hostname)
		if h == geneos.ALL {
			h = geneos.LOCAL
		}

		suffix, ok := compression[saveCmdCompression]
		if !ok {
			return fmt.Errorf("invalid compression type, select one of: %s", strings.Join(slices.Collect(maps.Keys(compression)), ", "))
		}

		if !saveCmdIncludeAll {
			maxsize, err = units.ParseStrictBytes(saveCmdLimitSize)
			if err != nil {
				return fmt.Errorf("invalid size: %w", err)
			}
		}

		if saveCmdOutput != "" {
			if strings.HasSuffix(saveCmdOutput, "/") {
				// user has asked for a directory, so try to create it as well
				archive = saveCmdOutput
				os.MkdirAll(archive, 0664)
			} else {
				// check for existing file, else try to create the directories above it
				st, err := os.Stat(saveCmdOutput)
				if err == nil {
					// file (or directory) exists, use it
					archive = saveCmdOutput
					// destination may be an existing directory, in which case append a '/'
					if st.IsDir() {
						archive += "/"
					}
				} else {
					if errors.Is(err, os.ErrNotExist) {
						archive = saveCmdOutput
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

			// include host name if not local
			if h != geneos.LOCAL {
				archive += "-" + h.String()
			}

			// include component name if given on command line
			if ct != nil {
				archive += "-" + ct.String()
			}

			instances := instance.Instances(h, ct, instance.FilterNames(names...))
			switch len(instances) {
			case 0:
				return fmt.Errorf("no matching instances found")
			case 1:
				i = instances[0]
				if ct == nil {
					archive += "-" + i.Type().String()
				}
				archive += "-" + strings.ReplaceAll(i.Name(), " ", "_")
			default:
				if len(names) == 1 {
					archive += "-" + strings.ReplaceAll(names[0], " ", "_")
				}
			}

			if saveCmdIncludeDatetime {
				archive += "-" + time.Now().Local().Format("20060102150405")
			}

			archive += ".tar"
			if suffix != "" {
				archive += suffix
			}
		}

		// contents is a list of files so that instances do not duplicate shared contents
		var contents []string
		resp := instance.Do(h, ct, names, saveInstance, &contents)
		if archive != "-" {
			resp.Write(os.Stdout)
		} else {
			resp.Write(os.Stderr)
		}

		slices.Sort(contents)
		contents = slices.Compact(contents)

		var out *os.File
		if archive == "-" {
			out = os.Stdout
		} else {
			out, err = os.Create(archive)
			if err != nil {
				return
			}
			defer out.Close()
		}

		var cout io.WriteCloser

		switch saveCmdCompression {
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
			if !fi.Mode().IsRegular() && !fi.IsDir() {
				return fmt.Errorf("%s is not a regular file or directory", p)
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
			if !fi.IsDir() {
				b, err := h.ReadFile(p)
				if err != nil {
					return err
				}
				if _, err = tw.Write(b); err != nil {
					return err
				}
			}
		}

		return nil
	},
}

func saveInstance(i geneos.Instance, params ...any) (resp *instance.Response) {
	var ignoreDirs, ignoreFiles []string

	resp = instance.NewResponse(i)

	cf := i.Config()
	ct := i.Type()

	if len(params) == 0 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}
	contents, ok := params[0].(*[]string)
	if !ok {
		log.Debug().Msgf("invalid contents parameter (%T and not `*[]string`)", contents)
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	if !saveCmdIncludeAll {
		ignore := strings.Split(config.GetString(ct.CleanList, config.Default(ct.ConfigAliases[ct.CleanList])), ":")
		ignore = append(ignore, strings.Split(config.GetString(ct.PurgeList, config.Default(ct.ConfigAliases[ct.PurgeList])), ":")...)
		if geneos.RootComponent.CleanList != "" {
			ignore = append(ignore, filepath.SplitList(geneos.RootComponent.CleanList)...)
		}
		if geneos.RootComponent.PurgeList != "" {
			ignore = append(ignore, filepath.SplitList(geneos.RootComponent.PurgeList)...)
		}

		if !saveCmdIncludeAES {
			ignore = append(ignore, "*.aes", "keyfiles/")
		}

		for _, i := range ignore {
			if strings.HasSuffix(i, "/") {
				ignoreDirs = append(ignoreDirs, strings.TrimSuffix(i, "/"))
			} else {
				ignoreFiles = append(ignoreFiles, i)
			}
		}
	}

	var ignoreSecure []string
	if !saveCmdIncludeAES {
		ignoreSecure = append(ignoreSecure, "keyfile", "prevkeyfile")
	}
	if !saveCmdIncludeTLS {
		ignoreSecure = append(ignoreSecure, "certificate", "privatekey", "certchain")
	}
	for _, ig := range ignoreSecure {
		if cf.IsSet(ig) {
			ignoreFiles = append(ignoreFiles, strings.TrimPrefix(cf.GetString(ig), i.Home()+"/"))
		}
	}

	// walk the instance home directory first
	if err := walkDir(
		i.Host(),
		i.Home(),
		strings.TrimPrefix(i.Home(), i.Host().PathTo()+"/"),
		contents,
		ignoreDirs,
		ignoreFiles,
	); err != nil {
		log.Debug().Err(err).Msg("")
	}

	if !saveCmdIncludeShared {
		return
	}

	// then walk the shared directory, checking and updating contents
	if err := walkDir(
		i.Host(),
		ct.Shared(i.Host()),
		strings.TrimPrefix(ct.Shared(i.Host()), i.Host().PathTo()+"/"),
		contents,
		ignoreDirs,
		ignoreFiles,
	); err != nil {
		log.Debug().Err(err).Msg("")
	}

	// add the gateway includes directory if `--shared` is selected
	if ct.IsA("gateway") {
		if err := walkDir(
			i.Host(),
			i.Host().PathTo(ct, "includes"),
			strings.TrimPrefix(i.Host().PathTo(ct, "includes"), i.Host().PathTo()+"/"),
			contents,
			ignoreDirs,
			ignoreFiles,
		); err != nil {
			log.Debug().Err(err).Msg("")
		}
	}

	return
}

func walkDir(h *geneos.Host, d, r string, contents *[]string, ignoreDirs, ignoreFiles []string) error {
	return h.WalkDir(d, func(file string, di fs.DirEntry, err error) error {
		if err != nil {
			log.Debug().Err(err).Msg(d)
			return err
		}
		fi, err := di.Info()
		if err != nil {
			return err
		}
		switch {
		case fi.IsDir():
			if !saveCmdIncludeAll {
				for _, ig := range ignoreDirs {
					if match, _ := filepath.Match(ig, file); match {
						return fs.SkipDir
					}
				}
			}
			*contents = append(*contents, filepath.Join(r, file)+"/")
			return nil
		case fi.Mode()&fs.ModeSymlink != 0:
			log.Debug().Msgf("ignoring symlink %s", file)
		default:
			if !saveCmdIncludeAll {
				for _, ig := range ignoreFiles {
					if match, _ := filepath.Match(ig, file); match {
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
}
