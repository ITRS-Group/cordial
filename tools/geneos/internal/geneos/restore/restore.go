package restore

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/dsnet/compress/bzip2"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var fileTypes = map[string]string{
	".tar.gz":  "gzip",
	".tgz":     "gzip",
	".tar.bz2": "bz2",
	".tar":     "none",
}

// Restore reads the file at the local archive path and restores
// matching instances matching component ct (all if nil) with names on
// host h. names can be in the form "dest=src" to allow renaming
// instances. If ct is nil then all matching types are restored and if
// names it empty then all instances are restored. Existing instances
// with matching names are not overwritten.
//
// TODO: shared file control (use flag selector and limit to ct, if not
// nil)
func Restore(archive string, options ...RestoreOption) (err error) {
	var tin io.ReadCloser
	var fileType string

	opts := evalRestoreOptions(options...)
	if archive == "-" {
		fileType = opts.compression
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

	names := opts.names
	ct := opts.component

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
		err = fmt.Errorf("unknown decompression type %s", opts.compression)
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

	// files counts the number of files restored for each type:name
	files := map[string]int64{}

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
		if ctName == "tls" && opts.shared {
			if _, ok := sizes["TLS:!SHARED"]; !ok {
				sizes["TLS:!SHARED"] = -1
				files["TLS:!SHARED"] = 0
			}
			if err = writeSharedFile(path.Join("tls", rest), tr, hdr, opts); err != nil {
				if errors.Is(err, os.ErrExist) && opts.list {
					// up the count regardless of existence when listing
					sizes["TLS:!SHARED"] += hdr.Size
					files["TLS:!SHARED"] += 1
				}
				continue
			}
			sizes["TLS:!SHARED"] += hdr.Size
			files["TLS:!SHARED"] += 1
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
		if opts.shared {
			if slices.Contains(nct.SharedDirectories, path.Join(nct.String(), ctSubdir)) {
				if ct == nil || ct == nct {
					if _, ok := sizes[nct.String()+":!SHARED"]; !ok {
						sizes[nct.String()+":!SHARED"] = -1
						files[nct.String()+":!SHARED"] = 0
					}
					if err = writeSharedFile(path.Join(nct.String(), ctSubdir, rest), tr, hdr, opts); err != nil {
						if errors.Is(err, os.ErrExist) && opts.list {
							// up the count regardless of existence when listing
							sizes[nct.String()+":!SHARED"] += hdr.Size
							files[nct.String()+":!SHARED"] += 1
						}
						continue
					}
					sizes[nct.String()+":!SHARED"] += hdr.Size
					files[nct.String()+":!SHARED"] += 1
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
						if err = processFile(i, fp, tr, hdr, sizes, Component(packageCt), Host(opts.host), List(opts.list)); err != nil {
							return
						}
						files[packageCt.String()+":"+i] += 1
					}
				} else {
					if v == i {
						// rename and save
						if err = processFile(k, fp, tr, hdr, sizes, Component(packageCt), Host(opts.host), List(opts.list)); err != nil {
							return
						}
						files[packageCt.String()+":"+k] += 1
					}
				}
			}
			continue
		}

		// otherwise, if "all", restore all instances in the archive
		if restoreAll {
			if err = processFile(i, fp, tr, hdr, sizes, Component(packageCt), Host(opts.host), List(opts.list)); err != nil {
				return
			}
			files[packageCt.String()+":"+i] += 1
		}
	}

	if opts.list {
		fmt.Fprintf(opts.progress, "Contents of archive %s:\n", archive)
	} else {
		fmt.Fprintf(opts.progress, "Restoring from archive %s:\n", archive)
	}
	// t := tabwriter.NewWriter(opts.progress, 3, 8, 2, ' ', 0)
	// fmt.Fprintf(t, "Type\tName\tSize\n")
	for _, k := range slices.Sorted(maps.Keys(sizes)) {
		ctName, name, _ := strings.Cut(k, ":")

		if name == "!SHARED" {
			name = "(shared dirs)"
		}
		if opts.list {
			fmt.Fprintf(opts.progress, "%s %s %d files, total %.3f MiB\n", ctName, name, files[k], float64(sizes[k])/(1024*1024))
			continue
		}

		if sizes[k] > -1 {
			if name != mapping[name] && !restoreAll {
				fmt.Fprintf(opts.progress, "%s %s (from %s) %d files, total %.3f MiB\n", ctName, name, mapping[name], files[k], float64(sizes[k])/(1024*1024))
			} else {
				fmt.Fprintf(opts.progress, "%s %s %d files, total %.3f MiB\n", ctName, name, files[k], float64(sizes[k])/(1024*1024))
			}

		} else {
			fmt.Fprintf(opts.progress, "%s %s already exists, skipping\n", ctName, name)
		}
	}
	// t.Flush()
	return
}

func processFile(i, fp string, tr *tar.Reader, hdr *tar.Header, sizes map[string]int64, options ...RestoreOption) (err error) {
	opts := evalRestoreOptions(options...)

	ct := opts.component
	h := opts.host

	// otherwise restore all instances in the archive
	if process, ok := sizes[ct.String()+":"+i]; ok {
		if process > 0 {
			if !opts.list {
				if err = writeFile(i, fp, tr, hdr, opts); err != nil {
					// if written ok, add one more file
					return
				}
			}
			sizes[ct.String()+":"+i] += hdr.Size
		}
		return
	}

	// init sizes entry
	if opts.list {
		sizes[ct.String()+":"+i] = hdr.Size
		return
	}

	// otherwise init to -1
	sizes[ct.String()+":"+i] = -1

	// write file and update sizes
	if _, err = h.Stat(h.PathTo(ct, ct.String()+"s", i)); err != nil {
		// instance does not yet exist
		if err = writeFile(i, fp, tr, hdr, opts); err != nil {
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
func writeFile(i string, fp string, tr *tar.Reader, hdr *tar.Header, opts *restoreOptions) (err error) {
	ct := opts.component
	h := opts.host

	if ct == nil {
		return geneos.ErrInvalidArgs
	}

	instanceDir := h.PathTo(ct, ct.String()+"s", i)
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

		// if the file is the instance config, then call rebuild to
		// update paths and ports as required, and to remove legacy
		// parameters, instead of just writing the file out
		if fp == ct.String()+".json" {
			return rebuildConfig(i, instanceDir, hdr.Name, tr, opts)
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
func writeSharedFile(fp string, tr *tar.Reader, hdr *tar.Header, opts *restoreOptions) (err error) {
	h := opts.host

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

func rebuildConfig(i, instanceDir string, fp string, r io.Reader, opts *restoreOptions) (err error) {
	ct := opts.component
	h := opts.host

	// restore config, update parameters for new root dir on dest host, write
	cf, err := config.Read(ct.String(), config.Reader(r))
	if err != nil {
		return err
	}

	// update name in case this is a rename
	config.Set(cf, "name", i)

	nct := config.Get[string](cf, "pkgtype", config.DefaultValue(ct.String()))

	oldHome := config.Get[string](cf, "home")
	newHome := instanceDir

	newGeneosDir := config.Get[string](h.Config, "geneos")
	oldGeneosDir := strings.TrimSuffix(oldHome, "/"+filepath.Dir(fp))

	// now set new home for Replace below
	config.Set(cf, "home", newHome)
	log.Debug().Msgf("setting home to %q for config rebuild. check %q", newHome, config.Get[string](cf, "home"))

	oldInstall := config.Get[string](cf, "install")
	newInstall := h.PathTo("packages", nct)

	oldShared := path.Join(path.Dir(path.Dir(oldHome)), ct.String()+"_shared")
	newShared := ct.Shared(h)

	version := config.Get[string](cf, "version", config.NoExpand())

	for _, k := range cf.AllKeys() {
		v := config.Get[any](cf, k, config.NoExpand())
		switch k {
		case "libpaths":
			// treat libpaths special, below
			continue
		case "port":
			ports := instance.GetAllPorts(h)
			if ports[config.Get[uint16](cf, k)] {
				// port already in use, get the next one
				config.Set(cf, k, instance.NextFreePort(h, ct))
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

				// finally replace any remaining top-level paths
				if oldGeneosDir != newGeneosDir {
					vs = strings.Replace(vs, oldGeneosDir, newGeneosDir, 1)
				}

				log.Debug().Msgf("setting config key %s to %q", k, vs)
				config.Set(cf, k, vs, config.Replace("home"))
				log.Debug().Msgf("after setting, config key %s is %q", k, config.Get[string](cf, k))
			}
		}
	}

	libpaths := []string{}
	np := "${config:install}"
	nv := "${config:version}"

	for _, p := range filepath.SplitList(config.Get[string](cf, "libpaths", config.NoExpand())) {
		if after, ok := strings.CutPrefix(p, oldInstall+"/"); ok {
			subpath := after
			if after, ok := strings.CutPrefix(subpath, version); ok {
				libpaths = append(libpaths, path.Join(np, nv, after))
			} else {
				libpaths = append(libpaths, path.Join(np, subpath))
			}
		} else {
			libpaths = append(libpaths, p)
		}
	}
	config.Set(cf, "libpaths", strings.Join(libpaths, ":"))

	config.Delete(cf, "user")

	if err = cf.Write(ct.String(),
		config.Host(h),
		config.SearchDirs(instanceDir),
		config.AppName(i),
		config.IgnoreEmptyValues(),
	); err != nil {
		return err
	}

	return
}
