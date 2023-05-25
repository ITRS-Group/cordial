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
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial/pkg/host"
	"github.com/rs/zerolog/log"
)

// CleanRelativePath given a path returns a cleaned version. If the
// cleaning results in and absolute path or one that tries to ascend the
// tree then return an error
func CleanRelativePath(path string) (clean string, err error) {
	clean = filepath.Clean(path)
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "../") {
		log.Debug().Msgf("path %q must be relative and descending only", clean)
		return "", host.ErrInvalidArgs
	}

	return
}
