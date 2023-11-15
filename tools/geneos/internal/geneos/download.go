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
func open(source string, options ...Options) (from io.ReadCloser, filename string, err error) {
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
func ReadFrom(source string, options ...Options) (b []byte, err error) {
	var from io.ReadCloser
	from, _, err = open(source, options...)
	if err != nil {
		return
	}
	defer from.Close()
	return io.ReadAll(from)
}
