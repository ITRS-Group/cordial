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

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/rs/zerolog/log"
)

// ImportSource copies the contents from source to the destination dest
// on host h. The destination filename can be given as a "NAME=" prefix
// in source. If no filename is given then it is derived from the
// source.
//
// source can be a path to a file or a http/https URL.
//
// If source is a local file and it is the same as the destination then
// an ErrExists is returned.
//
// If source is a local directory then it is copied to dest only if dest
// is a directory or does not exist and a directory with that name can
// be created. If source ends with a '/' then the contents are copied
// without the source directory prefix, otherwise the final directory
// component in source is also copied, e.g. 'conf/' means import the
// contents of the directory 'conf', which 'conf' means a new directory
// 'conf' is created in dest and the contents copied.
//
// TODO: add templating option - perhaps ImportFileTemplated() - to run
// through text/template with given data (probabl instance config)
func ImportSource(h *Host, dest string, source string) (filename string, err error) {
	var backuppath string
	var from io.ReadCloser
	var noDestPrefix bool // plain file as source?

	if h == ALL {
		err = ErrInvalidArgs
		return
	}

	// destdir becomes the absolute path for the imported file
	destdir := dest

	// destfile is the basename of the import path, empty if the source
	// filename should be kept
	destfile := ""

	// if the source contains the start of a URL then only split if the
	// '=' is directly before - "=" is valid later in the URL
	if (strings.Contains(source, "https://") && !strings.HasPrefix(source, "https://")) ||
		(strings.Contains(source, "http://") && !strings.HasPrefix(source, "http://")) {
		s := strings.SplitN(source, "=https://", 2)
		if len(s) != 2 { // not found
			s = strings.SplitN(source, "=http://", 2)
			if len(s) != 2 {
				// ERROR
			}
			if s[1] == "" {
				log.Fatal().Msg("no source defined")
			}
			source = "http://" + s[1]
		} else {
			if s[1] == "" {
				log.Fatal().Msg("no source defined")
			}
			source = "https://" + s[1]
		}

		if s[0] == "" {
			// should this be treated the same as "." ?
			log.Fatal().Msg("dest path empty")
		}

		if destfile, err = CleanRelativePath(s[0]); err != nil {
			log.Fatal().Msg("dest path must be relative to (and in) instance directory")
		}

		// if the destination exists is it a directory?
		if s, err := h.Stat(path.Join(dest, destfile)); err == nil {
			if s.IsDir() {
				destdir = path.Join(dest, destfile)
				destfile = ""
			}
		}
	} else {
		noDestPrefix = true
		s := strings.SplitN(source, "=", 2)
		if len(s) > 1 {
			// do some basic validation on user-supplied destination
			if s[0] == "" {
				log.Fatal().Msg("dest path empty")
			}
			if destfile, err = CleanRelativePath(s[0]); err != nil {
				log.Fatal().Msg("dest path must be relative to (and in) instance directory")
			}
			// if the destination exists is it a directory?
			if s, err := h.Stat(path.Join(dest, destfile)); err == nil {
				if s.IsDir() {
					destdir = path.Join(dest, destfile)
					destfile = ""
				}
			}
			source = s[1]
			if source == "" {
				log.Fatal().Msg("no source defined")
			}
		}
	}

	from, filename, _, err = openSourceFile(source)
	if err != nil {
		if errors.Is(err, ErrIsADirectory) {
			err = nil
			// import directory

			// check dest is a directory
			if destfile != "" && destdir == "" {
				err = ErrInvalidArgs
				log.Debug().Err(err).Msgf("source %q is a directory, destdir is empty and destfile is %q, skipping", source, destfile)
				return
			}

			if !strings.HasSuffix(source, "/") {
				destdir = filepath.Join(destdir, filepath.Base(source))
			}

			fs.WalkDir(os.DirFS(source), ".", func(file string, di fs.DirEntry, _ error) error {
				destfile = path.Join(destdir, file)

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
						log.Fatal().Msg("dest exists and is not a plain file")
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
		log.Fatal().Err(err).Msg("")
	}
	defer from.Close()

	// only use the returned filename if no explicit destination is given
	if destfile == "" {
		destfile = filename
	}

	// return final basename
	destfile = path.Join(destdir, destfile)
	filename = path.Base(destfile)

	// test for same source and dest, return err
	if noDestPrefix && h.IsLocal() {
		source = config.ExpandHome(source)
		sfi, err := h.Stat(source)
		if err != nil {
			return "", err
		}
		dfi, err := h.Stat(destfile)
		if err == nil {
			if os.SameFile(sfi, dfi) {
				// same
				fmt.Printf("import skipped, source and destination are the same file: %q\n", source)
				return filename, ErrExists
			}
		}
	}

	// check to containing directory, as destfile above may be a
	// relative path under destdir and not just a filename
	if _, err := h.Stat(path.Dir(destfile)); err != nil {
		err = h.MkdirAll(path.Dir(destfile), 0775)
		if err != nil && !errors.Is(err, fs.ErrExist) {
			log.Fatal().Err(err).Msg("")
		}
	}

	// xxx - wrong way around. create tmp first, move over later
	if s, err := h.Stat(destfile); err == nil {
		if !s.Mode().IsRegular() {
			log.Fatal().Msg("dest exists and is not a plain file")
		}
		datetime := time.Now().UTC().Format("20060102150405")
		backuppath = destfile + "." + datetime + ".old"
		if err = h.Rename(destfile, backuppath); err != nil {
			return filename, err
		}
	}

	cf, err := h.Create(destfile, 0664)
	if err != nil {
		return
	}
	defer cf.Close()

	if _, err = io.Copy(cf, from); err != nil {
		return
	}
	fmt.Printf("imported %q to %s:%s\n", source, h.String(), destfile)
	return
}

// ImportCommons copies a file to an instance common directory.
func ImportCommons(r *Host, ct *Component, common string, params []string) (filenames []string, err error) {
	if ct == nil || ct == &RootComponent {
		err = ErrNotSupported
		return
	}

	if len(params) == 0 {
		log.Fatal().Msg("no file/url provided")
	}

	dir := r.PathTo(ct, common)
	for _, source := range params {
		var filename string
		if filename, err = ImportSource(r, dir, source); err != nil && err != ErrExists {
			return
		}
		filenames = append(filenames, filename)
	}
	err = nil // reset in case above returns ErrExists
	return
}
