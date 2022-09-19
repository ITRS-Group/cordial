package geneos

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
)

const defaultURL = "https://resources.itrsgroup.com/download/latest/"

func init() {
	config.GetConfig().SetDefault("download.url", defaultURL)
}

// how to split an archive name into type and version
var archiveRE = regexp.MustCompile(`^geneos-(web-server|fixanalyser2-netprobe|file-agent|\w+)-([\w\.-]+?)[\.-]?linux`)

func Install(r *host.Host, ct *Component, options ...GeneosOptions) (err error) {
	if r == host.ALL {
		return ErrInvalidArgs
	}

	if ct == nil {
		for _, t := range RealComponents() {
			if err = Install(r, t, options...); err != nil {
				if errors.Is(err, fs.ErrExist) {
					continue
				}
				return
			}
		}
		return nil
	}

	osinfo := r.GetStringMapString("osinfo")
	if p, ok := osinfo["PLATFORM_ID"]; ok {
		options = append(options, PlatformID(p))
	}
	reader, filename, err := OpenComponentArchive(ct, options...)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err = Unarchive(r, ct, filename, reader, options...); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil
		}
		return err
	}
	return
}

func FilenameFromHTTPResp(resp *http.Response, u *url.URL) (filename string, err error) {
	cd, ok := resp.Header[http.CanonicalHeaderKey("content-disposition")]
	if !ok && resp.Request.Response != nil {
		cd, ok = resp.Request.Response.Header[http.CanonicalHeaderKey("content-disposition")]
	}
	if ok {
		_, params, err := mime.ParseMediaType(cd[0])
		if err == nil {
			if f, ok := params["filename"]; ok {
				filename = f
			}
		}
	}

	// if no content-disposition, then grab the path from the response URL
	if filename == "" {
		filename, err = host.CleanRelativePath(path.Base(u.Path))
		if err != nil {
			return
		}
	}
	return
}

func OpenLocalFileOrURL(source string, options ...GeneosOptions) (from io.ReadCloser, filename string, err error) {
	opts := EvalOptions(options...)

	u, err := url.Parse(source)
	if err != nil {
		return
	}

	switch {
	case u.Scheme == "https" || u.Scheme == "http":
		var resp *http.Response
		resp, err = http.Get(u.String())
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
		// only use auth if required
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			if opts.username != "" {
				var req *http.Request
				client := &http.Client{}
				if req, err = http.NewRequest("GET", u.String(), nil); err != nil {
					log.Fatal().Err(err).Msg("")
				}
				req.SetBasicAuth(opts.username, opts.password)
				if resp, err = client.Do(req); err != nil {
					log.Fatal().Err(err).Msg("")
				}
			}
		}

		if resp.StatusCode > 299 {
			return nil, "", fmt.Errorf("server returned %s for %q", resp.Status, source)
		}

		from = resp.Body
		filename, err = FilenameFromHTTPResp(resp, resp.Request.URL)
	case source == "-":
		from = os.Stdin
		filename = "STDIN"
	default:
		from, err = os.Open(source)
		filename = filepath.Base(source)
	}
	return
}

func ReadLocalFileOrURL(source string) (b []byte, err error) {
	var from io.ReadCloser
	from, _, err = OpenLocalFileOrURL(source)
	if err != nil {
		return
	}
	defer from.Close()
	return io.ReadAll(from)
}
