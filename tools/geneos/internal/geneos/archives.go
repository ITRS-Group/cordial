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
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	zlog "github.com/rs/zerolog/log"
)

type downloadauth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type dirtime struct {
	Name     string
	Modified time.Time
}

var osMap = map[string]string{
	"amd64":   "x64",
	"x86_64":  "x64",
	"x64":     "x64",
	"arm64":   "aarch64",
	"aarch64": "aarch64",
}

// openArchive locates and returns an io.ReadCloser for an archive for
// the component ct.
//
// Options to control behaviour are passed as PackageOptions.
//
// If no options are set then the "latest" release from the ITRS
// releases web site is downloaded and returned using stored
// credentials. The source of the archive is given as an option.
//
// The order of precedence for options is:
//
// LocalOnly()
// DownloadOnly()
func openArchive(ct *Component, options ...PackageOption) (body io.ReadCloser, filename string, filesize int64, err error) {
	var resp *http.Response

	opts := evalOptions(options...)

	if !opts.downloadonly {
		// try to open the given source directly first, if it exists and
		// is not a directory then return it

		zlog.Debug().Msgf("source %q", opts.source)
		body, filename, filesize, err = openSource(opts.source, options...)
		if err == nil {
			zlog.Debug().Msgf("source opened, returning")
			return
		}
		if IsDir(opts.source) && opts.source != path.Join(LocalRoot(), "packages", "downloads") {
			zlog.Debug().Msg("source is a directory, and not the download cache, so setting local directory search flag")
			opts.localOnly = true
		}
	}

	if opts.localOnly {
		// archive directory is local only
		if opts.version == "latest" {
			opts.version = ""
		}
		// matching rules for local files
		var archives []string
		archives, err = LocalArchives(ct, options...)
		if err != nil {
			zlog.Error().Err(err).Msg("")
			return
		}
		zlog.Debug().Msgf("archives: %v", archives)
		// filter list of archives by version
		archives = slices.DeleteFunc(archives, func(n string) bool {
			var ctMatch bool
			if ct.DownloadInfix != "" {
				ctMatch = strings.Contains(n, ct.DownloadInfix)
			} else {
				ctMatch = strings.Contains(n, ct.String())
			}

			if opts.version != "" {
				return !ctMatch || !strings.Contains(n, opts.version)
			}

			return !ctMatch
		})
		zlog.Debug().Msgf("archives (filtered by version %q): %v", opts.version, archives)
		if len(archives) > 0 {
			filename = archives[len(archives)-1]
		}

		if filename == "" {
			err = fmt.Errorf("local installation selected but no suitable file found for %s (%w)", ct, fs.ErrNotExist)
			return
		}
		var f io.ReadSeekCloser
		fp := path.Join(opts.source, filename)
		if f, err = LOCAL.Open(fp); err != nil {
			err = fmt.Errorf("local installation selected but no suitable file found for %s (%w)", ct, err)
			return
		}
		body = f

		var s fs.FileInfo
		s, err = LOCAL.Stat(fp)
		if err != nil {
			// Can this fail?
		}
		filesize = s.Size()
		return
	}

	// else try one of the built-in download methods
	if filename, resp, err = openRemoteArchive(ct, options...); err != nil {
		return
	}

	LOCAL.MkdirAll(opts.source, 0775)
	archivePath := path.Join(opts.source, filename)
	s, err := LOCAL.Stat(archivePath)
	if err == nil && s.Size() == resp.ContentLength {
		if f, err := LOCAL.Open(archivePath); err == nil {
			zlog.Debug().Msgf("not downloading, file with same size already exists: %s", archivePath)
			resp.Body.Close()
			return f, filename, -1, nil
		}
	}

	filesize = resp.ContentLength

	// transient download
	if opts.nosave {
		err = nil
		body = resp.Body
		return
	}

	// ensure destination directory exists
	if err = LOCAL.MkdirAll(opts.source, 0775); err != nil {
		zlog.Debug().Err(err).Msgf("cannot create directory for archive: %q", opts.source)
		return
	}

	// save the file archive and rewind, return
	var w *os.File
	w, err = os.Create(archivePath)
	if err != nil {
		return
	}
	bar, isterm := getbar(os.Stdout, filename, resp.ContentLength)

	if _, err = io.Copy(io.MultiWriter(w, bar), resp.Body); err != nil {
		return
	}

	defer func() {
		if !isterm {
			fmt.Println(bar.String())
		}
		bar.Close()
	}()

	if _, err = w.Seek(0, 0); err != nil {
		return
	}
	body = w

	return
}

// unarchive unpacks the gzipped archive passed as an io.Reader on the
// host given for the component. If there is an error then the caller
// must close the io.Reader
func unarchive(h *Host, ct *Component, archive io.Reader, filename string, filesize int64, options ...PackageOption) (dest string, err error) {
	var version, platform, suffix string

	opts := evalOptions(options...)

	if opts.override == "" {
		var ctFromFile *Component
		ctFromFile, version, platform, suffix, err = FilenameToComponentVersion(ct, filename)
		if err != nil {
			return
		}
		if platform != "" {
			version += "+" + platform
		}
		zlog.Debug().Msgf("ctFromFile %q, version %q, platform %q, suffix %q", ctFromFile, version, platform, suffix)
		if ctFromFile == nil {
			ctFromFile = &RootComponent
		}
		// check the component in the filename
		// special handling for SANs
		switch ct.Name {
		// XXX abstract this
		// components that use other components... i.e. netprobes
		case RootComponentName, "san", "floating", "ca3":
			ct = ctFromFile
		case ctFromFile.Name:
			break
		default:
			// mismatch
			zlog.Debug().Msgf("component type and archive mismatch: %q is not a %q", filename, ct)
			return
		}
	} else {
		ct, version, err = OverrideToComponentVersion(opts.override)
		if err != nil {
			return "", err
		}

		// grab suffix
		suffix = strings.TrimPrefix(path.Ext(filename), ".")
	}

	// unarchive in non-parent package dir, e.g. fa2 not netprobe
	basedir := h.PathTo("packages", ct.String(), version)
	zlog.Debug().Msgf("basedir=%s ct=%s version=%s suffix=%s", basedir, ct, version, suffix)
	if _, err = h.Stat(basedir); err == nil {
		return h.HostPath(basedir), fs.ErrExist
	}
	if err = h.MkdirAll(basedir, 0775); err != nil {
		return
	}

	stripFirstDir := func(name string) string {
		if s := strings.Index(name, "/"); s != -1 {
			name = name[s+1:]
		}
		return name
	}
	if ct.ArchiveLeaveFirstDir {
		stripFirstDir = nil
	}

	switch suffix {
	case "tar.gz", "gz", "tgz":
		var gziplen uint32
		s, ok := archive.(io.ReadSeeker)
		if ok {
			_, err = s.Seek(-4, io.SeekEnd)
			if err != nil {
				zlog.Debug().Err(err).Msg("")
			} else {
				err = binary.Read(s, binary.LittleEndian, &gziplen)
				if err != nil {
					zlog.Debug().Err(err).Msg("")
				}
			}
			// reset
			_, err = s.Seek(0, io.SeekStart)
		}
		zlog.Debug().Msgf("gziplen %d", gziplen)

		var tin *gzip.Reader
		tin, err = gzip.NewReader(archive)
		if err != nil {
			// cannot gunzip file
			return
		}
		defer tin.Close()
		tin.Multistream(false)

		if err = untar(h, basedir, tin, int64(gziplen), stripFirstDir); err != nil {
			return
		}

	case "zip":
		if err = unzip(h, basedir, archive, filesize, stripFirstDir); err != nil {
			return
		}

	default:
		err = fmt.Errorf("file extension %q not valid", suffix)
		return
	}

	fmt.Printf("installed %q to %q\n", filename, h.HostPath(basedir))

	// only create a new base link, not overwrite
	basedir = h.PathTo("packages", ct.String())
	basepath := path.Join(basedir, opts.basename)

	zlog.Debug().Msgf("basepath: %s, version: %s", basepath, version)

	if _, err = h.Stat(basepath); err == nil {
		return h.HostPath(basedir), nil
	}

	if err = h.Symlink(version, basepath); err != nil {
		zlog.Debug().Err(err).Msgf("version %s base %s", version, basepath)
		return h.HostPath(basedir), err
	}

	fmt.Printf("%s %q on %s set to %s\n", ct, path.Base(basepath), h, version)
	return h.HostPath(basedir), nil

}
