package instance

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func ImportFile(h *geneos.Host, home string, source string, options ...geneos.Options) (filename string, err error) {
	var backuppath string
	var from io.ReadCloser

	if h == geneos.ALL {
		err = geneos.ErrInvalidArgs
		return
	}

	// destdir becomes the absolute path for the imported file
	destdir := home
	// destfile is the basename of the import path, empty if the source
	// filename should be kept
	destfile := ""

	// if the source is a http(s) url then skip '=' split (protect queries in URL)
	if !strings.HasPrefix(source, "https://") && !strings.HasPrefix(source, "http://") {
		splitsource := strings.SplitN(source, "=", 2)
		if len(splitsource) > 1 {
			// do some basic validation on user-supplied destination
			if splitsource[0] == "" {
				log.Fatal().Msg("dest path empty")
			}
			destfile, err = geneos.CleanRelativePath(splitsource[0])
			if err != nil {
				log.Fatal().Msg("dest path must be relative to (and in) instance directory")
			}
			// if the destination exists is it a directory?
			if s, err := h.Stat(filepath.Join(home, destfile)); err == nil {
				if s.IsDir() {
					destdir = filepath.Join(home, destfile)
					destfile = ""
				}
			}
			source = splitsource[1]
			if source == "" {
				log.Fatal().Msg("no source defined")
			}
		}
	}

	from, filename, err = geneos.Open(source)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	defer from.Close()

	if destfile == "" {
		destfile = filename
	}
	// return final basename
	filename = filepath.Base(destfile)
	destfile = filepath.Join(destdir, destfile)

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

func ImportCommons(r *geneos.Host, ct *geneos.Component, common string, params []string) (filename string, err error) {
	if ct == nil || !ct.RealComponent {
		err = geneos.ErrNotSupported
		return
	}

	if len(params) == 0 {
		log.Fatal().Msg("no file/url provided")
	}

	dir := r.Filepath(ct, common)
	for _, source := range params {
		if filename, err = ImportFile(r, dir, source); err != nil {
			return
		}
	}
	return
}
