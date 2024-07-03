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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/hashicorp/go-version"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"

	"github.com/itrs-group/cordial/pkg/config"
)

type downloadauth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// openArchive locates and returns an io.ReadCloser for an archive for
// the component ct. The source of the archive is given as an option. If
// no options are set then the "latest" release from the ITRS releases
// web site is downloaded and returned using stored credentials.
func openArchive(ct *Component, options ...PackageOptions) (body io.ReadCloser, filename string, err error) {
	var resp *http.Response

	opts := evalOptions(options...)

	if !opts.downloadonly {
		if opts.localArchive != "" {
			body, filename, err = open(opts.localArchive, options...)
			if err == nil || !errors.Is(err, ErrIsADirectory) {
				// if success or it's not a directory, return
				return
			}
			log.Debug().Msg("source is a directory, setting local")
			opts.local = true
		} else {
			// default location
			opts.localArchive = path.Join(LocalRoot(), "packages", "downloads")
		}
	}

	if opts.local {
		// archive directory is local only
		if opts.version == "latest" {
			opts.version = ""
		}
		// matching rules for local files
		filename, err = LatestLocalArchive(LOCAL, opts.localArchive, opts.version,
			func(v os.DirEntry) bool {
				check := ct.String()

				if ct.ParentType != nil && len(ct.PackageTypes) > 0 {
					check = ct.ParentType.String()
				}

				if ct.DownloadInfix != "" {
					check = ct.DownloadInfix
				}

				return strings.Contains(v.Name(), check)
			})
		if err != nil {
			log.Debug().Err(err).Msg("latest() returned err")
		}
		if filename == "" {
			err = fmt.Errorf("local installation selected but no suitable file found for %s (%w)", ct, ErrNotExist)
			return
		}
		var f io.ReadSeekCloser
		if f, err = LOCAL.Open(path.Join(opts.localArchive, filename)); err != nil {
			err = fmt.Errorf("local installation selected but no suitable file found for %s (%w)", ct, err)
			return
		}
		body = f
		return
	}

	if filename, resp, err = openRemoteArchive(ct, options...); err != nil {
		return
	}

	LOCAL.MkdirAll(opts.localArchive, 0775)
	archivePath := path.Join(opts.localArchive, filename)
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
		err = nil
		body = resp.Body
		return
	}

	// save the file archive and rewind, return
	var w *os.File
	w, err = os.Create(archivePath)
	if err != nil {
		return
	}
	isterm := term.IsTerminal(int(os.Stderr.Fd()))

	out := io.Discard
	if isterm {
		out = os.Stdout
	}
	bar := progressbar.NewOptions64(
		resp.ContentLength,
		progressbar.OptionSetDescription(filename),
		progressbar.OptionSetWriter(out),
		progressbar.OptionShowBytes(true),
		// progressbar.OptionSetWidth(10),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionEnableColorCodes(true),
	)

	if _, err = io.Copy(io.MultiWriter(w, bar), resp.Body); err != nil {
		return
	}

	if !isterm {
		fmt.Println(bar.String())
	}

	if _, err = w.Seek(0, 0); err != nil {
		return
	}
	body = w

	return
}

// unarchive unpacks the gzipped archive passed as an io.Reader on the
// host given for the component. If there is an error then the caller
// must close the io.Reader
func unarchive(h *Host, ct *Component, archive io.Reader, filename string, options ...PackageOptions) (dir string, err error) {
	var version string

	opts := evalOptions(options...)

	if opts.override == "" {
		var ctFromFile *Component
		ctFromFile, version, err = filenameToComponent(filename)
		if err != nil {
			return
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
			log.Debug().Msgf("component type and archive mismatch: %q is not a %q", filename, ct)
			return
		}
	} else {
		s := strings.SplitN(opts.override, ":", 2)
		if len(s) != 2 {
			err = fmt.Errorf("type/version override must be in the form TYPE:VERSION (%w)", ErrInvalidArgs)
			return
		}
		ct = ParseComponent(s[0])
		if ct == nil {
			return "", fmt.Errorf("invalid component type %q (%w)", s[0], ErrInvalidArgs)
		}
		version = s[1]
		if !matchVersion(version) {
			return "", fmt.Errorf("invalid version %q (%w)", s[1], ErrInvalidArgs)
		}
	}

	// unarchive in non-parent package dir, e.g. fa2 not netprobe
	basedir := h.PathTo("packages", ct.String(), version)
	log.Debug().Msgf("basedir=%s ct=%s version=%s", basedir, ct, version)
	if _, err = h.Stat(basedir); err == nil {
		return h.Path(basedir), fs.ErrExist
	}
	if err = h.MkdirAll(basedir, 0775); err != nil {
		return
	}

	// var fnname func(string) string

	t, err := gzip.NewReader(archive)
	if err != nil {
		// cannot gunzip file
		return
	}
	defer t.Close()
	t.Multistream(false)

	fnname := func(name string) string {
		return strings.TrimPrefix(name, ct.String()+"/")
	}
	if ct.StripArchivePrefix != nil {
		fnname = func(name string) string {
			return strings.TrimPrefix(name, *ct.StripArchivePrefix)
		}
	}

	if err = untar(h, basedir, t, fnname); err != nil {
		return
	}

	fmt.Printf("installed %q to %q\n", filename, h.Path(basedir))
	// options = append(options, Version(version))
	// only create a new base link, not overwrite
	basedir = h.PathTo("packages", ct.String())
	basepath := path.Join(h.Path(basedir), opts.basename)

	log.Debug().Msgf("basepath: %s, version: %s", basepath, version)

	if _, err = h.Stat(basepath); err == nil {
		return h.Path(basedir), nil
	}

	if err = h.Symlink(version, basepath); err != nil {
		log.Debug().Err(err).Msgf("version %s base %s", version, basepath)
		return h.Path(basedir), err
	}

	fmt.Printf("%s %q on %s set to %s\n", ct, path.Base(basepath), h, version)
	return h.Path(basedir), nil
}

// untar the archive from an io.Reader onto host h in directory dir.
// Call stripPrefix for each file to remove configurable prefix
func untar(h *Host, dir string, t io.Reader, stripPrefix func(string) string) (err error) {
	var name string
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

		// do not trust tar archives to contain safe paths

		if name = stripPrefix(hdr.Name); name == "" {
			continue
		}
		if name, err = CleanRelativePath(name); err != nil {
			return
		}
		fullpath := path.Join(dir, name)
		switch hdr.Typeflag {
		case tar.TypeReg:
			// check (and created) containing directories - account for munged tar files
			dir := path.Dir(fullpath)
			if err = h.MkdirAll(dir, 0775); err != nil {
				return
			}

			var out io.WriteCloser
			if out, err = h.Create(fullpath, hdr.FileInfo().Mode()); err != nil {
				return
			}
			var n int64
			n, err = io.Copy(out, tr)
			if err != nil {
				out.Close()
				return
			}
			if n != hdr.Size {
				log.Error().Msgf("lengths different: %d %d", hdr.Size, n)
			}
			out.Close()

		case tar.TypeDir:
			if err = h.MkdirAll(fullpath, hdr.FileInfo().Mode()); err != nil {
				return
			}

		case tar.TypeSymlink, tar.TypeGNULongLink:
			if path.IsAbs(hdr.Linkname) {
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
	return
}

// openRemoteArchive locates and opens a remote software archive file
// using the Geneos download conventions. It returns the underlying
// filename and the archives as a http.Response object.
//
// GeneosOptions supported are PlatformID, UseNexus, UseSnapshots,
// Version, Username and Password. PlatformID and Version cannot be set
// at the same time.
func openRemoteArchive(ct *Component, options ...PackageOptions) (filename string, resp *http.Response, err error) {
	var source string

	opts := evalOptions(options...)

	switch opts.downloadtype {
	case "nexus":
		source, resp, err = openRemoteNexusArchive(ct, opts)
		if err != nil {
			return
		}

	default:
		source, resp, err = openRemoteDefaultArchive(ct, opts)
		if err != nil {
			return
		}
	}

	// process both nexus and resources status codes below
	if resp.StatusCode > 299 {
		resp.Body.Close()
		switch resp.StatusCode {
		case 404:
			fmt.Printf("cannot find %s package that matches version %s\n", ct, opts.version)
			err = fs.ErrNotExist
		default:
			err = fmt.Errorf("cannot access %s package at %q version %s: %s", ct, source, opts.version, resp.Status)
		}
		return
	}

	filename, err = FilenameFromHTTPResp(resp, resp.Request.URL)
	if err != nil {
		return
	}

	log.Debug().Msgf("download check for %s versions %q returned %s (%d bytes)", ct, opts.version, filename, resp.ContentLength)
	return
}

func openRemoteDefaultArchive(ct *Component, opts *geneosOptions) (source string, resp *http.Response, err error) {
	// cannot fetch partial versions for el8 - restriction on download search interface
	platform := ""
	if opts.platformId != "" {
		s := strings.Split(opts.platformId, ":")
		if len(s) > 1 {
			platform = s[1]
		}
	}

	baseurl := config.GetString(config.Join("download", "url"))
	downloadURL, _ := url.Parse(baseurl)
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

	basepaths := strings.FieldsFunc(ct.DownloadBase.Default, func(r rune) bool {
		return unicode.IsSpace(r) || r == ','
	})

	for _, bp := range basepaths {
		// first try plain unauthenticated GET
		basepath, _ := url.Parse(bp)
		basepath.RawQuery = v.Encode()
		source = downloadURL.ResolveReference(basepath).String()

		log.Debug().Msgf("source url: %s", source)

		if resp, err = http.Get(source); err != nil {
			log.Error().Err(err).Msg("source, trying next if configured")
			continue
		}

		if resp.StatusCode < 300 {
			return
		}

		if resp.StatusCode == 404 && platform != "" {
			resp.Body.Close()
			v.Del("title")
			basepath.RawQuery = v.Encode()
			source = downloadURL.ResolveReference(basepath).String()

			log.Debug().Msgf("platform download failed, retry source url: %q", source)
			if resp, err = http.Get(source); err != nil {
				log.Error().Err(err).Msg("source, trying next if configured")
				continue
			}
			if resp.StatusCode < 300 {
				return
			}
		}

		// if that fails, check for creds
		if opts.username == "" {
			creds := config.FindCreds(source, config.SetAppName(execname))
			if creds != nil {
				opts.username = creds.GetString("username")
				opts.password = creds.GetPassword("password")
			}
		}

		// only use auth if required - but save auth for potential reuse below
		var authBody []byte
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			if opts.username != "" {
				da := downloadauth{
					Username: opts.username,
					Password: opts.password.String(),
				}
				authBody, err = json.Marshal(da)
				if err != nil {
					log.Error().Err(err).Msg("source, trying next if configured")
					continue
				}
				// make a copy as bytes.NewBuffer() takes ownership
				ba := bytes.Clone(authBody)
				authReader := bytes.NewBuffer(ba)
				if resp, err = http.Post(source, "application/json", authReader); err != nil {
					log.Error().Err(err).Msg("source, trying next if configured")
					continue
				}
				if resp.StatusCode < 300 {
					return
				}
			}
		}

		if resp.StatusCode == 404 && platform != "" {
			resp.Body.Close()
			// try without platform type (e.g. no '-el8')
			v.Del("title")
			basepath.RawQuery = v.Encode()
			source = downloadURL.ResolveReference(basepath).String()

			log.Debug().Msgf("platform download failed, retry source url: %q", source)
			authReader := bytes.NewBuffer(authBody)
			if resp, err = http.Post(source, "application/json", authReader); err != nil {
				log.Error().Err(err).Msg("source, trying next if configured")
				continue
			}
			if resp.StatusCode < 300 {
				return
			}
		}
		log.Debug().Msgf("%s not found, trying next if configured", source)
	}
	return
}

func openRemoteNexusArchive(ct *Component, opts *geneosOptions) (source string, resp *http.Response, err error) {
	platform := ""
	if opts.platformId != "" {
		s := strings.Split(opts.platformId, ":")
		if len(s) > 1 {
			platform = s[1]
		}
	}

	baseurl := "https://nexus.itrsgroup.com/service/rest/v1/search/assets/download"
	downloadURL, _ := url.Parse(baseurl)

	v := url.Values{}

	v.Set("maven.groupId", "com.itrsgroup.geneos")
	v.Set("maven.extension", "tar.gz")
	v.Set("sort", "version")

	v.Set("repository", opts.downloadbase)
	v.Set("maven.classifier", "linux-x64")
	if platform != "" {
		v.Set("maven.classifier", platform+"-linux-x64")
	}

	if opts.version != "latest" {
		v.Set("maven.baseVersion", opts.version)
	}

	// check for fallback creds
	if opts.username == "" {
		creds := config.FindCreds(baseurl, config.SetAppName(execname))
		if creds != nil {
			opts.username = creds.GetString("username")
			opts.password = creds.GetPassword("password")
		}
	}

	client := &http.Client{}
	var req *http.Request

	artifacts := strings.FieldsFunc(ct.DownloadBase.Nexus, func(r rune) bool {
		return unicode.IsSpace(r) || r == ','
	})

	for _, artifactId := range artifacts {
		v.Set("maven.artifactId", artifactId)
		downloadURL.RawQuery = v.Encode()
		source = downloadURL.String()
		log.Debug().Msgf("nexus url: %s", source)
		if req, err = http.NewRequest("GET", source, nil); err != nil {
			return
		}
		if opts.username != "" {
			log.Debug().Msgf("setting creds for %s", opts.username)
			req.SetBasicAuth(opts.username, opts.password.String())
		}
		if resp, err = client.Do(req); err != nil {
			log.Debug().Err(err).Msg("req failed")
			return
		}
		log.Debug().Msg(resp.Status)
		if resp.StatusCode < 300 {
			return
		}
	}
	return
}

var anchoredVersRE = regexp.MustCompile(`^(\d+(\.\d+){0,2})$`)

func matchVersion(v string) bool {
	return anchoredVersRE.MatchString(v)
}

// LatestLocalArchive returns the name of the latest archive file for
// component ct on host h based on semantic versioning. A prefix filter
// can be used to limit matches and a filter function to further refine
// matches.
//
// If the archive contains metadata then this is checked against the
// host platform_id and if they do not match then the file is skipped
// *unless* the versionFilter has a suffix of "+[meta]", e.g.
// "6.7.0+el8" - note the changes from '-' to '+' in the versionFilter.
func LatestLocalArchive(h *Host, dir, versionFilter string, filterFunc func(os.DirEntry) bool) (latest string, err error) {
	// read all entries from dir and remove any that doe not match versionFilter if it is non-empty
	log.Debug().Msgf("looking in %s (with version filter '%s')", dir, versionFilter)
	entries, err := h.ReadDir(dir)
	if err != nil {
		return
	}
	if versionFilter != "" {
		entries = slices.DeleteFunc(entries, func(d fs.DirEntry) bool { return strings.Contains(d.Name(), versionFilter) })
	}

	var matchingVersions = make(map[string]*version.Version)
	var originals = make(map[string]string, len(entries)) // map of processed names to original entries

	platformid := h.GetString("platform_id")

	for _, dirent := range entries {
		// skip if fails filter function (when set)
		if filterFunc != nil && !filterFunc(dirent) {
			continue
		}

		name := dirent.Name()

		// check if this is a valid release archive
		ct, v, err := filenameToComponent(name)
		if err == nil {
			log.Debug().Msgf("found archive of %s with version %s", ct, v)
			nv, _ := version.NewVersion(v)

			// skip non-matching metadata *unless* the filter string
			// includes it after a "+", e.g. if a user says "use this
			// metadata" then we do
			meta := nv.Metadata()
			if meta != "" && meta != platformid && !strings.HasSuffix(versionFilter, "+"+meta) {
				continue
			}

			matchingVersions[name] = nv
			originals[nv.Original()] = name
			continue
		}
	}

	if len(matchingVersions) == 0 {
		return "", nil
	}

	// map to Collection for sorting
	versions := version.Collection{}
	for _, v := range matchingVersions {
		versions = append(versions, v)
	}
	sort.Sort(versions)

	return originals[versions[len(versions)-1].Original()], nil
}

// split an package archive name into type and version
var archiveRE = regexp.MustCompile(`^geneos-(?<component>[\w-]+)-(?<version>[\d\.]+)[\.-]?linux`)

// filenameToComponent transforms an archive filename and returns the
// component and version or an error if the file format is not
// recognised
func filenameToComponent(filename string) (ct *Component, version string, err error) {
	parts := archiveRE.FindStringSubmatch(filename)
	if len(parts) != 3 {
		err = fmt.Errorf("%q: %w", filename, ErrInvalidArgs)
		return
	}
	version = parts[2]
	// replace '-' prefix of recognised platform suffixes with '+' so work with semver as metadata
	for _, m := range platformSuffixList {
		version = strings.ReplaceAll(version, "-"+m, "+"+m)
	}

	ct = ParseComponent(parts[1])
	return
}
