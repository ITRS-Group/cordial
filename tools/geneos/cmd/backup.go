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
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/units"
	"github.com/dsnet/compress/bzip2"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

var backupCmdOutput string

var backupCmdIncludeAll, backupCmdIncludeShared bool
var backupCmdIncludeAES, backupCmdIncludeTLS bool
var backupCmdIncludeDatetime bool
var backupCmdLimitSize, backupCmdCompression string

var maxsize int64

func init() {
	Cmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVarP(&backupCmdOutput, "output", "o", "",
		`Write to "DEST". Without a destination filename the command creates
a file in the current directory with a name based on the contents of the
archive. If DEST is a directory or has a "/" suffix then the file is
written to that directory using the same naming format as if no file was
given. Directories are created as required.`,
	)

	backupCmd.Flags().BoolVarP(&backupCmdIncludeDatetime, "datetime", "D", false,
		"Add a datetime string (YYYYMMDDhhmmss) the auto-generated file names",
	)

	backupCmd.Flags().StringVarP(&backupCmdCompression, "compress", "z", "gzip",
		"Compression `type`. One of `gzip`, `bzip2` or `none`.",
	)

	backupCmd.Flags().StringVarP(&backupCmdLimitSize, "size", "s", "2MiB",
		`Skip files larger than this size unless --all is used. Accepts suffixes
with common scale units such as K, M, G with both B and iB units,
e.g. "2MiB". "0" (zero) means no limit to file sizes.`,
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
		CmdGlobal:                "true",
		CmdRequireHome:           "true",
		CmdWildcardNames:         "true",
		CmdAllInstancesMustMatch: "true",
		CmdNonInstanceArgsError:  "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		var archive string

		ct, names, _, err := FetchArgs(command)
		if err != nil {
			return err
		}

		if ct == nil && len(names) == 0 {
			return command.Usage()
		}

		// if no host is given, make it local only
		h := geneos.GetHost(Hostname)
		if h == nil || h == geneos.ALL {
			h = geneos.LOCAL
		}

		suffix, ok := compression[backupCmdCompression]
		if !ok {
			return fmt.Errorf("invalid compression type, select one of: %s", strings.Join(slices.Sorted(maps.Keys(compression)), ", "))
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
				if err = os.MkdirAll(archive, 0775); err != nil {
					return err
				}
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
						if archive != "-" {
							if err = os.MkdirAll(filepath.Dir(archive), 0775); err != nil {
								return err
							}
						}
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

			instances := instance.Instances(h, ct, instance.MatchNames(names...))
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
		var contents []backupEntry
		resp := instance.Do(command.CommandPath(), h, ct, names, getInstanceFilePaths, &contents)

		// messages must not be mixed with an archive written to stdout
		out2 := os.Stdout
		if archive == "-" {
			out2 = os.Stderr
		}
		// an archive is still written for the instances that succeeded
		reportErr := resp.Report(out2)
		if backupCmdIncludeShared {
			fmt.Fprintln(out2, "matching shared directories also included")
		}

		slices.SortFunc(contents, func(a, b backupEntry) int { return strings.Compare(a.name, b.name) })
		contents = slices.CompactFunc(contents, func(a, b backupEntry) bool { return a.name == b.name })

		if archive == "-" {
			if err = writeBackup(h, contents, os.Stdout, backupCmdCompression); err != nil {
				return err
			}
		} else {
			if err = createBackup(h, contents, archive, backupCmdCompression); err != nil {
				return err
			}
		}

		fmt.Fprintln(out2, "archive written to", archive)
		return reportErr
	},
}

// createBackup writes the archive to the file named archive, removing
// it if it cannot be completed.
func createBackup(h *geneos.Host, contents []backupEntry, archive, compression string) (err error) {
	out, err := os.Create(archive)
	if err != nil {
		return err
	}
	err = writeBackup(h, contents, out, compression)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(archive)
	}
	return
}

// writeBackup writes a tar archive of contents, relative to the Geneos
// directory on host h, to out using the given compression. All write
// and close errors are returned.
func writeBackup(h *geneos.Host, contents []backupEntry, out io.Writer, compression string) (err error) {
	var w io.Writer

	switch compression {
	case "gzip":
		var zw *gzip.Writer
		if zw, err = gzip.NewWriterLevel(out, gzip.BestCompression); err != nil {
			return
		}
		defer func() { err = errors.Join(err, zw.Close()) }()
		w = zw
	case "bzip2":
		var zw io.WriteCloser
		if zw, err = bzip2.NewWriter(out, &bzip2.WriterConfig{Level: 9}); err != nil {
			return
		}
		defer func() { err = errors.Join(err, zw.Close()) }()
		w = zw
	default:
		w = out
	}

	tw := tar.NewWriter(w)
	defer func() { err = errors.Join(err, tw.Close()) }()

	root := h.PathTo()
	for _, e := range contents {
		p := h.Join(root, e.name)
		fi := e.info
		var link string
		if fi.Mode()&fs.ModeSymlink != 0 {
			link, err = h.Readlink(p)
			if err != nil {
				return err
			}
			if link, err = geneos.ArchiveSymlinkTarget(h, root, e.name, link); err != nil {
				return err
			}
		} else if !fi.Mode().IsRegular() && !fi.IsDir() {
			return fmt.Errorf("%s is not a regular file, directory or symlink", p)
		}
		th, err := tar.FileInfoHeader(fi, link)
		if err != nil {
			return err
		}
		th.Name = e.name
		th.Format = tar.FormatUnknown
		th.Uid, th.Gid = host.GetFileOwner(h, fi)
		if err = tw.WriteHeader(th); err != nil {
			return err
		}
		if fi.IsDir() || fi.Mode()&fs.ModeSymlink != 0 {
			continue
		}
		if err = copyFile(h, p, tw, fi.Size()); err != nil {
			return err
		}
	}
	return nil
}

// copyFile streams exactly size bytes of the file p on host h to w. The
// size is that written to the tar header, so the file is rejected if it
// shrank since it was found and any additional data is ignored.
func copyFile(h *geneos.Host, p string, w io.Writer, size int64) error {
	f, err := h.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = io.CopyN(w, f, size); err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	return nil
}

// backupEntry is a file, directory or symlink to be included in a backup, with
// the name relative to the Geneos directory and the file information
// found when walking the directory
type backupEntry struct {
	name string
	info fs.FileInfo
}

// getInstanceFilePaths returns a list of paths to backup for the
// instance and returns the results in the backupEntry slice pointer in
// params[0].
func getInstanceFilePaths(i geneos.Instance, params ...any) (resp *responses.General) {
	var ignoreDirs, ignoreFiles []string

	resp = responses.New[responses.General](i)

	cf := i.Config()
	ct := i.Type()

	if len(params) == 0 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}
	contents, ok := params[0].(*[]backupEntry)
	if !ok {
		i.Log().Debug("invalid contents parameter", slog.Any("contents", params[0]))
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	if !backupCmdIncludeAll {
		ignore := strings.Split(ct.GlobalSetting(ct.CleanList), ":")
		ignore = append(ignore, strings.Split(ct.GlobalSetting(ct.PurgeList), ":")...)
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
			if before, ok := strings.CutSuffix(i, "/"); ok {
				ignoreDirs = append(ignoreDirs, before)
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
		ignoreSecure = append(ignoreSecure,
			// legacy path parameters
			instance.CERTIFICATE,
			instance.PRIVATEKEY,
			instance.CERTCHAIN,
			// new path parameters, under TLSBASE
			cf.Join(instance.TLSBASE, instance.CERTIFICATE),
			cf.Join(instance.TLSBASE, instance.PRIVATEKEY),
			cf.Join(instance.TLSBASE, instance.CABUNDLE),
		)
	}
	for _, ig := range ignoreSecure {
		if p, ok := config.Lookup[string](cf, ig); ok {
			ignoreFiles = append(ignoreFiles, strings.TrimPrefix(p, i.Home()+"/"))
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
		false,
	); err != nil {
		// missing dirs and inaccessible files are probably not errors
		i.Log().Debug("error walking instance home directory", slog.Any("error", err))
	}

	resp.Completed = []string{"included in backup"}

	// add global tls directory, in all cases
	if err := walkDir(i.Host(), i.Host().PathTo("tls")+"/", "tls", contents, ignoreDirs, []string{}, false); err != nil {
		i.Log().Debug("error walking global TLS directory", slog.Any("error", err))
	}

	if !backupCmdIncludeShared {
		return
	}

	// then walk the shared directories, if any, for ct checking and
	// updating contents.
	for _, s := range ct.SharedDirectories {
		if err := walkDir(
			i.Host(),
			i.Host().PathTo(s),
			s,
			contents,
			ignoreDirs,
			ignoreFiles,
			true,
		); err != nil {
			// missing dirs and inaccessible files are probably not errors
			i.Log().Debug("error walking shared directory", slog.Any("error", err))
		}
	}

	return
}

var contentsMutex sync.Mutex

// walkDir walks the given directory on host h, adding files and directories
// to contents slice, using relative as the base path. ignoreDirs and ignoreFiles
// are patterns to skip. walkDir is concurrency safe for the contents slice.
func walkDir(h *geneos.Host, dir, relative string, contents *[]backupEntry, ignoreDirs, ignoreFiles []string, includeSymlinks bool) error {
	return h.WalkDir(dir, func(file string, di fs.DirEntry, err error) error {
		if err != nil {
			log.Debug("error walking directory", slog.Any("error", err), slog.String("dir", dir))
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
			contentsMutex.Lock()
			defer contentsMutex.Unlock()
			*contents = append(*contents, backupEntry{name: filepath.Join(relative, file) + "/", info: fi})
			return nil
		case !fi.Mode().IsRegular() && fi.Mode()&fs.ModeSymlink == 0:
			log.Debug("ignoring special file", slog.String("file", file))
		default:
			if fi.Mode()&fs.ModeSymlink != 0 && !includeSymlinks {
				return nil
			}
			if !backupCmdIncludeAll {
				for _, ig := range ignoreFiles {
					if match, _ := filepath.Match(ig, file); match {
						return nil
					}
				}
				if maxsize != 0 && fi.Size() > maxsize {
					log.Debug("skipping large file", slog.String("file", filepath.Join(relative, file)))
					return nil
				}
			}
			contentsMutex.Lock()
			defer contentsMutex.Unlock()
			*contents = append(*contents, backupEntry{name: filepath.Join(relative, file), info: fi})
			return nil
		}
		return nil
	})
}
