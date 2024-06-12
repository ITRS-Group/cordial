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
		log.Debug().Msgf("going to remove %s", path.Join(i.Home(), p))
		// glob here
		m, err := i.Host().Glob(path.Join(i.Home(), p))
		if err != nil {
			return err
		}
		for _, f := range m {
			log.Debug().Msgf("trying to RemoveAll(%s)", f)
			if err = i.Host().RemoveAll(f); err != nil {
				log.Error().Err(err).Msg("")
				continue
			}
			fmt.Printf("removed %s\n", i.Host().Path(f))
		}
	}
	return
}
