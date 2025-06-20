//go:build !windows

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

package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// listOpenFiles is a placeholder for functionality to come later
func listOpenFiles(i geneos.Instance) (lines []string) {
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
		lines = append(lines, fmt.Sprintf("\t%d:%s (%d bytes) %s", n, perms, files[n].Stat.Size(), path))
	}
	return
}
