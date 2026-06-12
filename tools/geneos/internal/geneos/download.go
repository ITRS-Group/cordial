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
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

const defaultURL = "https://resources.itrsgroup.com/download/latest/"

// FilenameFromHTTPResp decodes and returns the filename from the
// HTTP(S) request. It tried to extract the filename from the
// COntent-Disposition header and if that fails returns the basename of
// the URL Path.
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
		if filename, err = CleanRelativePath(path.Base(u.Path)); err != nil {
			return
		}
	}
	return
}

// ReadAll opens and, if successful, reads the contents of the source
// passed, returning a byte slice of the contents or an error. source
// can be a local file or a URL. While ReadAll calls the internal
// openSourceFile(), no options are supported
func ReadAll(source string) (b []byte, err error) {
	var from io.ReadCloser
	from, _, _, err = openSource(source)
	if err != nil {
		return
	}
	defer from.Close()
	return io.ReadAll(from)
}

// openSource returns an io.ReadCloser and the base filename for the
// given source. The source can be a `https` or `http` URL, an `sftp`
// URL (including directories) or a path to a file or '-' for STDIN.
//
// `http` and `https` URLs are parsed and fetched with the `http.Get`
// function and so support proxies and basic authentication as supported
// by the http package.
//
// `sftp` URLs are parsed and fetched using the cordial `pkg/host` package.
//
// As a special case, if the file path begins '~/' then it is relative
// to the home directory of the calling user, otherwise it is opened
// relative to the working directory of the process. If passed the
// option `geneos.Homedir()` then this is used instead of the calling
// user's home directory
//
// If source is a path to a directory then `geneos.ErrIsADirectory` is
// returned. If any other stage fails then err is returned from the
// underlying package.
func openSource(source string, options ...PackageOption) (from io.ReadCloser, filename string, filesize int64, err error) {
	opts := evalOptions(options...)

	filesize = -1 // unknown

	switch {
	case IsURL(source):
		u, err2 := url.ParseRequestURI(source)
		if err2 != nil {
			return nil, "", -1, err2
		}

		switch u.Scheme {
		case "http", "https":

			var req *http.Request
			var resp *http.Response

			client := &http.Client{}

			req, err = http.NewRequest("GET", source, nil)
			if err != nil {
				return nil, "", -1, err
			}

			// add any headers
			for _, h := range opts.headers {
				name, value, found := strings.Cut(h, "=")
				if found {
					req.Header.Add(name, value)
				}
			}
			req1 := req.Clone(req.Context())
			resp, err = client.Do(req1)
			if err != nil {
				return
			}

			// only use auth if required
			if resp.StatusCode == 401 || resp.StatusCode == 403 {
				if opts.username != "" {
					req2 := req.Clone(req.Context())
					pw := opts.password
					req2.SetBasicAuth(opts.username, string(pw))
					if resp, err = client.Do(req2); err != nil {
						return
					}
				}
			}

			if resp.StatusCode > 299 {
				return nil, "", -1, fmt.Errorf("server returned %s for %q", resp.Status, source)
			}

			from = resp.Body
			filename, err = FilenameFromHTTPResp(resp, resp.Request.URL)
			filesize = resp.ContentLength
		case "sftp":
			// use the SSHRemote implementation to open the source. If
			// it's a directory then scan for matching releases
			port, err := strconv.Atoi(u.Port())
			if err != nil {
				port = 22
			}
			password, _ := u.User.Password()
			h := host.NewSSHRemote(
				u.Host,
				host.Hostname(u.Hostname()),
				host.Username(u.User.Username()), // username is the login name for the remote host
				host.Password([]byte(password)),  // password is the login password for the remote host
				host.Port(uint16(port)),
			)
			p := strings.TrimPrefix(u.Path, "/")
			s, err := h.Stat(p)
			if err != nil {
				return nil, "", -1, err
			}
			if s.IsDir() {
				// scan dir, later
				return nil, "", -1, fmt.Errorf("source is a directory, scanning for matching releases is not yet implemented")
			} else {
				from, err = h.Open(p)
				if err != nil {
					return nil, "", -1, err
				}
				filename = path.Base(p)
				filesize = s.Size()
			}
		default:
			return nil, "", -1, fmt.Errorf("unsupported URL scheme %q", u.Scheme)
		}
	case source == "-":
		from = os.Stdin
		filename = "STDIN"
	default:
		var s os.FileInfo
		source = config.ResolveHome(source)
		log.Debug("looking at source", slog.String("src", source))
		s, err = os.Stat(source)
		if err != nil {
			log.Debug("error stat-ing source", slog.String("src", source), slog.Any("error", err))
			return nil, "", -1, err
		}
		if s.IsDir() {
			return nil, "", -1, ErrIsADirectory
		}
		source, _ = filepath.Abs(source)
		source = filepath.ToSlash(source)
		from, err = os.Open(source)
		filename = path.Base(source)
		filesize = s.Size()
	}
	return
}
