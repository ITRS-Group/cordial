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

package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/rs/zerolog/log"
)

// listOpenFiles is a placeholder for functionality to come later
func listOpenFiles(i geneos.Instance) {

	// list open files (test code)

	instdir := i.Home()
	files := instance.Files(i)
	fds := make([]int, len(files))
	j := 0
	for f := range files {
		fds[j] = f
		j++
	}
	sort.Ints(fds)
	for _, n := range fds {
		fdPath := files[n].FD
		perms := ""
		p := files[n].FDMode & 0700
		log.Debug().Msgf("%s perms %o", fdPath, p)
		if p&0400 == 0400 {
			perms += "r"
		}
		if p&0200 == 0200 {
			perms += "w"
		}

		path := files[n].Path
		if strings.HasPrefix(path, instdir) {
			path = strings.Replace(path, instdir, ".", 1)
		}
		fmt.Fprintf(psTabWriter, "\t%d:%s (%d bytes) %s\n", n, perms, files[n].Stat.Size(), path)
	}
}
