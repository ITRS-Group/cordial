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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
)

type downloadauth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// openArchive locates and returns an io.ReadCloser for an archive for
// the component ct. The source of the archive is given as an option. If
// no options are set then the "latest" release from the ITRS releases
// web site is downloaded and returned.
func openArchive(ct *Component, options ...Options) (body io.ReadCloser, filename string, err error) {
	var resp *http.Response

	opts := EvalOptions(options...)

	if opts.source != "" {
		body, filename, err = Open(opts.source, options...)
		if err == nil || !errors.Is(err, ErrIsADirectory) {
			// if success or it's not a directory, return
			return
		}
		log.Debug().Msg("source is a directory, setting local")
		opts.local = true
	}

	if opts.local {
		// archive directory is local only
		if opts.version == "latest" {
			opts.version = ""
		}
		archiveDir := LOCAL.Filepath("packages", "downloads")
		if opts.source != "" {
			archiveDir = opts.source
		}
		filename, err = LatestArchive(LOCAL, archiveDir, opts.version, func(v os.DirEntry) bool {
			log.Debug().Msgf("check %s for %s", v.Name(), ct.String())
			switch ct.String() {
			case "webserver":
				return strings.Contains(v.Name(), "web-server")
			case "fa2":
				return strings.Contains(v.Name(), "fixanalyser2-netprobe")
			case "fileagent":
				return strings.Contains(v.Name(), "file-agent")
			default:
				return strings.Contains(v.Name(), ct.String())
			}
		})
		if err != nil {
			log.Debug().Err(err).Msg("latest() returned err")
		}
		if filename == "" {
			err = fmt.Errorf("local installation selected but no suitable file found for %s (%w)", ct, ErrInvalidArgs)
			return
		}
		var f io.ReadSeekCloser
		if f, err = LOCAL.Open(filepath.Join(archiveDir, filename)); err != nil {
			err = fmt.Errorf("local installation selected but no suitable file found for %s (%w)", ct, err)
			return
		}
		body = f
		return
	}

	if filename, resp, err = openRemoteArchive(ct, options...); err != nil {
		return
	}

	archiveDir := filepath.Join(Root(), "packages", "downloads")
	LOCAL.MkdirAll(archiveDir, 0775)
	archivePath := filepath.Join(archiveDir, filename)
	s, err := LOCAL.Stat(archivePath)
	if err == nil && s.Size() == resp.ContentLength {
		if f, err := LOCAL.Open(archivePath); err == nil {
			log.Debug().Msgf("not downloading, file with same size already exists: %s", archivePath)
			resp.Body.Close()
			return f, filename, nil
		}
	}

	// transient download
	if opts.nosave {
		body = resp.Body
		return
	}

	// save the file archive and rewind, return
	var w *os.File
	w, err = os.Create(archivePath)
	if err != nil {
		return
	}
	fmt.Printf("downloading %s package version %q to %s\n", ct, opts.version, archivePath)
	t1 := time.Now()
	if _, err = io.Copy(w, resp.Body); err != nil {
		return
	}
	t2 := time.Now()
	resp.Body.Close()
	b, dr := resp.ContentLength, t2.Sub(t1).Seconds()
	bps := 0.0
	if dr > 0 {
		bps = float64(b) / dr
	}
	fmt.Printf("downloaded %d bytes in %.3f seconds (%.0f bytes/sec)\n", b, dr, bps)
	if _, err = w.Seek(0, 0); err != nil {
		return
	}
	body = w

	return
}

// unarchive unpacks the gzipped, open archive passed as an io.Reader on
// the host given for the component. If there is anm error then the
// caller must close the io.Reader
func unarchive(h *Host, ct *Component, filename string, gz io.Reader, options ...Options) (err error) {
	var version string

	opts := EvalOptions(options...)

	if opts.override == "" {
		var ctFromFile *Component
		ctFromFile, version, err = FilenameToComponent(filename)
		// check the component in the filename
		// special handling for SANs
		switch ct.Name {
		// XXX abstract this
		case "none", "san", "ca3":
			ct = ctFromFile
		case ctFromFile.Name:
			break
		default:
			// mismatch
			log.Debug().Msgf("component type and archive mismatch: %q is not a %q", filename, ct)
			return
		}
	} else {
		s := strings.SplitN(opts.override, ":", 2)
		if len(s) != 2 {
			err = fmt.Errorf("type/version override must be in the form TYPE:VERSION (%w)", ErrInvalidArgs)
			return
		}
		ct = ParseComponentName(s[0])
		if ct == nil {
			return fmt.Errorf("invalid component type %q (%w)", s[0], ErrInvalidArgs)
		}
		version = s[1]
		if !MatchVersion(version) {
			return fmt.Errorf("invalid version %q (%w)", s[1], ErrInvalidArgs)
		}
	}

	basedir := h.Filepath("packages", ct, version)
	log.Debug().Msg(basedir)
	if _, err = h.Stat(basedir); err == nil {
		// something is already using that dir
		// XXX - option to delete and overwrite?
		return
	}
	if err = h.MkdirAll(basedir, 0775); err != nil {
		return
	}

	t, err := gzip.NewReader(gz)
	if err != nil {
		// cannot gunzip file
		return
	}
	defer t.Close()

	var name string
	var fnname func(string) string

	switch ct.Name {
	case "webserver":
		fnname = func(name string) string { return name }
	case "fa2":
		fnname = func(name string) string {
			return strings.TrimPrefix(name, "fix-analyser2/")
		}
	case "fileagent":
		fnname = func(name string) string {
			return strings.TrimPrefix(name, "agent/")
		}
	default:
		fnname = func(name string) string {
			return strings.TrimPrefix(name, ct.String()+"/")
		}
	}

	tr := tar.NewReader(t)
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
		// strip leading component name (XXX - except webserver)
		// do not trust tar archives to contain safe paths

		if name = fnname(hdr.Name); name == "" {
			continue
		}
		if name, err = CleanRelativePath(name); err != nil {
			return
		}
		fullpath := path.Join(basedir, name)
		switch hdr.Typeflag {
		case tar.TypeReg:
			// check (and created) containing directories - account for munged tar files
			dir := path.Dir(fullpath)
			if err = h.MkdirAll(dir, 0775); err != nil {
				return
			}

			var out io.WriteCloser
			if out, err = h.Create(fullpath, hdr.FileInfo().Mode()); err != nil {
				return err
			}
			n, err := io.Copy(out, tr)
			if err != nil {
				out.Close()
				return err
			}
			if n != hdr.Size {
				log.Error().Msgf("lengths different: %s %d", hdr.Size, n)
			}
			out.Close()

		case tar.TypeDir:
			if err = h.MkdirAll(fullpath, hdr.FileInfo().Mode()); err != nil {
				return
			}

		case tar.TypeSymlink, tar.TypeGNULongLink:
			if filepath.IsAbs(hdr.Linkname) {
				err = fmt.Errorf("archive contains absolute symlink target")
				return
			}
			if _, err = h.Stat(fullpath); err != nil {
				if err = h.Symlink(hdr.Linkname, fullpath); err != nil {
					return
				}

			}

		default:
			log.Warn().Msgf("unsupported file type %c\n", hdr.Typeflag)
		}
	}

	fmt.Printf("installed %q to %q\n", filename, h.Path(basedir))
	options = append(options, Version(version))
	return Update(h, ct, options...)
}

// openRemoteArchive locates and opens a remote software archive file
// using the Geneos download conventions. It returns the underlying
// filename and the archives as a http.Response object.
//
// GeneosOptions supported are PlatformID, UseNexus, UseSnapshots,
// Version, Username and Password. PlatformID and Version cannot be set
// at the same time.
func openRemoteArchive(ct *Component, options ...Options) (filename string, resp *http.Response, err error) {
	var source string

	opts := EvalOptions(options...)

	// cannot fetch partial versions for el8 - restriction on download search interface
	platform := ""
	if opts.platform_id != "" {
		s := strings.Split(opts.platform_id, ":")
		if len(s) > 1 {
			platform = s[1]
		}
	}

	switch opts.downloadtype {
	case "nexus":
		baseurl := "https://nexus.itrsgroup.com/service/rest/v1/search/assets/download"
		downloadURL, _ := url.Parse(baseurl)
		v := url.Values{}

		v.Set("maven.groupId", "com.itrsgroup.geneos")
		v.Set("maven.extension", "tar.gz")
		v.Set("sort", "version")

		v.Set("repository", opts.downloadbase)
		v.Set("maven.artifactId", ct.DownloadBase.Nexus)
		v.Set("maven.classifier", "linux-x64")
		if platform != "" {
			v.Set("maven.classifier", platform+"-linux-x64")
		}

		if opts.version != "latest" {
			v.Set("maven.baseVersion", opts.version)
		}

		downloadURL.RawQuery = v.Encode()
		source = downloadURL.String()

		log.Debug().Msgf("nexus url: %s", source)

		if opts.username != "" {
			var req *http.Request
			client := &http.Client{}
			if req, err = http.NewRequest("GET", source, nil); err != nil {
				return
			}
			req.SetBasicAuth(opts.username, string(opts.password))
			if resp, err = client.Do(req); err != nil {
				return
			}
		} else {
			if resp, err = http.Get(source); err != nil {
				return
			}
		}

	default:
		baseurl := config.GetString("download.url")
		downloadURL, _ := url.Parse(baseurl)
		realpath, _ := url.Parse(ct.DownloadBase.Resources)
		v := url.Values{}

		v.Set("os", "linux")
		if opts.version != "latest" {
			if platform != "" {
				log.Error().Msgf("cannot download specific version for this platform (%q) - please download manually", platform)
				err = ErrInvalidArgs
				return
			}
			v.Set("title", opts.version)
		} else if platform != "" {
			v.Set("title", "-"+platform)
		}

		realpath.RawQuery = v.Encode()
		source = downloadURL.ResolveReference(realpath).String()

		log.Debug().Msgf("source url: %s", source)

		if resp, err = http.Get(source); err != nil {
			return
		}

		if resp.StatusCode == 404 && platform != "" {
			resp.Body.Close()
			v.Del("title")
			realpath.RawQuery = v.Encode()
			source = downloadURL.ResolveReference(realpath).String()

			log.Debug().Msgf("platform download failed, retry source url: %q", source)
			if resp, err = http.Get(source); err != nil {
				return
			}
		}

		// only use auth if required - but save auth for potential reuse below
		var auth_body []byte
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			if opts.username != "" {
				da := downloadauth{opts.username, string(opts.password)}
				auth_body, err = json.Marshal(da)
				if err != nil {
					return
				}
				ba := auth_body
				auth_reader := bytes.NewBuffer(ba)
				if resp, err = http.Post(source, "application/json", auth_reader); err != nil {
					return
				}
			}
		}

		if resp.StatusCode == 404 && platform != "" {
			resp.Body.Close()
			// try without platform type (e.g. no '-el8')
			v.Del("title")
			realpath.RawQuery = v.Encode()
			source = downloadURL.ResolveReference(realpath).String()

			log.Debug().Msgf("platform download failed, retry source url: %q", source)
			auth_reader := bytes.NewBuffer(auth_body)
			if resp, err = http.Post(source, "application/json", auth_reader); err != nil {
				return
			}
		}
	}

	if resp.StatusCode > 299 {
		err = fmt.Errorf("cannot access %s package at %q version %s: %s", ct, source, opts.version, resp.Status)
		resp.Body.Close()
		return
	}

	filename, err = FilenameFromHTTPResp(resp, resp.Request.URL)
	if err != nil {
		return
	}

	log.Debug().Msgf("download check for %s versions %q returned %s (%d bytes)", ct, opts.version, filename, resp.ContentLength)
	return
}

var anchoredVersRE = regexp.MustCompile(`^(\d+(\.\d+){0,2})$`)

func MatchVersion(v string) bool {
	return anchoredVersRE.MatchString(v)
}

// LatestArchive returns the latest archive file for component ct on
// host r based on semver. A prefix filter can be used to limit matches
// and a filter function to further refine matches.
//
// If there is semver metadata, check for platform_id on host and remove
// any non-platform metadata from list before sorting
func LatestArchive(r *Host, dir, filterString string, filterFunc func(os.DirEntry) bool) (latest string, err error) {
	ents, err := r.ReadDir(dir)
	if err != nil {
		return
	}

	// remove (in place) all entries that do not contain 'filterString'
	if filterString != "" {
		i := 0
		for _, d := range ents {
			if strings.Contains(d.Name(), filterString) {
				ents[i] = d
				i++
			}
		}
		ents = ents[:i]
	}

	var versions = make(map[string]*version.Version)
	var originals = make(map[string]string, len(ents)) // map of processed names to original entries

	for _, d := range ents {
		// skip if fails filter function (when set)
		if filterFunc != nil && !filterFunc(d) {
			continue
		}

		n := d.Name()

		// check if this is a valid release archive
		ct, v, err := FilenameToComponent(n)
		if err == nil {
			log.Debug().Msgf("found archive of %s with version %s", ct, v)
			nv, _ := version.NewVersion(v)
			versions[n] = nv
			originals[nv.Original()] = n
			continue
		}

		// originals[n] = n

		// // get the first non numeric part, remove it
		// // this deals with "RA" vs "GA"
		// v1p := strings.FieldsFunc(n, func(r rune) bool {
		// 	return !unicode.IsLetter(r)
		// })
		// if len(v1p) > 0 && v1p[0] != "" {
		// 	p := strings.TrimPrefix(n, v1p[0])
		// 	originals[p] = n
		// 	n = p
		// }

		// v1, err := version.NewVersion(n)
		// if err == nil { // valid version
		// 	if v1.Metadata() != "" {
		// 		delete(versions, v1.Core().String())
		// 	}
		// 	versions[n] = v1
		// }
	}

	if len(versions) == 0 {
		return "", nil
	}

	// map to slice for sorting
	vers := []*version.Version{}
	for _, v := range versions {
		vers = append(vers, v)
	}
	sort.Sort(version.Collection(vers))

	return originals[vers[len(vers)-1].Original()], nil
}
