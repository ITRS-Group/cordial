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
	"archive/zip"
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

	"github.com/itrs-group/cordial"
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
func openArchive(ct *Component, options ...PackageOptions) (body io.ReadCloser, filename string, filesize int64, err error) {
	var resp *http.Response

	opts := evalOptions(options...)

	if !opts.downloadonly {
		log.Debug().Msgf("localArchive %q", opts.localArchive)
		body, filename, filesize, err = openSourceFile(opts.localArchive, options...)
		if err == nil {
			log.Debug().Msgf("source opened, returning")
			return
		} else if !errors.Is(err, ErrIsADirectory) {
			// if success or the error indicates it's a directory,
			// just return
			log.Debug().Err(err).Msgf("source %q not opened as %q, returning error", opts.localArchive, filename)
			return
		}
		if opts.localArchive != path.Join(LocalRoot(), "packages", "downloads") {
			log.Debug().Msg("source is a directory, setting local directory search flag")
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

		if filename == "" {
			err = fmt.Errorf("local installation selected but no suitable file found for %s (%w)", ct, fs.ErrNotExist)
			return
		}
		var f io.ReadSeekCloser
		filepath := path.Join(opts.localArchive, filename)
		if f, err = LOCAL.Open(filepath); err != nil {
			err = fmt.Errorf("local installation selected but no suitable file found for %s (%w)", ct, err)
			return
		}
		body = f

		var s fs.FileInfo
		s, err = LOCAL.Stat(filepath)
		if err != nil {
			// Can this fail?
		}
		filesize = s.Size()
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

type dirtime struct {
	Name     string
	Modified time.Time
}

// unarchive unpacks the gzipped archive passed as an io.Reader on the
// host given for the component. If there is an error then the caller
// must close the io.Reader
func unarchive(h *Host, ct *Component, archive io.Reader, filename string, filesize int64, options ...PackageOptions) (dest string, err error) {
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
		log.Debug().Msgf("ctFromFile %q, version %q, platform %q, suffix %q", ctFromFile, version, platform, suffix)
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
			log.Debug().Msgf("component type and archive mismatch: %q is not a %q", filename, ct)
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
	log.Debug().Msgf("basedir=%s ct=%s version=%s suffix=%s", basedir, ct, version, suffix)
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
				log.Debug().Err(err).Msg("")
			} else {
				err = binary.Read(s, binary.LittleEndian, &gziplen)
				if err != nil {
					log.Debug().Err(err).Msg("")
				}
			}
			// reset
			_, err = s.Seek(0, io.SeekStart)
		}
		log.Debug().Msgf("gziplen %d", gziplen)

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

func unzip(h *Host, basedir string, archive io.Reader, filesize int64, stripPrefix func(string) string) (err error) {
	// zip files are read into memory for now
	var z *zip.Reader
	var r []byte
	r, err = io.ReadAll(archive)
	if err != nil {
		panic(err)
	}
	b := bytes.NewReader(r)
	z, err = zip.NewReader(b, int64(len(r)))

	label := "installing"
	if h != LOCAL {
		label += " on " + h.String()
	}
	bar, isterm := getbar(os.Stdout, label, int64(len(r)))

	defer func() {
		if !isterm {
			fmt.Println(bar.String())
		} else {
			bar.Finish()
			bar.Close()
		}
	}()

	var dirtimes []dirtime

	for _, f := range z.File {
		var name string
		if stripPrefix != nil {
			if name = stripPrefix(f.Name); name == "" {
				continue
			}
		}
		name, err = CleanRelativePath(name)
		if err != nil {
			panic(err)
		}
		fullpath := path.Join(basedir, name)

		// dir
		if strings.HasSuffix(f.Name, "/") {
			if err = h.MkdirAll(fullpath, f.Mode()); err != nil {
				panic(err)
			}
			// save the dir to update modified times later
			dirtimes = append(dirtimes, dirtime{Name: fullpath, Modified: f.Modified})
			continue
		}

		var c io.ReadCloser
		c, err = f.Open()
		if err != nil {
			panic(err)
		}

		dir := path.Dir(fullpath)
		var st fs.FileInfo
		if st, err = h.Stat(dir); errors.Is(err, fs.ErrNotExist) {
			if err = h.MkdirAll(dir, 0775); err != nil {
				c.Close()
				return
			}
		} else if !st.IsDir() {
			panic(err)
		}

		var out io.WriteCloser
		if out, err = h.Create(fullpath, f.Mode()); err != nil {
			c.Close()
			return
		}

		var n int64
		n, err = io.CopyN(out, c, int64(f.UncompressedSize64))
		if err != nil {
			out.Close()
			c.Close()
			return
		}
		bar.Add64(int64(f.CompressedSize64))
		if n != int64(f.UncompressedSize64) {
			log.Error().Msgf("lengths different: %d %d", f.UncompressedSize64, n)
		}
		out.Close()
		c.Close()

		if err := h.Chtimes(fullpath, time.Time{}, f.Modified); err != nil {
			log.Debug().Err(err).Msg("cannot update mtime (symlink?)")
		}
	}

	slices.Reverse(dirtimes)
	for _, d := range dirtimes {
		log.Debug().Msgf("updating %q to %v", d.Name, d.Modified)
		if err := h.Chtimes(d.Name, time.Time{}, d.Modified); err != nil {
			log.Warn().Err(err).Msgf("cannot update mtime on %q", d.Name)
		}
	}
	return
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

	var dirtimes []dirtime

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

		name = hdr.Name
		if stripPrefix != nil {
			if name = stripPrefix(hdr.Name); name == "" {
				continue
			}
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
			if err := h.Chtimes(fullpath, hdr.AccessTime, hdr.ModTime); err != nil {
				log.Warn().Err(err).Msg("cannot update mtime")
			}
		case tar.TypeDir:
			if err = h.MkdirAll(fullpath, hdr.FileInfo().Mode()); err != nil {
				return
			}
			dirtimes = append(dirtimes, dirtime{Name: fullpath, Modified: hdr.ModTime})

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
			if err := h.Lchtimes(fullpath, hdr.AccessTime, hdr.ModTime); h == LOCAL && err != nil {
				// ignore on remotes, sftp cannot set times on symlinks
				log.Debug().Err(err).Msg("cannot update mtime (symlink?)")
			}

		default:
			log.Warn().Msgf("unsupported file type %c\n", hdr.Typeflag)
		}

	}

	slices.Reverse(dirtimes)
	for _, d := range dirtimes {
		if err := h.Chtimes(d.Name, time.Time{}, d.Modified); err != nil {
			log.Warn().Err(err).Msgf("cannot update mtime on %q", d.Name)
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

var osMap = map[string]string{
	"amd64":   "x64",
	"x86_64":  "x64",
	"x64":     "x64",
	"arm64":   "aarch64",
	"aarch64": "aarch64",
}

func openRemoteDefaultArchive(ct *Component, opts *packageOptions) (source string, resp *http.Response, err error) {
	// cannot fetch partial versions for el8 - restriction on download search interface
	platform := getPlatformId(opts.platformId)

	baseurl := config.GetString(config.Join("download", "url"))
	downloadURL, _ := url.Parse(baseurl)

	os := opts.host.GetString("os")
	arch := osMap[opts.host.GetString("arch")]

	v := url.Values{}

	if ct.DownloadParams == nil {
		v.Set("os", os)

		if opts.version != "latest" {
			if platform != "" {
				log.Error().Msgf("cannot download specific version for this platform (%q) - please download manually", platform)
				err = ErrInvalidArgs
				return
			}
			v.Set("title", opts.version)
		} else if platform != "" {
			v.Set("title", "-"+platform+"-"+os+"-"+arch)
		} else {
			v.Set("title", os+"-"+arch)
		}
	} else {
		for _, param := range *ct.DownloadParams {
			s := strings.SplitN(param, "=", 2)
			if len(s) != 2 {
				continue
			}
			v.Set(s[0], s[1])
		}
		if opts.version != "latest" {
			v.Set("title", opts.version)
		}
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

		// if we get a not found status and we are looking for a
		// platform specific archive then also try without the platform
		// in case the release is not available for the platform, eg
		// webserver
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
			creds := config.FindCreds(source, config.SetAppName(cordial.ExecutableName()))
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

func openRemoteNexusArchive(ct *Component, opts *packageOptions) (source string, resp *http.Response, err error) {
	os := opts.host.GetString("os")
	arch := osMap[opts.host.GetString("arch")]

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
	v.Set("sort", "version")
	v.Set("repository", opts.downloadbase)

	if ct.DownloadParamsNexus == nil {
		v.Set("maven.groupId", "com.itrsgroup.geneos")
		v.Set("maven.extension", "tar.gz")
		v.Set("maven.classifier", os+"-"+arch)
	} else {
		for _, param := range *ct.DownloadParamsNexus {
			s := strings.SplitN(param, "=", 2)
			if len(s) != 2 {
				continue
			}
			v.Set(s[0], s[1])
		}
	}

	if platform != "" && ct.DownloadParamsNexus == nil {
		v.Set("maven.classifier", platform+"-"+os+"-"+arch)
	}

	if opts.version != "latest" {
		v.Set("maven.baseVersion", opts.version)
	}

	// check for fallback creds
	if opts.username == "" {
		creds := config.FindCreds(baseurl, config.SetAppName(cordial.ExecutableName()))
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
var archiveRE = regexp.MustCompile(`^geneos-(?<component>[\w-]+)-(?<version>[\d\-\.]+)(-(?<platform>\w+))?[\.-]linux.*?\.(?<suffix>[\w\.]+)$`)

// FilenameToComponentVersion transforms an archive filename and returns
// the component and version or an error if the file format is not
// recognised
func FilenameToComponentVersion(oct *Component, filename string) (ct *Component, version, platform, suffix string, err error) {
	var re *regexp.Regexp
	var parts []string

	for nct := range oct.OrList() {
		re = archiveRE
		if nct.DownloadNameRegexp != nil {
			re = nct.DownloadNameRegexp
		}
		parts = re.FindStringSubmatch(filename)
		if len(parts) > 0 {
			break
		}
	}

	if len(parts) == 0 {
		err = fmt.Errorf("%q: filename not in expected format: %w", filename, ErrInvalidArgs)
		return
	}
	versionIndex := re.SubexpIndex("version")
	componentIndex := re.SubexpIndex("component")
	suffixIndex := re.SubexpIndex("suffix")

	if versionIndex == -1 || componentIndex == -1 || suffixIndex == -1 || len(parts) < versionIndex+1 {
		err = fmt.Errorf("%q: filename not in expected format: %w", filename, ErrInvalidArgs)
		return
	}
	version = parts[versionIndex]
	// replace '-' prefix of recognised platform suffixes with '+' so work with semver as metadata
	for _, m := range platformSuffixList {
		version = strings.ReplaceAll(version, "-"+m, "+"+m)
	}

	ct = ParseComponent(parts[componentIndex])
	platformIndex := re.SubexpIndex("platform")
	if platformIndex != -1 && len(parts) > platformIndex {
		platform = parts[platformIndex]
	}

	suffix = parts[suffixIndex]
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
