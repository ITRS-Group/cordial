package restore

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/dsnet/compress/bzip2"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var log = logger.Logger

var fileTypes = map[string]string{
	".tar.gz":  "gzip",
	".tgz":     "gzip",
	".tar.bz2": "bzip2",
	".tar":     "none",
}

// compressionAliases maps alternative compression names to those used
// by Restore
var compressionAliases = map[string]string{
	"gz":  "gzip",
	"bz2": "bzip2",
}

// restoreStat records what is restored, or listed, for an instance or
// for the shared directories
type restoreStat struct {
	skipped bool  // true if nothing is restored, e.g. because the instance already exists
	size    int64 // total size of the entries restored or listed
}

// Restore reads the file at the local archive path and restores
// matching instances matching component ct (all if nil) with names.
// names can be in the form "dest=src" to allow renaming instances. If
// ct is nil then all matching types are restored and if names it empty
// then all instances are restored. Existing instances with matching
// names are not overwritten.
//
// The compression of the archive is taken from the file extension or,
// for STDIN (an archive of "-"), the Compression option, which must be
// one of "gzip", "bzip2" or "none". It is an error if the compression
// cannot be determined.
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
		if fileType == "" {
			return fmt.Errorf("%w: compression type is required when restoring from STDIN", geneos.ErrInvalidArgs)
		}
	} else {
		for s, t := range fileTypes {
			if strings.HasSuffix(archive, s) {
				fileType = t
				break
			}
		}
		if fileType == "" {
			return fmt.Errorf("%w: cannot determine the compression of %q (expected one of %s)", geneos.ErrInvalidArgs,
				archive, strings.Join(slices.Sorted(maps.Keys(fileTypes)), ", "))
		}
	}
	if alias, ok := compressionAliases[fileType]; ok {
		fileType = alias
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
	case "none":
		tin = in
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
		err = fmt.Errorf("%w: unknown decompression type %q", geneos.ErrInvalidArgs, fileType)
		return
	}

	// store paths processed mapped to types and instances, with record
	// of skipping, to prevent overwrites, to anticipate instances that
	// are spread out

	// stats - key is type:name. An entry is added when type:name is
	// first found in archive and is marked as skipped if the
	// destination already exists.
	stats := map[string]*restoreStat{}

	// files counts the number of files restored for each type:name
	files := map[string]int64{}

	// mapping instance names from archive names to potentially new
	// ones.
	//
	// wildcards are only allowed when no renaming is taking place
	mapping := map[string]string{}

	// matched records the names, or patterns, that matched an instance
	// in the archive
	matched := map[string]bool{}

	restoreAll := false
	if len(names) == 1 && names[0] == "all" {
		restoreAll = true
	} else {
		for _, name := range names {
			dest, src, found := strings.Cut(name, "=")
			if found {
				if !instance.ValidName(src) || !instance.ValidName(dest) {
					log.Debug("invalid instance name format when using DEST=SRC format", slog.String("name", name))
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
	// remember instance directories created by this restore so that a
	// failure part way through the archive does not leave partial
	// instances behind
	created := map[string]bool{}
	extracted := false
	defer func() {
		if extracted || opts.list {
			return
		}
		for dir := range created {
			log.Debug("removing partially restored instance", slog.String("directory", dir))
			opts.host.RemoveAll(dir)
		}
	}()

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
		if ctName == "tls" {
			if opts.shared {
				if err = restoreShared(stats, files, "TLS:!SHARED", hdr, opts, func() error {
					return writeSharedTLSFile(path.Join(instance.TLSBASE, rest), tr, hdr, opts)
				}); err != nil {
					return
				}
			}
			continue
		}

		// check ctName is valid and if ct is not nil, filter for a match
		nct := geneos.ParseComponent(ctName)
		if nct == nil {
			log.Debug("unknown component type, skipping", slog.String("filename", filename))
			continue
		}

		// if a component type is wanted, reject others
		if ct != nil && ct != nct {
			log.Debug("component type does not match, skipping", slog.String("expected", ct.String()), slog.String("found", nct.String()))
			continue
		}

		if rest != "" {
			ctSubdir, rest, _ = strings.Cut(rest, "/")
		}

		// check for shared directories for given nct and restore if asked to
		if opts.shared {
			if slices.Contains(nct.SharedDirectories, path.Join(nct.String(), ctSubdir)) {
				if ct == nil || ct == nct {
					if err = restoreShared(stats, files, nct.String()+":!SHARED", hdr, opts, func() error {
						return writeSharedFile(path.Join(nct.String(), ctSubdir, rest), tr, hdr, opts)
					}); err != nil {
						return
					}
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
			continue
		}

		// "rest" is (should be) now "instance/content"
		instanceName, instRelfilePath, _ := strings.Cut(rest, "/")

		// we now have type (nct), instance name and filepath fp

		// does the instance name match any of the names list, including wildcards?
		if len(mapping) > 0 {
			for dest, src := range mapping {
				if dest == src {
					if ok, _ := filepath.Match(dest, instanceName); ok {
						matched[dest] = true
						// save
						if err = processFile(instanceName, instRelfilePath, tr, hdr, stats, created, Component(packageCt), Host(opts.host), List(opts.list)); err != nil {
							return
						}
						files[packageCt.String()+":"+instanceName] += 1
					}
				} else {
					if src == instanceName {
						matched[dest] = true
						// rename and save
						if err = processFile(dest, instRelfilePath, tr, hdr, stats, created, Component(packageCt), Host(opts.host), List(opts.list)); err != nil {
							return
						}
						files[packageCt.String()+":"+dest] += 1
					}
				}
			}
			continue
		}

		// otherwise restore all instances in the archive
		if restoreAll {
			if err = processFile(instanceName, instRelfilePath, tr, hdr, stats, created, Component(packageCt), Host(opts.host), List(opts.list)); err != nil {
				return
			}
			files[packageCt.String()+":"+instanceName] += 1
		}
	}

	extracted = true

	if !opts.list && opts.beforeRebuild != nil {
		if err := opts.beforeRebuild(); err != nil {
			return fmt.Errorf("preparing restored instances for rebuild: %w", err)
		}
	}

	// the names that were asked for but are not in the archive are an
	// error, but the instances that are in the archive are still
	// restored
	var errs []error
	for _, dest := range slices.Sorted(maps.Keys(mapping)) {
		if !matched[dest] {
			errs = append(errs, fmt.Errorf("%s: not found in archive %s", mapping[dest], archive))
		}
	}

	if opts.list {
		fmt.Fprintf(opts.progress, "Contents of archive %s:\n", archive)
	} else {
		fmt.Fprintf(opts.progress, "Restoring from archive %s:\n", archive)
	}

	for _, k := range slices.Sorted(maps.Keys(stats)) {
		ctName, name, _ := strings.Cut(k, ":")
		stat := stats[k]

		if name == "!SHARED" {
			name = "(shared dirs)"
		} else if !opts.list && !stat.skipped {
			// rebuild the configuration of each instance restored
			i, gerr := instance.GetWithHost(opts.host, geneos.ParseComponent(ctName), name)
			if gerr != nil {
				log.Debug("getting instance for final rebuild", slog.Any("error", gerr), slog.String("ctName", ctName), slog.String("name", name))
				errs = append(errs, fmt.Errorf("%s %s: %w", ctName, name, gerr))
			} else if _, rerr := i.Rebuild(false); rerr != nil {
				if !errors.Is(rerr, geneos.ErrNotSupported) {
					i.Log().Debug("rebuild of instance failed", slog.Any("error", rerr), slog.String("ctName", ctName), slog.String("name", name))
					errs = append(errs, fmt.Errorf("%s %s: rebuild failed: %w", ctName, name, rerr))
				}
			} else {
				fmt.Fprintf(opts.progress, "%s %s config rebuild complete\n", ctName, name)
			}
		}

		if opts.list {
			fmt.Fprintf(opts.progress, "%s %s %d files, total %.3f MiB\n", ctName, name, files[k], float64(stat.size)/(1024*1024))
			continue
		}

		if !stat.skipped {
			if name != mapping[name] && !restoreAll {
				fmt.Fprintf(opts.progress, "%s %s (from %s) %d files, total %.3f MiB\n", ctName, name, mapping[name], files[k], float64(stat.size)/(1024*1024))
			} else {
				fmt.Fprintf(opts.progress, "%s %s %d files, total %.3f MiB\n", ctName, name, files[k], float64(stat.size)/(1024*1024))
			}

		} else {
			fmt.Fprintf(opts.progress, "%s %s already exists, skipping\n", ctName, name)
		}
	}
	return errors.Join(errs...)
}

// restoreShared writes the entry in a shared directory, by calling
// write, or only counts it if listing. The result is added to the
// totals for key. Entries that already exist are skipped.
func restoreShared(stats map[string]*restoreStat, files map[string]int64, key string, hdr *tar.Header, opts *restoreOptions, write func() error) error {
	stat, ok := stats[key]
	if !ok {
		stat = &restoreStat{skipped: true}
		stats[key] = stat
		files[key] = 0
	}

	if !opts.list {
		if err := write(); err != nil {
			if errors.Is(err, os.ErrExist) {
				return nil
			}
			return err
		}
	}

	stat.skipped = false
	stat.size += hdr.Size
	files[key]++
	return nil
}

func processFile(instanceName, instRelFilePath string, tr *tar.Reader, hdr *tar.Header, stats map[string]*restoreStat, created map[string]bool, options ...RestoreOption) (err error) {
	opts := evalRestoreOptions(options...)

	ct := opts.component
	h := opts.host
	key := ct.String() + ":" + instanceName

	// the instance has already been seen in the archive
	if stat, ok := stats[key]; ok {
		if !stat.skipped {
			if !opts.list {
				if err = writeFile(instanceName, instRelFilePath, tr, hdr, opts); err != nil {
					return
				}
			}
			stat.size += hdr.Size
		}
		return
	}

	// first entry for the instance
	if opts.list {
		stats[key] = &restoreStat{size: hdr.Size}
		return
	}

	// otherwise skipped unless the instance does not exist and is
	// written
	stat := &restoreStat{skipped: true}
	stats[key] = stat

	// write file and update stats
	instDir := h.PathTo(ct, ct.String()+"s", instanceName)
	if _, err = h.Stat(instDir); err != nil {
		// instance does not yet exist
		created[instDir] = true
		if err = writeFile(instanceName, instRelFilePath, tr, hdr, opts); err != nil {
			return
		}
		// first file written
		stat.skipped = false
		stat.size = hdr.Size
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
	if err := checkRestorePath(h, destPath); err != nil {
		return err
	}
	if fi, err := h.Lstat(destPath); err == nil && fi.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("cannot overwrite restored symlink %q", destPath)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	switch hdr.Typeflag {
	case tar.TypeDir:
		// create and set permissions
		return h.MkdirAll(destPath, hdr.FileInfo().Mode().Perm())
	case tar.TypeReg:
		var w io.WriteCloser

		// create intermediate dirs, write, set perms
		if err = h.MkdirAll(h.Dir(destPath), 0775); err != nil {
			return
		}

		// if the file is the instance config, then call rebuild to
		// update paths and ports as required, and to remove legacy
		// parameters, instead of just writing the file out
		if instRelFilePath == ct.String()+".json" {
			return rebuildConfig(instanceName, hdr.Name, tr, opts)
		}

		if w, err = h.Create(destPath, hdr.FileInfo().Mode()); err != nil {
			log.Debug("creating file failed", slog.Any("error", err), slog.String("destPath", destPath))
			return
		}
		defer func() {
			if cerr := w.Close(); err == nil {
				err = cerr
			}
		}()

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
	return writeSharedEntry(fp, tr, hdr, opts, false)
}

// writeSharedTLSFile reads the file from the tar reader and, if a
// regular file, checks for the global CA Bundle, in which case it
// merges the contents, otherwise it writes it to the path fp relative
// to the host's root geneos, creating intermediate directories as
// required, and setting permissions as per tar header. Owner and group
// are ignored. Directory entries are used to create directories with
// matching permissions, again ignoring owner and group.
func writeSharedTLSFile(fp string, tr *tar.Reader, hdr *tar.Header, opts *restoreOptions) (err error) {
	return writeSharedEntry(fp, tr, hdr, opts, true)
}

func writeSharedEntry(fp string, tr *tar.Reader, hdr *tar.Header, opts *restoreOptions, tls bool) (err error) {
	h := opts.host
	destPath := h.PathTo(fp)
	if err := checkRestorePath(h, destPath); err != nil {
		return err
	}

	switch hdr.Typeflag {
	case tar.TypeSymlink:
		return restoreSymlink(h, destPath, hdr.Linkname)
	case tar.TypeDir:
		// create and set permissions
		if _, err := h.Lstat(destPath); err == nil {
			return os.ErrExist
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return h.MkdirAll(destPath, hdr.FileInfo().Mode().Perm())
	case tar.TypeReg:
		var w io.WriteCloser

		if fi, err := h.Lstat(destPath); err == nil {
			if fi.Mode()&fs.ModeSymlink != 0 {
				return os.ErrExist
			}
			if tls && path.Base(fp) == geneos.CABundleBasename+certs.PEMExtension {
				truststore := h.PathTo(path.Dir(fp), geneos.CABundleBasename+certs.KeystoreExtension)
				if fi, err := h.Lstat(truststore); err == nil && fi.Mode()&fs.ModeSymlink != 0 {
					return fmt.Errorf("cannot merge CA bundle through symlink %q", truststore)
				} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
					return err
				}
				roots, err := certs.ReadCertificatesFrom(tr)
				if err != nil {
					return err
				}
				updated, err := certs.UpdateCACertsFiles(h, h.PathTo(path.Dir(fp), geneos.CABundleBasename), roots...)
				if err != nil {
					return err
				}
				if updated {
					log.Debug("CA bundle updated")
				}
				return nil
			}
			// UpdateCACertsFiles writes both PEM and Java truststores.
			return os.ErrExist
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}

		// create intermediate dirs, write, set perms
		if err = h.MkdirAll(h.Dir(destPath), 0775); err != nil {
			return
		}
		if w, err = h.Create(destPath, hdr.FileInfo().Mode()); err != nil {
			return
		}
		defer func() {
			if cerr := w.Close(); err == nil {
				err = cerr
			}
		}()

		_, err = io.Copy(w, tr)
		return
	default:
		// ignore
	}
	return
}

func checkRestorePath(h *geneos.Host, destPath string) error {
	relative, err := h.Rel(h.PathTo(), destPath)
	if err != nil {
		return err
	}
	relative, err = geneos.CleanRelativePath(h.ToSlash(relative))
	if err != nil {
		return err
	}
	parent := h.PathTo()
	parts := strings.Split(relative, "/")
	for _, part := range parts[:len(parts)-1] {
		parent = h.Join(parent, part)
		fi, err := h.Lstat(parent)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if fi.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("restore path %q traverses symlink %q", destPath, parent)
		}
	}
	return nil
}

func restoreSymlink(h *geneos.Host, destination, target string) error {
	if _, err := h.Lstat(destination); err == nil {
		return fs.ErrExist
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	name, err := h.Rel(h.PathTo(), destination)
	if err != nil {
		return err
	}
	target, err = geneos.ArchiveSymlinkTarget(h, h.PathTo(), name, target)
	if err != nil {
		return err
	}
	if err := h.MkdirAll(h.Dir(destination), 0o775); err != nil {
		return err
	}
	if err := h.Symlink(target, destination); err != nil {
		return fmt.Errorf("restore symlink %q -> %q: %w", destination, target, err)
	}
	return nil
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

	instanceDir := h.PathTo(ct, ct.String()+"s", instanceName)
	if err = instance.RefactorConfig(h, ct, cf, instance.NewName(instanceName), instance.NewDir(instanceDir)); err != nil {
		return err
	}

	// fix earlier mistake with default settings and quoting `none`
	if listenip, ok := config.Lookup[string](cf, "listenip"); ok && listenip == `"none"` {
		config.Set(cf, "listenip", "none")
	}

	return cf.Write(ct.String(),
		config.Host(h),
		config.SearchDirs(instanceDir),
		config.AppName(instanceName),
		config.OmitEmptyValues(),
	)
}
