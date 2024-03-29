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

package instance

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// Clean removes all the files and directories listed in the component
// type of the instance i for removal. First all entries in the
// CleanList are removed and then, if full is true, the instance is
// stopped and the entries in FullClean are removed. Any instances
// stopped are started up, but any that were already stopped will be
// left stopped.
func Clean(i geneos.Instance, full bool) (err error) {
	var stopped bool

	cleanlist := config.GetString(i.Type().CleanList)
	purgelist := config.GetString(i.Type().PurgeList)

	if !full {
		if cleanlist != "" {
			if err = RemovePaths(i, cleanlist); err == nil {
				log.Debug().Msgf("%s cleaned", i)
			}
		}
		return
	}

	if !IsRunning(i) {
		stopped = false
		// stop failed?
	} else if err = Stop(i, true, false); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return
	} else {
		stopped = true
	}

	if cleanlist != "" {
		if err = RemovePaths(i, cleanlist); err != nil {
			return
		}
	}
	if purgelist != "" {
		if err = RemovePaths(i, purgelist); err != nil {
			return
		}
	}
	log.Debug().Msgf("%s fully cleaned", i)
	if stopped {
		err = Start(i)
	}
	return
}

// RemovePaths removes all files and directories in paths, each file or directory is separated by ListSeperator
func RemovePaths(i geneos.Instance, paths string) (err error) {
	list := filepath.SplitList(paths)
	for _, p := range list {
		// clean path, error on absolute or parent paths, like 'import'
		// walk globbed directories, remove everything
		if p, err = geneos.CleanRelativePath(p); err != nil {
			return fmt.Errorf("%s %w", p, err)
		}
		// glob here
		m, err := i.Host().Glob(path.Join(i.Home(), p))
		if err != nil {
			return err
		}
		for _, f := range m {
			if err = i.Host().RemoveAll(f); err != nil {
				log.Error().Err(err).Msg("")
				continue
			}
			fmt.Printf("removed %s\n", i.Host().Path(f))
		}
	}
	return
}
