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
	"encoding/binary"
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
	"strings"
	"time"
	"unicode"

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
func openArchive(ct *Component, options ...PackageOptions) (body io.ReadCloser, filename string, err error) {
	var resp *http.Response

	opts := evalOptions(options...)

	// log.Debug().Msgf("opts: %#v", opts)

	if !opts.downloadonly {
		body, filename, err = openSourceFile(opts.localArchive, options...)
		if err == nil {
			log.Debug().Msgf("source opened, returning")
			return
		} else if !errors.Is(err, ErrIsADirectory) {
			// if success or the error indicates it's a directory,
			// just return
			log.Debug().Err(err).Msgf("source not opened, returning error")
			return
		}
		log.Debug().Msg("source is a directory, setting local directory search flag")
		opts.localOnly = true
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
			log.Error().Err(err).Msg("")
			return
		}
		log.Debug().Msgf("archives: %v", archives)
		archives = slices.DeleteFunc(archives, func(n string) bool {
			return !strings.Contains(n, opts.version)
		})
		log.Debug().Msgf("archives (filters by version): %v", archives)
		if len(archives) > 0 {
			filename = archives[len(archives)-1]
		}

		// filename, err = LatestLocalArchive(func(v os.DirEntry) bool {
		// 	check := ct.String()

		// 	if ct.ParentType != nil && len(ct.PackageTypes) > 0 {
		// 		check = ct.ParentType.String()
		// 	}

		// 	if ct.DownloadInfix != "" {
		// 		check = ct.DownloadInfix
		// 	}

		// 	return strings.Contains(v.Name(), check)
		// }, options...)
		// if err != nil {
		// 	log.Debug().Err(err).Msg("latest() returned err")
		// }
		if filename == "" {
			err = fmt.Errorf("local installation selected but no suitable file found for %s (%w)", ct, fs.ErrNotExist)
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

func getbar(console *os.File, name string, size int64) (bar *progressbar.ProgressBar, isterm bool) {
	isterm = term.IsTerminal(int(console.Fd()))

	out := io.Discard
	if isterm {
		out = console
	}
	bar = progressbar.NewOptions64(
		size,
		progressbar.OptionSetDescription(name),
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
	return
}

// unarchive unpacks the gzipped archive passed as an io.Reader on the
// host given for the component. If there is an error then the caller
// must close the io.Reader
func unarchive(h *Host, ct *Component, archive io.Reader, filename string, options ...PackageOptions) (dest string, err error) {
	var version, platform string

	opts := evalOptions(options...)

	if opts.override == "" {
		var ctFromFile *Component
		ctFromFile, version, platform, err = FilenameToComponentVersion(filename)
		if err != nil {
			return
		}
		if platform != "" {
			version += "+" + platform
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
		ct, version, err = OverrideToComponentVersion(opts.override)
		if err != nil {
			return "", err
		}
	}

	// unarchive in non-parent package dir, e.g. fa2 not netprobe
	basedir := h.PathTo("packages", ct.String(), version)
	log.Debug().Msgf("basedir=%s ct=%s version=%s", basedir, ct, version)
	if _, err = h.Stat(basedir); err == nil {
		return h.HostPath(basedir), fs.ErrExist
	}
	if err = h.MkdirAll(basedir, 0775); err != nil {
		return
	}

	var gziplen uint32
	// var fnname func(string) string
	s, ok := archive.(io.ReadSeeker)
	if ok {
		log.Debug().Msg("can read and seek gzip archive")
		_, err = s.Seek(-4, io.SeekEnd)
		if err != nil {
			log.Debug().Err(err).Msg("")
		} else {
			err = binary.Read(s, binary.LittleEndian, &gziplen)
			if err != nil {
				log.Debug().Err(err).Msg("")
			}
		}
		// reset
		_, err = s.Seek(0, io.SeekStart)
	} else {
		log.Debug().Msg("cannot read and seek gzip archive")
	}
	log.Debug().Msgf("gziplen %d", gziplen)

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

	if err = untar(h, basedir, t, int64(gziplen), fnname); err != nil {
		return
	}

	fmt.Printf("installed %q to %q\n", filename, h.HostPath(basedir))
	// options = append(options, Version(version))
	// only create a new base link, not overwrite
	basedir = h.PathTo("packages", ct.String())
	basepath := path.Join(basedir, opts.basename)

	log.Debug().Msgf("basepath: %s, version: %s", basepath, version)

	if _, err = h.Stat(basepath); err == nil {
		return h.HostPath(basedir), nil
	}

	if err = h.Symlink(version, basepath); err != nil {
		log.Debug().Err(err).Msgf("version %s base %s", version, basepath)
		return h.HostPath(basedir), err
	}

	fmt.Printf("%s %q on %s set to %s\n", ct, path.Base(basepath), h, version)
	return h.HostPath(basedir), nil
}

// untar the archive from an io.Reader onto host h in directory dir.
// Call stripPrefix for each file to remove configurable prefix
func untar(h *Host, dir string, tarfile io.Reader, filelen int64, stripPrefix func(string) string) (err error) {
	var name string

	tr := tar.NewReader(tarfile)
	label := "installing"
	if h != LOCAL {
		label += " on " + h.String()
	}
	bar, isterm := getbar(os.Stdout, label, filelen)

	defer func() {
		if !isterm {
			fmt.Println(bar.String())
		} else {
			bar.Finish()
			bar.Close()
		}
	}()

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
			bar.Add64(n)
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

func getPlatformId(value string) (id string) {
	s := strings.Split(value, ":")
	if len(s) > 1 {
		id = s[1]
	}
	return
}

func openRemoteDefaultArchive(ct *Component, opts *geneosOptions) (source string, resp *http.Response, err error) {
	// cannot fetch partial versions for el8 - restriction on download search interface
	platform := getPlatformId(opts.platformId)

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

// split an package archive name into type and version
//
// geneos-gateway-7.1.0-20240828.194610-12-linux-x64.tar.gz
var archiveRE = regexp.MustCompile(`^geneos-(?<component>[\w-]+)-(?<version>[\d\-\.]+)(-(?<platform>\w+))?-linux`)

// FilenameToComponentVersion transforms an archive filename and returns
// the component and version or an error if the file format is not
// recognised
func FilenameToComponentVersion(filename string) (ct *Component, version, platform string, err error) {
	parts := archiveRE.FindStringSubmatch(filename)
	versionIndex := archiveRE.SubexpIndex("version")
	componentIndex := archiveRE.SubexpIndex("component")

	if versionIndex == -1 || componentIndex == -1 || len(parts) < versionIndex+1 {
		err = fmt.Errorf("%q: filename not in expected format: %w", filename, ErrInvalidArgs)
		return
	}
	version = parts[versionIndex]
	// replace '-' prefix of recognised platform suffixes with '+' so work with semver as metadata
	for _, m := range platformSuffixList {
		version = strings.ReplaceAll(version, "-"+m, "+"+m)
	}

	ct = ParseComponent(parts[componentIndex])
	platformIndex := archiveRE.SubexpIndex("platform")
	if platformIndex != -1 && len(parts) > platformIndex {
		platform = parts[platformIndex]
	}
	return
}

var anchoredVersRE = regexp.MustCompile(`^(\d+(\.\d+){0,2})$`)

func matchVersion(v string) bool {
	return anchoredVersRE.MatchString(v)
}

func OverrideToComponentVersion(override string) (ct *Component, version string, err error) {
	s := strings.SplitN(override, ":", 2)
	if len(s) != 2 {
		err = fmt.Errorf("type/version override must be in the form TYPE:VERSION (%w)", ErrInvalidArgs)
		return
	}
	ct = ParseComponent(s[0])
	if ct == nil {
		err = fmt.Errorf("invalid component type %q (%w)", s[0], ErrInvalidArgs)
		return
	}
	version = s[1]
	if !matchVersion(version) {
		err = fmt.Errorf("invalid version %q (%w)", s[1], ErrInvalidArgs)
		return
	}
	return
}
