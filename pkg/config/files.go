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

package config

import (
	"io"

	"github.com/itrs-group/cordial/pkg/host"
)

// OpenPromoteFile searches paths on remote r for the first file to exist and be
// readable. If this is not the first element in paths it then renames/moves ths
// found file to the path in the first element in the slice (unless it's an
// empty string). It returns an io.ReadSeekCloser for the open file and the
// final path. If there is an error moving the file then the returned path is
// that of the originally opened file.
func OpenPromoteFile(r host.Remote, paths []string) (f io.ReadSeekCloser, final string) {
	for i, path := range paths {
		var err error
		f, err = r.Open(path)
		// if open is successful then the file is useful
		if err == nil {
			final = path
			if i == 0 || paths[0] == "" {
				return
			}
			if err = r.Rename(path, paths[0]); err != nil {
				return
			}
			final = paths[0]
			return
		}
	}

	return nil, ""
}

func ReadConfig(r host.Remote) {

}

func WriteConfig(r host.Remote) {

}
