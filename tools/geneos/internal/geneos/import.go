/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package geneos

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/rs/zerolog/log"
)

// ImportFile copies the file from source to the directory dir on host
// h. The destination filename can be given as a "NAME=" prefix in
// source. If no filename is given then it is derived from the source.
//
// source can be a path to a file or a http/https URL.
//
// If source is a (local) file and it is the same as the destination then an
// ErrExists is returned.
func ImportFile(h *Host, dir string, source string) (filename string, err error) {
	var backuppath string
	var from io.ReadCloser
	var isPlain bool // plain file as source?

	if h == ALL {
		err = ErrInvalidArgs
		return
	}

	// destdir becomes the absolute path for the imported file
	destdir := dir
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
			log.Fatal().Msg("dest path empty")
		}
		if destfile, err = CleanRelativePath(s[0]); err != nil {
			log.Fatal().Msg("dest path must be relative to (and in) instance directory")
		}
		// if the destination exists is it a directory?
		if s, err := h.Stat(path.Join(dir, destfile)); err == nil {
			if s.IsDir() {
				destdir = path.Join(dir, destfile)
				destfile = ""
			}
		}
	} else {
		isPlain = true
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
			if s, err := h.Stat(path.Join(dir, destfile)); err == nil {
				if s.IsDir() {
					destdir = path.Join(dir, destfile)
					destfile = ""
				}
			}
			source = s[1]
			if source == "" {
				log.Fatal().Msg("no source defined")
			}
		}
	}

	from, filename, err = open(source)
	if err != nil {
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
	if isPlain && h.IsLocal() {
		if strings.HasPrefix(source, "~/") {
			home, _ := config.UserHomeDir()
			source = path.Join(home, strings.TrimPrefix(source, "~/"))
		}
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
		if filename, err = ImportFile(r, dir, source); err != nil && err != ErrExists {
			return
		}
		filenames = append(filenames, filename)
	}
	err = nil // reset in case above returns ErrExists
	return
}
