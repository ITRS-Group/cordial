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
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/dsnet/compress/bzip2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var loadCmdCommon, loadCmdFile, loadCmdCompression string
var loadCmdInstanceShared bool

func init() {
	GeneosCmd.AddCommand(loadCmd)

	loadCmd.Flags().StringVarP(&loadCmdFile, "input", "i", "", "import one or more instances from `FILE`\nFILE can be `-` for STDIN")

	loadCmd.Flags().BoolVarP(&loadCmdInstanceShared, "shared", "s", false, "include shared files when using --instances")

	loadCmd.Flags().StringVarP(&loadCmdCompression, "decompress", "z", "", "use decompression `type`, one of `gzip`, `bzip2` or `none`\nif not given then the file name is used to guess the type")

	loadCmd.Flags().SortFlags = false
}

var fileTypes = map[string]string{
	".tar.gz":  "gzip",
	".tgz":     "gzip",
	".tar.bz2": "bz2",
	".tar":     "none",
}

//go:embed _docs/load.md
var loadCmdDescription string

var loadCmd = &cobra.Command{
	Use:     "load [flags] [TYPE] [[DEST=]NAME...]",
	GroupID: CommandGroupConfig,
	Short:   "Load Instances from archive",
	Long:    loadCmdDescription,
	Example: strings.ReplaceAll(`

geneos load gateway ABC x.tgz

`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:        "true",
		CmdRequireHome:   "true",
		CmdWildcardNames: "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {

		ct, names, params := ParseTypeNamesParams(cmd)

		h := geneos.NewHost(Hostname)
		if h == geneos.ALL {
			h = geneos.LOCAL
		}

		log.Debug().Msgf("ct %s, names %v, params %v", ct, names, params)

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

		log.Debug().Msgf("files %v, names %v", files, names)
		names = append(names, params...)

		// if no file names are found in args then assume STDIN. later
		// we could look for named file patterns, if it proves useful
		if len(files) == 0 {
			if err = loadFromFile(h, ct, "-", names); err != nil {
				return
			}
		} else {
			for _, f := range files {
				// process file
				log.Debug().Msgf("loading ct %s, host %s from file %s", ct, h, f)
				if err = loadFromFile(h, ct, f, names); err != nil {
					return
				}
			}
		}

		return
	},
}

// loadFromFile reads the archive f and looks for types ct and names,
// which can be in the form "dest=src" to allow renaming instances. If
// ct is nil then all matching types are loaded and if names it empty
// then all instances are loaded. Existing instances with matching names
// are not overwritten.
func loadFromFile(h *geneos.Host, ct *geneos.Component, f string, names []string) (err error) {
	var tin io.ReadCloser
	var fileType string

	if f == "-" {
		fileType = loadCmdCompression
		f = ""
	} else {
		for s, t := range fileTypes {
			if strings.HasSuffix(f, s) {
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
	if f != "" {
		if in, err = os.Open(f); err != nil {
			return
		}
		defer in.Close()
	}

	switch fileType {
	case "", "none":
		switch {
		case strings.HasSuffix(f, ".gz") || strings.HasSuffix(f, ".tgz"):
			if tin, err = gzip.NewReader(in); err != nil {
				return
			}
			defer tin.Close()
		case strings.HasSuffix(f, ".bz2"):
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
		err = fmt.Errorf("unknown decompression type %s", loadCmdCompression)
		return
	}

	// store paths processed mapped to types and instances, with record
	// of skipping, to prevent overwrites, to anticipate instances that
	// are spread out

	// processed - key is type:name and the value is false to skip, true
	// to started loading. a missing key means not yet seen. name is as
	// per archive file, not destination
	processed := map[string]bool{}

	// mapping instance names to potentially new ones.
	//
	// key is an entry from names, unchanged and may include '=', the
	// value is the instance name in the archive
	// mapping := map[string]geneos.Instance{}

	//
	//

	tr := tar.NewReader(tin)
	for {
		var hdr *tar.Header
		hdr, err = tr.Next()
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			return
		}
		filename := hdr.Name
		if filename, err = geneos.CleanRelativePath(filename); err != nil {
			return
		}

		ctname, rest, found := strings.Cut(filename, "/")
		if !found {
			log.Debug().Msgf("invalid top-level entry found: %s, skipping", filename)
			continue
		}

		// check ctname is valid and if ct is not nil, filter for a match
		nct := geneos.ParseComponent(ctname)
		if nct == nil {
			log.Debug().Msgf("unknown type: %s, skipping", filename)
			continue
		}

		if ct != nil && ct != nct {
			log.Debug().Msgf("type does not match wanted: %s != %s, skipping", ct, filename)
			continue
		}

		cthome, rest, found := strings.Cut(rest, "/")
		if !found {
			//
		}

		if ctname+"s" != cthome {
			log.Debug().Msgf("top-level entry and home entry not matched: %s, skipping", filename)
			continue
		}

		// "rest" is (should be) now "instance/content"
		instance, fp, found := strings.Cut(rest, "/")

		// we now have type (nct), instance name and filepath fp

		// does the instance name match any of the names list, including wildcards?
		if len(names) > 0 {
			log.Debug().Msgf("len(names) = %d", len(names))
		}

		// otherwise load all instances in the archive
		if process, ok := processed[nct.String()+":"+instance]; ok {
			if process {
				if err = writeFile(h, nct, instance, fp, tr, hdr); err != nil {
					return
				}
			}
			continue
		}

		// write file and update processed
		if _, err = h.Stat(path.Join(nct.InstancesDir(h), instance)); err != nil {
			if err = writeFile(h, nct, instance, fp, tr, hdr); err != nil {
				return
			}
			processed[nct.String()+":"+instance] = true
		} else {
			processed[nct.String()+":"+instance] = false
		}
	}

	for k, v := range processed {
		if v {
			fmt.Printf("loaded %s from %s\n", k, f)
		} else {
			fmt.Printf("skipped loading %s from %s\n", k, f)
		}
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

	instanceDir := path.Join(ct.InstancesDir(h), instance)
	destPath := path.Join(instanceDir, fp)

	switch hdr.Typeflag {
	case tar.TypeDir:
		// create and set permissions
		return h.MkdirAll(destPath, hdr.FileInfo().Mode().Perm())
	case tar.TypeReg:
		// create intermediate dirs, write, set perms
		if err = h.MkdirAll(path.Dir(destPath), 0775); err != nil {
			return
		}

		if fp == ct.String()+".json" {
			return rebuildConfig(h, ct, instance, instanceDir, tr)
		}

		var w io.WriteCloser
		if w, err = h.Create(destPath, hdr.FileInfo().Mode()); err != nil {
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

func rebuildConfig(h *geneos.Host, ct *geneos.Component, instance, instanceDir string, r io.Reader) (err error) {
	// load config, update parameters for new root dir on dest host, write
	cf, err := config.Load(ct.String(), config.SetConfigReader(r))
	if err != nil {
		log.Debug().Err(err).Msg("instance config cannot be loaded")
		return err
	}

	oldHome := cf.GetString("home")
	newHome := instanceDir

	oldInstall := cf.GetString("install")
	newInstall := h.PathTo("packages", ct.String())

	oldShared := path.Join(path.Dir(path.Dir(oldHome)), ct.String()+"_shared")
	newShared := ct.Shared(h)

	log.Debug().Msgf("oldShared %s newShared %s", oldShared, newShared)

	version := cf.GetString("version", config.NoExpand())

	for k, v := range cf.AllSettings() {
		if k == "libpaths" {
			// treat libpaths special, below
			continue
		}
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
		config.SetAppName(instance),
	); err != nil {
		return err
	}
	log.Debug().Msgf("saved config for %s", instance)

	return
}
