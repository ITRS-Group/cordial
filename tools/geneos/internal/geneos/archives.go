package geneos

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
)

// locate and return an open archive for the host and component given
// archives must be local
func OpenComponentArchive(ct *Component, options ...GeneosOptions) (body io.ReadCloser, filename string, err error) {
	var resp *http.Response

	opts := EvalOptions(options...)

	if opts.filename != "" {
		return OpenLocalFileOrURL(opts.filename)
	}

	if opts.local {
		// archive directory is local only
		archiveDir := host.LOCAL.Filepath("packages", "downloads")
		filename = latest(host.LOCAL, archiveDir, opts.version, func(v os.DirEntry) bool {
			logDebug.Println(v.Name(), ct.String())
			switch ct.String() {
			case "webserver":
				return !strings.Contains(v.Name(), "web-server")
			case "fa2":
				return !strings.Contains(v.Name(), "fixanalyser2-netprobe")
			case "fileagent":
				return !strings.Contains(v.Name(), "file-agent")
			default:
				return !strings.Contains(v.Name(), ct.String())
			}
		})
		if filename == "" {
			err = fmt.Errorf("local installation selected but no suitable file found for %s (%w)", ct, ErrInvalidArgs)
			return
		}
		var f io.ReadSeekCloser
		if f, err = host.LOCAL.Open(filepath.Join(archiveDir, filename)); err != nil {
			err = fmt.Errorf("local installation selected but no suitable file found for %s (%w)", ct, err)
			return
		}
		body = f
		return
	}

	if filename, resp, err = checkArchive(host.LOCAL, ct, options...); err != nil {
		return
	}

	archiveDir := filepath.Join(host.Geneos(), "packages", "downloads")
	host.LOCAL.MkdirAll(archiveDir, 0775)
	archivePath := filepath.Join(archiveDir, filename)
	s, err := host.LOCAL.Stat(archivePath)
	if err == nil && s.St.Size() == resp.ContentLength {
		if f, err := host.LOCAL.Open(archivePath); err == nil {
			logDebug.Println("not downloading, file already exists:", archivePath)
			resp.Body.Close()
			return f, filename, nil
		}
	}

	if resp.StatusCode > 299 {
		err = fmt.Errorf("cannot download %s package version %q: %s", ct, opts.version, resp.Status)
		resp.Body.Close()
		return
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
	log.Printf("downloading %s package version %q to %s", ct, opts.version, archivePath)
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
	log.Printf("downloaded %d bytes in %.3f seconds (%.0f bytes/sec)", b, dr, bps)
	if _, err = w.Seek(0, 0); err != nil {
		return
	}
	body = w
	return
}

func Unarchive(r *host.Host, ct *Component, filename string, gz io.Reader, options ...GeneosOptions) (err error) {
	var version string

	opts := EvalOptions(options...)

	if opts.override == "" {
		parts := archiveRE.FindStringSubmatch(filename)
		if len(parts) == 0 {
			return fmt.Errorf("%q: %w", filename, ErrInvalidArgs)
		}
		version = parts[2]
		// check the component in the filename
		// special handling for Sans
		ctFromFile := ParseComponentName(parts[1])
		switch ct.Name {
		case "none", "san":
			ct = ctFromFile
		case ctFromFile.Name:
			break
		default:
			// mismatch
			logDebug.Printf("component type and archive mismatch: %q is not a %q", filename, ct)
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

	basedir := r.Filepath("packages", ct, version)
	logDebug.Println(basedir)
	if _, err = r.Stat(basedir); err == nil {
		// something is already using that dir
		// XXX - option to delete and overwrite?
		return
	}
	if err = r.MkdirAll(basedir, 0775); err != nil {
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
		if name, err = host.CleanRelativePath(name); err != nil {
			logError.Fatalln(err)
		}
		fullpath := filepath.Join(basedir, name)
		switch hdr.Typeflag {
		case tar.TypeReg:
			// check (and created) containing directories - account for munged tar files
			dir := filepath.Dir(fullpath)
			if err = r.MkdirAll(dir, 0775); err != nil {
				return
			}

			var out io.WriteCloser
			if out, err = r.Create(fullpath, hdr.FileInfo().Mode()); err != nil {
				return err
			}
			n, err := io.Copy(out, tr)
			if err != nil {
				out.Close()
				return err
			}
			if n != hdr.Size {
				log.Println("lengths different:", hdr.Size, n)
			}
			out.Close()

		case tar.TypeDir:
			if err = r.MkdirAll(fullpath, hdr.FileInfo().Mode()); err != nil {
				return
			}

		case tar.TypeSymlink, tar.TypeGNULongLink:
			if filepath.IsAbs(hdr.Linkname) {
				logError.Fatalln("archive contains absolute symlink target")
			}
			if _, err = r.Stat(fullpath); err != nil {
				if err = r.Symlink(hdr.Linkname, fullpath); err != nil {
					logError.Fatalln(err)
				}
			}

		default:
			log.Printf("unsupported file type %c\n", hdr.Typeflag)
		}
	}
	log.Printf("installed %q to %q\n", filename, r.Path(basedir))
	options = append(options, Version(version))
	return Update(r, ct, options...)
}

// locate and open the archive using the download conventions
// XXX this is where we do nexus or resources or something else?
func checkArchive(r *host.Host, ct *Component, options ...GeneosOptions) (filename string, resp *http.Response, err error) {
	var source string

	opts := EvalOptions(options...)

	// XXX OS filter for EL8 here - to test
	// cannot fetch partial versions for el8
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

		logDebug.Println("nexus url:", source)

		if config.GetString("download.username") != "" {
			var req *http.Request
			client := &http.Client{}
			if req, err = http.NewRequest("GET", source, nil); err != nil {
				logError.Fatalln(err)
			}
			req.SetBasicAuth(config.GetString("download.username"), config.GetString("download.password"))
			if resp, err = client.Do(req); err != nil {
				logError.Fatalln(err)
			}
		} else {
			if resp, err = http.Get(source); err != nil {
				logError.Fatalln(err)
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
				log.Fatalf("cannot download specific version for this platform (%q) - please download manually", platform)
			}
			v.Set("title", opts.version)
		} else if platform != "" {
			v.Set("title", "-"+platform)
		}

		realpath.RawQuery = v.Encode()
		source = downloadURL.ResolveReference(realpath).String()

		logDebug.Println("source url:", source)

		if resp, err = http.Get(source); err != nil {
			logError.Fatalln(err)
		}

		if resp.StatusCode == 404 && platform != "" {
			resp.Body.Close()
			v.Del("title")
			realpath.RawQuery = v.Encode()
			source = downloadURL.ResolveReference(realpath).String()

			logDebug.Printf("platform download failed, retry source url: %q", source)
			if resp, err = http.Get(source); err != nil {
				logError.Fatalln(err)
			}
		}

		// only use auth if required - but save auth for potential reuse below
		var auth_body []byte
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			if config.GetString("download.username") != "" {
				da := downloadauth{config.GetString("download.username"), config.GetString("download.password")}
				auth_body, err = json.Marshal(da)
				if err != nil {
					logError.Fatalln(err)
				}
				ba := auth_body
				auth_reader := bytes.NewBuffer(ba)
				if resp, err = http.Post(source, "application/json", auth_reader); err != nil {
					logError.Fatalln(err)
				}
			}
		}

		if resp.StatusCode == 404 && platform != "" {
			resp.Body.Close()
			// try without platform type (e.g. no '-el8')
			v.Del("title")
			realpath.RawQuery = v.Encode()
			source = downloadURL.ResolveReference(realpath).String()

			logDebug.Printf("platform download failed, retry source url: %q", source)
			auth_reader := bytes.NewBuffer(auth_body)
			if resp, err = http.Post(source, "application/json", auth_reader); err != nil {
				logError.Fatalln(err)
			}
		}
	}

	if resp.StatusCode > 299 {
		err = fmt.Errorf("cannot access %s package version %s: %s", ct, opts.version, resp.Status)
		resp.Body.Close()
		return
	}

	filename, err = FilenameFromHTTPResp(resp, resp.Request.URL)
	if err != nil {
		return
	}

	logDebug.Printf("download check for %s versions %q returned %s (%d bytes)", ct, opts.version, filename, resp.ContentLength)
	return
}

type downloadauth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

var versRE = regexp.MustCompile(`(\d+(\.\d+){0,2})`)
var anchoredVersRE = regexp.MustCompile(`^(\d+(\.\d+){0,2})$`)

func MatchVersion(v string) bool {
	return anchoredVersRE.MatchString(v)
}

// given a directory find the "latest" version of the form
// [GA]M.N.P[-DATE] M, N, P are numbers, DATE is treated as a string
func latest(r *host.Host, dir, filter string, fn func(os.DirEntry) bool) (latest string) {
	dirs, err := r.ReadDir(dir)
	if err != nil {
		return
	}

	filterRE, err := regexp.Compile(filter)
	if err != nil {
		logDebug.Printf("invalid filter regexp %q", filter)
	}
	for n := 0; n < len(dirs); n++ {
		if !filterRE.MatchString(dirs[n].Name()) {
			dirs[n] = dirs[len(dirs)-1]
			dirs = dirs[:len(dirs)-1]
		}
	}

	max := make([]int, 3)
	for _, v := range dirs {
		if fn(v) {
			continue
		}
		// strip 'GA' prefix and get name
		d := strings.TrimPrefix(v.Name(), "GA")
		x := versRE.FindString(d)
		if x == "" {
			logDebug.Println(d, "does not match a valid directory pattern")
			continue
		}
		s := strings.SplitN(x, ".", 3)

		// make sure we have three levels, fill with 0
		for len(s) < len(max) {
			s = append(s, "0")
		}
		next := sliceAtoi(s)

	OUTER:
		for i := range max {
			switch {
			case next[i] < max[i]:
				break OUTER
			case next[i] > max[i]:
				// do a final lexical scan for suffixes?
				latest = v.Name()
				max[i] = next[i]
			default:
				// if equal and we are on last number, lexical comparison
				// to pick up suffixes
				if len(max) == i+1 && v.Name() > latest {
					latest = v.Name()
				}
			}
		}
	}
	return
}

func sliceAtoi(s []string) (n []int) {
	for _, x := range s {
		i, err := strconv.Atoi(x)
		if err != nil {
			i = 0
		}
		n = append(n, i)
	}
	return
}
