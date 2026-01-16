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
	"archive/tar"
	"compress/gzip"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/dsnet/compress/bzip2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var restoreCmdCommon, restoreCmdCompression string
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
		ct, names, params := ParseTypeNamesParams(command)

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
			if restoreCmdCompression == "" {
				return fmt.Errorf("when reading from STDIN the compression type must be selected using --decompress/-z")
			}
			if err = restoreFromFile(h, ct, "-", names); err != nil {
				return
			}
		} else {
			for _, f := range files {
				// process file
				log.Debug().Msgf("checking file %s for ct %s and names %v", f, ct, names)
				if err = restoreFromFile(h, ct, f, names); err != nil {
					return
				}
			}
		}

		return
	},
}

// restoreFromFile reads the archive and looks for types ct and names,
// which can be in the form "dest=src" to allow renaming instances. If
// ct is nil then all matching types are restored and if names it empty
// then all instances are restored. Existing instances with matching
// names are not overwritten.
//
// TODO: shared file control (use flag selector and limit to ct, if not
// nil)
func restoreFromFile(h *geneos.Host, ct *geneos.Component, archive string, names []string) (err error) {
	var tin io.ReadCloser
	var fileType string

	if archive == "-" {
		fileType = restoreCmdCompression
		archive = ""
	} else {
		for s, t := range fileTypes {
			if strings.HasSuffix(archive, s) {
				fileType = t
				break
			}
		}
	}

	if fileType == "" {
		return
	}

	// default source is STDIN
	in := os.Stdin
	if archive != "" {
		if in, err = os.Open(archive); err != nil {
			return
		}
		defer in.Close()
	}

	switch fileType {
	case "", "none":
		switch {
		case strings.HasSuffix(archive, ".gz") || strings.HasSuffix(archive, ".tgz"):
			if tin, err = gzip.NewReader(in); err != nil {
				return
			}
			defer tin.Close()
		case strings.HasSuffix(archive, ".bz2"):
			if tin, err = bzip2.NewReader(in, nil); err != nil {
				return
			}
			defer tin.Close()
		default:
			tin = in
		}
	case "gzip":
		if tin, err = gzip.NewReader(in); err != nil {
			return
		}
		defer tin.Close()
	case "bzip2":
		if tin, err = bzip2.NewReader(in, nil); err != nil {
			return
		}
		defer tin.Close()
	default:
		err = fmt.Errorf("unknown decompression type %s", restoreCmdCompression)
		return
	}

	// store paths processed mapped to types and instances, with record
	// of skipping, to prevent overwrites, to anticipate instances that
	// are spread out

	// sizes - key is type:name and the value is the number of files
	// written out. Initial value, when type:name is found in archive,
	// is -1 to indicate destination either not tested or already
	// exists.
	sizes := map[string]int64{}

	// mapping instance names from archive names to potentially new
	// ones.
	//
	// wildcards are only allowed when no renaming is taking place
	mapping := map[string]string{}

	restoreAll := false
	if len(names) == 1 && names[0] == "all" {
		restoreAll = true
	} else {
		for _, name := range names {
			dest, src, found := strings.Cut(name, "=")
			if found {
				if !instance.ValidName(src) || !instance.ValidName(dest) {
					log.Debug().Msgf("invalid instance name format when using DEST=SRC format: %s", name)
					return geneos.ErrInvalidArgs
				}
				mapping[dest] = src
			} else {
				mapping[name] = name
			}
		}
	}

	tr := tar.NewReader(tin)
	for {
		var ctSubdir string
		var hdr *tar.Header

		hdr, err = tr.Next()
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			return
		}

		filename, err2 := geneos.CleanRelativePath(hdr.Name)
		if err2 != nil {
			return err2
		}

		// decompose cleaned path, check for component type at top
		// level, ignore all other dirs
		ctName, rest, _ := strings.Cut(filename, "/")

		// restore top level tls is restoring shared files
		if ctName == "tls" && restoreCmdShared {
			if _, ok := sizes["TLS:!SHARED"]; !ok {
				sizes["TLS:!SHARED"] = -1
			}
			if err = writeSharedFile(h, path.Join("tls", rest), tr, hdr); err != nil {
				if errors.Is(err, os.ErrExist) && restoreCmdList {
					// up the count regardless of existence when listing
					sizes["TLS:!SHARED"] += hdr.Size
				}
				continue
			}
			sizes["TLS:!SHARED"] += hdr.Size
			continue
		}

		// check ctName is valid and if ct is not nil, filter for a match
		nct := geneos.ParseComponent(ctName)
		if nct == nil {
			log.Debug().Msgf("unknown component type: %s, skipping", filename)
			continue
		}

		// if a component type is wanted, reject others
		if ct != nil && ct != nct {
			log.Debug().Msgf("component type does not match: %s != %s, skipping", ct, nct)
			continue
		}

		if rest != "" {
			ctSubdir, rest, _ = strings.Cut(rest, "/")
		}

		// check for shared directories for given nct and restore if asked to
		if restoreCmdShared {
			if slices.Contains(nct.SharedDirectories, path.Join(nct.String(), ctSubdir)) {
				if ct == nil || ct == nct {
					if _, ok := sizes[nct.String()+":!SHARED"]; !ok {
						sizes[nct.String()+":!SHARED"] = -1
					}
					if err = writeSharedFile(h, path.Join(nct.String(), ctSubdir, rest), tr, hdr); err != nil {
						if errors.Is(err, os.ErrExist) && restoreCmdList {
							// up the count regardless of existence when listing
							sizes[nct.String()+":!SHARED"] += hdr.Size
						}
						continue
					}
					sizes[nct.String()+":!SHARED"] += hdr.Size
					continue
				}
			}
		}

		if !strings.HasSuffix(ctSubdir, "s") {
			continue
		}
		ctSubdir = strings.TrimSuffix(ctSubdir, "s")
		packageCt := geneos.ParseComponent(ctSubdir)

		if packageCt == nil || (nct != packageCt && nct != packageCt.ParentType) {
			log.Debug().Msgf("top-level entry and home entry not matched: %s, skipping", filename)
			continue
		}

		// "rest" is (should be) now "instance/content"
		i, fp, _ := strings.Cut(rest, "/")

		// we now have type (nct), instance name and filepath fp

		// does the instance name match any of the names list, including wildcards?
		if len(mapping) > 0 {
			for k, v := range mapping {
				if k == v {
					if matched, _ := filepath.Match(k, i); matched {
						// save
						if err = processFile(h, packageCt, i, fp, tr, hdr, sizes); err != nil {
							return
						}
					}
				} else {
					if v == i {
						// rename and save
						if err = processFile(h, packageCt, k, fp, tr, hdr, sizes); err != nil {
							return
						}
					}
				}
			}
			continue
		}

		// otherwise, if "all", restore all instances in the archive
		if restoreAll {
			if err = processFile(h, packageCt, i, fp, tr, hdr, sizes); err != nil {
				return
			}
		}
	}

	fmt.Printf("\n%s:\n\n", archive)
	t := tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
	fmt.Fprintf(t, "Type\tName\tSize\n")
	for _, k := range slices.Sorted(maps.Keys(sizes)) {
		ctName, name, _ := strings.Cut(k, ":")

		if name == "!SHARED" {
			name = "(shared dirs)"
		}
		if restoreCmdList {
			fmt.Fprintf(t, "%s\t%s\t%.3f MiB\n", ctName, name, float64(sizes[k])/(1024*1024))
			continue
		}

		if sizes[k] > -1 {
			if name != mapping[name] && !restoreAll {
				fmt.Fprintf(t, "%s\t%s (from %s)\t%.3f MiB\n", ctName, name, mapping[name], float64(sizes[k])/(1024*1024))
			} else {
				fmt.Fprintf(t, "%s\t%s\t%.3f MiB\n", ctName, name, float64(sizes[k])/(1024*1024))
			}

		} else {
			fmt.Fprintf(t, "%s\t%s\tskipped restoring\n", ctName, name)
		}
	}
	t.Flush()
	return
}

func processFile(h *geneos.Host, ct *geneos.Component, i, fp string, tr *tar.Reader, hdr *tar.Header, sizes map[string]int64) (err error) {
	// otherwise restore all instances in the archive
	if process, ok := sizes[ct.String()+":"+i]; ok {
		if process > 0 {
			if !restoreCmdList {
				if err = writeFile(h, ct, i, fp, tr, hdr); err != nil {
					// if written ok, add one more file
					return
				}
			}
			sizes[ct.String()+":"+i] += hdr.Size
		}
		return
	}

	// init sizes entry
	if restoreCmdList {
		sizes[ct.String()+":"+i] = hdr.Size
		return
	}

	// otherwise init to -1
	sizes[ct.String()+":"+i] = -1

	// write file and update sizes
	if _, err = h.Stat(h.PathTo(ct, ct.String()+"s", i)); err != nil {
		// instance does not yet exist
		if err = writeFile(h, ct, i, fp, tr, hdr); err != nil {
			return
		}
		// first file written
		sizes[ct.String()+":"+i] = hdr.Size
	}
	return
}

// writeFile reads the file from the tar reader and, if a regular file,
// writes it to the instance directory, creating intermediate
// directories as required, and setting permissions as per tar header.
// Owner and group are ignored. Directory entries are used to create
// directories with matching permissions, again ignoring owner and
// group.
func writeFile(h *geneos.Host, ct *geneos.Component, instance string, fp string, tr *tar.Reader, hdr *tar.Header) (err error) {
	if ct == nil {
		return geneos.ErrInvalidArgs
	}

	instanceDir := h.PathTo(ct, ct.String()+"s", instance)
	destPath := path.Join(instanceDir, fp)

	switch hdr.Typeflag {
	case tar.TypeDir:
		// create and set permissions
		return h.MkdirAll(destPath, hdr.FileInfo().Mode().Perm())
	case tar.TypeReg:
		var w io.WriteCloser

		// create intermediate dirs, write, set perms
		if err = h.MkdirAll(path.Dir(destPath), 0775); err != nil {
			return
		}

		if fp == ct.String()+".json" {
			return rebuildConfig(h, ct, instance, instanceDir, tr)
		}

		if w, err = h.Create(destPath, hdr.FileInfo().Mode()); err != nil {
			log.Debug().Err(err).Msg("")
			return
		}
		defer w.Close()

		_, err = io.Copy(w, tr)
		return
	default:
		// ignore
	}
	return
}

// writeSharedFile reads the file from the tar reader and, if a regular
// file, writes it to the path fp relative to the host's root geneos,
// creating intermediate directories as required, and setting
// permissions as per tar header. Owner and group are ignored. Directory
// entries are used to create directories with matching permissions,
// again ignoring owner and group.
func writeSharedFile(h *geneos.Host, fp string, tr *tar.Reader, hdr *tar.Header) (err error) {
	switch hdr.Typeflag {
	case tar.TypeDir:
		// create and set permissions
		if _, err := h.Stat(h.PathTo(fp)); err == nil {
			return os.ErrExist
		}
		return h.MkdirAll(h.PathTo(fp), hdr.FileInfo().Mode().Perm())
	case tar.TypeReg:
		var w io.WriteCloser

		if _, err := h.Stat(h.PathTo(fp)); err == nil {
			return os.ErrExist
		}

		// create intermediate dirs, write, set perms
		if err = h.MkdirAll(path.Dir(h.PathTo(fp)), 0775); err != nil {
			return
		}

		if w, err = h.Create(h.PathTo(fp), hdr.FileInfo().Mode()); err != nil {
			return
		}
		defer w.Close()

		_, err = io.Copy(w, tr)
		return
	default:
		// ignore
	}
	return
}

func rebuildConfig(h *geneos.Host, ct *geneos.Component, i, instanceDir string, r io.Reader) (err error) {
	// restore config, update parameters for new root dir on dest host, write
	cf, err := config.Load(ct.String(), config.SetConfigReader(r))
	if err != nil {
		return err
	}

	// update name in case this is a rename
	cf.Set("name", i)

	oldHome := cf.GetString("home")
	newHome := instanceDir

	oldInstall := cf.GetString("install")
	newInstall := h.PathTo("packages", ct.String())

	oldShared := path.Join(path.Dir(path.Dir(oldHome)), ct.String()+"_shared")
	newShared := ct.Shared(h)

	version := cf.GetString("version", config.NoExpand())

	for k, v := range cf.AllSettings() {
		switch k {
		case "libpaths":
			// treat libpaths special, below
			continue
		case "port":
			ports := instance.GetAllPorts(h)
			if ports[cf.GetUint16(k)] {
				// port already in use, get the next one
				cf.Set(k, instance.NextFreePort(h, ct))
			}
		default:
			if vs, ok := v.(string); ok {
				// replace home (unanchored)
				if oldHome != newHome {
					vs = strings.Replace(vs, oldHome, newHome, 1)
				}

				// replace install (unanchored)
				if oldInstall != newInstall {
					vs = strings.Replace(vs, oldInstall, newInstall, 1)
				}

				// replace shared (unanchored)
				if oldShared != newShared {
					vs = strings.Replace(vs, oldShared, newShared, 1)
				}

				cf.Set(k, vs)
			}
		}
	}

	libpaths := []string{}
	np := "${config:install}"
	nv := "${config:version}"

	for _, p := range filepath.SplitList(cf.GetString("libpaths", config.NoExpand())) {
		if strings.HasPrefix(p, oldInstall+"/") {
			subpath := strings.TrimPrefix(p, oldInstall+"/")
			if strings.HasPrefix(subpath, version) {
				libpaths = append(libpaths, path.Join(np, nv, strings.TrimPrefix(subpath, version)))
			} else {
				libpaths = append(libpaths, path.Join(np, subpath))
			}
		} else {
			libpaths = append(libpaths, p)
		}
	}
	cf.Set("libpaths", strings.Join(libpaths, ":"))

	if err = cf.Save(ct.String(),
		config.Host(h),
		config.AddDirs(instanceDir),
		config.SetAppName(i),
	); err != nil {
		return err
	}

	return
}
