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

package geneos

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	zlog "github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
)

// ImportSource copies the contents from item to the destination
// directory dir on host h. The destination filename can be given as a
// "NAME=" prefix in item. If no filename is given then it is derived
// from the base filename of item.
//
// item, without any destination prefix, can be a path to a file, a
// directory or a http/https URL.
//
// If item points to a local file and it is the same as the destination
// then an ErrExists is returned.
//
// If item points to a local directory then it is copied to the
// destination only if the destination is a directory or does not exist
// and a directory with that name can be created.
//
// If item ends with a '/' then the contents are copied without the
// directory prefix, otherwise the final directory component in source
// is also copied, e.g. 'conf/' means import the contents of the
// directory 'conf', which 'conf' means a new directory 'conf' is
// created in dest and the contents copied.
//
// TODO: add templating option - perhaps ImportFileTemplated() - to run
// through text/template with given data (e.g. instance config)
func ImportSource(h *Host, dir, item string) (filename string, err error) {
	// host must be valid and not `ALL`
	if h == nil || h == ALL {
		err = ErrInvalidArgs
		zlog.Debug().Err(err).Msgf("host invalid, skipping")
		return
	}

	// check item is not empty
	if item == "" {
		err = ErrInvalidArgs
		zlog.Debug().Err(err).Msgf("no source defined, skipping")
		return
	}

	// check dir is not empty and actually a directory
	if dir == "" {
		err = ErrInvalidArgs
		zlog.Debug().Err(err).Msgf("dir is empty, skipping")
		return
	}

	if st, err := h.Stat(dir); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			zlog.Debug().Err(err).Msgf("dir check failed")
			return "", err
		}

		// try and create it
		if err = h.MkdirAll(dir, 0775); err != nil {
			zlog.Debug().Err(err).Msgf("failed to create dir %q, skipping", dir)
			return "", err
		}
	} else {
		if !st.IsDir() {
			err = ErrInvalidArgs
			zlog.Debug().Err(err).Msgf("dir %q is not a directory, skipping", dir)
			return "", err
		}
	}

	// split out any "NAME=" prefix
	var dstIsDir bool
	dst, src, ok := strings.Cut(item, "=")
	if ok {
		// do some basic validation on user-supplied destination
		if dst == "" {
			err = ErrInvalidArgs
			zlog.Debug().Err(err).Msg("dest path empty")
			return
		}
		// if the dest contains "://" then it is probably part of
		// the URL and not a prefix, so ignore it - restore values
		if strings.Contains(dst, "://") {
			dst = ""
		} else if strings.HasSuffix(dst, "/") {
			// create a directory
			dstIsDir = true
			if dst, err = CleanRelativePath(dst); err != nil {
				zlog.Fatal().Msg("dest path must be relative")
			}
		} else if dst, err = CleanRelativePath(dst); err != nil {
			zlog.Fatal().Msg("dest path must be relative")
		}
		item = src
	} else {
		dst = ""
	}

	// if the destination exists, is it a directory?
	if s, err := h.Stat(path.Join(dir, dst)); err == nil {
		if s.IsDir() {
			dir = path.Join(dir, dst)
			dst = ""
			dstIsDir = false
		}
	}

	if dstIsDir {
		dir = path.Join(dir, dst)
		dst = ""
	}

	zlog.Debug().Msgf("importing source %q to dir %q with dest %q", item, dir, dst)

	// set the Source(item) option so that any optional credentials are loaded
	from, filename, _, err := openSource(item, Source(item))
	if err != nil {
		// special case the source is a directory
		if errors.Is(err, ErrIsADirectory) {
			err = nil

			// import directory
			err = importDirectory(h, dir, item)
			if err != nil {
				zlog.Fatal().Err(err).Msgf("failed to import directory %q", item)
			}

			return
		}
		zlog.Fatal().Err(err).Msgf("cannot open source %q", item)
	}
	defer from.Close()

	zlog.Debug().Msgf("opened source %q for filename %q", item, filename)

	// only use the returned filename if no explicit destination is given
	if dst == "" {
		dst = filename
	}

	// return final basename
	dst = path.Join(dir, dst)
	filename = path.Base(dst)

	// test for same source and dest, return err
	if h.IsLocalhost() {
		item = config.ResolveHome(item)
		sfi, err := h.Stat(item)
		if err != nil {
			return "", err
		}
		dfi, err := h.Stat(dst)
		if err == nil {
			if os.SameFile(sfi, dfi) {
				// same
				fmt.Printf("import skipped, source and destination are the same file: %q\n", item)
				return filename, ErrExists
			}
		}
	}

	// check to containing directory, as destfile above may be a
	// relative path under destdir and not just a filename
	if _, err := h.Stat(path.Dir(dst)); err != nil {
		err = h.MkdirAll(path.Dir(dst), 0775)
		if err != nil && !errors.Is(err, fs.ErrExist) {
			zlog.Fatal().Err(err).Msg("")
		}
	}

	// create a backup if the file exists, save the destination path
	// in case we need to restore it
	var backuppath string
	if s, err := h.Stat(dst); err == nil {
		if !s.Mode().IsRegular() {
			zlog.Fatal().Msg("dest exists and is not a plain file")
		}
		backuppath = dst + "." + time.Now().UTC().Format("20060102150405") + ".old"
		// datetime := time.Now().UTC().Format("20060102150405")
		if err = h.Rename(dst, backuppath); err != nil {
			return filename, err
		}
	}

	cf, err := h.Create(dst, 0664)
	if err != nil {
		if backuppath != "" {
			// try and restore backup
			_ = h.Rename(backuppath, dst)
		}
		return
	}
	defer cf.Close()

	if _, err = io.Copy(cf, from); err != nil {
		if backuppath != "" {
			// try and restore backup
			_ = h.Rename(backuppath, dst)
		}
		return
	}
	fmt.Printf("imported %q to %s:%s\n", item, h.String(), dst)

	return
}

// import directory from local source directory to dir on host h
func importDirectory(h *Host, dir, source string) (err error) {
	var backuppath string

	// check dest is a directory
	if dir == "" {
		err = ErrInvalidArgs
		zlog.Debug().Err(err).Msgf("source %q is a directory, dir is empty , skipping", source)
		return
	}

	// if source ends with '/' then copy contents only, not the directory named
	if !strings.HasSuffix(source, "/") {
		dir = filepath.Join(dir, filepath.Base(source))
	}

	fs.WalkDir(os.DirFS(source), ".", func(file string, di fs.DirEntry, _ error) error {
		destfile := path.Join(dir, file)

		st, err := di.Info()
		if err != nil {
			return err
		}

		if di.IsDir() {
			return h.MkdirAll(destfile, st.Mode().Perm())
		}

		from, err := os.Open(path.Join(source, file))
		if err != nil {
			return err
		}

		if s, err := h.Stat(destfile); err == nil {
			if !s.Mode().IsRegular() {
				zlog.Fatal().Msg("dest exists and is not a plain file")
			}
			datetime := time.Now().UTC().Format("20060102150405")
			backuppath = destfile + "." + datetime + ".old"
			if err = h.Rename(destfile, backuppath); err != nil {
				return err
			}
		}

		to, err := h.Create(destfile, st.Mode().Perm())
		if err != nil {
			return err
		}
		defer to.Close()

		if _, err = io.Copy(to, from); err != nil {
			return err
		}
		fmt.Printf("imported %q to %s:%s\n", path.Join(source, file), h.String(), destfile)
		return nil
	})
	return
}

// ImportCommons copies a file to an instance common directory.
func ImportCommons(h *Host, ct *Component, common string, params []string) (filenames []string, err error) {
	if ct == nil || ct == &RootComponent {
		err = ErrNotSupported
		return
	}

	if len(params) == 0 {
		zlog.Fatal().Msg("no file/url provided")
	}

	dir := h.PathTo(ct, common)
	for _, source := range params {
		var filename string
		if filename, err = ImportSource(h, dir, source); err != nil && err != ErrExists {
			return
		}
		filenames = append(filenames, filename)
	}
	err = nil // reset in case above returns ErrExists
	return
}
