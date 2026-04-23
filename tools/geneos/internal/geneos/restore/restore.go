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
// matching instances matching component ct (all if nil) with names.
// names can be in the form "dest=src" to allow renaming instances. If
// ct is nil then all matching types are restored and if names it empty
// then all instances are restored. Existing instances with matching
// names are not overwritten.
//
// TODO: shared file control (use flag selector and limit to ct, if not
// nil)
func Restore(archive string, options ...RestoreOption) (err error) {
	// instancesRestored restored, for final Rebuilds
	instancesRestored := map[string]geneos.Instance{}

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
				// no mapping, just restore name as is, but save in case of wildcards
				mapping[name] = name
			}
		}
	}

	// we cannot guarantee the order of files in the archive, so we need
	// to read through the whole archive and process files as we go,
	// keeping track of what we've done to prevent overwriting files
	// from the same archive, and to allow for wildcards and renaming.
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
		instanceName, instRelfilePath, _ := strings.Cut(rest, "/")

		// we now have type (nct), instance name and filepath fp

		// does the instance name match any of the names list, including wildcards?
		if len(mapping) > 0 {
			for dest, src := range mapping {
				if dest == src {
					if matched, _ := filepath.Match(dest, instanceName); matched {
						// save
						if err = processFile(instanceName, instRelfilePath, tr, hdr, sizes, Component(packageCt), Host(opts.host), List(opts.list)); err != nil {
							return
						}
						files[packageCt.String()+":"+instanceName] += 1
						if _, ok := instancesRestored[instanceName]; !ok {
							instancesRestored[instanceName] = nil
						}
					}
				} else {
					if src == instanceName {
						// rename and save
						if err = processFile(dest, instRelfilePath, tr, hdr, sizes, Component(packageCt), Host(opts.host), List(opts.list)); err != nil {
							return
						}
						files[packageCt.String()+":"+dest] += 1
						if _, ok := instancesRestored[dest]; !ok {
							instancesRestored[dest] = nil
						}
					}
				}
			}
			continue
		}

		// otherwise restore all instances in the archive
		if restoreAll {
			if err = processFile(instanceName, instRelfilePath, tr, hdr, sizes, Component(packageCt), Host(opts.host), List(opts.list)); err != nil {
				return
			}
			files[packageCt.String()+":"+instanceName] += 1
			if _, ok := instancesRestored[instanceName]; !ok {
				instancesRestored[instanceName] = nil
			}
		}
	}

	if opts.list {
		fmt.Fprintf(opts.progress, "Contents of archive %s:\n", archive)
	} else {
		fmt.Fprintf(opts.progress, "Restoring from archive %s:\n", archive)
	}

	for _, k := range slices.Sorted(maps.Keys(sizes)) {
		ctName, name, _ := strings.Cut(k, ":")

		if name == "!SHARED" {
			name = "(shared dirs)"
		} else {
			if i, ok := instancesRestored[name]; ok && i == nil {
				i, err = instance.GetWithHost(opts.host, geneos.ParseComponent(ctName), name)
				if err != nil {
					log.Debug().Err(err).Msgf("getting instance %s:%s for final rebuild", ctName, name)
				} else {
					instancesRestored[name] = i
					if err = i.Rebuild(true); err != nil {
						if !errors.Is(err, geneos.ErrNotSupported) {
							log.Debug().Err(err).Msgf("rebuild of instance %s:%s", ctName, name)
						} else {
							err = nil
						}
					} else {
						fmt.Fprintf(opts.progress, "%s %s config rebuild complete\n", ctName, name)
					}
				}
			}
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
	return
}

func processFile(instanceName, instRelFilePath string, tr *tar.Reader, hdr *tar.Header, sizes map[string]int64, options ...RestoreOption) (err error) {
	opts := evalRestoreOptions(options...)

	ct := opts.component
	h := opts.host

	// otherwise restore all instances in the archive
	if process, ok := sizes[ct.String()+":"+instanceName]; ok {
		if process > 0 {
			if !opts.list {
				if err = writeFile(instanceName, instRelFilePath, tr, hdr, opts); err != nil {
					// if written ok, add one more file
					return
				}
			}
			sizes[ct.String()+":"+instanceName] += hdr.Size
		}
		return
	}

	// init sizes entry
	if opts.list {
		sizes[ct.String()+":"+instanceName] = hdr.Size
		return
	}

	// otherwise init to -1
	sizes[ct.String()+":"+instanceName] = -1

	// write file and update sizes
	if _, err = h.Stat(h.PathTo(ct, ct.String()+"s", instanceName)); err != nil {
		// instance does not yet exist
		if err = writeFile(instanceName, instRelFilePath, tr, hdr, opts); err != nil {
			return
		}
		// first file written
		sizes[ct.String()+":"+instanceName] = hdr.Size
	}
	return
}

// writeFile reads the file from the tar reader and, if a regular file,
// writes it to the instance directory, creating intermediate
// directories as required, and setting permissions as per tar header.
// Owner and group are ignored. Directory entries are used to create
// directories with matching permissions, again ignoring owner and
// group.
func writeFile(instanceName string, instRelFilePath string, tr *tar.Reader, hdr *tar.Header, opts *restoreOptions) (err error) {
	ct := opts.component
	h := opts.host

	if ct == nil {
		return geneos.ErrInvalidArgs
	}

	destPath := h.PathTo(ct, ct.String()+"s", instanceName, instRelFilePath)

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
		if instRelFilePath == ct.String()+".json" {
			return rebuildConfig(instanceName, hdr.Name, tr, opts)
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

// rebuildConfig reads the component config file from the tar reader,
// updates paths and ports as required, removes legacy parameters, and
// writes the updated config to the instance directory. Paths are
// updated based on the old home value in the config file and the new
// home value based on the destination instance directory. Ports are
// updated to avoid clashes with existing instances on the destination
// host.
func rebuildConfig(instanceName, homeRelFilePath string, r io.Reader, opts *restoreOptions) (err error) {
	ct := opts.component
	h := opts.host

	// restore config, update parameters for new root dir on dest host, write
	cf, err := config.Read(ct.String(), config.Reader(r))
	if err != nil {
		return err
	}

	if err = instance.RefactorConfig(h, ct, cf, instance.NewName(instanceName)); err != nil {
		return err
	}

	// fix earlier mistake with default settings and quoting `none`
	if listenip, ok := config.Lookup[string](cf, "listenip"); ok && listenip == `"none"` {
		config.Set(cf, "listenip", "none")
	}

	if err = cf.Write(ct.String(),
		config.Host(h),
		config.SearchDirs(h.PathTo(ct, ct.String()+"s", instanceName)),
		config.AppName(instanceName),
		config.OmitEmptyValues(),
	); err != nil {
		return err
	}

	return
}
