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

var backupCmdOutput string

var backupCmdIncludeAll, backupCmdIncludeShared bool
var backupCmdIncludeAES, backupCmdIncludeTLS bool
var backupCmdIncludeDatetime bool
var backupCmdLimitSize, backupCmdCompression string

var maxsize int64

func init() {
	GeneosCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVarP(&backupCmdOutput, "output", "o", "",
		"Write to `DEST`. Without a destination filename the command creates\na file name based on the contents of the archive. If DEST is a directory\nor has a '/' suffix then the file is written to that directory using the\nsame naming format as if no file was given. Directories are created\nas required.",
	)

	backupCmd.Flags().BoolVarP(&backupCmdIncludeDatetime, "datetime", "D", false,
		"Add a datetime string (YYYYMMDDhhmmss) the auto-generated file names",
	)

	backupCmd.Flags().StringVarP(&backupCmdCompression, "compress", "z", "gzip",
		"Compression `type`. One of `gzip`, `bzip2` or `none`.",
	)

	backupCmd.Flags().StringVarP(&backupCmdLimitSize, "size", "s", "2MiB",
		"Skip files larger than this size unless --all is used. Accepts suffixes\nwith common scale units such as K, M, G with both B and iB units,\ne.g. `2MiB`. `0` (zero) means no limit to file sizes.",
	)

	backupCmd.Flags().BoolVarP(&backupCmdIncludeAll, "all", "A", false,
		"Include all files except AES key files, certificates and private keys.",
	)

	backupCmd.Flags().BoolVar(&backupCmdIncludeShared, "shared", false,
		"Include per-component shared directories from outside instance directories.\n",
	)

	backupCmd.Flags().BoolVar(&backupCmdIncludeAES, "aes", false,
		"Include AES key files.",
	)
	backupCmd.Flags().BoolVar(&backupCmdIncludeTLS, "tls", false,
		"Include certificates, private keys and certificate chains.",
	)

	backupCmd.Flags().SortFlags = false
}

//go:embed _docs/backup.md
var backupCmdDescription string

var compression = map[string]string{
	"gzip":  ".gz",
	"bzip2": ".bz2",
	"none":  "",
}

var backupCmd = &cobra.Command{
	Use:     "backup [flags] [all] | [TYPE] [NAME...]",
	Aliases: []string{"save"},
	GroupID: CommandGroupConfig,
	Short:   "Backup Instances",
	Long:    backupCmdDescription,
	Example: strings.ReplaceAll(`
geneos backup gateway gw1
geneos backup all
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:        "false",
		CmdRequireHome:   "true",
		CmdWildcardNames: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		var archive string

		ct, names := ParseTypeNames(command)

		if ct == nil && len(names) == 0 {
			return command.Usage()
		}

		// if no host is given, make it local only
		h := geneos.GetHost(Hostname)
		if h == geneos.ALL {
			h = geneos.LOCAL
		}

		suffix, ok := compression[backupCmdCompression]
		if !ok {
			return fmt.Errorf("invalid compression type, select one of: %s", strings.Join(slices.Collect(maps.Keys(compression)), ", "))
		}

		if !backupCmdIncludeAll {
			maxsize, err = units.ParseStrictBytes(backupCmdLimitSize)
			if err != nil {
				return fmt.Errorf("invalid size: %w", err)
			}
		}

		if backupCmdOutput != "" {
			if strings.HasSuffix(backupCmdOutput, "/") {
				// user has asked for a directory, so try to create it as well
				archive = backupCmdOutput
				os.MkdirAll(archive, 0664)
			} else {
				// check for existing file, else try to create the directories above it
				st, err := os.Stat(backupCmdOutput)
				if err == nil {
					// file (or directory) exists, use it
					archive = backupCmdOutput
					// destination may be an existing directory, in which case append a '/'
					if st.IsDir() {
						archive += "/"
					}
				} else {
					if errors.Is(err, os.ErrNotExist) {
						archive = backupCmdOutput
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

			if backupCmdIncludeDatetime {
				archive += "-" + time.Now().Local().Format("20060102150405")
			}

			archive += ".tar"
			if suffix != "" {
				archive += suffix
			}
		}

		// contents is a list of files so that instances do not duplicate shared contents
		var contents []string
		resp := instance.Do(h, ct, names, getInstanceFilePaths, &contents)

		out2 := os.Stderr
		if archive != "-" {
			out2 = os.Stdout
		}
		resp.Write(out2)
		if backupCmdIncludeShared {
			fmt.Println("matching shared directories also included")
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

		var w io.WriteCloser

		switch backupCmdCompression {
		case "gzip":
			w, err = gzip.NewWriterLevel(out, gzip.BestCompression)
			if err != nil {
				return
			}
			defer w.Close()
		case "bzip2":
			w, err = bzip2.NewWriter(out, &bzip2.WriterConfig{Level: 9})
			if err != nil {
				return
			}
			defer w.Close()
		case "none":
			w = out
		}

		tw := tar.NewWriter(w)
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

		fmt.Fprintln(out2, "archive written to", archive)
		return nil
	},
}

// getInstanceFilePaths returns a list of paths to backup for the
// instance and returns the results in the string slice pointer in
// params[0].
func getInstanceFilePaths(i geneos.Instance, params ...any) (resp *instance.Response) {
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

	if !backupCmdIncludeAll {
		ignore := strings.Split(config.GetString(ct.CleanList, config.Default(ct.ConfigAliases[ct.CleanList])), ":")
		ignore = append(ignore, strings.Split(config.GetString(ct.PurgeList, config.Default(ct.ConfigAliases[ct.PurgeList])), ":")...)
		if geneos.RootComponent.CleanList != "" {
			ignore = append(ignore, filepath.SplitList(geneos.RootComponent.CleanList)...)
		}
		if geneos.RootComponent.PurgeList != "" {
			ignore = append(ignore, filepath.SplitList(geneos.RootComponent.PurgeList)...)
		}

		if !backupCmdIncludeAES {
			ignore = append(ignore, "*.aes", "keyfiles/")
		}
		if !backupCmdIncludeTLS {
			ignore = append(ignore, "*.pem", "*.key", "*.crt")
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
	if !backupCmdIncludeAES {
		ignoreSecure = append(ignoreSecure, "keyfile", "prevkeyfile")
	}
	if !backupCmdIncludeTLS {
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
		// missing dirs and inaccessible files are probably not errors
		log.Debug().Err(err).Msg("")
	}

	resp.Completed = []string{"included in backup"}

	if !backupCmdIncludeShared {
		return
	}

	// then walk the shared directories, if any, for ct checking and updating contents
	for _, s := range ct.SharedDirectories {
		if err := walkDir(
			i.Host(),
			i.Host().PathTo(s),
			s,
			contents,
			ignoreDirs,
			ignoreFiles,
		); err != nil {
			// missing dirs and inaccessible files are probably not errors
			log.Debug().Err(err).Msg("")
		}
	}

	return
}

func walkDir(h *geneos.Host, dir, relative string, contents *[]string, ignoreDirs, ignoreFiles []string) error {
	return h.WalkDir(dir, func(file string, di fs.DirEntry, err error) error {
		if err != nil {
			log.Debug().Err(err).Msg(dir)
			return err
		}
		fi, err := di.Info()
		if err != nil {
			return err
		}
		switch {
		case fi.IsDir():
			if !backupCmdIncludeAll {
				for _, ig := range ignoreDirs {
					if match, _ := filepath.Match(ig, file); match {
						return fs.SkipDir
					}
				}
			}
			*contents = append(*contents, filepath.Join(relative, file)+"/")
			return nil
		case fi.Mode()&fs.ModeSymlink != 0:
			log.Debug().Msgf("ignoring symlink %s", file)
		default:
			if !backupCmdIncludeAll {
				for _, ig := range ignoreFiles {
					if match, _ := filepath.Match(ig, file); match {
						return nil
					}
				}
				if maxsize != 0 && fi.Size() > maxsize {
					log.Debug().Msgf("skipping large file %s", filepath.Join(relative, file))
					return nil
				}
			}
			*contents = append(*contents, filepath.Join(relative, file))
			return nil
		}
		return nil
	})
}
