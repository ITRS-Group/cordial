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
	"log/slog"
	"net/http"
	"os"
	"path"
	"slices"
	"strings"
	"time"
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

// OpenArchive locates and returns an io.ReadCloser for an archive for
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
func OpenArchive(ct *Component, options ...PackageOption) (body io.ReadCloser, filename string, filesize int64, err error) {
	var resp *http.Response

	opts := evalOptions(options...)

	if !opts.downloadonly {
		// try to open the given source directly first, if it exists and
		// is not a directory then return it

		log.Debug("source", slog.String("src", opts.source))
		body, filename, filesize, err = openSource(opts.source, options...)
		if err == nil {
			log.Debug("source opened, returning")
			return
		}
		if IsDir(opts.source) && opts.source != path.Join(LocalRoot(), "packages", "downloads") {
			log.Debug("source is a directory, and not the download cache, so setting local directory search flag")
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
			log.Error("error retrieving local archives", slog.Any("error", err))
			return
		}
		log.Debug("local archives found", slog.Any("archives", archives))
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
		log.Debug("archives (filtered by version)", slog.String("version", opts.version), slog.Any("archives", archives))
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
			log.Debug("not downloading, file with same size already exists", slog.String("file", archivePath))
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
		log.Error("cannot create directory for archive", slog.String("directory", opts.source), slog.Any("error", err))
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
		log.Debug("ctFromFile", slog.Any("ctFromFile", ctFromFile), slog.String("version", version), slog.String("platform", platform), slog.String("suffix", suffix))
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
			log.Debug("component type and archive mismatch", slog.String("filename", filename), slog.String("expected", ct.String()))
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
	log.Debug("basedir", slog.String("basedir", basedir), slog.String("ct", ct.String()), slog.String("version", version), slog.String("suffix", suffix))
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
				log.Debug("error seeking in archive", slog.Any("error", err))
			} else {
				err = binary.Read(s, binary.LittleEndian, &gziplen)
				if err != nil {
					log.Debug("error reading gzip length", slog.Any("error", err))
				}
			}
			// reset
			_, err = s.Seek(0, io.SeekStart)
		}
		log.Debug("gzip length", slog.Int64("gziplen", int64(gziplen)))

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

	log.Debug("installed archive", slog.String("filename", filename), slog.String("destination", h.HostPath(basedir)))

	// only create a new base link, not overwrite
	basedir = h.PathTo("packages", ct.String())
	basepath := path.Join(basedir, opts.basename)

	log.Debug("basepath", slog.String("basepath", basepath), slog.String("version", version))

	if _, err = h.Stat(basepath); err == nil {
		return h.HostPath(basedir), nil
	}

	if err = h.Symlink(version, basepath); err != nil {
		log.Debug("error creating symlink", slog.String("version", version), slog.String("basepath", basepath), slog.Any("error", err))
		return h.HostPath(basedir), err
	}

	fmt.Printf("%s %q on %s set to %s\n", ct, path.Base(basepath), h, version)
	return h.HostPath(basedir), nil

}
