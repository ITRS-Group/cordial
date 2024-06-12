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
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
)

const defaultURL = "https://resources.itrsgroup.com/download/latest/"

func init() {
	config.GetConfig().SetDefault(config.Join("download", "url"), defaultURL)
}

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

// open returns an io.ReadCloser and the base filename for the
// given source. The source can be a `https` or `http“ URL or a path to
// a file or '-' for STDIN.
//
// URLs are Parsed and fetched with the `http.Get“ function and so
// support proxies and basic authentication as supported by the http
// package.
//
// As a special case, if the file path begins '~/' then it is relative
// to the home directory of the calling user, otherwise it is opened
// relative to the working directory of the process. If passed the
// option `geneos.Homedir()“ then this is used instead of the calling
// user's home directory
//
// If source is a path to a directory then `geneos.ErrIsADirectory` is
// returned. If any other stage fails then err is returned from the
// underlying package.
func open(source string, options ...PackageOptions) (from io.ReadCloser, filename string, err error) {
	opts := evalOptions(options...)

	u, err := url.Parse(source)
	if err != nil {
		return
	}

	switch {
	case u.Scheme == "https" || u.Scheme == "http":
		var resp *http.Response
		resp, err = http.Get(u.String())
		if err != nil {
			return
		}
		// only use auth if required
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			if opts.username != "" {
				var req *http.Request
				client := &http.Client{}
				if req, err = http.NewRequest("GET", u.String(), nil); err != nil {
					return
				}
				pw, _ := opts.password.Open()
				req.SetBasicAuth(opts.username, pw.String())
				pw.Destroy()
				if resp, err = client.Do(req); err != nil {
					return
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
		if strings.HasPrefix(source, "~/") {
			home, _ := config.UserHomeDir()
			source = path.Join(home, strings.TrimPrefix(source, "~/"))
		}
		var s os.FileInfo
		s, err = os.Stat(source)
		if err != nil {
			return nil, "", err
		}
		if s.IsDir() {
			return nil, "", ErrIsADirectory
		}
		source, _ = filepath.Abs(source)
		source = filepath.ToSlash(source)
		from, err = os.Open(source)
		filename = path.Base(source)
	}
	return
}

// ReadFrom opens and, if successful, reads the contents of the source
// passed, returning a byte slice of the contents or an error. source
// can be a local file or a URL.
func ReadFrom(source string, options ...PackageOptions) (b []byte, err error) {
	var from io.ReadCloser
	from, _, err = open(source, options...)
	if err != nil {
		return
	}
	defer from.Close()
	return io.ReadAll(from)
}
